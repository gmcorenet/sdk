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
