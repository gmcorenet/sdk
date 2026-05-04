package gmcore_templating

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	twigOutputPattern      = regexp.MustCompile(`\{\{\s*(.*?)\s*\}\}`)
	twigIfPattern          = regexp.MustCompile(`\{%\s*if\s+(.+?)\s*%\}`)
	twigElseIfPattern      = regexp.MustCompile(`\{%\s*elseif\s+(.+?)\s*%\}`)
	twigElsePattern        = regexp.MustCompile(`\{%\s*else\s*%\}`)
	twigEndIfPattern       = regexp.MustCompile(`\{%\s*endif\s*%\}`)
	twigForPattern         = regexp.MustCompile(`\{%\s*for\s+([A-Za-z_][A-Za-z0-9_]*)\s+in\s+(.+?)\s*%\}`)
	twigForKeyValuePattern = regexp.MustCompile(`\{%\s*for\s+([A-Za-z_][A-Za-z0-9_]*)\s*,\s*([A-Za-z_][A-Za-z0-9_]*)\s+in\s+(.+?)\s*%\}`)
	twigEndForPattern      = regexp.MustCompile(`\{%\s*endfor\s*%\}`)
	twigSetPattern         = regexp.MustCompile(`\{%\s*set\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.+?)\s*%\}`)
	twigApplyPattern       = regexp.MustCompile(`(?s)\{%\s*apply\s+(.+?)\s*%\}(.*?)\{%\s*endapply\s*%\}`)
	twigSpacelessPattern   = regexp.MustCompile(`(?s)\{%\s*spaceless\s*%\}(.*?)\{%\s*endspaceless\s*%\}`)
	twigEmbedStartPattern  = regexp.MustCompile(`\{%\s*embed\s+"([^"]+)"(?:\s+with\s+(.+?))?\s*%\}`)
	twigEmbedWithPattern   = regexp.MustCompile(`\{%\s*embed\s+"([^"]+)"\s+with\s+(.+?)\s*%\}`)
	twigEmbedPattern       = regexp.MustCompile(`\{%\s*embed\s+"([^"]+)"\s*%\}`)
	twigEndEmbedPattern    = regexp.MustCompile(`\{%\s*endembed\s*%\}`)
	twigWithPattern        = regexp.MustCompile(`\{%\s*with\s+(.+?)\s*%\}`)
	twigWithOnlyPattern    = regexp.MustCompile(`\{%\s*with\s+(.+?)\s+only\s*%\}`)
	twigEndWithPattern     = regexp.MustCompile(`\{%\s*endwith\s*%\}`)
	twigVerbatimPattern    = regexp.MustCompile(`(?s)\{%\s*verbatim\s*%\}(.*?)\{%\s*endverbatim\s*%\}`)
)

func preprocessTwigMarkup(source string) string {
	source = expandTwigApplyBlocks(source)
	source = expandTwigIncludeTags(source)
	source = expandTwigVerbatim(source)
	source = twigEmbedWithPattern.ReplaceAllString(source, `{{embed "$1" $2}}`)
	source = twigEmbedPattern.ReplaceAllString(source, `{{embed "$1" .}}`)
	source = twigEndEmbedPattern.ReplaceAllString(source, "")
	source = twigWithOnlyPattern.ReplaceAllStringFunc(source, func(raw string) string {
		match := twigWithOnlyPattern.FindStringSubmatch(raw)
		return `{{with twigOnlyWith . ` + quoteTemplateString(match[1]) + `}}`
	})
	source = twigWithPattern.ReplaceAllStringFunc(source, func(raw string) string {
		match := twigWithPattern.FindStringSubmatch(raw)
		return `{{with twigWith . ` + quoteTemplateString(match[1]) + `}}`
	})
	source = twigEndWithPattern.ReplaceAllString(source, `{{end}}`)
	source = twigIfPattern.ReplaceAllStringFunc(source, func(raw string) string {
		match := twigIfPattern.FindStringSubmatch(raw)
		return `{{if twigIf . ` + quoteTemplateString(match[1]) + `}}`
	})
	source = twigElseIfPattern.ReplaceAllStringFunc(source, func(raw string) string {
		match := twigElseIfPattern.FindStringSubmatch(raw)
		return `{{else if twigIf . ` + quoteTemplateString(match[1]) + `}}`
	})
	source = twigElsePattern.ReplaceAllString(source, `{{else}}`)
	source = twigEndIfPattern.ReplaceAllString(source, `{{end}}`)
	source = twigForKeyValuePattern.ReplaceAllStringFunc(source, func(raw string) string {
		match := twigForKeyValuePattern.FindStringSubmatch(raw)
		return fmt.Sprintf(`{{range $%s, $%s := twigIterKV . %s}}`, match[1], match[2], quoteTemplateString(match[3]))
	})
	source = twigForPattern.ReplaceAllStringFunc(source, func(raw string) string {
		match := twigForPattern.FindStringSubmatch(raw)
		return fmt.Sprintf(`{{range twigIter . %s}}`, quoteTemplateString(match[2]))
	})
	source = twigEndForPattern.ReplaceAllString(source, `{{end}}`)
	source = twigSetPattern.ReplaceAllStringFunc(source, func(raw string) string {
		match := twigSetPattern.FindStringSubmatch(raw)
		return fmt.Sprintf(`{{twigSet . %q %s}}`, match[1], quoteTemplateString(match[2]))
	})
	source = twigOutputPattern.ReplaceAllStringFunc(source, func(raw string) string {
		match := twigOutputPattern.FindStringSubmatch(raw)
		expr := strings.TrimSpace(match[1])
		if !shouldTransformTwigOutput(expr) {
			return raw
		}
		return `{{twigPrint . ` + quoteTemplateString(expr) + `}}`
	})
	return source
}

func expandTwigVerbatim(source string) string {
	return twigVerbatimPattern.ReplaceAllStringFunc(source, func(raw string) string {
		match := twigVerbatimPattern.FindStringSubmatch(raw)
		if len(match) < 2 {
			return raw
		}
		content := match[1]
		placeholder := fmt.Sprintf("{{\x00VERBATIM_%d\x00}}", len(content))
		return placeholder
	})
}

func quoteTemplateString(value string) string {
	return fmt.Sprintf("%q", strings.TrimSpace(value))
}

func shouldTransformTwigOutput(expr string) bool {
	if expr == "" {
		return false
	}
	switch expr {
	case "end", "else":
		return false
	}
	for _, prefix := range []string{"if ", "with ", "range ", "else if "} {
		if strings.HasPrefix(expr, prefix) {
			return false
		}
	}
	if strings.HasPrefix(expr, ".") || strings.HasPrefix(expr, "$") {
		return false
	}
	if strings.Contains(expr, ":=") || strings.Contains(expr, " = ") {
		return false
	}
	switch strings.TrimSpace(expr) {
	case "end", "else":
		return false
	}
	if strings.Contains(expr, "|") || strings.Contains(expr, " is ") || strings.Contains(expr, " and ") || strings.Contains(expr, " or ") || strings.Contains(expr, " ~ ") || strings.Contains(expr, " in ") || strings.Contains(expr, " ?? ") {
		return true
	}
	open := strings.Index(expr, "(")
	space := strings.Index(expr, " ")
	if open > 0 && strings.HasSuffix(expr, ")") && (space == -1 || open < space) {
		return true
	}
	if strings.Contains(expr, " ") {
		return false
	}
	for _, operator := range []string{"==", "!=", ">=", "<=", ">", "<"} {
		if strings.Contains(expr, operator) {
			return true
		}
	}
	if regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*\(.*\)$`).MatchString(expr) {
		return true
	}
	return regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*$`).MatchString(expr)
}

func expandTwigIncludeTags(source string) string {
	return twigIncludeTagPattern.ReplaceAllStringFunc(source, func(raw string) string {
		matches := twigIncludeTagPattern.FindStringSubmatch(raw)
		if len(matches) != 3 {
			return raw
		}
		name := normalizeTemplateName(matches[1])
		options := strings.TrimSpace(matches[2])
		helper := "include"
		value := "."
		if strings.HasPrefix(options, "with ") {
			right := strings.TrimSpace(strings.TrimPrefix(options, "with "))
			value = strings.TrimSpace(right)
		}
		if left, _, ok := strings.Cut(value, " ignore missing"); ok {
			value = strings.TrimSpace(left)
			helper = "includeMissing"
		}
		if left, _, ok := strings.Cut(value, " only"); ok {
			value = strings.TrimSpace(left)
			helper = "includeOnly"
		}
		if strings.Contains(options, "ignore missing") && helper == "include" {
			helper = "includeMissing"
		}
		if strings.Contains(options, "only") && helper == "include" {
			helper = "includeOnly"
		}
		return `{{` + helper + ` ` + quoteTemplateString(name) + ` ` + value + `}}`
	})
}
