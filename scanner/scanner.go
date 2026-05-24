package scanner

import (
	"fmt"

	"github.com/murraycode/skill-wiz/result"
	"github.com/murraycode/skill-wiz/rules"
	"github.com/murraycode/skill-wiz/skill"
)

type Analyzer interface {
	Analyze(*skill.Skill) (result.Result, error)
}

type AnalyzerFunc func(*skill.Skill) (result.Result, error)

func (f AnalyzerFunc) Analyze(s *skill.Skill) (result.Result, error) {
	return f(s)
}

type Scanner struct {
	Rules    []rules.Rule
	Analyzer Analyzer
}

func (s Scanner) Scan(skillFile *skill.Skill) (result.Result, error) {
	if skillFile == nil {
		return result.Result{}, fmt.Errorf("scan skill: nil skill")
	}

	if ruleResult := rules.Scan(skillFile, s.Rules...); !ruleResult.Clean() {
		return ruleResult, nil
	}

	if s.Analyzer == nil {
		return result.NewCleanResult(), nil
	}

	return s.Analyzer.Analyze(skillFile)
}
