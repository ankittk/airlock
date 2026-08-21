package rollback

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ankittk/airlock/internal/manifest"
	"github.com/ankittk/airlock/internal/store"
)

// Decision is a machine-readable routing hint for gateways (LiteLLM/Portkey/etc).
type Decision struct {
	PreferModel      string    `json:"prefer_model,omitempty"`
	PreferSnapshotID string    `json:"prefer_snapshot_id"`
	Reason           string    `json:"reason"`
	Ts               time.Time `json:"ts"`
}

// RestoreManifest copies the snapshot's manifest into the working .airlock/manifest.json
// (known-good re-approval of controllable inputs).
func RestoreManifest(root, snapshotID string) (*manifest.Snapshot, error) {
	p := store.ForRoot(root)
	path := filepath.Join(p.Snapshots, snapshotID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("snapshot %s: %w", snapshotID, err)
	}
	var snap manifest.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	if err := p.Ensure(); err != nil {
		return nil, err
	}
	if err := store.WriteManifest(p, &snap.Manifest); err != nil {
		return nil, err
	}
	return &snap, nil
}

// WriteRoutingHint writes routing_decision.json under .airlock/.
func WriteRoutingHint(airlockDir string, d Decision) (string, error) {
	if d.Ts.IsZero() {
		d.Ts = time.Now().UTC()
	}
	path := filepath.Join(airlockDir, "routing_decision.json")
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// PreferModelFromSnapshot picks the first model string from the snapshot manifest.
func PreferModelFromSnapshot(snap *manifest.Snapshot) string {
	if snap == nil {
		return ""
	}
	for _, m := range snap.Manifest.Models {
		if m.Model != "" {
			return m.Model
		}
	}
	return ""
}
