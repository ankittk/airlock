package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ankittk/airlock/internal/manifest"
)

// ponytail: line/regex parsers for lockfiles; upgrade path = ecosystem-native parsers.

var cargoPkgRE = regexp.MustCompile(`(?m)^name = "([^"]+)"\nversion = "([^"]+)"`)

func scanLockfileDeps(root string, m *manifest.Manifest) error {
	if err := scanGoSum(root, m); err != nil {
		return err
	}
	if err := scanPackageLock(root, m); err != nil {
		return err
	}
	return scanCargoLock(root, m)
}

func scanGoSum(root string, m *manifest.Manifest) error {
	path := filepath.Join(root, "go.sum")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	m.Sources = append(m.Sources, manifest.Source{Kind: "go.sum", Path: "go.sum"})
	seen := depIndex(m)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		mod, ver := fields[0], fields[1]
		if strings.Contains(mod, "/go.mod ") {
			continue
		}
		id := slug(mod)
		if seen[id] {
			continue
		}
		h := fields[len(fields)-1]
		if !strings.HasPrefix(h, "h1:") && len(fields) >= 3 {
			h = manifest.HashString(mod + "|" + ver + "|" + fields[2])
		}
		m.Dependencies = append(m.Dependencies, manifest.Dependency{
			ID: id, Ecosystem: "go", Version: ver, Hash: h, Source: "go.sum",
		})
		seen[id] = true
	}
	return nil
}

func scanPackageLock(root string, m *manifest.Manifest) error {
	for _, name := range []string{"package-lock.json", "npm-shrinkwrap.json"} {
		path := filepath.Join(root, name)
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		m.Sources = append(m.Sources, manifest.Source{Kind: "package-lock", Path: name})
		var lock struct {
			Packages map[string]struct {
				Version string `json:"version"`
				Name    string `json:"name"`
			} `json:"packages"`
			Dependencies map[string]struct {
				Version string `json:"version"`
			} `json:"dependencies"`
		}
		if err := json.Unmarshal(data, &lock); err != nil {
			return err
		}
		seen := depIndex(m)
		add := func(id, ver, eco string) {
			if id == "" || ver == "" || seen[id] {
				return
			}
			m.Dependencies = append(m.Dependencies, manifest.Dependency{
				ID: id, Ecosystem: eco, Version: ver,
				Hash: manifest.HashString(id + "|" + ver), Source: name,
			})
			seen[id] = true
		}
		for path, pkg := range lock.Packages {
			if path == "" {
				continue
			}
			name := pkg.Name
			if name == "" {
				name = strings.TrimPrefix(path, "node_modules/")
			}
			add(slug(name), pkg.Version, "npm")
		}
		for name, pkg := range lock.Dependencies {
			add(slug(name), pkg.Version, "npm")
		}
		return nil
	}
	return nil
}

func scanCargoLock(root string, m *manifest.Manifest) error {
	path := filepath.Join(root, "Cargo.lock")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	m.Sources = append(m.Sources, manifest.Source{Kind: "cargo.lock", Path: "Cargo.lock"})
	seen := depIndex(m)
	for _, block := range strings.Split(string(data), "[[package]]") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		nameM := regexp.MustCompile(`(?m)^name = "([^"]+)"`).FindStringSubmatch(block)
		verM := regexp.MustCompile(`(?m)^version = "([^"]+)"`).FindStringSubmatch(block)
		if len(nameM) < 2 || len(verM) < 2 {
			continue
		}
		name, ver := nameM[1], verM[1]
		id := slug(name)
		if seen[id] {
			continue
		}
		m.Dependencies = append(m.Dependencies, manifest.Dependency{
			ID: id, Ecosystem: "cargo", Version: ver,
			Hash: manifest.HashString(name + "|" + ver), Source: "Cargo.lock",
		})
		seen[id] = true
	}
	_ = cargoPkgRE // kept for future stricter parse
	return nil
}

func depIndex(m *manifest.Manifest) map[string]bool {
	seen := map[string]bool{}
	for _, d := range m.Dependencies {
		seen[d.ID] = true
	}
	return seen
}
