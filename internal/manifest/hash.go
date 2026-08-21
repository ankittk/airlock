package manifest

import (
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/zeebo/blake3"
)

// HashBytes returns a BLAKE3 hex digest of b.
func HashBytes(b []byte) string {
	sum := blake3.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// HashFile returns a BLAKE3 hex digest of the file at path.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := blake3.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	sum := h.Sum(nil)
	return hex.EncodeToString(sum), nil
}

// HashString is HashBytes for a string.
func HashString(s string) string {
	return HashBytes([]byte(s))
}

// HashTree hashes a sorted list of relative paths under root (file contents).
func HashTree(root string, rels []string) (string, error) {
	type pair struct{ path, hash string }
	pairs := make([]pair, 0, len(rels))
	for _, rel := range rels {
		p := filepath.Join(root, rel)
		h, err := HashFile(p)
		if err != nil {
			return "", err
		}
		pairs = append(pairs, pair{path: filepath.ToSlash(rel), hash: h})
	}
	slices.SortFunc(pairs, func(a, b pair) int { return strings.Compare(a.path, b.path) })
	var b strings.Builder
	for _, p := range pairs {
		b.WriteString(p.path)
		b.WriteByte(':')
		b.WriteString(p.hash)
		b.WriteByte('\n')
	}
	return HashString(b.String()), nil
}
