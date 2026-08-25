package evalcase

import (
	"os"
	"path/filepath"
)

// PromoteOptions controls run → eval-case promotion.
type PromoteOptions struct {
	FailuresOnly bool
	Tag          string
	Limit        int
}

func appendCases(outPath string, newCases []Case) (int, error) {
	if len(newCases) == 0 {
		return 0, nil
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return 0, err
	}
	existing, _ := LoadJSONL(outPath)
	seen := map[string]bool{}
	for _, c := range existing {
		seen[c.ID] = true
	}
	added := 0
	for _, c := range newCases {
		if seen[c.ID] {
			continue
		}
		existing = append(existing, c)
		seen[c.ID] = true
		added++
	}
	if err := WriteJSONL(outPath, existing); err != nil {
		return 0, err
	}
	return added, nil
}

// WriteBindingsStub writes a commented eval-bindings.yml if missing.
func WriteBindingsStub(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	stub := `# Artifact → suite bindings (Phase 4 eval flexibility)
version: 1
bindings:
  - when: { kind: prompt }
    suite: suite.yml
  - when: { kind: mcp }
    suite: suite.adversarial.yml
  - when: { kind: skill }
    suite: suite.adversarial.yml
`
	return os.WriteFile(path, []byte(stub), 0o644)
}

// AppendCases merges new cases into a JSONL file (dedupe by id).
func AppendCases(outPath string, newCases []Case) (int, error) {
	return appendCases(outPath, newCases)
}
