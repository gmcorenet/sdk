package gmcore_templating

import (
	"fmt"
	"regexp"
	"strings"
)

type twigTemplate struct {
	Extends string
	Body    string
	Blocks  map[string]string
	Macros  map[string]twigMacro
	Imports map[string]string
}

type twigMacro struct {
	Name string
	Args []string
	Body string
}

var (
	twigExtendsPattern    = regexp.MustCompile(`(?s)^\s*\{%\s*extends\s+"([^"]+)"\s*%\}\s*`)
	twigImportPattern     = regexp.MustCompile(`\{%\s*import\s+"([^"]+)"\s+as\s+([A-Za-z_][A-Za-z0-9_]*)\s*%\}`)
	twigFromImportPattern = regexp.MustCompile(`\{%\s*from\s+"([^"]+)"\s+import\s+(.+?)\s*%\}`)
	twigIncludeTagPattern = regexp.MustCompile(`\{%\s*include\s+"([^"]+)"(.*?)%\}`)
	twigTagPattern        = regexp.MustCompile(`\{%\s*(block\s+[A-Za-z0-9_.-]+|endblock)\s*%\}`)
	twigMarkerPattern     = regexp.MustCompile(`@@GMCORE_BLOCK:([A-Za-z0-9_.-]+)@@`)
	twigMacroPattern      = regexp.MustCompile(`(?s)\{%\s*macro\s+([A-Za-z_][A-Za-z0-9_]*)\s*\((.*?)\)\s*%\}(.*?)\{%\s*endmacro\s*%\}`)
)

func hasTwigSyntax(source string) bool {
	if strings.Contains(source, "{%") {
		return true
	}
	for _, match := range twigOutputPattern.FindAllStringSubmatch(source, -1) {
		if len(match) == 2 && shouldTransformTwigOutput(strings.TrimSpace(match[1])) {
			return true
		}
	}
	return false
}

func parseTwigTemplate(source string) (twigTemplate, error) {
	out := twigTemplate{Blocks: map[string]string{}, Macros: map[string]twigMacro{}, Imports: map[string]string{}}
	source = strings.ReplaceAll(source, "\r\n", "\n")
	if matches := twigExtendsPattern.FindStringSubmatch(source); len(matches) == 2 {
		out.Extends = normalizeTemplateName(matches[1])
		source = twigExtendsPattern.ReplaceAllString(source, "")
	}
	source = extractTwigImports(source, out.Imports)
	source = expandTwigIncludeTags(source)
	source, out.Macros = extractTwigMacros(source)
	body, blocks, endFound, err := parseTwigSegments(source, false)
	if err != nil {
		return twigTemplate{}, err
	}
	if endFound {
		return twigTemplate{}, fmt.Errorf("unexpected endblock")
	}
	out.Body = preprocessTwigMarkup(body)
	out.Blocks = map[string]string{}
	for name, content := range blocks {
		out.Blocks[name] = preprocessTwigMarkup(content)
	}
	for name, macro := range out.Macros {
		macro.Body = preprocessTwigMarkup(macro.Body)
		out.Macros[name] = macro
	}
	return out, nil
}

func extractTwigImports(source string, imports map[string]string) string {
	source = twigImportPattern.ReplaceAllStringFunc(source, func(raw string) string {
		matches := twigImportPattern.FindStringSubmatch(raw)
		if len(matches) != 3 {
			return raw
		}
		imports[strings.TrimSpace(matches[2])] = normalizeTemplateName(matches[1])
		return ""
	})
	return twigFromImportPattern.ReplaceAllStringFunc(source, func(raw string) string {
		matches := twigFromImportPattern.FindStringSubmatch(raw)
		if len(matches) != 3 {
			return raw
		}
		templateName := normalizeTemplateName(matches[1])
		for _, item := range strings.Split(matches[2], ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if left, right, ok := strings.Cut(item, " as "); ok {
				imports[strings.TrimSpace(right)] = templateName + "#" + strings.TrimSpace(left)
				continue
			}
			imports[item] = templateName + "#" + item
		}
		return ""
	})
}

func extractTwigMacros(source string) (string, map[string]twigMacro) {
	macros := map[string]twigMacro{}
	clean := twigMacroPattern.ReplaceAllStringFunc(source, func(raw string) string {
		matches := twigMacroPattern.FindStringSubmatch(raw)
		if len(matches) != 4 {
			return raw
		}
		name := strings.TrimSpace(matches[1])
		macros[name] = twigMacro{
			Name: name,
			Args: parseTwigMacroArgs(matches[2]),
			Body: matches[3],
		}
		return ""
	})
	return clean, macros
}

func parseTwigMacroArgs(raw string) []string {
	args := []string{}
	for _, item := range strings.Split(strings.TrimSpace(raw), ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		args = append(args, item)
	}
	return args
}

func copyTwigMacros(values map[string]twigMacro) map[string]twigMacro {
	out := map[string]twigMacro{}
	for key, value := range values {
		copiedArgs := append([]string(nil), value.Args...)
		out[key] = twigMacro{
			Name: value.Name,
			Args: copiedArgs,
			Body: value.Body,
		}
	}
	return out
}

func parseTwigSegments(source string, allowEnd bool) (string, map[string]string, bool, error) {
	blocks := map[string]string{}
	var body strings.Builder
	cursor := 0
	for {
		loc := twigTagPattern.FindStringSubmatchIndex(source[cursor:])
		if loc == nil {
			body.WriteString(source[cursor:])
			return body.String(), blocks, false, nil
		}
		start := cursor + loc[0]
		end := cursor + loc[1]
		tagValue := strings.TrimSpace(source[cursor+loc[2] : cursor+loc[3]])
		body.WriteString(source[cursor:start])
		switch {
		case strings.HasPrefix(tagValue, "block "):
			blockName := strings.TrimSpace(strings.TrimPrefix(tagValue, "block "))
			blockBody, childBlocks, foundEnd, err := parseTwigSegments(source[end:], true)
			if err != nil {
				return "", nil, false, err
			}
			if !foundEnd {
				return "", nil, false, fmt.Errorf("unterminated block %q", blockName)
			}
			for key, value := range childBlocks {
				blocks[key] = value
			}
			blocks[blockName] = blockBody
			body.WriteString(blockMarker(blockName))
			cursor = end + consumedUntilEndblock(source[end:])
			continue
		case tagValue == "endblock":
			if !allowEnd {
				return "", nil, false, fmt.Errorf("unexpected endblock")
			}
			return body.String(), blocks, true, nil
		default:
			body.WriteString(source[start:end])
			cursor = end
		}
	}
}

func consumedUntilEndblock(source string) int {
	depth := 0
	cursor := 0
	for {
		loc := twigTagPattern.FindStringSubmatchIndex(source[cursor:])
		if loc == nil {
			return len(source)
		}
		tagValue := strings.TrimSpace(source[cursor+loc[2] : cursor+loc[3]])
		tagEnd := cursor + loc[1]
		switch {
		case strings.HasPrefix(tagValue, "block "):
			depth++
		case tagValue == "endblock":
			if depth == 0 {
				return tagEnd
			}
			depth--
		}
		cursor = tagEnd
	}
}

func renderTwigBody(body string, defaults map[string]string, overrides map[string]string) string {
	return twigMarkerPattern.ReplaceAllStringFunc(body, func(marker string) string {
		matches := twigMarkerPattern.FindStringSubmatch(marker)
		if len(matches) != 2 {
			return marker
		}
		name := matches[1]
		content := defaults[name]
		if override, ok := overrides[name]; ok {
			content = override
		}
		return renderTwigBody(content, defaults, overrides)
	})
}

func copyOverrides(values map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range values {
		out[key] = value
	}
	return out
}

func blockMarker(name string) string {
	return "@@GMCORE_BLOCK:" + strings.TrimSpace(name) + "@@"
}
