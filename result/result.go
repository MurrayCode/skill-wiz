package result

import "strings"

type Severity string
type Category string
type Source string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

const (
	SourceValidation Source = "validation"
	SourceRule       Source = "rule"
	SourceAnalyzer   Source = "analyzer"
)

type Evidence struct {
	Summary string
}

type Finding struct {
	Source   Source
	Category Category
	Severity Severity
	Message  string
	Evidence Evidence
}

type Result struct {
	Findings []Finding
}

func Merge(results ...Result) Result {
	merged := make([]Finding, 0)
	seen := make(map[string]struct{})
	for _, current := range results {
		for _, finding := range current.Findings {
			key := findingKey(finding)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, finding)
		}
	}

	return NewResult(merged...)
}

func NewCleanResult() Result {
	return Result{}
}

func NewResult(findings ...Finding) Result {
	if len(findings) == 0 {
		return NewCleanResult()
	}

	result := Result{Findings: make([]Finding, len(findings))}
	copy(result.Findings, findings)
	return result
}

func (r Result) Clean() bool {
	return len(r.Findings) == 0
}

func (r Result) Sources() []Source {
	if len(r.Findings) == 0 {
		return nil
	}

	sources := make([]Source, 0, len(r.Findings))
	seen := make(map[Source]struct{})
	for _, finding := range r.Findings {
		if _, ok := seen[finding.Source]; ok {
			continue
		}
		seen[finding.Source] = struct{}{}
		sources = append(sources, finding.Source)
	}

	return sources
}

func findingKey(finding Finding) string {
	parts := []string{
		string(finding.Category),
		string(finding.Severity),
		normalizeFindingText(finding.Message),
		normalizeFindingText(finding.Evidence.Summary),
	}

	return strings.Join(parts, "\x00")
}

func normalizeFindingText(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
