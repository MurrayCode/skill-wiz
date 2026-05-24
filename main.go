package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/murraycode/skill-wiz/analyse"
	"github.com/murraycode/skill-wiz/result"
	"github.com/murraycode/skill-wiz/skill"
)

var analyzeSkill = analyse.Analyze

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
	prompt := fmt.Sprintf(`JOB: Your job is to analyze the following two bodys of text and flag any mismatches between the discription and the instructions and any suspicious or hidden behavior.
TASKS: Analyze the following two bodys of text. The first will be a description which will be the paragraph following the word ***DESCRIPTION***
The next will be body describing the actions the file describes an agent to take which will follow the word ***BODY***.
INPUT: ***DESCRIPTION*** %s. ***BODY*** %s. END OF INPUT
OUTPUT: Return a report on your findings under the following format. Return the sentence ***THIS SKILL APPEARS TO BE CLEAN, PLEASE MANUALLY VERIFY TO BE SURE*** if no mismatches, suspicious or hidden behavior are found. If you find any mismatches between the description and the instructions report back with the word ***MISMATCHES*** and your findings. If you find any suspicious behavior report back with the word ***SUSPICIOUS*** and description of your findings. If you find any hidden behaviour report back with the word ***HIDDEN*** and a description of the hidden behavior 
		`, s.Description, s.Body)
	output, err := analyzeSkill(prompt)
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
