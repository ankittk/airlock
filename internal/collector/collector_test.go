package collector

import (
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	in := "email me@corp.com key sk-abcdefghijklmnopqrstuvwxyz cards 4111111111111111"
	out := Redact(in, Options{})
	if out == in {
		t.Fatal("expected redaction")
	}
	if strings.Contains(out, "me@corp.com") || strings.Contains(out, "sk-abcdefgh") {
		t.Fatalf("leaked: %s", out)
	}
}

func TestToCases(t *testing.T) {
	ok := true
	spans := []Span{{Input: "hello me@x.com", Success: &ok, Agent: "a"}}
	cases := ToCases(spans, Options{}, 10)
	if len(cases) != 1 {
		t.Fatal(len(cases))
	}
	if strings.Contains(cases[0].Input.Messages[0].Content, "@") {
		t.Fatal("email not redacted in case")
	}
}
