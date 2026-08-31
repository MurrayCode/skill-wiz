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
	"github.com/murraycode/skill-wiz/discover"
	"github.com/murraycode/skill-wiz/report"
	"github.com/murraycode/skill-wiz/result"
	"github.com/murraycode/skill-wiz/rules"
	"github.com/murraycode/skill-wiz/scanner"
	"github.com/murraycode/skill-wiz/skill"
)

const reportFileName = "skill-wiz-report.html"

// Exit codes. A flagged scan is deliberately distinct from an operational
// failure so a pipeline can tell "this skill looks wrong" from "the scan did
// not run". Do not renumber these.
const (
	exitClean    = 0
	exitFailure  = 1
	exitFindings = 2
)

// severityRank orders severities so that a threshold can be compared against a
// finding. An unrecognised severity ranks lowest, so it gates only the most
// permissive threshold rather than silently failing a build.
var severityRank = map[result.Severity]int{
	result.SeverityInfo:    0,
	result.SeverityWarning: 1,
	result.SeverityError:   2,
}

// newSkillAnalyzer builds the analyzer for a run. It is a var so tests can
// substitute the model without touching the flag plumbing.
var newSkillAnalyzer = func(config analyse.Config) scanner.Analyzer {
	return analyse.GeminiAnalyzer{Config: config}
}

var skillRules = rules.Default()
var reportPath = defaultReportPath

// options is the parsed command line for a single run.
type options struct {
	paths   []string
	json    bool
	model   string
	timeout time.Duration
	failOn  result.Severity
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
			return exitClean
		}
		fmt.Fprintln(stderr, err)
		return exitFailure
	}

	files, err := discover.Files(opts.paths)
	if err != nil {
		fmt.Fprintf(stderr, "failed to collect skill files: %v\n", err)
		return exitFailure
	}

	analyzer := newSkillAnalyzer(opts.analyzerConfig())

	scans := make([]fileScan, 0, len(files))
	failed := false
	for _, file := range files {
		scan, err := scanFile(file, analyzer)
		if err != nil {
			// One unreadable or unparseable file must not hide the rest: report
			// it, remember the failure for the exit code, and carry on.
			fmt.Fprintln(stderr, scanError(file, err, len(files)))
			failed = true
			continue
		}

		scans = append(scans, scan)
	}
	if len(scans) == 0 {
		return exitFailure
	}

	// One run, one report: every scanned skill lands on the same page.
	destination := writeReport(scans, stderr)

	if opts.json {
		rendered, err := renderJSON(jsonInputs(scans, destination))
		if err != nil {
			fmt.Fprintf(stderr, "failed to render JSON output: %v\n", err)
			return exitFailure
		}

		fmt.Fprintln(stdout, rendered)
		return exitCode(scans, failed, opts.failOn)
	}

	fmt.Fprint(stdout, renderScans(scans, len(files)))
	if destination != "" {
		fmt.Fprint(stdout, renderReportPointer(destination))
	}
	return exitCode(scans, failed, opts.failOn)
}

// exitCode maps a completed run onto its exit code. An operational failure
// outranks findings: a run that could not scan every file reports that first,
// even when the files it did scan were flagged.
func exitCode(scans []fileScan, failed bool, threshold result.Severity) int {
	if failed {
		return exitFailure
	}

	for _, scan := range scans {
		for _, finding := range scan.result.Findings {
			if severityRank[finding.Severity] >= severityRank[threshold] {
				return exitFindings
			}
		}
	}

	return exitClean
}

// fileScan is the outcome of scanning one skill file.
type fileScan struct {
	path   string
	skill  *skill.Skill
	result result.Result
}

// scanFile parses and scans a single file. Validation short-circuits: a skill
// missing required metadata is never handed to the rules or the analyzer.
func scanFile(path string, analyzer scanner.Analyzer) (fileScan, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return fileScan{}, fmt.Errorf("failed to read file: %w", err)
	}
	s, err := skill.Parse(string(content))
	if err != nil {
		return fileScan{}, fmt.Errorf("Failed to parse skill: %w", err)
	}

	output := validationResultForSkill(s)
	if output.Clean() {
		output, err = scanner.Scanner{Rules: skillRules, Analyzer: analyzer}.Scan(s)
		if err != nil {
			return fileScan{}, fmt.Errorf("failed to analyze skill: %w", err)
		}
	}

	return fileScan{path: path, skill: s, result: output}, nil
}

// scanError names the file only when a run covers more than one, so single-file
// output stays as terse as it has always been.
func scanError(path string, err error, total int) string {
	if total == 1 {
		return err.Error()
	}

	return fmt.Sprintf("%s: %v", path, err)
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
	failOn := flags.String("fail-on", string(result.SeverityError), "lowest finding severity that fails the run: error, warning, or info")

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
	threshold, err := parseSeverity(*failOn)
	if err != nil {
		return options{}, err
	}

	positional := flags.Args()
	if len(positional) == 0 {
		printUsage(flags, stderr)
		return options{}, errors.New("Please provide a path to a skill file")
	}

	return options{
		paths:   positional,
		json:    *emitJSON,
		model:   strings.TrimSpace(*model),
		timeout: *timeout,
		failOn:  threshold,
	}, nil
}

// parseSeverity turns a --fail-on value into the threshold that gates the exit
// code, rejecting anything outside the known severities.
func parseSeverity(value string) (result.Severity, error) {
	severity := result.Severity(strings.ToLower(strings.TrimSpace(value)))
	if _, ok := severityRank[severity]; !ok {
		return "", fmt.Errorf("invalid -fail-on %q: must be one of error, warning, info", strings.TrimSpace(value))
	}

	return severity, nil
}

func printUsage(flags *flag.FlagSet, w io.Writer) {
	fmt.Fprintln(w, "Usage: skill-wiz [flags] <path-to-skill-file-or-directory>...")
	fmt.Fprintln(w, "\nFlags:")
	flags.SetOutput(w)
	flags.PrintDefaults()
	flags.SetOutput(io.Discard)
}

// writeReport saves the run's HTML report and returns where it landed, or ""
// when it could not be written. A report that cannot be written is a warning,
// not a scan failure: the console output already carries every finding.
func writeReport(scans []fileScan, stderr io.Writer) string {
	destination, err := reportPath()
	if err != nil {
		fmt.Fprintf(stderr, "failed to resolve HTML report path: %v\n", err)
		return ""
	}

	inputs := make([]report.Input, 0, len(scans))
	for _, scan := range scans {
		inputs = append(inputs, report.Input{
			SkillName:        scan.skill.Name,
			SkillDescription: scan.skill.Description,
			SourcePath:       scan.path,
			GeneratedAt:      time.Now(),
			Result:           scan.result,
		})
	}

	if err := report.Write(destination, inputs...); err != nil {
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

// jsonInputs pairs every scan with the one report the run wrote, so a consumer
// reading a single entry still knows where to look.
func jsonInputs(scans []fileScan, reportPath string) []jsonInput {
	inputs := make([]jsonInput, 0, len(scans))
	for _, scan := range scans {
		inputs = append(inputs, jsonInput{
			Path:       scan.path,
			Skill:      scan.skill,
			Result:     scan.result,
			ReportPath: reportPath,
		})
	}

	return inputs
}

// renderJSON encodes the scans. A single scan stays the object it has always
// been; several scans become an array of that same object.
func renderJSON(inputs []jsonInput) (string, error) {
	reports := make([]jsonReport, 0, len(inputs))
	for _, input := range inputs {
		reports = append(reports, newJSONReport(input))
	}

	var payload any = reports
	if len(reports) == 1 {
		payload = reports[0]
	}

	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode JSON result: %w", err)
	}

	return string(encoded), nil
}

func newJSONReport(input jsonInput) jsonReport {
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

	return payload
}

// renderScans renders every scan in order. A run over a single file renders
// exactly as it did before multi-file support; a run over several files heads
// each result with its path, so findings keep their file even when some of the
// files failed to scan.
func renderScans(scans []fileScan, total int) string {
	var builder strings.Builder
	for index, scan := range scans {
		if total > 1 {
			if index > 0 {
				builder.WriteString("\n")
			}
			fmt.Fprintf(&builder, "=== %s ===\n", scan.path)
		}

		builder.WriteString(renderResult(scan.result))
	}

	return builder.String()
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
