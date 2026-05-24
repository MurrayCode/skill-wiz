package rules

import (
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
		wantMessages []string
		wantEvidence []string
	}{
		{
			name:         "blank body is flagged",
			skill:        &skill.Skill{Name: "test skill", Description: "desc", Body: " \n\t "},
			wantClean:    false,
			wantFindings: 1,
			wantMessages: []string{"skill body is empty"},
			wantEvidence: []string{"parsed skill body is blank"},
		},
		{
			name:         "body topic diverges from description",
			skill:        &skill.Skill{Name: "test skill", Description: "Provides formula one information to the agent", Body: "Look up Formula 1 race news and then give detailed bird watching holiday advice for rare seabirds."},
			wantClean:    false,
			wantFindings: 1,
			wantMessages: []string{"skill instructions diverge from declared purpose"},
			wantEvidence: []string{"description keywords [agent formula information provides] conflict with instruction section [advice bird detailed give holiday rare seabirds watching]"},
		},
		{
			name:         "non blank body is clean",
			skill:        &skill.Skill{Name: "test skill", Description: "desc", Body: "do something useful"},
			wantClean:    true,
			wantFindings: 0,
		},
		{
			name:         "matching description and body stay clean",
			skill:        &skill.Skill{Name: "test skill", Description: "Returns hello world in every answer", Body: "Add the words hello world to every answer you produce."},
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

				for i, want := range tt.wantMessages {
					if got.Findings[i].Message != want {
						t.Fatalf("Scan(Default()).Findings[%d].Message = %q, want %q", i, got.Findings[i].Message, want)
					}
					if got.Findings[i].Evidence.Summary != tt.wantEvidence[i] {
						t.Fatalf("Scan(Default()).Findings[%d].Evidence.Summary = %q, want %q", i, got.Findings[i].Evidence.Summary, tt.wantEvidence[i])
					}
				}
			})
		}
}
