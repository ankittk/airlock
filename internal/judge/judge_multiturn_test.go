package judge_test

import (
	"context"
	"testing"

	"github.com/ankittk/airlock/internal/evalcase"
	"github.com/ankittk/airlock/internal/judge"
	"github.com/ankittk/airlock/internal/providers"
)

func TestMultiTurnJudge(t *testing.T) {
	reg := &judge.Registry{
		ByID: map[string]judge.Spec{
			"multi": {
				ID: "multi", Provider: "mock", PassContains: "PASS",
				Turns: []evalcase.Message{
					{Role: "system", Content: "Rubric: {{rubric}}"},
					{Role: "user", Content: "Input: {{input}}\nOutput: {{output}}"},
				},
				Rubric: "be helpful",
			},
		},
	}
	c := evalcase.Case{Input: evalcase.Input{Messages: []evalcase.Message{{Role: "user", Content: "hi"}}}}
	ok, err := reg.Score(context.Background(), "multi", c, providers.Response{Text: "PASS"})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected mock multi-turn judge pass")
	}
}
