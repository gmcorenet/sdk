package gmcore_asset

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type AssetType = string

const (
	AssetTypeCSS  AssetType = "css"
	AssetTypeJS   AssetType = "js"
	AssetTypeImg  AssetType = "img"
	AssetTypeFont AssetType = "font"
)

type AssetPosition = string

const (
	PositionHead      AssetPosition = "head"
	PositionBodyStart AssetPosition = "body_start"
	PositionBodyEnd   AssetPosition = "body_end"
)

type assetRef struct {
	Namespace string
	Path      string
}

func ParseAssetReference(ref string) (assetRef, bool) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) == 2 {
		return assetRef{Namespace: parts[0], Path: parts[1]}, true
	}
	return assetRef{Path: ref}, true
}

type Collector struct {
	registry   *Registry
	themeMgr   *ThemeManager
	manifest   map[string]ManifestEntry
	baseURL    string
	version    string
	publicDir  string
	versionMap map[string]string
}

func NewCollector(opts ...CollectorOption) *Collector {
	c := &Collector{
		registry:   DefaultRegistry(),
		themeMgr:   DefaultThemeManager(),
		manifest:   make(map[string]ManifestEntry),
		versionMap: make(map[string]string),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type CollectorOption func(*Collector)

func WithRegistry(r *Registry) CollectorOption {
	return func(c *Collector) {
		c.registry = r
	}
}

func WithThemeManager(tm *ThemeManager) CollectorOption {
	return func(c *Collector) {
		c.themeMgr = tm
	}
}

func WithManifest(m Manifest) CollectorOption {
	return func(c *Collector) {
		c.manifest = m
	}
}

func WithBaseURL(url string) CollectorOption {
	return func(c *Collector) {
		c.baseURL = url
	}
}

func WithCollectorVersion(v string) CollectorOption {
	return func(c *Collector) {
		c.version = v
	}
}

func WithPublicDir(dir string) CollectorOption {
	return func(c *Collector) {
		c.publicDir = dir
	}
}

func (c *Collector) SetBaseURL(url string) {
	c.baseURL = url
}

func (c *Collector) SetVersion(v string) {
	c.version = v
}

func (c *Collector) CSS(position AssetPosition) []string {
	return c.Assets(AssetTypeCSS, position)
}

func (c *Collector) JS(position AssetPosition) []string {
	return c.Assets(AssetTypeJS, position)
}

func (c *Collector) Assets(assetType AssetType, position AssetPosition) []string {
	theme := c.themeMgr.Active()
	assets := c.registry.GetForThemeAndPosition(theme, position)

	var filtered []Asset
	for _, a := range assets {
		if a.Type == assetType {
			filtered = append(filtered, a)
		}
	}

	if position == PositionHead {
		commonAssets := c.registry.GetForThemeAndPosition("", PositionHead)
		for _, a := range commonAssets {
			if a.Type == assetType {
				filtered = append(filtered, a)
			}
		}
	}

	filtered = c.sortByDeps(filtered)

	var tags []string
	for _, asset := range filtered {
		tag := c.renderAsset(asset)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

func (c *Collector) Theme() []string {
	return c.themeMgr.RenderThemeTags()
}

func (c *Collector) AllTags() []string {
	var tags []string
	tags = append(tags, c.Theme()...)
	tags = append(tags, c.CSS(PositionHead)...)
	return tags
}

func (c *Collector) BodyEndTags() []string {
	var tags []string
	tags = append(tags, c.JS(PositionBodyEnd)...)
	return tags
}

func (c *Collector) renderAsset(asset Asset) string {
	path := c.resolvePath(asset.Path)

	if asset.Inline {
		return c.renderInline(asset.Type, path)
	}

	var tag string
	switch asset.Type {
	case AssetTypeCSS:
		tag = fmt.Sprintf(`<link rel="stylesheet" href="%s">`, path)
	case AssetTypeJS:
		tag = fmt.Sprintf(`<script src="%s"></script>`, path)
	default:
		tag = fmt.Sprintf(`<%s href="%s">`, "link", path)
	}
	return tag
}

func (c *Collector) ResolvePath(assetType AssetType, assetPath string) string {
	asset := Asset{Type: assetType, Path: assetPath}
	return c.resolvePath(asset.Path)
}

func (c *Collector) renderInline(assetType AssetType, path string) string {
	if c.publicDir == "" {
		return ""
	}
	fullPath := path
	if !filepath.IsAbs(path) {
		fullPath = filepath.Join(c.publicDir, path)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Sprintf(`<!-- Error loading inline asset: %s -->`, html.EscapeString(path))
	}

	content := string(data)
	switch assetType {
	case AssetTypeCSS:
		return fmt.Sprintf(`<style>%s</style>`, content)
	case AssetTypeJS:
		return fmt.Sprintf(`<script>%s</script>`, content)
	}
	return ""
}

func (c *Collector) resolvePath(assetPath string) string {
	if strings.HasPrefix(assetPath, "/") {
		if entry, ok := c.manifest[assetPath]; ok {
			assetPath = entry.Version
		}
	}

	if c.version != "" && !strings.Contains(assetPath, "?") {
		assetPath = assetPath + "?v=" + c.version
	}

	if c.baseURL != "" && !strings.HasPrefix(assetPath, "http") {
		base := strings.TrimRight(c.baseURL, "/")
		assetPath = base + assetPath
	}

	return assetPath
}

func (c *Collector) sortByDeps(assets []Asset) []Asset {
	result := make([]Asset, 0, len(assets))
	added := make(map[string]bool)

	var add func(a Asset)
	add = func(a Asset) {
		key := a.Namespace + ":" + a.Path
		if added[key] {
			return
		}
		for _, dep := range a.Depends {
			depRef, ok := ParseAssetReference(dep)
			if !ok {
				continue
			}
			depAsset := Asset{
				Namespace: depRef.Namespace,
				Path:      depRef.Path,
				Type:      a.Type,
			}
			add(depAsset)
		}
		result = append(result, a)
		added[key] = true
	}

	for _, a := range assets {
		add(a)
	}

	return result
}

func (c *Collector) RenderCSS(position AssetPosition) string {
	return c.renderTags(c.CSS(position))
}

func (c *Collector) RenderJS(position AssetPosition) string {
	return c.renderTags(c.JS(position))
}

func (c *Collector) RenderTheme() string {
	return c.renderTags(c.Theme())
}

func (c *Collector) RenderAll() string {
	return c.renderTags(c.AllTags())
}

func (c *Collector) RenderBodyEnd() string {
	return c.renderTags(c.BodyEndTags())
}

func (c *Collector) renderTags(tags []string) string {
	var buf bytes.Buffer
	for _, tag := range tags {
		buf.WriteString(tag)
		buf.WriteString("\n")
	}
	return buf.String()
}

func (c *Collector) LoadManifest(path string) error {
	m, err := LoadManifestFile(path)
	if err != nil {
		return err
	}
	c.manifest = m.Assets
	return nil
}

func (c *Collector) BuildManifest(assets []Asset) map[string]ManifestEntry {
	m := make(map[string]ManifestEntry)
	for _, a := range assets {
		if strings.HasPrefix(a.Path, "/") {
			m[a.Path] = ManifestEntry{Version: a.Path}
		}
	}
	return m
}

func (c *Collector) ScanAndRegister(dir string, namespace string) error {
	scanner := NewScanner(dir)

	cssAssets, err := scanner.ScanDir(filepath.Join(dir, "assets", "css"), true)
	if err != nil {
		return err
	}
	for _, a := range cssAssets {
		c.registry.Register(AssetTypeCSS, namespace+":"+a.Path, WithPosition(PositionHead))
	}

	jsAssets, err := scanner.ScanDir(filepath.Join(dir, "assets", "js"), true)
	if err != nil {
		return err
	}
	for _, a := range jsAssets {
		c.registry.Register(AssetTypeJS, namespace+":"+a.Path, WithPosition(PositionBodyEnd))
	}

	themeAssets, err := scanner.ScanThemesDir(true)
	if err != nil {
		return err
	}
	for _, a := range themeAssets {
		opts := []AssetOption{WithPosition(PositionHead)}
		if a.Theme != "" {
			opts = append(opts, WithTheme(a.Theme))
		}
		c.registry.Register(AssetTypeCSS, namespace+":"+a.Path, opts...)
	}

	return nil
}

func (c *Collector) ScanThemes(dir, namespace string) error {
	scanner := NewScanner(dir)
	assets, err := scanner.ScanThemesDir(true)
	if err != nil {
		return err
	}
	for _, a := range assets {
		opts := []AssetOption{WithPosition(PositionHead)}
		if a.Theme != "" {
			opts = append(opts, WithTheme(a.Theme))
		}
		c.registry.Register(AssetTypeCSS, namespace+":"+a.Path, opts...)
	}
	return nil
}

func (c *Collector) SortedAssetList() []Asset {
	all := c.registry.GetAll()
	theme := c.themeMgr.Active()

	var css, js, common []Asset
	for _, a := range all {
		if !a.MatchesTheme(theme) && !a.IsCommon() {
			continue
		}
		if a.Type == AssetTypeCSS {
			if a.IsCommon() {
				common = append(common, a)
			} else {
				css = append(css, a)
			}
		} else if a.Type == AssetTypeJS {
			js = append(js, a)
		}
	}

	sort.Slice(common, func(i, j int) bool {
		return common[i].Path < common[j].Path
	})
	sort.Slice(css, func(i, j int) bool {
		return css[i].Path < css[j].Path
	})
	sort.Slice(js, func(i, j int) bool {
		return js[i].Path < js[j].Path
	})

	result := make([]Asset, 0, len(common)+len(css)+len(js))
	result = append(result, common...)
	result = append(result, css...)
	result = append(result, js...)
	return result
}