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

// Load reads and parses a policy file. It checks the document's shape only;
// whether the rules it names exist is Validate's job, because only the caller
// knows the active rule set.
func Load(path string) (Policy, error) {
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

	return Policy{path: path, rules: parsed.Rules, require: parsed.Require}, nil
}

// Loaded reports whether a policy file backs this Policy.
func (p Policy) Loaded() bool {
	return p.path != ""
}

// Path reports where the policy was loaded from, empty for the zero Policy.
func (p Policy) Path() string {
	return p.path
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
