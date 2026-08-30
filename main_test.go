package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
			want: options{paths: []string{"skill.md"}, model: analyse.DefaultModel, timeout: analyse.DefaultTimeout},
		},
		{
			name: "flags are applied before the path",
			args: []string{"--json", "--model", "gemini-2.5-pro", "--timeout", "15s", "skill.md"},
			want: options{paths: []string{"skill.md"}, json: true, model: "gemini-2.5-pro", timeout: 15 * time.Second},
		},
		{
			name: "single dash flags are accepted",
			args: []string{"-json", "-timeout=5s", "skill.md"},
			want: options{paths: []string{"skill.md"}, json: true, model: analyse.DefaultModel, timeout: 5 * time.Second},
		},
		{
			name: "every positional argument becomes a path",
			args: []string{"first.md", "skills", "second.md"},
			want: options{
				paths:   []string{"first.md", "skills", "second.md"},
				model:   analyse.DefaultModel,
				timeout: analyse.DefaultTimeout,
			},
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

func TestRenderScans(t *testing.T) {
	clean := fileScan{
		path:   filepath.Join("examples", "CLEANSKILL.md"),
		result: result.NewCleanResult(),
	}
	flagged := fileScan{
		path: filepath.Join("examples", "HIDDENBASHSKILL.md"),
		result: result.NewResult(result.Finding{
			Source:   result.SourceRule,
			Category: result.Category("shell"),
			Severity: result.SeverityError,
			Message:  "skill references local shell script execution",
			Evidence: result.Evidence{Summary: "./scripts/racing.sh"},
		}),
	}

	tests := []struct {
		name        string
		scans       []fileScan
		total       int
		wants       []string
		wantMissing []string
	}{
		{
			name:        "a single file is rendered without a path header",
			scans:       []fileScan{clean},
			total:       1,
			wants:       []string{"THIS SKILL APPEARS TO BE CLEAN"},
			wantMissing: []string{"===", "HTML report:"},
		},
		{
			name:  "several files are headed by their path",
			scans: []fileScan{clean, flagged},
			total: 2,
			wants: []string{
				"=== " + clean.path + " ===",
				"THIS SKILL APPEARS TO BE CLEAN",
				"=== " + flagged.path + " ===",
				"[error] shell (rule): skill references local shell script execution",
			},
		},
		{
			name:  "a surviving scan keeps its path when another file failed",
			scans: []fileScan{flagged},
			total: 2,
			wants: []string{"=== " + flagged.path + " ==="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderScans(tt.scans, tt.total)

			for _, want := range tt.wants {
				if !strings.Contains(got, want) {
					t.Fatalf("renderScans() = %q, want substring %q", got, want)
				}
			}
			for _, missing := range tt.wantMissing {
				if strings.Contains(got, missing) {
					t.Fatalf("renderScans() = %q, want no substring %q", got, missing)
				}
			}
		})
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
		"Scan flagged 2 finding(s) from validation checks",
		"[error] metadata (validation): field name is required",
		"Evidence: missing required field: name",
		"[error] metadata (validation): field description is required",
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
			got := renderResult(tt.result)
			for _, want := range tt.wants {
				if !strings.Contains(got, want) {
					t.Fatalf("renderResult() = %q, want substring %q", got, want)
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
			wantCode:   1,
			wantOutput: []string{"Please provide a path to a skill file"},
		},
		{
			name:     "mismatch example is flagged by rules before analyzer",
			args:     []string{filepath.Join("examples", "MISMATCHSKILL.md")},
			wantCode: 0,
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
			wantCode:    0,
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
			wantCode:    0,
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
			wantCode:    0,
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
			wantCode: 1,
			analyzer: scanner.AnalyzerFunc(func(*skill.Skill) (result.Result, error) {
				return result.Result{}, errors.New("missing GEMINI_API_KEY")
			}),
			wantAnalyze: true,
			wantOutput:  []string{"failed to analyze skill: missing GEMINI_API_KEY"},
		},
		{
			name:        "analyzer failure after rule findings still reports the scan",
			wantCode:    0,
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
			wantCode:    0,
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
			wantCode:    1,
			wantAnalyze: true,
			analyzer: scanner.AnalyzerFunc(func(*skill.Skill) (result.Result, error) {
				return result.Result{}, errors.New("generate analysis: upstream unavailable")
			}),
			wantOutput:   []string{"failed to analyze skill: generate analysis: upstream unavailable"},
			wantNoStdout: true,
		},
		{
			name:       "unknown flag returns a clear error",
			args:       []string{"--nope", filepath.Join("examples", "CLEANSKILL.md")},
			wantCode:   1,
			wantOutput: []string{"flag provided but not defined: -nope"},
		},
		{
			name:        "model and timeout flags reach the analyzer",
			flags:       []string{"--model", "gemini-2.5-pro", "--timeout", "12s"},
			wantCode:    0,
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
			wantCode:    0,
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

			var gotConfig analyse.Config
			analyzer := newSkillAnalyzer
			scanRules := skillRules
			newSkillAnalyzer = func(config analyse.Config) scanner.Analyzer {
				gotConfig = config
				if tt.analyzer != nil {
					return tt.analyzer
				}
				return analyzer(config)
			}
			if tt.rules != nil {
				skillRules = tt.rules
			}
			defer func() {
				newSkillAnalyzer = analyzer
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
			wantCode:    0,
			wantHeaders: []string{"CLEANSKILL.md", "HIDDENBASHSKILL.md"},
			wantStdout: []string{
				"THIS SKILL APPEARS TO BE CLEAN",
				"[error] shell (rule): skill references local shell script execution",
				"Evidence: ./scripts/racing.sh",
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
			wantCode:          0,
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
			wantCode:      1,
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
			name: "json mode emits one report per file",
			files: []skillFile{
				{name: "CLEANSKILL.md", content: clean},
				{name: "HIDDENBASHSKILL.md", content: hidden},
			},
			scanDirectory:     true,
			flags:             []string{"--json"},
			wantCode:          0,
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

			reportDirectory := t.TempDir()
			originalReportPath := reportPath
			reportPath = func() (string, error) {
				return filepath.Join(reportDirectory, "skill-wiz-report.html"), nil
			}
			originalAnalyzer := newSkillAnalyzer
			newSkillAnalyzer = func(analyse.Config) scanner.Analyzer {
				return scanner.AnalyzerFunc(func(*skill.Skill) (result.Result, error) {
					return result.NewCleanResult(), nil
				})
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

			gotCode := run(args, &stdout, &stderr)
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

	if got := run([]string{missing}, &stdout, &stderr); got != 1 {
		t.Fatalf("run() code = %d, want 1", got)
	}
	if !strings.Contains(stderr.String(), missing) {
		t.Fatalf("run() stderr = %q, want the missing path", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("run() stdout = %q, want empty", stdout.String())
	}
}

func readFixture(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", path, err)
	}

	return string(content)
}
