package approval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Record is a local human approval for a snapshot pair that needs review.
type Record struct {
	Base      string    `json:"base"`
	Head      string    `json:"head"`
	Reasons   []string  `json:"reasons,omitempty"`
	DecidedBy string    `json:"decided_by,omitempty"`
	Note      string    `json:"note,omitempty"`
	Ts        time.Time `json:"ts"`
}

func Filename(base, head string) string {
	safe := func(s string) string {
		s = strings.ReplaceAll(s, "/", "_")
		s = strings.ReplaceAll(s, ":", "_")
		return s
	}
	return safe(base) + "__" + safe(head) + ".json"
}

func Path(dir, base, head string) string {
	return filepath.Join(dir, Filename(base, head))
}

func Write(dir string, rec Record) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if rec.Ts.IsZero() {
		rec.Ts = time.Now().UTC()
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(dir, rec.Base, rec.Head), append(data, '\n'), 0o644)
}

func Has(dir, base, head string) bool {
	_, err := os.Stat(Path(dir, base, head))
	return err == nil
}

func Load(dir, base, head string) (Record, error) {
	data, err := os.ReadFile(Path(dir, base, head))
	if err != nil {
		return Record{}, err
	}
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return Record{}, err
	}
	return rec, nil
}

func Require(dir, base, head string, needs bool) error {
	if !needs {
		return nil
	}
	if Has(dir, base, head) {
		return nil
	}
	return fmt.Errorf("NEEDS_APPROVAL without ledger entry (run: airlock approve --base %s --head %s)", base, head)
}
