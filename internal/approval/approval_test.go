package approval_test

import (
	"path/filepath"
	"testing"

	"github.com/ankittk/airlock/internal/approval"
)

func TestLedger(t *testing.T) {
	dir := t.TempDir()
	rec := approval.Record{Base: "a", Head: "b", Reasons: []string{"write tool"}, DecidedBy: "dev"}
	if err := approval.Write(dir, rec); err != nil {
		t.Fatal(err)
	}
	if !approval.Has(dir, "a", "b") {
		t.Fatal("missing")
	}
	got, err := approval.Load(dir, "a", "b")
	if err != nil || got.DecidedBy != "dev" {
		t.Fatalf("%+v %v", got, err)
	}
	if err := approval.Require(dir, "a", "b", true); err != nil {
		t.Fatal(err)
	}
	if err := approval.Require(filepath.Join(dir, "empty"), "x", "y", true); err == nil {
		t.Fatal("expected error")
	}
}
