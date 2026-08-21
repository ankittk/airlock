package evaluation

import (
	"context"
	"testing"

	"github.com/ankittk/airlock/internal/evalcase"
	"github.com/ankittk/airlock/internal/policy"
)

func TestBudgetStop(t *testing.T) {
	cases := []evalcase.Case{{
		ID: "a",
		Input: evalcase.Input{
			Provider: "mock",
			Messages: []evalcase.Message{{Role: "user", Content: "ping"}},
		},
		Expect: evalcase.Expect{JSONValid: true},
	}}
	minRate := 0.5
	pol := policy.Policy{
		Gates:   map[string]policy.GateSpec{"json_valid": {Min: &minRate, Confidence: 0.95}},
		Budgets: policy.Budgets{MaxCostPerPR: 0.00015}, // ~1 mock call
	}
	suite := evalcase.DefaultSuite()
	suite.K = 10
	suite.MaxSamplesPerCase = 10
	suite.MinSamples = 1
	res, err := Run(context.Background(), cases, Config{Suite: suite, Policy: pol})
	if err != nil {
		t.Fatal(err)
	}
	if !res.BudgetStopped && res.TotalCostUSD < 0.00015 {
		t.Fatalf("expected budget stop or cost near ceiling: cost=%v stopped=%v samples=%d", res.TotalCostUSD, res.BudgetStopped, len(res.Samples))
	}
}

func TestBaselinePairing(t *testing.T) {
	base := &RunResult{Samples: []SampleResult{
		{CaseID: "a", SampleIdx: 0, Scores: map[string]bool{"task_success": true}},
		{CaseID: "a", SampleIdx: 1, Scores: map[string]bool{"task_success": true}},
	}}
	m := BaselineFromResult(base)
	if len(m["task_success"]) != 2 {
		t.Fatalf("%v", m)
	}
}
