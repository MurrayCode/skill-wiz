package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/murraycode/skill-wiz/analyse"
	"github.com/murraycode/skill-wiz/report"
	"github.com/murraycode/skill-wiz/result"
	"github.com/murraycode/skill-wiz/rules"
	"github.com/murraycode/skill-wiz/scanner"
	"github.com/murraycode/skill-wiz/skill"
)

const reportFileName = "skill-wiz-report.html"

var skillAnalyzer scanner.Analyzer = analyse.GeminiAnalyzer{}
var skillRules = rules.Default()
var reportPath = defaultReportPath

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "Please provide a path to a skill file")
		return 1
	}

	path := args[0]
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "failed to read file: %v\n", err)
		return 1
	}
	s, err := skill.Parse(string(content))
	if err != nil {
		fmt.Fprintf(stderr, "Failed to parse skill: %v\n", err)
		return 1
	}
	// Validation short-circuits: a skill missing required metadata is never
	// handed to the rules or the analyzer.
	output := validationResultForSkill(s)
	if output.Clean() {
		output, err = scanner.Scanner{Rules: skillRules, Analyzer: skillAnalyzer}.Scan(s)
		if err != nil {
			fmt.Fprintf(stderr, "failed to analyze skill: %v\n", err)
			return 1
		}
	}

	fmt.Fprint(stdout, renderResult(output))
	writeReport(s, path, output, stdout, stderr)
	return 0
}

// writeReport saves the HTML report and points the user at it. A report that
// cannot be written is a warning, not a scan failure: the console output above
// already carries every finding.
func writeReport(s *skill.Skill, sourcePath string, scanResult result.Result, stdout io.Writer, stderr io.Writer) {
	destination, err := reportPath()
	if err != nil {
		fmt.Fprintf(stderr, "failed to resolve HTML report path: %v\n", err)
		return
	}

	if err := report.Write(destination, report.Input{
		SkillName:        s.Name,
		SkillDescription: s.Description,
		SourcePath:       sourcePath,
		GeneratedAt:      time.Now(),
		Result:           scanResult,
	}); err != nil {
		fmt.Fprintf(stderr, "failed to write HTML report: %v\n", err)
		return
	}

	fmt.Fprint(stdout, renderReportPointer(destination))
}

func renderReportPointer(destination string) string {
	return fmt.Sprintf("\nHTML report: %s\nOpen it in your browser: %s\n", destination, fileURL(destination))
}

func fileURL(destination string) string {
	absolute, err := filepath.Abs(destination)
	if err != nil {
		absolute = destination
	}

	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}).String()
}

func defaultReportPath() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	return filepath.Join(workingDirectory, reportFileName), nil
}

func renderResult(scanResult result.Result) string {
	if scanResult.Clean() {
		return analyseCleanMessage()
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "Scan flagged %d finding(s)", len(scanResult.Findings))
	if sources := scanResult.Sources(); len(sources) > 0 {
		fmt.Fprintf(&builder, " from %s checks", formatSources(sources))
	}
	builder.WriteString("\n")
	for _, finding := range scanResult.Findings {
		fmt.Fprintf(&builder, "[%s] %s (%s): %s\n", finding.Severity, finding.Category, finding.Source, finding.Message)
		if finding.Evidence.Summary != "" {
			fmt.Fprintf(&builder, "Evidence: %s\n", finding.Evidence.Summary)
		}
	}

	return builder.String()
}

func formatSources(sources []result.Source) string {
	parts := make([]string, 0, len(sources))
	for _, source := range sources {
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

func analyseCleanMessage() string {
	return "THIS SKILL APPEARS TO BE CLEAN, PLEASE MANUALLY VERIFY TO BE SURE"
}

func validationResultForSkill(s *skill.Skill) result.Result {
	if err := s.Validate(); err != nil {
		validationErrs, ok := err.(skill.ValidationErrors)
		if !ok {
			return result.NewResult(result.Finding{
				Source:   result.SourceValidation,
				Category: result.Category("metadata"),
				Severity: result.SeverityError,
				Message:  err.Error(),
			})
		}

		findings := make([]result.Finding, 0, len(validationErrs))
		for _, validationErr := range validationErrs {
			findings = append(findings, result.Finding{
				Source:   result.SourceValidation,
				Category: result.Category("metadata"),
				Severity: result.SeverityError,
				Message:  validationErr.Error(),
				Evidence: result.Evidence{Summary: fmt.Sprintf("missing required field: %s", validationErr.Field)},
			})
		}

		return result.NewResult(findings...)
	}

	return result.NewCleanResult()
}
