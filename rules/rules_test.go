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
	}{
		{
			name:         "blank body is flagged",
			skill:        &skill.Skill{Name: "test skill", Description: "desc", Body: " \n\t "},
			wantClean:    false,
			wantFindings: 1,
			wantMessage:  "skill body is empty",
		},
		{
			name:         "non blank body is clean",
			skill:        &skill.Skill{Name: "test skill", Description: "desc", Body: "do something useful"},
			wantClean:    true,
			wantFindings: 0,
		},
		{
			name: "mismatch example flags unrelated domain",
			skill: mustParseSkillFile(t, filepath.Join("..", "examples", "MISMATCHSKILL.md")),
			wantClean: false,
			wantFindings: 1,
			wantMessage: "URL domain appears unrelated to the skill purpose",
		},
		{
			name: "related urls stay clean",
			skill: &skill.Skill{
				Name: "formula one updates",
				Description: "Help the agent find current Formula 1 team and driver information",
				Body: "Check https://www.formula1.com/en/teams and https://www.formula1.com/en/drivers for the latest Formula 1 updates.",
			},
			wantClean: true,
			wantFindings: 0,
		},
		{
			name: "mixed related and unrelated urls only flag the unrelated domain",
			skill: &skill.Skill{
				Name: "formula one updates",
				Description: "Help the agent find current Formula 1 team and driver information",
				Body: "Use https://www.formula1.com/en/drivers for F1 details, then check https://birdwatching.example.com/hotspots for extra reading.",
			},
			wantClean: false,
			wantFindings: 1,
			wantMessage: "URL domain appears unrelated to the skill purpose",
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
		})
	}
}

func mustParseSkillFile(t *testing.T, path string) *skill.Skill {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}

	parsed, err := skill.Parse(string(content))
	if err != nil {
		t.Fatalf("skill.Parse(%q) error = %v", path, err)
	}

	return parsed
}
