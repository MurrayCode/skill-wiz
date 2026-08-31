package result

import (
	"fmt"
	"strings"
)

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

// severityOrder is the one severity ordering in the codebase, most serious
// first. The exit-code gate, the console renderer, and the HTML report all
// derive their behaviour from it, so the three cannot drift apart. Nothing else
// should hold a severity table of its own.
var severityOrder = []Severity{SeverityError, SeverityWarning, SeverityInfo}

// Severities returns the known severities, most serious first. It returns a
// copy so a caller iterating them cannot reorder the source of truth.
func Severities() []Severity {
	ordered := make([]Severity, len(severityOrder))
	copy(ordered, severityOrder)

	return ordered
}

// Known reports whether a severity is one this package recognises, which is
// what makes it usable as a threshold.
func Known(severity Severity) bool {
	for _, known := range severityOrder {
		if known == severity {
			return true
		}
	}

	return false
}

// UnknownGateRank is the gating rank of a severity this package does not
// recognise. It sits strictly below every known severity — not level with info —
// because callers compare with >=: sharing info's rank would let a malformed
// finding fail a build under --fail-on info, which is exactly what the guarantee
// below rules out.
const UnknownGateRank = -1

// GateRank ranks a severity for comparison against a failure threshold: higher
// is more serious. An unrecognised severity ranks below every known one, so it
// can never fail a build on its own, at any threshold.
func GateRank(severity Severity) int {
	for index, known := range severityOrder {
		if known == severity {
			return len(severityOrder) - 1 - index
		}
	}

	return UnknownGateRank
}

// DisplayRank ranks a severity for display: lower sorts first. An unrecognised
// severity sorts last, so a malformed finding never outranks a real one.
//
// This is deliberately not GateRank inverted. The two disagree about the
// unknown case on purpose: gating must not let a malformed finding fail a
// build, while display must not let it jump the queue.
func DisplayRank(severity Severity) int {
	for index, known := range severityOrder {
		if known == severity {
			return index
		}
	}

	return len(severityOrder)
}

// FormatSources lists finding sources as readable prose: "a", "a and b", or
// "a, b, and c". Empty sources are dropped rather than rendered as a gap.
func FormatSources(sources []Source) string {
	parts := make([]string, 0, len(sources))
	for _, source := range sources {
		if source == "" {
			continue
		}
		parts = append(parts, string(source))
	}

	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
	}
}

// Pluralize renders a count with its noun, adding a trailing "s" for anything
// other than one.
func Pluralize(count int, noun string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, noun)
	}

	return fmt.Sprintf("%d %ss", count, noun)
}

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
