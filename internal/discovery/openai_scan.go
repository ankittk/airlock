package discovery

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xdlc-labs/airlock/internal/manifest"
)

// ponytail: regex + go/ast heuristics, not full Python/TS AST; upgrade path = tree-sitter per language.

var (
	openAIModelRE     = regexp.MustCompile(`(?i)(?:model|model_name|model_id)\s*[=:]\s*["']([a-z0-9._:/-]+)["']`)
	chatOpenAIRE      = regexp.MustCompile(`(?i)ChatOpenAI\s*\([^)]*model\s*[=:]\s*["']([a-z0-9._:/-]+)["']`)
	openAIClientRE    = regexp.MustCompile(`(?i)OpenAI\s*\([^)]*model\s*[=:]\s*["']([a-z0-9._:/-]+)["']`)
	langGraphImportRE = regexp.MustCompile(`(?i)(?:from|import)\s+langgraph[\w.]*|langgraph\.`)
	langGraphAgentRE  = regexp.MustCompile(`(?i)create_react_agent\s*\(|StateGraph\s*\(`)
)

// scanOpenAIStack discovers models from OpenAI SDK + LangGraph patterns in source trees.
func scanOpenAIStack(root string, m *manifest.Manifest) error {
	seen := map[string]bool{}
	for _, x := range m.Models {
		seen[x.Model] = true
	}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", ".airlock", "vendor", "__pycache__", ".venv", "venv":
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".py", ".ts", ".tsx", ".js", ".jsx":
			models, langgraph := scanTextModels(path, root)
			for _, model := range models {
				addScannedModel(m, seen, model, path, root, langgraph)
			}
		case ".go":
			models := scanGoModels(path, root)
			for _, model := range models {
				addScannedModel(m, seen, model, path, root, false)
			}
		}
		return nil
	})
	return err
}

func scanTextModels(path, root string) ([]string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	text := string(data)
	langgraph := langGraphImportRE.MatchString(text) || langGraphAgentRE.MatchString(text)
	var models []string
	for _, re := range []*regexp.Regexp{openAIModelRE, chatOpenAIRE, openAIClientRE} {
		for _, match := range re.FindAllStringSubmatch(text, -1) {
			if len(match) > 1 && looksLikeModel(match[1]) {
				models = append(models, match[1])
			}
		}
	}
	return unique(models), langgraph
}

func scanGoModels(path, root string) []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil
	}
	var models []string
	ast.Inspect(f, func(n ast.Node) bool {
		be, ok := n.(*ast.BasicLit)
		if !ok || be.Kind != token.STRING {
			return true
		}
		val := strings.Trim(be.Value, `"`)
		if !looksLikeModel(val) {
			return true
		}
		// ponytail: only strings near Model field names in Go struct literals
		if parentIsModelField(fset, f, be) {
			models = append(models, val)
		}
		return true
	})
	return unique(models)
}

func parentIsModelField(fset *token.FileSet, f *ast.File, lit *ast.BasicLit) bool {
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok || kv.Value != lit {
			return true
		}
		if ident, ok := kv.Key.(*ast.Ident); ok {
			name := strings.ToLower(ident.Name)
			if name == "model" || strings.HasSuffix(name, "model") {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func addScannedModel(m *manifest.Manifest, seen map[string]bool, model, path, root string, langgraph bool) {
	if seen[model] {
		return
	}
	seen[model] = true
	id := "model-" + slug(model)
	if hasModel(m, id) {
		return
	}
	src := "openai-sdk-scan"
	detail := rel(root, path)
	if langgraph {
		src = "langgraph-scan"
	}
	provider := guessProvider(model)
	m.Models = append(m.Models, manifest.Model{
		ID: id, Provider: provider, Model: model,
		ContentHash: manifest.HashString(provider + "|" + model),
		Source:      detail, Confidence: "medium",
	})
	m.Sources = append(m.Sources, manifest.Source{
		Kind: src, Path: detail, Detail: model,
	})
}
