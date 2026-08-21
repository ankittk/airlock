package drift

import "testing"

func TestCompareRegression(t *testing.T) {
	r := CompareSuccessRates(90, 100, 10, 100, 1.0, 0.95)
	if r.Verdict != "FAIL" {
		t.Fatalf("want FAIL got %s: %s", r.Verdict, r.Reason)
	}
}

func TestCompareOK(t *testing.T) {
	r := CompareSuccessRates(50, 50, 50, 50, 1.0, 0.95)
	if r.Verdict == "FAIL" {
		t.Fatalf("unexpected FAIL: %+v", r)
	}
}
