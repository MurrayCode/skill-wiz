package analyse

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/murraycode/skill-wiz/result"
	"google.golang.org/genai"
)

const cleanMessage = "THIS SKILL APPEARS TO BE CLEAN, PLEASE MANUALLY VERIFY TO BE SURE"

var errMissingAPIKey = errors.New("missing GEMINI_API_KEY")

type contentGenerator interface {
	GenerateContent(ctx context.Context, model string, content []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)
}

var newGenerator = func(ctx context.Context, apiKey string) (contentGenerator, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		return nil, err
	}

	return client.Models, nil
}

func Analyze(prompt string) (result.Result, error) {
	ctx := context.Background()
	apiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if apiKey == "" {
		return result.Result{}, errMissingAPIKey
	}

	generator, err := newGenerator(ctx, apiKey)
	if err != nil {
		return result.Result{}, fmt.Errorf("create genai client: %w", err)
	}

	response, err := generator.GenerateContent(
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
