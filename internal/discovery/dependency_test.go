package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAPMPackagesBecomeDependencies verifies that apm.lock.yaml `packages`
// entries with a non-AI-artifact kind (npm/pip/go/…) — or no kind at all —
// are captured as manifest.Dependency instead of silently dropped. That is
// what feeds the agent-driven supply-chain gate in internal/diff.
func TestAPMPackagesBecomeDependencies(t *testing.T) {
	root := t.TempDir()
	lock := `
version: 1
packages:
  left-pad:
    name: left-pad
    version: 1.3.0
    kind: npm
  no-kind-pkg:
    name: no-kind-pkg
    version: 2.0.0
prompts: {}
skills: {}
mcp: {}
dependencies: {}
`
	if err := os.WriteFile(filepath.Join(root, "apm.lock.yaml"), []byte(lock), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := Scan(root)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	got := map[string]string{} // id -> ecosystem
	for _, d := range m.Dependencies {
		got[d.ID] = d.Ecosystem
	}
	if eco, ok := got["left-pad"]; !ok || eco != "npm" {
		t.Fatalf("expected left-pad dependency with ecosystem npm, got %+v", got)
	}
	if eco, ok := got["no-kind-pkg"]; !ok || eco != "package" {
		t.Fatalf("expected no-kind-pkg dependency defaulting to ecosystem package, got %+v", got)
	}
	for _, d := range m.Dependencies {
		if d.Hash == "" {
			t.Fatalf("dependency %s missing hash", d.ID)
		}
	}
}
