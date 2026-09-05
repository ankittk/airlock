package evalcase_test

import (
	"testing"

	"github.com/xdlc-labs/airlock/internal/evalcase"
)

func TestSelectSuitesSpecificity(t *testing.T) {
	b := evalcase.BindingsFile{
		Bindings: []evalcase.Binding{
			{When: evalcase.BindingWhen{Kind: "prompt"}, Suite: "suite.yml"},
			{When: evalcase.BindingWhen{Kind: "prompt", ID: "system-prompt"}, Suite: "suites/prompt.yml"},
			{When: evalcase.BindingWhen{Kind: "mcp"}, Suite: "suite.adversarial.yml"},
		},
	}
	changes := []evalcase.ArtifactChange{
		{Kind: "prompt", ID: "system-prompt", Status: "changed"},
	}
	got := evalcase.SelectSuites(changes, b)
	if len(got) != 1 || got[0] != "suites/prompt.yml" {
		t.Fatalf("want specific prompt suite, got %v", got)
	}
}

func TestSelectSuitesUnionSameSpecificity(t *testing.T) {
	b := evalcase.BindingsFile{
		Bindings: []evalcase.Binding{
			{When: evalcase.BindingWhen{Kind: "mcp"}, Suite: "suite.adversarial.yml"},
			{When: evalcase.BindingWhen{Kind: "skill"}, Suite: "suite.adversarial.yml"},
		},
	}
	changes := []evalcase.ArtifactChange{
		{Kind: "mcp", ID: "local-fs", Status: "changed"},
		{Kind: "skill", ID: "x", Status: "changed"},
	}
	got := evalcase.SelectSuites(changes, b)
	if len(got) != 1 || got[0] != "suite.adversarial.yml" {
		t.Fatalf("want deduped adversarial suite, got %v", got)
	}
}
