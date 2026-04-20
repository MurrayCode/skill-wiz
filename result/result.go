package result

type Severity string
type Category string
type Source string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

const (
	SourceValidation Source = "validation"
	SourceRule       Source = "rule"
	SourceAnalyzer   Source = "analyzer"
)

type Evidence struct {
	Summary string
}

type Finding struct {
	Source   Source
	Category Category
	Severity Severity
	Message  string
	Evidence Evidence
}

type Result struct {
	Findings []Finding
}

func NewCleanResult() Result {
	return Result{}
}

func NewResult(findings ...Finding) Result {
	if len(findings) == 0 {
		return NewCleanResult()
	}

	result := Result{Findings: make([]Finding, len(findings))}
	copy(result.Findings, findings)
	return result
}

func (r Result) Clean() bool {
	return len(r.Findings) == 0
}
