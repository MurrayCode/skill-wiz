package result

import (
	"reflect"
	"testing"
)

func TestResultClean(t *testing.T) {
	tests := []struct {
		name      string
		result    Result
		wantClean bool
	}{
		{
			name:      "clean result has no findings",
			result:    NewCleanResult(),
			wantClean: true,
		},
		{
			name: "result with finding is flagged",
			result: NewResult(Finding{
				Source:   SourceAnalyzer,
				Category: Category("analysis"),
				Severity: SeverityWarning,
				Message:  "potential mismatch",
				Evidence: Evidence{Summary: "model reported mismatch"},
			}),
			wantClean: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Clean(); got != tt.wantClean {
				t.Fatalf("Result.Clean() = %v, want %v", got, tt.wantClean)
			}
		})
	}
}

func TestNewResultPreservesFindings(t *testing.T) {
	findings := []Finding{
		{
			Source:   SourceValidation,
			Category: Category("metadata"),
			Severity: SeverityError,
			Message:  "missing description",
			Evidence: Evidence{Summary: "field description is empty"},
		},
		{
			Source:   SourceRule,
			Category: Category("shell"),
			Severity: SeverityWarning,
			Message:  "command executes shell",
			Evidence: Evidence{Summary: "bash command found"},
		},
	}

	got := NewResult(findings...)

	if len(got.Findings) != len(findings) {
		t.Fatalf("len(Result.Findings) = %d, want %d", len(got.Findings), len(findings))
	}

	for i := range findings {
		if got.Findings[i] != findings[i] {
			t.Fatalf("Result.Findings[%d] = %#v, want %#v", i, got.Findings[i], findings[i])
		}
	}

	findings[0].Message = "changed after creation"
	if got.Findings[0].Message == findings[0].Message {
		t.Fatal("NewResult() did not protect stored findings from caller mutation")
	}
}

func TestNewResultWithoutFindingsReturnsCleanResult(t *testing.T) {
	got := NewResult()

	if !got.Clean() {
		t.Fatalf("NewResult().Clean() = %v, want true", got.Clean())
	}
	if len(got.Findings) != 0 {
		t.Fatalf("len(NewResult().Findings) = %d, want 0", len(got.Findings))
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name        string
		results     []Result
		wantClean   bool
		wantCount   int
		wantSources []Source
	}{
		{
			name: "clean inputs stay clean",
			results: []Result{
				NewCleanResult(),
				NewCleanResult(),
			},
			wantClean: true,
			wantCount: 0,
		},
		{
			name: "distinct rule and analyzer findings are combined in order",
			results: []Result{
				NewResult(Finding{
					Source:   SourceRule,
					Category: Category("shell"),
					Severity: SeverityError,
					Message:  "skill references local shell script execution",
					Evidence: Evidence{Summary: "./scripts/deploy.sh"},
				}),
				NewResult(Finding{
					Source:   SourceAnalyzer,
					Category: Category("hidden"),
					Severity: SeverityWarning,
					Message:  "Skill hides an extra step",
					Evidence: Evidence{Summary: "model found hidden execution step"},
				}),
			},
			wantClean:   false,
			wantCount:   2,
			wantSources: []Source{SourceRule, SourceAnalyzer},
		},
		{
			name: "overlapping findings are de-duplicated across sources",
			results: []Result{
				NewResult(Finding{
					Source:   SourceRule,
					Category: Category("shell"),
					Severity: SeverityWarning,
					Message:  "skill references shell execution",
					Evidence: Evidence{Summary: "bash -lc 'ls'"},
				}),
				NewResult(Finding{
					Source:   SourceAnalyzer,
					Category: Category("shell"),
					Severity: SeverityWarning,
					Message:  "skill references shell execution",
					Evidence: Evidence{Summary: "bash -lc 'ls'"},
				}),
			},
			wantClean:   false,
			wantCount:   1,
			wantSources: []Source{SourceRule},
		},
		{
			name: "same category with different evidence is preserved",
			results: []Result{
				NewResult(Finding{
					Source:   SourceRule,
					Category: Category("mismatch"),
					Severity: SeverityWarning,
					Message:  "skill instructions diverge from declared purpose",
					Evidence: Evidence{Summary: "description conflicts with body section"},
				}),
				NewResult(Finding{
					Source:   SourceAnalyzer,
					Category: Category("mismatch"),
					Severity: SeverityWarning,
					Message:  "description and body appear inconsistent",
					Evidence: Evidence{Summary: "model noticed different operational intent"},
				}),
			},
			wantClean:   false,
			wantCount:   2,
			wantSources: []Source{SourceRule, SourceAnalyzer},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Merge(tt.results...)

			if got.Clean() != tt.wantClean {
				t.Fatalf("Merge().Clean() = %v, want %v", got.Clean(), tt.wantClean)
			}
			if len(got.Findings) != tt.wantCount {
				t.Fatalf("len(Merge().Findings) = %d, want %d", len(got.Findings), tt.wantCount)
			}

			sources := got.Sources()
			if len(sources) != len(tt.wantSources) {
				t.Fatalf("len(Merge().Sources()) = %d, want %d", len(sources), len(tt.wantSources))
			}
			for i, want := range tt.wantSources {
				if sources[i] != want {
					t.Fatalf("Merge().Sources()[%d] = %q, want %q", i, sources[i], want)
				}
			}
		})
	}
}

func TestGateRankAndDisplayRankDisagreeOnUnknownSeverities(t *testing.T) {
	tests := []struct {
		name            string
		severity        Severity
		wantGateRank    int
		wantDisplayRank int
	}{
		{name: "error", severity: SeverityError, wantGateRank: 2, wantDisplayRank: 0},
		{name: "warning", severity: SeverityWarning, wantGateRank: 1, wantDisplayRank: 1},
		{name: "info", severity: SeverityInfo, wantGateRank: 0, wantDisplayRank: 2},
		{
			// Gating must put an unknown severity below every known one so a
			// malformed finding cannot fail a build on its own — level with info
			// is not low enough, because the comparison is >= and info is a
			// selectable threshold. Display must sort it last so it never
			// outranks a real finding.
			name:            "unknown severity gates below info and displays last",
			severity:        Severity("critical"),
			wantGateRank:    UnknownGateRank,
			wantDisplayRank: 3,
		},
		{name: "empty severity gates below info and displays last", severity: Severity(""), wantGateRank: UnknownGateRank, wantDisplayRank: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GateRank(tt.severity); got != tt.wantGateRank {
				t.Fatalf("GateRank(%q) = %d, want %d", tt.severity, got, tt.wantGateRank)
			}
			if got := DisplayRank(tt.severity); got != tt.wantDisplayRank {
				t.Fatalf("DisplayRank(%q) = %d, want %d", tt.severity, got, tt.wantDisplayRank)
			}
		})
	}
}

// TestUnknownGateRankIsBelowEverySeverity states the guarantee directly rather
// than leaving it implied by the table: no threshold a user can select is low
// enough for an unrecognised severity to fail a build.
func TestUnknownGateRankIsBelowEverySeverity(t *testing.T) {
	for _, severity := range Severities() {
		if UnknownGateRank >= GateRank(severity) {
			t.Fatalf("UnknownGateRank = %d, want strictly below GateRank(%q) = %d", UnknownGateRank, severity, GateRank(severity))
		}
	}
}

func TestKnownAndSeverities(t *testing.T) {
	want := []Severity{SeverityError, SeverityWarning, SeverityInfo}

	got := Severities()
	if len(got) != len(want) {
		t.Fatalf("Severities() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Severities() = %v, want %v", got, want)
		}
		if !Known(want[i]) {
			t.Fatalf("Known(%q) = false, want true", want[i])
		}
	}

	// The returned slice is a copy: reordering it must not move the source.
	got[0] = SeverityInfo
	if Severities()[0] != SeverityError {
		t.Fatal("Severities() returned a slice aliasing the package ordering")
	}

	if Known(Severity("critical")) {
		t.Fatal(`Known("critical") = true, want false`)
	}
}

func TestFormatSources(t *testing.T) {
	tests := []struct {
		name    string
		sources []Source
		want    string
	}{
		{name: "none", sources: nil, want: ""},
		{name: "one", sources: []Source{SourceRule}, want: "rule"},
		{name: "two", sources: []Source{SourceRule, SourceAnalyzer}, want: "rule and analyzer"},
		{
			name:    "three",
			sources: []Source{SourceValidation, SourceRule, SourceAnalyzer},
			want:    "validation, rule, and analyzer",
		},
		{name: "empty sources are dropped", sources: []Source{SourceRule, Source("")}, want: "rule"},
		{name: "only empty sources", sources: []Source{Source("")}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatSources(tt.sources); got != tt.want {
				t.Fatalf("FormatSources(%v) = %q, want %q", tt.sources, got, tt.want)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		name  string
		count int
		noun  string
		want  string
	}{
		{name: "zero", count: 0, noun: "file", want: "0 files"},
		{name: "one", count: 1, noun: "file", want: "1 file"},
		{name: "many", count: 3, noun: "finding", want: "3 findings"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Pluralize(tt.count, tt.noun); got != tt.want {
				t.Fatalf("Pluralize(%d, %q) = %q, want %q", tt.count, tt.noun, got, tt.want)
			}
		})
	}
}

func TestSummarize(t *testing.T) {
	shell := Finding{Source: SourceRule, Category: Category("shell"), Severity: SeverityError, Message: "shell"}
	mismatch := Finding{Source: SourceRule, Category: Category("mismatch"), Severity: SeverityWarning, Message: "mismatch"}
	hidden := Finding{Source: SourceAnalyzer, Category: Category("hidden"), Severity: SeverityWarning, Message: "hidden"}
	note := Finding{Source: SourceAnalyzer, Category: Category("shell"), Severity: SeverityInfo, Message: "note"}

	tests := []struct {
		name        string
		results     []Result
		filesFailed int
		want        Summary
	}{
		{
			name:    "an empty run counts nothing",
			results: nil,
			want:    Summary{BySeverity: []Count{}, ByCategory: []Count{}, BySource: []Count{}},
		},
		{
			name:    "a clean run counts the file and no findings",
			results: []Result{NewCleanResult()},
			want: Summary{
				FilesScanned: 1,
				FilesClean:   1,
				BySeverity:   []Count{},
				ByCategory:   []Count{},
				BySource:     []Count{},
			},
		},
		{
			name:        "failures are counted separately from scanned files",
			results:     []Result{NewCleanResult()},
			filesFailed: 2,
			want: Summary{
				FilesScanned: 1,
				FilesClean:   1,
				FilesFailed:  2,
				BySeverity:   []Count{},
				ByCategory:   []Count{},
				BySource:     []Count{},
			},
		},
		{
			name: "breakdowns rank by count then name",
			results: []Result{
				NewResult(shell, mismatch, note),
				NewResult(hidden),
				NewCleanResult(),
			},
			want: Summary{
				FilesScanned: 3,
				FilesClean:   1,
				FilesFlagged: 2,
				Findings:     4,
				BySeverity:   []Count{{Name: "error", Count: 1}, {Name: "warning", Count: 2}, {Name: "info", Count: 1}},
				ByCategory:   []Count{{Name: "shell", Count: 2}, {Name: "hidden", Count: 1}, {Name: "mismatch", Count: 1}},
				BySource:     []Count{{Name: "analyzer", Count: 2}, {Name: "rule", Count: 2}},
			},
		},
		{
			name:    "an unrecognised severity never leads the breakdown",
			results: []Result{NewResult(Finding{Source: SourceRule, Category: Category("odd"), Severity: Severity("critical")}, note)},
			want: Summary{
				FilesScanned: 1,
				FilesFlagged: 1,
				Findings:     2,
				BySeverity:   []Count{{Name: "info", Count: 1}, {Name: "critical", Count: 1}},
				ByCategory:   []Count{{Name: "odd", Count: 1}, {Name: "shell", Count: 1}},
				BySource:     []Count{{Name: "analyzer", Count: 1}, {Name: "rule", Count: 1}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Summarize(tt.results, tt.filesFailed)

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Summarize() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSummarizeCountsMatchTheFindings(t *testing.T) {
	// The summary is only useful if it agrees with the detail it sits above, so
	// derive both from the same fixture and compare.
	results := []Result{
		NewResult(
			Finding{Source: SourceRule, Category: Category("shell"), Severity: SeverityError},
			Finding{Source: SourceRule, Category: Category("url"), Severity: SeverityWarning},
		),
		NewResult(Finding{Source: SourceAnalyzer, Category: Category("hidden"), Severity: SeverityWarning}),
		NewCleanResult(),
	}

	got := Summarize(results, 0)

	findings := 0
	for _, scanned := range results {
		findings += len(scanned.Findings)
	}
	if got.Findings != findings {
		t.Fatalf("Summarize().Findings = %d, want %d", got.Findings, findings)
	}

	for _, breakdown := range [][]Count{got.BySeverity, got.ByCategory, got.BySource} {
		total := 0
		for _, row := range breakdown {
			total += row.Count
		}
		if total != findings {
			t.Fatalf("breakdown %v totals %d, want %d", breakdown, total, findings)
		}
	}
	if got.FilesClean+got.FilesFlagged != got.FilesScanned {
		t.Fatalf("clean %d + flagged %d != scanned %d", got.FilesClean, got.FilesFlagged, got.FilesScanned)
	}
}
