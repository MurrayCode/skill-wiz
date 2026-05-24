package analyse

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/murraycode/skill-wiz/result"
	"github.com/murraycode/skill-wiz/skill"
	"google.golang.org/genai"
)

type stubGenerator struct {
	responseText string
	err          error
	prompt       string
}


func (s *stubGenerator) GenerateContent(_ context.Context, _ string, content []*genai.Content, _ *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	s.prompt = textFromContents(content)
	if s.err != nil {
		return nil, s.err
	}

	return &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{{
			Content: &genai.Content{Parts: []*genai.Part{{Text: s.responseText}}},
		}},
	}, nil
}

func textFromContents(contents []*genai.Content) string {
	var text string
	for _, content := range contents {
		text += textFromContent(content)
	}

	return text
}

func textFromContent(content *genai.Content) string {
	if content == nil {
		return ""
	}

	var text string
	for _, part := range content.Parts {
		text += part.Text
	}

	return text
}

func TestAnalyze(t *testing.T) {
	tests := []struct {
		name          string
		apiKey        string
		generator     *stubGenerator
		wantErr       string
		wantClean     bool
		wantFindings  int
		wantPrompt    string
		wantEvidence  string
	}{
		{
			name:    "missing api key returns explicit error",
			wantErr: "missing GEMINI_API_KEY",
		},
		{
			name:      "upstream failure returns wrapped error",
			apiKey:    "test-key",
			generator: &stubGenerator{err: errors.New("upstream unavailable")},
			wantErr:   "generate analysis: upstream unavailable",
			wantPrompt: "prompt text",
		},
		{
			name:         "successful analysis returns structured result",
			apiKey:       "test-key",
			generator:    &stubGenerator{responseText: "SUSPICIOUS: hidden shell execution"},
			wantClean:    false,
			wantFindings: 1,
			wantPrompt:   "prompt text",
			wantEvidence: "SUSPICIOUS: hidden shell execution",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GEMINI_API_KEY", tt.apiKey)
			generator := newGenerator
			if tt.generator != nil {
				newGenerator = func(context.Context, string) (contentGenerator, error) {
					return tt.generator, nil
				}
			}
			defer func() {
				newGenerator = generator
			}()

			got, err := Analyze("prompt text")
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Analyze() error = nil, want %q", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("Analyze() error = %q, want %q", err.Error(), tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("Analyze() error = %v, want nil", err)
			}

			if tt.generator != nil && tt.generator.prompt != tt.wantPrompt {
				t.Fatalf("generator prompt = %q, want %q", tt.generator.prompt, tt.wantPrompt)
			}

			if tt.wantErr != "" {
				return
			}

			if got.Clean() != tt.wantClean {
				t.Fatalf("Analyze().Clean() = %v, want %v", got.Clean(), tt.wantClean)
			}
			if len(got.Findings) != tt.wantFindings {
				t.Fatalf("len(Analyze().Findings) = %d, want %d", len(got.Findings), tt.wantFindings)
			}
			if tt.wantFindings > 0 && got.Findings[0].Evidence.Summary != tt.wantEvidence {
				t.Fatalf("Analyze().Findings[0].Evidence.Summary = %q, want %q", got.Findings[0].Evidence.Summary, tt.wantEvidence)
			}
		})
	}
}

func TestGeminiAnalyzerAnalyze(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")
	stub := &stubGenerator{responseText: "SUSPICIOUS: hidden shell execution"}
	generator := newGenerator
	newGenerator = func(context.Context, string) (contentGenerator, error) {
		return stub, nil
	}
	defer func() {
		newGenerator = generator
	}()

	got, err := (GeminiAnalyzer{}).Analyze(&skill.Skill{
		Description: "Checks for hidden shell execution",
		Body:        "Inspect the repository and report risks.",
	})
	if err != nil {
		t.Fatalf("GeminiAnalyzer.Analyze() error = %v, want nil", err)
	}
	if len(got.Findings) != 1 {
		t.Fatalf("len(GeminiAnalyzer.Analyze().Findings) = %d, want 1", len(got.Findings))
	}
	if !strings.Contains(stub.prompt, "***DESCRIPTION*** Checks for hidden shell execution") {
		t.Fatalf("analyzer prompt = %q, want description content", stub.prompt)
	}
	if !strings.Contains(stub.prompt, "***BODY*** Inspect the repository and report risks.") {
		t.Fatalf("analyzer prompt = %q, want body content", stub.prompt)
	}
}

func TestGeminiAnalyzerAnalyzeNilSkill(t *testing.T) {
	_, err := (GeminiAnalyzer{}).Analyze(nil)
	if err == nil {
		t.Fatal("GeminiAnalyzer.Analyze() error = nil, want non-nil")
	}
	if err.Error() != "nil skill" {
		t.Fatalf("GeminiAnalyzer.Analyze() error = %q, want %q", err.Error(), "nil skill")
	}
}

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
