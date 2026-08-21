package evaluation

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ankittk/airlock/internal/evalcase"
	"github.com/ankittk/airlock/internal/policy"
)

func TestMockJSONAndTool(t *testing.T) {
	cases := []evalcase.Case{
		{
			ID:    "json-1",
			Agent: "support-bot",
			Input: evalcase.Input{
				Provider: "mock",
				Messages: []evalcase.Message{{Role: "user", Content: "hello"}},
			},
			Expect: evalcase.Expect{JSONValid: true},
		},
		{
			ID:    "tool-1",
			Agent: "support-bot",
			Input: evalcase.Input{
				Provider: "mock",
				Messages: []evalcase.Message{{Role: "user", Content: "lookup order ORD-1"}},
				Tools: []evalcase.ToolDef{{
					Name:       "search_orders",
					Parameters: json.RawMessage(`{"type":"object","required":["order_id"],"properties":{"order_id":{"type":"string"}}}`),
				}},
			},
			Expect: evalcase.Expect{ToolArgsSchema: "search_orders", ToolName: "search_orders"},
		},
	}
	p := policy.Default()
	// relax mins so small N can PASS
	minJSON := 0.5
	minTool := 0.5
	p.Gates["json_valid"] = policy.GateSpec{Min: &minJSON, Confidence: 0.95}
	p.Gates["tool_success"] = policy.GateSpec{Min: &minTool, Confidence: 0.95}

	suite := evalcase.DefaultSuite()
	suite.K = 3
	suite.MinSamples = 2
	suite.MaxSamplesPerCase = 3
	suite.Mode = "live"

	res, err := Run(context.Background(), cases, Config{Suite: suite, Policy: p})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Samples) == 0 {
		t.Fatal("no samples")
	}
	for _, s := range res.Samples {
		if s.Err != "" {
			t.Fatalf("sample err: %s", s.Err)
		}
	}
	if res.Report.Overall == policy.Fail {
		t.Fatalf("unexpected FAIL: %s", policy.FormatTable(res.Report))
	}
}
