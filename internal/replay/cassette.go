package replay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/ankittk/airlock/internal/manifest"
)

type Mode string

const (
	ModeReplay Mode = "replay"
	ModeRecord Mode = "record"
	ModeLive   Mode = "live"
)

type MissPolicy string

const (
	MissFail MissPolicy = "fail"
	MissLive MissPolicy = "live"
)

// Entry is one recorded HTTP exchange.
type Entry struct {
	Key        string              `json:"key"`
	Method     string              `json:"method"`
	URL        string              `json:"url"`
	ReqBody    json.RawMessage     `json:"req_body,omitempty"`
	Status     int                 `json:"status"`
	RespHeader map[string][]string `json:"resp_header,omitempty"`
	RespBody   json.RawMessage     `json:"resp_body"`
}

// Store is a directory of JSONL cassettes keyed by request hash.
type Store struct {
	Dir string
	mu  sync.Mutex
	idx map[string]Entry
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{Dir: dir, idx: map[string]Entry{}}
	path := filepath.Join(dir, "cassettes.jsonl")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	for {
		var e Entry
		if err := dec.Decode(&e); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		s.idx[e.Key] = e
	}
	return s, nil
}

func (s *Store) Get(key string) (Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.idx[key]
	return e, ok
}

func (s *Store) Put(e Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.idx[e.Key] = e
	path := filepath.Join(s.Dir, "cassettes.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(e)
}

func RequestKey(method, url string, body []byte) string {
	canonical := method + "\n" + url + "\n" + string(body)
	return manifest.HashString(canonical)
}

// Transport wraps an http.RoundTripper with record/replay.
type Transport struct {
	Store   *Store
	Mode    Mode
	Miss    MissPolicy
	Wrapped http.RoundTripper
}

func (t *Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	wrapped := t.Wrapped
	if wrapped == nil {
		wrapped = http.DefaultTransport
	}
	body, err := readBody(r)
	if err != nil {
		return nil, err
	}
	key := RequestKey(r.Method, r.URL.String(), body)

	if t.Mode == ModeReplay || t.Mode == ModeRecord {
		if e, ok := t.Store.Get(key); ok {
			return entryToResponse(e), nil
		}
		if t.Mode == ModeReplay {
			if t.Miss == MissFail || t.Miss == "" {
				return nil, fmt.Errorf("cassette miss for %s %s (key=%s)", r.Method, r.URL.String(), key[:12])
			}
			// fall through to live
		}
	}

	if t.Mode == ModeLive || t.Mode == ModeRecord || (t.Mode == ModeReplay && t.Miss == MissLive) {
		r2 := r.Clone(r.Context())
		r2.Body = io.NopCloser(bytes.NewReader(body))
		r2.ContentLength = int64(len(body))
		resp, err := wrapped.RoundTrip(r2)
		if err != nil {
			return nil, err
		}
		if t.Mode == ModeRecord {
			respBody, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				return nil, err
			}
			e := Entry{
				Key:        key,
				Method:     r.Method,
				URL:        r.URL.String(),
				ReqBody:    json.RawMessage(body),
				Status:     resp.StatusCode,
				RespHeader: resp.Header.Clone(),
				RespBody:   json.RawMessage(respBody),
			}
			if err := t.Store.Put(e); err != nil {
				return nil, err
			}
			resp.Body = io.NopCloser(bytes.NewReader(respBody))
		}
		return resp, nil
	}
	return nil, fmt.Errorf("replay: unsupported mode %q", t.Mode)
}

func readBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewReader(b))
	return b, nil
}

func entryToResponse(e Entry) *http.Response {
	hdr := http.Header{}
	for k, vs := range e.RespHeader {
		for _, v := range vs {
			hdr.Add(k, v)
		}
	}
	if hdr.Get("Content-Type") == "" {
		hdr.Set("Content-Type", "application/json")
	}
	return &http.Response{
		StatusCode:    e.Status,
		Status:        http.StatusText(e.Status),
		Header:        hdr,
		Body:          io.NopCloser(bytes.NewReader(e.RespBody)),
		ContentLength: int64(len(e.RespBody)),
	}
}

// Client builds an http.Client using this transport.
func (t *Transport) Client() *http.Client {
	return &http.Client{Transport: t}
}
