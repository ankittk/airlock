package policy

import (
	"strings"
	"testing"

	"github.com/ankittk/airlock/internal/stats"
)

func TestEvaluateFailWhenCIExcludesMin(t *testing.T) {
	minRate := 0.99
	p := Policy{Gates: map[string]GateSpec{"tool_success": {Min: &minRate, Confidence: 0.95}}}
	r := Evaluate(p, []MetricRates{{Name: "tool_success", Successes: 0, N: 50}})
	if r.Overall != Fail {
		t.Fatalf("want FAIL got %s\n%s", r.Overall, FormatTable(r))
	}
	_ = stats.WilsonCI(0, 50, 0.95)
}

func TestEvaluateInconclusiveSmallN(t *testing.T) {
	minRate := 0.99
	p := Policy{Gates: map[string]GateSpec{"json_valid": {Min: &minRate, Confidence: 0.95}}}
	r := Evaluate(p, []MetricRates{{Name: "json_valid", Successes: 5, N: 5}})
	if r.Overall != Inconclusive {
		t.Fatalf("want INCONCLUSIVE got %s", r.Overall)
	}
}

// A PR comment reader must still see the real (eval-gate) failure reason, not
// have it silently replaced by the unrelated approval note.
func TestWithNeedsApprovalDoesNotClobberFailSummary(t *testing.T) {
	r := Report{Overall: Fail, Summary: "CI high 0.94 < min 0.99"}
	r = WithNeedsApproval(r, true, []string{"added skill (needs review): refund-policy"})
	if r.Overall != Fail {
		t.Fatalf("want FAIL to still dominate, got %s", r.Overall)
	}
	if !strings.Contains(r.Summary, "CI high 0.94 < min 0.99") {
		t.Fatalf("original fail reason lost, got summary: %q", r.Summary)
	}
	if !strings.Contains(r.Summary, "needs_approval") {
		t.Fatalf("approval reason missing from summary: %q", r.Summary)
	}
}

func TestWithNeedsApprovalSetsSummaryWhenNotFail(t *testing.T) {
	r := Report{Overall: Pass, Summary: "no gated metrics"}
	r = WithNeedsApproval(r, true, []string{"added MCP server: fs"})
	if r.Overall != NeedsApproval {
		t.Fatalf("want NEEDS_APPROVAL, got %s", r.Overall)
	}
	if r.Summary != "needs_approval: added MCP server: fs" {
		t.Fatalf("got summary: %q", r.Summary)
	}
}
