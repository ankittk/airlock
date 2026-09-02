package diff

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ankittk/airlock/internal/manifest"
	"github.com/ankittk/airlock/internal/xslices"
)

// Change is one artifact that added, removed, or changed hash.
type Change struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Status  string `json:"status"` // added|removed|changed
	OldHash string `json:"old_hash,omitempty"`
	NewHash string `json:"new_hash,omitempty"`
}

// Result is a static blast-radius diff between two snapshots.
type Result struct {
	BaseID          string   `json:"base_id"`
	HeadID          string   `json:"head_id"`
	Changes         []Change `json:"changes"`
	AffectedAgents  []string `json:"affected_agents"`
	AffectedEvals   []string `json:"affected_evals"`
	NeedsApproval   bool     `json:"needs_approval"`
	ApprovalReasons []string `json:"approval_reasons,omitempty"`
}

// Compare computes artifact set-diff and walks the head graph for blast radius.
func Compare(base, head *manifest.Snapshot) *Result {
	r := &Result{
		BaseID: base.ID,
		HeadID: head.ID,
	}
	baseMap := index(base.Artifacts)
	headMap := index(head.Artifacts)

	keys := map[string]struct{}{}
	for k := range baseMap {
		keys[k] = struct{}{}
	}
	for k := range headMap {
		keys[k] = struct{}{}
	}
	keyList := make([]string, 0, len(keys))
	for k := range keys {
		keyList = append(keyList, k)
	}
	slices.Sort(keyList)

	changedKeys := map[string]bool{}
	for _, k := range keyList {
		b, bok := baseMap[k]
		h, hok := headMap[k]
		switch {
		case !bok:
			r.Changes = append(r.Changes, Change{Kind: h.Kind, ID: h.ID, Status: "added", NewHash: h.Hash})
			changedKeys[k] = true
		case !hok:
			r.Changes = append(r.Changes, Change{Kind: b.Kind, ID: b.ID, Status: "removed", OldHash: b.Hash})
			changedKeys[k] = true
		case b.Hash != h.Hash:
			r.Changes = append(r.Changes, Change{Kind: h.Kind, ID: h.ID, Status: "changed", OldHash: b.Hash, NewHash: h.Hash})
			changedKeys[k] = true
		}
	}

	r.AffectedAgents, r.AffectedEvals = blastRadius(&head.Manifest, changedKeys)
	// also consider base graph for removed deps
	if len(r.AffectedAgents) == 0 {
		r.AffectedAgents, r.AffectedEvals = blastRadius(&base.Manifest, changedKeys)
	} else {
		a2, e2 := blastRadius(&base.Manifest, changedKeys)
		r.AffectedAgents = xslices.UniqueSorted(append(r.AffectedAgents, a2...))
		r.AffectedEvals = xslices.UniqueSorted(append(r.AffectedEvals, e2...))
	}
	r.NeedsApproval, r.ApprovalReasons = permissionExpansion(base, head, r.Changes)
	if depNeeds, depReasons := dependencyExpansion(r.Changes); depNeeds {
		r.NeedsApproval = true
		r.ApprovalReasons = xslices.UniqueSorted(append(r.ApprovalReasons, depReasons...))
	}
	return r
}

// dependencyExpansion flags a new (non-AI) package dependency landing in the
// same PR as an AI artifact change — agent-driven supply chain: the diff looks
// like "prompt tweak" but quietly widens what the app depends on. A dep-only
// PR with no AI artifact change is left alone; that is SCA's job, not ours.
func dependencyExpansion(changes []Change) (bool, []string) {
	var aiChanged bool
	var added []string
	for _, c := range changes {
		if c.Kind == "dependency" {
			if c.Status == "added" {
				added = append(added, c.ID)
			}
			continue
		}
		if c.Status == "added" || c.Status == "changed" {
			aiChanged = true
		}
	}
	if !aiChanged || len(added) == 0 {
		return false, nil
	}
	reasons := make([]string, 0, len(added))
	for _, id := range added {
		reasons = append(reasons, "new dependency: "+id)
	}
	return true, reasons
}

func permissionExpansion(base, head *manifest.Snapshot, changes []Change) (bool, []string) {
	baseTools := map[string]manifest.Tool{}
	for _, t := range base.Manifest.Tools {
		baseTools[t.ID] = t
	}
	baseMCP := map[string]manifest.MCPServer{}
	for _, m := range base.Manifest.MCPServers {
		baseMCP[m.ID] = m
	}
	baseSkills := map[string]manifest.Skill{}
	for _, s := range base.Manifest.Skills {
		baseSkills[s.ID] = s
	}
	var reasons []string
	for _, c := range changes {
		if c.Status != "added" && c.Status != "changed" {
			continue
		}
		switch c.Kind {
		case "tool":
			for _, t := range head.Manifest.Tools {
				if t.ID != c.ID {
					continue
				}
				old, had := baseTools[t.ID]
				se := strings.ToLower(t.SideEffect)
				if se == "write" || looksWriteTool(t.Name) {
					if !had || strings.ToLower(old.SideEffect) != "write" {
						reasons = append(reasons, "new/expanded write tool: "+t.ID)
					}
				}
				if c.Status == "added" && (se == "write" || se == "unknown" || looksWriteTool(t.Name)) {
					if !slices.Contains(reasons, "new/expanded write tool: "+t.ID) {
						reasons = append(reasons, "added tool (needs review): "+t.ID)
					}
				}
			}
		case "skill":
			for _, s := range head.Manifest.Skills {
				if s.ID != c.ID {
					continue
				}
				if _, had := baseSkills[s.ID]; !had {
					reasons = append(reasons, "added skill (needs review): "+s.ID)
					continue
				}
				if c.Status == "changed" {
					reasons = append(reasons, "skill content changed: "+s.ID)
				}
			}
		case "mcp":
			for _, m := range head.Manifest.MCPServers {
				if m.ID != c.ID {
					continue
				}
				old, had := baseMCP[m.ID]
				if !had {
					reasons = append(reasons, "added MCP server: "+m.ID)
					continue
				}
				if len(m.Permissions) > len(old.Permissions) {
					reasons = append(reasons, "MCP permissions expanded: "+m.ID)
				}
				for _, p := range m.Permissions {
					if !slices.Contains(old.Permissions, p) {
						reasons = append(reasons, "MCP new permission "+p+" on "+m.ID)
						break
					}
				}
				// Live tools/list diff: Permissions is only ever hand-maintained
				// via apm.lock.yaml, so a server whose actual tool list grows
				// (discovered live, see mcp_fetch.go) would otherwise pass
				// through as a bare "changed" with no approval signal at all.
				for _, name := range m.ToolNames {
					if slices.Contains(old.ToolNames, name) {
						continue
					}
					if looksWriteTool(name) {
						reasons = append(reasons, "MCP new write-looking tool "+name+" on "+m.ID)
					} else {
						reasons = append(reasons, "MCP new tool (needs review): "+name+" on "+m.ID)
					}
				}
			}
		}
	}
	return len(reasons) > 0, xslices.UniqueSorted(reasons)
}

func looksWriteTool(name string) bool {
	n := strings.ToLower(name)
	// "post" dropped: too generic, false-positives on read-only names like
	// "post_processing_helper" / "status_update_checker" style tools.
	for _, k := range []string{"delete", "send", "write", "create", "payment", "email", "update"} {
		if strings.Contains(n, k) {
			return true
		}
	}
	return false
}

func index(arts []manifest.ArtifactRef) map[string]manifest.ArtifactRef {
	m := make(map[string]manifest.ArtifactRef, len(arts))
	for _, a := range arts {
		m[a.Kind+":"+a.ID] = a
	}
	return m
}

func blastRadius(m *manifest.Manifest, changed map[string]bool) (agents, evals []string) {
	agentSet := map[string]bool{}
	evalSet := map[string]bool{}

	for k := range changed {
		if strings.HasPrefix(k, "agent:") {
			agentSet[strings.TrimPrefix(k, "agent:")] = true
		}
		if strings.HasPrefix(k, "eval:") {
			evalSet[strings.TrimPrefix(k, "eval:")] = true
		}
		if strings.HasPrefix(k, "env:") {
			for _, a := range m.Agents {
				agentSet[a.ID] = true
			}
		}
	}

	for _, e := range m.Graph {
		if !strings.HasPrefix(e.From, "agent:") {
			continue
		}
		aid := strings.TrimPrefix(e.From, "agent:")
		if changed[e.To] || changed[e.From] {
			agentSet[aid] = true
		}
	}

	for _, a := range m.Agents {
		check := func(kind string, ids []string) {
			for _, id := range ids {
				if changed[kind+":"+id] {
					agentSet[a.ID] = true
				}
			}
		}
		check("model", a.Models)
		check("prompt", a.Prompts)
		check("tool", a.Tools)
		check("skill", a.Skills)
		check("mcp", a.MCP)
		check("eval", a.Evals)
		for _, id := range a.Evals {
			if agentSet[a.ID] {
				evalSet[id] = true
			}
		}
	}

	for id := range agentSet {
		agents = append(agents, id)
	}
	for id := range evalSet {
		evals = append(evals, id)
	}
	slices.Sort(agents)
	slices.Sort(evals)
	return agents, evals
}

// HasKind reports whether any change has the given artifact kind (e.g. "mcp").
func HasKind(r *Result, kind string) bool {
	if r == nil {
		return false
	}
	for _, c := range r.Changes {
		if c.Kind == kind {
			return true
		}
	}
	return false
}

// FormatText renders a human-readable report.
func FormatText(r *Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "diff %s → %s\n", r.BaseID, r.HeadID)
	if len(r.Changes) == 0 {
		b.WriteString("No AI artifact changes.\n")
		return b.String()
	}
	b.WriteString("Changed AI artifacts:\n")
	for _, c := range r.Changes {
		switch c.Status {
		case "added":
			fmt.Fprintf(&b, "  + %s:%s\n", c.Kind, c.ID)
		case "removed":
			fmt.Fprintf(&b, "  - %s:%s\n", c.Kind, c.ID)
		case "changed":
			fmt.Fprintf(&b, "  ~ %s:%s (%s → %s)\n", c.Kind, c.ID, short(c.OldHash), short(c.NewHash))
		}
	}
	if len(r.AffectedAgents) > 0 {
		fmt.Fprintf(&b, "Blast radius — agents: %s\n", strings.Join(r.AffectedAgents, ", "))
	} else {
		b.WriteString("Blast radius — agents: (none linked)\n")
	}
	if len(r.AffectedEvals) > 0 {
		fmt.Fprintf(&b, "Blast radius — evals: %s\n", strings.Join(r.AffectedEvals, ", "))
	}
	if r.NeedsApproval {
		fmt.Fprintf(&b, "NEEDS_APPROVAL: %s\n", strings.Join(r.ApprovalReasons, "; "))
	}
	return b.String()
}

// FormatComment is the GitHub PR comment body.
func FormatComment(r *Result) string {
	if len(r.Changes) == 0 {
		return "### Airlock\nNo AI artifact changes in this PR."
	}
	var b strings.Builder
	b.WriteString("### Airlock\n")
	b.WriteString("This PR changes AI artifacts:\n")
	for _, c := range r.Changes {
		fmt.Fprintf(&b, "- `%s` **%s:%s**\n", c.Status, c.Kind, c.ID)
	}
	if len(r.AffectedAgents) > 0 {
		fmt.Fprintf(&b, "\nBlast radius: agents **%s**\n", strings.Join(r.AffectedAgents, ", "))
	}
	if len(r.AffectedEvals) > 0 {
		fmt.Fprintf(&b, "\nBlast radius: evals **%s**\n", strings.Join(r.AffectedEvals, ", "))
	}
	if r.NeedsApproval {
		fmt.Fprintf(&b, "\n**NEEDS_APPROVAL:** %s\n", strings.Join(r.ApprovalReasons, "; "))
	}
	return b.String()
}

func short(h string) string {
	if len(h) > 8 {
		return h[:8]
	}
	return h
}

// HasChanges reports whether any AI artifacts differ.
func HasChanges(r *Result) bool {
	return len(r.Changes) > 0
}
