package braintrust_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xdlc-labs/airlock/internal/braintrust"
)

func TestImportFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rows.jsonl")
	data := `{"input":"hello","expected":"hi"}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	cases, err := braintrust.ImportFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 1 || cases[0].Expect.Contains != "hi" {
		t.Fatalf("unexpected cases: %+v", cases)
	}
}
