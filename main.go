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
	"sync"
	"time"

	"github.com/murraycode/skill-wiz/analyse"
	"github.com/murraycode/skill-wiz/discover"
	"github.com/murraycode/skill-wiz/policy"
	"github.com/murraycode/skill-wiz/render"
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

// defaultConcurrency bounds how many files are scanned at once. The work is
// network-bound rather than CPU-bound, so the right default follows what the
// API tolerates, not how many cores the machine has.
const defaultConcurrency = 8

// colorEnabled decides whether severity labels are coloured. Colour is opt-out
// twice over: --no-color and the NO_COLOR convention both silence it, and a
// non-terminal writer never gets it in the first place.
func colorEnabled(terminal bool, noColor bool) bool {
	if !terminal || noColor {
		return false
	}

	return os.Getenv("NO_COLOR") == ""
}

// isTerminal reports whether a file is attached to a character device, which is
// as close to "a human is watching" as the standard library gets.
func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

// newSkillAnalyzer builds the analyzer for a run. It is a var so tests can
// substitute the model without touching the flag plumbing.
var newSkillAnalyzer = func(config analyse.Config) scanner.Analyzer {
	return analyse.GeminiAnalyzer{Config: config}
}

var skillRules = rules.Default()
var reportPath = defaultReportPath

// policyDirectory is where an undeclared policy file is looked for. It is a var
// so tests can point discovery at a temporary directory instead of depending on
// where the suite happens to run.
var policyDirectory = os.Getwd

// options is the parsed command line for a single run.
type options struct {
	paths       []string
	json        bool
	noColor     bool
	model       string
	timeout     time.Duration
	failOn      result.Severity
	concurrency int
	policy      string
	profile     string
}

func (o options) analyzerConfig() analyse.Config {
	return analyse.Config{Model: o.model, Timeout: o.timeout}
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, isTerminal(os.Stdout)))
}

func run(args []string, stdout io.Writer, stderr io.Writer, terminal bool) int {
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

	// A policy that cannot be read, parsed, or validated stops the run before
	// anything is scanned. Carrying on with a rule set the operator did not ask
	// for would report a verdict nobody configured.
	activePolicy, activeRules, err := resolvePolicy(opts.policy, opts.profile)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitFailure
	}

	// Preflight the credential once for the whole run. Without it the analysis
	// leg cannot run, but the deterministic rules can and must: the project's
	// design goal is that obvious detections never depend on the model. So warn
	// once, pass a nil analyzer, and carry on rules-only rather than failing a
	// file at a time.
	var analyzer scanner.Analyzer
	analysisSkipped := !analyse.HasAPIKey()
	if analysisSkipped {
		fmt.Fprintln(stderr, render.AnalysisSkippedWarning)
	} else {
		analyzer = newSkillAnalyzer(opts.analyzerConfig())
	}

	outcomes := scanFiles(files, scanSettings{rules: activeRules, policy: activePolicy, analyzer: analyzer}, opts.concurrency)

	// Both the results and the failures are consumed in file order, not in the
	// order the workers finished, so console output, the report, the JSON array,
	// the tally, and the stderr failures all stay deterministic.
	scans := make([]fileScan, 0, len(files))
	failed := false
	for index, outcome := range outcomes {
		if outcome.err != nil {
			// One unreadable or unparseable file must not hide the rest: report
			// it, remember the failure for the exit code, and carry on.
			fmt.Fprintln(stderr, scanError(files[index], outcome.err, len(files)))
			failed = true
			continue
		}

		scans = append(scans, outcome.scan)
	}
	if len(scans) == 0 {
		return exitFailure
	}

	// One run, one report: every scanned skill lands on the same page.
	destination := writeReport(scans, analysisSkipped, stderr)

	if opts.json {
		rendered, err := renderJSON(jsonInputs(scans, destination, analysisSkipped))
		if err != nil {
			fmt.Fprintf(stderr, "failed to render JSON output: %v\n", err)
			return exitFailure
		}

		fmt.Fprintln(stdout, rendered)
		return exitCode(scans, failed, opts.failOn)
	}

	if analysisSkipped {
		fmt.Fprint(stdout, render.AnalysisSkippedNote())
	}
	fmt.Fprint(stdout, render.Scans(renderInputs(scans), len(files), render.Style{Color: colorEnabled(terminal, opts.noColor)}))
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
			if result.GateRank(finding.Severity) >= result.GateRank(threshold) {
				return exitFindings
			}
		}
	}

	return exitClean
}

// scanOutcome is one file's result or the error that stopped it, held together
// so a worker can report either without touching shared state.
type scanOutcome struct {
	scan fileScan
	err  error
}

// scanFiles scans every file through a bounded worker pool. The pool is bounded
// rather than one goroutine per file because a directory scan can cover
// hundreds of skills and each one is an API request: unbounded goroutines would
// mean hundreds of simultaneous requests and near-certain rate limiting.
//
// Each worker writes to its own index of a pre-sized slice and never appends, so
// results come back in file order however completion order fell out. There is no
// shared cancellation: one bad file must never hide the rest, and a fail-fast
// pool would break exactly that.
func scanFiles(files []string, settings scanSettings, concurrency int) []scanOutcome {
	outcomes := make([]scanOutcome, len(files))
	if len(files) == 0 {
		return outcomes
	}

	workers := concurrency
	if workers < 1 {
		workers = 1
	}
	if workers > len(files) {
		workers = len(files)
	}

	indexes := make(chan int)
	var waitGroup sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for index := range indexes {
				scan, err := scanFile(files[index], settings)
				outcomes[index] = scanOutcome{scan: scan, err: err}
			}
		}()
	}

	for index := range files {
		indexes <- index
	}
	close(indexes)
	waitGroup.Wait()

	return outcomes
}

// scanSettings is everything a worker needs to scan one file. It is read-only
// for the whole run, so every worker shares one copy.
type scanSettings struct {
	rules    []rules.Rule
	policy   policy.Policy
	analyzer scanner.Analyzer
}

// fileScan is the outcome of scanning one skill file.
type fileScan struct {
	path   string
	skill  *skill.Skill
	result result.Result
}

// scanFile parses and scans a single file. Validation short-circuits: a skill
// missing required metadata is never handed to the rules or the analyzer.
func scanFile(path string, settings scanSettings) (fileScan, error) {
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
		output, err = scanner.Scanner{Rules: settings.rules, Analyzer: settings.analyzer}.Scan(s)
		if err != nil {
			return fileScan{}, fmt.Errorf("failed to analyze skill: %w", err)
		}
	}

	// Severity overrides are applied here, after the scanner has merged the
	// rule and analyzer findings and before anything renders them, so a policy
	// can never change what Merge collapses.
	return fileScan{path: path, skill: s, result: settings.policy.Apply(output)}, nil
}

// scanError names the file only when a run covers more than one, so single-file
// output stays as terse as it has always been.
func scanError(path string, err error, total int) string {
	if total == 1 {
		return err.Error()
	}

	return fmt.Sprintf("%s: %v", path, err)
}

// resolvePolicy loads the policy for a run and filters the rule set through it,
// returning both: the rules decide what runs, and the policy still has severity
// overrides to apply to what they find. Policy resolution lives here rather
// than in scanner or rules — the scanner still takes a plain rule slice and
// knows nothing about configuration.
func resolvePolicy(requested string, profile string) (policy.Policy, []rules.Rule, error) {
	active, err := loadPolicy(requested, profile)
	if err != nil {
		return policy.Policy{}, nil, err
	}
	if err := active.Validate(rules.IDs(skillRules)); err != nil {
		return policy.Policy{}, nil, err
	}

	return active, enabledRules(skillRules, active), nil
}

// loadPolicy applies the discovery order: an explicit --policy wins, and is a
// failure when it does not exist, because the operator asked for that file by
// name. Otherwise the working directory is checked, and finding nothing there
// is an ordinary policy-free run.
func loadPolicy(requested string, profile string) (policy.Policy, error) {
	if requested != "" {
		return policy.LoadProfile(requested, profile)
	}

	directory, err := policyDirectory()
	if err != nil {
		return policy.Policy{}, fmt.Errorf("failed to resolve the working directory for policy discovery: %w", err)
	}

	discovered := policy.Discover(directory)
	if discovered == "" {
		// A profile only exists inside a policy file, so asking for one when
		// there is no policy is a broken configuration rather than a run with
		// nothing configured.
		if profile != "" {
			return policy.Policy{}, fmt.Errorf("profile %q was requested but no policy file was found (looked for %s in %s)", profile, policy.FileName, directory)
		}

		return policy.Policy{}, nil
	}

	return policy.LoadProfile(discovered, profile)
}

// enabledRules keeps the rule order the default set declares, so console and
// report ordering is unchanged by which rules a policy switched off.
func enabledRules(ruleSet []rules.Rule, active policy.Policy) []rules.Rule {
	enabled := make([]rules.Rule, 0, len(ruleSet))
	for _, rule := range ruleSet {
		if !active.Enabled(rule.ID()) {
			continue
		}
		enabled = append(enabled, rule)
	}

	return enabled
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
	noColor := flags.Bool("no-color", false, "never colour severity labels, even when stdout is a terminal")
	model := flags.String("model", analyse.DefaultModel, "Gemini model used for the analysis leg")
	timeout := flags.Duration("timeout", analyse.DefaultTimeout, "maximum time to wait for the analysis leg")
	failOn := flags.String("fail-on", string(result.SeverityError), "lowest finding severity that fails the run: error, warning, or info")
	concurrency := flags.Int("concurrency", defaultConcurrency, "how many files to scan at once; --timeout still bounds each analysis request individually")
	policyPath := flags.String("policy", "", "path to a policy file; defaults to "+policy.FileName+" in the working directory when one is present")
	profileName := flags.String("profile", "", "name of the policy profile to apply; the base policy is used when omitted")

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
	if *concurrency <= 0 {
		return options{}, errors.New("invalid -concurrency: must be greater than zero")
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
		paths:       positional,
		json:        *emitJSON,
		noColor:     *noColor,
		model:       strings.TrimSpace(*model),
		timeout:     *timeout,
		failOn:      threshold,
		concurrency: *concurrency,
		policy:      strings.TrimSpace(*policyPath),
		profile:     strings.TrimSpace(*profileName),
	}, nil
}

// parseSeverity turns a --fail-on value into the threshold that gates the exit
// code, rejecting anything outside the known severities.
func parseSeverity(value string) (result.Severity, error) {
	severity := result.Severity(strings.ToLower(strings.TrimSpace(value)))
	if !result.Known(severity) {
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
func writeReport(scans []fileScan, analysisSkipped bool, stderr io.Writer) string {
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
			AnalysisSkipped:  analysisSkipped,
		})
	}

	if err := report.Write(destination, inputs...); err != nil {
		fmt.Fprintf(stderr, "failed to write HTML report: %v\n", err)
		return ""
	}

	return destination
}

// renderInputs maps completed scans onto what the console renderer needs. The
// renderer never reads the parsed skill, so it never receives one.
func renderInputs(scans []fileScan) []render.Input {
	inputs := make([]render.Input, 0, len(scans))
	for _, scan := range scans {
		inputs = append(inputs, render.Input{Path: scan.path, Result: scan.result})
	}

	return inputs
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
	Path            string
	Skill           *skill.Skill
	Result          result.Result
	ReportPath      string
	AnalysisSkipped bool
}

// jsonReport is the stable shape of --json output. Keep field names additive:
// downstream tooling parses this.
type jsonReport struct {
	Path       string        `json:"path"`
	Skill      jsonSkill     `json:"skill"`
	Clean      bool          `json:"clean"`
	Findings   []jsonFinding `json:"findings"`
	ReportPath string        `json:"report_path,omitempty"`
	// AnalysisSkipped marks a rules-only result. It is additive and omitted
	// from a complete scan, so a consumer written before it keeps working while
	// one written after it can tell the two apart.
	AnalysisSkipped bool `json:"analysis_skipped,omitempty"`
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
	// OverriddenFrom is the severity this finding carried before a policy
	// changed it. It is additive and omitted when no policy touched the
	// finding, so severity keeps meaning the effective value that gates the
	// exit code and a consumer written before this field still reads it.
	OverriddenFrom string `json:"overridden_from,omitempty"`
}

// jsonInputs pairs every scan with the one report the run wrote, so a consumer
// reading a single entry still knows where to look.
func jsonInputs(scans []fileScan, reportPath string, analysisSkipped bool) []jsonInput {
	inputs := make([]jsonInput, 0, len(scans))
	for _, scan := range scans {
		inputs = append(inputs, jsonInput{
			Path:            scan.path,
			Skill:           scan.skill,
			Result:          scan.result,
			ReportPath:      reportPath,
			AnalysisSkipped: analysisSkipped,
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
		Path:            input.Path,
		Clean:           input.Result.Clean(),
		Findings:        make([]jsonFinding, 0, len(input.Result.Findings)),
		ReportPath:      input.ReportPath,
		AnalysisSkipped: input.AnalysisSkipped,
	}
	if input.Skill != nil {
		payload.Skill = jsonSkill{Name: input.Skill.Name, Description: input.Skill.Description}
	}
	for _, finding := range input.Result.Findings {
		payload.Findings = append(payload.Findings, jsonFinding{
			Source:         string(finding.Source),
			Category:       string(finding.Category),
			Severity:       string(finding.Severity),
			Message:        finding.Message,
			Evidence:       finding.Evidence.Summary,
			OverriddenFrom: string(finding.OverriddenFrom),
		})
	}

	return payload
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
