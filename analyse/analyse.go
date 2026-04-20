package analyse

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/murraycode/skill-wiz/result"
	"google.golang.org/genai"
)

const cleanMessage = "THIS SKILL APPEARS TO BE CLEAN, PLEASE MANUALLY VERIFY TO BE SURE"

func Analyze(prompt string) (result.Result, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: os.Getenv("GEMINI_API_KEY"),
	})
	if err != nil {
		return result.Result{}, fmt.Errorf("create genai client: %w", err)
	}
	response, err := client.Models.GenerateContent(
		ctx,
		"gemini-2.5-flash",
		genai.Text(prompt),
		nil,
	)
	if err != nil {
		return result.Result{}, fmt.Errorf("generate analysis: %w", err)
	}

	return resultFromText(response.Text()), nil
}

func resultFromText(text string) result.Result {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || trimmed == cleanMessage {
		return result.NewCleanResult()
	}

	return result.NewResult(result.Finding{
		Source:   result.SourceAnalyzer,
		Category: result.Category("analysis"),
		Severity: result.SeverityWarning,
		Message:  "Analyzer reported potential issues",
		Evidence: result.Evidence{Summary: trimmed},
	})
}
