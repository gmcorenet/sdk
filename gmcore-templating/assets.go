package gmcoretemplating

type AssetHelper struct {
	CSSFunc       func() string
	JSFunc        func() string
	ThemeFunc     func() string
	AllFunc       func() string
	BodyEndFunc   func() string
	ResolveURLFunc func(assetType, path string) string
}

func NewAssetHelper() *AssetHelper {
	return &AssetHelper{}
}

func (h *AssetHelper) CSS() string {
	if h.CSSFunc != nil {
		return h.CSSFunc()
	}
	return ""
}

func (h *AssetHelper) JS() string {
	if h.JSFunc != nil {
		return h.JSFunc()
	}
	return ""
}

func (h *AssetHelper) Theme() string {
	if h.ThemeFunc != nil {
		return h.ThemeFunc()
	}
	return ""
}

func (h *AssetHelper) All() string {
	if h.AllFunc != nil {
		return h.AllFunc()
	}
	return ""
}

func (h *AssetHelper) BodyEnd() string {
	if h.BodyEndFunc != nil {
		return h.BodyEndFunc()
	}
	return ""
}

func (h *AssetHelper) ResolveURL(assetType, path string) string {
	if h.ResolveURLFunc != nil {
		return h.ResolveURLFunc(assetType, path)
	}
	return path
}

type ThemeHelper struct {
	ModeFunc      func() string
	ActiveFunc    func() string
	SetModeFunc   func(mode string)
	SetActiveFunc func(name string) error
	ListFunc      func() []string
	TagsFunc      func() []string
	IsDarkFunc    func() bool
	IsLightFunc   func() bool
	IsAutoFunc    func() bool
}

func NewThemeHelper() *ThemeHelper {
	return &ThemeHelper{}
}

func (h *ThemeHelper) Mode() string {
	if h.ModeFunc != nil {
		return h.ModeFunc()
	}
	return "auto"
}

func (h *ThemeHelper) Active() string {
	if h.ActiveFunc != nil {
		return h.ActiveFunc()
	}
	return ""
}

func (h *ThemeHelper) SetMode(mode string) {
	if h.SetModeFunc != nil {
		h.SetModeFunc(mode)
	}
}

func (h *ThemeHelper) SetActive(name string) error {
	if h.SetActiveFunc != nil {
		return h.SetActiveFunc(name)
	}
	return nil
}

func (h *ThemeHelper) List() []string {
	if h.ListFunc != nil {
		return h.ListFunc()
	}
	return nil
}

func (h *ThemeHelper) Tags() []string {
	if h.TagsFunc != nil {
		return h.TagsFunc()
	}
	return nil
}

func (h *ThemeHelper) IsDark() bool {
	if h.IsDarkFunc != nil {
		return h.IsDarkFunc()
	}
	return false
}

func (h *ThemeHelper) IsLight() bool {
	if h.IsLightFunc != nil {
		return h.IsLightFunc()
	}
	return true
}

func (h *ThemeHelper) IsAuto() bool {
	if h.IsAutoFunc != nil {
		return h.IsAutoFunc()
	}
	return true
}

type AssetFuncs func() map[string]interface{}

func AssetHelperFuncMap(helper *AssetHelper) map[string]interface{} {
	return map[string]interface{}{
		"assets": func() *AssetHelper {
			return helper
		},
		"asset_css": func() string {
			return helper.CSS()
		},
		"asset_js": func() string {
			return helper.JS()
		},
		"asset_theme": func() string {
			return helper.Theme()
		},
		"asset_all": func() string {
			return helper.All()
		},
		"asset_body_end": func() string {
			return helper.BodyEnd()
		},
	}
}

func ThemeHelperFuncMap(helper *ThemeHelper) map[string]interface{} {
	return map[string]interface{}{
		"theme": func() *ThemeHelper {
			return helper
		},
		"theme_mode": func() string {
			return helper.Mode()
		},
		"theme_active": func() string {
			return helper.Active()
		},
		"is_dark": func() bool {
			return helper.IsDark()
		},
		"is_light": func() bool {
			return helper.IsLight()
		},
	}
}

func MergeFuncMaps(funcMaps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, fm := range funcMaps {
		for k, v := range fm {
			result[k] = v
		}
	}
	return result
}