package result

import (
	"fmt"
	"sort"
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
	// RuleID names the deterministic rule that produced this finding. It is
	// empty for validation and analyzer findings, which have no rule identity,
	// and it is how policy addresses a severity override at a finding.
	RuleID string
	// OverriddenFrom records the severity this finding carried before policy
	// changed it, and is empty when nothing did. Severity always holds the
	// effective value: a consumer that ignores this field still reads the
	// severity that gates the exit code.
	OverriddenFrom Severity
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

// findingKey hashes what a finding says, not where it came from. Source and
// RuleID are deliberately excluded so a rule and the model reporting the same
// issue collapse into one; rules merge first, so the rule's provenance wins.
// OverriddenFrom is excluded because policy is applied after Merge and must
// never change what collapses.
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

// Count is one row of a summary breakdown: a name and how many findings carried
// it.
type Count struct {
	Name  string
	Count int
}

// Summary aggregates a whole run. It counts what was reported, so findings a
// policy switched off are absent from every figure here — they are not findings
// any more by the time a Result reaches this package.
type Summary struct {
	FilesScanned int
	FilesClean   int
	FilesFlagged int
	FilesFailed  int
	Findings     int
	BySeverity   []Count
	ByCategory   []Count
	BySource     []Count
}

// Summarize aggregates the results of a run.
//
// Files that failed to scan are passed in as a count rather than inferred:
// this package knows nothing about paths or the filesystem, and teaching it
// would couple the leaf package to the CLI. Every result handed in is a file
// that was scanned.
func Summarize(results []Result, filesFailed int) Summary {
	summary := Summary{FilesScanned: len(results), FilesFailed: filesFailed}

	severities := make(map[string]int)
	categories := make(map[string]int)
	sources := make(map[string]int)
	for _, scanned := range results {
		if scanned.Clean() {
			summary.FilesClean++
		} else {
			summary.FilesFlagged++
		}
		for _, finding := range scanned.Findings {
			summary.Findings++
			severities[string(finding.Severity)]++
			categories[string(finding.Category)]++
			sources[string(finding.Source)]++
		}
	}

	summary.BySeverity = severityCounts(severities)
	summary.ByCategory = rankedCounts(categories)
	summary.BySource = rankedCounts(sources)

	return summary
}

// severityCounts lists the severities that occurred in the one order this
// package recognises, most serious first, with anything unrecognised ranked
// after them so a malformed finding never leads the breakdown.
func severityCounts(counts map[string]int) []Count {
	rows := make([]Count, 0, len(counts))
	for _, severity := range severityOrder {
		if count := counts[string(severity)]; count > 0 {
			rows = append(rows, Count{Name: string(severity), Count: count})
		}
	}

	unknown := make(map[string]int)
	for name, count := range counts {
		if !Known(Severity(name)) {
			unknown[name] = count
		}
	}

	return append(rows, rankedCounts(unknown)...)
}

// rankedCounts orders a breakdown by count descending, then name ascending, so
// repeated runs produce the same rows in the same order and a diff of two runs
// shows only what actually changed.
func rankedCounts(counts map[string]int) []Count {
	rows := make([]Count, 0, len(counts))
	for name, count := range counts {
		rows = append(rows, Count{Name: name, Count: count})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Count != rows[j].Count {
			return rows[i].Count > rows[j].Count
		}

		return rows[i].Name < rows[j].Name
	})

	return rows
}
