package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanOpenAIStackPython(t *testing.T) {
	root := t.TempDir()
	code := `
from langgraph.prebuilt import create_react_agent
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(model="gpt-4o-mini")
graph = create_react_agent(llm, tools=[])
`
	if err := os.MkdirAll(filepath.Join(root, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app", "agent.py"), []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, mod := range m.Models {
		if mod.Model == "gpt-4o-mini" {
			found = true
			if mod.Source == "" {
				t.Fatal("expected source path")
			}
		}
	}
	if !found {
		t.Fatalf("expected gpt-4o-mini from langgraph scan, got %+v", m.Models)
	}
	foundLG := false
	for _, s := range m.Sources {
		if s.Kind == "langgraph-scan" {
			foundLG = true
		}
	}
	if !foundLG {
		t.Fatal("expected langgraph-scan source kind")
	}
}

func TestScanOpenAIStackGo(t *testing.T) {
	root := t.TempDir()
	code := `package main

type cfg struct {
	Model string
}

func main() {
	c := cfg{Model: "claude-sonnet-4-20250514"}
	_ = c
}
`
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, mod := range m.Models {
		if mod.Model == "claude-sonnet-4-20250514" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected claude model from go scan, got %+v", m.Models)
	}
}
