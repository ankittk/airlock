package diff_test

import (
	"testing"

	"github.com/ankittk/airlock/internal/diff"
	"github.com/ankittk/airlock/internal/manifest"
)

func TestNeedsApprovalOnWriteTool(t *testing.T) {
	base := &manifest.Snapshot{
		ID: "base",
		Artifacts: []manifest.ArtifactRef{
			{Kind: "tool", ID: "search", Hash: "aaa"},
		},
		Manifest: manifest.Manifest{
			Tools: []manifest.Tool{{ID: "search", Name: "search", SchemaHash: "aaa", SideEffect: "read"}},
		},
	}
	head := &manifest.Snapshot{
		ID: "head",
		Artifacts: []manifest.ArtifactRef{
			{Kind: "tool", ID: "search", Hash: "aaa"},
			{Kind: "tool", ID: "send-email", Hash: "bbb"},
		},
		Manifest: manifest.Manifest{
			Tools: []manifest.Tool{
				{ID: "search", Name: "search", SchemaHash: "aaa", SideEffect: "read"},
				{ID: "send-email", Name: "send_email", SchemaHash: "bbb", SideEffect: "write"},
			},
		},
	}
	r := diff.Compare(base, head)
	if !r.NeedsApproval {
		t.Fatalf("expected needs approval: %+v", r)
	}
}

func TestNeedsApprovalOnSkillChange(t *testing.T) {
	base := &manifest.Snapshot{
		ID: "base",
		Artifacts: []manifest.ArtifactRef{
			{Kind: "skill", ID: "ship-check", Hash: "aaa"},
		},
		Manifest: manifest.Manifest{
			Skills: []manifest.Skill{{ID: "ship-check", Name: "ship-check", ContentHash: "aaa"}},
		},
	}
	head := &manifest.Snapshot{
		ID: "head",
		Artifacts: []manifest.ArtifactRef{
			{Kind: "skill", ID: "ship-check", Hash: "bbb"},
		},
		Manifest: manifest.Manifest{
			Skills: []manifest.Skill{{ID: "ship-check", Name: "ship-check", ContentHash: "bbb"}},
		},
	}
	r := diff.Compare(base, head)
	if !r.NeedsApproval {
		t.Fatalf("expected needs approval on skill change: %+v", r)
	}
	found := false
	for _, reason := range r.ApprovalReasons {
		if reason == "skill content changed: ship-check" {
			found = true
		}
	}
	if !found {
		t.Fatalf("reasons: %v", r.ApprovalReasons)
	}
}
