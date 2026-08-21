package judge

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMockJudgeAndKappa(t *testing.T) {
	dir := t.TempDir()
	spec := Spec{ID: "default", Provider: "mock", Rubric: "json_valid", KappaFloor: 0.2}
	data, _ := json.Marshal(spec)
	_ = os.WriteFile(filepath.Join(dir, "default.json"), data, 0o644)
	reg, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	items := []CalibrationItem{
		{Input: "a", Output: `{"ok":true}`, Label: true},
		{Input: "b", Output: `{"ok":true}`, Label: true},
		{Input: "c", Output: `not-json`, Label: false},
		{Input: "d", Output: `also-bad`, Label: false},
	}
	rep, err := reg.Calibrate(context.Background(), "default", items)
	if err != nil {
		t.Fatal(err)
	}
	if rep.N != 4 || !rep.FloorOK {
		t.Fatalf("%+v", rep)
	}
}
