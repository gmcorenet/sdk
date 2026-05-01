package gmcoreview

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"sync"

	gmcorei18n "gmcore-i18n"
	gmcoreresolver "gmcore-resolver"
	gmcoretemplating "gmcore-templating"
)

type Config struct {
	AppRoot             string
	SystemRoot          string
	BundleRoots         []string
	RouteURL            func(string, map[string]string) string
	RouteManifest       func() map[string]string
	Translate           func(string) string
	TranslateWithLocale func(string, string) string
	TranslateWithDomain func(string, string, string) string
	TranslateChoice     func(string, string, int, gmcorei18n.Params) string
	Translations        func(string) map[string]string
	TranslationsSelect  func(string, string, []string) map[string]string
	AssetURL            func(string) string
	ShowCRUD            func(context.Context, string, map[string]interface{}) template.HTML
	CSRFToken           func(context.Context, string) string
	OpenAPIJSON         func(context.Context) string
}

type Renderer struct {
	cfg         Config
	funcsReg    *FuncRegistry
}

type FuncRegistry struct {
	mu     sync.RWMutex
	funcs  template.FuncMap
}

func NewFuncRegistry() *FuncRegistry {
	return &FuncRegistry{
		funcs: make(template.FuncMap),
	}
}

func (r *FuncRegistry) Register(name string, fn interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.funcs[name] = fn
}

func (r *FuncRegistry) MergeInto(base template.FuncMap) template.FuncMap {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for name, fn := range r.funcs {
		base[name] = fn
	}
	return base
}

func New(cfg Config) *Renderer {
	return &Renderer{
		cfg:      cfg,
		funcsReg: NewFuncRegistry(),
	}
}

func (r *Renderer) Funcs() *FuncRegistry {
	return r.funcsReg
}

func (r *Renderer) Render(name string, data map[string]interface{}) (string, error) {
	return r.RenderContext(context.Background(), name, data)
}

func (r *Renderer) RenderContext(ctx context.Context, name string, data map[string]interface{}) (string, error) {
	ctx = withAssetRegistry(ctx)
	payload := map[string]interface{}{}
	for key, value := range data {
		payload[key] = value
	}
	currentLocale := "es"
	if raw, ok := payload["Locale"]; ok {
		if value := strings.TrimSpace(fmt.Sprint(raw)); value != "" {
			currentLocale = value
		}
	} else if raw, ok := payload["CurrentLocale"]; ok {
		if value := strings.TrimSpace(fmt.Sprint(raw)); value != "" {
			currentLocale = value
		}
	}
	payload["Locale"] = currentLocale

	safeCtx := contextWithoutCancel(ctx)
	engine := gmcoretemplating.New(gmcoretemplating.Config{
		AppRoot:     r.cfg.AppRoot,
		SystemRoot:  r.cfg.SystemRoot,
		BundleRoots: r.cfg.BundleRoots,
		Funcs:       r.buildFuncs(safeCtx, currentLocale),
	})
	rendered, err := engine.RenderContext(ctx, name, payload)
	if err != nil {
		return "", err
	}
	headHTML := string(renderHeadAssets(ctx))
	if strings.Contains(rendered, headAssetsPlaceholder) {
		rendered = strings.ReplaceAll(rendered, headAssetsPlaceholder, headHTML)
	} else if headHTML != "" && strings.Contains(strings.ToLower(rendered), "</head>") {
		rendered = strings.Replace(rendered, "</head>", headHTML+"</head>", 1)
	}
	return rendered, nil
}

func (r *Renderer) buildFuncs(ctx context.Context, locale string) template.FuncMap {
	funcs := r.funcsReg.MergeInto(make(template.FuncMap))
	r.registerBuiltinFuncs(funcs, ctx, locale)
	return funcs
}

func (r *Renderer) registerBuiltinFuncs(funcs template.FuncMap, ctx context.Context, locale string) {
	funcs["path"] = func(routeName string) string {
		if r.cfg.RouteURL == nil {
			return ""
		}
		return r.cfg.RouteURL(routeName, nil)
	}
	funcs["route"] = funcs["path"]
	funcs["routeParams"] = func(routeName string, params map[string]string) string {
		if r.cfg.RouteURL == nil {
			return ""
		}
		return r.cfg.RouteURL(routeName, params)
	}
	funcs["asset"] = func(path string) string {
		if r.cfg.AssetURL != nil {
			return r.cfg.AssetURL(path)
		}
		return "/" + strings.TrimLeft(strings.TrimSpace(path), "/")
	}
	funcs["routes"] = func() map[string]string {
		if r.cfg.RouteManifest == nil {
			return map[string]string{}
		}
		return r.cfg.RouteManifest()
	}
	funcs["routes_json"] = func() template.JS {
		data, err := json.Marshal(funcs["routes"].(func() map[string]string)())
		if err != nil {
			return template.JS("{}")
		}
		return template.JS(data)
	}
	funcs["route_script"] = func(globalName ...string) template.HTML {
		target := safeJSGlobalTarget("window.gmcoreRoutes", globalName...)
		return template.HTML("<script>" + target + " = " + string(funcs["routes_json"].(func() template.JS)()) + ";</script>")
	}
	funcs["show_crud"] = func(ctx context.Context, name string, extra ...interface{}) template.HTML {
		if r.cfg.ShowCRUD == nil {
			return template.HTML("")
		}
		return r.cfg.ShowCRUD(ctx, name, mergeTemplatePayload(map[string]interface{}{}, extra...))
	}
	funcs["push_head"] = func(ctx context.Context, key string, html string) string {
		RegisterHeadAsset(ctx, key, template.HTML(html))
		return ""
	}
	funcs["render_head_assets"] = func() template.HTML {
		return template.HTML(headAssetsPlaceholder)
	}
	funcs["csrf_token"] = func(ctx context.Context, id ...string) string {
		if r.cfg.CSRFToken == nil {
			return ""
		}
		target := ""
		if len(id) > 0 {
			target = strings.TrimSpace(id[0])
		}
		return r.cfg.CSRFToken(ctx, target)
	}
	funcs["csrf_input"] = func(ctx context.Context, id ...string) template.HTML {
		token := funcs["csrf_token"].(func(context.Context, ...string) string)(ctx, id...)
		if token == "" {
			return template.HTML("")
		}
		return template.HTML(`<input type="hidden" name="_csrf_token" value="` + template.HTMLEscapeString(token) + `">`)
	}
	funcs["openapi_json"] = func(ctx context.Context) template.JS {
		if r.cfg.OpenAPIJSON == nil {
			return template.JS("{}")
		}
		return template.JS(r.cfg.OpenAPIJSON(ctx))
	}
	funcs["openapi_script"] = func(ctx context.Context, globalName ...string) template.HTML {
		target := safeJSGlobalTarget("window.gmcoreOpenAPI", globalName...)
		return template.HTML("<script>" + target + " = " + string(funcs["openapi_json"].(func(context.Context) template.JS)(ctx)) + ";</script>")
	}
	funcs["trans"] = func(key string, extra ...interface{}) string {
		params := mergeI18nParams(extra...)
		domain := ""
		if len(extra) > 1 {
			domain = strings.TrimSpace(fmt.Sprint(extra[1]))
		}
		if domain != "" && r.cfg.TranslateWithDomain != nil {
			return interpolateTranslatedValue(r.cfg.TranslateWithDomain(locale, domain, key), params)
		}
		if r.cfg.TranslateWithLocale != nil {
			return interpolateTranslatedValue(r.cfg.TranslateWithLocale(locale, key), params)
		}
		if r.cfg.Translate == nil {
			return interpolateTranslatedValue(key, params)
		}
		return interpolateTranslatedValue(r.cfg.Translate(key), params)
	}
	funcs["trans_locale"] = func(locale, key string, extra ...interface{}) string {
		params := mergeI18nParams(extra...)
		domain := ""
		if len(extra) > 1 {
			domain = strings.TrimSpace(fmt.Sprint(extra[1]))
		}
		if domain != "" && r.cfg.TranslateWithDomain != nil {
			return interpolateTranslatedValue(r.cfg.TranslateWithDomain(locale, domain, key), params)
		}
		if r.cfg.TranslateWithLocale != nil {
			return interpolateTranslatedValue(r.cfg.TranslateWithLocale(locale, key), params)
		}
		if r.cfg.Translate == nil {
			return interpolateTranslatedValue(key, params)
		}
		return interpolateTranslatedValue(r.cfg.Translate(key), params)
	}
	funcs["trans_choice"] = func(key string, count int, extra ...interface{}) string {
		params := mergeI18nParams(extra...)
		params["count"] = count
		if r.cfg.TranslateChoice != nil {
			return r.cfg.TranslateChoice(locale, key, count, params)
		}
		if r.cfg.TranslateWithLocale != nil {
			return interpolateTranslatedValue(r.cfg.TranslateWithLocale(locale, key), params)
		}
		if r.cfg.Translate == nil {
			return interpolateTranslatedValue(key, params)
		}
		return interpolateTranslatedValue(r.cfg.Translate(key), params)
	}
	funcs["translations"] = func(localeOverride ...string) map[string]string {
		targetLocale := locale
		targetDomain := ""
		prefixes := []string{}
		if len(localeOverride) > 0 && strings.TrimSpace(localeOverride[0]) != "" {
			targetLocale = strings.TrimSpace(localeOverride[0])
		}
		if len(localeOverride) > 1 && strings.TrimSpace(localeOverride[1]) != "" {
			targetDomain = strings.TrimSpace(localeOverride[1])
		}
		if len(localeOverride) > 2 {
			prefixes = append(prefixes, localeOverride[2:]...)
		}
		if r.cfg.TranslationsSelect != nil {
			return r.cfg.TranslationsSelect(targetLocale, targetDomain, prefixes)
		}
		if r.cfg.Translations == nil {
			return map[string]string{}
		}
		return r.cfg.Translations(targetLocale)
	}
	funcs["translations_json"] = func(localeOverride ...string) template.JS {
		targetLocale := locale
		targetDomain := ""
		prefixes := []string{}
		if len(localeOverride) > 0 && strings.TrimSpace(localeOverride[0]) != "" {
			targetLocale = strings.TrimSpace(localeOverride[0])
		}
		if len(localeOverride) > 1 && strings.TrimSpace(localeOverride[1]) != "" {
			targetDomain = strings.TrimSpace(localeOverride[1])
		}
		if len(localeOverride) > 2 {
			prefixes = append(prefixes, localeOverride[2:]...)
		}
		var payload map[string]string
		if r.cfg.TranslationsSelect != nil {
			payload = r.cfg.TranslationsSelect(targetLocale, targetDomain, prefixes)
		} else if r.cfg.Translations != nil {
			payload = r.cfg.Translations(targetLocale)
		}
		if payload == nil {
			payload = map[string]string{}
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return template.JS("{}")
		}
		return template.JS(data)
	}
	funcs["translations_script"] = func(globalName ...string) template.HTML {
		target := safeJSGlobalTarget("window.gmcoreTranslations", globalName...)
		return template.HTML("<script>" + target + " = " + string(funcs["translations_json"].(func(...string) template.JS)()) + ";</script>")
	}
}

func contextWithoutCancel(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	type contextKey struct{}
	return context.WithValue(ctx, contextKey{}, struct{}{})
}

func mergeTemplatePayload(base map[string]interface{}, extra ...interface{}) map[string]interface{} {
	merged := map[string]interface{}{}
	for key, value := range base {
		merged[key] = value
	}
	if len(extra) == 0 || extra[0] == nil {
		return merged
	}
	switch current := extra[0].(type) {
	case map[string]interface{}:
		for key, value := range current {
			merged[key] = value
		}
	case map[string]string:
		for key, value := range current {
			merged[key] = value
		}
	}
	return merged
}

func mergeI18nParams(extra ...interface{}) gmcorei18n.Params {
	params := gmcorei18n.Params{}
	if len(extra) == 0 || extra[0] == nil {
		return params
	}
	switch current := extra[0].(type) {
	case gmcorei18n.Params:
		for key, value := range current {
			params[key] = value
		}
	case map[string]interface{}:
		for key, value := range current {
			params[key] = value
		}
	case map[string]string:
		for key, value := range current {
			params[key] = value
		}
	}
	return params
}

func interpolateTranslatedValue(value string, params gmcorei18n.Params) string {
	if len(params) == 0 {
		return value
	}
	return gmcorei18n.Params(params).Interpolate(value)
}

var jsGlobalTargetPattern = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*(\.[A-Za-z_$][A-Za-z0-9_$]*)*$`)

func safeJSGlobalTarget(fallback string, values ...string) string {
	if len(values) == 0 {
		return fallback
	}
	current := strings.TrimSpace(values[0])
	if current == "" || !jsGlobalTargetPattern.MatchString(current) {
		return fallback
	}
	return current
}

func ResolveTemplate(cfg Config, name string) (gmcoreresolver.ResolvedFile, bool) {
	return gmcoretemplating.ResolveTemplate(gmcoretemplating.Config{
		AppRoot:     cfg.AppRoot,
		SystemRoot:  cfg.SystemRoot,
		BundleRoots: cfg.BundleRoots,
	}, name)
}
