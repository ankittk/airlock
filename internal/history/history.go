package history

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type Entry struct {
	ID      string    `json:"id"`
	Kind    string    `json:"kind"` // snapshot|result
	Path    string    `json:"path"`
	ModTime time.Time `json:"mod_time"`
	Verdict string    `json:"verdict,omitempty"`
	Summary string    `json:"summary,omitempty"`
}

func List(airlockDir string) ([]Entry, error) {
	var out []Entry
	snapDir := filepath.Join(airlockDir, "snapshots")
	resDir := filepath.Join(airlockDir, "results")
	if err := filepath.Walk(snapDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		out = append(out, Entry{
			ID:      strings.TrimSuffix(info.Name(), ".json"),
			Kind:    "snapshot",
			Path:    path,
			ModTime: info.ModTime(),
		})
		return nil
	}); err != nil {
		return nil, err
	}
	if err := filepath.Walk(resDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		e := Entry{
			ID:      strings.TrimSuffix(info.Name(), ".json"),
			Kind:    "result",
			Path:    path,
			ModTime: info.ModTime(),
		}
		data, err := os.ReadFile(path)
		if err == nil {
			var raw map[string]any
			if json.Unmarshal(data, &raw) == nil {
				if rep, ok := raw["report"].(map[string]any); ok {
					if v, ok := rep["overall"].(string); ok {
						e.Verdict = v
					}
					if s, ok := rep["summary"].(string); ok {
						e.Summary = s
					}
				}
			}
		}
		out = append(out, e)
		return nil
	}); err != nil {
		return nil, err
	}
	slices.SortFunc(out, func(a, b Entry) int {
		return b.ModTime.Compare(a.ModTime)
	})
	return out, nil
}

func RenderHTML(entries []Entry) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>Airlock history</title>
<style>
body{font-family:ui-monospace,monospace;margin:2rem;background:#0f1115;color:#e8eaed}
table{border-collapse:collapse;width:100%}
th,td{border-bottom:1px solid #333;padding:.5rem;text-align:left}
a{color:#8ab4f8} .PASS{color:#81c995} .FAIL{color:#f28b82} .INCONCLUSIVE{color:#fdd663} .NEEDS_APPROVAL{color:#c58af9}
</style></head><body>
<h1>Airlock history</h1>
<p>Read-only local view of snapshots and eval results under <code>.airlock/</code>.</p>
<table><tr><th>when</th><th>kind</th><th>id</th><th>verdict</th><th>file</th></tr>`)
	for _, e := range entries {
		b.WriteString("<tr>")
		fmt.Fprintf(&b, "<td>%s</td>", html.EscapeString(e.ModTime.UTC().Format(time.RFC3339)))
		fmt.Fprintf(&b, "<td>%s</td>", html.EscapeString(e.Kind))
		fmt.Fprintf(&b, "<td>%s</td>", html.EscapeString(e.ID))
		fmt.Fprintf(&b, `<td class="%s">%s</td>`, html.EscapeString(e.Verdict), html.EscapeString(e.Verdict))
		fmt.Fprintf(&b, `<td><a href="/raw?path=%s">json</a></td>`, html.EscapeString(e.Path))
		b.WriteString("</tr>")
	}
	b.WriteString(`</table></body></html>`)
	return b.String()
}

// Serve starts a read-only HTTP server. root must be the repo root; only paths under .airlock are served.
func Serve(addr, repoRoot string) error {
	air := filepath.Join(repoRoot, ".airlock")
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		entries, err := List(air)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(RenderHTML(entries)))
	})
	mux.HandleFunc("/raw", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Query().Get("path")
		abs, err := filepath.Abs(p)
		if err != nil {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		airAbs, _ := filepath.Abs(air)
		if !strings.HasPrefix(abs, airAbs) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	})
	fmt.Printf("Airlock history at http://127.0.0.1%s/\n", addr)
	return http.ListenAndServe(addr, mux)
}
