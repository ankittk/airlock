package evalcase

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// Binding maps changed artifacts to an eval suite file (under .airlock/evals/).
type Binding struct {
	When  BindingWhen `yaml:"when"`
	Suite string      `yaml:"suite"`
}

type BindingWhen struct {
	Kind string `yaml:"kind,omitempty"` // prompt|skill|mcp|model|tool|agent|dependency
	ID   string `yaml:"id,omitempty"`
}

type BindingsFile struct {
	Version  int       `yaml:"version"`
	Bindings []Binding `yaml:"bindings"`
}

// LoadBindings reads .airlock/eval-bindings.yml (optional).
func LoadBindings(path string) (BindingsFile, error) {
	var f BindingsFile
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return f, nil
		}
		return f, err
	}
	if err := yaml.Unmarshal(data, &f); err != nil {
		return f, fmt.Errorf("eval-bindings: %w", err)
	}
	return f, nil
}

// ArtifactChange is the minimal diff input for suite selection.
type ArtifactChange struct {
	Kind   string
	ID     string
	Status string // added|removed|changed
}

// SelectSuites returns suite paths (relative to evalsDir) for changed artifacts.
// ponytail: most-specific match (kind+id) beats kind-only; union deduped.
func SelectSuites(changes []ArtifactChange, bindings BindingsFile) []string {
	if len(bindings.Bindings) == 0 || len(changes) == 0 {
		return nil
	}
	type match struct {
		specificity int
		suite       string
	}
	var hits []match
	for _, c := range changes {
		if c.Status == "removed" {
			continue
		}
		for _, b := range bindings.Bindings {
			if !bindingMatches(b.When, c) {
				continue
			}
			spec := 1
			if b.When.ID != "" {
				spec = 2
			}
			hits = append(hits, match{specificity: spec, suite: b.Suite})
		}
	}
	if len(hits) == 0 {
		return nil
	}
	slices.SortFunc(hits, func(a, b match) int {
		return b.specificity - a.specificity
	})
	maxSpec := hits[0].specificity
	var out []string
	seen := map[string]bool{}
	for _, h := range hits {
		if h.specificity < maxSpec {
			break
		}
		if h.suite == "" || seen[h.suite] {
			continue
		}
		seen[h.suite] = true
		out = append(out, h.suite)
	}
	return out
}

func bindingMatches(w BindingWhen, c ArtifactChange) bool {
	if w.Kind != "" && w.Kind != c.Kind {
		return false
	}
	if w.ID != "" && w.ID != c.ID {
		return false
	}
	return w.Kind != "" || w.ID != ""
}

// LoadBoundCases loads and merges cases from bound suite files under evalsDir.
func LoadBoundCases(evalsDir string, suiteNames []string) (Suite, []Case, error) {
	if len(suiteNames) == 0 {
		return Suite{}, nil, fmt.Errorf("no suites selected")
	}
	seenCase := map[string]bool{}
	var merged []Case
	var suite Suite
	for _, name := range suiteNames {
		path := name
		if !filepath.IsAbs(path) {
			path = filepath.Join(evalsDir, name)
		}
		s, cases, err := LoadSuiteFile(path)
		if err != nil {
			return Suite{}, nil, fmt.Errorf("binding suite %s: %w", name, err)
		}
		if suite.K == 0 {
			suite = s
		}
		for _, c := range cases {
			if seenCase[c.ID] {
				continue
			}
			seenCase[c.ID] = true
			merged = append(merged, c)
		}
	}
	if len(merged) == 0 {
		return suite, nil, fmt.Errorf("bound suites had no cases")
	}
	return suite, merged, nil
}

// DefaultBindingsPath is .airlock/eval-bindings.yml under root.
func DefaultBindingsPath(root string) string {
	return filepath.Join(root, ".airlock", "eval-bindings.yml")
}

// NormalizeSuiteName trims suite path for logs.
func NormalizeSuiteName(s string) string {
	return strings.TrimSpace(s)
}
