// Package cliargs is a minimal argv flag stripper for the airlock CLI.
package cliargs

import (
	"path/filepath"
	"slices"
	"strings"
)

// List is a mutable argv slice that supports --flag / --flag=val stripping.
type List []string

// Path extracts --path DIR or --path=DIR (default ".") and returns abs path + remainder.
func (a List) Path() (string, List, error) {
	root := "."
	out := slices.Clone(a)
	for i := 0; i < len(out); i++ {
		arg := out[i]
		switch {
		case arg == "--path" && i+1 < len(out):
			root = out[i+1]
			out = append(out[:i], out[i+2:]...)
			i--
		case strings.HasPrefix(arg, "--path="):
			root = strings.TrimPrefix(arg, "--path=")
			out = append(out[:i], out[i+1:]...)
			i--
		}
	}
	abs, err := filepath.Abs(root)
	return abs, out, err
}

// Bool removes all occurrences of name and reports whether any were present.
func (a *List) Bool(name string) bool {
	out := slices.Clone(*a)
	found := false
	for i := 0; i < len(out); i++ {
		if out[i] == name {
			found = true
			out = append(out[:i], out[i+1:]...)
			i--
		}
	}
	*a = out
	return found
}

// Val removes --name VALUE or --name=VALUE and returns VALUE (first wins).
func (a *List) Val(name string) string {
	out := slices.Clone(*a)
	eq := name + "="
	for i := 0; i < len(out); i++ {
		arg := out[i]
		if arg == name && i+1 < len(out) {
			val := out[i+1]
			*a = append(out[:i], out[i+2:]...)
			return val
		}
		if rest, ok := strings.CutPrefix(arg, eq); ok {
			*a = append(out[:i], out[i+1:]...)
			return rest
		}
	}
	*a = out
	return ""
}
