package collector

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/ankittk/airlock/internal/evalcase"
	"github.com/ankittk/airlock/internal/manifest"
)

// Span is a minimal OTel GenAI-ish JSONL record.
type Span struct {
	Name       string         `json:"name"`
	Attributes map[string]any `json:"attributes"`
	// flattened conveniences
	Model   string `json:"model,omitempty"`
	Input   string `json:"input,omitempty"`
	Output  string `json:"output,omitempty"`
	Success *bool  `json:"success,omitempty"`
	Agent   string `json:"agent,omitempty"`
}

var (
	reEmail = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	reKey   = regexp.MustCompile(`(?i)(sk-[a-zA-Z0-9]{16,}|api[_-]?key[=:]\s*\S+)`)
	reCC    = regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)
	rePhone = regexp.MustCompile(`\b(?:\+?1[-.\s]?)?(?:\(?\d{3}\)?[-.\s]?)\d{3}[-.\s]?\d{4}\b`)
	reSSN   = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
)

type Options struct {
	HashStrings bool
	Mode        string // pii|hash|off (empty = pii)
}

// ParseRedact maps CLI --redact values to Options.
func ParseRedact(s string) Options {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "off", "none", "false":
		return Options{Mode: "off"}
	case "hash":
		return Options{Mode: "hash", HashStrings: true}
	default:
		return Options{Mode: "pii"}
	}
}

// Redact replaces PII-ish patterns locally.
func Redact(s string, opt Options) string {
	if opt.Mode == "off" {
		return s
	}
	if opt.Mode == "" || opt.Mode == "pii" || opt.Mode == "hash" || opt.HashStrings {
		s = reEmail.ReplaceAllString(s, "[REDACTED_EMAIL]")
		s = reKey.ReplaceAllString(s, "[REDACTED_KEY]")
		s = rePhone.ReplaceAllString(s, "[REDACTED_PHONE]")
		s = reSSN.ReplaceAllString(s, "[REDACTED_SSN]")
		s = reCC.ReplaceAllStringFunc(s, func(m string) string {
			digits := 0
			for _, r := range m {
				if r >= '0' && r <= '9' {
					digits++
				}
			}
			if digits >= 13 {
				return "[REDACTED_CARD]"
			}
			return m
		})
	}
	if opt.HashStrings || opt.Mode == "hash" {
		s = "[HASH:" + manifest.HashString(s)[:12] + "]"
	}
	return s
}

// LoadJSONL reads OTel-ish spans.
func LoadJSONL(path string) ([]Span, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Span
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		b := strings.TrimSpace(sc.Text())
		if b == "" {
			continue
		}
		var sp Span
		if err := json.Unmarshal([]byte(b), &sp); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, lineNo, err)
		}
		normalize(&sp)
		out = append(out, sp)
	}
	return out, sc.Err()
}

func normalize(sp *Span) {
	if sp.Attributes == nil {
		return
	}
	get := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := sp.Attributes[k]; ok {
				switch t := v.(type) {
				case string:
					return t
				}
			}
		}
		return ""
	}
	if sp.Model == "" {
		sp.Model = get("gen_ai.request.model", "llm.model", "model")
	}
	if sp.Input == "" {
		sp.Input = get("gen_ai.prompt", "gen_ai.input.messages", "input", "prompt")
	}
	if sp.Output == "" {
		sp.Output = get("gen_ai.completion", "gen_ai.output.messages", "output", "completion")
	}
	if sp.Agent == "" {
		sp.Agent = get("airlock.agent", "agent", "service.name")
	}
	if sp.Success == nil {
		if v, ok := sp.Attributes["gen_ai.response.success"]; ok {
			if b, ok := v.(bool); ok {
				sp.Success = &b
			}
		}
	}
}

// ToCases converts redacted spans into eval cases (user message = input; expect json_valid if output looks json).
func ToCases(spans []Span, opt Options, limit int) []evalcase.Case {
	if limit <= 0 {
		limit = len(spans)
	}
	out := make([]evalcase.Case, 0, min(limit, len(spans)))
	for i, sp := range spans {
		if i >= limit {
			break
		}
		in := Redact(sp.Input, opt)
		if in == "" {
			continue
		}
		agent := sp.Agent
		if agent == "" {
			agent = "default"
		}
		id := fmt.Sprintf("ingest-%03d", i+1)
		c := evalcase.Case{
			ID:    id,
			Agent: agent,
			Input: evalcase.Input{
				Provider: "mock",
				Model:    sp.Model,
				Messages: []evalcase.Message{{Role: "user", Content: in}},
			},
			Tags: []string{"ingest", "otel"},
			Expect: evalcase.Expect{
				JSONValid: true, // weak default proxy for structured apps
			},
		}
		if sp.Success != nil {
			c.Expect.TaskSuccess = sp.Success
			c.Expect.JSONValid = false
		}
		out = append(out, c)
	}
	return out
}

// WriteIngest dumps redacted spans as JSONL.
func WriteIngest(path string, spans []Span, opt Options) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, sp := range spans {
		sp.Input = Redact(sp.Input, opt)
		sp.Output = Redact(sp.Output, opt)
		if err := enc.Encode(sp); err != nil {
			return err
		}
	}
	return nil
}

// SuccessRate from spans with Success set.
func SuccessRate(spans []Span) (ok, n int) {
	for _, sp := range spans {
		if sp.Success == nil {
			continue
		}
		n++
		if *sp.Success {
			ok++
		}
	}
	return ok, n
}
