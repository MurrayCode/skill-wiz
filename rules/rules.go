package rules

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/murraycode/skill-wiz/result"
	"github.com/murraycode/skill-wiz/skill"
)

var stopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "any": {}, "are": {}, "as": {}, "at": {}, "by": {},
	"for": {}, "from": {}, "in": {}, "into": {}, "is": {}, "it": {}, "of": {}, "on": {},
	"or": {}, "the": {}, "this": {}, "to": {}, "up": {}, "where": {}, "with": {},
}

type Rule interface {
	Check(*skill.Skill) []result.Finding
}

type RuleFunc func(*skill.Skill) []result.Finding

func (f RuleFunc) Check(s *skill.Skill) []result.Finding {
	return f(s)
}

func Default() []Rule {
	return []Rule{
		RuleFunc(emptyBodyRule),
		RuleFunc(descriptionMismatchRule),
	}
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

func descriptionMismatchRule(s *skill.Skill) []result.Finding {
	descriptionKeywords := keywords(s.Description)
	if len(descriptionKeywords) < 2 {
		return nil
	}

	segments := bodySegments(s.Body)
	hasMatchingSection := false
	for _, segment := range segments {
		segmentKeywords := keywords(segment)
		if len(segmentKeywords) < 3 {
			continue
		}

		overlap := intersect(descriptionKeywords, segmentKeywords)
		if len(overlap) > 0 {
			hasMatchingSection = true
			continue
		}

		if !hasMatchingSection {
			continue
		}

		descriptionSample := sampleKeywords(descriptionKeywords, 5)
		segmentSample := sampleKeywords(segmentKeywords, 8)

		return []result.Finding{{
			Source:   result.SourceRule,
			Category: result.Category("mismatch"),
			Severity: result.SeverityWarning,
			Message:  "skill instructions diverge from declared purpose",
			Evidence: result.Evidence{Summary: fmt.Sprintf("description keywords [%s] conflict with instruction section [%s]", strings.Join(descriptionSample, " "), strings.Join(segmentSample, " "))},
		}}
	}

	return nil
}

func keywords(text string) map[string]struct{} {
	tokens := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	keywords := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		if len(token) < 4 {
			continue
		}
		if _, skip := stopWords[token]; skip {
			continue
		}
		keywords[token] = struct{}{}
	}

	return keywords
}

func intersect(left, right map[string]struct{}) []string {
	shared := make([]string, 0)
	for token := range left {
		if _, ok := right[token]; ok {
			shared = append(shared, token)
		}
	}
	sort.Strings(shared)
	return shared
}

func sampleKeywords(tokens map[string]struct{}, limit int) []string {
	words := make([]string, 0, len(tokens))
	for token := range tokens {
		words = append(words, token)
	}
	sort.Strings(words)
	if len(words) > limit {
		return words[:limit]
	}
	return words
}

func bodySegments(body string) []string {
	normalized := strings.ToLower(body)
	replacer := strings.NewReplacer(
		" and then ", "\n",
		" then ", "\n",
		" after ", "\n",
		" before ", "\n",
	)
	normalized = replacer.Replace(normalized)

	return strings.FieldsFunc(normalized, func(r rune) bool {
		switch r {
		case '\n', '.', '!', '?':
			return true
		default:
			return false
		}
	})
}
