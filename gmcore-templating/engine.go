package gmcore_templating

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var extensionRegistry = map[string]interface{}{}
var extMu sync.RWMutex

func RegisterFunc(name string, fn interface{}) {
	extMu.Lock()
	defer extMu.Unlock()
	extensionRegistry[name] = fn
}

func GetFuncs() template.FuncMap {
	extMu.RLock()
	defer extMu.RUnlock()
	fm := template.FuncMap{}
	for k, v := range extensionRegistry {
		fm[k] = v
	}
	return fm
}

type Config struct {
	AppRoot      string
	SystemRoot   string
	BundleRoots  []string
	Funcs        template.FuncMap
	DisableCache bool
	Mode         string
}

type Engine struct {
	cfg   Config
	mu    sync.RWMutex
	cache parsedCache
}

type parsedCache struct {
	files      map[string]string
	signatures map[string]int64
	twig       map[string]cachedTwigSource
}

type resolvedTwigSource struct {
	Body         string
	Macros       map[string]twigMacro
	Chain        []string
	Dependencies []string
}

type cachedTwigSource struct {
	Source     resolvedTwigSource
	Signatures map[string]int64
}

const (
	ModeDev  = "dev"
	ModeProd = "prod"
)

func New(cfg Config) *Engine {
	return &Engine{cfg: cfg}
}

func (e *Engine) RenderContext(ctx context.Context, name string, data map[string]interface{}) (string, error) {
	name = normalizeTemplateName(name)
	if name == "" {
		return "", errors.New("missing template name")
	}
	payload := clonePayload(data)
	sourcePath, ok := e.templatePath(name)
	if !ok {
		return "", fmt.Errorf("template not found: %s", name)
	}
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}
	if hasTwigSyntax(string(source)) {
		return e.renderTwig(ctx, name, payload)
	}
	return e.renderLegacy(ctx, name, payload)
}

func (e *Engine) TemplateExists(name string) bool {
	_, ok := e.templatePath(name)
	return ok
}

type resolvedFileStub struct {
	Path   string
	Source string
}

func (e *Engine) Resolve(name string) (resolvedFileStub, bool) {
	name = normalizeTemplateName(name)
	if name == "" {
		return resolvedFileStub{}, false
	}
	path, found := e.templatePath(name)
	if !found {
		return resolvedFileStub{}, false
	}
	source := e.determineSource(path)
	return resolvedFileStub{Path: path, Source: source}, true
}

func (e *Engine) determineSource(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	if e.cfg.AppRoot != "" {
		appRoot, _ := filepath.Abs(e.cfg.AppRoot)
		if strings.HasPrefix(absPath, appRoot) {
			return "app"
		}
	}
	if e.cfg.SystemRoot != "" {
		systemRoot, _ := filepath.Abs(e.cfg.SystemRoot)
		if strings.HasPrefix(absPath, systemRoot) {
			return "system"
		}
	}
	for _, bundleRoot := range e.cfg.BundleRoots {
		if bundleRoot == "" {
			continue
		}
		br, _ := filepath.Abs(bundleRoot)
		if strings.HasPrefix(absPath, br) {
			rel, _ := filepath.Rel(br, absPath)
			parts := strings.Split(rel, string(filepath.Separator))
			if len(parts) > 0 {
				return parts[0]
			}
		}
	}
	return "unknown"
}

func (e *Engine) renderLegacy(ctx context.Context, name string, payload map[string]interface{}) (string, error) {
	files, err := e.templateFiles()
	if err != nil {
		return "", err
	}
	funcs := e.baseFuncs(ctx, payload, nil)
	tpl := template.New("").Funcs(funcs)
	funcs["templateExists"] = func(name string) bool {
		return e.lookupTemplate(tpl, name) != ""
	}
	funcs["include"] = func(name string, value interface{}) template.HTML {
		target := e.lookupTemplate(tpl, name)
		if target == "" {
			return template.HTML("")
		}
		var buf bytes.Buffer
		if err := tpl.ExecuteTemplate(&buf, target, value); err != nil {
			return template.HTML("")
		}
		return template.HTML(buf.String())
	}
	tpl = tpl.Funcs(funcs)

	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, rel := range keys {
		content, err := os.ReadFile(files[rel])
		if err != nil {
			return "", err
		}
		if _, err := tpl.New(rel).Parse(string(content)); err != nil {
			return "", fmt.Errorf("parse %s: %w", rel, err)
		}
		base := filepath.Base(rel)
		if base != rel && tpl.Lookup(base) == nil {
			if _, err := tpl.New(base).Parse(`{{template "` + rel + `" .}}`); err != nil {
				return "", err
			}
		}
	}

	target := e.lookupTemplate(tpl, name)
	if target == "" {
		return "", fmt.Errorf("template not found: %s", name)
	}
	resolved, ok := e.Resolve(name)
	if ok {
		_ = resolved
	}
	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, target, payload); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (e *Engine) renderTwig(ctx context.Context, name string, payload map[string]interface{}) (string, error) {
	source, err := e.resolveTwigSource(name, nil, nil, map[string]bool{})
	if err != nil {
		return "", err
	}
	funcs := e.baseFuncs(ctx, payload, nil)
	tpl := template.New(name).Funcs(funcs)
	funcs["templateExists"] = func(name string) bool {
		return e.TemplateExists(name)
	}
	funcs["include"] = func(name string, value interface{}) template.HTML {
		return e.renderIncludedTemplate(ctx, name, payload, value, false)
	}
	funcs["includeOnly"] = func(name string, value interface{}) template.HTML {
		return e.renderIncludedTemplate(ctx, name, payload, value, true)
	}
	funcs["includeMissing"] = func(name string, value interface{}) template.HTML {
		if !e.TemplateExists(name) {
			return template.HTML("")
		}
		return e.renderIncludedTemplate(ctx, name, payload, value, false)
	}
	funcs["embed"] = funcs["include"]
	funcs["twigWith"] = func(current interface{}, expr string) interface{} {
		value := evaluateTwigExpression(current, expr, funcs)
		if mapped, ok := value.(map[string]interface{}); ok {
			return mergePayload(payload, mapped)
		}
		return mergeTemplatePayload(payload, value)
	}
	funcs["twigOnlyWith"] = func(current interface{}, expr string) interface{} {
		value := evaluateTwigExpression(current, expr, funcs)
		if mapped, ok := value.(map[string]interface{}); ok {
			return clonePayload(mapped)
		}
		return mergeTemplatePayload(nil, value)
	}
	funcs["twigPrint"] = func(current interface{}, expr string) interface{} {
		return evaluateTwigExpression(current, expr, funcs)
	}
	funcs["twigIf"] = func(current interface{}, expr string) bool {
		return truthy(evaluateTwigExpression(current, expr, funcs))
	}
	funcs["twigIter"] = func(current interface{}, expr string) []interface{} {
		return toSlice(evaluateTwigExpression(current, expr, funcs))
	}
	funcs["twigIterKV"] = func(current interface{}, expr string) map[string]interface{} {
		out := map[string]interface{}{}
		value := evaluateTwigExpression(current, expr, funcs)
		rv := reflect.ValueOf(value)
		for rv.IsValid() && (rv.Kind() == reflect.Interface || rv.Kind() == reflect.Pointer) {
			if rv.IsNil() {
				return out
			}
			rv = rv.Elem()
		}
		if !rv.IsValid() || rv.Kind() != reflect.Map {
			return out
		}
		for _, key := range rv.MapKeys() {
			out[fmt.Sprint(key.Interface())] = rv.MapIndex(key).Interface()
		}
		return out
	}
	funcs["twigSet"] = func(current interface{}, key string, expr string) string {
		target, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		target[strings.TrimSpace(key)] = evaluateTwigExpression(current, expr, funcs)
		return ""
	}
	funcs["twigCaptureSet"] = func(current interface{}, key string, body string) string {
		target, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		target[strings.TrimSpace(key)] = strings.TrimSpace(fmt.Sprint(callTemplateFunc(e.buildTwigRenderBlockFunc(name, funcs), current, body, "")))
		return ""
	}
	funcs["twigRenderBlock"] = e.buildTwigRenderBlockFunc(name, funcs)
	macroNames := registerImportedTwigMacros(funcs, source.Macros, func(macro twigMacro) func(...interface{}) template.HTML {
		return e.buildTwigMacroFunc(&tpl, macro)
	})
	tpl = tpl.Funcs(funcs)
	for _, macroName := range macroNames {
		if _, err := tpl.Parse(buildTwigMacroTemplate(source.Macros[macroName])); err != nil {
			return "", fmt.Errorf("parse macro %s: %w", macroName, err)
		}
	}
	if _, err := tpl.Parse(`{{define "` + name + `"}}` + source.Body + `{{end}}`); err != nil {
		return "", fmt.Errorf("parse %s (chain: %s): %w", name, strings.Join(source.Chain, " -> "), err)
	}
	resolved, ok := e.Resolve(name)
	if ok {
		_ = resolved
	}
	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, name, payload); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (e *Engine) resolveTwigSource(name string, overrides map[string]string, macros map[string]twigMacro, stack map[string]bool) (resolvedTwigSource, error) {
	name = normalizeTemplateName(name)
	if len(overrides) == 0 && len(macros) == 0 {
		if cached, ok := e.cachedTwigSource(name); ok {
			return cached, nil
		}
	}
	if stack[name] {
		return resolvedTwigSource{}, fmt.Errorf("cyclic template inheritance detected for %s", name)
	}
	stack[name] = true
	defer delete(stack, name)

	sourcePath, ok := e.templatePath(name)
	if !ok {
		return resolvedTwigSource{}, fmt.Errorf("template not found: %s", name)
	}
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		return resolvedTwigSource{}, err
	}
	expanded, embeddedDependencies, err := e.expandTwigEmbeds(string(raw), stack)
	if err != nil {
		return resolvedTwigSource{}, fmt.Errorf("expand embeds %s: %w", name, err)
	}
	parsed, err := parseTwigTemplate(expanded)
	if err != nil {
		return resolvedTwigSource{}, fmt.Errorf("parse twig %s: %w", name, err)
	}
	currentOverrides := copyOverrides(overrides)
	currentMacros := copyTwigMacros(macros)
	for blockName, blockContent := range parsed.Blocks {
		if inherited, ok := currentOverrides[blockName]; ok {
			next := strings.ReplaceAll(inherited, "{{ parent() }}", blockContent)
			next = strings.ReplaceAll(next, "{{parent()}}", blockContent)
			currentOverrides[blockName] = next
			continue
		}
		currentOverrides[blockName] = blockContent
	}
	for macroName, macro := range parsed.Macros {
		if _, exists := currentMacros[macroName]; exists {
			continue
		}
		currentMacros[macroName] = macro
	}
	importedMacros, importDependencies, err := e.resolveImportedTwigMacros(parsed.Imports, stack)
	if err != nil {
		return resolvedTwigSource{}, err
	}
	for macroName, macro := range importedMacros {
		if _, exists := currentMacros[macroName]; exists {
			continue
		}
		currentMacros[macroName] = macro
	}
	if parsed.Extends != "" {
		resolved, err := e.resolveTwigSource(parsed.Extends, currentOverrides, currentMacros, stack)
		if err != nil {
			return resolvedTwigSource{}, err
		}
		resolved.Chain = append([]string{name}, resolved.Chain...)
		resolved.Dependencies = mergeTwigDependencies(
			[]string{sourcePath},
			embeddedDependencies,
			importDependencies,
			resolved.Dependencies,
		)
		if len(overrides) == 0 && len(macros) == 0 {
			e.storeTwigSource(name, resolved)
		}
		return resolved, nil
	}
	resolved := resolvedTwigSource{
		Body:         renderTwigBody(parsed.Body, parsed.Blocks, currentOverrides),
		Macros:       currentMacros,
		Chain:        []string{name},
		Dependencies: mergeTwigDependencies([]string{sourcePath}, embeddedDependencies, importDependencies),
	}
	if len(overrides) == 0 && len(macros) == 0 {
		e.storeTwigSource(name, resolved)
	}
	return resolved, nil
}

func (e *Engine) resolveImportedTwigMacros(imports map[string]string, stack map[string]bool) (map[string]twigMacro, []string, error) {
	out := map[string]twigMacro{}
	dependencies := []string{}
	for alias, templateName := range imports {
		specificMacro := ""
		if strings.Contains(templateName, "#") {
			left, right, _ := strings.Cut(templateName, "#")
			templateName = left
			specificMacro = strings.TrimSpace(right)
		}
		sourcePath, ok := e.templatePath(templateName)
		if !ok {
			return nil, nil, fmt.Errorf("import template not found: %s", templateName)
		}
		raw, err := os.ReadFile(sourcePath)
		if err != nil {
			return nil, nil, err
		}
		expanded, embeddedDependencies, err := e.expandTwigEmbeds(string(raw), stack)
		if err != nil {
			return nil, nil, err
		}
		parsed, err := parseTwigTemplate(expanded)
		if err != nil {
			return nil, nil, fmt.Errorf("parse imported twig %s: %w", templateName, err)
		}
		dependencies = append(dependencies, sourcePath)
		dependencies = append(dependencies, embeddedDependencies...)
		for macroName, macro := range parsed.Macros {
			if specificMacro != "" && macroName != specificMacro {
				continue
			}
			macro.Name = alias + "." + macroName
			key := alias + "." + macroName
			if specificMacro != "" {
				macro.Name = alias
				key = alias
			}
			out[key] = macro
		}
		if len(parsed.Imports) > 0 {
			nested, nestedDependencies, err := e.resolveImportedTwigMacros(parsed.Imports, stack)
			if err != nil {
				return nil, nil, err
			}
			for key, value := range nested {
				out[key] = value
			}
			dependencies = append(dependencies, nestedDependencies...)
		}
	}
	return out, mergeTwigDependencies(dependencies), nil
}

func (e *Engine) buildTwigMacroFunc(tpl **template.Template, macro twigMacro) func(...interface{}) template.HTML {
	return func(args ...interface{}) template.HTML {
		context := map[string]interface{}{}
		for idx, name := range macro.Args {
			if idx < len(args) {
				context[name] = args[idx]
				continue
			}
			context[name] = nil
		}
		var buf bytes.Buffer
		if *tpl == nil {
			return ""
		}
		if err := (*tpl).ExecuteTemplate(&buf, twigMacroTemplateName(macro.Name), context); err != nil {
			return ""
		}
		return template.HTML(buf.String())
	}
}

func buildTwigMacroTemplate(macro twigMacro) string {
	return `{{define "` + twigMacroTemplateName(macro.Name) + `"}}` + macro.Body + `{{end}}`
}

func twigMacroTemplateName(name string) string {
	return "__gmcore_macro_" + strings.TrimSpace(name)
}

func (e *Engine) baseFuncs(ctx context.Context, payload map[string]interface{}, extra template.FuncMap) template.FuncMap {
	funcs := template.FuncMap{}
	for key, value := range e.cfg.Funcs {
		funcs[key] = value
	}
	for key, value := range extra {
		funcs[key] = value
	}
	for key, value := range GetFuncs() {
		if _, exists := funcs[key]; !exists {
			funcs[key] = value
		}
	}
	funcs["default"] = func(value interface{}, fallback interface{}) interface{} {
		if value == nil {
			return fallback
		}
		if current, ok := value.(string); ok && strings.TrimSpace(current) == "" {
			return fallback
		}
		return value
	}
	funcs["dict"] = func(values ...interface{}) map[string]interface{} {
		out := map[string]interface{}{}
		for i := 0; i+1 < len(values); i += 2 {
			out[fmt.Sprint(values[i])] = values[i+1]
		}
		return out
	}
	funcs["list"] = func(values ...interface{}) []interface{} { return values }
	funcs["join"] = func(values []string, sep string) string { return strings.Join(values, sep) }
	funcs["split"] = func(value, sep string) []string { return strings.Split(value, sep) }
	funcs["lower"] = strings.ToLower
	funcs["upper"] = strings.ToUpper
	funcs["title"] = strings.Title
	funcs["trim"] = strings.TrimSpace
	funcs["replace"] = strings.ReplaceAll
	funcs["contains"] = strings.Contains
	funcs["hasPrefix"] = strings.HasPrefix
	funcs["hasSuffix"] = strings.HasSuffix
	funcs["slug"] = func(value string) string {
		value = strings.ToLower(strings.TrimSpace(value))
		value = strings.ReplaceAll(value, " ", "-")
		value = strings.ReplaceAll(value, "/", "-")
		value = strings.ReplaceAll(value, "_", "-")
		return value
	}
	funcs["coalesce"] = func(values ...interface{}) interface{} {
		for _, value := range values {
			if !isEmptyValue(value) {
				return value
			}
		}
		return nil
	}
	funcs["empty"] = isEmptyValue
	funcs["len"] = func(value interface{}) int { return collectionLen(value) }
	funcs["keys"] = func(value interface{}) []string { return mapKeys(value) }
	funcs["values"] = func(value interface{}) []interface{} { return mapValues(value) }
	funcs["first"] = func(value interface{}) interface{} { return firstValue(value) }
	funcs["last"] = func(value interface{}) interface{} { return lastValue(value) }
	funcs["merge"] = func(values ...interface{}) map[string]interface{} {
		merged := map[string]interface{}{}
		for _, current := range values {
			switch typed := current.(type) {
			case map[string]interface{}:
				for key, value := range typed {
					merged[key] = value
				}
			case map[string]string:
				for key, value := range typed {
					merged[key] = value
				}
			}
		}
		return merged
	}
	funcs["json"] = func(value interface{}) string {
		data, err := json.Marshal(value)
		if err != nil {
			return ""
		}
		return string(data)
	}
	funcs["safeHTML"] = func(value interface{}) template.HTML {
		return template.HTML(fmt.Sprint(value))
	}
	funcs["nl2br"] = func(value interface{}) template.HTML {
		escaped := template.HTMLEscapeString(fmt.Sprint(value))
		return template.HTML(strings.ReplaceAll(escaped, "\n", "<br>"))
	}
	funcs["date"] = func(value interface{}, layout ...string) string {
		parsed, ok := normalizeTime(value)
		if !ok {
			return ""
		}
		targetLayout := time.RFC3339
		if len(layout) > 0 && strings.TrimSpace(layout[0]) != "" {
			targetLayout = layout[0]
		}
		return parsed.Format(targetLayout)
	}
	funcs["dateHuman"] = func(value interface{}) string {
		parsed, ok := normalizeTime(value)
		if !ok {
			return ""
		}
		return parsed.Format("2006-01-02 15:04")
	}
	funcs["templateExists"] = func(name string) bool {
		return e.TemplateExists(name)
	}
	funcs["include"] = func(name string, value interface{}) template.HTML {
		return e.renderIncludedTemplate(ctx, name, payload, value, false)
	}
	funcs["includeOnly"] = func(name string, value interface{}) template.HTML {
		return e.renderIncludedTemplate(ctx, name, payload, value, true)
	}
	funcs["includeMissing"] = func(name string, value interface{}) template.HTML {
		if !e.TemplateExists(name) {
			return template.HTML("")
		}
		return e.renderIncludedTemplate(ctx, name, payload, value, false)
	}
	funcs["component"] = funcs["include"]
	return funcs
}

func isEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}
	switch current := value.(type) {
	case string:
		return strings.TrimSpace(current) == ""
	case []string:
		return len(current) == 0
	case []interface{}:
		return len(current) == 0
	case map[string]interface{}:
		return len(current) == 0
	case map[string]string:
		return len(current) == 0
	case bool:
		return !current
	}
	current := reflect.ValueOf(value)
	switch current.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
		return current.Len() == 0
	case reflect.Bool:
		return !current.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return current.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return current.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return current.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return current.IsNil()
	}
	return false
}

func collectionLen(value interface{}) int {
	if value == nil {
		return 0
	}
	current := reflect.ValueOf(value)
	switch current.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return current.Len()
	default:
		return 0
	}
}

func mapKeys(value interface{}) []string {
	out := []string{}
	current := reflect.ValueOf(value)
	if !current.IsValid() || current.Kind() != reflect.Map {
		return out
	}
	for _, key := range current.MapKeys() {
		out = append(out, fmt.Sprint(key.Interface()))
	}
	sort.Strings(out)
	return out
}

func mapValues(value interface{}) []interface{} {
	out := []interface{}{}
	current := reflect.ValueOf(value)
	if !current.IsValid() || current.Kind() != reflect.Map {
		return out
	}
	keys := current.MapKeys()
	sort.SliceStable(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
	})
	for _, key := range keys {
		out = append(out, current.MapIndex(key).Interface())
	}
	return out
}

func firstValue(value interface{}) interface{} {
	current := reflect.ValueOf(value)
	if !current.IsValid() {
		return nil
	}
	switch current.Kind() {
	case reflect.Array, reflect.Slice:
		if current.Len() == 0 {
			return nil
		}
		return current.Index(0).Interface()
	case reflect.String:
		if current.Len() == 0 {
			return ""
		}
		return string([]rune(current.String())[0])
	}
	return nil
}

func lastValue(value interface{}) interface{} {
	current := reflect.ValueOf(value)
	if !current.IsValid() {
		return nil
	}
	switch current.Kind() {
	case reflect.Array, reflect.Slice:
		if current.Len() == 0 {
			return nil
		}
		return current.Index(current.Len() - 1).Interface()
	case reflect.String:
		runes := []rune(current.String())
		if len(runes) == 0 {
			return ""
		}
		return string(runes[len(runes)-1])
	}
	return nil
}

func normalizeTime(value interface{}) (time.Time, bool) {
	switch current := value.(type) {
	case time.Time:
		return current, true
	case *time.Time:
		if current == nil {
			return time.Time{}, false
		}
		return *current, true
	case int64:
		return time.Unix(current, 0), true
	case int:
		return time.Unix(int64(current), 0), true
	case float64:
		return time.Unix(int64(current), 0), true
	case string:
		current = strings.TrimSpace(current)
		if current == "" {
			return time.Time{}, false
		}
		if parsed, err := time.Parse(time.RFC3339, current); err == nil {
			return parsed, true
		}
		if parsed, err := time.Parse("2006-01-02 15:04:05", current); err == nil {
			return parsed, true
		}
		if parsed, err := time.Parse("2006-01-02", current); err == nil {
			return parsed, true
		}
		if unix, err := strconv.ParseInt(current, 10, 64); err == nil {
			return time.Unix(unix, 0), true
		}
	}
	return time.Time{}, false
}

func (e *Engine) lookupTemplate(tpl *template.Template, name string) string {
	name = normalizeTemplateName(name)
	if name == "" {
		return ""
	}
	if tpl.Lookup(name) != nil {
		return name
	}
	base := filepath.Base(name)
	if tpl.Lookup(base) != nil {
		return base
	}
	return ""
}

func (e *Engine) templatePath(name string) (string, bool) {
	name = normalizeTemplateName(name)
	if name == "" {
		return "", false
	}
	files, err := e.templateFiles()
	if err != nil {
		return "", false
	}
	if path, ok := files[name]; ok {
		return path, true
	}
	base := filepath.Base(name)
	for rel, path := range files {
		if filepath.Base(rel) == base {
			return path, true
		}
	}
	return "", false
}

func (e *Engine) templateFiles() (map[string]string, error) {
	e.mu.RLock()
	if !e.cacheDisabled() && len(e.cache.files) > 0 && e.cacheValidLocked() {
		files := make(map[string]string, len(e.cache.files))
		for k, v := range e.cache.files {
			files[k] = v
		}
		e.mu.RUnlock()
		return files, nil
	}
	e.mu.RUnlock()

	files := map[string]string{}
	signatures := map[string]int64{}
	roots := make([]string, 0, 2+len(e.cfg.BundleRoots))
	if strings.TrimSpace(e.cfg.SystemRoot) != "" {
		roots = append(roots, filepath.Join(e.cfg.SystemRoot, "templates"))
	}
	for _, bundleRoot := range e.cfg.BundleRoots {
		bundleRoot = strings.TrimSpace(bundleRoot)
		if bundleRoot != "" {
			roots = append(roots, filepath.Join(bundleRoot, "templates"))
		}
	}
	if strings.TrimSpace(e.cfg.AppRoot) != "" {
		roots = append(roots, filepath.Join(e.cfg.AppRoot, "templates"))
	}
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		err = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files[filepath.ToSlash(rel)] = path
			signatures[path] = info.ModTime().UnixNano()
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	e.mu.Lock()
	if e.cache.twig == nil {
		e.cache.twig = map[string]cachedTwigSource{}
	}
	e.cache.files = files
	e.cache.signatures = signatures
	e.mu.Unlock()
	return files, nil
}

func (e *Engine) cacheDisabled() bool {
	if e == nil {
		return true
	}
	if e.cfg.DisableCache {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(e.cfg.Mode), ModeDev)
}

func (e *Engine) cacheValidLocked() bool {
	for path, signature := range e.cache.signatures {
		info, err := os.Stat(path)
		if err != nil || info.ModTime().UnixNano() != signature {
			return false
		}
	}
	return true
}

func (e *Engine) cachedTwigSource(name string) (resolvedTwigSource, bool) {
	if e == nil || e.cacheDisabled() {
		return resolvedTwigSource{}, false
	}
	e.mu.RLock()
	entry, ok := e.cache.twig[normalizeTemplateName(name)]
	e.mu.RUnlock()
	if !ok || !twigSourceCacheValid(entry) {
		return resolvedTwigSource{}, false
	}
	return entry.Source, true
}

func (e *Engine) storeTwigSource(name string, source resolvedTwigSource) {
	if e == nil || e.cacheDisabled() {
		return
	}
	signatures := map[string]int64{}
	for _, path := range mergeTwigDependencies(source.Dependencies) {
		info, err := os.Stat(path)
		if err != nil {
			return
		}
		signatures[path] = info.ModTime().UnixNano()
	}
	e.mu.Lock()
	if e.cache.twig == nil {
		e.cache.twig = map[string]cachedTwigSource{}
	}
	e.cache.twig[normalizeTemplateName(name)] = cachedTwigSource{Source: source, Signatures: signatures}
	e.mu.Unlock()
}

func twigSourceCacheValid(entry cachedTwigSource) bool {
	if len(entry.Signatures) == 0 {
		return false
	}
	for path, signature := range entry.Signatures {
		info, err := os.Stat(path)
		if err != nil || info.ModTime().UnixNano() != signature {
			return false
		}
	}
	return true
}

func mergeTwigDependencies(groups ...[]string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, group := range groups {
		for _, path := range group {
			path = strings.TrimSpace(path)
			if path == "" || seen[path] {
				continue
			}
			seen[path] = true
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out
}

func normalizeTemplateName(name string) string {
	return strings.Trim(strings.TrimSpace(name), "/")
}

func clonePayload(data map[string]interface{}) map[string]interface{} {
	payload := map[string]interface{}{}
	for key, value := range data {
		payload[key] = value
	}
	return payload
}

func mergePayload(base map[string]interface{}, extra map[string]interface{}) map[string]interface{} {
	merged := clonePayload(base)
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func mergeTemplatePayload(base map[string]interface{}, extra interface{}) map[string]interface{} {
	if extra == nil {
		return clonePayload(base)
	}
	merged := clonePayload(base)
	switch current := extra.(type) {
	case map[string]interface{}:
		for key, value := range current {
			merged[key] = value
		}
	case map[string]string:
		for key, value := range current {
			merged[key] = value
		}
	default:
		rv := reflect.ValueOf(extra)
		if rv.Kind() == reflect.Map {
			iter := rv.MapRange()
			for iter.Next() {
				merged[fmt.Sprint(iter.Key().Interface())] = iter.Value().Interface()
			}
		}
	}
	return merged
}

func (e *Engine) renderIncludedTemplate(ctx context.Context, name string, payload map[string]interface{}, value interface{}, only bool) template.HTML {
	merged := payload
	if only {
		merged = map[string]interface{}{}
	}
	if mapped, ok := value.(map[string]interface{}); ok {
		merged = mergePayload(merged, mapped)
	} else if value != nil {
		merged = mergeTemplatePayload(merged, value)
	}
	rendered, err := e.RenderContext(ctx, name, merged)
	if err != nil {
		return template.HTML("")
	}
	return template.HTML(rendered)
}

func ResolveTemplate(cfg Config, name string) (resolvedFileStub, bool) {
	engine := New(cfg)
	return engine.Resolve(name)
}
