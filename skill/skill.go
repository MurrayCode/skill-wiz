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

func Parse(content string) (*Skill, error) {

	sContent := string(content)
	const delimiter = "---\n"

	if !strings.HasPrefix(sContent, delimiter) {
		return nil, errors.New("invalid skill format: must start with ---")
	}

	rest := sContent[len(delimiter):]
	endIndex := strings.Index(rest, "\n---")
	if endIndex == -1 {
		return nil, errors.New("invalid skill format: missing closing ---")
	}

	yamlPart := rest[:endIndex]

	const closingFence = "\n---"
	bodyStart := endIndex + len(closingFence)

	if bodyStart < len(rest) && rest[bodyStart] == '\n' {
		bodyStart++
	}

	bodyPart := rest[bodyStart:]

	var s Skill
	if err := yaml.Unmarshal([]byte(yamlPart), &s); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}
	s.Body = strings.TrimSpace(bodyPart)
	return &s, nil
}
