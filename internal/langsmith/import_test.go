package langsmith_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ankittk/airlock/internal/langsmith"
)

func TestImportFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dataset.json")
	data := `{"examples":[{"inputs":{"question":"hello"},"outputs":{"answer":"hi"}}]}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	cases, err := langsmith.ImportFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 1 || cases[0].Expect.Contains != "hi" {
		t.Fatalf("unexpected cases: %+v", cases)
	}
}
