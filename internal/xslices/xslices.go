// Package xslices holds tiny generic helpers not yet in the standard library.
package xslices

import (
	"cmp"
	"slices"
)

// UniqueSorted returns a sorted copy with consecutive duplicates removed.
func UniqueSorted[T cmp.Ordered](in []T) []T {
	if len(in) == 0 {
		return nil
	}
	out := slices.Clone(in)
	slices.Sort(out)
	return slices.Compact(out)
}

// Keys returns map keys in indeterminate order (caller may sort).
func Keys[M ~map[K]V, K comparable, V any](m M) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
