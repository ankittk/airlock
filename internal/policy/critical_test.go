package policy_test

import (
	"testing"

	"github.com/xdlc-labs/airlock/internal/policy"
)

func TestMaxNewCritical(t *testing.T) {
	zero := 0
	p := policy.Policy{Gates: map[string]policy.GateSpec{
		"adversarial_critical": {MaxNewCritical: &zero, Confidence: 0.95},
	}}
	base := []bool{true, true, false}  // 1 critical
	cand := []bool{true, false, false} // 2 criticals → +1 new
	r := policy.Evaluate(p, []policy.MetricRates{{
		Name: "adversarial_critical", Successes: 1, N: 3, BasePass: base, CandPass: cand,
	}})
	if r.Overall != policy.Fail {
		t.Fatalf("want FAIL got %s %s", r.Overall, r.Summary)
	}
}
