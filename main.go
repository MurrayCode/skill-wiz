package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/murraycode/skill-wiz/analyse"
	"github.com/murraycode/skill-wiz/rules"
	"github.com/murraycode/skill-wiz/result"
	"github.com/murraycode/skill-wiz/scanner"
	"github.com/murraycode/skill-wiz/skill"
)

var skillAnalyzer scanner.Analyzer = analyse.GeminiAnalyzer{}
var skillRules = rules.Default()

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
	if validationResult := validationResultForSkill(s); !validationResult.Clean() {
		fmt.Fprint(stdout, renderResult(validationResult))
		return 0
	}
	output, err := scanner.Scanner{Rules: skillRules, Analyzer: skillAnalyzer}.Scan(s)
	if err != nil {
		fmt.Fprintf(stderr, "failed to analyze skill: %v\n", err)
		return 1
	}

	fmt.Fprint(stdout, renderResult(output))
	return 0
}

func renderResult(scanResult result.Result) string {
	if scanResult.Clean() {
		return analyseCleanMessage()
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "Scan flagged %d finding(s)\n", len(scanResult.Findings))
	for _, finding := range scanResult.Findings {
		fmt.Fprintf(&builder, "[%s] %s: %s\n", finding.Severity, finding.Category, finding.Message)
		if finding.Evidence.Summary != "" {
			fmt.Fprintf(&builder, "Evidence: %s\n", finding.Evidence.Summary)
		}
	}

	return builder.String()
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
