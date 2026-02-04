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
			name: "empty body",
			content: `---
name: test skill
---
`,
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
