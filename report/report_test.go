package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/murraycode/skill-wiz/result"
)

func TestRender(t *testing.T) {
	generatedAt := time.Date(2026, 8, 30, 14, 5, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    Input
		wants    []string
		notWants []string
	}{
		{
			name: "clean result renders a clean verdict",
			input: Input{
				SkillName:        "racing lookup",
				SkillDescription: "Finds up to date racing information",
				SourcePath:       "examples/CLEANSKILL.md",
				GeneratedAt:      generatedAt,
				Result:           result.NewCleanResult(),
			},
			wants: []string{
				"<!doctype html>",
				"racing lookup",
				"Finds up to date racing information",
				"examples/CLEANSKILL.md",
				"No findings",
				"manually verify",
				"2026-08-30",
			},
			notWants: []string{`class="finding-card"`},
		},
		{
			name: "flagged result renders each finding with severity, category, source and evidence",
			input: Input{
				SkillName:   "harmless skill",
				SourcePath:  "examples/HIDDENBASHSKILL.md",
				GeneratedAt: generatedAt,
				Result: result.NewResult(
					result.Finding{
						Source:   result.SourceRule,
						Category: result.Category("shell"),
						Severity: result.SeverityError,
						Message:  "skill references local shell script execution",
						Evidence: result.Evidence{Summary: "./scripts/racing.sh"},
					},
					result.Finding{
						Source:   result.SourceAnalyzer,
						Category: result.Category("hidden"),
						Severity: result.SeverityWarning,
						Message:  "hidden follow-up action detected",
						Evidence: result.Evidence{Summary: "model found extra hidden action"},
					},
				),
			},
			wants: []string{
				"2 findings",
				"skill references local shell script execution",
				"hidden follow-up action detected",
				"./scripts/racing.sh",
				"model found extra hidden action",
				"shell",
				"hidden",
				"rule",
				"analyzer",
				"severity-error",
				"severity-warning",
			},
		},
		{
			name: "findings are ordered by severity",
			input: Input{
				SkillName:   "ordering skill",
				SourcePath:  "skill.md",
				GeneratedAt: generatedAt,
				Result: result.NewResult(
					result.Finding{
						Source:   result.SourceAnalyzer,
						Category: result.Category("style"),
						Severity: result.SeverityInfo,
						Message:  "informational finding",
					},
					result.Finding{
						Source:   result.SourceRule,
						Category: result.Category("url"),
						Severity: result.SeverityWarning,
						Message:  "warning finding",
					},
					result.Finding{
						Source:   result.SourceValidation,
						Category: result.Category("metadata"),
						Severity: result.SeverityError,
						Message:  "error finding",
					},
				),
			},
			wants: []string{"error finding", "warning finding", "informational finding"},
		},
		{
			name: "untrusted skill content is escaped",
			input: Input{
				SkillName:        "<script>alert('name')</script>",
				SkillDescription: "<img src=x onerror=alert('desc')>",
				SourcePath:       "skill.md",
				GeneratedAt:      generatedAt,
				Result: result.NewResult(result.Finding{
					Source:   result.SourceAnalyzer,
					Category: result.Category("hidden"),
					Severity: result.SeverityWarning,
					Message:  "<script>alert('message')</script>",
					Evidence: result.Evidence{Summary: "</style><script>alert('evidence')</script>"},
				}),
			},
			wants: []string{"&lt;script&gt;", "&lt;img"},
			notWants: []string{
				"<script>alert('name')</script>",
				"<script>alert('message')</script>",
				"<script>alert('evidence')</script>",
				"<img src=x",
			},
		},
		{
			name: "missing skill name falls back to the source file name",
			input: Input{
				SourcePath:  filepath.Join("examples", "BROKENSKILL.md"),
				GeneratedAt: generatedAt,
				Result: result.NewResult(result.Finding{
					Source:   result.SourceValidation,
					Category: result.Category("metadata"),
					Severity: result.SeverityError,
					Message:  "field name is required",
					Evidence: result.Evidence{Summary: "missing required field: name"},
				}),
			},
			wants: []string{"BROKENSKILL.md", "1 finding"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.input)
			if err != nil {
				t.Fatalf("Render() error = %v, want nil", err)
			}

			for _, want := range tt.wants {
				if !strings.Contains(got, want) {
					t.Fatalf("Render() output missing substring %q", want)
				}
			}
			for _, notWant := range tt.notWants {
				if strings.Contains(got, notWant) {
					t.Fatalf("Render() output contains unwanted substring %q", notWant)
				}
			}
		})
	}
}

func TestRenderOrdersFindingsBySeverity(t *testing.T) {
	got, err := Render(Input{
		SkillName:  "ordering skill",
		SourcePath: "skill.md",
		Result: result.NewResult(
			result.Finding{Severity: result.SeverityInfo, Message: "informational finding"},
			result.Finding{Severity: result.SeverityWarning, Message: "warning finding"},
			result.Finding{Severity: result.SeverityError, Message: "error finding"},
		),
	})
	if err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	order := []string{"error finding", "warning finding", "informational finding"}
	previous := -1
	for _, message := range order {
		index := strings.Index(got, message)
		if index == -1 {
			t.Fatalf("Render() output missing message %q", message)
		}
		if index < previous {
			t.Fatalf("Render() placed %q out of severity order", message)
		}
		previous = index
	}
}

func TestWrite(t *testing.T) {
	tests := []struct {
		name        string
		destination string
		want        string
	}{
		{
			name:        "writes report to an existing directory",
			destination: "report.html",
			want:        "skill-wiz",
		},
		{
			name:        "creates missing parent directories",
			destination: filepath.Join("nested", "dir", "report.html"),
			want:        "skill-wiz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), tt.destination)

			if err := Write(destination, Input{
				SkillName:  "written skill",
				SourcePath: "skill.md",
				Result:     result.NewCleanResult(),
			}); err != nil {
				t.Fatalf("Write() error = %v, want nil", err)
			}

			content, err := os.ReadFile(destination)
			if err != nil {
				t.Fatalf("os.ReadFile() error = %v, want nil", err)
			}
			if !strings.Contains(string(content), tt.want) {
				t.Fatalf("Write() content missing substring %q", tt.want)
			}
			if !strings.Contains(string(content), "written skill") {
				t.Fatalf("Write() content missing the skill name")
			}
		})
	}
}

func TestRenderMultipleSkills(t *testing.T) {
	generatedAt := time.Date(2026, 8, 30, 14, 5, 0, 0, time.UTC)

	got, err := Render(
		Input{
			SkillName:   "clean skill",
			SourcePath:  filepath.Join("examples", "CLEANSKILL.md"),
			GeneratedAt: generatedAt,
			Result:      result.NewCleanResult(),
		},
		Input{
			SkillName:   "harmless skill",
			SourcePath:  filepath.Join("examples", "HIDDENBASHSKILL.md"),
			GeneratedAt: generatedAt,
			Result: result.NewResult(result.Finding{
				Source:   result.SourceRule,
				Category: result.Category("shell"),
				Severity: result.SeverityError,
				Message:  "skill references local shell script execution",
				Evidence: result.Evidence{Summary: "./scripts/racing.sh"},
			}),
		},
	)
	if err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	wants := []string{
		`<select id="skill-picker"`,
		`<option value="skill-1"`,
		`<option value="skill-2"`,
		`id="skill-1"`,
		`id="skill-2"`,
		"CLEANSKILL.md — no findings",
		"HIDDENBASHSKILL.md — 1 finding",
		"clean skill",
		"harmless skill",
		"skill references local shell script execution",
		"2 skills",
	}
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("Render() output missing substring %q", want)
		}
	}
}

func TestRenderSingleSkillHasNoPicker(t *testing.T) {
	got, err := Render(Input{
		SkillName:  "only skill",
		SourcePath: "skill.md",
		Result:     result.NewCleanResult(),
	})
	if err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	for _, notWant := range []string{"skill-picker", "<option"} {
		if strings.Contains(got, notWant) {
			t.Fatalf("Render() output contains unwanted substring %q", notWant)
		}
	}
	if !strings.Contains(got, "only skill") {
		t.Fatalf("Render() output missing the skill name")
	}
}

func TestRenderWithoutScans(t *testing.T) {
	if _, err := Render(); err == nil {
		t.Fatal("Render() error = nil, want an error for an empty run")
	}
}

func TestRenderLabelsDisambiguateSharedFileNames(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		wants []string
	}{
		{
			name:  "distinct file names are labelled by file name",
			paths: []string{filepath.Join("a", "alpha.md"), filepath.Join("b", "beta.md")},
			wants: []string{"alpha.md — no findings", "beta.md — no findings"},
		},
		{
			name:  "shared file names fall back to the full path",
			paths: []string{filepath.Join("a", "SKILL.md"), filepath.Join("b", "SKILL.md")},
			wants: []string{
				filepath.Join("a", "SKILL.md") + " — no findings",
				filepath.Join("b", "SKILL.md") + " — no findings",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputs := make([]Input, 0, len(tt.paths))
			for _, path := range tt.paths {
				inputs = append(inputs, Input{SkillName: "shared", SourcePath: path, Result: result.NewCleanResult()})
			}

			got, err := Render(inputs...)
			if err != nil {
				t.Fatalf("Render() error = %v, want nil", err)
			}

			for _, want := range tt.wants {
				if !strings.Contains(got, want) {
					t.Fatalf("Render() output missing substring %q", want)
				}
			}
		})
	}
}

func TestRenderEscapesPickerLabels(t *testing.T) {
	got, err := Render(
		Input{SkillName: "first", SourcePath: "first.md", Result: result.NewCleanResult()},
		Input{SkillName: "second", SourcePath: "</option><script>alert('label')</script>.md", Result: result.NewCleanResult()},
	)
	if err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	if strings.Contains(got, "<script>alert('label')</script>") {
		t.Fatal("Render() output contains an unescaped picker label")
	}
}

func TestWriteMultipleSkills(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "skill-wiz-report.html")

	if err := Write(destination,
		Input{SkillName: "first skill", SourcePath: "first.md", Result: result.NewCleanResult()},
		Input{SkillName: "second skill", SourcePath: "second.md", Result: result.NewCleanResult()},
	); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	content, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v, want nil", err)
	}
	for _, want := range []string{"first skill", "second skill", "skill-picker"} {
		if !strings.Contains(string(content), want) {
			t.Fatalf("Write() content missing substring %q", want)
		}
	}
}
