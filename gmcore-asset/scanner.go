package gmcoreasset

import (
	"os"
	"path/filepath"
	"strings"
)

type Scanner struct {
	baseDir    string
	namespaces map[string]string
}

func NewScanner(baseDir string) *Scanner {
	return &Scanner{
		baseDir:    baseDir,
		namespaces: make(map[string]string),
	}
}

func (s *Scanner) ScanPatterns(patterns ...string) ([]Asset, error) {
	var assets []Asset
	for _, pattern := range patterns {
		found, err := s.ScanPattern(pattern)
		if err != nil {
			return nil, err
		}
		assets = append(assets, found...)
	}
	return assets, nil
}

func (s *Scanner) ScanPattern(pattern string) ([]Asset, error) {
	var assets []Asset
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || info.IsDir() {
			continue
		}
		rel, err := filepath.Rel(s.baseDir, match)
		if err != nil {
			continue
		}
		asset := s.fileToAsset(rel)
		assets = append(assets, asset)
	}
	return assets, nil
}

func (s *Scanner) ScanDir(dir string, recursive bool) ([]Asset, error) {
	var assets []Asset
	root := filepath.Join(s.baseDir, dir)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if !recursive && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(s.baseDir, path)
		if err != nil {
			return nil
		}
		asset := s.fileToAsset(rel)
		assets = append(assets, asset)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return assets, nil
}

func (s *Scanner) ScanAssetsDir(recursive bool) ([]Asset, error) {
	assetsDir := filepath.Join(s.baseDir, "assets")
	if _, err := os.Stat(assetsDir); os.IsNotExist(err) {
		return nil, nil
	}
	var allAssets []Asset

	cssDir := filepath.Join(assetsDir, "css")
	if info, err := os.Stat(cssDir); err == nil && info.IsDir() {
		cssAssets, err := s.scanExtensionsDir(cssDir, "css", recursive)
		if err != nil {
			return nil, err
		}
		allAssets = append(allAssets, cssAssets...)
	}

	jsDir := filepath.Join(assetsDir, "js")
	if info, err := os.Stat(jsDir); err == nil && info.IsDir() {
		jsAssets, err := s.scanExtensionsDir(jsDir, "js", recursive)
		if err != nil {
			return nil, err
		}
		allAssets = append(allAssets, jsAssets...)
	}

	imgDir := filepath.Join(assetsDir, "images")
	if info, err := os.Stat(imgDir); err == nil && info.IsDir() {
		imgAssets, err := s.scanExtensionsDir(imgDir, "img", recursive)
		if err != nil {
			return nil, err
		}
		allAssets = append(allAssets, imgAssets...)
	}

	return allAssets, nil
}

func (s *Scanner) ScanThemesDir(recursive bool) ([]Asset, error) {
	themesDir := filepath.Join(s.baseDir, "themes")
	if _, err := os.Stat(themesDir); os.IsNotExist(err) {
		return nil, nil
	}
	var allAssets []Asset

	entries, err := os.ReadDir(themesDir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		themeName := entry.Name()
		if themeName == "_common" {
			themeName = ""
		}

		themeDir := filepath.Join(themesDir, entry.Name())
		cssFiles, err := s.scanExtensionsDir(themeDir, "css", recursive)
		if err != nil {
			continue
		}
		for i := range cssFiles {
			if cssFiles[i].Type == AssetTypeCSS && themeName != "" {
				cssFiles[i].Theme = themeName
			}
		}
		allAssets = append(allAssets, cssFiles...)
	}
	return allAssets, nil
}

func (s *Scanner) scanExtensionsDir(dir, ext string, recursive bool) ([]Asset, error) {
	var assets []Asset
	pattern := filepath.Join(dir, "*."+ext)
	if recursive {
		pattern = filepath.Join(dir, "**", "*."+ext)
	}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || info.IsDir() {
			continue
		}
		rel, err := filepath.Rel(s.baseDir, match)
		if err != nil {
			continue
		}
		asset := s.fileToAsset(rel)
		assets = append(assets, asset)
	}
	return assets, nil
}

func (s *Scanner) fileToAsset(relPath string) Asset {
	relPath = filepath.ToSlash(relPath)
	ext := strings.TrimPrefix(filepath.Ext(relPath), ".")

	var assetType AssetType
	switch ext {
	case "css":
		assetType = AssetTypeCSS
	case "js":
		assetType = AssetTypeJS
	case "png", "jpg", "jpeg", "gif", "svg", "webp", "ico":
		assetType = AssetTypeImg
	default:
		assetType = AssetTypeImg
	}

	return Asset{
		Type:     assetType,
		Path:     relPath,
		Position: PositionHead,
	}
}

func (s *Scanner) RegisterNamespace(alias, path string) {
	s.namespaces[alias] = path
}

func (s *Scanner) ResolveNamespace(alias string) (string, bool) {
	path, ok := s.namespaces[alias]
	return path, ok
}

func ScanBundleAssets(bundlePath string) ([]Asset, error) {
	scanner := NewScanner(bundlePath)
	var allAssets []Asset

	assetsDir := filepath.Join(bundlePath, "assets")
	if info, err := os.Stat(assetsDir); err == nil && info.IsDir() {
		assets, err := scanner.ScanAssetsDir(true)
		if err != nil {
			return nil, err
		}
		allAssets = append(allAssets, assets...)
	}

	themesDir := filepath.Join(bundlePath, "themes")
	if info, err := os.Stat(themesDir); err == nil && info.IsDir() {
		themes, err := scanner.ScanThemesDir(true)
		if err != nil {
			return nil, err
		}
		allAssets = append(allAssets, themes...)
	}

	return allAssets, nil
}