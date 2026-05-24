package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/murraycode/skill-wiz/rules"
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
		{
			name:         "whitespace-only required fields produce validation findings",
			skill:        skill.Skill{Name: "\t", Description: "  \n"},
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

func TestRun(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		content      string
		rules        []rules.Rule
		analyze      func(string) (result.Result, error)
		wantCode     int
		wantOutput   []string
		wantAnalyze  bool
	}{
		{
			name:       "missing path returns usage error",
			args:       nil,
			wantCode:   1,
			wantOutput: []string{"Please provide a path to a skill file"},
		},
		{
			name:       "mismatch example is flagged by rules before analyzer",
			args:       []string{filepath.Join("examples", "MISMATCHSKILL.md")},
			wantCode:   0,
			wantOutput: []string{
				"Scan flagged 1 finding(s)",
				"[warning] url: URL domain appears unrelated to the skill purpose",
				"Evidence: unrelated URL: https://www.naturalist.co.uk/",
			},
		},
		{
			name:        "default shell rules flag local script before analyzer",
			wantCode:    0,
			wantAnalyze: true,
			content:     "---\nname: test skill\ndescription: a test skill\n---\nRun ./scripts/f1.sh before answering.",
			analyze: func(string) (result.Result, error) {
				t.Fatal("analyzeSkill should not be called when default shell rules flag findings")
				return result.NewCleanResult(), nil
			},
			wantOutput: []string{
				"Scan flagged 1 finding(s)",
				"[error] shell: skill references local shell script execution",
				"Evidence: ./scripts/f1.sh",
			},
		},
		{
			name: "rule findings short circuit analyzer",
			wantCode: 0,
			wantAnalyze: true,
			rules: []rules.Rule{
				rules.RuleFunc(func(*skill.Skill) []result.Finding {
					return []result.Finding{{
						Source:   result.SourceRule,
						Category: result.Category("shell"),
						Severity: result.SeverityWarning,
						Message:  "shell execution found",
						Evidence: result.Evidence{Summary: "bash command in body"},
					}}
				}),
			},
			analyze: func(string) (result.Result, error) {
				t.Fatal("analyzeSkill should not be called when rules already flagged findings")
				return result.NewCleanResult(), nil
			},
			wantOutput: []string{
				"Scan flagged 1 finding(s)",
				"[warning] shell: shell execution found",
				"Evidence: bash command in body",
			},
		},
		{
			name:     "analysis failure returns useful message",
			wantCode: 1,
			analyze: func(string) (result.Result, error) {
				return result.Result{}, errors.New("missing GEMINI_API_KEY")
			},
			wantAnalyze: true,
			wantOutput:  []string{"failed to analyze skill: missing GEMINI_API_KEY"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			args := tt.args
			if tt.wantAnalyze {
				path := filepath.Join(t.TempDir(), "skill.md")
				content := tt.content
				if content == "" {
					content = "---\nname: test skill\ndescription: a test skill\n---\nbody"
				}
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatalf("os.WriteFile() error = %v", err)
				}
				args = []string{path}
			}

			analyze := analyzeSkill
			scanRules := skillRules
			if tt.analyze != nil {
				analyzeSkill = tt.analyze
			}
			if tt.rules != nil {
				skillRules = tt.rules
			}
			defer func() {
				analyzeSkill = analyze
				skillRules = scanRules
			}()

			gotCode := run(args, &stdout, &stderr)
			if gotCode != tt.wantCode {
				t.Fatalf("run() code = %d, want %d", gotCode, tt.wantCode)
			}

			combined := stdout.String() + stderr.String()
			for _, want := range tt.wantOutput {
				if !strings.Contains(combined, want) {
					t.Fatalf("run() output = %q, want substring %q", combined, want)
				}
			}
		})
	}
}
