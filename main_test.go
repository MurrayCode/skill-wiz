package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/murraycode/skill-wiz/analyse"
	"github.com/murraycode/skill-wiz/result"
	"github.com/murraycode/skill-wiz/rules"
	"github.com/murraycode/skill-wiz/scanner"
	"github.com/murraycode/skill-wiz/skill"
)

func TestParseOptions(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    options
		wantErr string
		wantIs  error
	}{
		{
			name: "path only uses defaults",
			args: []string{"skill.md"},
			want: options{paths: []string{"skill.md"}, model: analyse.DefaultModel, timeout: analyse.DefaultTimeout, failOn: result.SeverityError, concurrency: defaultConcurrency},
		},
		{
			name: "flags are applied before the path",
			args: []string{"--json", "--model", "gemini-2.5-pro", "--timeout", "15s", "skill.md"},
			want: options{paths: []string{"skill.md"}, json: true, model: "gemini-2.5-pro", timeout: 15 * time.Second, failOn: result.SeverityError, concurrency: defaultConcurrency},
		},
		{
			name: "no-color is parsed",
			args: []string{"--no-color", "skill.md"},
			want: options{
				paths:       []string{"skill.md"},
				noColor:     true,
				model:       analyse.DefaultModel,
				timeout:     analyse.DefaultTimeout,
				failOn:      result.SeverityError,
				concurrency: defaultConcurrency,
			},
		},
		{
			name: "single dash flags are accepted",
			args: []string{"-json", "-timeout=5s", "skill.md"},
			want: options{paths: []string{"skill.md"}, json: true, model: analyse.DefaultModel, timeout: 5 * time.Second, failOn: result.SeverityError, concurrency: defaultConcurrency},
		},
		{
			name: "every positional argument becomes a path",
			args: []string{"first.md", "skills", "second.md"},
			want: options{
				paths:       []string{"first.md", "skills", "second.md"},
				model:       analyse.DefaultModel,
				timeout:     analyse.DefaultTimeout,
				failOn:      result.SeverityError,
				concurrency: defaultConcurrency,
			},
		},
		{
			name: "fail-on lowers the gate",
			args: []string{"--fail-on", "warning", "skill.md"},
			want: options{
				paths:       []string{"skill.md"},
				model:       analyse.DefaultModel,
				timeout:     analyse.DefaultTimeout,
				failOn:      result.SeverityWarning,
				concurrency: defaultConcurrency,
			},
		},
		{
			name: "fail-on is case insensitive and trimmed",
			args: []string{"--fail-on", " INFO ", "skill.md"},
			want: options{
				paths:       []string{"skill.md"},
				model:       analyse.DefaultModel,
				timeout:     analyse.DefaultTimeout,
				failOn:      result.SeverityInfo,
				concurrency: defaultConcurrency,
			},
		},
		{
			name: "concurrency is parsed",
			args: []string{"--concurrency", "3", "skill.md"},
			want: options{
				paths:       []string{"skill.md"},
				model:       analyse.DefaultModel,
				timeout:     analyse.DefaultTimeout,
				failOn:      result.SeverityError,
				concurrency: 3,
			},
		},
		{
			name:    "non-positive concurrency is rejected",
			args:    []string{"--concurrency", "0", "skill.md"},
			wantErr: "invalid -concurrency: must be greater than zero",
		},
		{
			name:    "unknown fail-on severity is rejected",
			args:    []string{"--fail-on", "critical", "skill.md"},
			wantErr: `invalid -fail-on "critical": must be one of error, warning, info`,
		},
		{
			name:    "missing path returns usage error",
			args:    nil,
			wantErr: "Please provide a path to a skill file",
		},
		{
			name:    "unknown flag returns a clear error",
			args:    []string{"--nope", "skill.md"},
			wantErr: "flag provided but not defined: -nope",
		},
		{
			name:    "invalid timeout value returns a clear error",
			args:    []string{"--timeout", "soon", "skill.md"},
			wantErr: `invalid value "soon" for flag -timeout`,
		},
		{
			name:    "non-positive timeout is rejected",
			args:    []string{"--timeout", "0s", "skill.md"},
			wantErr: "invalid -timeout: must be greater than zero",
		},
		{
			name:    "empty model is rejected",
			args:    []string{"--model", "  ", "skill.md"},
			wantErr: "invalid -model: must not be empty",
		},
		{
			name:   "help request is not an error",
			args:   []string{"-h"},
			wantIs: flag.ErrHelp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer

			got, err := parseOptions(tt.args, &stderr)

			if tt.wantIs != nil {
				if !errors.Is(err, tt.wantIs) {
					t.Fatalf("parseOptions() error = %v, want %v", err, tt.wantIs)
				}
				return
			}
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("parseOptions() error = nil, want %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parseOptions() error = %q, want substring %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOptions() error = %v, want nil", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseOptions() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRenderJSON(t *testing.T) {
	tests := []struct {
		name         string
		result       result.Result
		wantClean    bool
		wantFindings int
	}{
		{
			name:      "clean result reports clean",
			result:    result.NewCleanResult(),
			wantClean: true,
		},
		{
			name: "flagged result carries every finding field",
			result: result.NewResult(result.Finding{
				Source:   result.SourceRule,
				Category: result.Category("shell"),
				Severity: result.SeverityError,
				Message:  "skill references local shell script execution",
				Evidence: result.Evidence{Summary: "./scripts/racing.sh"},
			}),
			wantFindings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered, err := renderJSON([]jsonInput{{
				Path:       "examples/HIDDENBASHSKILL.md",
				Skill:      &skill.Skill{Name: "racing news", Description: "links to racing news"},
				Result:     tt.result,
				ReportPath: "/tmp/skill-wiz-report.html",
			}})
			if err != nil {
				t.Fatalf("renderJSON() error = %v, want nil", err)
			}

			var decoded jsonReport
			if err := json.Unmarshal([]byte(rendered), &decoded); err != nil {
				t.Fatalf("json.Unmarshal(renderJSON()) error = %v, want nil", err)
			}

			if decoded.Path != "examples/HIDDENBASHSKILL.md" {
				t.Fatalf("report path = %q, want %q", decoded.Path, "examples/HIDDENBASHSKILL.md")
			}
			if decoded.Skill.Name != "racing news" || decoded.Skill.Description != "links to racing news" {
				t.Fatalf("report skill = %+v, want name and description", decoded.Skill)
			}
			if decoded.ReportPath != "/tmp/skill-wiz-report.html" {
				t.Fatalf("report_path = %q, want %q", decoded.ReportPath, "/tmp/skill-wiz-report.html")
			}
			if decoded.Clean != tt.wantClean {
				t.Fatalf("clean = %v, want %v", decoded.Clean, tt.wantClean)
			}
			if len(decoded.Findings) != tt.wantFindings {
				t.Fatalf("len(findings) = %d, want %d", len(decoded.Findings), tt.wantFindings)
			}
			if tt.wantFindings > 0 {
				finding := decoded.Findings[0]
				if finding.Source != "rule" || finding.Category != "shell" || finding.Severity != "error" {
					t.Fatalf("finding = %+v, want rule/shell/error", finding)
				}
				if finding.Message == "" || finding.Evidence == "" {
					t.Fatalf("finding = %+v, want message and evidence", finding)
				}
			}
		})
	}
}

func TestRenderJSONMultipleFiles(t *testing.T) {
	rendered, err := renderJSON([]jsonInput{
		{
			Path:   "examples/CLEANSKILL.md",
			Skill:  &skill.Skill{Name: "clean skill", Description: "a clean skill"},
			Result: result.NewCleanResult(),
		},
		{
			Path:  "examples/HIDDENBASHSKILL.md",
			Skill: &skill.Skill{Name: "harmless skill", Description: "racing information"},
			Result: result.NewResult(result.Finding{
				Source:   result.SourceRule,
				Category: result.Category("shell"),
				Severity: result.SeverityError,
				Message:  "skill references local shell script execution",
				Evidence: result.Evidence{Summary: "./scripts/racing.sh"},
			}),
			ReportPath: "/tmp/skill-wiz-report.html",
		},
	})
	if err != nil {
		t.Fatalf("renderJSON() error = %v, want nil", err)
	}

	var decoded []jsonReport
	if err := json.Unmarshal([]byte(rendered), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(renderJSON()) error = %v, rendered = %q", err, rendered)
	}

	if len(decoded) != 2 {
		t.Fatalf("len(reports) = %d, want 2", len(decoded))
	}
	if decoded[0].Path != "examples/CLEANSKILL.md" || !decoded[0].Clean {
		t.Fatalf("report[0] = %+v, want the clean skill", decoded[0])
	}
	if decoded[1].Path != "examples/HIDDENBASHSKILL.md" || decoded[1].Clean {
		t.Fatalf("report[1] = %+v, want the flagged skill", decoded[1])
	}
	if len(decoded[1].Findings) != 1 {
		t.Fatalf("len(report[1].findings) = %d, want 1", len(decoded[1].Findings))
	}
	if decoded[1].ReportPath != "/tmp/skill-wiz-report.html" {
		t.Fatalf("report[1].report_path = %q, want the run report", decoded[1].ReportPath)
	}
}

func TestValidationResultForSkill(t *testing.T) {
	tests := []struct {
		name         string
		skill        skill.Skill
		wantClean    bool
		wantFindings int
		wantMessages []string
		wantEvidence []string
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

func TestRenderReportPointer(t *testing.T) {
	got := renderReportPointer("/tmp/reports/skill-wiz-report.html")

	wants := []string{
		"/tmp/reports/skill-wiz-report.html",
		"file:///tmp/reports/skill-wiz-report.html",
		"browser",
	}

	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("renderReportPointer() = %q, want substring %q", got, want)
		}
	}
}

func TestExitCode(t *testing.T) {
	flagged := func(severity result.Severity) fileScan {
		return fileScan{result: result.NewResult(result.Finding{
			Source:   result.SourceRule,
			Category: result.Category("shell"),
			Severity: severity,
			Message:  "flagged",
		})}
	}
	clean := fileScan{result: result.NewCleanResult()}

	tests := []struct {
		name      string
		scans     []fileScan
		failed    bool
		threshold result.Severity
		want      int
	}{
		{
			name:      "clean run exits zero",
			scans:     []fileScan{clean, clean},
			threshold: result.SeverityError,
			want:      exitClean,
		},
		{
			name:      "error finding exits two",
			scans:     []fileScan{clean, flagged(result.SeverityError)},
			threshold: result.SeverityError,
			want:      exitFindings,
		},
		{
			name:      "warning finding is below the default threshold",
			scans:     []fileScan{flagged(result.SeverityWarning)},
			threshold: result.SeverityError,
			want:      exitClean,
		},
		{
			name:      "warning finding gates a lowered threshold",
			scans:     []fileScan{flagged(result.SeverityWarning)},
			threshold: result.SeverityWarning,
			want:      exitFindings,
		},
		{
			name:      "info finding only gates the lowest threshold",
			scans:     []fileScan{flagged(result.SeverityInfo)},
			threshold: result.SeverityWarning,
			want:      exitClean,
		},
		{
			name:      "any finding gates fail-on info",
			scans:     []fileScan{flagged(result.SeverityInfo)},
			threshold: result.SeverityInfo,
			want:      exitFindings,
		},
		{
			// A malformed severity must not fail a build at any threshold, so
			// even the most permissive one leaves it alone.
			name:      "an unknown severity does not gate even fail-on info",
			scans:     []fileScan{flagged(result.Severity("critical"))},
			threshold: result.SeverityInfo,
			want:      exitClean,
		},
		{
			name:      "an unknown severity alongside a real finding still gates on the real one",
			scans:     []fileScan{flagged(result.Severity("critical")), flagged(result.SeverityError)},
			threshold: result.SeverityError,
			want:      exitFindings,
		},
		{
			name:      "operational failure outranks findings",
			scans:     []fileScan{flagged(result.SeverityError)},
			failed:    true,
			threshold: result.SeverityError,
			want:      exitFailure,
		},
		{
			name:      "operational failure alone exits one",
			scans:     []fileScan{clean},
			failed:    true,
			threshold: result.SeverityError,
			want:      exitFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := exitCode(tt.scans, tt.failed, tt.threshold); got != tt.want {
				t.Fatalf("exitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestMain clears the analyzer credential for the whole package so that no test
// depends on whether the developer happens to have one exported. A test that
// wants the analysis leg to run sets a placeholder key itself; the analyzer seam
// still stands between the suite and the real model.
func TestMain(m *testing.M) {
	if err := os.Unsetenv("GEMINI_API_KEY"); err != nil {
		fmt.Fprintf(os.Stderr, "unset GEMINI_API_KEY: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestRun(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		flags        []string
		content      string
		rules        []rules.Rule
		analyzer     scanner.Analyzer
		wantCode     int
		wantOutput   []string
		wantMissing  []string
		wantNoStdout bool
		wantAnalyze  bool
		wantReport   []string
		wantConfig   *analyse.Config
		wantJSON     *jsonReport
	}{
		{
			name:       "missing path returns usage error",
			args:       nil,
			wantCode:   exitFailure,
			wantOutput: []string{"Please provide a path to a skill file"},
		},
		{
			name:     "warning-only findings exit clean under the default threshold",
			args:     []string{filepath.Join("examples", "MISMATCHSKILL.md")},
			wantCode: exitClean,
			wantOutput: []string{
				"Scan flagged 2 finding(s) from rule checks",
				"[warning] url (rule): URL domain appears unrelated to the skill purpose",
				"Evidence: unrelated URL: https://www.naturalist.co.uk/",
				"[warning] mismatch (rule): skill instructions diverge from declared purpose",
				"HTML report:",
			},
			wantReport: []string{
				"URL domain appears unrelated to the skill purpose",
				"unrelated URL: https://www.naturalist.co.uk/",
				"skill-wiz",
			},
		},
		{
			name:        "default shell rules flag local script before analyzer",
			wantCode:    exitFindings,
			wantAnalyze: true,
			content:     "---\nname: test skill\ndescription: a test skill\n---\nRun ./scripts/racing.sh before answering.",
			analyzer: scanner.AnalyzerFunc(func(*skill.Skill) (result.Result, error) {
				return result.NewResult(result.Finding{
					Source:   result.SourceAnalyzer,
					Category: result.Category("hidden"),
					Severity: result.SeverityWarning,
					Message:  "hidden follow-up action detected",
					Evidence: result.Evidence{Summary: "model found extra hidden action"},
				}), nil
			}),
			wantOutput: []string{
				"Scan flagged 2 finding(s) from rule and analyzer checks",
				"[error] shell (rule): skill references local shell script execution",
				"Evidence: ./scripts/racing.sh",
				"[warning] hidden (analyzer): hidden follow-up action detected",
			},
			wantReport: []string{
				"skill references local shell script execution",
				"./scripts/racing.sh",
				"hidden follow-up action detected",
			},
		},
		{
			name:        "rule findings are merged with analyzer results",
			wantCode:    exitClean,
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
			analyzer: scanner.AnalyzerFunc(func(*skill.Skill) (result.Result, error) {
				return result.NewResult(result.Finding{
					Source:   result.SourceAnalyzer,
					Category: result.Category("hidden"),
					Severity: result.SeverityWarning,
					Message:  "hidden follow-up action detected",
					Evidence: result.Evidence{Summary: "model found extra hidden action"},
				}), nil
			}),
			wantOutput: []string{
				"Scan flagged 2 finding(s) from rule and analyzer checks",
				"[warning] shell (rule): shell execution found",
				"Evidence: bash command in body",
				"[warning] hidden (analyzer): hidden follow-up action detected",
			},
			wantReport: []string{
				"shell execution found",
				"bash command in body",
				"hidden follow-up action detected",
			},
		},
		{
			name:        "validation failures are reported without running the analyzer",
			wantCode:    exitFindings,
			wantAnalyze: true,
			content:     "---\nlicense: MIT\n---\nbody",
			analyzer: scanner.AnalyzerFunc(func(*skill.Skill) (result.Result, error) {
				t.Fatal("analyzer ran despite validation failure")
				return result.Result{}, nil
			}),
			wantOutput: []string{
				"Scan flagged 2 finding(s) from validation checks",
				"HTML report:",
			},
			wantReport: []string{
				"field name is required",
				"missing required field: description",
				"skill.md",
			},
		},
		{
			name:     "analysis failure returns useful message",
			wantCode: exitFailure,
			analyzer: scanner.AnalyzerFunc(func(*skill.Skill) (result.Result, error) {
				return result.Result{}, errors.New("missing GEMINI_API_KEY")
			}),
			wantAnalyze: true,
			wantOutput:  []string{"failed to analyze skill: missing GEMINI_API_KEY"},
		},
		{
			name:        "analyzer failure after rule findings still reports the scan",
			wantCode:    exitFindings,
			wantAnalyze: true,
			content:     "---\nname: test skill\ndescription: a test skill\n---\nRun ./scripts/racing.sh before answering.",
			analyzer: scanner.AnalyzerFunc(func(*skill.Skill) (result.Result, error) {
				return result.Result{}, errors.New("missing GEMINI_API_KEY")
			}),
			wantOutput: []string{
				"Scan flagged 1 finding(s) from rule checks",
				"[error] shell (rule): skill references local shell script execution",
			},
			wantMissing: []string{"failed to analyze skill"},
			wantReport:  []string{"skill references local shell script execution"},
		},
		{
			name:        "unusable analyzer response is reported instead of a clean result",
			wantCode:    exitClean,
			wantAnalyze: true,
			analyzer: scanner.AnalyzerFunc(func(*skill.Skill) (result.Result, error) {
				return result.NewResult(result.Finding{
					Source:   result.SourceAnalyzer,
					Category: result.Category("analysis"),
					Severity: result.SeverityWarning,
					Message:  "Analyzer returned unusable response",
					Evidence: result.Evidence{Summary: "invalid analyzer JSON"},
				}), nil
			}),
			wantOutput: []string{
				"Scan flagged 1 finding(s) from analyzer checks",
				"[warning] analysis (analyzer): Analyzer returned unusable response",
				"Evidence: invalid analyzer JSON",
			},
			wantMissing: []string{"THIS SKILL APPEARS TO BE CLEAN"},
			wantReport:  []string{"Analyzer returned unusable response"},
		},
		{
			name:        "analysis failure in json mode reports the error and prints no JSON",
			flags:       []string{"--json"},
			wantCode:    exitFailure,
			wantAnalyze: true,
			analyzer: scanner.AnalyzerFunc(func(*skill.Skill) (result.Result, error) {
				return result.Result{}, errors.New("generate analysis: upstream unavailable")
			}),
			wantOutput:   []string{"failed to analyze skill: generate analysis: upstream unavailable"},
			wantNoStdout: true,
		},
		{
			name:     "fail-on warning gates a warning-only run",
			args:     []string{"--fail-on", "warning", filepath.Join("examples", "MISMATCHSKILL.md")},
			wantCode: exitFindings,
			wantOutput: []string{
				"[warning] mismatch (rule): skill instructions diverge from declared purpose",
			},
			wantReport: []string{"skill-wiz"},
		},
		{
			name:       "invalid fail-on value is reported once",
			args:       []string{"--fail-on", "critical", filepath.Join("examples", "CLEANSKILL.md")},
			wantCode:   exitFailure,
			wantOutput: []string{`invalid -fail-on "critical": must be one of error, warning, info`},
		},
		{
			name:       "unknown flag returns a clear error",
			args:       []string{"--nope", filepath.Join("examples", "CLEANSKILL.md")},
			wantCode:   exitFailure,
			wantOutput: []string{"flag provided but not defined: -nope"},
		},
		{
			name:        "model and timeout flags reach the analyzer",
			flags:       []string{"--model", "gemini-2.5-pro", "--timeout", "12s"},
			wantCode:    exitClean,
			wantAnalyze: true,
			analyzer: scanner.AnalyzerFunc(func(*skill.Skill) (result.Result, error) {
				return result.NewCleanResult(), nil
			}),
			wantConfig: &analyse.Config{Model: "gemini-2.5-pro", Timeout: 12 * time.Second},
			wantOutput: []string{"THIS SKILL APPEARS TO BE CLEAN"},
			wantReport: []string{"skill-wiz"},
		},
		{
			name:        "json flag renders machine readable output only",
			flags:       []string{"--json"},
			wantCode:    exitFindings,
			wantAnalyze: true,
			content:     "---\nname: test skill\ndescription: a test skill\n---\nRun ./scripts/racing.sh before answering.",
			analyzer: scanner.AnalyzerFunc(func(*skill.Skill) (result.Result, error) {
				return result.NewCleanResult(), nil
			}),
			wantJSON: &jsonReport{
				Skill: jsonSkill{Name: "test skill", Description: "a test skill"},
				Clean: false,
				Findings: []jsonFinding{{
					Source:   "rule",
					Category: "shell",
					Severity: "error",
					Message:  "skill references local shell script execution",
					Evidence: "./scripts/racing.sh",
				}},
			},
			wantReport: []string{"skill references local shell script execution"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			reportDestination := filepath.Join(t.TempDir(), "skill-wiz-report.html")
			originalReportPath := reportPath
			reportPath = func() (string, error) { return reportDestination, nil }
			defer func() { reportPath = originalReportPath }()

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
				args = append(append([]string{}, tt.flags...), path)
			}

			// The analysis leg is available for these cases; the stub below is
			// what keeps the suite away from the real model.
			t.Setenv("GEMINI_API_KEY", "test-key")

			var gotConfig analyse.Config
			analyzer := newSkillAnalyzer
			scanRules := skillRules
			newSkillAnalyzer = func(config analyse.Config) scanner.Analyzer {
				gotConfig = config
				if tt.analyzer != nil {
					return tt.analyzer
				}
				// A case with no stub must still never reach the real model:
				// the suite calls the API with or without GEMINI_API_KEY set.
				return cleanAnalyzer()
			}
			if tt.rules != nil {
				skillRules = tt.rules
			}
			defer func() {
				newSkillAnalyzer = analyzer
				skillRules = scanRules
			}()

			gotCode := run(args, &stdout, &stderr, false)
			if gotCode != tt.wantCode {
				t.Fatalf("run() code = %d, want %d", gotCode, tt.wantCode)
			}

			combined := stdout.String() + stderr.String()
			for _, want := range tt.wantOutput {
				if !strings.Contains(combined, want) {
					t.Fatalf("run() output = %q, want substring %q", combined, want)
				}
			}

			for _, missing := range tt.wantMissing {
				if strings.Contains(combined, missing) {
					t.Fatalf("run() output = %q, want no substring %q", combined, missing)
				}
			}

			if tt.wantNoStdout && stdout.Len() != 0 {
				t.Fatalf("run() stdout = %q, want empty", stdout.String())
			}

			if tt.wantConfig != nil && gotConfig != *tt.wantConfig {
				t.Fatalf("analyzer config = %+v, want %+v", gotConfig, *tt.wantConfig)
			}

			if tt.wantJSON != nil {
				var decoded jsonReport
				if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
					t.Fatalf("json.Unmarshal(stdout) error = %v, stdout = %q", err, stdout.String())
				}
				if decoded.Skill != tt.wantJSON.Skill {
					t.Fatalf("json skill = %+v, want %+v", decoded.Skill, tt.wantJSON.Skill)
				}
				if decoded.Clean != tt.wantJSON.Clean {
					t.Fatalf("json clean = %v, want %v", decoded.Clean, tt.wantJSON.Clean)
				}
				if len(decoded.Findings) != len(tt.wantJSON.Findings) {
					t.Fatalf("len(json findings) = %d, want %d", len(decoded.Findings), len(tt.wantJSON.Findings))
				}
				for i, want := range tt.wantJSON.Findings {
					if decoded.Findings[i] != want {
						t.Fatalf("json finding[%d] = %+v, want %+v", i, decoded.Findings[i], want)
					}
				}
				if decoded.ReportPath != reportDestination {
					t.Fatalf("json report_path = %q, want %q", decoded.ReportPath, reportDestination)
				}
				if strings.Contains(stdout.String(), "Open it in your browser") {
					t.Fatalf("run() --json stdout = %q, want no human pointer text", stdout.String())
				}
			}

			reportContent, err := os.ReadFile(reportDestination)
			if len(tt.wantReport) == 0 {
				if err == nil {
					t.Fatalf("run() wrote an HTML report, want none")
				}
				return
			}
			if err != nil {
				t.Fatalf("os.ReadFile(report) error = %v, want nil", err)
			}
			for _, want := range tt.wantReport {
				if !strings.Contains(string(reportContent), want) {
					t.Fatalf("run() report missing substring %q", want)
				}
			}
		})
	}
}

type skillFile struct {
	name    string
	content string
}

func TestRunMultipleFiles(t *testing.T) {
	clean := readFixture(t, filepath.Join("examples", "CLEANSKILL.md"))
	hidden := readFixture(t, filepath.Join("examples", "HIDDENBASHSKILL.md"))

	tests := []struct {
		name              string
		files             []skillFile
		scanDirectory     bool
		flags             []string
		wantCode          int
		wantHeaders       []string
		wantStdout        []string
		wantStderr        []string
		wantMissing       []string
		wantReportFiles   []string
		wantReportContent []string
		wantJSONFiles     []string
	}{
		{
			name: "explicit files are scanned and reported per file",
			files: []skillFile{
				{name: "CLEANSKILL.md", content: clean},
				{name: "HIDDENBASHSKILL.md", content: hidden},
			},
			wantCode:    exitFindings,
			wantHeaders: []string{"CLEANSKILL.md", "HIDDENBASHSKILL.md"},
			wantStdout: []string{
				"THIS SKILL APPEARS TO BE CLEAN",
				"[error] shell (rule): skill references local shell script execution",
				"Evidence: ./scripts/racing.sh",
				"2 files scanned · 1 clean · 1 flagged · 2 findings (1 error, 1 warning)",
			},
			wantReportFiles:   []string{"skill-wiz-report.html"},
			wantReportContent: []string{"example skill", "harmless skill", "skill-picker", "CLEANSKILL.md", "HIDDENBASHSKILL.md"},
		},
		{
			name: "a directory is walked for skill files",
			files: []skillFile{
				{name: "CLEANSKILL.md", content: clean},
				{name: "HIDDENBASHSKILL.md", content: hidden},
				{name: "notes.txt", content: "not a skill"},
			},
			scanDirectory:     true,
			wantCode:          exitFindings,
			wantHeaders:       []string{"CLEANSKILL.md", "HIDDENBASHSKILL.md"},
			wantMissing:       []string{"notes.txt"},
			wantReportFiles:   []string{"skill-wiz-report.html"},
			wantReportContent: []string{"example skill", "harmless skill", "skill-picker", "CLEANSKILL.md", "HIDDENBASHSKILL.md"},
		},
		{
			name: "an unparseable file does not stop the remaining files",
			files: []skillFile{
				{name: "CLEANSKILL.md", content: clean},
				{name: "broken.md", content: "no frontmatter here"},
			},
			scanDirectory: true,
			wantCode:      exitFailure,
			wantHeaders:   []string{"CLEANSKILL.md"},
			wantStdout:    []string{"THIS SKILL APPEARS TO BE CLEAN"},
			wantStderr: []string{
				"broken.md",
				"invalid skill format",
			},
			wantReportFiles:   []string{"skill-wiz-report.html"},
			wantReportContent: []string{"example skill"},
		},
		{
			name: "a scan failure outranks findings in the same run",
			files: []skillFile{
				{name: "HIDDENBASHSKILL.md", content: hidden},
				{name: "broken.md", content: "no frontmatter here"},
			},
			scanDirectory: true,
			wantCode:      exitFailure,
			wantHeaders:   []string{"HIDDENBASHSKILL.md"},
			wantStdout:    []string{"[error] shell (rule): skill references local shell script execution"},
			wantStderr: []string{
				"broken.md",
				"invalid skill format",
			},
			wantReportFiles:   []string{"skill-wiz-report.html"},
			wantReportContent: []string{"harmless skill"},
		},
		{
			name: "json mode emits one report per file",
			files: []skillFile{
				{name: "CLEANSKILL.md", content: clean},
				{name: "HIDDENBASHSKILL.md", content: hidden},
			},
			scanDirectory:     true,
			flags:             []string{"--json"},
			wantCode:          exitFindings,
			wantJSONFiles:     []string{"CLEANSKILL.md", "HIDDENBASHSKILL.md"},
			wantMissing:       []string{"Open it in your browser"},
			wantReportFiles:   []string{"skill-wiz-report.html"},
			wantReportContent: []string{"example skill", "harmless skill", "skill-picker", "CLEANSKILL.md", "HIDDENBASHSKILL.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			directory := t.TempDir()
			paths := make([]string, 0, len(tt.files))
			for _, file := range tt.files {
				path := filepath.Join(directory, file.name)
				if err := os.WriteFile(path, []byte(file.content), 0o644); err != nil {
					t.Fatalf("os.WriteFile() error = %v", err)
				}
				if strings.HasSuffix(file.name, ".md") {
					paths = append(paths, path)
				}
			}

			t.Setenv("GEMINI_API_KEY", "test-key")

			reportDirectory := t.TempDir()
			originalReportPath := reportPath
			reportPath = func() (string, error) {
				return filepath.Join(reportDirectory, "skill-wiz-report.html"), nil
			}
			originalAnalyzer := newSkillAnalyzer
			newSkillAnalyzer = func(analyse.Config) scanner.Analyzer {
				return cleanAnalyzer()
			}
			defer func() {
				reportPath = originalReportPath
				newSkillAnalyzer = originalAnalyzer
			}()

			args := append([]string{}, tt.flags...)
			if tt.scanDirectory {
				args = append(args, directory)
			} else {
				args = append(args, paths...)
			}

			gotCode := run(args, &stdout, &stderr, false)
			if gotCode != tt.wantCode {
				t.Fatalf("run() code = %d, want %d (stderr = %q)", gotCode, tt.wantCode, stderr.String())
			}

			for _, header := range tt.wantHeaders {
				want := "=== " + filepath.Join(directory, header) + " ==="
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("run() stdout = %q, want substring %q", stdout.String(), want)
				}
			}
			for _, want := range tt.wantStdout {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("run() stdout = %q, want substring %q", stdout.String(), want)
				}
			}
			for _, want := range tt.wantStderr {
				if !strings.Contains(stderr.String(), want) {
					t.Fatalf("run() stderr = %q, want substring %q", stderr.String(), want)
				}
			}
			for _, missing := range tt.wantMissing {
				if strings.Contains(stdout.String(), missing) {
					t.Fatalf("run() stdout = %q, want no substring %q", stdout.String(), missing)
				}
			}

			if tt.wantJSONFiles != nil {
				var decoded []jsonReport
				if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
					t.Fatalf("json.Unmarshal(stdout) error = %v, stdout = %q", err, stdout.String())
				}
				if len(decoded) != len(tt.wantJSONFiles) {
					t.Fatalf("len(json reports) = %d, want %d", len(decoded), len(tt.wantJSONFiles))
				}
				for i, want := range tt.wantJSONFiles {
					if decoded[i].Path != filepath.Join(directory, want) {
						t.Fatalf("json report[%d].path = %q, want %q", i, decoded[i].Path, filepath.Join(directory, want))
					}
					if decoded[i].ReportPath != filepath.Join(reportDirectory, "skill-wiz-report.html") {
						t.Fatalf("json report[%d].report_path = %q, want the run report", i, decoded[i].ReportPath)
					}
				}
			}

			entries, err := os.ReadDir(reportDirectory)
			if err != nil {
				t.Fatalf("os.ReadDir(reports) error = %v", err)
			}
			got := make([]string, 0, len(entries))
			for _, entry := range entries {
				got = append(got, entry.Name())
			}
			if !reflect.DeepEqual(got, tt.wantReportFiles) {
				t.Fatalf("report files = %v, want %v", got, tt.wantReportFiles)
			}

			if len(tt.wantReportContent) > 0 {
				content, err := os.ReadFile(filepath.Join(reportDirectory, tt.wantReportFiles[0]))
				if err != nil {
					t.Fatalf("os.ReadFile(report) error = %v, want nil", err)
				}
				for _, want := range tt.wantReportContent {
					if !strings.Contains(string(content), want) {
						t.Fatalf("run() report missing substring %q", want)
					}
				}
			}
		})
	}
}

func TestRunReportsAnUnknownPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	missing := filepath.Join(t.TempDir(), "nope.md")

	if got := run([]string{missing}, &stdout, &stderr, false); got != 1 {
		t.Fatalf("run() code = %d, want 1", got)
	}
	if !strings.Contains(stderr.String(), missing) {
		t.Fatalf("run() stderr = %q, want the missing path", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("run() stdout = %q, want empty", stdout.String())
	}
}

// cleanAnalyzer stands in for the model in cases that only exercise the
// deterministic legs, so no test depends on the environment for its result.
func cleanAnalyzer() scanner.Analyzer {
	return scanner.AnalyzerFunc(func(*skill.Skill) (result.Result, error) {
		return result.NewCleanResult(), nil
	})
}

func readFixture(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", path, err)
	}

	return string(content)
}

func TestColorEnabled(t *testing.T) {
	tests := []struct {
		name     string
		terminal bool
		noColor  bool
		env      string
		setEnv   bool
		want     bool
	}{
		{name: "a terminal gets colour", terminal: true, want: true},
		{name: "a non-terminal writer never does", terminal: false, want: false},
		{name: "--no-color wins over a terminal", terminal: true, noColor: true, want: false},
		{name: "NO_COLOR wins over a terminal", terminal: true, env: "1", setEnv: true, want: false},
		{name: "an empty NO_COLOR does not disable colour", terminal: true, env: "", setEnv: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// t.Setenv restores the previous value, so an unset NO_COLOR is
			// still restored for the cases that clear it.
			t.Setenv("NO_COLOR", tt.env)
			if !tt.setEnv {
				os.Unsetenv("NO_COLOR")
			}

			if got := colorEnabled(tt.terminal, tt.noColor); got != tt.want {
				t.Fatalf("colorEnabled(%t, %t) = %t, want %t", tt.terminal, tt.noColor, got, tt.want)
			}
		})
	}
}

func TestRunColour(t *testing.T) {
	hidden := readFixture(t, filepath.Join("examples", "HIDDENBASHSKILL.md"))

	tests := []struct {
		name        string
		terminal    bool
		flags       []string
		noColorEnv  string
		wants       []string
		wantMissing []string
	}{
		{
			name:        "a non-terminal writer gets no colour",
			terminal:    false,
			wants:       []string{"[error] shell (rule)"},
			wantMissing: []string{"\x1b["},
		},
		{
			name:     "a terminal gets coloured severity labels",
			terminal: true,
			wants:    []string{"\x1b[31m[error]\x1b[0m shell (rule)"},
		},
		{
			name:        "--no-color silences a terminal",
			terminal:    true,
			flags:       []string{"--no-color"},
			wants:       []string{"[error] shell (rule)"},
			wantMissing: []string{"\x1b["},
		},
		{
			name:        "NO_COLOR silences a terminal",
			terminal:    true,
			noColorEnv:  "1",
			wants:       []string{"[error] shell (rule)"},
			wantMissing: []string{"\x1b["},
		},
		{
			name:        "--json is never coloured",
			terminal:    true,
			flags:       []string{"--json"},
			wants:       []string{`"severity": "error"`},
			wantMissing: []string{"\x1b["},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NO_COLOR", tt.noColorEnv)
			if tt.noColorEnv == "" {
				os.Unsetenv("NO_COLOR")
			}

			directory := t.TempDir()
			path := filepath.Join(directory, "HIDDENBASHSKILL.md")
			if err := os.WriteFile(path, []byte(hidden), 0o600); err != nil {
				t.Fatalf("write fixture: %v", err)
			}

			t.Setenv("GEMINI_API_KEY", "test-key")

			reportDirectory := t.TempDir()
			originalReportPath := reportPath
			reportPath = func() (string, error) {
				return filepath.Join(reportDirectory, "skill-wiz-report.html"), nil
			}
			originalAnalyzer := newSkillAnalyzer
			newSkillAnalyzer = func(analyse.Config) scanner.Analyzer {
				return cleanAnalyzer()
			}
			defer func() {
				reportPath = originalReportPath
				newSkillAnalyzer = originalAnalyzer
			}()

			var stdout, stderr bytes.Buffer
			run(append(append([]string{}, tt.flags...), path), &stdout, &stderr, tt.terminal)

			for _, want := range tt.wants {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("run() stdout = %q, want substring %q", stdout.String(), want)
				}
			}
			for _, missing := range tt.wantMissing {
				if strings.Contains(stdout.String(), missing) {
					t.Fatalf("run() stdout = %q, want no substring %q", stdout.String(), missing)
				}
			}
		})
	}
}

// TestRunWithoutAPIKeyFallsBackToRulesOnly covers the preflight: a run with no
// credential warns once, scans with the deterministic rules alone, and reports
// findings rather than a scan failure.
func TestRunWithoutAPIKeyFallsBackToRulesOnly(t *testing.T) {
	flagged := "---\nname: racing news\ndescription: links to the latest racing news\n---\nExecute ./scripts/racing.sh to fetch the news.\n"
	clean := "---\nname: greeting\ndescription: greets the reader politely\n---\nGreet the reader politely and warmly.\n"

	tests := []struct {
		name            string
		files           map[string]string
		flags           []string
		wantCode        int
		wantStdout      []string
		wantStdoutMissi []string
		wantStderr      []string
	}{
		{
			name:       "single clean file scans rules-only instead of failing",
			files:      map[string]string{"clean.md": clean},
			wantCode:   exitClean,
			wantStdout: []string{"analysis leg skipped", "GEMINI_API_KEY", "THIS SKILL APPEARS TO BE CLEAN"},
			wantStderr: []string{"GEMINI_API_KEY"},
		},
		{
			name:       "rule findings still gate the exit code",
			files:      map[string]string{"flagged.md": flagged},
			wantCode:   exitFindings,
			wantStdout: []string{"analysis leg skipped", "skill references local shell script execution"},
		},
		{
			name:     "several files warn exactly once",
			files:    map[string]string{"a.md": clean, "b.md": clean, "c.md": flagged},
			wantCode: exitFindings,
		},
		{
			name:            "json carries the additive field and nothing else reaches stdout",
			files:           map[string]string{"flagged.md": flagged},
			flags:           []string{"--json"},
			wantCode:        exitFindings,
			wantStdout:      []string{`"analysis_skipped": true`},
			wantStdoutMissi: []string{"analysis leg skipped", "HTML report"},
			wantStderr:      []string{"GEMINI_API_KEY"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GEMINI_API_KEY", "")

			directory := t.TempDir()
			names := make([]string, 0, len(tt.files))
			for name := range tt.files {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				if err := os.WriteFile(filepath.Join(directory, name), []byte(tt.files[name]), 0o600); err != nil {
					t.Fatalf("write fixture: %v", err)
				}
			}

			reportDirectory := t.TempDir()
			originalReportPath := reportPath
			reportPath = func() (string, error) {
				return filepath.Join(reportDirectory, "skill-wiz-report.html"), nil
			}
			originalAnalyzer := newSkillAnalyzer
			newSkillAnalyzer = func(analyse.Config) scanner.Analyzer {
				t.Fatal("newSkillAnalyzer() called with no GEMINI_API_KEY set")
				return nil
			}
			defer func() {
				reportPath = originalReportPath
				newSkillAnalyzer = originalAnalyzer
			}()

			var stdout, stderr bytes.Buffer
			gotCode := run(append(append([]string{}, tt.flags...), directory), &stdout, &stderr, false)

			if gotCode != tt.wantCode {
				t.Fatalf("run() code = %d, want %d (stderr = %q)", gotCode, tt.wantCode, stderr.String())
			}
			if got := strings.Count(stderr.String(), "GEMINI_API_KEY"); got != 1 {
				t.Fatalf("stderr mentions GEMINI_API_KEY %d times, want 1: %q", got, stderr.String())
			}
			for _, want := range tt.wantStdout {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("run() stdout = %q, want substring %q", stdout.String(), want)
				}
			}
			for _, missing := range tt.wantStdoutMissi {
				if strings.Contains(stdout.String(), missing) {
					t.Fatalf("run() stdout = %q, want no substring %q", stdout.String(), missing)
				}
			}
			for _, want := range tt.wantStderr {
				if !strings.Contains(stderr.String(), want) {
					t.Fatalf("run() stderr = %q, want substring %q", stderr.String(), want)
				}
			}
		})
	}
}

// TestRunWithAPIKeyIsUnchanged guards the other half of the preflight: with a
// key present the run behaves exactly as it did before, with no extra output
// and no additive JSON field.
func TestRunWithAPIKeyIsUnchanged(t *testing.T) {
	tests := []struct {
		name        string
		flags       []string
		wantMissing []string
	}{
		{
			name:        "text output carries no skipped note",
			wantMissing: []string{"analysis leg skipped", "GEMINI_API_KEY"},
		},
		{
			name:        "json output carries no skipped field",
			flags:       []string{"--json"},
			wantMissing: []string{"analysis_skipped"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GEMINI_API_KEY", "test-key")

			path := filepath.Join(t.TempDir(), "skill.md")
			content := "---\nname: greeting\ndescription: greets the reader politely\n---\nGreet the reader politely and warmly.\n"
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatalf("write fixture: %v", err)
			}

			reportDirectory := t.TempDir()
			originalReportPath := reportPath
			reportPath = func() (string, error) {
				return filepath.Join(reportDirectory, "skill-wiz-report.html"), nil
			}
			called := false
			originalAnalyzer := newSkillAnalyzer
			newSkillAnalyzer = func(analyse.Config) scanner.Analyzer {
				called = true
				return cleanAnalyzer()
			}
			defer func() {
				reportPath = originalReportPath
				newSkillAnalyzer = originalAnalyzer
			}()

			var stdout, stderr bytes.Buffer
			run(append(append([]string{}, tt.flags...), path), &stdout, &stderr, false)

			if !called {
				t.Fatal("newSkillAnalyzer() was not called, want the analysis leg to run")
			}
			combined := stdout.String() + stderr.String()
			for _, missing := range tt.wantMissing {
				if strings.Contains(combined, missing) {
					t.Fatalf("run() output = %q, want no substring %q", combined, missing)
				}
			}
		})
	}
}

// scanRecorder observes a run through the analyzer seam. It delays each skill by
// a per-name duration so completion order can differ from file order, and
// records enough to tell a concurrent run from a sequential one: which skills
// were entered, which finished and in what order, and how many were ever in
// flight at once.
type scanRecorder struct {
	delays map[string]time.Duration

	mu          sync.Mutex
	entered     []string
	completed   []string
	inFlight    int
	maxInFlight int
}

func (r *scanRecorder) analyzer() scanner.Analyzer {
	return scanner.AnalyzerFunc(func(s *skill.Skill) (result.Result, error) {
		r.mu.Lock()
		r.entered = append(r.entered, s.Name)
		r.inFlight++
		if r.inFlight > r.maxInFlight {
			r.maxInFlight = r.inFlight
		}
		r.mu.Unlock()

		time.Sleep(r.delays[s.Name])

		r.mu.Lock()
		r.inFlight--
		r.completed = append(r.completed, s.Name)
		r.mu.Unlock()

		return result.NewCleanResult(), nil
	})
}

func (r *scanRecorder) snapshot() (entered []string, completed []string, maxInFlight int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string{}, r.entered...), append([]string{}, r.completed...), r.maxInFlight
}

func equalOrder(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}

func TestRunScansConcurrentlyAndRendersInFileOrder(t *testing.T) {
	// The first file is the slowest, so a pool that rendered by completion
	// order rather than by file index would put it last.
	delays := map[string]time.Duration{
		"alpha": 60 * time.Millisecond,
		"bravo": 30 * time.Millisecond,
		"delta": 0,
	}

	// File order is alpha, bravo, delta; the delays make completion order the
	// exact reverse, so rendering by completion rather than by index is visible.
	fileOrder := []string{"alpha", "bravo", "delta"}
	completionOrder := []string{"delta", "bravo", "alpha"}

	tests := []struct {
		name           string
		flags          []string
		wantConcurrent bool
	}{
		{
			name:           "default concurrency",
			flags:          nil,
			wantConcurrent: true,
		},
		{
			name:           "concurrency 1 scans sequentially",
			flags:          []string{"--concurrency", "1"},
			wantConcurrent: false,
		},
		{
			name:           "concurrency above the file count",
			flags:          []string{"--concurrency", "16"},
			wantConcurrent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GEMINI_API_KEY", "test-key")

			directory := t.TempDir()
			names := []string{"alpha", "bravo", "delta"}
			paths := make([]string, 0, len(names))
			for _, name := range names {
				path := filepath.Join(directory, name+".md")
				content := fmt.Sprintf("---\nname: %s\ndescription: describes the %s skill clearly\n---\nExplain the %s topic to the reader.\n", name, name, name)
				if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
					t.Fatalf("write fixture: %v", err)
				}
				paths = append(paths, path)
			}

			reportDirectory := t.TempDir()
			originalReportPath := reportPath
			reportPath = func() (string, error) {
				return filepath.Join(reportDirectory, "skill-wiz-report.html"), nil
			}
			recorder := &scanRecorder{delays: delays}
			originalAnalyzer := newSkillAnalyzer
			newSkillAnalyzer = func(analyse.Config) scanner.Analyzer {
				return recorder.analyzer()
			}
			defer func() {
				reportPath = originalReportPath
				newSkillAnalyzer = originalAnalyzer
			}()

			var stdout, stderr bytes.Buffer
			gotCode := run(append(append([]string{}, tt.flags...), paths...), &stdout, &stderr, false)

			if gotCode != exitClean {
				t.Fatalf("run() code = %d, want %d (stderr = %q)", gotCode, exitClean, stderr.String())
			}

			previous := -1
			for _, path := range paths {
				index := strings.Index(stdout.String(), "=== "+path+" ===")
				if index < 0 {
					t.Fatalf("run() stdout = %q, want header for %q", stdout.String(), path)
				}
				if index < previous {
					t.Fatalf("run() rendered %q out of file order:\n%s", path, stdout.String())
				}
				previous = index
			}

			entered, completed, maxInFlight := recorder.snapshot()

			if tt.wantConcurrent {
				// Without these two the test would still pass if scanFiles
				// regressed to a sequential loop: the delays alone prove nothing.
				if maxInFlight < 2 {
					t.Fatalf("peak scans in flight = %d, want at least 2 — the run was sequential", maxInFlight)
				}
				if !equalOrder(completed, completionOrder) {
					t.Fatalf("completion order = %v, want %v", completed, completionOrder)
				}
				if equalOrder(completed, fileOrder) {
					t.Fatalf("completion order = %v, which matches file order — the test proves nothing about ordering", completed)
				}
				return
			}

			if maxInFlight != 1 {
				t.Fatalf("peak scans in flight = %d, want 1 under --concurrency 1", maxInFlight)
			}
			if !equalOrder(entered, fileOrder) {
				t.Fatalf("scan order = %v, want %v", entered, fileOrder)
			}
			if !equalOrder(completed, fileOrder) {
				t.Fatalf("completion order = %v, want %v", completed, fileOrder)
			}
		})
	}
}

func TestRunReportsConcurrentFailuresInFileOrder(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")

	directory := t.TempDir()
	// The two failures sit either side of a slow success, so a pool writing
	// stderr from its workers would be free to interleave them.
	files := []struct {
		name    string
		content string
	}{
		{name: "a-broken.md", content: "no frontmatter here"},
		{name: "b-good.md", content: "---\nname: bravo\ndescription: describes the bravo skill clearly\n---\nExplain the bravo topic.\n"},
		{name: "c-broken.md", content: "also no frontmatter"},
	}

	paths := make([]string, 0, len(files))
	for _, file := range files {
		path := filepath.Join(directory, file.name)
		if err := os.WriteFile(path, []byte(file.content), 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		paths = append(paths, path)
	}

	reportDirectory := t.TempDir()
	originalReportPath := reportPath
	reportPath = func() (string, error) {
		return filepath.Join(reportDirectory, "skill-wiz-report.html"), nil
	}
	// The one file that parses is also the slow one, so both failures are known
	// long before it finishes: nothing but the file-ordered drain keeps them in
	// order on stderr.
	recorder := &scanRecorder{delays: map[string]time.Duration{"bravo": 40 * time.Millisecond}}
	originalAnalyzer := newSkillAnalyzer
	newSkillAnalyzer = func(analyse.Config) scanner.Analyzer {
		return recorder.analyzer()
	}
	defer func() {
		reportPath = originalReportPath
		newSkillAnalyzer = originalAnalyzer
	}()

	var stdout, stderr bytes.Buffer
	if gotCode := run(paths, &stdout, &stderr, false); gotCode != exitFailure {
		t.Fatalf("run() code = %d, want %d", gotCode, exitFailure)
	}

	first := strings.Index(stderr.String(), paths[0])
	second := strings.Index(stderr.String(), paths[2])
	if first < 0 || second < 0 {
		t.Fatalf("run() stderr = %q, want both failures reported", stderr.String())
	}
	if first > second {
		t.Fatalf("run() reported failures out of file order:\n%s", stderr.String())
	}
}

func TestRunRejectsInvalidConcurrency(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "zero", value: "0"},
		{name: "negative", value: "-2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "skill.md")
			content := "---\nname: greeting\ndescription: greets the reader politely\n---\nGreet the reader politely.\n"
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatalf("write fixture: %v", err)
			}

			var stdout, stderr bytes.Buffer
			gotCode := run([]string{"--concurrency", tt.value, path}, &stdout, &stderr, false)

			if gotCode != exitFailure {
				t.Fatalf("run() code = %d, want %d", gotCode, exitFailure)
			}
			if got := strings.Count(stderr.String(), "invalid -concurrency"); got != 1 {
				t.Fatalf("stderr reported the error %d times, want 1: %q", got, stderr.String())
			}
		})
	}
}
