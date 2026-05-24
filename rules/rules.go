package rules

import (
	"strings"

	"github.com/murraycode/skill-wiz/result"
	"github.com/murraycode/skill-wiz/skill"
)

type Rule interface {
	Check(*skill.Skill) []result.Finding
}

type RuleFunc func(*skill.Skill) []result.Finding

func (f RuleFunc) Check(s *skill.Skill) []result.Finding {
	return f(s)
}

func Default() []Rule {
	return []Rule{RuleFunc(emptyBodyRule)}
}

func Scan(s *skill.Skill, rules ...Rule) result.Result {
	findings := make([]result.Finding, 0)
	for _, rule := range rules {
		findings = append(findings, rule.Check(s)...)
	}

	return result.NewResult(findings...)
}

func emptyBodyRule(s *skill.Skill) []result.Finding {
	if strings.TrimSpace(s.Body) != "" {
		return nil
	}

	return []result.Finding{{
		Source:   result.SourceRule,
		Category: result.Category("content"),
		Severity: result.SeverityWarning,
		Message:  "skill body is empty",
		Evidence: result.Evidence{Summary: "parsed skill body is blank"},
	}}
}
