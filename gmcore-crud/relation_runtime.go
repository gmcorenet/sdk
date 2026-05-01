package gmcorecrud

import (
	"fmt"
	"strings"
)

type resolvedRelationMetadata struct {
	targetSchema string
	targetField  string
	primaryField string
}

func resolveRelationMetadataByTable(db interface{}, relation Relation) (resolvedRelationMetadata, error) {
	if db == nil {
		return resolvedRelationMetadata{}, fmt.Errorf("missing db")
	}
	schemaName := strings.TrimSpace(relation.TargetSchema)
	if schemaName == "" {
		return resolvedRelationMetadata{}, fmt.Errorf("relation %q missing target schema", relation.Name)
	}
	if !isValidIdentifier(schemaName) {
		return resolvedRelationMetadata{}, fmt.Errorf("relation %q invalid target schema name", relation.Name)
	}
	return resolvedRelationMetadata{
		targetSchema: schemaName,
		targetField:  relation.ForeignKey,
		primaryField: relation.TargetPrimaryKey,
	}, nil
}

func quotedIdentifier(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("missing sql identifier")
	}
	if !sqlIdentifierPattern.MatchString(value) {
		return "", fmt.Errorf("invalid sql identifier %q", value)
	}
	return value, nil
}

func emptyFieldValue(fieldType string, nullable bool) interface{} {
	if nullable {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(fieldType)) {
	case "json", "array":
		return "[]"
	case "bool", "boolean":
		return false
	case "int", "integer", "number", "float", "double", "decimal":
		return 0
	default:
		return ""
	}
}
