package main

import (
	"strings"
	"testing"

	"github.com/murraycode/skill-wiz/result"
	"github.com/murraycode/skill-wiz/skill"
)

func TestValidationResultForSkill(t *testing.T) {
	tests := []struct {
		name          string
		skill         skill.Skill
		wantClean     bool
		wantFindings  int
		wantMessages  []string
		wantEvidence  []string
	}{
		{
			name:         "valid skill is clean",
			skill:        skill.Skill{Name: "test skill", Description: "a test skill"},
			wantClean:    true,
			wantFindings: 0,
		},
		{
			name:         "missing required fields produce validation findings",
			skill:        skill.Skill{},
			wantClean:    false,
			wantFindings: 2,
			wantMessages: []string{"field name is required", "field description is required"},
			wantEvidence: []string{"missing required field: name", "missing required field: description"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validationResultForSkill(&tt.skill)

			if got.Clean() != tt.wantClean {
				t.Fatalf("validationResultForSkill().Clean() = %v, want %v", got.Clean(), tt.wantClean)
			}
			if len(got.Findings) != tt.wantFindings {
				t.Fatalf("len(validationResultForSkill().Findings) = %d, want %d", len(got.Findings), tt.wantFindings)
			}

			for i, want := range tt.wantMessages {
				if got.Findings[i].Source != result.SourceValidation {
					t.Fatalf("Finding[%d].Source = %q, want %q", i, got.Findings[i].Source, result.SourceValidation)
				}
				if got.Findings[i].Severity != result.SeverityError {
					t.Fatalf("Finding[%d].Severity = %q, want %q", i, got.Findings[i].Severity, result.SeverityError)
				}
				if got.Findings[i].Category != result.Category("metadata") {
					t.Fatalf("Finding[%d].Category = %q, want %q", i, got.Findings[i].Category, result.Category("metadata"))
				}
				if got.Findings[i].Message != want {
					t.Fatalf("Finding[%d].Message = %q, want %q", i, got.Findings[i].Message, want)
				}
				if got.Findings[i].Evidence.Summary != tt.wantEvidence[i] {
					t.Fatalf("Finding[%d].Evidence.Summary = %q, want %q", i, got.Findings[i].Evidence.Summary, tt.wantEvidence[i])
				}
			}
		})
	}
}

func TestRenderValidationResult(t *testing.T) {
	got := renderResult(result.NewResult(
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
	))

	wants := []string{
		"Scan flagged 2 finding(s)",
		"[error] metadata: field name is required",
		"Evidence: missing required field: name",
		"[error] metadata: field description is required",
		"Evidence: missing required field: description",
	}

	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("renderResult() = %q, want substring %q", got, want)
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
				"Scan flagged 1 finding(s)",
				"[warning] analysis: Analyzer reported potential issues",
				"Evidence: SUSPICIOUS: hidden shell execution",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderResult(tt.result)
			for _, want := range tt.wants {
				if !strings.Contains(got, want) {
					t.Fatalf("renderResult() = %q, want substring %q", got, want)
				}
			}
		})
	}
}
