package diff_test

import (
	"testing"

	"github.com/xdlc-labs/airlock/internal/diff"
	"github.com/xdlc-labs/airlock/internal/manifest"
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

func TestNeedsApprovalOnDependencyAddedWithAIChange(t *testing.T) {
	base := &manifest.Snapshot{
		ID: "base",
		Artifacts: []manifest.ArtifactRef{
			{Kind: "prompt", ID: "p1", Hash: "aaa"},
		},
		Manifest: manifest.Manifest{
			Prompts: []manifest.Prompt{{ID: "p1", ContentHash: "aaa"}},
		},
	}
	head := &manifest.Snapshot{
		ID: "head",
		Artifacts: []manifest.ArtifactRef{
			{Kind: "prompt", ID: "p1", Hash: "bbb"},
			{Kind: "dependency", ID: "left-pad", Hash: "ccc"},
		},
		Manifest: manifest.Manifest{
			Prompts:      []manifest.Prompt{{ID: "p1", ContentHash: "bbb"}},
			Dependencies: []manifest.Dependency{{ID: "left-pad", Ecosystem: "npm", Version: "1.3.0", Hash: "ccc"}},
		},
	}
	r := diff.Compare(base, head)
	if !r.NeedsApproval {
		t.Fatalf("expected needs approval when dep lands alongside AI change: %+v", r)
	}
	found := false
	for _, reason := range r.ApprovalReasons {
		if reason == "new dependency: left-pad" {
			found = true
		}
	}
	if !found {
		t.Fatalf("reasons: %v", r.ApprovalReasons)
	}
}

func TestDependencyAloneDoesNotNeedApproval(t *testing.T) {
	base := &manifest.Snapshot{
		ID:        "base",
		Artifacts: []manifest.ArtifactRef{},
		Manifest:  manifest.Manifest{},
	}
	head := &manifest.Snapshot{
		ID: "head",
		Artifacts: []manifest.ArtifactRef{
			{Kind: "dependency", ID: "left-pad", Hash: "ccc"},
		},
		Manifest: manifest.Manifest{
			Dependencies: []manifest.Dependency{{ID: "left-pad", Ecosystem: "npm", Version: "1.3.0", Hash: "ccc"}},
		},
	}
	r := diff.Compare(base, head)
	if r.NeedsApproval {
		t.Fatalf("dep-only PR should not need approval (that's SCA's job): %+v", r)
	}
}

// TestNeedsApprovalOnMCPNewLiveTool guards the gap where Permissions[] (only
// ever hand-maintained via apm.lock.yaml) stays unchanged but the server's
// live tools/list actually grew a new tool — previously invisible to
// permissionExpansion entirely.
func TestNeedsApprovalOnMCPNewLiveTool(t *testing.T) {
	base := &manifest.Snapshot{
		ID: "base",
		Artifacts: []manifest.ArtifactRef{
			{Kind: "mcp", ID: "fs", Hash: "aaa"},
		},
		Manifest: manifest.Manifest{
			MCPServers: []manifest.MCPServer{{ID: "fs", Name: "fs", SchemaHash: "aaa", ToolNames: []string{"read_file"}}},
		},
	}
	head := &manifest.Snapshot{
		ID: "head",
		Artifacts: []manifest.ArtifactRef{
			{Kind: "mcp", ID: "fs", Hash: "bbb"},
		},
		Manifest: manifest.Manifest{
			MCPServers: []manifest.MCPServer{{ID: "fs", Name: "fs", SchemaHash: "bbb", ToolNames: []string{"read_file", "delete_file"}}},
		},
	}
	r := diff.Compare(base, head)
	if !r.NeedsApproval {
		t.Fatalf("expected needs approval on new live MCP tool: %+v", r)
	}
	found := false
	for _, reason := range r.ApprovalReasons {
		if reason == "MCP new write-looking tool delete_file on fs" {
			found = true
		}
	}
	if !found {
		t.Fatalf("reasons: %v", r.ApprovalReasons)
	}
}

// TestMCPUnchangedLiveToolsNoApproval guards against false positives: same
// tool set, only SchemaHash churn (e.g. description wording), should not
// re-flag tools already present on both sides.
func TestMCPUnchangedLiveToolsNoApproval(t *testing.T) {
	base := &manifest.Snapshot{
		ID:        "base",
		Artifacts: []manifest.ArtifactRef{{Kind: "mcp", ID: "fs", Hash: "aaa"}},
		Manifest: manifest.Manifest{
			MCPServers: []manifest.MCPServer{{ID: "fs", Name: "fs", SchemaHash: "aaa", ToolNames: []string{"read_file"}}},
		},
	}
	head := &manifest.Snapshot{
		ID:        "head",
		Artifacts: []manifest.ArtifactRef{{Kind: "mcp", ID: "fs", Hash: "bbb"}},
		Manifest: manifest.Manifest{
			MCPServers: []manifest.MCPServer{{ID: "fs", Name: "fs", SchemaHash: "bbb", ToolNames: []string{"read_file"}}},
		},
	}
	r := diff.Compare(base, head)
	if r.NeedsApproval {
		t.Fatalf("same tool set should not need approval: %+v", r)
	}
}

func TestBenignToolNameNoLongerFalsePositive(t *testing.T) {
	base := &manifest.Snapshot{
		ID: "base",
		Artifacts: []manifest.ArtifactRef{
			{Kind: "tool", ID: "post-processing-helper", Hash: "aaa"},
		},
		Manifest: manifest.Manifest{
			Tools: []manifest.Tool{{ID: "post-processing-helper", Name: "post_processing_helper", SchemaHash: "aaa", SideEffect: "read"}},
		},
	}
	head := &manifest.Snapshot{
		ID: "head",
		Artifacts: []manifest.ArtifactRef{
			{Kind: "tool", ID: "post-processing-helper", Hash: "bbb"},
		},
		Manifest: manifest.Manifest{
			Tools: []manifest.Tool{{ID: "post-processing-helper", Name: "post_processing_helper", SchemaHash: "bbb", SideEffect: "read"}},
		},
	}
	r := diff.Compare(base, head)
	if r.NeedsApproval {
		t.Fatalf("read-only tool named with 'post' should not need approval: %+v", r)
	}
}

func TestDependencyVersionBumpAloneDoesNotNeedApproval(t *testing.T) {
	base := &manifest.Snapshot{
		ID: "base",
		Artifacts: []manifest.ArtifactRef{
			{Kind: "dependency", ID: "left-pad", Hash: "ccc"},
			{Kind: "prompt", ID: "p1", Hash: "aaa"},
		},
		Manifest: manifest.Manifest{
			Prompts:      []manifest.Prompt{{ID: "p1", ContentHash: "aaa"}},
			Dependencies: []manifest.Dependency{{ID: "left-pad", Ecosystem: "npm", Version: "1.3.0", Hash: "ccc"}},
		},
	}
	head := &manifest.Snapshot{
		ID: "head",
		Artifacts: []manifest.ArtifactRef{
			{Kind: "dependency", ID: "left-pad", Hash: "ddd"},
			{Kind: "prompt", ID: "p1", Hash: "aaa"},
		},
		Manifest: manifest.Manifest{
			Prompts:      []manifest.Prompt{{ID: "p1", ContentHash: "aaa"}},
			Dependencies: []manifest.Dependency{{ID: "left-pad", Ecosystem: "npm", Version: "1.3.1", Hash: "ddd"}},
		},
	}
	r := diff.Compare(base, head)
	if r.NeedsApproval {
		t.Fatalf("version bump (no AI change, no new dep) should not need approval: %+v", r)
	}
}
