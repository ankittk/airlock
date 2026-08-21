package boundary_test

import (
	"testing"

	"github.com/ankittk/airlock/internal/boundary"
)

func TestScan(t *testing.T) {
	if got := boundary.Scan("hello"); len(got) != 0 {
		t.Fatalf("%v", got)
	}
	hits := boundary.Scan("email me@corp.com and sk-abcdefghijklmnopqrstuvwxyz")
	if len(hits) < 2 {
		t.Fatalf("%v", hits)
	}
}
