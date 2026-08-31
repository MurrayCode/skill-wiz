package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/murraycode/skill-wiz/result"
)

var defaultRuleIDs = []string{"empty-body", "shell-script", "shell-command", "unrelated-url", "description-mismatch"}

func TestDiscover(t *testing.T) {
	tests := []struct {
		name  string
		write bool
		want  bool
	}{
		{name: "a policy file in the directory is found", write: true, want: true},
		{name: "no policy file is not an error", write: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directory := t.TempDir()
			if tt.write {
				writePolicy(t, directory, "rules:\n  shell-script:\n    enabled: false\n")
			}

			got := Discover(directory)

			if tt.want && got != filepath.Join(directory, FileName) {
				t.Fatalf("Discover() = %q, want %q", got, filepath.Join(directory, FileName))
			}
			if !tt.want && got != "" {
				t.Fatalf("Discover() = %q, want empty", got)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     string
		wantEnabled map[string]bool
		wantRequire []string
	}{
		{
			name:        "a rule can be disabled",
			content:     "rules:\n  shell-script:\n    enabled: false\n",
			wantEnabled: map[string]bool{"shell-script": false, "shell-command": true},
		},
		{
			name:        "a rule can be enabled explicitly",
			content:     "rules:\n  shell-script:\n    enabled: true\n",
			wantEnabled: map[string]bool{"shell-script": true},
		},
		{
			name:        "require is read",
			content:     "require:\n  - shell-script\n  - unrelated-url\n",
			wantRequire: []string{"shell-script", "unrelated-url"},
		},
		{
			name:        "an empty file is a valid policy",
			content:     "",
			wantEnabled: map[string]bool{"shell-script": true},
		},
		{
			name:    "a malformed document is rejected",
			content: "rules: [this is not a mapping\n",
			wantErr: "parse policy",
		},
		{
			name:    "an unknown top level key is rejected",
			content: "ruels:\n  shell-script:\n    enabled: false\n",
			wantErr: "parse policy",
		},
		{
			name:    "an unknown rule key is rejected",
			content: "rules:\n  shell-script:\n    enabledd: false\n",
			wantErr: "parse policy",
		},
		{
			name:    "a second document is rejected",
			content: "require: []\n---\nrequire: []\n",
			wantErr: "single YAML document",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writePolicy(t, t.TempDir(), tt.content)

			got, err := Load(path)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Load() error = nil, want error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Load() error = %q, want substring %q", err, tt.wantErr)
				}
				if !strings.Contains(err.Error(), path) {
					t.Fatalf("Load() error = %q, want the policy path %q in the message", err, path)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v, want nil", err)
			}

			if !got.Loaded() {
				t.Fatal("Load().Loaded() = false, want true")
			}
			if got.Path() != path {
				t.Fatalf("Load().Path() = %q, want %q", got.Path(), path)
			}
			for id, want := range tt.wantEnabled {
				if enabled := got.Enabled(id); enabled != want {
					t.Fatalf("Load().Enabled(%q) = %v, want %v", id, enabled, want)
				}
			}
			if len(tt.wantRequire) > 0 {
				if err := got.Validate(defaultRuleIDs); err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestLoadReportsAMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)

	_, err := Load(path)

	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("Load() error = %q, want the policy path in the message", err)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "a policy naming only known rules is valid",
			content: "rules:\n  shell-script:\n    enabled: false\nrequire:\n  - unrelated-url\n",
		},
		{
			name:    "require naming an unknown rule is rejected",
			content: "require:\n  - shell-scrpt\n",
			wantErr: `require lists unknown rule "shell-scrpt"`,
		},
		{
			name:    "a rules entry naming an unknown rule is rejected",
			content: "rules:\n  shell-scrpt:\n    enabled: false\n",
			wantErr: `rules names unknown rule "shell-scrpt"`,
		},
		{
			name:    "requiring a rule the same policy disables is rejected",
			content: "rules:\n  shell-script:\n    enabled: false\nrequire:\n  - shell-script\n",
			wantErr: `require lists "shell-script" but the policy disables it`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writePolicy(t, t.TempDir(), tt.content)
			loaded, err := Load(path)
			if err != nil {
				t.Fatalf("Load() error = %v, want nil", err)
			}

			err = loaded.Validate(defaultRuleIDs)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() error = nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %q, want substring %q", err, tt.wantErr)
			}
			if !strings.Contains(err.Error(), path) {
				t.Fatalf("Validate() error = %q, want the policy path in the message", err)
			}
		})
	}
}

func TestZeroPolicyEnablesEverything(t *testing.T) {
	var none Policy

	if none.Loaded() {
		t.Fatal("Policy{}.Loaded() = true, want false")
	}
	if none.Path() != "" {
		t.Fatalf("Policy{}.Path() = %q, want empty", none.Path())
	}
	if err := none.Validate(defaultRuleIDs); err != nil {
		t.Fatalf("Policy{}.Validate() error = %v, want nil", err)
	}
	for _, id := range defaultRuleIDs {
		if !none.Enabled(id) {
			t.Fatalf("Policy{}.Enabled(%q) = false, want true", id)
		}
	}
}

func writePolicy(t *testing.T, directory string, content string) string {
	t.Helper()

	path := filepath.Join(directory, FileName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write policy: %v", err)
	}

	return path
}

func TestLoadProfile(t *testing.T) {
	const document = `rules:
  shell-script:
    enabled: false
  unrelated-url:
    enabled: false
require:
  - description-mismatch
profiles:
  ci:
    rules:
      shell-script:
        enabled: true
    require:
      - shell-script
      - description-mismatch
  local:
    rules:
      description-mismatch:
        enabled: false
`

	tests := []struct {
		name        string
		profile     string
		wantErr     string
		wantProfile string
		wantEnabled map[string]bool
	}{
		{
			name:        "no profile selects the base policy",
			wantProfile: "",
			wantEnabled: map[string]bool{
				"shell-script":         false,
				"unrelated-url":        false,
				"description-mismatch": true,
			},
		},
		{
			name:        "a profile overlays the base rule by rule",
			profile:     "ci",
			wantProfile: "ci",
			wantEnabled: map[string]bool{
				// The profile re-enables this one...
				"shell-script": true,
				// ...and says nothing about this one, so the base still applies.
				"unrelated-url":        false,
				"description-mismatch": true,
			},
		},
		{
			name:        "a profile can disable a rule the base leaves alone",
			profile:     "local",
			wantProfile: "local",
			wantEnabled: map[string]bool{
				"shell-script":         false,
				"unrelated-url":        false,
				"description-mismatch": false,
			},
		},
		{
			name:    "an unknown profile is rejected and lists the ones that exist",
			profile: "staging",
			wantErr: `unknown profile "staging" (available profiles: ci, local)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writePolicy(t, t.TempDir(), document)

			got, err := LoadProfile(path, tt.profile)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("LoadProfile() error = nil, want error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("LoadProfile() error = %q, want substring %q", err, tt.wantErr)
				}
				if !strings.Contains(err.Error(), path) {
					t.Fatalf("LoadProfile() error = %q, want the policy path in the message", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadProfile() error = %v, want nil", err)
			}

			if got.Profile() != tt.wantProfile {
				t.Fatalf("LoadProfile().Profile() = %q, want %q", got.Profile(), tt.wantProfile)
			}
			for id, want := range tt.wantEnabled {
				if enabled := got.Enabled(id); enabled != want {
					t.Fatalf("LoadProfile(%q).Enabled(%q) = %v, want %v", tt.profile, id, enabled, want)
				}
			}
		})
	}
}

func TestProfileReplacesRatherThanMergesTheBaseEntry(t *testing.T) {
	// The profile names the rule but says nothing about it, so the base entry
	// is replaced rather than merged into: the rule goes back to its default.
	const document = `rules:
  shell-script:
    enabled: false
profiles:
  local:
    rules:
      shell-script:
`

	path := writePolicy(t, t.TempDir(), document)

	base, err := LoadProfile(path, "")
	if err != nil {
		t.Fatalf("LoadProfile() error = %v, want nil", err)
	}
	overlaid, err := LoadProfile(path, "local")
	if err != nil {
		t.Fatalf("LoadProfile() error = %v, want nil", err)
	}

	if base.Enabled("shell-script") {
		t.Fatal(`base.Enabled("shell-script") = true, want false`)
	}
	if !overlaid.Enabled("shell-script") {
		t.Fatal(`profile "local" did not replace the base entry for "shell-script"`)
	}
}

func TestProfileRequireReplacesTheBaseList(t *testing.T) {
	const document = `rules:
  shell-script:
    enabled: false
require:
  - description-mismatch
profiles:
  ci:
    require:
      - unrelated-url
`

	path := writePolicy(t, t.TempDir(), document)

	overlaid, err := LoadProfile(path, "ci")
	if err != nil {
		t.Fatalf("LoadProfile() error = %v, want nil", err)
	}

	// The base required description-mismatch and disabled shell-script; the
	// profile's list replaces the base's outright, so validation now turns on
	// unrelated-url alone.
	if err := overlaid.Validate([]string{"shell-script", "description-mismatch"}); err == nil {
		t.Fatal("Validate() error = nil, want the profile's require list to be the one checked")
	}
	if err := overlaid.Validate(defaultRuleIDs); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestLoadProfileFromAPolicyWithNoProfiles(t *testing.T) {
	path := writePolicy(t, t.TempDir(), "rules:\n  shell-script:\n    enabled: false\n")

	_, err := LoadProfile(path, "ci")

	if err == nil {
		t.Fatal("LoadProfile() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "defines no profiles") {
		t.Fatalf("LoadProfile() error = %q, want it to say the policy defines no profiles", err)
	}
}

func TestApply(t *testing.T) {
	shellFinding := result.Finding{
		Source:   result.SourceRule,
		Category: result.Category("shell"),
		Severity: result.SeverityError,
		Message:  "skill references local shell script execution",
		Evidence: result.Evidence{Summary: "./scripts/racing.sh"},
		RuleID:   "shell-script",
	}
	mismatchFinding := result.Finding{
		Source:   result.SourceRule,
		Category: result.Category("mismatch"),
		Severity: result.SeverityWarning,
		Message:  "skill instructions diverge from declared purpose",
		RuleID:   "description-mismatch",
	}
	analyzerFinding := result.Finding{
		Source:   result.SourceAnalyzer,
		Category: result.Category("hidden"),
		Severity: result.SeverityWarning,
		Message:  "hidden follow-up action detected",
	}

	tests := []struct {
		name           string
		content        string
		want           []result.Severity
		wantOverridden []result.Severity
	}{
		{
			name:           "no policy leaves every severity alone",
			content:        "",
			want:           []result.Severity{result.SeverityError, result.SeverityWarning, result.SeverityWarning},
			wantOverridden: []result.Severity{"", "", ""},
		},
		{
			name:           "a severity can be lowered",
			content:        "rules:\n  shell-script:\n    severity: info\n",
			want:           []result.Severity{result.SeverityInfo, result.SeverityWarning, result.SeverityWarning},
			wantOverridden: []result.Severity{result.SeverityError, "", ""},
		},
		{
			name:           "a severity can be raised",
			content:        "rules:\n  description-mismatch:\n    severity: error\n",
			want:           []result.Severity{result.SeverityError, result.SeverityError, result.SeverityWarning},
			wantOverridden: []result.Severity{"", result.SeverityWarning, ""},
		},
		{
			name:           "an override that restates the default leaves no trace",
			content:        "rules:\n  shell-script:\n    severity: error\n",
			want:           []result.Severity{result.SeverityError, result.SeverityWarning, result.SeverityWarning},
			wantOverridden: []result.Severity{"", "", ""},
		},
		{
			name:           "off drops the finding",
			content:        "rules:\n  shell-script:\n    severity: off\n",
			want:           []result.Severity{result.SeverityWarning, result.SeverityWarning},
			wantOverridden: []result.Severity{"", ""},
		},
		{
			name:           "an analyzer finding is never overridden",
			content:        "rules:\n  shell-script:\n    severity: info\n  description-mismatch:\n    severity: info\n",
			want:           []result.Severity{result.SeverityInfo, result.SeverityInfo, result.SeverityWarning},
			wantOverridden: []result.Severity{result.SeverityError, result.SeverityWarning, ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writePolicy(t, t.TempDir(), tt.content)
			active, err := Load(path)
			if err != nil {
				t.Fatalf("Load() error = %v, want nil", err)
			}

			got := active.Apply(result.NewResult(shellFinding, mismatchFinding, analyzerFinding))

			if len(got.Findings) != len(tt.want) {
				t.Fatalf("len(Apply().Findings) = %d, want %d", len(got.Findings), len(tt.want))
			}
			for i, want := range tt.want {
				if got.Findings[i].Severity != want {
					t.Fatalf("Apply().Findings[%d].Severity = %q, want %q", i, got.Findings[i].Severity, want)
				}
				if got.Findings[i].OverriddenFrom != tt.wantOverridden[i] {
					t.Fatalf("Apply().Findings[%d].OverriddenFrom = %q, want %q", i, got.Findings[i].OverriddenFrom, tt.wantOverridden[i])
				}
			}
		})
	}
}

func TestApplyDoesNotChangeWhatMergeCollapses(t *testing.T) {
	// The rule and the model report the same thing, so Merge collapses them
	// into one finding with the rule's provenance. Applying an override must
	// not change that, which is why Apply runs after Merge and never before.
	ruleFinding := result.Finding{
		Source:   result.SourceRule,
		Category: result.Category("shell"),
		Severity: result.SeverityError,
		Message:  "skill references local shell script execution",
		Evidence: result.Evidence{Summary: "./scripts/racing.sh"},
		RuleID:   "shell-script",
	}
	analyzerFinding := ruleFinding
	analyzerFinding.Source = result.SourceAnalyzer
	analyzerFinding.RuleID = ""

	path := writePolicy(t, t.TempDir(), "rules:\n  shell-script:\n    severity: info\n")
	active, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	merged := result.Merge(result.NewResult(ruleFinding), result.NewResult(analyzerFinding))
	if len(merged.Findings) != 1 {
		t.Fatalf("len(Merge().Findings) = %d, want 1", len(merged.Findings))
	}

	got := active.Apply(merged)

	if len(got.Findings) != 1 {
		t.Fatalf("len(Apply(Merge()).Findings) = %d, want 1", len(got.Findings))
	}
	if got.Findings[0].Source != result.SourceRule {
		t.Fatalf("Apply(Merge()).Findings[0].Source = %q, want %q", got.Findings[0].Source, result.SourceRule)
	}
	if got.Findings[0].Severity != result.SeverityInfo {
		t.Fatalf("Apply(Merge()).Findings[0].Severity = %q, want %q", got.Findings[0].Severity, result.SeverityInfo)
	}
	if got.Findings[0].OverriddenFrom != result.SeverityError {
		t.Fatalf("Apply(Merge()).Findings[0].OverriddenFrom = %q, want %q", got.Findings[0].OverriddenFrom, result.SeverityError)
	}
}

func TestLoadRejectsAnUnknownSeverity(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "in the base policy",
			content: "rules:\n  shell-script:\n    severity: critical\n",
			wantErr: `rules.shell-script.severity "critical" is not one of error, warning, info, off`,
		},
		{
			name:    "in a profile that was not selected",
			content: "profiles:\n  ci:\n    rules:\n      shell-script:\n        severity: eror\n",
			wantErr: `profiles.ci.rules.shell-script.severity "eror" is not one of error, warning, info, off`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writePolicy(t, t.TempDir(), tt.content)

			_, err := Load(path)

			if err == nil {
				t.Fatalf("Load() error = nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Load() error = %q, want substring %q", err, tt.wantErr)
			}
			if !strings.Contains(err.Error(), path) {
				t.Fatalf("Load() error = %q, want the policy path in the message", err)
			}
		})
	}
}

func TestSeverityOverridesSurviveAProfileOverlay(t *testing.T) {
	const document = `rules:
  shell-script:
    severity: info
profiles:
  ci:
    rules:
      shell-script:
        severity: error
`

	path := writePolicy(t, t.TempDir(), document)
	finding := result.Finding{
		Source:   result.SourceRule,
		Category: result.Category("shell"),
		Severity: result.SeverityWarning,
		Message:  "shell",
		RuleID:   "shell-script",
	}

	tests := []struct {
		name    string
		profile string
		want    result.Severity
	}{
		{name: "the base lowers it", profile: "", want: result.SeverityInfo},
		{name: "the ci profile raises it", profile: "ci", want: result.SeverityError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			active, err := LoadProfile(path, tt.profile)
			if err != nil {
				t.Fatalf("LoadProfile() error = %v, want nil", err)
			}

			got := active.Apply(result.NewResult(finding))

			if len(got.Findings) != 1 {
				t.Fatalf("len(Apply().Findings) = %d, want 1", len(got.Findings))
			}
			if got.Findings[0].Severity != tt.want {
				t.Fatalf("Apply().Findings[0].Severity = %q, want %q", got.Findings[0].Severity, tt.want)
			}
		})
	}
}
