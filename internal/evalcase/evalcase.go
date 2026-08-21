package evalcase

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Case is one evaluation unit (JSONL line).
type Case struct {
	ID     string         `json:"id"`
	Agent  string         `json:"agent,omitempty"`
	Input  Input          `json:"input"`
	Expect Expect         `json:"expect"`
	Tags   []string       `json:"tags,omitempty"`
	Meta   map[string]any `json:"meta,omitempty"`
}

type Input struct {
	Messages []Message `json:"messages"`
	Model    string    `json:"model,omitempty"`
	Provider string    `json:"provider,omitempty"` // openai|anthropic|mock
	Tools    []ToolDef `json:"tools,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// Expect holds deterministic assertions.
type Expect struct {
	ExactMatch     string `json:"exact_match,omitempty"`
	Contains       string `json:"contains,omitempty"`
	NotContains    string `json:"not_contains,omitempty"`
	Regex          string `json:"regex,omitempty"`
	JSONValid      bool   `json:"json_valid,omitempty"`
	ToolArgsSchema string `json:"tool_args_schema,omitempty"`
	ToolName       string `json:"tool_name,omitempty"`
	TaskSuccess    *bool  `json:"task_success,omitempty"`
	Judge          string `json:"judge,omitempty"` // judge id from registry
}

// Suite is suite.yml metadata.
type Suite struct {
	K                 int     `yaml:"k" json:"k"`
	MaxCostUSD        float64 `yaml:"max_cost_usd" json:"max_cost_usd"`
	Seed              int64   `yaml:"seed" json:"seed"`
	MinSamples        int     `yaml:"min_samples" json:"min_samples"`
	MaxSamplesPerCase int     `yaml:"max_samples_per_case" json:"max_samples_per_case"`
	CasesPath         string  `yaml:"cases" json:"cases"`             // relative to suite dir
	Mode              string  `yaml:"mode" json:"mode"`               // replay|record|live
	ReplayMiss        string  `yaml:"replay_miss" json:"replay_miss"` // fail|live
	Kind              string  `yaml:"kind" json:"kind"`               // ""|adversarial
}

func DefaultSuite() Suite {
	return Suite{
		K:                 5,
		MinSamples:        2,
		MaxSamplesPerCase: 5,
		CasesPath:         "default.jsonl",
		Mode:              "replay",
		ReplayMiss:        "fail",
		Seed:              42,
	}
}

// LoadSuite reads suite.yml from dir (or defaults) and loads cases.
func LoadSuite(dir string) (Suite, []Case, error) {
	return LoadSuiteFile(filepath.Join(dir, "suite.yml"))
}

// LoadSuiteFile reads a specific suite YAML path and loads cases relative to its directory.
func LoadSuiteFile(suitePath string) (Suite, []Case, error) {
	s := DefaultSuite()
	dir := filepath.Dir(suitePath)
	if data, err := os.ReadFile(suitePath); err == nil {
		if err := yaml.Unmarshal(data, &s); err != nil {
			return s, nil, fmt.Errorf("%s: %w", suitePath, err)
		}
	} else if !os.IsNotExist(err) {
		return s, nil, err
	}
	if s.K <= 0 {
		s.K = 5
	}
	if s.MaxSamplesPerCase <= 0 {
		s.MaxSamplesPerCase = s.K
	}
	if s.MinSamples <= 0 {
		s.MinSamples = 1
	}
	if s.CasesPath == "" {
		s.CasesPath = "default.jsonl"
	}
	if s.Mode == "" {
		s.Mode = "replay"
	}
	casesPath := s.CasesPath
	if !filepath.IsAbs(casesPath) {
		casesPath = filepath.Join(dir, casesPath)
	}
	cases, err := LoadJSONL(casesPath)
	return s, cases, err
}

func LoadJSONL(path string) ([]Case, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Case
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var c Case
		if err := json.Unmarshal(line, &c); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, lineNo, err)
		}
		if c.ID == "" {
			return nil, fmt.Errorf("%s:%d: missing id", path, lineNo)
		}
		out = append(out, c)
	}
	return out, sc.Err()
}

func WriteJSONL(path string, cases []Case) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, c := range cases {
		if err := enc.Encode(c); err != nil {
			return err
		}
	}
	return nil
}

func FilterByAgents(cases []Case, agents []string) []Case {
	if len(agents) == 0 {
		return cases
	}
	allow := map[string]bool{}
	for _, a := range agents {
		allow[a] = true
	}
	var out []Case
	for _, c := range cases {
		if c.Agent == "" || allow[c.Agent] {
			out = append(out, c)
		}
	}
	return out
}

func FilterByTag(cases []Case, tag string) []Case {
	if tag == "" {
		return cases
	}
	var out []Case
	for _, c := range cases {
		for _, t := range c.Tags {
			if t == tag {
				out = append(out, c)
				break
			}
		}
	}
	return out
}
