package replay_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ankittk/airlock/internal/replay"
)

func TestOpenAIShapedCassetteReplay(t *testing.T) {
	dir := t.TempDir()
	store, err := replay.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
	}))
	defer upstream.Close()

	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}`
	rt := &replay.Transport{Store: store, Mode: replay.ModeRecord, Wrapped: http.DefaultTransport}
	req, _ := http.NewRequest(http.MethodPost, upstream.URL+"/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	store2, _ := replay.Open(dir)
	rt2 := &replay.Transport{Store: store2, Mode: replay.ModeReplay, Miss: replay.MissFail}
	upstream.Close()
	req2, _ := http.NewRequest(http.MethodPost, upstream.URL+"/v1/chat/completions", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := rt2.RoundTrip(req2)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp2.Body)
	_ = resp2.Body.Close()
	if len(b) == 0 {
		t.Fatal("empty replay")
	}
}
