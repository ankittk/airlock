package stats

import "testing"

func TestWilsonDecisive(t *testing.T) {
	ci := WilsonCI(1000, 1000, 0.95)
	if !PassesMin(ci, 0.99) {
		t.Fatalf("expected pass: %+v", ci)
	}
	ci2 := WilsonCI(0, 100, 0.95)
	if !FailsMin(ci2, 0.99) {
		t.Fatalf("expected fail: %+v", ci2)
	}
	ci3 := WilsonCI(1, 2, 0.95)
	if PassesMin(ci3, 0.99) || FailsMin(ci3, 0.99) {
		t.Logf("inconclusive ok: %+v", ci3)
	}
}
