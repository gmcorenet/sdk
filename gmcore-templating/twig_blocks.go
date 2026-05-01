package gmcoretemplating

import (
	"fmt"
	"html/template"
	"sort"
	"strings"
)

func (e *Engine) expandTwigEmbeds(source string, stack map[string]bool) (string, []string, error) {
	var dependencies []string
	for {
		start, end, templateName, withExpr, inner, found, err := findTwigEmbedBlock(source)
		if err != nil {
			return "", nil, err
		}
		if !found {
			return source, mergeTwigDependencies(dependencies), nil
		}
		parsed, err := parseTwigTemplate(inner)
		if err != nil {
			return "", nil, err
		}
		macros := copyTwigMacros(parsed.Macros)
		importedMacros, importDependencies, err := e.resolveImportedTwigMacros(parsed.Imports, stack)
		if err != nil {
			return "", nil, err
		}
		for key, value := range importedMacros {
			macros[key] = value
		}
		resolved, err := e.resolveTwigSource(templateName, parsed.Blocks, macros, stack)
		if err != nil {
			return "", nil, err
		}
		replacement := resolved.Body
		if strings.TrimSpace(withExpr) != "" {
			replacement = `{{with twigWith . ` + quoteTemplateString(withExpr) + `}}` + replacement + `{{end}}`
		}
		source = source[:start] + replacement + source[end:]
		dependencies = append(dependencies, resolved.Dependencies...)
		dependencies = append(dependencies, importDependencies...)
	}
}

func findTwigEmbedBlock(source string) (int, int, string, string, string, bool, error) {
	startLoc := twigEmbedStartPattern.FindStringSubmatchIndex(source)
	if startLoc == nil {
		return 0, 0, "", "", "", false, nil
	}
	start := startLoc[0]
	tagEnd := startLoc[1]
	match := twigEmbedStartPattern.FindStringSubmatch(source[start:tagEnd])
	if len(match) < 2 {
		return 0, 0, "", "", "", false, fmt.Errorf("invalid embed block")
	}
	templateName := normalizeTemplateName(match[1])
	withExpr := ""
	if len(match) > 2 {
		withExpr = strings.TrimSpace(match[2])
	}
	depth := 1
	cursor := tagEnd
	for cursor < len(source) {
		nextStart := twigEmbedStartPattern.FindStringIndex(source[cursor:])
		nextEnd := twigEndEmbedPattern.FindStringIndex(source[cursor:])
		if nextEnd == nil {
			return 0, 0, "", "", "", false, fmt.Errorf("unterminated embed for %s", templateName)
		}
		if nextStart != nil && nextStart[0] < nextEnd[0] {
			depth++
			cursor += nextStart[1]
			continue
		}
		endStart := cursor + nextEnd[0]
		end := cursor + nextEnd[1]
		depth--
		if depth == 0 {
			return start, end, templateName, withExpr, source[tagEnd:endStart], true, nil
		}
		cursor = end
	}
	return 0, 0, "", "", "", false, fmt.Errorf("unterminated embed for %s", templateName)
}

func expandTwigApplyBlocks(source string) string {
	for {
		next := twigApplyPattern.ReplaceAllStringFunc(source, func(raw string) string {
			match := twigApplyPattern.FindStringSubmatch(raw)
			if len(match) != 3 {
				return raw
			}
			return `{{twigRenderBlock . ` + quoteTemplateString(match[2]) + ` ` + quoteTemplateString(match[1]) + `}}`
		})
		next = twigSpacelessPattern.ReplaceAllStringFunc(next, func(raw string) string {
			match := twigSpacelessPattern.FindStringSubmatch(raw)
			if len(match) != 2 {
				return raw
			}
			return `{{twigRenderBlock . ` + quoteTemplateString(match[1]) + ` "spaceless"}}`
		})
		if next == source {
			return source
		}
		source = next
	}
}

func (e *Engine) buildTwigRenderBlockFunc(currentName string, funcs template.FuncMap) func(interface{}, string, string) template.HTML {
	return func(current interface{}, body string, pipeline string) template.HTML {
		fragment := preprocessTwigMarkup(body)
		inlineFuncs := template.FuncMap{}
		for key, value := range funcs {
			inlineFuncs[key] = value
		}
		tpl, err := template.New(currentName + "__block").Funcs(inlineFuncs).Parse(`{{define "block"}}` + fragment + `{{end}}`)
		if err != nil {
			return ""
		}
		var rendered strings.Builder
		if err := tpl.ExecuteTemplate(&rendered, "block", current); err != nil {
			return ""
		}
		output := rendered.String()
		for _, step := range splitPipeline(strings.TrimSpace(pipeline)) {
			step = strings.TrimSpace(step)
			if step == "" {
				continue
			}
			if strings.EqualFold(step, "spaceless") {
				output = strings.Join(strings.Fields(output), " ")
				continue
			}
			filterName, args := parseInvocation(step)
			if filterName == "" {
				filterName = step
			}
			resolvedArgs := evaluateTwigArgs(current, args, funcs)
			output = fmt.Sprint(applyFilter(filterName, output, resolvedArgs...))
		}
		return template.HTML(output)
	}
}

func registerImportedTwigMacros(funcs template.FuncMap, macros map[string]twigMacro, build func(twigMacro) func(...interface{}) template.HTML) []string {
	names := make([]string, 0, len(macros))
	for macroName := range macros {
		names = append(names, macroName)
	}
	sort.Strings(names)
	for _, macroName := range names {
		funcs[normalizeTwigCallableName(macroName)] = build(macros[macroName])
	}
	return names
}
