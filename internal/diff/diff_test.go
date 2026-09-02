package diff_test

import (
	"strings"
	"testing"

	"github.com/ankittk/airlock/internal/diff"
	"github.com/ankittk/airlock/internal/manifest"
)

func TestCompareDetectsHashChange(t *testing.T) {
	base := &manifest.Snapshot{
		ID: "base",
		Artifacts: []manifest.ArtifactRef{
			{Kind: "prompt", ID: "p1", Hash: "aaa"},
		},
		Manifest: manifest.Manifest{
			Agents: []manifest.Agent{{ID: "a1", Prompts: []string{"p1"}}},
			Graph:  []manifest.Edge{{From: "agent:a1", To: "prompt:p1"}},
		},
	}
	head := &manifest.Snapshot{
		ID: "head",
		Artifacts: []manifest.ArtifactRef{
			{Kind: "prompt", ID: "p1", Hash: "bbb"},
		},
		Manifest: base.Manifest,
	}
	r := diff.Compare(base, head)
	if len(r.Changes) != 1 || r.Changes[0].Status != "changed" {
		t.Fatalf("got %+v", r.Changes)
	}
	if len(r.AffectedAgents) != 1 || r.AffectedAgents[0] != "a1" {
		t.Fatalf("blast radius %+v", r.AffectedAgents)
	}
	if !diff.HasKind(r, "prompt") {
		t.Fatal("expected prompt kind")
	}
	if diff.HasKind(r, "mcp") {
		t.Fatal("unexpected mcp")
	}
}

func TestFormatCommentIncludesEvalBlastRadius(t *testing.T) {
	base := &manifest.Snapshot{
		ID:        "base",
		Artifacts: []manifest.ArtifactRef{{Kind: "prompt", ID: "p1", Hash: "aaa"}},
		Manifest: manifest.Manifest{
			Agents: []manifest.Agent{{ID: "a1", Prompts: []string{"p1"}, Evals: []string{"e1"}}},
		},
	}
	head := &manifest.Snapshot{
		ID:        "head",
		Artifacts: []manifest.ArtifactRef{{Kind: "prompt", ID: "p1", Hash: "bbb"}},
		Manifest:  base.Manifest,
	}
	r := diff.Compare(base, head)
	if len(r.AffectedEvals) == 0 {
		t.Fatalf("expected affected evals, got %+v", r)
	}
	body := diff.FormatComment(r)
	if !strings.Contains(body, "Blast radius: evals **e1**") {
		t.Fatalf("PR comment missing evals blast radius, got:\n%s", body)
	}
}
