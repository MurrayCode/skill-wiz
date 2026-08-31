// Package policy loads the optional configuration file that decides which
// deterministic rules a run enforces.
//
// A policy is opt-in in both directions: with no file and no --policy flag the
// zero Policy applies, and it enables every rule, so a policy-free run behaves
// exactly as it did before this package existed. A policy that cannot be read,
// parsed, or validated is an operational failure that stops the run — it is
// never downgraded to a finding, because a scanner that quietly ignores its own
// configuration is worse than one that refuses to start.
//
// It knows nothing about rules or skills: rules arrive as identifiers, and the
// caller filters its own rule set through Enabled.
package policy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// FileName is the policy file looked for in the working directory. Discovery
// deliberately does not search upwards, check the home directory, or read an
// environment variable: a scanner whose verdict depends on where it was run
// from is hard to trust.
const FileName = ".skill-wiz.yaml"

// document is the on-disk shape. Decoding is strict, so a misspelled key fails
// the load rather than silently doing nothing — the realistic way to lose an
// enforcement is a typo, not a hostile edit.
type document struct {
	Rules    map[string]ruleConfig `yaml:"rules"`
	Require  []string              `yaml:"require"`
	Profiles map[string]profile    `yaml:"profiles"`
}

// profile is one named variation on the base policy. It carries the same keys
// as the base and, deliberately, no profiles of its own: profiles do not
// inherit from each other, so there is exactly one overlay to reason about.
type profile struct {
	Rules   map[string]ruleConfig `yaml:"rules"`
	Require []string              `yaml:"require"`
}

// ruleConfig is what a policy can say about one rule. Every field is a pointer
// so that "absent" stays distinguishable from "set to the zero value".
type ruleConfig struct {
	Enabled *bool `yaml:"enabled"`
}

// Policy is the effective configuration for a run. Its zero value is the
// no-policy case and enables everything.
type Policy struct {
	path    string
	profile string
	rules   map[string]ruleConfig
	require []string
}

// Discover returns the path of the policy file in directory, or "" when there
// is none. Anything other than a plain missing file — an unreadable directory,
// a directory named .skill-wiz.yaml — is left for Load to report with its path.
func Discover(directory string) string {
	path := filepath.Join(directory, FileName)
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ""
		}

		return path
	}
	if info.IsDir() {
		return ""
	}

	return path
}

// Load reads and parses a policy file, selecting no profile.
func Load(path string) (Policy, error) {
	return LoadProfile(path, "")
}

// LoadProfile reads a policy file and resolves it against a named profile. It
// checks the document's shape only; whether the rules it names exist is
// Validate's job, because only the caller knows the active rule set.
//
// Naming a profile the file does not define is a failure rather than a silent
// fall back to the base, because the realistic case is a broken CI
// configuration that would otherwise enforce the wrong rules quietly.
func LoadProfile(path string, name string) (Policy, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("read policy %s: %w", path, err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)

	var parsed document
	if err := decoder.Decode(&parsed); err != nil && !errors.Is(err, io.EOF) {
		return Policy{}, fmt.Errorf("parse policy %s: %w", path, err)
	}

	// One document, so that a stray --- cannot hide half a policy.
	var trailing document
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Policy{}, fmt.Errorf("parse policy %s: must be a single YAML document", path)
	}

	resolved := Policy{path: path, rules: parsed.Rules, require: parsed.Require}
	if name == "" {
		return resolved, nil
	}

	selected, ok := parsed.Profiles[name]
	if !ok {
		if len(parsed.Profiles) == 0 {
			return Policy{}, fmt.Errorf("policy %s: profile %q was requested but the policy defines no profiles", path, name)
		}

		return Policy{}, fmt.Errorf("policy %s: unknown profile %q (available profiles: %s)", path, name, strings.Join(profileNames(parsed.Profiles), ", "))
	}

	return resolved.overlay(name, selected), nil
}

// overlay applies a profile on top of the base policy. The overlay is key by
// key and never a merge: a rule the profile names takes the profile's entry
// whole, a rule it does not name keeps the base's, and a require list replaces
// the base's outright. Merging would make a profile's effect depend on what the
// base happened to say, which is exactly what makes layered configuration hard
// to reason about.
func (p Policy) overlay(name string, selected profile) Policy {
	overlaid := Policy{path: p.path, profile: name, require: p.require}

	overlaid.rules = make(map[string]ruleConfig, len(p.rules)+len(selected.Rules))
	for id, config := range p.rules {
		overlaid.rules[id] = config
	}
	for id, config := range selected.Rules {
		overlaid.rules[id] = config
	}

	if selected.Require != nil {
		overlaid.require = selected.Require
	}

	return overlaid
}

// profileNames lists the profiles a document defines, sorted so the failure
// message reads the same on every run.
func profileNames(profiles map[string]profile) []string {
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

// Loaded reports whether a policy file backs this Policy.
func (p Policy) Loaded() bool {
	return p.path != ""
}

// Path reports where the policy was loaded from, empty for the zero Policy.
func (p Policy) Path() string {
	return p.path
}

// Profile reports which profile was applied, empty when the base policy is in
// force.
func (p Policy) Profile() string {
	return p.profile
}

// Enabled reports whether a rule should run. A rule the policy says nothing
// about is enabled, so a policy only ever has to name what it changes.
func (p Policy) Enabled(id string) bool {
	config, ok := p.rules[id]
	if !ok || config.Enabled == nil {
		return true
	}

	return *config.Enabled
}

// Validate checks the policy against the identifiers of the active rule set.
//
// Naming a rule that does not exist is a failure in both directions: under
// rules it is almost always a typo that would silently do nothing, and under
// require it means the policy stopped enforcing something a refactor removed.
// Requiring a rule the same policy disables is the same mistake said twice.
func (p Policy) Validate(known []string) error {
	if !p.Loaded() {
		return nil
	}

	index := make(map[string]struct{}, len(known))
	for _, id := range known {
		index[id] = struct{}{}
	}

	for _, id := range sortedKeys(p.rules) {
		if _, ok := index[id]; !ok {
			return fmt.Errorf("policy %s: rules names unknown rule %q (known rules: %s)", p.path, id, strings.Join(known, ", "))
		}
	}

	for _, id := range p.require {
		if _, ok := index[id]; !ok {
			return fmt.Errorf("policy %s: require lists unknown rule %q (known rules: %s)", p.path, id, strings.Join(known, ", "))
		}
		if !p.Enabled(id) {
			return fmt.Errorf("policy %s: require lists %q but the policy disables it", p.path, id)
		}
	}

	return nil
}

// sortedKeys keeps validation failures deterministic: a policy with two bad
// rule names reports the same one on every run.
func sortedKeys(rules map[string]ruleConfig) []string {
	keys := make([]string, 0, len(rules))
	for key := range rules {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return keys
}
