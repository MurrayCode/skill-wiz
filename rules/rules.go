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

// Rule identifiers. These are a public contract in the same way the JSON field
// names are: a policy file names a rule by its ID, so renaming one silently
// changes what every policy that mentions it enforces. Add IDs; do not rename
// them.
const (
	IDEmptyBody           = "empty-body"
	IDShellScript         = "shell-script"
	IDShellCommand        = "shell-command"
	IDUnrelatedURL        = "unrelated-url"
	IDDescriptionMismatch = "description-mismatch"
)

// Rule is one deterministic check with a stable identity.
type Rule interface {
	// ID names the rule for configuration. See the ID constants above.
	ID() string
	Check(*skill.Skill) []result.Finding
}

// RuleFunc adapts a plain check function to the Rule contract by pairing it
// with the identifier policy refers to.
type RuleFunc struct {
	RuleID  string
	Checker func(*skill.Skill) []result.Finding
}

func (f RuleFunc) ID() string {
	return f.RuleID
}

func (f RuleFunc) Check(s *skill.Skill) []result.Finding {
	if f.Checker == nil {
		return nil
	}

	return f.Checker(s)
}

func Default() []Rule {
	return []Rule{
		RuleFunc{RuleID: IDEmptyBody, Checker: emptyBodyRule},
		RuleFunc{RuleID: IDShellScript, Checker: shellScriptRule},
		RuleFunc{RuleID: IDShellCommand, Checker: shellCommandRule},
		RuleFunc{RuleID: IDUnrelatedURL, Checker: unrelatedURLRule},
		RuleFunc{RuleID: IDDescriptionMismatch, Checker: descriptionMismatchRule},
	}
}

// IDs lists the identifiers of a rule set in order, which is what a policy is
// validated against.
func IDs(ruleSet []Rule) []string {
	ids := make([]string, 0, len(ruleSet))
	for _, rule := range ruleSet {
		ids = append(ids, rule.ID())
	}

	return ids
}

// Scan runs every rule in order and stamps each finding with the ID of the rule
// that produced it. Stamping here rather than in the rules themselves means a
// rule cannot forget to, and cannot claim an identity other than the one it was
// registered under.
func Scan(s *skill.Skill, rules ...Rule) result.Result {
	findings := make([]result.Finding, 0)
	for _, rule := range rules {
		id := rule.ID()
		for _, finding := range rule.Check(s) {
			finding.RuleID = id
			findings = append(findings, finding)
		}
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
		segmentKeywords := keywordTokens(segment)
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
	return keywordTokens(urlPattern.ReplaceAllString(text, " "))
}

// keywordTokens is the tokenising half of keywords, for callers whose text has
// already had its URLs stripped.
func keywordTokens(text string) map[string]struct{} {
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

	intentTokens := intentTokens(s)
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

// intentTokens is the vocabulary the skill declares about itself, with the body's
// URLs removed so that a link cannot vouch for itself.
//
// The strip is one regexp pass over the body rather than one ReplaceAll pass per
// extracted URL, which is what makes the rule linear. It is applied to the body
// *before* joining, because that is the only text extractURLs ever drew from:
// stripping the joined string would also remove URLs the name or description
// declare, and those are part of the stated intent.
func intentTokens(s *skill.Skill) map[string]struct{} {
	body := urlPattern.ReplaceAllString(s.Body, " ")

	return tokenSet(strings.Join([]string{s.Name, s.Description, body}, " "))
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

func shellScriptRule(s *skill.Skill) []result.Finding {
	body := strings.TrimSpace(s.Body)
	if body == "" {
		return nil
	}

	matches := localShellScriptPattern.FindStringSubmatch(body)
	if len(matches) < 2 {
		return nil
	}

	return []result.Finding{{
		Source:   result.SourceRule,
		Category: result.Category("shell"),
		Severity: result.SeverityError,
		Message:  "skill references local shell script execution",
		Evidence: result.Evidence{Summary: matches[1]},
	}}
}

// shellCommandRule reports a loose bash or sh mention. It stands down entirely
// on a body that names a local script, because shellScriptRule has already
// reported that body at error severity and the line naming the script usually
// matches this pattern too — reporting it twice would double-count one problem.
// Deferring on the pattern rather than on whether the script rule ran keeps the
// two independent: disabling shell-script through policy removes that finding
// without a warning appearing in its place.
func shellCommandRule(s *skill.Skill) []result.Finding {
	body := strings.TrimSpace(s.Body)
	if body == "" {
		return nil
	}

	if localShellScriptPattern.MatchString(body) {
		return nil
	}

	for _, line := range strings.Split(body, "\n") {
		if !mentionsShellToken(line) {
			continue
		}

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

// mentionsShellToken is a necessary condition for shellCommandPattern to match:
// every alternative it accepts contains "sh". The check is allocation-free so
// that lines with no shell reference never reach the regexp engine, and never
// reach benignShellMention's lowercased copy either.
//
// It must fold exactly as the pattern does or it would silently drop real
// matches. The pattern is (?i), which is Unicode simple folding, not ASCII
// folding: "s" folds to {s, S, ſ} — U+017F LATIN SMALL LETTER LONG S — so
// "baſh script.txt" matches the pattern. "h" folds to {h, H} only. TestFoldSets
// pins both sets against unicode.SimpleFold.
func mentionsShellToken(line string) bool {
	for i := 0; i < len(line); i++ {
		width := foldedSWidth(line, i)
		if width == 0 {
			continue
		}
		if next := i + width; next < len(line) && (line[next] == 'h' || line[next] == 'H') {
			return true
		}
	}

	return false
}

// foldedSWidth returns the byte width of the "s" at index i under the pattern's
// case folding, or 0 when there is none. ſ is the only non-ASCII member of the
// set, and encodes as the two bytes 0xC5 0xBF.
func foldedSWidth(line string, i int) int {
	switch {
	case line[i] == 's' || line[i] == 'S':
		return 1
	case line[i] == 0xC5 && i+1 < len(line) && line[i+1] == 0xBF:
		return 2
	default:
		return 0
	}
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

	// Range over runes rather than bytes: indexing a string yields single bytes,
	// so classifying a UTF-8 continuation byte as a rune would split multi-byte
	// characters apart and put invalid UTF-8 into the token set.
	parts := make([]string, 0, len(token))
	start := 0
	first := true
	previousIsLetter := false
	for i, r := range token {
		isLetter := unicode.IsLetter(r)
		if first {
			first = false
			previousIsLetter = isLetter
			continue
		}
		if isLetter == previousIsLetter {
			continue
		}

		parts = append(parts, token[start:i])
		start = i
		previousIsLetter = isLetter
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
