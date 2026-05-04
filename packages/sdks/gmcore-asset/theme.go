package gmcore_asset

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type ThemeMode string

const (
	ThemeModeLight    ThemeMode = "light"
	ThemeModeDark     ThemeMode = "dark"
	ThemeModeAuto     ThemeMode = "auto"
)

type Theme struct {
	Name      string `json:"name"`
	Base      string `json:"base,omitempty"`
	Light     string `json:"light,omitempty"`
	Dark      string `json:"dark,omitempty"`
	Variables string `json:"variables,omitempty"`
	Preview   string `json:"preview,omitempty"`
}

func (t Theme) HasVariant(mode ThemeMode) bool {
	switch mode {
	case ThemeModeLight:
		return t.Light != ""
	case ThemeModeDark:
		return t.Dark != ""
	}
	return false
}

type ThemeAssets struct {
	Variables []string `json:"variables,omitempty"`
	Base      []string `json:"base,omitempty"`
	Light     []string `json:"light,omitempty"`
	Dark      []string `json:"dark,omitempty"`
}

type ThemeManager struct {
	mu         sync.RWMutex
	themes     map[string]Theme
	active     string
	mode       ThemeMode
	assetsBase string
}

var defaultThemeManager *ThemeManager
var themeManagerOnce sync.Once

func DefaultThemeManager() *ThemeManager {
	themeManagerOnce.Do(func() {
		defaultThemeManager = NewThemeManager("")
	})
	return defaultThemeManager
}

func NewThemeManager(assetsBase string) *ThemeManager {
	return &ThemeManager{
		themes:     make(map[string]Theme),
		active:     "default",
		mode:       ThemeModeAuto,
		assetsBase: assetsBase,
	}
}

func (tm *ThemeManager) Register(name string, theme Theme) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if theme.Name == "" {
		theme.Name = name
	}
	tm.themes[name] = theme
}

func (tm *ThemeManager) Unregister(name string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.themes, name)
}

func (tm *ThemeManager) Get(name string) (Theme, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	theme, ok := tm.themes[name]
	return theme, ok
}

func (tm *ThemeManager) Active() string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.active
}

func (tm *ThemeManager) SetActive(name string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if _, ok := tm.themes[name]; !ok {
		return fmt.Errorf("theme %q not found", name)
	}
	tm.active = name
	return nil
}

func (tm *ThemeManager) Mode() ThemeMode {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.mode
}

func (tm *ThemeManager) SetMode(mode ThemeMode) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.mode = mode
}

func (tm *ThemeManager) ResolvedMode() ThemeMode {
	mode := tm.Mode()
	if mode != ThemeModeAuto {
		return mode
	}
	return tm.detectSystemMode()
}

func (tm *ThemeManager) detectSystemMode() ThemeMode {
	if os.Getenv("NO_COLOR") != "" {
		return ThemeModeLight
	}
	if os.Getenv("COLOR_SCHEME") == "dark" {
		return ThemeModeDark
	}
	return ThemeModeLight
}

func (tm *ThemeManager) GetThemeAssets(themeName string) (ThemeAssets, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	theme, ok := tm.themes[themeName]
	if !ok {
		return ThemeAssets{}, false
	}
	return ThemeAssets{
		Variables: filterEmpty([]string{theme.Variables}),
		Base:      filterEmpty([]string{theme.Base}),
		Light:     filterEmpty([]string{theme.Light}),
		Dark:      filterEmpty([]string{theme.Dark}),
	}, true
}

func (tm *ThemeManager) List() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	names := make([]string, 0, len(tm.themes))
	for name := range tm.themes {
		names = append(names, name)
	}
	return names
}

func (tm *ThemeManager) SetAssetsBase(base string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.assetsBase = base
}

func (tm *ThemeManager) AssetsBase() string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.assetsBase
}

func (tm *ThemeManager) ThemePath(relative string) string {
	base := tm.AssetsBase()
	if base == "" {
		return relative
	}
	return filepath.Join(base, relative)
}

func (tm *ThemeManager) RenderThemeTags() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	themeName := tm.active
	theme, ok := tm.themes[themeName]
	if !ok {
		return nil
	}
	mode := tm.ResolvedMode()
	var tags []string

	if theme.Variables != "" {
		path := tm.ThemePath(theme.Variables)
		tags = append(tags, fmt.Sprintf(`<link rel="stylesheet" href="%s" data-theme-vars>`, path))
	}

	if theme.Base != "" {
		path := tm.ThemePath(theme.Base)
		tags = append(tags, fmt.Sprintf(`<link rel="stylesheet" href="%s">`, path))
	}

	switch mode {
	case ThemeModeLight:
		if theme.Light != "" {
			path := tm.ThemePath(theme.Light)
			tags = append(tags, fmt.Sprintf(`<link rel="stylesheet" href="%s" data-theme="light">`, path))
		}
	case ThemeModeDark:
		if theme.Dark != "" {
			path := tm.ThemePath(theme.Dark)
			tags = append(tags, fmt.Sprintf(`<link rel="stylesheet" href="%s" data-theme="dark">`, path))
		}
	}

	if mode == ThemeModeAuto && (theme.Light != "" || theme.Dark != "") {
		script := tm.renderThemeSwitchScript(themeName, theme)
		tags = append(tags, script)
	}

	return tags
}

func (tm *ThemeManager) renderThemeSwitchScript(themeName string, theme Theme) string {
	lightHref := ""
	darkHref := ""
	if theme.Light != "" {
		lightHref = tm.ThemePath(theme.Light)
	}
	if theme.Dark != "" {
		darkHref = tm.ThemePath(theme.Dark)
	}
	return fmt.Sprintf(`<script>
(function(){
  var m=matchMedia('(prefers-color-scheme: dark)');
  var s='%s';
  function apply(){
    document.documentElement.setAttribute('data-theme', m.matches?'dark':'light');
    var link=document.querySelector('link[data-theme]');
    if(link) link.href=m.matches?'%s':'%s';
  }
  apply();
  m.addEventListener('change', apply);
})();
</script>`, themeName, darkHref, lightHref)
}

func filterEmpty(ss []string) []string {
	var out []string
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}