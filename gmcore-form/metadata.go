package gmcore_form

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	gmcore_validation "github.com/gmcorenet/sdk/gmcore-validation"
)

type DefinitionOptions struct {
	Name  string
	Title string
}

func DefinitionFromStruct(value interface{}, options DefinitionOptions) (Definition, error) {
	typ := reflect.TypeOf(value)
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ == nil || typ.Kind() != reflect.Struct {
		return Definition{}, fmt.Errorf("form definition source must be a struct")
	}
	definition := Definition{
		Name:  firstNonEmpty(options.Name, typ.Name()),
		Title: firstNonEmpty(options.Title, typ.Name()),
	}
	for idx := 0; idx < typ.NumField(); idx++ {
		field := typ.Field(idx)
		tag := strings.TrimSpace(field.Tag.Get("form"))
		if tag == "" || tag == "-" {
			continue
		}
		meta := parseMetadataTag(tag)
		formField := Field{
			Name:        firstNonEmpty(meta["name"], lowerFirst(field.Name)),
			Label:       meta["label"],
			LabelKey:    meta["labelKey"],
			Type:        InputType(firstNonEmpty(meta["type"], "string")),
			Widget:      WidgetType(meta["widget"]),
			Required:    parseMetadataBool(meta["required"]),
			ReadOnly:    parseMetadataBool(meta["readOnly"]),
			WriteOnly:   parseMetadataBool(meta["writeOnly"]),
			AutoManaged: parseMetadataBool(meta["autoManaged"]),
			Multiple:    parseMetadataBool(meta["multiple"]),
			Placeholder: meta["placeholder"],
			HelpKey:     meta["helpKey"],
			Step:        meta["step"],
			RowSpan:     parseMetadataInt(meta["rows"]),
			ColSpan:     parseMetadataInt(meta["colSpan"]),
			Hidden:      parseMetadataBool(meta["hidden"]),
			Validation:  normalizeValidationTag(field.Tag.Get("validate")),
		}
		if formField.Required && !containsString(formField.Validation, "required") {
			formField.Validation = append([]string{"required"}, formField.Validation...)
		}
		definition.Fields = append(definition.Fields, formField)
	}
	return definition, nil
}

func ValidationSchemaFromStruct(value interface{}) (gmcore_validation.Schema, error) {
	return nil, fmt.Errorf("ValidationSchemaFromStruct not implemented: use DefinitionFromStruct which extracts validation tags into Field.Validation")
}

func normalizeValidationTag(raw string) []string {
	items := []string{}
	for _, item := range strings.Split(strings.TrimSpace(raw), ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		items = append(items, item)
	}
	return items
}

func parseMetadataTag(raw string) map[string]string {
	out := map[string]string{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			out[item] = "true"
			continue
		}
		out[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	return out
}

func parseMetadataBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseMetadataInt(raw string) int {
	value, _ := strconv.Atoi(strings.TrimSpace(raw))
	return value
}

func lowerFirst(value string) string {
	if value == "" {
		return value
	}
	return strings.ToLower(value[:1]) + value[1:]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(needle)) {
			return true
		}
	}
	return false
}
