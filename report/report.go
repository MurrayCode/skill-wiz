// Package report renders scan results as a self-contained static HTML page.
//
// It depends on the result model only: findings arrive already merged, and this
// package decides nothing about what counts as a problem, only how to show it.
package report

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/murraycode/skill-wiz/result"
)

//go:embed template.html
var templateSource string

var pageTemplate = template.Must(template.New("report").Parse(templateSource))

const timestampLayout = "2006-01-02 15:04 MST"

// Input is everything the report needs about a single scanned skill.
type Input struct {
	SkillName        string
	SkillDescription string
	SourcePath       string
	GeneratedAt      time.Time
	Result           result.Result
	// AnalysisSkipped marks a result produced by the deterministic rules alone,
	// so the page cannot be read as a complete scan.
	AnalysisSkipped bool
}

// Write renders the report for a whole run and saves it to destination,
// creating parent directories when they are missing.
func Write(destination string, inputs ...Input) error {
	page, err := Render(inputs...)
	if err != nil {
		return err
	}

	if directory := filepath.Dir(destination); directory != "" {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return fmt.Errorf("create report directory: %w", err)
		}
	}
	if err := os.WriteFile(destination, []byte(page), 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	return nil
}

// Render returns the full HTML document for a run. Every scanned skill lands on
// the one page; which of them is on show is a reader-side choice.
func Render(inputs ...Input) (string, error) {
	if len(inputs) == 0 {
		return "", errors.New("render report: no scans")
	}

	var buffer bytes.Buffer
	if err := pageTemplate.Execute(&buffer, newPageView(inputs)); err != nil {
		return "", fmt.Errorf("render report: %w", err)
	}

	return buffer.String(), nil
}

type pageView struct {
	Title      string
	SkillCount int
	Multiple   bool
	Skills     []skillView
}

type skillView struct {
	ID               string
	Label            string
	SkillName        string
	SkillDescription string
	SourcePath       string
	GeneratedAt      string
	Sources          string
	AnalysisSkipped  bool
	Verdict          string
	VerdictDetail    string
	VerdictTone      string
	Counts           []countView
	Findings         []findingView
}

type countView struct {
	Severity string
	Label    string
	Count    int
}

type findingView struct {
	Severity string
	Category string
	Source   string
	Message  string
	Evidence string
	// OverriddenFrom is the severity a policy moved this finding away from,
	// empty when none did. Severity is always the effective one.
	OverriddenFrom string
}

func newPageView(inputs []Input) pageView {
	labels := pickerLabels(inputs)

	page := pageView{
		Title:      skillTitle(inputs[0]),
		SkillCount: len(inputs),
		Multiple:   len(inputs) > 1,
		Skills:     make([]skillView, 0, len(inputs)),
	}
	if page.Multiple {
		page.Title = fmt.Sprintf("%d skills", len(inputs))
	}

	for index, input := range inputs {
		skill := newSkillView(input)
		skill.ID = fmt.Sprintf("skill-%d", index+1)
		skill.Label = labels[index]
		page.Skills = append(page.Skills, skill)
	}

	return page
}

// pickerLabels names each skill in the dropdown. File names are enough until
// two scans share one, at which point every label falls back to the full path
// so that no two entries read the same.
func pickerLabels(inputs []Input) []string {
	seen := make(map[string]int, len(inputs))
	for _, input := range inputs {
		seen[filepath.Base(strings.TrimSpace(input.SourcePath))]++
	}

	labels := make([]string, 0, len(inputs))
	for _, input := range inputs {
		path := strings.TrimSpace(input.SourcePath)
		name := filepath.Base(path)

		display := name
		if path == "" {
			display = skillTitle(input)
		} else if seen[name] > 1 {
			display = path
		}

		labels = append(labels, fmt.Sprintf("%s — %s", display, tally(len(input.Result.Findings))))
	}

	return labels
}

func tally(count int) string {
	if count == 0 {
		return "no findings"
	}

	return result.Pluralize(count, "finding")
}

func newSkillView(input Input) skillView {
	findings := sortedFindings(input.Result.Findings)

	view := skillView{
		SkillName:        skillTitle(input),
		SkillDescription: strings.TrimSpace(input.SkillDescription),
		SourcePath:       input.SourcePath,
		GeneratedAt:      timestamp(input.GeneratedAt),
		Sources:          result.FormatSources(input.Result.Sources()),
		AnalysisSkipped:  input.AnalysisSkipped,
		Counts:           severityCounts(findings),
		Findings:         make([]findingView, 0, len(findings)),
	}

	for _, finding := range findings {
		view.Findings = append(view.Findings, findingView{
			Severity:       string(finding.Severity),
			Category:       string(finding.Category),
			Source:         string(finding.Source),
			Message:        finding.Message,
			Evidence:       strings.TrimSpace(finding.Evidence.Summary),
			OverriddenFrom: string(finding.OverriddenFrom),
		})
	}

	view.Verdict, view.VerdictDetail, view.VerdictTone = verdict(findings)

	return view
}

func verdict(findings []result.Finding) (headline string, detail string, tone string) {
	if len(findings) == 0 {
		return "No findings",
			"Nothing was flagged, but please manually verify this skill before trusting it.",
			"clean"
	}

	headline = result.Pluralize(len(findings), "finding")
	tone = string(findings[0].Severity)
	switch findings[0].Severity {
	case result.SeverityError:
		detail = "At least one finding is an error. Review the evidence below before installing this skill."
	case result.SeverityWarning:
		detail = "Review the evidence below and confirm the skill does only what it claims."
	default:
		detail = "Informational only, but worth a read before installing this skill."
		tone = "info"
	}

	return headline, detail, tone
}

func severityCounts(findings []result.Finding) []countView {
	counts := make(map[result.Severity]int)
	for _, finding := range findings {
		counts[finding.Severity]++
	}

	ordered := result.Severities()
	views := make([]countView, 0, len(ordered))
	for _, severity := range ordered {
		if counts[severity] == 0 {
			continue
		}
		views = append(views, countView{
			Severity: string(severity),
			Label:    string(severity),
			Count:    counts[severity],
		})
	}

	return views
}

// sortedFindings groups findings by severity for reading order while keeping
// the merge order the scanner produced within each severity.
func sortedFindings(findings []result.Finding) []result.Finding {
	sorted := make([]result.Finding, len(findings))
	copy(sorted, findings)
	sort.SliceStable(sorted, func(i, j int) bool {
		return result.DisplayRank(sorted[i].Severity) < result.DisplayRank(sorted[j].Severity)
	})

	return sorted
}

func skillTitle(input Input) string {
	if name := strings.TrimSpace(input.SkillName); name != "" {
		return name
	}
	if base := filepath.Base(strings.TrimSpace(input.SourcePath)); base != "." && base != string(filepath.Separator) {
		return base
	}

	return "Untitled skill"
}

func timestamp(generatedAt time.Time) string {
	if generatedAt.IsZero() {
		generatedAt = time.Now()
	}

	return generatedAt.Format(timestampLayout)
}
