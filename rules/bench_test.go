package rules

import (
	"fmt"
	"strings"
	"testing"

	"github.com/murraycode/skill-wiz/skill"
)

// benchSkill builds a synthetic skill of repeated prose with one URL per
// paragraph, which is the shape the P6-004 scaling measurements used.
func benchSkill(paragraphs int) *skill.Skill {
	var body strings.Builder
	for i := 0; i < paragraphs; i++ {
		fmt.Fprintf(&body, "Look up the latest racing results and summarise the meeting for the reader.\n")
		fmt.Fprintf(&body, "See https://example%d.com/racing/results/meeting-%d for the full card.\n\n", i, i)
	}

	return &skill.Skill{
		Name:        "racing-results",
		Description: "Finds the latest racing results and summarises each meeting.",
		Body:        body.String(),
	}
}

func BenchmarkScanBySize(b *testing.B) {
	sizes := []struct {
		name       string
		paragraphs int
	}{
		{name: "small", paragraphs: 10},
		{name: "medium", paragraphs: 100},
		{name: "large", paragraphs: 400},
	}

	rules := Default()
	for _, size := range sizes {
		s := benchSkill(size.paragraphs)
		b.Run(size.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				Scan(s, rules...)
			}
		})
	}
}

func BenchmarkUnrelatedURLRule(b *testing.B) {
	s := benchSkill(200)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		unrelatedURLRule(s)
	}
}

func BenchmarkShellScriptRule(b *testing.B) {
	s := benchSkill(200)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		shellScriptRule(s)
	}
}

func BenchmarkShellCommandRule(b *testing.B) {
	s := benchSkill(200)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		shellCommandRule(s)
	}
}

func BenchmarkDescriptionMismatchRule(b *testing.B) {
	s := benchSkill(200)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		descriptionMismatchRule(s)
	}
}
