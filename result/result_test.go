package result

import "testing"

func TestResultClean(t *testing.T) {
	tests := []struct {
		name      string
		result    Result
		wantClean bool
	}{
		{
			name:      "clean result has no findings",
			result:    NewCleanResult(),
			wantClean: true,
		},
		{
			name: "result with finding is flagged",
			result: NewResult(Finding{
				Source:   SourceAnalyzer,
				Category: Category("analysis"),
				Severity: SeverityWarning,
				Message:  "potential mismatch",
				Evidence: Evidence{Summary: "model reported mismatch"},
			}),
			wantClean: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Clean(); got != tt.wantClean {
				t.Fatalf("Result.Clean() = %v, want %v", got, tt.wantClean)
			}
		})
	}
}

func TestNewResultPreservesFindings(t *testing.T) {
	findings := []Finding{
		{
			Source:   SourceValidation,
			Category: Category("metadata"),
			Severity: SeverityError,
			Message:  "missing description",
			Evidence: Evidence{Summary: "field description is empty"},
		},
		{
			Source:   SourceRule,
			Category: Category("shell"),
			Severity: SeverityWarning,
			Message:  "command executes shell",
			Evidence: Evidence{Summary: "bash command found"},
		},
	}

	got := NewResult(findings...)

	if len(got.Findings) != len(findings) {
		t.Fatalf("len(Result.Findings) = %d, want %d", len(got.Findings), len(findings))
	}

	for i := range findings {
		if got.Findings[i] != findings[i] {
			t.Fatalf("Result.Findings[%d] = %#v, want %#v", i, got.Findings[i], findings[i])
		}
	}

	findings[0].Message = "changed after creation"
	if got.Findings[0].Message == findings[0].Message {
		t.Fatal("NewResult() did not protect stored findings from caller mutation")
	}
}

func TestNewResultWithoutFindingsReturnsCleanResult(t *testing.T) {
	got := NewResult()

	if !got.Clean() {
		t.Fatalf("NewResult().Clean() = %v, want true", got.Clean())
	}
	if len(got.Findings) != 0 {
		t.Fatalf("len(NewResult().Findings) = %d, want 0", len(got.Findings))
	}
}
