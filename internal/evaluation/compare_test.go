package evaluation_test

import (
	"testing"

	"github.com/ankittk/airlock/internal/evaluation"
	"github.com/ankittk/airlock/internal/policy"
	"github.com/ankittk/airlock/internal/stats"
)

func TestCompareResults(t *testing.T) {
	base := &evaluation.RunResult{
		Report: policy.Report{
			Metrics: []policy.MetricEvidence{{
				Name: "task_success",
				CI:   stats.Interval{Estimate: 0.95, Low: 0.90, High: 0.99},
			}},
		},
	}
	cand := &evaluation.RunResult{
		Report: policy.Report{
			Metrics: []policy.MetricEvidence{{
				Name:    "task_success",
				CI:      stats.Interval{Estimate: 0.90, Low: 0.85, High: 0.95},
				Delta:   &stats.Interval{Estimate: -0.05, Low: -0.10, High: 0.0},
				Verdict: policy.Fail,
			}},
		},
	}
	rows := evaluation.CompareResults(base, cand, policy.Default())
	if len(rows) != 1 || rows[0].Metric != "task_success" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}
