package replay

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecordReplay(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	rt := &Transport{Store: store, Mode: ModeRecord, Wrapped: http.DefaultTransport}
	client := rt.Client()
	resp, err := client.Post(upstream.URL+"/v1/chat", "application/json", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if string(body) != `{"ok":true}` {
		t.Fatalf("body %s", body)
	}

	// replay without upstream
	store2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	rt2 := &Transport{Store: store2, Mode: ModeReplay, Miss: MissFail, Wrapped: http.DefaultTransport}
	client2 := rt2.Client()
	upstream.Close()
	resp2, err := client2.Post(upstream.URL+"/v1/chat", "application/json", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	_ = resp2.Body.Close()
	if string(body2) != `{"ok":true}` {
		t.Fatalf("replay body %s", body2)
	}
}
