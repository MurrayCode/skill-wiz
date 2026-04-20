package main

import (
	"strings"
	"testing"

	"github.com/murraycode/skill-wiz/result"
)

func TestRenderResult(t *testing.T) {
	tests := []struct {
		name   string
		result result.Result
		wants  []string
	}{
		{
			name:   "clean result renders clean message",
			result: result.NewCleanResult(),
			wants:  []string{"THIS SKILL APPEARS TO BE CLEAN, PLEASE MANUALLY VERIFY TO BE SURE"},
		},
		{
			name: "flagged result renders finding details",
			result: result.NewResult(result.Finding{
				Source:   result.SourceAnalyzer,
				Category: result.Category("analysis"),
				Severity: result.SeverityWarning,
				Message:  "Analyzer reported potential issues",
				Evidence: result.Evidence{Summary: "SUSPICIOUS: hidden shell execution"},
			}),
			wants: []string{
				"Scan flagged 1 finding(s)",
				"[warning] analysis: Analyzer reported potential issues",
				"Evidence: SUSPICIOUS: hidden shell execution",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderResult(tt.result)
			for _, want := range tt.wants {
				if !strings.Contains(got, want) {
					t.Fatalf("renderResult() = %q, want substring %q", got, want)
				}
			}
		})
	}
}
