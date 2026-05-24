package skill

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantErr      bool
		wantName     string
		wantBody     string
		wantAudience string
	}{
		{
			name: "valid skill",
			content: `---
name: test skill
description: a test skill
license: MIT
compatibility: opencode
metadata:
  audience: developers
  workflow: standard
---
# Skill Body
This is the body.`,
			wantErr:      false,
			wantName:     "test skill",
			wantBody:     "# Skill Body\nThis is the body.",
			wantAudience: "developers",
		},
		{
			name:         "valid skill with CRLF line endings",
			content:      "---\r\nname: test skill\r\ndescription: a test skill\r\nlicense: MIT\r\ncompatibility: opencode\r\nmetadata:\r\n  audience: developers\r\n  workflow: standard\r\n---\r\n# Skill Body\r\nThis is the body.\r\n",
			wantErr:      false,
			wantName:     "test skill",
			wantBody:     "# Skill Body\nThis is the body.",
			wantAudience: "developers",
		},
		{
			name: "valid skill without trailing newline",
			content: `---
name: test skill
description: a test skill
license: MIT
compatibility: opencode
metadata:
  audience: developers
  workflow: standard
---
# Skill Body
This is the body.`,
			wantErr:      false,
			wantName:     "test skill",
			wantBody:     "# Skill Body\nThis is the body.",
			wantAudience: "developers",
		},
		{
			name: "valid skill with trailing newline",
			content: `---
name: test skill
description: a test skill
license: MIT
compatibility: opencode
metadata:
  audience: developers
  workflow: standard
---
# Skill Body
This is the body.
`,
			wantErr:      false,
			wantName:     "test skill",
			wantBody:     "# Skill Body\nThis is the body.",
			wantAudience: "developers",
		},
		{
			name: "missing frontmatter start",
			content: `name: test skill
---
body`,
			wantErr: true,
		},
		{
			name: "missing frontmatter end",
			content: `---
name: test skill`,
			wantErr: true,
		},
		{
			name: "malformed frontmatter",
			content: `---
name: [test skill
---
body`,
			wantErr: true,
		},
		{
			name: "empty body",
			content: `---
name: test skill
---
`,
			wantErr:  false,
			wantName: "test skill",
			wantBody: "",
		},
		{
			name: "empty body at end of file",
			content: `---
name: test skill
---`,
			wantErr:  false,
			wantName: "test skill",
			wantBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := Parse(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Name != tt.wantName {
					t.Errorf("Parse() Name = %v, want %v", got.Name, tt.wantName)
				}
				if got.Body != tt.wantBody {
					t.Errorf("Parse() Body = %q, want %q", got.Body, tt.wantBody)
				}
				if got.Metadata.Audience != tt.wantAudience {
					t.Errorf("Parse() Metadata.Audience = %v, want %v", got.Metadata.Audience, tt.wantAudience)
				}
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name         string
		skill        Skill
		wantMessages []string
	}{
		{
			name:  "valid skill passes validation",
			skill: Skill{Name: "test skill", Description: "a test skill"},
		},
		{
			name:         "missing name is reported",
			skill:        Skill{Description: "a test skill"},
			wantMessages: []string{"field name is required"},
		},
		{
			name:         "missing description is reported",
			skill:        Skill{Name: "test skill"},
			wantMessages: []string{"field description is required"},
		},
		{
			name:         "multiple missing fields are reported",
			skill:        Skill{},
			wantMessages: []string{"field name is required", "field description is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.skill.Validate()
			if len(tt.wantMessages) == 0 {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatal("Validate() error = nil, want validation error")
			}

			validationErr, ok := err.(ValidationErrors)
			if !ok {
				t.Fatalf("Validate() error type = %T, want ValidationErrors", err)
			}

			if len(validationErr) != len(tt.wantMessages) {
				t.Fatalf("len(ValidationErrors) = %d, want %d", len(validationErr), len(tt.wantMessages))
			}

			for i, want := range tt.wantMessages {
				if validationErr[i].Error() != want {
					t.Fatalf("ValidationErrors[%d] = %q, want %q", i, validationErr[i].Error(), want)
				}
			}
		})
	}
}
