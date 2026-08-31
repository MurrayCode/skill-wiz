// Package render turns scan results into the console output a person reads.
//
// It depends on the result model only — no skills, no rules, no analyzer, no
// filesystem, and never os.Stdout. Colour is decided by the caller and arrives
// as a Style value, which is what keeps colour out of tests that write to a
// buffer.
package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/murraycode/skill-wiz/result"
)

// Input is what the renderer needs about one scanned skill. It deliberately
// carries no parsed skill: nothing here reads one, and taking it would couple
// this package to skill.
type Input struct {
	Path   string
	Result result.Result
}

// AnalysisSkippedWarning is the one sentence describing a rules-only run. main
// writes it to stderr as a warning; AnalysisSkippedNote heads the console
// output with it, so a rules-only run cannot be mistaken for a complete one.
const AnalysisSkippedWarning = "Warning: analysis leg skipped — GEMINI_API_KEY is not set, so this run used the deterministic rules only."

// AnalysisSkippedNote heads a rules-only run.
func AnalysisSkippedNote() string {
	return AnalysisSkippedWarning + "\n\n"
}

// maxEvidenceRunes bounds an evidence summary in the console. The HTML report
// keeps the full text, so truncating here loses nothing.
const maxEvidenceRunes = 200

// ANSI colours for severity labels. Only the label is coloured, so the rest of
// a finding line stays greppable.
const (
	colorReset   = "\x1b[0m"
	colorError   = "\x1b[31m"
	colorWarning = "\x1b[33m"
	colorInfo    = "\x1b[36m"
)

var severityColor = map[result.Severity]string{
	result.SeverityError:   colorError,
	result.SeverityWarning: colorWarning,
	result.SeverityInfo:    colorInfo,
}

// Style carries the presentation decisions taken by the caller. Whether
// stdout is a terminal has to arrive as a value rather than be sniffed from
// inside this package: the tests write to buffers, and colour must stay absent
// there.
type Style struct {
	Color bool
}

// Scans renders every scan in order. A run over a single file renders
// exactly as it did before multi-file support; a run over several files heads
// each result with its path, so findings keep their file even when some of the
// files failed to scan.
func Scans(scans []Input, total int, style Style) string {
	var builder strings.Builder
	for index, scan := range scans {
		if total > 1 {
			if index > 0 {
				builder.WriteString("\n")
			}
			fmt.Fprintf(&builder, "=== %s ===\n", scan.Path)
		}

		builder.WriteString(Result(scan.Result, style))
	}

	rendered := builder.String()
	if total <= 1 {
		return rendered
	}

	// The clean verdict carries no newline of its own, so close the last
	// section before the tally rather than running on from it.
	if rendered != "" && !strings.HasSuffix(rendered, "\n") {
		rendered += "\n"
	}

	return rendered + "\n" + tally(scans) + "\n"
}

// tally closes a multi-file run with counts that match the findings
// printed above it. Aggregation by category belongs to the summary story, not
// here.
func tally(scans []Input) string {
	counts := make(map[result.Severity]int)
	clean, flagged, findings := 0, 0, 0
	for _, scan := range scans {
		if scan.Result.Clean() {
			clean++
		} else {
			flagged++
		}
		for _, finding := range scan.Result.Findings {
			findings++
			counts[finding.Severity]++
		}
	}

	tally := fmt.Sprintf("%s scanned · %d clean · %d flagged · %s",
		result.Pluralize(len(scans), "file"), clean, flagged, result.Pluralize(findings, "finding"))
	if breakdown := severityBreakdown(counts); breakdown != "" {
		tally += " (" + breakdown + ")"
	}

	return tally
}

// severityBreakdown lists the known severities that actually occurred, highest
// first. An unrecognised severity still counts towards the total; it just has
// no bucket to sit in.
func severityBreakdown(counts map[result.Severity]int) string {
	parts := make([]string, 0, 3)
	for _, severity := range result.Severities() {
		if count := counts[severity]; count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", count, severity))
		}
	}

	return strings.Join(parts, ", ")
}

// Result renders one scanned skill's findings, or the clean verdict.
func Result(scanResult result.Result, style Style) string {
	if scanResult.Clean() {
		return cleanMessage()
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "Scan flagged %d finding(s)", len(scanResult.Findings))
	if sources := scanResult.Sources(); len(sources) > 0 {
		fmt.Fprintf(&builder, " from %s checks", result.FormatSources(sources))
	}
	builder.WriteString("\n")
	for _, finding := range orderedFindings(scanResult.Findings) {
		fmt.Fprintf(&builder, "%s %s (%s): %s%s\n", severityLabel(finding.Severity, style), finding.Category, finding.Source, finding.Message, overrideNote(finding))
		if finding.Evidence.Summary != "" {
			fmt.Fprintf(&builder, "Evidence: %s\n", truncateEvidence(finding.Evidence.Summary))
		}
	}

	return builder.String()
}

// orderedFindings sorts a copy for display, highest severity first. The sort is
// stable so rule findings stay ahead of analyzer ones within a severity, and it
// works on a copy so result.Result — and therefore the JSON contract — keeps
// its merge order.
func orderedFindings(findings []result.Finding) []result.Finding {
	ordered := make([]result.Finding, len(findings))
	copy(ordered, findings)
	sort.SliceStable(ordered, func(i, j int) bool {
		return result.DisplayRank(ordered[i].Severity) < result.DisplayRank(ordered[j].Severity)
	})

	return ordered
}

// overrideNote marks a finding whose severity policy changed. It goes after the
// message rather than into the label so that the label stays exactly what it
// has always been — the severity that actually gates the exit code — and so a
// grep for "[error]" keeps working.
func overrideNote(finding result.Finding) string {
	if finding.OverriddenFrom == "" {
		return ""
	}

	return fmt.Sprintf(" (severity overridden from %s)", finding.OverriddenFrom)
}

func severityLabel(severity result.Severity, style Style) string {
	label := fmt.Sprintf("[%s]", severity)
	if !style.Color {
		return label
	}

	color, ok := severityColor[severity]
	if !ok {
		return label
	}

	return color + label + colorReset
}

// truncateEvidence keeps a long snippet from swamping the console. The HTML
// report still carries the full text.
func truncateEvidence(summary string) string {
	runes := []rune(summary)
	if len(runes) <= maxEvidenceRunes {
		return summary
	}

	return string(runes[:maxEvidenceRunes]) + "…"
}

func cleanMessage() string {
	return "THIS SKILL APPEARS TO BE CLEAN, PLEASE MANUALLY VERIFY TO BE SURE"
}
