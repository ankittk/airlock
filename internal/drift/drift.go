package drift

import (
	"fmt"

	"github.com/xdlc-labs/airlock/internal/collector"
	"github.com/xdlc-labs/airlock/internal/policy"
	"github.com/xdlc-labs/airlock/internal/stats"
)

type Report struct {
	BaselineOK int                `json:"baseline_ok"`
	BaselineN  int                `json:"baseline_n"`
	LiveOK     int                `json:"live_ok"`
	LiveN      int                `json:"live_n"`
	BaselineCI stats.Interval     `json:"baseline_ci"`
	LiveCI     stats.Interval     `json:"live_ci"`
	Delta      stats.Interval     `json:"delta"`
	Verdict    policy.VerdictKind `json:"verdict"`
	Reason     string             `json:"reason"`
}

// CompareSuccessRates checks live ingest success vs baseline rates with Wilson + bootstrap paired when possible.
func CompareSuccessRates(baseOK, baseN, liveOK, liveN int, maxRegressionPP float64, level float64) Report {
	if level <= 0 {
		level = 0.95
	}
	r := Report{
		BaselineOK: baseOK,
		BaselineN:  baseN,
		LiveOK:     liveOK,
		LiveN:      liveN,
		BaselineCI: stats.WilsonCI(baseOK, baseN, level),
		LiveCI:     stats.WilsonCI(liveOK, liveN, level),
	}
	n := min(baseN, liveN)
	if n > 0 && maxRegressionPP > 0 {
		base := expand(baseOK, baseN, n)
		live := expand(liveOK, liveN, n)
		r.Delta = stats.BootstrapPairedDeltaCI(base, live, level, 999, 7)
		switch {
		case stats.FailsMaxRegression(r.Delta, maxRegressionPP):
			r.Verdict = policy.Fail
			r.Reason = fmt.Sprintf("live success regressed (delta CI [%.3f, %.3f])", r.Delta.Low, r.Delta.High)
		case stats.PassesMaxRegression(r.Delta, maxRegressionPP):
			r.Verdict = policy.Pass
			r.Reason = "no significant success regression"
		default:
			r.Verdict = policy.Inconclusive
			r.Reason = "drift inconclusive"
		}
	} else if liveN == 0 || baseN == 0 {
		r.Verdict = policy.Inconclusive
		r.Reason = "insufficient samples"
	} else {
		r.Verdict = policy.Pass
		r.Reason = "no regression gate configured"
	}
	return r
}

func FromSpans(base, live []collector.Span, maxRegressionPP float64) Report {
	bok, bn := collector.SuccessRate(base)
	lok, ln := collector.SuccessRate(live)
	return CompareSuccessRates(bok, bn, lok, ln, maxRegressionPP, 0.95)
}

func expand(ok, n, want int) []bool {
	out := make([]bool, 0, want)
	for i := range want {
		out = append(out, i < ok*want/max(n, 1))
	}
	return out
}

func FormatText(r Report) string {
	return fmt.Sprintf("Drift verdict: %s\n  baseline %d/%d (%.1f%%) CI[%.1f, %.1f]\n  live     %d/%d (%.1f%%) CI[%.1f, %.1f]\n  delta    %.3f [%0.3f, %.3f]\n  %s\n",
		r.Verdict,
		r.BaselineOK, r.BaselineN, r.BaselineCI.Estimate*100, r.BaselineCI.Low*100, r.BaselineCI.High*100,
		r.LiveOK, r.LiveN, r.LiveCI.Estimate*100, r.LiveCI.Low*100, r.LiveCI.High*100,
		r.Delta.Estimate, r.Delta.Low, r.Delta.High,
		r.Reason,
	)
}
