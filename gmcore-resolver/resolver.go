package gmcore_resolver

import (
	"os"
	"path/filepath"
	"strings"
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

type Resolver struct {
	roots []string
}

func New(roots ...string) *Resolver {
	return &Resolver{roots: roots}
}

func (r *Resolver) AddRoot(root string) {
	r.roots = append(r.roots, root)
}

func (r *Resolver) Resolve(path string) (string, error) {
	for _, root := range r.roots {
		fullPath := filepath.Join(root, path)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
	}
	return "", os.ErrNotExist
}

func (r *Resolver) Exists(path string) bool {
	_, err := r.Resolve(path)
	return err == nil
}

func (r *Resolver) List(path string) ([]string, error) {
	var files []string
	for _, root := range r.roots {
		fullPath := filepath.Join(root, path)
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			continue
		}
		for _, e := range entries {
			files = append(files, e.Name())
		}
	}
	return files, nil
}

func (r *Resolver) Glob(pattern string) ([]string, error) {
	var matches []string
	for _, root := range r.roots {
		fullPattern := filepath.Join(root, pattern)
		entries, err := filepath.Glob(fullPattern)
		if err != nil {
			continue
		}
		matches = append(matches, entries...)
	}
	return matches, nil
}

func (r *Resolver) Read(path string) ([]byte, error) {
	fullPath, err := r.Resolve(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(fullPath)
}

func (r *Resolver) TemplatePaths(name string) []string {
	exts := []string{".html", ".tmpl", ".gohtml", ".tpl"}
	paths := make([]string, 0)
	for _, ext := range exts {
		paths = append(paths, name+ext)
	}
	return paths
}

func (r *Resolver) FindTemplate(name string) (string, error) {
	for _, path := range r.TemplatePaths(name) {
		if fullPath, err := r.Resolve(path); err == nil {
			return fullPath, nil
		}
	}
	return "", os.ErrNotExist
}

func (r *Resolver) IsAbsolute(path string) bool {
	return filepath.IsAbs(path)
}

func (r *Resolver) Join(elem ...string) string {
	return filepath.Join(elem...)
}

func (r *Resolver) Ext(name string) string {
	return filepath.Ext(name)
}

func (r *Resolver) Base(name string) string {
	return filepath.Base(name)
}

func (r *Resolver) Dir(name string) string {
	return filepath.Dir(name)
}

func (r *Resolver) Clean(name string) string {
	return filepath.Clean(name)
}

func (r *Resolver) Rel(base, target string) (string, error) {
	return filepath.Rel(base, target)
}

func (r *Resolver) Split(name string) (dir, file string) {
	return filepath.Split(name)
}

func (r *Resolver) Match(pattern, name string) (bool, error) {
	return filepath.Match(pattern, name)
}

func (r *Resolver) Walk(root string, fn filepath.WalkFunc) error {
	for _, rootPath := range r.roots {
		fullRoot := filepath.Join(rootPath, root)
		filepath.Walk(fullRoot, fn)
	}
	return nil
}

func (r *Resolver) GetRoots() []string {
	return r.roots
}

func (r *Resolver) Normalize(name string) string {
	return strings.ReplaceAll(name, "\\", "/")
}

func ResolveRelativeFile(cfg Config, path string) (*ResolvedFile, bool) {
	path = cleanRelative(path)
	if path == "" {
		return nil, false
	}

	if cfg.AppRoot != "" {
		fullPath := filepath.Join(cfg.AppRoot, path)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			return &ResolvedFile{Path: fullPath, Source: "app"}, true
		}
	}

	for _, bundleRoot := range cfg.BundleRoots {
		fullPath := filepath.Join(bundleRoot, path)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			source := bundleSource(bundleRoot)
			return &ResolvedFile{Path: fullPath, Source: source}, true
		}
	}

	if cfg.SystemRoot != "" {
		fullPath := filepath.Join(cfg.SystemRoot, path)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			return &ResolvedFile{Path: fullPath, Source: "system"}, true
		}
	}

	return nil, false
}

func ResolveTemplate(cfg Config, path string) (*ResolvedFile, bool) {
	path = cleanRelative(path)
	if path == "" {
		return nil, false
	}

	templatesPath := filepath.Join("templates", path)
	if rf, ok := ResolveRelativeFile(cfg, templatesPath); ok {
		return rf, true
	}

	return ResolveRelativeFile(cfg, path)
}

func ResolveSource(cfg Config, path string) (*ResolvedFile, bool) {
	path = cleanRelative(path)
	if path == "" {
		return nil, false
	}

	internalPath := filepath.Join("internal", path)
	if rf, ok := ResolveRelativeFile(cfg, internalPath); ok {
		return rf, true
	}

	return ResolveRelativeFile(cfg, path)
}

func cleanRelative(path string) string {
	path = filepath.ToSlash(path)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimLeft(path, " ")
	for {
		newPath := filepath.Clean(path)
		if newPath == path {
			break
		}
		path = newPath
	}
	if path == "." {
		return ""
	}
	return path
}

func bundleSource(bundleDir string) string {
	if bundleDir == "" {
		return "bundle"
	}
	manifestPath := filepath.Join(bundleDir, "bundle.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return filepath.Base(bundleDir)
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "name:") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			if name != "" {
				return name
			}
		}
	}
	return filepath.Base(bundleDir)
}
