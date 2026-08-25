package evaluation

import (
	"fmt"

	"github.com/ankittk/airlock/internal/collector"
	"github.com/ankittk/airlock/internal/evalcase"
)

// PromoteFromIngest appends OTel ingest spans as eval cases (optionally failures only).
func PromoteFromIngest(ingestPath, outPath string, opt evalcase.PromoteOptions) (int, error) {
	spans, err := collector.LoadJSONL(ingestPath)
	if err != nil {
		return 0, err
	}
	if opt.FailuresOnly {
		var failed []collector.Span
		for _, sp := range spans {
			if sp.Success != nil && !*sp.Success {
				failed = append(failed, sp)
			}
		}
		spans = failed
	}
	tag := opt.Tag
	if tag == "" {
		tag = "promoted"
	}
	cases := collector.ToCases(spans, collector.Options{Mode: "pii"}, opt.Limit)
	for i := range cases {
		cases[i].Tags = append(cases[i].Tags, tag)
	}
	return evalcase.AppendCases(outPath, cases)
}

// PromoteFromResults appends failed eval samples as new cases for replay.
func PromoteFromResults(resultsPath, outPath string, opt evalcase.PromoteOptions) (int, error) {
	res, err := LoadResult(resultsPath)
	if err != nil {
		return 0, err
	}
	tag := opt.Tag
	if tag == "" {
		tag = "promoted"
	}
	var cases []evalcase.Case
	n := 0
	for _, s := range res.Samples {
		if opt.FailuresOnly {
			failed := s.Err != ""
			if !failed {
				for _, pass := range s.Scores {
					if !pass {
						failed = true
						break
					}
				}
			}
			if !failed {
				continue
			}
		}
		if opt.Limit > 0 && n >= opt.Limit {
			break
		}
		id := fmt.Sprintf("promoted-%s-%d", s.CaseID, s.SampleIdx)
		fail := true
		cases = append(cases, evalcase.Case{
			ID:    id,
			Agent: "default",
			Input: evalcase.Input{
				Provider: "mock",
				Messages: []evalcase.Message{{Role: "user", Content: s.Response.Text}},
			},
			Tags: []string{tag, "from-results"},
			Expect: evalcase.Expect{
				TaskSuccess: &fail,
			},
			Meta: map[string]any{
				"source_case": s.CaseID,
				"sample_idx":  s.SampleIdx,
			},
		})
		n++
	}
	return evalcase.AppendCases(outPath, cases)
}
