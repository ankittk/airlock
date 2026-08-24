package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ankittk/airlock/internal/snapshot"
	"github.com/ankittk/airlock/internal/store"
)

// TestCmdCIFailOnApprovalStillGatesWithComment guards a regression where
// `airlock ci --comment` returned before the --fail-on-approval / --fail-on-eval
// / --fail-on-ai_change checks ran, silently exiting 0 on NEEDS_APPROVAL as
// long as --comment was also passed. --comment must only change the stdout
// format, never bypass the gate, and the PR-comment file must still be written.
func TestCmdCIFailOnApprovalStillGatesWithComment(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "prompts"), 0o755); err != nil {
		t.Fatal(err)
	}
	promptPath := filepath.Join(root, "prompts", "system.md")
	if err := os.WriteFile(promptPath, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}

	base, err := snapshot.Create(root, true)
	if err != nil {
		t.Fatalf("baseline snapshot: %v", err)
	}

	// Edit the prompt (AI-artifact change) and add a package dependency in
	// the same PR — the co-occurrence that should raise NEEDS_APPROVAL.
	if err := os.WriteFile(promptPath, []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	lock := `version: 1
packages:
  left-pad:
    name: left-pad
    version: 1.3.0
    kind: npm
`
	if err := os.WriteFile(filepath.Join(root, "apm.lock.yaml"), []byte(lock), 0o644); err != nil {
		t.Fatal(err)
	}

	err = cmdCI([]string{"--path", root, "--base", base.ID, "--fail-on-approval", "--comment", "--skip-eval"})
	if err == nil {
		t.Fatal("expected cmdCI to fail closed on NEEDS_APPROVAL even with --comment")
	}

	commentPath := filepath.Join(store.ForRoot(root).Airlock, "ci-comment.md")
	if _, statErr := os.Stat(commentPath); statErr != nil {
		t.Fatalf("expected ci-comment.md to be written even though the gate failed: %v", statErr)
	}
}
