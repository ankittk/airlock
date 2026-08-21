package manifest

import "testing"

func TestHashBytesStable(t *testing.T) {
	a := HashBytes([]byte("hello"))
	b := HashBytes([]byte("hello"))
	if a != b {
		t.Fatalf("unstable hash: %s vs %s", a, b)
	}
	if len(a) != 64 {
		t.Fatalf("want 64 hex chars, got %d", len(a))
	}
	if HashBytes([]byte("hello")) == HashBytes([]byte("world")) {
		t.Fatal("different inputs same hash")
	}
}

func TestValidateRequiresHashes(t *testing.T) {
	m := &Manifest{
		Version: 1,
		Models:  []Model{{ID: "m1", Model: "claude", ContentHash: ""}},
	}
	if err := Validate(m); err == nil {
		t.Fatal("expected error for empty content_hash")
	}
}
