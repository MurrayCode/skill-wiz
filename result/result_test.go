package result

import "testing"

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
			// Gating must treat an unknown severity as lowest so a malformed
			// finding cannot fail a build on its own; display must sort it last
			// so it never outranks a real finding.
			name:            "unknown severity gates lowest and displays last",
			severity:        Severity("critical"),
			wantGateRank:    0,
			wantDisplayRank: 3,
		},
		{name: "empty severity gates lowest and displays last", severity: Severity(""), wantGateRank: 0, wantDisplayRank: 3},
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
