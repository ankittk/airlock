package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanGoSumDependencies(t *testing.T) {
	root := t.TempDir()
	sum := "github.com/foo/bar v1.2.3 h1:abc123\n"
	if err := os.WriteFile(filepath.Join(root, "go.sum"), []byte(sum), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range m.Dependencies {
		if d.Ecosystem == "go" && d.Version == "v1.2.3" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected go.sum dependency, got %+v", m.Dependencies)
	}
}

func TestScanPackageLockDependencies(t *testing.T) {
	root := t.TempDir()
	lock := `{
  "packages": {
    "node_modules/left-pad": { "name": "left-pad", "version": "1.3.0" }
  }
}`
	if err := os.WriteFile(filepath.Join(root, "package-lock.json"), []byte(lock), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range m.Dependencies {
		if d.ID == "left-pad" && d.Ecosystem == "npm" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected npm dependency, got %+v", m.Dependencies)
	}
}

func TestScanCargoLockDependencies(t *testing.T) {
	root := t.TempDir()
	lock := `[[package]]
name = "serde"
version = "1.0.200"
`
	if err := os.WriteFile(filepath.Join(root, "Cargo.lock"), []byte(lock), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range m.Dependencies {
		if d.ID == "serde" && d.Ecosystem == "cargo" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected cargo dependency, got %+v", m.Dependencies)
	}
}
