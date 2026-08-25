package sentinel_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ankittk/airlock/internal/manifest"
	"github.com/ankittk/airlock/internal/providers"
	"github.com/ankittk/airlock/internal/sentinel"
)

func TestProbeAndDetectDrift(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := sentinel.DefaultPath(dir)

	m := &manifest.Manifest{
		Models: []manifest.Model{{
			ID: "model-gpt-test", Provider: "mock", Model: "gpt-test",
			ContentHash: manifest.HashString("mock|gpt-test"),
		}},
	}
	if _, err := sentinel.ProbeAll(ctx, m, nil, path); err != nil {
		t.Fatal(err)
	}
	st, err := sentinel.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Records) != 1 || st.Records[0].Fingerprint == "" {
		t.Fatalf("expected stored fingerprint, got %+v", st.Records)
	}

	rep, err := sentinel.Check(ctx, m, nil, path)
	if err != nil {
		t.Fatal(err)
	}
	if rep.HasDrift() {
		t.Fatalf("unexpected drift on stable mock: %+v", rep.Drifts)
	}

	// Simulate silent provider drift: same model string, different output.
	old := providers.MockSentinelReply
	providers.MockSentinelReply = "AIRLOCK_SENTINEL_v1_DRIFT"
	t.Cleanup(func() { providers.MockSentinelReply = old })

	rep, err = sentinel.Check(ctx, m, nil, path)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.HasDrift() {
		t.Fatal("expected drift after provider output change")
	}
	if len(rep.Drifts) != 1 || !rep.Drifts[0].ConfigMatch {
		t.Fatalf("want config-stable drift, got %+v", rep.Drifts)
	}
}

func TestApplyToManifestChangesContentHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fp.json")
	m := &manifest.Manifest{
		Models: []manifest.Model{{
			ID: "model-gpt-test", Provider: "mock", Model: "gpt-test",
			ContentHash: manifest.HashString("mock|gpt-test"),
		}},
	}
	before := m.Models[0].ContentHash
	st := &sentinel.Store{
		Version: 1,
		Records: []sentinel.Record{{
			ModelID: "model-gpt-test", Provider: "mock", Model: "gpt-test",
			Fingerprint: "abc123fingerprint",
			PromptVer:   sentinel.PromptVersion,
		}},
	}
	sentinel.ApplyToManifest(m, st)
	if m.Models[0].ContentHash == before {
		t.Fatal("ApplyToManifest should fold fingerprint into content_hash")
	}
	if m.Models[0].ParamsHash == "" {
		t.Fatal("expected params_hash for config identity")
	}
	_ = path
}
