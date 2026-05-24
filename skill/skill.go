package skill

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Metadata represents the metadata section of a skill
type Metadata struct {
	Audience string `yaml:"audience"`
	Workflow string `yaml:"workflow"`
}

// Skill represents the parsed skill file
type Skill struct {
	Name          string   `yaml:"name"`
	Description   string   `yaml:"description"`
	License       string   `yaml:"license"`
	Compatibility string   `yaml:"compatibility"`
	Metadata      Metadata `yaml:"metadata"`
	Body          string   `yaml:"-"`
}

// ValidationError reports a single invalid field on a parsed skill.
type ValidationError struct {
	Field string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("field %s is required", e.Field)
}

// ValidationErrors preserves field-level validation failures.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	messages := make([]string, 0, len(e))
	for _, validationErr := range e {
		messages = append(messages, validationErr.Error())
	}
	return strings.Join(messages, "; ")
}

func (s Skill) Validate() error {
	var errs ValidationErrors

	if strings.TrimSpace(s.Name) == "" {
		errs = append(errs, ValidationError{Field: "name"})
	}
	if strings.TrimSpace(s.Description) == "" {
		errs = append(errs, ValidationError{Field: "description"})
	}

	if len(errs) == 0 {
		return nil
	}

	return errs
}

func Parse(content string) (*Skill, error) {
	sContent := strings.ReplaceAll(content, "\r\n", "\n")
	const delimiter = "---\n"

	if !strings.HasPrefix(sContent, delimiter) {
		return nil, errors.New("invalid skill format: must start with ---")
	}

	rest := sContent[len(delimiter):]
	endIndex := strings.Index(rest, "\n---\n")
	closingFenceLen := len("\n---\n")
	if endIndex == -1 && strings.HasSuffix(rest, "\n---") {
		endIndex = len(rest) - len("\n---")
		closingFenceLen = len("\n---")
	}
	if endIndex == -1 {
		return nil, errors.New("invalid skill format: missing closing ---")
	}

	yamlPart := rest[:endIndex]
	bodyStart := endIndex + closingFenceLen

	bodyPart := rest[bodyStart:]

	var s Skill
	if err := yaml.Unmarshal([]byte(yamlPart), &s); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}
	s.Body = strings.TrimSpace(bodyPart)
	return &s, nil
}
