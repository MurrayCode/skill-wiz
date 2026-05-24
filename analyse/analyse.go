package analyse

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/murraycode/skill-wiz/result"
	"github.com/murraycode/skill-wiz/skill"
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

type GeminiAnalyzer struct{}

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

func (GeminiAnalyzer) Analyze(s *skill.Skill) (result.Result, error) {
	if s == nil {
		return result.Result{}, errors.New("nil skill")
	}

	return Analyze(promptForSkill(s))
}

func promptForSkill(s *skill.Skill) string {
	return fmt.Sprintf(`JOB: Your job is to analyze the following two bodys of text and flag any mismatches between the discription and the instructions and any suspicious or hidden behavior.
TASKS: Analyze the following two bodys of text. The first will be a description which will be the paragraph following the word ***DESCRIPTION***
The next will be body describing the actions the file describes an agent to take which will follow the word ***BODY***.
INPUT: ***DESCRIPTION*** %s. ***BODY*** %s. END OF INPUT
OUTPUT: Return a report on your findings under the following format. Return the sentence ***THIS SKILL APPEARS TO BE CLEAN, PLEASE MANUALLY VERIFY TO BE SURE*** if no mismatches, suspicious or hidden behavior are found. If you find any mismatches between the description and the instructions report back with the word ***MISMATCHES*** and your findings. If you find any suspicious behavior report back with the word ***SUSPICIOUS*** and description of your findings. If you find any hidden behaviour report back with the word ***HIDDEN*** and a description of the hidden behavior 
		`, s.Description, s.Body)
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
