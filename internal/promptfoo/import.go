package promptfoo

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/xdlc-labs/airlock/internal/evalcase"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Prompts   []any  `yaml:"prompts"`
	Providers []any  `yaml:"providers"`
	Tests     []Test `yaml:"tests"`
}

type Test struct {
	Vars        map[string]any `yaml:"vars"`
	Assert      []Assertion    `yaml:"assert"`
	Description string         `yaml:"description"`
}

type Assertion struct {
	Type      string  `yaml:"type"`
	Value     any     `yaml:"value"`
	Threshold float64 `yaml:"threshold"`
}

type Result struct {
	Cases    []evalcase.Case
	Warnings []string
}

// ImportFile converts promptfoo YAML into Airlock cases.
func ImportFile(path string) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Result{}, err
	}
	prompt := firstPrompt(cfg.Prompts)
	provider, model := firstProvider(cfg.Providers)

	var out Result
	for i, t := range cfg.Tests {
		id := t.Description
		if id == "" {
			id = "pf-" + strconv.Itoa(i+1)
		}
		user := prompt
		if v, ok := t.Vars["prompt"].(string); ok {
			user = v
		} else if v, ok := t.Vars["query"].(string); ok {
			user = v
		} else if v, ok := t.Vars["message"].(string); ok {
			user = v
		} else if prompt != "" && len(t.Vars) > 0 {
			for _, val := range t.Vars {
				if s, ok := val.(string); ok {
					user = s
					break
				}
			}
		}
		c := evalcase.Case{
			ID:    id,
			Agent: "default",
			Input: evalcase.Input{
				Provider: provider,
				Model:    model,
				Messages: []evalcase.Message{
					{Role: "user", Content: user},
				},
			},
			Tags: []string{"promptfoo"},
		}
		for _, a := range t.Assert {
			warn := applyAssert(&c.Expect, a)
			if warn != "" {
				out.Warnings = append(out.Warnings, fmt.Sprintf("%s: %s", id, warn))
			}
		}
		out.Cases = append(out.Cases, c)
	}
	if len(out.Cases) == 0 {
		return out, fmt.Errorf("no tests found in %s", path)
	}
	return out, nil
}

func applyAssert(e *evalcase.Expect, a Assertion) string {
	switch strings.ToLower(a.Type) {
	case "equals", "exact":
		if s, ok := a.Value.(string); ok {
			e.ExactMatch = s
		}
	case "contains":
		if s, ok := a.Value.(string); ok {
			e.Contains = s
		}
	case "not-contains", "not_contains":
		if s, ok := a.Value.(string); ok {
			e.NotContains = s
		}
	case "icontains":
		if s, ok := a.Value.(string); ok {
			// ponytail: store as contains; scoring is case-sensitive — note in warning
			e.Contains = s
			return "icontains imported as contains (case-sensitive)"
		}
	case "regex", "regex-match":
		if s, ok := a.Value.(string); ok {
			e.Regex = s
		}
	case "is-json", "is-json-object":
		e.JSONValid = true
	case "javascript", "llm-rubric", "similar", "cost", "latency":
		return fmt.Sprintf("unsupported assert %q skipped", a.Type)
	default:
		if a.Type != "" {
			return fmt.Sprintf("unknown assert %q skipped", a.Type)
		}
	}
	return ""
}

func firstPrompt(prompts []any) string {
	for _, p := range prompts {
		if s, ok := p.(string); ok {
			return s
		}
	}
	return ""
}

func firstProvider(providers []any) (provider, model string) {
	provider = "mock"
	for _, p := range providers {
		switch v := p.(type) {
		case string:
			if before, after, ok := strings.Cut(v, ":"); ok {
				return before, after
			}
			if v != "" {
				provider = v
			}
		case map[string]any:
			if id, ok := v["id"].(string); ok {
				if before, after, ok := strings.Cut(id, ":"); ok {
					return before, after
				}
				return id, ""
			}
		}
	}
	return provider, model
}
