package gmcoreasset

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ManifestV1 struct {
	Version string                 `json:"version"`
	BuiltAt string                 `json:"built_at"`
	BaseURL string                 `json:"base_url,omitempty"`
	Assets  map[string]ManifestEntry `json:"assets"`
}

type ManifestEntry struct {
	Version  string `json:"version"`
	Integrity string `json:"integrity,omitempty"`
	FileSize  int64  `json:"file_size,omitempty"`
}

func NewManifest() *ManifestV1 {
	return &ManifestV1{
		Version: "1.0",
		Assets:  make(map[string]ManifestEntry),
	}
}

func LoadManifestFile(path string) (*ManifestV1, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest ManifestV1
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (m *ManifestV1) Save(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (m *ManifestV1) Add(assetPath, version string) {
	m.Assets[assetPath] = ManifestEntry{
		Version: version,
	}
}

func (m *ManifestV1) Get(assetPath string) (ManifestEntry, bool) {
	entry, ok := m.Assets[assetPath]
	return entry, ok
}

func (m *ManifestV1) Has(assetPath string) bool {
	_, ok := m.Assets[assetPath]
	return ok
}

func (m *ManifestV1) ResolvePath(originalPath string) (string, bool) {
	entry, ok := m.Assets[originalPath]
	if !ok {
		return originalPath, false
	}
	if entry.Version == "" {
		return originalPath, false
	}
	return entry.Version, true
}

type ManifestBuilder struct {
	publicDir   string
	baseDir     string
	version     string
	algorithm   string
	excludeDirs []string
}

func NewManifestBuilder(publicDir, baseDir string) *ManifestBuilder {
	return &ManifestBuilder{
		publicDir:   publicDir,
		baseDir:     baseDir,
		version:     "",
		algorithm:   "sha256",
		excludeDirs: []string{"node_modules", ".git", "vendor"},
	}
}

func (mb *ManifestBuilder) SetVersion(v string) *ManifestBuilder {
	mb.version = v
	return mb
}

func (mb *ManifestBuilder) SetAlgorithm(algo string) *ManifestBuilder {
	mb.algorithm = algo
	return mb
}

func (mb *ManifestBuilder) SetExcludeDirs(dirs ...string) *ManifestBuilder {
	mb.excludeDirs = dirs
	return mb
}

func (mb *ManifestBuilder) Build() (*ManifestV1, error) {
	manifest := NewManifest()
	manifest.BaseURL = mb.baseDir

	if mb.version != "" {
		manifest.Version = mb.version
	}
	manifest.BuiltAt = time.Now().UTC().Format(time.RFC3339)

	if mb.publicDir == "" {
		return manifest, nil
	}

	err := filepath.Walk(mb.publicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := filepath.Base(path)
			for _, excluded := range mb.excludeDirs {
				if name == excluded {
					return filepath.SkipDir
				}
			}
			return nil
		}

		ext := strings.TrimPrefix(filepath.Ext(path), ".")
		if !mb.isAssetExtension(ext) {
			return nil
		}

		relPath, err := filepath.Rel(mb.publicDir, path)
		if err != nil {
			return nil
		}

		versionedPath := mb.computeVersionedPath(relPath)
		integrity := mb.computeIntegrity(path)

		entry := ManifestEntry{
			Version:   versionedPath,
			Integrity: integrity,
			FileSize:  info.Size(),
		}

		manifest.Assets["/"+filepath.ToSlash(relPath)] = entry
		return nil
	})

	if err != nil {
		return nil, err
	}

	return manifest, nil
}

func (mb *ManifestBuilder) isAssetExtension(ext string) bool {
	assetExts := []string{"css", "js", "png", "jpg", "jpeg", "gif", "svg", "webp", "ico", "woff", "woff2", "ttf", "eot"}
	for _, e := range assetExts {
		if ext == e {
			return true
		}
	}
	return false
}

func (mb *ManifestBuilder) computeVersionedPath(original string) string {
	dir := filepath.Dir(original)
	name := filepath.Base(original)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	hash := mb.computeHashForFile(filepath.Join(mb.publicDir, original))
	versioned := fmt.Sprintf("%s/%s.%s%s", dir, base, hash[:8], ext)
	return "/" + filepath.ToSlash(versioned)
}

func (mb *ManifestBuilder) computeHashForFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (mb *ManifestBuilder) computeIntegrity(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	switch mb.algorithm {
	case "sha256":
		hash := sha256.Sum256(data)
		return "sha256-" + hex.EncodeToString(hash[:])
	default:
		return ""
	}
}

func BuildManifestForDir(dir, namespace string) (*ManifestV1, error) {
	scanner := NewScanner(dir)
	assets, err := scanner.ScanAssetsDir(true)
	if err != nil {
		return nil, err
	}

	manifest := NewManifest()
	for _, a := range assets {
		manifest.Add(a.Path, a.Path)
	}

	return manifest, nil
}

func MergeManifests(base, overlay *ManifestV1) *ManifestV1 {
	result := NewManifest()
	result.Version = overlay.Version
	if result.Version == "" {
		result.Version = base.Version
	}
	result.BaseURL = overlay.BaseURL
	if result.BaseURL == "" {
		result.BaseURL = base.BaseURL
	}

	for k, v := range base.Assets {
		result.Assets[k] = v
	}
	for k, v := range overlay.Assets {
		result.Assets[k] = v
	}

	return result
}

type ManifestDiff struct {
	Added   []string
	Removed []string
	Updated []string
}

func DiffManifests(old, new *ManifestV1) ManifestDiff {
	var diff ManifestDiff

	for path := range old.Assets {
		if _, ok := new.Assets[path]; !ok {
			diff.Removed = append(diff.Removed, path)
		} else if old.Assets[path] != new.Assets[path] {
			diff.Updated = append(diff.Updated, path)
		}
	}

	for path := range new.Assets {
		if _, ok := old.Assets[path]; !ok {
			diff.Added = append(diff.Added, path)
		}
	}

	sort.Strings(diff.Added)
	sort.Strings(diff.Removed)
	sort.Strings(diff.Updated)

	return diff
}