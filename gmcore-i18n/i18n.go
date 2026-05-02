package gmcorei18n

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	gmerr "github.com/gmcorenet/gmcore-error"
)

type Catalog map[string]string
type Params map[string]interface{}
type DomainCatalogs map[string]Catalog

type FrontendPayload struct {
	Locale   string            `json:"locale"`
	Domain   string            `json:"domain"`
	Prefixes []string          `json:"prefixes,omitempty"`
	Scope    string            `json:"scope,omitempty"`
	Version  string            `json:"version,omitempty"`
	Hash     string            `json:"hash,omitempty"`
	Messages map[string]string `json:"messages"`
}

type ExtractionReport struct {
	Keys   []string            `json:"keys"`
	ByFile map[string][]string `json:"by_file"`
}

type AuditReport struct {
	Locale   string   `json:"locale"`
	Domain   string   `json:"domain"`
	Missing  []string `json:"missing"`
	Orphan   []string `json:"orphan"`
	Coverage float64  `json:"coverage"`
}

type SourceAuditReport struct {
	Locale    string   `json:"locale"`
	Domain    string   `json:"domain"`
	Extracted []string `json:"extracted"`
	Missing   []string `json:"missing"`
	Orphan    []string `json:"orphan"`
	Coverage  float64  `json:"coverage"`
}

func (p Params) Interpolate(message string) string {
	return interpolateMessage(message, p)
}

type Translator struct {
	defaultLocale string
	catalogs      map[string]DomainCatalogs
}

func LoadDir(root, defaultLocale string) (*Translator, error) {
	catalogs := map[string]DomainCatalogs{}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return &Translator{defaultLocale: normalizeLocale(defaultLocale), catalogs: catalogs}, nil
		}
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		locale, domain := parseCatalogName(strings.TrimSuffix(entry.Name(), ".yaml"))
		catalog, err := loadCatalog(filepath.Join(root, entry.Name()))
		if err != nil {
			return nil, err
		}
		normalized := normalizeLocale(locale)
		if catalogs[normalized] == nil {
			catalogs[normalized] = DomainCatalogs{}
		}
		target := catalogs[normalized][domain]
		if target == nil {
			target = Catalog{}
			catalogs[normalized][domain] = target
		}
		for key, value := range catalog {
			target[key] = value
		}
	}
	return &Translator{
		defaultLocale: normalizeLocale(defaultLocale),
		catalogs:      catalogs,
	}, nil
}

func LoadDirs(roots []string, defaultLocale string) (*Translator, error) {
	merged := &Translator{
		defaultLocale: normalizeLocale(defaultLocale),
		catalogs:      map[string]DomainCatalogs{},
	}
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		current, err := LoadDir(root, defaultLocale)
		if err != nil {
			return nil, err
		}
		for locale, domains := range current.catalogs {
			targetDomains := merged.catalogs[locale]
			if targetDomains == nil {
				targetDomains = DomainCatalogs{}
				merged.catalogs[locale] = targetDomains
			}
			for domain, catalog := range domains {
				target := targetDomains[domain]
				if target == nil {
					target = Catalog{}
					targetDomains[domain] = target
				}
				for key, value := range catalog {
					target[key] = value
				}
			}
		}
	}
	return merged, nil
}

func (t *Translator) T(locale, key string, params ...Params) string {
	domain, resolvedKey := splitDomainKey(key)
	return t.TDomain(locale, domain, resolvedKey, params...)
}

func (t *Translator) TDomain(locale, domain, key string, params ...Params) string {
	resolvedLocale := normalizeLocale(locale)
	message := t.lookupWithFallback(resolvedLocale, normalizeDomain(domain), strings.TrimSpace(key))
	if message == "" {
		return strings.TrimSpace(key)
	}
	return renderLocalizedMessage(resolvedLocale, message, mergeParams(params...))
}

func (t *Translator) TC(locale, key string, count int, params ...Params) string {
	domain, resolvedKey := splitDomainKey(key)
	return t.TCDomain(locale, domain, resolvedKey, count, params...)
}

func (t *Translator) TCDomain(locale, domain, key string, count int, params ...Params) string {
	merged := mergeParams(params...)
	merged["count"] = count
	for _, candidate := range t.localeCandidates(locale) {
		if message := t.lookupPlural(candidate, normalizeDomain(domain), key, count); message != "" {
			return interpolateMessage(message, merged)
		}
	}
	message := t.lookupPlural(t.defaultLocale, normalizeDomain(domain), key, count)
	if message == "" {
		return t.TDomain(locale, domain, key, merged)
	}
	return renderLocalizedMessage(t.ResolveLocale(locale), message, merged)
}

func (t *Translator) Has(locale, key string) bool {
	domain, resolvedKey := splitDomainKey(key)
	return t.lookupWithFallback(locale, domain, resolvedKey) != ""
}

func (t *Translator) Catalog(locale string) Catalog {
	return t.CatalogDomain(locale, "")
}

func (t *Translator) CatalogDomain(locale, domain string) Catalog {
	if t == nil {
		return Catalog{}
	}
	normalized := t.ResolveLocale(locale)
	domains, ok := t.catalogs[normalized]
	if !ok {
		return Catalog{}
	}
	current, ok := domains[normalizeDomain(domain)]
	if !ok {
		return Catalog{}
	}
	out := Catalog{}
	for key, value := range current {
		out[key] = value
	}
	return out
}

func (t *Translator) Domains(locale string) []string {
	out := []string{}
	if t == nil {
		return out
	}
	for domain := range t.catalogs[t.ResolveLocale(locale)] {
		out = append(out, domain)
	}
	sort.Strings(out)
	return out
}

func (t *Translator) Namespaces(locale string) []string {
	return t.Domains(locale)
}

func (t *Translator) Messages(locale string, selectors ...string) map[string]string {
	domain, prefixes := parseMessageSelectors(selectors)
	catalog := t.CatalogDomain(locale, domain)
	out := map[string]string{}
	for key, value := range catalog {
		if len(prefixes) > 0 && !matchesAnyPrefix(key, prefixes) {
			continue
		}
		out[key] = value
	}
	return out
}

func (t *Translator) MessagesWithFallback(locale string, selectors ...string) map[string]string {
	if t == nil {
		return map[string]string{}
	}
	domain, prefixes := parseMessageSelectors(selectors)
	selectorArgs := []string{"domain=" + domain}
	for _, prefix := range prefixes {
		selectorArgs = append(selectorArgs, "prefix="+prefix)
	}
	base := t.Messages(t.defaultLocale, selectorArgs...)
	current := t.Messages(locale, selectorArgs...)
	out := map[string]string{}
	for key, value := range base {
		out[key] = value
	}
	for key, value := range current {
		out[key] = value
	}
	return out
}

func (t *Translator) MissingKeys(locale string, keys ...string) []string {
	out := []string{}
	for _, key := range keys {
		if !t.Has(locale, key) {
			out = append(out, key)
		}
	}
	return out
}

func (t *Translator) MissingTranslations(locale string, selectors ...string) []string {
	if t == nil {
		return nil
	}
	domain, prefixes := parseMessageSelectors(selectors)
	defaultMessages := t.Messages(t.defaultLocale, append([]string{"domain=" + domain}, prefixes...)...)
	if len(defaultMessages) == 0 {
		return nil
	}
	targetMessages := t.Messages(locale, append([]string{"domain=" + domain}, prefixes...)...)
	missing := make([]string, 0, len(defaultMessages))
	for key := range defaultMessages {
		if strings.TrimSpace(targetMessages[key]) == "" {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return missing
}

func (t *Translator) OrphanTranslations(locale string, selectors ...string) []string {
	if t == nil {
		return nil
	}
	domain, prefixes := parseMessageSelectors(selectors)
	selectorArgs := append([]string{"domain=" + domain}, prefixesToSelectors(prefixes)...)
	defaultMessages := t.Messages(t.defaultLocale, selectorArgs...)
	targetMessages := t.Messages(locale, selectorArgs...)
	orphan := []string{}
	for key := range targetMessages {
		if strings.TrimSpace(defaultMessages[key]) == "" {
			orphan = append(orphan, key)
		}
	}
	sort.Strings(orphan)
	return orphan
}

func (t *Translator) AuditLocale(locale string, selectors ...string) AuditReport {
	domain, prefixes := parseMessageSelectors(selectors)
	missing := t.MissingTranslations(locale, selectors...)
	orphan := t.OrphanTranslations(locale, selectors...)
	baseCount := len(t.Messages(t.defaultLocale, append([]string{"domain=" + domain}, prefixesToSelectors(prefixes)...)...))
	coverage := 1.0
	if baseCount > 0 {
		coverage = float64(baseCount-len(missing)) / float64(baseCount)
	}
	return AuditReport{
		Locale:   t.ResolveLocale(locale),
		Domain:   domain,
		Missing:  missing,
		Orphan:   orphan,
		Coverage: coverage,
	}
}

func (t *Translator) FrontendPayload(locale string, selectors ...string) FrontendPayload {
	domain, prefixes := parseMessageSelectors(selectors)
	messages := t.MessagesWithFallback(locale, selectors...)
	scope := domain
	if len(prefixes) > 0 {
		scope += ":" + strings.Join(prefixes, ",")
	}
	version := scope + "@" + t.ResolveLocale(locale)
	return FrontendPayload{
		Locale:   t.ResolveLocale(locale),
		Domain:   domain,
		Prefixes: append([]string(nil), prefixes...),
		Scope:    scope,
		Version:  version,
		Hash:     hashMessages(messages),
		Messages: messages,
	}
}

func (t *Translator) JSON(locale string, selectors ...string) string {
	data, err := json.Marshal(t.Messages(locale, selectors...))
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (t *Translator) FrontendJSON(locale string, selectors ...string) string {
	data, err := json.Marshal(t.FrontendPayload(locale, selectors...))
	if err != nil {
		return `{"locale":"","domain":"messages","messages":{}}`
	}
	return string(data)
}

func (t *Translator) HTMLJSON(locale string, selectors ...string) template.JS {
	return template.JS(t.JSON(locale, selectors...))
}

func (t *Translator) FrontendHTMLJSON(locale string, selectors ...string) template.JS {
	return template.JS(t.FrontendJSON(locale, selectors...))
}

func (t *Translator) SupportedLocales() []string {
	out := make([]string, 0, len(t.catalogs))
	for locale := range t.catalogs {
		out = append(out, locale)
	}
	sort.Strings(out)
	return out
}

func (t *Translator) ResolveLocale(candidate string) string {
	candidates := t.localeCandidates(candidate)
	for _, locale := range candidates {
		if _, ok := t.catalogs[locale]; ok {
			return locale
		}
	}
	return t.defaultLocale
}

func LocaleFromRequest(r *http.Request, translator *Translator) string {
	if translator == nil {
		return "en"
	}
	queryLocale := normalizeLocale(r.URL.Query().Get("_locale"))
	if queryLocale != "" {
		return translator.ResolveLocale(queryLocale)
	}
	if cookie, err := r.Cookie("gmcore_locale"); err == nil {
		return translator.ResolveLocale(cookie.Value)
	}
	for _, accepted := range acceptedLocales(r.Header.Get("Accept-Language")) {
		if accepted == "" {
			continue
		}
		return translator.ResolveLocale(accepted)
	}
	return translator.ResolveLocale("")
}

func HandleLocaleSwitch(w http.ResponseWriter, r *http.Request, translator *Translator) bool {
	if translator == nil {
		return false
	}
	raw := strings.TrimSpace(r.URL.Query().Get("_locale"))
	if raw == "" {
		return false
	}
	locale := translator.ResolveLocale(raw)
	http.SetCookie(w, &http.Cookie{
		Name:     "gmcore_locale",
		Value:    locale,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
	target := cloneURLWithoutLocale(r.URL)
	http.Redirect(w, r, target.String(), http.StatusSeeOther)
	return true
}

func loadCatalog(path string) (Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := Catalog{}
	flattenCatalog("", raw, out)
	return out, nil
}

func flattenCatalog(prefix string, input map[string]interface{}, out Catalog) {
	for key, value := range input {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		switch current := value.(type) {
		case string:
			out[fullKey] = current
		case int, int8, int16, int32, int64, float32, float64, bool:
			out[fullKey] = fmt.Sprint(current)
		case map[string]interface{}:
			flattenCatalog(fullKey, current, out)
		case map[interface{}]interface{}:
			converted := map[string]interface{}{}
			for subKey, subValue := range current {
				converted[fmt.Sprint(subKey)] = subValue
			}
			flattenCatalog(fullKey, converted, out)
		}
	}
}

func normalizeLocale(locale string) string {
	locale = strings.ToLower(strings.TrimSpace(locale))
	locale = strings.ReplaceAll(locale, "_", "-")
	locale = strings.TrimPrefix(locale, "-")
	locale = strings.TrimSuffix(locale, "-")
	return locale
}

func cloneURLWithoutLocale(current *url.URL) *url.URL {
	copyURL := *current
	query := copyURL.Query()
	query.Del("_locale")
	copyURL.RawQuery = query.Encode()
	return &copyURL
}

func (t *Translator) lookup(locale, domain, key string) string {
	if t == nil {
		return ""
	}
	domains, ok := t.catalogs[normalizeLocale(locale)]
	if !ok {
		return ""
	}
	catalog, ok := domains[normalizeDomain(domain)]
	if !ok {
		return ""
	}
	return catalog[strings.TrimSpace(key)]
}

func (t *Translator) lookupWithFallback(locale, domain, key string) string {
	for _, candidate := range t.localeCandidates(locale) {
		if value := t.lookup(candidate, domain, key); value != "" {
			return value
		}
	}
	if value := t.lookup(t.defaultLocale, domain, key); value != "" {
		return value
	}
	defaultBase := baseLocale(t.defaultLocale)
	if defaultBase != "" && defaultBase != t.defaultLocale {
		return t.lookup(defaultBase, domain, key)
	}
	return ""
}

func (t *Translator) lookupPlural(locale, domain, key string, count int) string {
	candidateKeys := pluralKeys(locale, strings.TrimSpace(key), count)
	for _, candidate := range candidateKeys {
		if value := t.lookup(locale, domain, candidate); value != "" {
			return value
		}
	}
	baseMessage := t.lookup(locale, domain, key)
	if baseMessage == "" {
		return ""
	}
	return selectInlinePlural(locale, baseMessage, count)
}

func (t *Translator) localeCandidates(locale string) []string {
	normalized := normalizeLocale(locale)
	out := []string{}
	if normalized != "" {
		out = append(out, normalized)
		if base := baseLocale(normalized); base != "" && base != normalized {
			out = append(out, base)
		}
	}
	if t != nil && t.defaultLocale != "" {
		if !containsLocale(out, t.defaultLocale) {
			out = append(out, t.defaultLocale)
		}
		if base := baseLocale(t.defaultLocale); base != "" && !containsLocale(out, base) {
			out = append(out, base)
		}
	}
	return out
}

func pluralKeys(locale, key string, count int) []string {
	out := []string{}
	for _, candidate := range pluralCategoriesForLocale(locale, count) {
		switch candidate {
		case "zero":
			out = append(out, key+".zero")
		case "one":
			out = append(out, key+".one", key+".singular")
		case "two":
			out = append(out, key+".two")
		case "few":
			out = append(out, key+".few")
		case "many":
			out = append(out, key+".many")
		case "other":
			out = append(out, key+".other", key+".plural")
		case "plural":
			out = append(out, key+".plural")
		}
	}
	return out
}

func selectInlinePlural(locale string, message string, count int) string {
	if !strings.Contains(message, "|") || !strings.Contains(message, ":") {
		return message
	}
	parts := strings.Split(message, "|")
	options := map[string]string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		options[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
	}
	for _, candidate := range pluralCategoriesForLocale(locale, count) {
		if value := options[candidate]; value != "" {
			return value
		}
	}
	if value := options["plural"]; value != "" {
		return value
	}
	if value := options["default"]; value != "" {
		return value
	}
	return message
}

func pluralCategoriesForLocale(locale string, count int) []string {
	switch baseLocale(locale) {
	case "fr", "pt":
		if count == 0 || count == 1 {
			return []string{"one", "singular", "other"}
		}
		return []string{"other", "plural"}
	case "ru":
		mod10 := count % 10
		mod100 := count % 100
		if mod10 == 1 && mod100 != 11 {
			return []string{"one", "other"}
		}
		if mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14) {
			return []string{"few", "other", "plural"}
		}
		return []string{"many", "other", "plural"}
	}
	switch count {
	case 0:
		return []string{"zero", "other"}
	case 1:
		return []string{"one", "singular", "other"}
	case 2:
		return []string{"two", "other"}
	default:
		if count >= 3 && count <= 4 {
			return []string{"few", "other", "plural"}
		}
		if count >= 5 {
			return []string{"many", "other", "plural"}
		}
		return []string{"other", "plural"}
	}
}

func ExtractKeysFromContent(content string) []string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\b(?:trans|trans_choice|T|TC|TDomain|TCDomain)\s*\(\s*"([^"]+)"`),
		regexp.MustCompile(`\{\{\s*(?:trans|trans_choice)\s+"([^"]+)"`),
	}
	seen := map[string]bool{}
	out := []string{}
	for _, pattern := range patterns {
		for _, match := range pattern.FindAllStringSubmatch(content, -1) {
			if len(match) != 2 {
				continue
			}
			key := strings.TrimSpace(match[1])
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func ExtractKeysFromFiles(paths ...string) (ExtractionReport, error) {
	report := ExtractionReport{Keys: []string{}, ByFile: map[string][]string{}}
	seen := map[string]bool{}
	for _, filePath := range paths {
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			continue
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return report, err
		}
		keys := ExtractKeysFromContent(string(data))
		report.ByFile[filePath] = keys
		for _, key := range keys {
			if seen[key] {
				continue
			}
			seen[key] = true
			report.Keys = append(report.Keys, key)
		}
	}
	sort.Strings(report.Keys)
	return report, nil
}

func ExtractKeysFromTree(paths ...string) (ExtractionReport, error) {
	files := []string{}
	for _, root := range paths {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		info, err := os.Stat(root)
		if err != nil {
			return ExtractionReport{}, err
		}
		if !info.IsDir() {
			files = append(files, root)
			continue
		}
		if err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() {
				return nil
			}
			extension := strings.ToLower(filepath.Ext(path))
			switch extension {
			case ".go", ".html", ".twig", ".tpl", ".tmpl":
				files = append(files, path)
			}
			return nil
		}); err != nil {
			return ExtractionReport{}, err
		}
	}
	sort.Strings(files)
	return ExtractKeysFromFiles(files...)
}

func (t *Translator) AuditSources(locale string, selectors []string, sourcePaths ...string) (SourceAuditReport, error) {
	report := SourceAuditReport{Locale: t.ResolveLocale(locale)}
	domain, _ := parseMessageSelectors(selectors)
	report.Domain = domain
	extracted, err := ExtractKeysFromTree(sourcePaths...)
	if err != nil {
		return report, err
	}
	report.Extracted = append(report.Extracted, extracted.Keys...)
	missing := []string{}
	for _, key := range extracted.Keys {
		if !t.hasAuditKey(locale, key) {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	report.Missing = missing
	report.Orphan = t.OrphanTranslations(locale, selectors...)
	total := len(report.Extracted)
	if total == 0 {
		report.Coverage = 1
		return report, nil
	}
	report.Coverage = float64(total-len(report.Missing)) / float64(total)
	return report, nil
}

func (t *Translator) hasAuditKey(locale string, key string) bool {
	if t.Has(locale, key) {
		return true
	}
	domain, resolvedKey := splitDomainKey(key)
	if resolvedKey == "" {
		return false
	}
	if strings.HasPrefix(resolvedKey, domain+".") {
		return false
	}
	return t.Has(locale, domain+":"+domain+"."+resolvedKey)
}

func NamespaceFromPath(path string) string {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" {
		return "messages"
	}
	parts := strings.Split(path, "/")
	out := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch {
		case part == "", part == "src", part == "templates", part == "translations", part == "config":
			continue
		case strings.HasSuffix(part, ".go"), strings.HasSuffix(part, ".html"), strings.HasSuffix(part, ".twig"), strings.HasSuffix(part, ".yaml"):
			part = strings.TrimSuffix(part, filepath.Ext(part))
		}
		out = append(out, slugNamespacePart(part))
	}
	if len(out) == 0 {
		return "messages"
	}
	if len(out) > 3 {
		out = out[len(out)-3:]
	}
	return strings.Join(out, ".")
}

func prefixesToSelectors(prefixes []string) []string {
	out := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		out = append(out, "prefix="+prefix)
	}
	return out
}

func hashMessages(messages map[string]string) string {
	if len(messages) == 0 {
		return ""
	}
	keys := make([]string, 0, len(messages))
	for key := range messages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	hasher := sha1.New()
	for _, key := range keys {
		_, _ = hasher.Write([]byte(key))
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write([]byte(messages[key]))
		_, _ = hasher.Write([]byte{0})
	}
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func slugNamespacePart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	value = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(value, "")
	value = strings.Trim(value, "-")
	if value == "" {
		return "messages"
	}
	return value
}

func mergeParams(values ...Params) Params {
	out := Params{}
	for _, current := range values {
		for key, value := range current {
			out[key] = value
		}
	}
	return out
}

func interpolateMessage(message string, params Params) string {
	message = renderLocalizedMessage("", message, params)
	return interpolatePlainMessage(message, params)
}

func interpolatePlainMessage(message string, params Params) string {
	if len(params) == 0 {
		return message
	}
	out := message
	for key, value := range params {
		placeholder := strings.TrimSpace(key)
		if placeholder == "" {
			continue
		}
		rendered := fmt.Sprint(value)
		out = strings.ReplaceAll(out, "%"+placeholder+"%", rendered)
		out = strings.ReplaceAll(out, "{"+placeholder+"}", rendered)
		out = strings.ReplaceAll(out, "{{ "+placeholder+" }}", rendered)
		out = strings.ReplaceAll(out, "{{"+placeholder+"}}", rendered)
	}
	return out
}

func renderLocalizedMessage(locale string, message string, params Params) string {
	formatted := renderICUMessage(locale, message, params)
	return interpolatePlainMessage(formatted, params)
}

func renderICUMessage(locale string, message string, params Params) string {
	var out strings.Builder
	cursor := 0
	for cursor < len(message) {
		open := strings.Index(message[cursor:], "{")
		if open < 0 {
			out.WriteString(message[cursor:])
			break
		}
		open += cursor
		out.WriteString(message[cursor:open])
		close := findClosingBrace(message, open)
		if close <= open {
			out.WriteString(message[open:])
			break
		}
		block := strings.TrimSpace(message[open+1 : close])
		rendered, ok := renderICUBlock(locale, block, params)
		if !ok {
			out.WriteString(message[open : close+1])
			cursor = close + 1
			continue
		}
		out.WriteString(rendered)
		cursor = close + 1
	}
	return out.String()
}

func renderICUBlock(locale string, block string, params Params) (string, bool) {
	name, kind, body, ok := parseICUHeader(block)
	if !ok {
		return "", false
	}
	options := parseICUOptions(body)
	if len(options) == 0 && kind != "number" && kind != "date" && kind != "time" {
		return "", false
	}
	switch kind {
	case "plural":
		count, ok := integerParam(params[name])
		if !ok {
			return "", false
		}
		if exact := options["="+strconv.Itoa(count)]; exact != "" {
			return strings.ReplaceAll(renderICUMessage(locale, exact, params), "#", strconv.Itoa(count)), true
		}
		for _, candidate := range pluralCategoriesForLocale(locale, count) {
			if option := options[candidate]; option != "" {
				return strings.ReplaceAll(renderICUMessage(locale, option, params), "#", strconv.Itoa(count)), true
			}
		}
		if option := options["other"]; option != "" {
			return strings.ReplaceAll(renderICUMessage(locale, option, params), "#", strconv.Itoa(count)), true
		}
	case "selectordinal":
		count, ok := integerParam(params[name])
		if !ok {
			return "", false
		}
		for _, candidate := range ordinalCategoriesForLocale(locale, count) {
			if option := options[candidate]; option != "" {
				return strings.ReplaceAll(renderICUMessage(locale, option, params), "#", strconv.Itoa(count)), true
			}
		}
		if option := options["other"]; option != "" {
			return strings.ReplaceAll(renderICUMessage(locale, option, params), "#", strconv.Itoa(count)), true
		}
	case "select":
		key := fmt.Sprint(params[name])
		if option := options[key]; option != "" {
			return renderICUMessage(locale, option, params), true
		}
		if option := options["other"]; option != "" {
			return renderICUMessage(locale, option, params), true
		}
	case "number":
		return formatICUNumber(locale, params[name]), true
	case "date":
		return formatICUTime(params[name], "2006-01-02"), true
	case "time":
		return formatICUTime(params[name], "15:04:05"), true
	}
	return "", false
}

func parseICUHeader(block string) (string, string, string, bool) {
	first, rest, ok := cutICUSegment(block)
	if !ok {
		return "", "", "", false
	}
	name := strings.TrimSpace(first)
	rest = strings.TrimSpace(rest)
	if !strings.Contains(rest, ",") {
		kind := strings.ToLower(strings.TrimSpace(rest))
		if name == "" || (kind != "number" && kind != "date" && kind != "time") {
			return "", "", "", false
		}
		return name, kind, "", true
	}
	second, body, ok := cutICUSegment(rest)
	if !ok {
		return "", "", "", false
	}
	kind := strings.ToLower(strings.TrimSpace(second))
	if name == "" || (kind != "plural" && kind != "select" && kind != "selectordinal" && kind != "number" && kind != "date" && kind != "time") {
		return "", "", "", false
	}
	return name, kind, strings.TrimSpace(body), true
}

func cutICUSegment(input string) (string, string, bool) {
	depth := 0
	for index, char := range input {
		switch char {
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				return input[:index], input[index+1:], true
			}
		}
	}
	return "", "", false
}

func parseICUOptions(body string) map[string]string {
	options := map[string]string{}
	for len(strings.TrimSpace(body)) > 0 {
		body = strings.TrimSpace(body)
		open := strings.Index(body, "{")
		if open <= 0 {
			break
		}
		key := strings.TrimSpace(body[:open])
		close := findClosingBrace(body, open)
		if close <= open {
			break
		}
		options[key] = strings.TrimSpace(body[open+1 : close])
		body = body[close+1:]
	}
	return options
}

func findClosingBrace(input string, open int) int {
	depth := 0
	for index := open; index < len(input); index++ {
		switch input[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func integerParam(value interface{}) (int, bool) {
	switch current := value.(type) {
	case int:
		return current, true
	case int8:
		return int(current), true
	case int16:
		return int(current), true
	case int32:
		return int(current), true
	case int64:
		return int(current), true
	case uint:
		return int(current), true
	case uint8:
		return int(current), true
	case uint16:
		return int(current), true
	case uint32:
		return int(current), true
	case uint64:
		return int(current), true
	case float32:
		return int(current), true
	case float64:
		return int(current), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(current))
		return parsed, err == nil
	default:
		return 0, false
	}
}

func ordinalCategoriesForLocale(locale string, count int) []string {
	switch baseLocale(locale) {
	case "en":
		mod10 := count % 10
		mod100 := count % 100
		switch {
		case mod10 == 1 && mod100 != 11:
			return []string{"one", "other"}
		case mod10 == 2 && mod100 != 12:
			return []string{"two", "other"}
		case mod10 == 3 && mod100 != 13:
			return []string{"few", "other"}
		default:
			return []string{"other"}
		}
	default:
		return []string{"other"}
	}
}

func formatICUNumber(locale string, value interface{}) string {
	number, ok := floatParam(value)
	if !ok {
		return fmt.Sprint(value)
	}
	decimal := "."
	thousands := ","
	if baseLocale(locale) == "es" || baseLocale(locale) == "fr" || baseLocale(locale) == "de" {
		decimal = ","
		thousands = "."
	}
	rendered := strconv.FormatFloat(number, 'f', -1, 64)
	left, right, hasFraction := strings.Cut(rendered, ".")
	left = addThousandsSeparator(left, thousands)
	if !hasFraction {
		return left
	}
	return left + decimal + right
}

func formatICUTime(value interface{}, layout string) string {
	normalized, ok := normalizeTimeValue(value)
	if !ok {
		return fmt.Sprint(value)
	}
	return normalized.Format(layout)
}

func normalizeTimeValue(value interface{}) (time.Time, bool) {
	switch current := value.(type) {
	case time.Time:
		return current, true
	case *time.Time:
		if current == nil {
			return time.Time{}, false
		}
		return *current, true
	case string:
		current = strings.TrimSpace(current)
		for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
			if parsed, err := time.Parse(layout, current); err == nil {
				return parsed, true
			}
		}
	}
	return time.Time{}, false
}

func floatParam(value interface{}) (float64, bool) {
	switch current := value.(type) {
	case float64:
		return current, true
	case float32:
		return float64(current), true
	case int:
		return float64(current), true
	case int8:
		return float64(current), true
	case int16:
		return float64(current), true
	case int32:
		return float64(current), true
	case int64:
		return float64(current), true
	case uint:
		return float64(current), true
	case uint8:
		return float64(current), true
	case uint16:
		return float64(current), true
	case uint32:
		return float64(current), true
	case uint64:
		return float64(current), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(current), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func addThousandsSeparator(input string, separator string) string {
	sign := ""
	if strings.HasPrefix(input, "-") {
		sign = "-"
		input = strings.TrimPrefix(input, "-")
	}
	if len(input) <= 3 {
		return sign + input
	}
	var out []string
	for len(input) > 3 {
		out = append([]string{input[len(input)-3:]}, out...)
		input = input[:len(input)-3]
	}
	if input != "" {
		out = append([]string{input}, out...)
	}
	return sign + strings.Join(out, separator)
}

func baseLocale(locale string) string {
	locale = normalizeLocale(locale)
	if locale == "" {
		return ""
	}
	parts := strings.Split(locale, "-")
	return parts[0]
}

func containsLocale(values []string, locale string) bool {
	for _, current := range values {
		if current == locale {
			return true
		}
	}
	return false
}

func acceptedLocales(header string) []string {
	parts := strings.Split(header, ",")
	type weighted struct {
		locale string
		weight float64
	}
	items := make([]weighted, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		locale := part
		weight := 1.0
		if left, right, ok := strings.Cut(part, ";"); ok {
			locale = strings.TrimSpace(left)
			for _, attr := range strings.Split(right, ";") {
				attr = strings.TrimSpace(attr)
				if !strings.HasPrefix(attr, "q=") {
					continue
				}
				if parsed, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(attr, "q=")), 64); err == nil {
					weight = parsed
				}
			}
		}
		items = append(items, weighted{locale: normalizeLocale(locale), weight: weight})
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].weight > items[j].weight
	})
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item.locale == "" || item.locale == "*" || containsLocale(out, item.locale) {
			continue
		}
		out = append(out, item.locale)
	}
	return out
}

func parseCatalogName(name string) (string, string) {
	parts := strings.Split(strings.TrimSpace(name), ".")
	if len(parts) == 0 {
		return "", "messages"
	}
	if len(parts) == 1 {
		return parts[0], "messages"
	}
	return parts[len(parts)-1], strings.Join(parts[:len(parts)-1], ".")
}

func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return "messages"
	}
	return domain
}

func splitDomainKey(key string) (string, string) {
	key = strings.TrimSpace(key)
	left, right, ok := strings.Cut(key, ":")
	if !ok || strings.Contains(left, "/") {
		return "messages", key
	}
	return normalizeDomain(left), strings.TrimSpace(right)
}

func parseMessageSelectors(selectors []string) (string, []string) {
	domain := "messages"
	prefixes := []string{}
	for _, selector := range selectors {
		selector = strings.TrimSpace(selector)
		if selector == "" {
			continue
		}
		if strings.HasPrefix(selector, "domain=") {
			domain = normalizeDomain(strings.TrimSpace(strings.TrimPrefix(selector, "domain=")))
			continue
		}
		if strings.HasPrefix(selector, "prefix=") {
			prefixes = append(prefixes, strings.TrimSpace(strings.TrimPrefix(selector, "prefix=")))
			continue
		}
		if !strings.Contains(selector, "=") && domain == "messages" {
			domain = normalizeDomain(selector)
			continue
		}
		prefixes = append(prefixes, selector)
	}
	return domain, prefixes
}

func matchesAnyPrefix(key string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return true
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(key, strings.TrimSpace(prefix)) {
			return true
		}
	}
	return false
}
