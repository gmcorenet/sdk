package gmcoreasset

import (
	"encoding/json"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	gmerr "github.com/gmcorenet/gmcore-error"
)

type Manifest map[string]string

type Manager struct {
	BaseURL  string
	Manifest Manifest
	Version  string
}

func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func (m Manager) URL(assetPath string) string {
	assetPath = "/" + strings.TrimLeft(strings.TrimSpace(assetPath), "/")
	if versioned := strings.TrimSpace(m.Manifest[assetPath]); versioned != "" {
		assetPath = versioned
	}
	base := strings.TrimRight(strings.TrimSpace(m.BaseURL), "/")
	if base == "" {
		base = ""
	}
	target := path.Clean("/" + strings.TrimLeft(assetPath, "/"))
	if strings.TrimSpace(m.Version) == "" {
		return base + target
	}
	values := url.Values{}
	values.Set("v", strings.TrimSpace(m.Version))
	return base + target + "?" + values.Encode()
}

func (m Manager) PackageURL(namespace string, assetPath string) string {
	namespace = strings.Trim(strings.TrimSpace(namespace), "/")
	if namespace != "" {
		assetPath = path.Join(namespace, assetPath)
	}
	return m.URL(assetPath)
}

type AssetType string

const (
	AssetTypeCSS AssetType = "css"
	AssetTypeJS  AssetType = "js"
	AssetTypeImg AssetType = "img"
)

type AssetPosition string

const (
	PositionHead    AssetPosition = "head"
	PositionBodyEnd AssetPosition = "body_end"
)

type Asset struct {
	Type      AssetType     `json:"type"`
	Path      string        `json:"path"`
	Version   string        `json:"version,omitempty"`
	Position  AssetPosition `json:"position,omitempty"`
	Theme     string        `json:"theme,omitempty"`
	Depends   []string      `json:"depends,omitempty"`
	Inline    bool          `json:"inline,omitempty"`
	Namespace string        `json:"namespace,omitempty"`
}

type AssetCollection struct {
	Assets []Asset `json:"assets"`
}

func (a Asset) IsCommon() bool {
	return a.Theme == "" || a.Theme == "_common"
}

func (a Asset) MatchesTheme(themeName string) bool {
	if a.IsCommon() {
		return true
	}
	return a.Theme == themeName
}

func (a Asset) FullPath(baseDir string) string {
	if baseDir == "" {
		return a.Path
	}
	return filepath.Join(baseDir, a.Path)
}

type AssetReference struct {
	Namespace string
	Path      string
}

func ParseAssetReference(ref string) (AssetReference, bool) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 {
		return AssetReference{}, false
	}
	return AssetReference{
		Namespace: parts[0],
		Path:      parts[1],
	}, true
}