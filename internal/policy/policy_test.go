package policy

import (
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
