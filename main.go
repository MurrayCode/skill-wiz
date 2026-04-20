package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/murraycode/skill-wiz/analyse"
	"github.com/murraycode/skill-wiz/result"
	"github.com/murraycode/skill-wiz/skill"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Please provide a path to a skill file")
	}

	path := os.Args[1]
	content, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}
	s, err := skill.Parse(string(content))
	if err != nil {
		log.Fatalf("Failed to parse skill: %v", err)
	}
	prompt := fmt.Sprintf(`JOB: Your job is to analyze the following two bodys of text and flag any mismatches between the discription and the instructions and any suspicious or hidden behavior.
TASKS: Analyze the following two bodys of text. The first will be a description which will be the paragraph following the word ***DESCRIPTION***
The next will be body describing the actions the file describes an agent to take which will follow the word ***BODY***.
INPUT: ***DESCRIPTION*** %s. ***BODY*** %s. END OF INPUT
OUTPUT: Return a report on your findings under the following format. Return the sentence ***THIS SKILL APPEARS TO BE CLEAN, PLEASE MANUALLY VERIFY TO BE SURE*** if no mismatches, suspicious or hidden behavior are found. If you find any mismatches between the description and the instructions report back with the word ***MISMATCHES*** and your findings. If you find any suspicious behavior report back with the word ***SUSPICIOUS*** and description of your findings. If you find any hidden behaviour report back with the word ***HIDDEN*** and a description of the hidden behavior 
		`, s.Description, s.Body)
	output, err := analyse.Analyze(prompt)
	if err != nil {
		log.Fatalf("failed to analyze skill: %v", err)
	}

	fmt.Print(renderResult(output))
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
