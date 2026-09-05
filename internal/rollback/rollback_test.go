package rollback_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/xdlc-labs/airlock/internal/manifest"
	"github.com/xdlc-labs/airlock/internal/rollback"
	"github.com/xdlc-labs/airlock/internal/store"
)

func TestRestoreAndHint(t *testing.T) {
	root := t.TempDir()
	p := store.ForRoot(root)
	_ = p.Ensure()
	snap := manifest.Snapshot{
		ID: "abc",
		Manifest: manifest.Manifest{
			Version: 1,
			Models:  []manifest.Model{{ID: "m1", Model: "gpt-test", ContentHash: "h"}},
		},
	}
	data, _ := json.Marshal(snap)
	_ = os.WriteFile(filepath.Join(p.Snapshots, "abc.json"), data, 0o644)
	got, err := rollback.RestoreManifest(root, "abc")
	if err != nil {
		t.Fatal(err)
	}
	if got.Manifest.Models[0].Model != "gpt-test" {
		t.Fatal(got.Manifest.Models)
	}
	path, err := rollback.WriteRoutingHint(p.Airlock, rollback.Decision{
		PreferSnapshotID: "abc",
		PreferModel:      rollback.PreferModelFromSnapshot(got),
		Reason:           "test",
	})
	if err != nil || path == "" {
		t.Fatal(err)
	}
}
