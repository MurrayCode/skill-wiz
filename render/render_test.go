package render

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/murraycode/skill-wiz/result"
)

func TestRenderScans(t *testing.T) {
	clean := Input{
		Path:   filepath.Join("examples", "CLEANSKILL.md"),
		Result: result.NewCleanResult(),
	}
	flagged := Input{
		Path: filepath.Join("examples", "HIDDENBASHSKILL.md"),
		Result: result.NewResult(result.Finding{
			Source:   result.SourceRule,
			Category: result.Category("shell"),
			Severity: result.SeverityError,
			Message:  "skill references local shell script execution",
			Evidence: result.Evidence{Summary: "./scripts/racing.sh"},
		}),
	}

	tests := []struct {
		name        string
		scans       []Input
		total       int
		wants       []string
		wantMissing []string
	}{
		{
			name:        "a single file is rendered without a path header",
			scans:       []Input{clean},
			total:       1,
			wants:       []string{"THIS SKILL APPEARS TO BE CLEAN"},
			wantMissing: []string{"===", "HTML report:"},
		},
		{
			name:  "several files are headed by their path",
			scans: []Input{clean, flagged},
			total: 2,
			wants: []string{
				"=== " + clean.Path + " ===",
				"THIS SKILL APPEARS TO BE CLEAN",
				"=== " + flagged.Path + " ===",
				"[error] shell (rule): skill references local shell script execution",
			},
		},
		{
			name:  "a surviving scan keeps its path when another file failed",
			scans: []Input{flagged},
			total: 2,
			wants: []string{"=== " + flagged.Path + " ==="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Scans(tt.scans, tt.total, Style{})

			for _, want := range tt.wants {
				if !strings.Contains(got, want) {
					t.Fatalf("Scans() = %q, want substring %q", got, want)
				}
			}
			for _, missing := range tt.wantMissing {
				if strings.Contains(got, missing) {
					t.Fatalf("Scans() = %q, want no substring %q", got, missing)
				}
			}
		})
	}
}

func TestRenderValidationResult(t *testing.T) {
	got := Result(result.NewResult(
		result.Finding{
			Source:   result.SourceValidation,
			Category: result.Category("metadata"),
			Severity: result.SeverityError,
			Message:  "field name is required",
			Evidence: result.Evidence{Summary: "missing required field: name"},
		},
		result.Finding{
			Source:   result.SourceValidation,
			Category: result.Category("metadata"),
			Severity: result.SeverityError,
			Message:  "field description is required",
			Evidence: result.Evidence{Summary: "missing required field: description"},
		},
	), Style{})

	wants := []string{
		"Scan flagged 2 finding(s) from validation checks",
		"[error] metadata (validation): field name is required",
		"Evidence: missing required field: name",
		"[error] metadata (validation): field description is required",
		"Evidence: missing required field: description",
	}

	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("Result() = %q, want substring %q", got, want)
		}
	}
}

func TestRenderResult(t *testing.T) {
	tests := []struct {
		name   string
		result result.Result
		wants  []string
	}{
		{
			name:   "clean result renders clean message",
			result: result.NewCleanResult(),
			wants:  []string{"THIS SKILL APPEARS TO BE CLEAN, PLEASE MANUALLY VERIFY TO BE SURE"},
		},
		{
			name: "flagged result renders finding details",
			result: result.NewResult(result.Finding{
				Source:   result.SourceAnalyzer,
				Category: result.Category("analysis"),
				Severity: result.SeverityWarning,
				Message:  "Analyzer reported potential issues",
				Evidence: result.Evidence{Summary: "SUSPICIOUS: hidden shell execution"},
			}),
			wants: []string{
				"Scan flagged 1 finding(s) from analyzer checks",
				"[warning] analysis (analyzer): Analyzer reported potential issues",
				"Evidence: SUSPICIOUS: hidden shell execution",
			},
		},
		{
			name: "merged result renders both sources in summary",
			result: result.Merge(
				result.NewResult(result.Finding{
					Source:   result.SourceRule,
					Category: result.Category("shell"),
					Severity: result.SeverityWarning,
					Message:  "shell execution found",
					Evidence: result.Evidence{Summary: "bash command in body"},
				}),
				result.NewResult(result.Finding{
					Source:   result.SourceAnalyzer,
					Category: result.Category("hidden"),
					Severity: result.SeverityWarning,
					Message:  "hidden follow-up action detected",
					Evidence: result.Evidence{Summary: "model found extra hidden action"},
				}),
			),
			wants: []string{
				"Scan flagged 2 finding(s) from rule and analyzer checks",
				"[warning] shell (rule): shell execution found",
				"[warning] hidden (analyzer): hidden follow-up action detected",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Result(tt.result, Style{})
			for _, want := range tt.wants {
				if !strings.Contains(got, want) {
					t.Fatalf("Result() = %q, want substring %q", got, want)
				}
			}
		})
	}
}

func TestRenderResultOrdersFindingsBySeverity(t *testing.T) {
	finding := func(source result.Source, severity result.Severity, message string) result.Finding {
		return result.Finding{
			Source:   source,
			Category: result.Category("mixed"),
			Severity: severity,
			Message:  message,
		}
	}

	tests := []struct {
		name   string
		result result.Result
		want   []string
	}{
		{
			name: "highest severity is printed first",
			result: result.NewResult(
				finding(result.SourceRule, result.SeverityInfo, "third"),
				finding(result.SourceRule, result.SeverityError, "first"),
				finding(result.SourceRule, result.SeverityWarning, "second"),
			),
			want: []string{"first", "second", "third"},
		},
		{
			name: "merge order is kept within a severity",
			result: result.NewResult(
				finding(result.SourceRule, result.SeverityWarning, "rule finding"),
				finding(result.SourceAnalyzer, result.SeverityWarning, "analyzer finding"),
				finding(result.SourceAnalyzer, result.SeverityError, "analyzer error"),
			),
			want: []string{"analyzer error", "rule finding", "analyzer finding"},
		},
		{
			name: "an unknown severity sorts last",
			result: result.NewResult(
				finding(result.SourceAnalyzer, result.Severity("critical"), "unknown severity"),
				finding(result.SourceRule, result.SeverityInfo, "known severity"),
			),
			want: []string{"known severity", "unknown severity"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Result(tt.result, Style{})

			previous := -1
			for _, want := range tt.want {
				index := strings.Index(got, want)
				if index < 0 {
					t.Fatalf("Result() = %q, want substring %q", got, want)
				}
				if index < previous {
					t.Fatalf("Result() = %q, want %q after the previous finding", got, want)
				}
				previous = index
			}
		})
	}
}

func TestRenderResultDoesNotReorderTheResult(t *testing.T) {
	scanResult := result.NewResult(
		result.Finding{Source: result.SourceRule, Category: result.Category("url"), Severity: result.SeverityInfo, Message: "info first"},
		result.Finding{Source: result.SourceRule, Category: result.Category("shell"), Severity: result.SeverityError, Message: "error second"},
	)

	Result(scanResult, Style{})

	if scanResult.Findings[0].Message != "info first" {
		t.Fatalf("Result() reordered the result: %+v", scanResult.Findings)
	}
}

func TestRenderResultTruncatesEvidence(t *testing.T) {
	tests := []struct {
		name            string
		evidence        string
		wantTruncated   bool
		wantRuneLength  int
		wantContainsAll string
	}{
		{
			name:            "short evidence is untouched",
			evidence:        "./scripts/racing.sh",
			wantContainsAll: "./scripts/racing.sh",
		},
		{
			name:           "evidence at the limit is untouched",
			evidence:       strings.Repeat("a", maxEvidenceRunes),
			wantRuneLength: maxEvidenceRunes,
		},
		{
			name:           "longer evidence is truncated with an ellipsis",
			evidence:       strings.Repeat("b", maxEvidenceRunes+50),
			wantTruncated:  true,
			wantRuneLength: maxEvidenceRunes + 1,
		},
		{
			name:           "truncation counts runes, not bytes",
			evidence:       strings.Repeat("é", maxEvidenceRunes+10),
			wantTruncated:  true,
			wantRuneLength: maxEvidenceRunes + 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Result(result.NewResult(result.Finding{
				Source:   result.SourceRule,
				Category: result.Category("shell"),
				Severity: result.SeverityError,
				Message:  "flagged",
				Evidence: result.Evidence{Summary: tt.evidence},
			}), Style{})

			_, rendered, ok := strings.Cut(got, "Evidence: ")
			if !ok {
				t.Fatalf("Result() = %q, want an evidence line", got)
			}
			rendered = strings.TrimSuffix(rendered, "\n")

			if tt.wantContainsAll != "" && rendered != tt.wantContainsAll {
				t.Fatalf("evidence = %q, want %q", rendered, tt.wantContainsAll)
			}
			if tt.wantRuneLength > 0 && len([]rune(rendered)) != tt.wantRuneLength {
				t.Fatalf("evidence rune length = %d, want %d", len([]rune(rendered)), tt.wantRuneLength)
			}
			if got := strings.HasSuffix(rendered, "…"); got != tt.wantTruncated {
				t.Fatalf("evidence truncated = %t, want %t", got, tt.wantTruncated)
			}
		})
	}
}

func TestRenderResultColour(t *testing.T) {
	scanResult := result.NewResult(
		result.Finding{Source: result.SourceRule, Category: result.Category("shell"), Severity: result.SeverityError, Message: "shell execution"},
		result.Finding{Source: result.SourceRule, Category: result.Category("url"), Severity: result.SeverityWarning, Message: "unrelated url"},
		result.Finding{Source: result.SourceAnalyzer, Category: result.Category("hidden"), Severity: result.SeverityInfo, Message: "worth a look"},
	)

	tests := []struct {
		name        string
		style       Style
		wants       []string
		wantMissing []string
	}{
		{
			name:        "plain output carries no escape codes",
			style:       Style{},
			wants:       []string{"[error] shell (rule): shell execution", "[warning] url", "[info] hidden"},
			wantMissing: []string{"\x1b["},
		},
		{
			name:  "colour wraps the severity label only",
			style: Style{Color: true},
			wants: []string{
				"\x1b[31m[error]\x1b[0m shell (rule): shell execution",
				"\x1b[33m[warning]\x1b[0m url (rule): unrelated url",
				"\x1b[36m[info]\x1b[0m hidden (analyzer): worth a look",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Result(scanResult, tt.style)

			for _, want := range tt.wants {
				if !strings.Contains(got, want) {
					t.Fatalf("Result() = %q, want substring %q", got, want)
				}
			}
			for _, missing := range tt.wantMissing {
				if strings.Contains(got, missing) {
					t.Fatalf("Result() = %q, want no substring %q", got, missing)
				}
			}
		})
	}
}

func TestRenderScansTally(t *testing.T) {
	clean := func(path string) Input {
		return Input{Path: path, Result: result.NewCleanResult()}
	}
	flagged := func(path string, severities ...result.Severity) Input {
		findings := make([]result.Finding, 0, len(severities))
		for _, severity := range severities {
			findings = append(findings, result.Finding{
				Source:   result.SourceRule,
				Category: result.Category("shell"),
				Severity: severity,
				Message:  "flagged",
			})
		}

		return Input{Path: path, Result: result.NewResult(findings...)}
	}

	tests := []struct {
		name        string
		scans       []Input
		total       int
		wants       []string
		wantMissing []string
	}{
		{
			name:        "a single file prints no tally",
			scans:       []Input{flagged("one.md", result.SeverityError)},
			total:       1,
			wantMissing: []string{"scanned ·", "files scanned"},
		},
		{
			name: "a multi file run ends with one tally",
			scans: []Input{
				clean("one.md"),
				flagged("two.md", result.SeverityError, result.SeverityWarning),
				flagged("three.md", result.SeverityWarning, result.SeverityWarning, result.SeverityInfo),
			},
			total: 3,
			wants: []string{"3 files scanned · 1 clean · 2 flagged · 5 findings (1 error, 3 warning, 1 info)"},
		},
		{
			name:  "a clean multi file run counts no findings",
			scans: []Input{clean("one.md"), clean("two.md")},
			total: 2,
			wants: []string{"2 files scanned · 2 clean · 0 flagged · 0 findings"},
			// No severity breakdown when there is nothing to break down.
			wantMissing: []string{"("},
		},
		{
			name:  "the tally counts scanned files, not discovered ones",
			scans: []Input{flagged("two.md", result.SeverityError)},
			total: 2,
			wants: []string{"1 file scanned · 0 clean · 1 flagged · 1 finding (1 error)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Scans(tt.scans, tt.total, Style{})

			for _, want := range tt.wants {
				if !strings.Contains(got, want) {
					t.Fatalf("Scans() = %q, want substring %q", got, want)
				}
			}
			for _, missing := range tt.wantMissing {
				if strings.Contains(got, missing) {
					t.Fatalf("Scans() = %q, want no substring %q", got, missing)
				}
			}
		})
	}
}

func TestRenderScansTallyFollowsACleanFinalFile(t *testing.T) {
	scans := []Input{
		{Path: "one.md", Result: result.NewResult(result.Finding{Source: result.SourceRule, Category: "shell", Severity: result.SeverityError, Message: "flagged"})},
		{Path: "two.md", Result: result.NewCleanResult()},
	}

	got := Scans(scans, 2, Style{})

	// The clean verdict has no trailing newline of its own, so the tally has to
	// supply one rather than running on from it.
	if strings.Contains(got, "SURE2 files") || !strings.Contains(got, "SURE\n") {
		t.Fatalf("Scans() = %q, want the tally on its own line", got)
	}
	if !strings.HasSuffix(got, "2 files scanned · 1 clean · 1 flagged · 1 finding (1 error)\n") {
		t.Fatalf("Scans() = %q, want a trailing tally line", got)
	}
}

func TestRenderResultMarksAnOverriddenSeverity(t *testing.T) {
	tests := []struct {
		name        string
		finding     result.Finding
		wantLine    string
		wantMissing string
	}{
		{
			name: "an overridden finding names the severity it carried",
			finding: result.Finding{
				Source:         result.SourceRule,
				Category:       result.Category("shell"),
				Severity:       result.SeverityInfo,
				Message:        "skill references local shell script execution",
				OverriddenFrom: result.SeverityError,
			},
			wantLine: "[info] shell (rule): skill references local shell script execution (severity overridden from error)",
		},
		{
			name: "an untouched finding reads exactly as it always has",
			finding: result.Finding{
				Source:   result.SourceRule,
				Category: result.Category("shell"),
				Severity: result.SeverityError,
				Message:  "skill references local shell script execution",
			},
			wantLine:    "[error] shell (rule): skill references local shell script execution",
			wantMissing: "overridden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Result(result.NewResult(tt.finding), Style{})

			if !strings.Contains(got, tt.wantLine) {
				t.Fatalf("Result() = %q, want substring %q", got, tt.wantLine)
			}
			if tt.wantMissing != "" && strings.Contains(got, tt.wantMissing) {
				t.Fatalf("Result() = %q, want no substring %q", got, tt.wantMissing)
			}
		})
	}
}
