package gmcore_asset

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Asset struct {
	Path      string
	Content   []byte
	Type      string
	Inline    bool
	Namespace string
	Depends   []string
	Theme     string
	Position  string
	Version   string
}

func (a Asset) MatchesTheme(theme string) bool {
	return a.Theme == "" || a.Theme == theme
}

func (a Asset) IsCommon() bool {
	return a.Theme == ""
}

type Manager struct {
	roots []string
	cache map[string]*Asset
}

func NewManager(roots ...string) *Manager {
	return &Manager{
		roots: roots,
		cache: make(map[string]*Asset),
	}
}

func (m *Manager) AddRoot(root string) {
	m.roots = append(m.roots, root)
}

func (m *Manager) Get(path string) (*Asset, error) {
	if cached, ok := m.cache[path]; ok {
		return cached, nil
	}

	cleanPath := filepath.Clean(path)
	if strings.HasPrefix(cleanPath, "..") {
		return nil, fmt.Errorf("path traversal attempt detected: %s", path)
	}

	for _, root := range m.roots {
		fullPath := filepath.Join(root, cleanPath)
		realPath, err := filepath.EvalSymlinks(fullPath)
		if err != nil && !os.IsNotExist(err) {
			continue
		}
		realRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(realPath, realRoot) {
			continue
		}
		content, err := os.ReadFile(fullPath)
		if err == nil {
			asset := &Asset{
				Path:    fullPath,
				Content: content,
				Type:    detectType(path),
			}
			m.cache[path] = asset
			return asset, nil
		}
	}
	return nil, os.ErrNotExist
}

func (m *Manager) Exists(path string) bool {
	_, err := m.Get(path)
	return err == nil
}

func (m *Manager) List(path string) ([]string, error) {
	var files []string
	for _, root := range m.roots {
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

func (m *Manager) Glob(pattern string) ([]string, error) {
	var matches []string
	for _, root := range m.roots {
		fullPattern := filepath.Join(root, pattern)
		entries, err := filepath.Glob(fullPattern)
		if err != nil {
			continue
		}
		for _, e := range entries {
			matches = append(matches, e)
		}
	}
	return matches, nil
}

func (m *Manager) ClearCache() {
	m.cache = make(map[string]*Asset)
}

func detectType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	case ".ico":
		return "image/x-icon"
	case ".pdf":
		return "application/pdf"
	case ".html":
		return "text/html"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

type Fingerprinter struct {
	assets *Manager
}

func NewFingerprinter(assets *Manager) *Fingerprinter {
	return &Fingerprinter{assets: assets}
}

func (f *Fingerprinter) Fingerprint(path string) (string, error) {
	asset, err := f.assets.Get(path)
	if err != nil {
		return path, err
	}
	hash := simpleHash(string(asset.Content))
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	return base + "." + hash + ext, nil
}

func simpleHash(s string) string {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	return strings.ToLower(fmt.Sprintf("%x", h))
}
