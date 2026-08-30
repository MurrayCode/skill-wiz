package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/murraycode/skill-wiz/analyse"
	"github.com/murraycode/skill-wiz/report"
	"github.com/murraycode/skill-wiz/result"
	"github.com/murraycode/skill-wiz/rules"
	"github.com/murraycode/skill-wiz/scanner"
	"github.com/murraycode/skill-wiz/skill"
)

const reportFileName = "skill-wiz-report.html"

// newSkillAnalyzer builds the analyzer for a run. It is a var so tests can
// substitute the model without touching the flag plumbing.
var newSkillAnalyzer = func(config analyse.Config) scanner.Analyzer {
	return analyse.GeminiAnalyzer{Config: config}
}

var skillRules = rules.Default()
var reportPath = defaultReportPath

// options is the parsed command line for a single run.
type options struct {
	path    string
	json    bool
	model   string
	timeout time.Duration
}

func (o options) analyzerConfig() analyse.Config {
	return analyse.Config{Model: o.model, Timeout: o.timeout}
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	opts, err := parseOptions(args, stderr)
	if err != nil {
		// -h and -help are a request, not a failure: the flag set has already
		// printed usage.
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintln(stderr, err)
		return 1
	}

	content, err := os.ReadFile(opts.path)
	if err != nil {
		fmt.Fprintf(stderr, "failed to read file: %v\n", err)
		return 1
	}
	s, err := skill.Parse(string(content))
	if err != nil {
		fmt.Fprintf(stderr, "Failed to parse skill: %v\n", err)
		return 1
	}
	// Validation short-circuits: a skill missing required metadata is never
	// handed to the rules or the analyzer.
	output := validationResultForSkill(s)
	if output.Clean() {
		output, err = scanner.Scanner{Rules: skillRules, Analyzer: newSkillAnalyzer(opts.analyzerConfig())}.Scan(s)
		if err != nil {
			fmt.Fprintf(stderr, "failed to analyze skill: %v\n", err)
			return 1
		}
	}

	destination := writeReport(s, opts.path, output, stderr)

	if opts.json {
		rendered, err := renderJSON(jsonInput{
			Path:       opts.path,
			Skill:      s,
			Result:     output,
			ReportPath: destination,
		})
		if err != nil {
			fmt.Fprintf(stderr, "failed to render JSON output: %v\n", err)
			return 1
		}

		fmt.Fprintln(stdout, rendered)
		return 0
	}

	fmt.Fprint(stdout, renderResult(output))
	if destination != "" {
		fmt.Fprint(stdout, renderReportPointer(destination))
	}
	return 0
}

// parseOptions turns raw arguments into options, reporting invalid flags and
// values as errors rather than exiting.
func parseOptions(args []string, stderr io.Writer) (options, error) {
	flags := flag.NewFlagSet("skill-wiz", flag.ContinueOnError)
	// The flag set stays quiet so that run reports every failure exactly once;
	// usage is printed here instead.
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}

	emitJSON := flags.Bool("json", false, "print the scan result as JSON instead of human-readable text")
	model := flags.String("model", analyse.DefaultModel, "Gemini model used for the analysis leg")
	timeout := flags.Duration("timeout", analyse.DefaultTimeout, "maximum time to wait for the analysis leg")

	if err := flags.Parse(args); err != nil {
		printUsage(flags, stderr)
		return options{}, err
	}

	if strings.TrimSpace(*model) == "" {
		return options{}, errors.New("invalid -model: must not be empty")
	}
	if *timeout <= 0 {
		return options{}, errors.New("invalid -timeout: must be greater than zero")
	}

	positional := flags.Args()
	if len(positional) == 0 {
		printUsage(flags, stderr)
		return options{}, errors.New("Please provide a path to a skill file")
	}
	if len(positional) > 1 {
		printUsage(flags, stderr)
		return options{}, fmt.Errorf("unexpected argument: %s", positional[1])
	}

	return options{
		path:    positional[0],
		json:    *emitJSON,
		model:   strings.TrimSpace(*model),
		timeout: *timeout,
	}, nil
}

func printUsage(flags *flag.FlagSet, w io.Writer) {
	fmt.Fprintln(w, "Usage: skill-wiz [flags] <path-to-skill-file>")
	fmt.Fprintln(w, "\nFlags:")
	flags.SetOutput(w)
	flags.PrintDefaults()
	flags.SetOutput(io.Discard)
}

// writeReport saves the HTML report and returns where it landed, or "" when it
// could not be written. A report that cannot be written is a warning, not a
// scan failure: the scan output already carries every finding.
func writeReport(s *skill.Skill, sourcePath string, scanResult result.Result, stderr io.Writer) string {
	destination, err := reportPath()
	if err != nil {
		fmt.Fprintf(stderr, "failed to resolve HTML report path: %v\n", err)
		return ""
	}

	if err := report.Write(destination, report.Input{
		SkillName:        s.Name,
		SkillDescription: s.Description,
		SourcePath:       sourcePath,
		GeneratedAt:      time.Now(),
		Result:           scanResult,
	}); err != nil {
		fmt.Fprintf(stderr, "failed to write HTML report: %v\n", err)
		return ""
	}

	return destination
}

func renderReportPointer(destination string) string {
	return fmt.Sprintf("\nHTML report: %s\nOpen it in your browser: %s\n", destination, fileURL(destination))
}

func fileURL(destination string) string {
	absolute, err := filepath.Abs(destination)
	if err != nil {
		absolute = destination
	}

	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}).String()
}

func defaultReportPath() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	return filepath.Join(workingDirectory, reportFileName), nil
}

// jsonInput is everything renderJSON needs about a completed scan.
type jsonInput struct {
	Path       string
	Skill      *skill.Skill
	Result     result.Result
	ReportPath string
}

// jsonReport is the stable shape of --json output. Keep field names additive:
// downstream tooling parses this.
type jsonReport struct {
	Path       string        `json:"path"`
	Skill      jsonSkill     `json:"skill"`
	Clean      bool          `json:"clean"`
	Findings   []jsonFinding `json:"findings"`
	ReportPath string        `json:"report_path,omitempty"`
}

type jsonSkill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type jsonFinding struct {
	Source   string `json:"source"`
	Category string `json:"category"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Evidence string `json:"evidence"`
}

func renderJSON(input jsonInput) (string, error) {
	payload := jsonReport{
		Path:       input.Path,
		Clean:      input.Result.Clean(),
		Findings:   make([]jsonFinding, 0, len(input.Result.Findings)),
		ReportPath: input.ReportPath,
	}
	if input.Skill != nil {
		payload.Skill = jsonSkill{Name: input.Skill.Name, Description: input.Skill.Description}
	}
	for _, finding := range input.Result.Findings {
		payload.Findings = append(payload.Findings, jsonFinding{
			Source:   string(finding.Source),
			Category: string(finding.Category),
			Severity: string(finding.Severity),
			Message:  finding.Message,
			Evidence: finding.Evidence.Summary,
		})
	}

	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode JSON result: %w", err)
	}

	return string(encoded), nil
}

func renderResult(scanResult result.Result) string {
	if scanResult.Clean() {
		return analyseCleanMessage()
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "Scan flagged %d finding(s)", len(scanResult.Findings))
	if sources := scanResult.Sources(); len(sources) > 0 {
		fmt.Fprintf(&builder, " from %s checks", formatSources(sources))
	}
	builder.WriteString("\n")
	for _, finding := range scanResult.Findings {
		fmt.Fprintf(&builder, "[%s] %s (%s): %s\n", finding.Severity, finding.Category, finding.Source, finding.Message)
		if finding.Evidence.Summary != "" {
			fmt.Fprintf(&builder, "Evidence: %s\n", finding.Evidence.Summary)
		}
	}

	return builder.String()
}

func formatSources(sources []result.Source) string {
	parts := make([]string, 0, len(sources))
	for _, source := range sources {
		parts = append(parts, string(source))
	}

	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
	}
}

func analyseCleanMessage() string {
	return "THIS SKILL APPEARS TO BE CLEAN, PLEASE MANUALLY VERIFY TO BE SURE"
}

func validationResultForSkill(s *skill.Skill) result.Result {
	if err := s.Validate(); err != nil {
		validationErrs, ok := err.(skill.ValidationErrors)
		if !ok {
			return result.NewResult(result.Finding{
				Source:   result.SourceValidation,
				Category: result.Category("metadata"),
				Severity: result.SeverityError,
				Message:  err.Error(),
			})
		}

		findings := make([]result.Finding, 0, len(validationErrs))
		for _, validationErr := range validationErrs {
			findings = append(findings, result.Finding{
				Source:   result.SourceValidation,
				Category: result.Category("metadata"),
				Severity: result.SeverityError,
				Message:  validationErr.Error(),
				Evidence: result.Evidence{Summary: fmt.Sprintf("missing required field: %s", validationErr.Field)},
			})
		}

		return result.NewResult(findings...)
	}

	return result.NewCleanResult()
}
