package gmcoreasset

import (
	"encoding/json"
	"net/url"
	"os"
	"path"
	"strings"
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
