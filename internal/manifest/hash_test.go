package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashBytesStable(t *testing.T) {
	a := HashBytes([]byte("hello"))
	b := HashBytes([]byte("hello"))
	if a != b {
		t.Fatalf("unstable hash: %s vs %s", a, b)
	}
	if len(a) != 64 {
		t.Fatalf("want 64 hex chars, got %d", len(a))
	}
	if HashBytes([]byte("hello")) == HashBytes([]byte("world")) {
		t.Fatal("different inputs same hash")
	}
}

// A skill (or any directory-shaped artifact) is its entry-point file plus
// whatever scripts/resources sit beside it. HashDirTree must change when a
// sibling file changes, not just when the entry point does — otherwise that
// change goes undetected by whoever hashes only the entry point.
func TestHashDirTreeChangesOnSiblingFileEdit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	scriptsDir := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join(scriptsDir, "run.sh")
	if err := os.WriteFile(scriptPath, []byte("echo v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	before, err := HashDirTree(dir)
	if err != nil {
		t.Fatal(err)
	}

	// SKILL.md itself is untouched — only the sibling script changes.
	if err := os.WriteFile(scriptPath, []byte("echo v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := HashDirTree(dir)
	if err != nil {
		t.Fatal(err)
	}

	if before == after {
		t.Fatal("HashDirTree did not change when a sibling script changed — script edits would go undetected")
	}
}

func TestHashDirTreeStableRegardlessOfWalkOrder(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	h1, err := HashDirTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := HashDirTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatalf("HashDirTree not stable across runs: %s vs %s", h1, h2)
	}
}

func TestValidateRequiresHashes(t *testing.T) {
	m := &Manifest{
		Version: 1,
		Models:  []Model{{ID: "m1", Model: "claude", ContentHash: ""}},
	}
	if err := Validate(m); err == nil {
		t.Fatal("expected error for empty content_hash")
	}
}
