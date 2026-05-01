package gmcorecrud

import "strings"

func RelationByName(relations []Relation, name string) (Relation, bool) {
	name = strings.TrimSpace(name)
	for _, relation := range relations {
		if strings.EqualFold(strings.TrimSpace(relation.Name), name) {
			return relation, true
		}
	}
	return Relation{}, false
}

func SplitSelectionValues(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
