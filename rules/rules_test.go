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
		wantMessages []string
		wantSeverity []result.Severity
		wantEvidence []string
	}{
		{
			name:         "blank body is flagged",
			skill:        &skill.Skill{Name: "test skill", Description: "desc", Body: " \n\t "},
			wantClean:    false,
			wantFindings: 1,
			wantMessages: []string{"skill body is empty"},
			wantSeverity: []result.Severity{result.SeverityWarning},
			wantEvidence: []string{"parsed skill body is blank"},
		},
		{
			name:         "body topic diverges from description",
			skill:        &skill.Skill{Name: "test skill", Description: "Provides formula one information to the agent", Body: "Look up Formula 1 race news and then give detailed bird watching holiday advice for rare seabirds."},
			wantClean:    false,
			wantFindings: 1,
			wantMessages: []string{"skill instructions diverge from declared purpose"},
			wantSeverity: []result.Severity{result.SeverityWarning},
			wantEvidence: []string{"description keywords [agent formula information provides] conflict with instruction section [advice bird detailed give holiday rare seabirds watching]"},
		},
		{
			name:         "generic bash reference is flagged as warning",
			skill:        &skill.Skill{Name: "test skill", Description: "desc", Body: "Run bash -lc 'ls' to inspect the repo."},
			wantClean:    false,
			wantFindings: 1,
			wantMessages: []string{"skill references shell execution"},
			wantSeverity: []result.Severity{result.SeverityWarning},
			wantEvidence: []string{"bash -lc 'ls'"},
		},
		{
			name:         "local shell script execution is flagged as error",
			skill:        &skill.Skill{Name: "test skill", Description: "desc", Body: "Execute the local helper with ./scripts/deploy.sh before answering."},
			wantClean:    false,
			wantFindings: 1,
			wantMessages: []string{"skill references local shell script execution"},
			wantSeverity: []result.Severity{result.SeverityError},
			wantEvidence: []string{"./scripts/deploy.sh"},
		},
		{
			name:         "benign shell mention remains clean",
			skill:        &skill.Skill{Name: "test skill", Description: "desc", Body: "Explain what a Unix shell is and when to use one."},
			wantClean:    true,
			wantFindings: 0,
		},
		{
			name:         "matching description and body stay clean",
			skill:        &skill.Skill{Name: "test skill", Description: "Returns hello world in every answer", Body: "Add the words hello world to every answer you produce."},
			wantClean:    true,
			wantFindings: 0,
		},
		{
			name:         "mismatch example flags unrelated domain",
			skill:        mustParseSkillFile(t, filepath.Join("..", "examples", "MISMATCHSKILL.md")),
			wantClean:    false,
			wantFindings: 1,
			wantMessages: []string{"URL domain appears unrelated to the skill purpose"},
			wantSeverity: []result.Severity{result.SeverityWarning},
			wantEvidence: []string{
				"unrelated URL: https://www.naturalist.co.uk/?gad_source=1&gad_campaignid=261771380&gbraid=0AAAAADlv47Q-DKFV9Nkw-BLD0MAaHqtJZ&gclid=Cj0KCQiA-YvMBhDtARIsAHZuUzLR9JOhk9SuaBpqQ1USQek8o8hA-vnA2NoB5DRu_Uz5djQnmn6-jg8aAp0pEALw_wcB (domain: naturalist.co.uk)",
			},
		},
		{
			name: "related urls stay clean",
			skill: &skill.Skill{
				Name:        "formula one updates",
				Description: "Help the agent find current Formula 1 team and driver information",
				Body:        "Check https://www.formula1.com/en/teams and https://www.formula1.com/en/drivers for the latest Formula 1 updates.",
			},
			wantClean:    true,
			wantFindings: 0,
		},
		{
			name: "mixed related and unrelated urls only flag the unrelated domain",
			skill: &skill.Skill{
				Name:        "formula one updates",
				Description: "Help the agent find current Formula 1 team and driver information",
				Body:        "Use https://www.formula1.com/en/drivers for F1 details, then check https://birdwatching.example.com/hotspots for extra reading.",
			},
			wantClean:    false,
			wantFindings: 1,
			wantMessages: []string{"URL domain appears unrelated to the skill purpose"},
			wantSeverity: []result.Severity{result.SeverityWarning},
			wantEvidence: []string{"unrelated URL: https://birdwatching.example.com/hotspots (domain: birdwatching.example.com)"},
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
				if got.Findings[i].Severity != tt.wantSeverity[i] {
					t.Fatalf("Scan(Default()).Findings[%d].Severity = %q, want %q", i, got.Findings[i].Severity, tt.wantSeverity[i])
				}
				if got.Findings[i].Evidence.Summary != tt.wantEvidence[i] {
					t.Fatalf("Scan(Default()).Findings[%d].Evidence.Summary = %q, want %q", i, got.Findings[i].Evidence.Summary, tt.wantEvidence[i])
				}
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
