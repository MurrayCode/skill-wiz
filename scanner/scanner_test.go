package scanner

import (
	"errors"
	"testing"

	"github.com/murraycode/skill-wiz/result"
	"github.com/murraycode/skill-wiz/rules"
	"github.com/murraycode/skill-wiz/skill"
)

type stubAnalyzer struct {
	called bool
	result result.Result
	err    error
}

func (s *stubAnalyzer) Analyze(*skill.Skill) (result.Result, error) {
	s.called = true
	if s.err != nil {
		return result.Result{}, s.err
	}

	return s.result, nil
}

func TestScanRunsWithoutAnalyzer(t *testing.T) {
	s := &skill.Skill{Name: "test skill", Description: "a test skill", Body: "body"}
	scanner := Scanner{
		Rules: []rules.Rule{
			rules.RuleFunc(func(*skill.Skill) []result.Finding { return nil }),
		},
	}

	got, err := scanner.Scan(s)
	if err != nil {
		t.Fatalf("Scan() error = %v, want nil", err)
	}
	if !got.Clean() {
		t.Fatalf("Scan().Clean() = %v, want true", got.Clean())
	}
}

func TestScanCallsInjectedAnalyzerWhenRulesAreClean(t *testing.T) {
	s := &skill.Skill{Name: "test skill", Description: "a test skill", Body: "body"}
	analyzer := &stubAnalyzer{result: result.NewResult(result.Finding{
		Source:   result.SourceAnalyzer,
		Category: result.Category("analysis"),
		Severity: result.SeverityWarning,
		Message:  "Analyzer reported potential issues",
		Evidence: result.Evidence{Summary: "SUSPICIOUS: hidden shell execution"},
	})}
	scanner := Scanner{
		Rules: []rules.Rule{
			rules.RuleFunc(func(*skill.Skill) []result.Finding { return nil }),
		},
		Analyzer: analyzer,
	}

	got, err := scanner.Scan(s)
	if err != nil {
		t.Fatalf("Scan() error = %v, want nil", err)
	}
	if !analyzer.called {
		t.Fatal("Scan() did not call injected analyzer")
	}
	if len(got.Findings) != 1 {
		t.Fatalf("len(Scan().Findings) = %d, want 1", len(got.Findings))
	}
	if got.Findings[0].Source != result.SourceAnalyzer {
		t.Fatalf("Scan().Findings[0].Source = %q, want %q", got.Findings[0].Source, result.SourceAnalyzer)
	}
}

func TestScanSkipsAnalyzerWhenRulesFlagFindings(t *testing.T) {
	s := &skill.Skill{Name: "test skill", Description: "a test skill", Body: "body"}
	analyzer := &stubAnalyzer{result: result.NewCleanResult()}
	scanner := Scanner{
		Rules: []rules.Rule{
			rules.RuleFunc(func(*skill.Skill) []result.Finding {
				return []result.Finding{{
					Source:   result.SourceRule,
					Category: result.Category("shell"),
					Severity: result.SeverityWarning,
					Message:  "shell execution found",
				}}
			}),
		},
		Analyzer: analyzer,
	}

	got, err := scanner.Scan(s)
	if err != nil {
		t.Fatalf("Scan() error = %v, want nil", err)
	}
	if analyzer.called {
		t.Fatal("Scan() called analyzer after rule findings")
	}
	if len(got.Findings) != 1 {
		t.Fatalf("len(Scan().Findings) = %d, want 1", len(got.Findings))
	}
}

func TestScanReturnsAnalyzerError(t *testing.T) {
	s := &skill.Skill{Name: "test skill", Description: "a test skill", Body: "body"}
	analyzer := &stubAnalyzer{err: errors.New("missing GEMINI_API_KEY")}
	scanner := Scanner{
		Rules: []rules.Rule{
			rules.RuleFunc(func(*skill.Skill) []result.Finding { return nil }),
		},
		Analyzer: analyzer,
	}

	_, err := scanner.Scan(s)
	if err == nil {
		t.Fatal("Scan() error = nil, want non-nil")
	}
	if err.Error() != "missing GEMINI_API_KEY" {
		t.Fatalf("Scan() error = %q, want %q", err.Error(), "missing GEMINI_API_KEY")
	}
	if !analyzer.called {
		t.Fatal("Scan() did not call injected analyzer")
	}
}
