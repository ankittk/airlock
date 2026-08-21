package xslices_test

import (
	"slices"
	"testing"

	"github.com/ankittk/airlock/internal/xslices"
)

func TestUniqueSorted(t *testing.T) {
	got := xslices.UniqueSorted([]string{"b", "a", "b", "c", "a"})
	want := []string{"a", "b", "c"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if xslices.UniqueSorted[string](nil) != nil {
		t.Fatal("nil in → nil out")
	}
}
