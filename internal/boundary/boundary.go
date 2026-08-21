package boundary

import (
	"regexp"
	"strings"
)

// Patterns mirror collector redaction — used as release blockers when policy enables data_boundary.
var (
	reEmail = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	reKey   = regexp.MustCompile(`(?i)(sk-[a-zA-Z0-9]{16,}|api[_-]?key[=:]\s*\S+)`)
	rePhone = regexp.MustCompile(`\b(?:\+?1[-.\s]?)?(?:\(?\d{3}\)?[-.\s]?)\d{3}[-.\s]?\d{4}\b`)
	reSSN   = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	reCC    = regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)
)

// Finding is one sensitive hit in model I/O.
type Finding struct {
	Kind    string // email|key|phone|ssn|card
	Snippet string
}

// Scan returns sensitive-class hits in s (empty if clean).
func Scan(s string) []Finding {
	if s == "" {
		return nil
	}
	var out []Finding
	add := func(kind string, re *regexp.Regexp) {
		if m := re.FindString(s); m != "" {
			out = append(out, Finding{Kind: kind, Snippet: truncate(m, 24)})
		}
	}
	add("email", reEmail)
	add("key", reKey)
	add("phone", rePhone)
	add("ssn", reSSN)
	if m := reCC.FindString(s); m != "" {
		digits := 0
		for _, r := range m {
			if r >= '0' && r <= '9' {
				digits++
			}
		}
		if digits >= 13 {
			out = append(out, Finding{Kind: "card", Snippet: truncate(m, 24)})
		}
	}
	return out
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
