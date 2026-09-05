package braintrust

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/xdlc-labs/airlock/internal/evalcase"
)

// ImportFile reads Braintrust JSONL export into Airlock eval cases.
func ImportFile(path string) ([]evalcase.Case, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []evalcase.Case
	lineNo := 0
	for _, line := range splitLines(string(data)) {
		line = trim(line)
		lineNo++
		if line == "" {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		in := stringify(row["input"])
		if in == "" {
			in = stringify(row["inputs"])
		}
		if in == "" {
			continue
		}
		exp := stringify(row["expected"])
		if exp == "" {
			exp = stringify(row["output"])
		}
		id := fmt.Sprintf("bt-%d", lineNo)
		if s, ok := row["id"].(string); ok && s != "" {
			id = "bt-" + s
		}
		c := evalcase.Case{
			ID:    id,
			Agent: "default",
			Input: evalcase.Input{
				Provider: "mock",
				Messages: []evalcase.Message{{Role: "user", Content: in}},
			},
			Tags: []string{"braintrust", "import"},
		}
		if exp != "" {
			c.Expect.Contains = exp
		} else {
			c.Expect.JSONValid = true
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("braintrust export: no rows")
	}
	return out, nil
}

func stringify(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		b, _ := json.Marshal(t)
		return string(b)
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprint(v)
	}
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
