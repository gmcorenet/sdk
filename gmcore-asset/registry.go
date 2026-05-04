package gmcore_asset

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type Registry struct {
	mu      sync.RWMutex
	assets  []Asset
	byType  map[AssetType][]Asset
	byTheme map[string][]Asset
}

var defaultRegistry *Registry
var registryOnce sync.Once

func DefaultRegistry() *Registry {
	registryOnce.Do(func() {
		defaultRegistry = NewRegistry()
	})
	return defaultRegistry
}

func NewRegistry() *Registry {
	return &Registry{
		assets:  make([]Asset, 0),
		byType:  make(map[AssetType][]Asset),
		byTheme: make(map[string][]Asset),
	}
}

func (r *Registry) Register(assetType AssetType, ref string, opts ...AssetOption) error {
	parts, ok := ParseAssetReference(ref)
	if !ok || parts.Path == "" {
		return fmt.Errorf("invalid asset reference %q", ref)
	}

	asset := Asset{
		Type: assetType,
		Path: parts.Path,
	}

	if parts.Namespace != "" {
		asset.Namespace = parts.Namespace
	}

	for _, opt := range opts {
		opt(&asset)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.assets = append(r.assets, asset)
	r.rebuildIndexes()
	return nil
}

func (r *Registry) RegisterForTheme(theme string, assetType AssetType, ref string, opts ...AssetOption) error {
	opts = append(opts, WithTheme(theme))
	return r.Register(assetType, ref, opts...)
}

func (r *Registry) RegisterForPosition(pos AssetPosition, assetType AssetType, ref string, opts ...AssetOption) error {
	opts = append(opts, WithPosition(pos))
	return r.Register(assetType, ref, opts...)
}

func (r *Registry) Unregister(namespace, path string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	newAssets := make([]Asset, 0)
	for _, a := range r.assets {
		if a.Namespace == namespace && a.Path == path {
			continue
		}
		newAssets = append(newAssets, a)
	}
	r.assets = newAssets
	r.rebuildIndexes()
}

func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assets = nil
	r.byType = make(map[AssetType][]Asset)
	r.byTheme = make(map[string][]Asset)
}

func (r *Registry) GetAll() []Asset {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.assets
}

func (r *Registry) GetByType(assetType AssetType) []Asset {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byType[assetType]
}

func (r *Registry) GetByTheme(theme string) []Asset {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byTheme[theme]
}

func (r *Registry) GetForPosition(pos AssetPosition) []Asset {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []Asset
	for _, a := range r.assets {
		if a.Position == "" || a.Position == pos {
			result = append(result, a)
		}
	}
	return result
}

func (r *Registry) GetForThemeAndPosition(theme string, pos AssetPosition) []Asset {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []Asset
	for _, a := range r.assets {
		if !a.MatchesTheme(theme) {
			continue
		}
		if pos != "" && a.Position != "" && a.Position != pos {
			continue
		}
		result = append(result, a)
	}
	return result
}

func (r *Registry) LoadYAML(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return r.ParseYAML(data)
}

func (r *Registry) ParseYAML(data []byte) error {
	var config YAMLConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for ns, nsConfig := range config.Assets {
		for _, cssEntry := range nsConfig.CSS {
			asset := Asset{
				Type:      AssetTypeCSS,
				Namespace: ns,
				Path:      cssEntry.Path,
				Theme:     cssEntry.Theme,
			}
			if cssEntry.Position != "" {
				asset.Position = AssetPosition(cssEntry.Position)
			}
			r.assets = append(r.assets, asset)
		}
		for _, jsEntry := range nsConfig.JS {
			asset := Asset{
				Type:      AssetTypeJS,
				Namespace: ns,
				Path:      jsEntry.Path,
				Theme:     jsEntry.Theme,
			}
			if jsEntry.Position != "" {
				asset.Position = AssetPosition(jsEntry.Position)
			}
			r.assets = append(r.assets, asset)
		}
	}

	r.rebuildIndexes()
	return nil
}

func (r *Registry) LoadFromBundle(bundlePath string) error {
	assetsPath := filepath.Join(bundlePath, "assets.yaml")
	if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
		return nil
	}
	return r.LoadYAML(assetsPath)
}

func (r *Registry) rebuildIndexes() {
	r.byType = make(map[AssetType][]Asset)
	r.byTheme = make(map[string][]Asset)
	for _, a := range r.assets {
		r.byType[a.Type] = append(r.byType[a.Type], a)
		theme := a.Theme
		if theme == "" {
			theme = "_common"
		}
		r.byTheme[theme] = append(r.byTheme[theme], a)
	}
}

type AssetOption func(*Asset)

func WithTheme(theme string) AssetOption {
	return func(a *Asset) {
		a.Theme = theme
	}
}

func WithPosition(pos AssetPosition) AssetOption {
	return func(a *Asset) {
		a.Position = pos
	}
}

func WithVersion(version string) AssetOption {
	return func(a *Asset) {
		a.Version = version
	}
}

func WithDepends(deps ...string) AssetOption {
	return func(a *Asset) {
		a.Depends = deps
	}
}

func WithInline() AssetOption {
	return func(a *Asset) {
		a.Inline = true
	}
}

type YAMLConfig struct {
	Assets map[string]YAMLNamespaceConfig `yaml:"assets"`
}

type YAMLNamespaceConfig struct {
	CSS []YAMLAssetEntry `yaml:"css"`
	JS  []YAMLAssetEntry `yaml:"js"`
}

type YAMLAssetEntry struct {
	Path     string `yaml:"path"`
	Position string `yaml:"position,omitempty"`
	Theme    string `yaml:"theme,omitempty"`
}

type RegisterFunc func(*Registry)

func BundleRegisterFunc(bundlePath string) RegisterFunc {
	return func(r *Registry) {
		r.LoadFromBundle(bundlePath)
	}
}

func NamespacePrefix(ns string) string {
	return strings.TrimSuffix(ns, "/")
}