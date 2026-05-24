package rules

import (
	"fmt"
	"net/url"
	"regexp"
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

var weakMismatchOverlap = map[string]struct{}{
	"agent": {}, "date": {}, "information": {}, "inform": {}, "informs": {},
	"look": {}, "most": {}, "skill": {}, "where": {},
}

var urlPattern = regexp.MustCompile(`https?://[^\s<>()]+`)
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
	return []Rule{
		RuleFunc(emptyBodyRule),
		RuleFunc(shellExecutionRule),
		RuleFunc(unrelatedURLRule),
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
		if len(segmentKeywords) < 4 {
			continue
		}

		overlap := intersect(descriptionKeywords, segmentKeywords)
		if hasMeaningfulOverlap(overlap) {
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
	text = urlPattern.ReplaceAllString(text, " ")
	tokens := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	keywords := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		if len(token) < 4 && token != "f1" {
			continue
		}
		if _, skip := stopWords[token]; skip {
			continue
		}
		keywords[token] = struct{}{}
		if alias := keywordAlias(token); alias != "" {
			keywords[alias] = struct{}{}
		}
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
	normalized = urlPattern.ReplaceAllString(normalized, " ")
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

func unrelatedURLRule(s *skill.Skill) []result.Finding {
	urls := extractURLs(s.Body)
	if len(urls) == 0 {
		return nil
	}

	intentTokens := intentTokens(s, urls)
	if len(intentTokens) == 0 {
		return nil
	}

	findings := make([]result.Finding, 0)
	for _, rawURL := range urls {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			continue
		}

		host := normalizeHost(parsed.Hostname())
		if host == "" {
			continue
		}

		if hasOverlap(urlTokens(parsed), intentTokens) {
			continue
		}

		findings = append(findings, result.Finding{
			Source:   result.SourceRule,
			Category: result.Category("url"),
			Severity: result.SeverityWarning,
			Message:  "URL domain appears unrelated to the skill purpose",
			Evidence: result.Evidence{Summary: "unrelated URL: " + rawURL + " (domain: " + host + ")"},
		})
	}

	return findings
}

func extractURLs(text string) []string {
	matches := urlPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}

	urls := make([]string, 0, len(matches))
	for _, match := range matches {
		urls = append(urls, strings.TrimRight(match, ".,;:!?)]}"))
	}

	return urls
}

func intentTokens(s *skill.Skill, urls []string) map[string]struct{} {
	text := strings.Join([]string{s.Name, s.Description, s.Body}, " ")
	for _, rawURL := range urls {
		text = strings.ReplaceAll(text, rawURL, " ")
	}

	return tokenSet(text)
}

func urlTokens(parsed *url.URL) map[string]struct{} {
	parts := tokenSet(parsed.Hostname())
	for token := range tokenSet(parsed.EscapedPath()) {
		parts[token] = struct{}{}
	}

	return parts
}

func hasOverlap(left map[string]struct{}, right map[string]struct{}) bool {
	for token := range left {
		if _, ok := right[token]; ok {
			return true
		}
	}

	return false
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

func normalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimPrefix(host, "www.")
	return host
}

func tokenSet(text string) map[string]struct{} {
	tokens := splitTokens(text)
	set := make(map[string]struct{}, len(tokens)*2)
	for _, token := range tokens {
		if ignoredToken(token) {
			continue
		}
		set[token] = struct{}{}
		if singular := singularToken(token); singular != token && !ignoredToken(singular) {
			set[singular] = struct{}{}
		}
	}

	for i := 0; i < len(tokens)-1; i++ {
		if ignoredToken(tokens[i]) || ignoredToken(tokens[i+1]) {
			continue
		}
		set[tokens[i]+tokens[i+1]] = struct{}{}

		left := singularToken(tokens[i])
		right := singularToken(tokens[i+1])
		if !ignoredToken(left) && !ignoredToken(right) {
			set[left+right] = struct{}{}
		}
	}

	return set
}

func splitTokens(text string) []string {
	text = strings.ToLower(text)
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	tokens := make([]string, 0, len(fields)*2)
	for _, field := range fields {
		if field == "" {
			continue
		}
		tokens = append(tokens, field)

		parts := splitAlphaNumeric(field)
		if len(parts) > 1 {
			tokens = append(tokens, parts...)
		}
	}

	return tokens
}

func splitAlphaNumeric(token string) []string {
	if token == "" {
		return nil
	}

	parts := make([]string, 0, len(token))
	start := 0
	for i := 1; i < len(token); i++ {
		if unicode.IsLetter(rune(token[i-1])) == unicode.IsLetter(rune(token[i])) {
			continue
		}
		parts = append(parts, token[start:i])
		start = i
	}
	parts = append(parts, token[start:])
	return parts
}

func ignoredToken(token string) bool {
	if len(token) <= 1 {
		return true
	}

	switch token {
	case "the", "and", "for", "with", "from", "that", "this", "where", "when", "your", "into", "look", "find", "use", "any", "are", "get", "you", "url", "http", "https", "www", "com", "org", "net", "co", "uk":
		return true
	default:
		return false
	}
}

func hasMeaningfulOverlap(tokens []string) bool {
	for _, token := range tokens {
		if _, weak := weakMismatchOverlap[token]; !weak {
			return true
		}
	}

	return false
}

func keywordAlias(token string) string {
	switch token {
	case "f1":
		return "formula"
	default:
		return ""
	}
}

func singularToken(token string) string {
	if len(token) <= 3 {
		return token
	}
	if strings.HasSuffix(token, "ies") {
		return token[:len(token)-3] + "y"
	}
	if strings.HasSuffix(token, "sses") {
		return token[:len(token)-2]
	}
	if strings.HasSuffix(token, "s") && !strings.HasSuffix(token, "ss") {
		return token[:len(token)-1]
	}
	return token
}
