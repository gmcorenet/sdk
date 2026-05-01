package gmcoreresolver

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	AppRoot     string
	SystemRoot  string
	BundleRoots []string
}

type ResolvedFile struct {
	Path   string
	Source string
}

func ResolveRelativeFile(cfg Config, relativePath string) (ResolvedFile, bool) {
	relativePath = cleanRelative(relativePath)
	if relativePath == "" {
		return ResolvedFile{}, false
	}

	candidates := make([]ResolvedFile, 0, 2+len(cfg.BundleRoots))
	// Override precedence is always app > bundles > system.
	if strings.TrimSpace(cfg.AppRoot) != "" {
		candidates = append(candidates, ResolvedFile{
			Path:   filepath.Join(cfg.AppRoot, relativePath),
			Source: "app",
		})
	}
	for _, root := range cfg.BundleRoots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		candidates = append(candidates, ResolvedFile{
			Path:   filepath.Join(root, relativePath),
			Source: bundleSource(root),
		})
	}
	if strings.TrimSpace(cfg.SystemRoot) != "" {
		candidates = append(candidates, ResolvedFile{
			Path:   filepath.Join(cfg.SystemRoot, relativePath),
			Source: "system",
		})
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate.Path)
		if err == nil && !info.IsDir() {
			return candidate, true
		}
	}
	return ResolvedFile{}, false
}

func ResolveTemplate(cfg Config, name string) (ResolvedFile, bool) {
	return ResolveRelativeFile(cfg, filepath.Join("templates", cleanRelative(name)))
}

func ResolveSource(cfg Config, name string) (ResolvedFile, bool) {
	return ResolveRelativeFile(cfg, cleanRelative(name))
}

func cleanRelative(path string) string {
	path = filepath.Clean(strings.TrimSpace(path))
	path = strings.TrimPrefix(path, "/")
	if path == "." {
		return ""
	}
	return path
}

func bundleSource(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return "bundle"
	}
	manifest := filepath.Join(root, "bundle.yaml")
	data, err := os.ReadFile(manifest)
	if err != nil {
		return filepath.Base(root)
	}
	var manifestData struct {
		Name string `yaml:"name"`
	}
	if err := yaml.Unmarshal(data, &manifestData); err != nil {
		return filepath.Base(root)
	}
	if name := strings.TrimSpace(manifestData.Name); name != "" {
		return name
	}
	return filepath.Base(root)
}
