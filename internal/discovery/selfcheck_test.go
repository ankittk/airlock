package discovery_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xdlc-labs/airlock/internal/diff"
	"github.com/xdlc-labs/airlock/internal/discovery"
	"github.com/xdlc-labs/airlock/internal/snapshot"
)

// Self-check: import fixture → snapshot → mutate prompt → diff lists prompt + agent.
func TestPhase0SelfCheck(t *testing.T) {
	src := filepath.Join("..", "..", "testdata", "toy-agent")
	src, err := filepath.Abs(src)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := copyTree(src, dir); err != nil {
		t.Fatal(err)
	}

	m, err := discovery.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Agents) == 0 {
		t.Fatal("expected agents from apm.lock.yaml")
	}
	if len(m.Prompts) == 0 {
		t.Fatal("expected prompts")
	}
	if len(m.Skills) == 0 {
		t.Fatal("expected skills from apm.lock / SKILL.md")
	}
	foundSkill := false
	for _, s := range m.Skills {
		if s.ID == "search-orders" || strings.Contains(s.Path, "SKILL.md") {
			foundSkill = true
		}
	}
	if !foundSkill {
		t.Fatalf("expected search-orders skill, got %+v", m.Skills)
	}
	foundRule := false
	for _, p := range m.Prompts {
		if p.Source == "cursor-rules" {
			foundRule = true
		}
	}
	if !foundRule {
		t.Fatal("expected cursor-rules prompt from .cursor/rules")
	}

	base, err := snapshot.Create(dir, true)
	if err != nil {
		t.Fatal(err)
	}

	promptPath := filepath.Join(dir, "prompts", "system.md")
	if err := os.WriteFile(promptPath, []byte("You are a DIFFERENT support agent.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	head, err := snapshot.FromWorkingTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	r := diff.Compare(base, head)
	if !diff.HasChanges(r) {
		t.Fatal("expected changes after prompt mutation")
	}

	foundPrompt := false
	for _, c := range r.Changes {
		if c.Kind == "prompt" && c.Status == "changed" {
			foundPrompt = true
		}
	}
	if !foundPrompt {
		t.Fatalf("expected changed prompt in diff: %+v", r.Changes)
	}

	foundAgent := false
	for _, a := range r.AffectedAgents {
		if a == "support-bot" || a == "default" {
			foundAgent = true
		}
	}
	if !foundAgent {
		t.Fatalf("expected blast radius to include support-bot, got %v", r.AffectedAgents)
	}
}

func TestScanSkillsWithoutAPM(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".claude", "skills", "ship-check")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte("# Ship check\nVerify release gates.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := discovery.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Skills) != 1 {
		t.Fatalf("want 1 skill, got %+v", m.Skills)
	}
	if m.Skills[0].ID != "skill-ship-check" {
		t.Fatalf("id: %s", m.Skills[0].ID)
	}

	base, err := snapshot.FromWorkingTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skillPath, []byte("# Ship check\nVerify release gates HARDER.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	head, err := snapshot.FromWorkingTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	r := diff.Compare(base, head)
	if !diff.HasKind(r, "skill") {
		t.Fatalf("expected skill change, got %+v", r.Changes)
	}
}

// A skill is commonly SKILL.md plus scripts/resources beside it. A change to
// one of those, with SKILL.md itself untouched, must still register as a
// skill change — skill hashing must cover the whole directory, not just the
// entry-point file.
func TestScanSkillsDetectsSiblingScriptChange(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".claude", "skills", "ship-check")
	scriptsDir := filepath.Join(skillDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Ship check\nVerify release gates.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join(scriptsDir, "check.sh")
	if err := os.WriteFile(scriptPath, []byte("echo v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	base, err := snapshot.FromWorkingTree(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Only the sibling script changes — SKILL.md stays byte-for-byte the same.
	if err := os.WriteFile(scriptPath, []byte("echo v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	head, err := snapshot.FromWorkingTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	r := diff.Compare(base, head)
	if !diff.HasKind(r, "skill") {
		t.Fatalf("expected sibling script edit to register as a skill change, got %+v", r.Changes)
	}
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}
