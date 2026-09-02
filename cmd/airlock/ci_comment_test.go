package main

import (
	"os"
	"path/filepath"
	"strings"
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
	data, statErr := os.ReadFile(commentPath)
	if statErr != nil {
		t.Fatalf("expected ci-comment.md to be written even though the gate failed: %v", statErr)
	}
	// A reviewer reading only the PR comment must be able to unblock the PR
	// without digging through CI logs for the exact approve command.
	wantRemediation := "airlock approve --base " + base.ID + " --head working"
	if !strings.Contains(string(data), wantRemediation) {
		t.Fatalf("comment missing remediation command %q, got:\n%s", wantRemediation, data)
	}
}

// TestCmdCIApprovalRemediationOmittedOnceApproved guards that the comment
// stops nagging for an `airlock approve` command once a ledger entry already
// covers this exact base/head pair.
func TestCmdCIApprovalRemediationOmittedOnceApproved(t *testing.T) {
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
	if err := cmdApprove([]string{"--path", root, "--base", base.ID, "--head", "working"}); err != nil {
		t.Fatalf("approve: %v", err)
	}

	if err := cmdCI([]string{"--path", root, "--base", base.ID, "--fail-on-approval", "--comment", "--skip-eval"}); err != nil {
		t.Fatalf("expected cmdCI to pass once approved: %v", err)
	}
	commentPath := filepath.Join(store.ForRoot(root).Airlock, "ci-comment.md")
	data, statErr := os.ReadFile(commentPath)
	if statErr != nil {
		t.Fatalf("read comment: %v", statErr)
	}
	if strings.Contains(string(data), "airlock approve") {
		t.Fatalf("comment should not nag for approval once already approved, got:\n%s", data)
	}
}
