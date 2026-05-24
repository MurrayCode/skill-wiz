package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/murraycode/skill-wiz/result"
	"github.com/murraycode/skill-wiz/skill"
)

func TestScanAggregatesFindingsInRuleOrder(t *testing.T) {
	var order []string
	s := &skill.Skill{Name: "test skill", Description: "desc"}

	ruleOne := RuleFunc(func(*skill.Skill) []result.Finding {
		order = append(order, "one")
		return []result.Finding{{
			Source:   result.SourceRule,
			Category: result.Category("first"),
			Severity: result.SeverityWarning,
			Message:  "first finding",
		}}
	})
	ruleTwo := RuleFunc(func(*skill.Skill) []result.Finding {
		order = append(order, "two")
		return []result.Finding{{
			Source:   result.SourceRule,
			Category: result.Category("second"),
			Severity: result.SeverityError,
			Message:  "second finding",
		}}
	})

	got := Scan(s, ruleOne, ruleTwo)

	if got.Clean() {
		t.Fatal("Scan().Clean() = true, want false")
	}
	if len(got.Findings) != 2 {
		t.Fatalf("len(Scan().Findings) = %d, want 2", len(got.Findings))
	}
	if got.Findings[0].Message != "first finding" {
		t.Fatalf("Scan().Findings[0].Message = %q, want %q", got.Findings[0].Message, "first finding")
	}
	if got.Findings[1].Message != "second finding" {
		t.Fatalf("Scan().Findings[1].Message = %q, want %q", got.Findings[1].Message, "second finding")
	}
	if len(order) != 2 || order[0] != "one" || order[1] != "two" {
		t.Fatalf("rule execution order = %v, want [one two]", order)
	}
}

func TestScanReturnsCleanResultWhenRulesDoNotReportFindings(t *testing.T) {
	s := &skill.Skill{Name: "test skill", Description: "desc"}

	got := Scan(s,
		RuleFunc(func(*skill.Skill) []result.Finding { return nil }),
		RuleFunc(func(*skill.Skill) []result.Finding { return []result.Finding{} }),
	)

	if !got.Clean() {
		t.Fatalf("Scan().Clean() = %v, want true", got.Clean())
	}
	if len(got.Findings) != 0 {
		t.Fatalf("len(Scan().Findings) = %d, want 0", len(got.Findings))
	}
}

func TestDefaultRules(t *testing.T) {
	tests := []struct {
		name         string
		skill        *skill.Skill
		wantClean    bool
		wantFindings int
		wantMessage  string
		wantSeverity result.Severity
		wantEvidence string
	}{
		{
			name:         "blank body is flagged",
			skill:        &skill.Skill{Name: "test skill", Description: "desc", Body: " \n\t "},
			wantClean:    false,
			wantFindings: 1,
			wantMessage:  "skill body is empty",
			wantSeverity: result.SeverityWarning,
			wantEvidence: "parsed skill body is blank",
		},
		{
			name:         "generic bash reference is flagged as warning",
			skill:        &skill.Skill{Name: "test skill", Description: "desc", Body: "Run bash -lc 'ls' to inspect the repo."},
			wantClean:    false,
			wantFindings: 1,
			wantMessage:  "skill references shell execution",
			wantSeverity: result.SeverityWarning,
			wantEvidence: "bash -lc 'ls'",
		},
		{
			name:         "local shell script execution is flagged as error",
			skill:        &skill.Skill{Name: "test skill", Description: "desc", Body: "Execute the local helper with ./scripts/deploy.sh before answering."},
			wantClean:    false,
			wantFindings: 1,
			wantMessage:  "skill references local shell script execution",
			wantSeverity: result.SeverityError,
			wantEvidence: "./scripts/deploy.sh",
		},
		{
			name:         "benign shell mention remains clean",
			skill:        &skill.Skill{Name: "test skill", Description: "desc", Body: "Explain what a Unix shell is and when to use one."},
			wantClean:    true,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := Scan(tt.skill, Default()...)

			if got.Clean() != tt.wantClean {
				t.Fatalf("Scan(Default()).Clean() = %v, want %v", got.Clean(), tt.wantClean)
			}
				if len(got.Findings) != tt.wantFindings {
					t.Fatalf("len(Scan(Default()).Findings) = %d, want %d", len(got.Findings), tt.wantFindings)
				}
				if tt.wantFindings > 0 && got.Findings[0].Message != tt.wantMessage {
					t.Fatalf("Scan(Default()).Findings[0].Message = %q, want %q", got.Findings[0].Message, tt.wantMessage)
				}
				if tt.wantFindings > 0 && got.Findings[0].Severity != tt.wantSeverity {
					t.Fatalf("Scan(Default()).Findings[0].Severity = %q, want %q", got.Findings[0].Severity, tt.wantSeverity)
				}
				if tt.wantFindings > 0 && got.Findings[0].Evidence.Summary != tt.wantEvidence {
					t.Fatalf("Scan(Default()).Findings[0].Evidence.Summary = %q, want %q", got.Findings[0].Evidence.Summary, tt.wantEvidence)
				}
			})
		}
}

func TestDefaultRulesFlagsHiddenBashFixture(t *testing.T) {
	path := filepath.Join("..", "examples", "HIDDENBASHSKILL.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}

	s, err := skill.Parse(string(content))
	if err != nil {
		t.Fatalf("skill.Parse() error = %v", err)
	}

	got := Scan(s, Default()...)
	if got.Clean() {
		t.Fatal("Scan(Default()).Clean() = true, want false")
	}

	finding := got.Findings[0]
	if finding.Message != "skill references local shell script execution" {
		t.Fatalf("finding.Message = %q, want %q", finding.Message, "skill references local shell script execution")
	}
	if finding.Evidence.Summary != "./scripts/f1.sh" {
		t.Fatalf("finding.Evidence.Summary = %q, want %q", finding.Evidence.Summary, "./scripts/f1.sh")
	}
}
