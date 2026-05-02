package gmcorebundle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	gmerr "github.com/gmcorenet/gmcore-error"
)

type InstallSpec struct {
	Entities string `yaml:"entities"`
	Examples string `yaml:"examples"`
	Config   string `yaml:"config"`
}

type BootstrapSpec struct {
	Import string `yaml:"import"`
}

type RecipeSpec struct {
	File string `yaml:"file"`
}

type Manifest struct {
	Name         string        `yaml:"name"`
	Package      string        `yaml:"package"`
	Module       string        `yaml:"module"`
	Version      string        `yaml:"version"`
	Description  string        `yaml:"description"`
	Dependencies []string      `yaml:"dependencies"`
	Install      InstallSpec   `yaml:"install"`
	Bootstrap    BootstrapSpec `yaml:"bootstrap"`
	Recipe       RecipeSpec    `yaml:"recipe"`
	Root         string        `yaml:"-"`
	ManifestPath string        `yaml:"-"`
}

func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, err
	}
	if strings.TrimSpace(manifest.Name) == "" {
		return Manifest{}, fmt.Errorf("bundle manifest %s missing name", path)
	}
	manifest.ManifestPath = path
	manifest.Root = filepath.Dir(path)
	if strings.TrimSpace(manifest.Package) == "" {
		manifest.Package = manifest.Name
	}
	return manifest, nil
}

func (m Manifest) BootstrapImportPath() string {
	moduleName := m.ModulePath()
	importPath := strings.Trim(strings.TrimSpace(m.Bootstrap.Import), "/")
	if moduleName == "" || importPath == "" {
		return ""
	}
	if importPath == "." {
		return moduleName
	}
	return moduleName + "/" + importPath
}

func (m Manifest) ModulePath() string {
	if module := strings.TrimSpace(m.Module); module != "" {
		return module
	}
	packageName := strings.TrimSpace(m.Package)
	if packageName == "" {
		return ""
	}
	return "gmcore-" + packageName
}

func Discover(root string) (map[string]Manifest, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("missing bundle root")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	out := map[string]Manifest{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(root, entry.Name(), "bundle.yaml")
		if _, err := os.Stat(manifestPath); err != nil {
			continue
		}
		manifest, err := LoadManifest(manifestPath)
		if err != nil {
			return nil, err
		}
		out[manifest.Name] = manifest
	}
	return out, nil
}

func ResolveDependencies(all map[string]Manifest, names []string) ([]Manifest, error) {
	seen := map[string]struct{}{}
	order := make([]Manifest, 0, len(names))
	var visit func(string) error
	visit = func(name string) error {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil
		}
		if _, ok := seen[name]; ok {
			return nil
		}
		manifest, ok := all[name]
		if !ok {
			return fmt.Errorf("bundle %q not found", name)
		}
		seen[name] = struct{}{}
		for _, dep := range manifest.Dependencies {
			if err := visit(dep); err != nil {
				return err
			}
		}
		order = append(order, manifest)
		return nil
	}
	for _, name := range names {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	return order, nil
}

func DiscoverRoots(root string) ([]string, error) {
	manifests, err := Discover(root)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(manifests))
	for name := range manifests {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, manifests[name].Root)
	}
	return out, nil
}

func LoadRegisteredNames(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	blockRe := regexp.MustCompile(`var\s+Bundles\s*=\s*\[\]\s*string\s*\{([\s\S]*?)\}`)
	match := blockRe.FindStringSubmatch(content)
	if len(match) < 2 {
		return nil, nil
	}
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(match[1], -1)
	out := make([]string, 0, len(matches))
	seen := map[string]struct{}{}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out, nil
}

func ResolveRegisteredRoots(searchRoots []string, names []string) ([]string, error) {
	index := map[string]Manifest{}
	for _, root := range searchRoots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		discovered, err := Discover(root)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		for name, manifest := range discovered {
			if _, exists := index[name]; exists {
				continue
			}
			index[name] = manifest
		}
	}
	out := make([]string, 0, len(names))
	seen := map[string]struct{}{}
	for _, name := range names {
		manifest, ok := index[name]
		if !ok {
			return nil, fmt.Errorf("registered bundle %q not found", name)
		}
		if _, exists := seen[manifest.Root]; exists {
			continue
		}
		seen[manifest.Root] = struct{}{}
		out = append(out, manifest.Root)
	}
	return out, nil
}
