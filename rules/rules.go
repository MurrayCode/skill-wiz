package rules

import (
	"regexp"
	"strings"

	"github.com/murraycode/skill-wiz/result"
	"github.com/murraycode/skill-wiz/skill"
)

var localShellScriptPattern = regexp.MustCompile(`(?i)(?:execute|run|invoke|launch)[^\n]*?(\./[^\s"'<>]+\.sh)`) 
var shellCommandPattern = regexp.MustCompile(`(?i)\b(?:bash|sh)\b(?:\s+-[a-z]+)?(?:\s+'[^'\n]+'|\s+"[^"\n]+"|\s+\./[^\s"'<>]+|\s+[^\s"'<>]+)?`)

type Rule interface {
	Check(*skill.Skill) []result.Finding
}

type RuleFunc func(*skill.Skill) []result.Finding

func (f RuleFunc) Check(s *skill.Skill) []result.Finding {
	return f(s)
}

func Default() []Rule {
	return []Rule{RuleFunc(emptyBodyRule), RuleFunc(shellExecutionRule)}
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

func shellExecutionRule(s *skill.Skill) []result.Finding {
	body := strings.TrimSpace(s.Body)
	if body == "" {
		return nil
	}

	if matches := localShellScriptPattern.FindStringSubmatch(body); len(matches) > 1 {
		return []result.Finding{{
			Source:   result.SourceRule,
			Category: result.Category("shell"),
			Severity: result.SeverityError,
			Message:  "skill references local shell script execution",
			Evidence: result.Evidence{Summary: matches[1]},
		}}
	}

	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if benignShellMention(trimmed) {
			continue
		}

		if match := shellCommandPattern.FindString(trimmed); match != "" {
			return []result.Finding{{
				Source:   result.SourceRule,
				Category: result.Category("shell"),
				Severity: result.SeverityWarning,
				Message:  "skill references shell execution",
				Evidence: result.Evidence{Summary: strings.TrimSpace(match)},
			}}
		}
	}

	return nil
}

func benignShellMention(line string) bool {
	lower := strings.ToLower(line)
	if !strings.Contains(lower, "shell") {
		return false
	}

	benignPhrases := []string{
		"what a unix shell is",
		"what a shell is",
		"explain what",
		"learn about shell",
	}
	for _, phrase := range benignPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}

	return false
}
