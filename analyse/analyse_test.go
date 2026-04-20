package analyse

import (
	"testing"

	"github.com/murraycode/skill-wiz/result"
)

func TestResultFromText(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		wantClean  bool
		wantCount  int
		wantSource result.Source
	}{
		{
			name:      "clean sentence returns clean result",
			text:      cleanMessage,
			wantClean: true,
			wantCount: 0,
		},
		{
			name:       "non clean text becomes analyzer finding",
			text:       "SUSPICIOUS: hidden shell execution",
			wantClean:  false,
			wantCount:  1,
			wantSource: result.SourceAnalyzer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resultFromText(tt.text)

			if got.Clean() != tt.wantClean {
				t.Fatalf("resultFromText(%q).Clean() = %v, want %v", tt.text, got.Clean(), tt.wantClean)
			}

			if len(got.Findings) != tt.wantCount {
				t.Fatalf("len(resultFromText(%q).Findings) = %d, want %d", tt.text, len(got.Findings), tt.wantCount)
			}

			if tt.wantCount == 0 {
				return
			}

			finding := got.Findings[0]
			if finding.Source != tt.wantSource {
				t.Fatalf("finding.Source = %q, want %q", finding.Source, tt.wantSource)
			}
			if finding.Category != result.Category("analysis") {
				t.Fatalf("finding.Category = %q, want %q", finding.Category, result.Category("analysis"))
			}
			if finding.Severity != result.SeverityWarning {
				t.Fatalf("finding.Severity = %q, want %q", finding.Severity, result.SeverityWarning)
			}
			if finding.Evidence.Summary != tt.text {
				t.Fatalf("finding.Evidence.Summary = %q, want %q", finding.Evidence.Summary, tt.text)
			}
		})
	}
}
