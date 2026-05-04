package gmcore_crud

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type RelationOptionQuery struct {
	Relation      Relation
	Table         string
	ValueColumn   string
	LabelColumns  []string
	SearchColumns []string
	OrderColumns  []string
	Label         func(map[string]string) string
}

func ResolveRelationOptionsFromQuery(ctx context.Context, db *gorm.DB, cfg RelationOptionQuery, query string, page, limit int) (RelationOptionsResult, error) {
	if db == nil {
		return RelationOptionsResult{}, fmt.Errorf("database unavailable")
	}
	if limit <= 0 || limit > 250 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	table := firstNonEmpty(cfg.Table, cfg.Relation.TargetTable)
	valueColumn := firstNonEmpty(cfg.ValueColumn, cfg.Relation.ValueField, cfg.Relation.TargetPrimaryKey)
	if valueColumn == "" {
		valueColumn = "id"
	}
	if !isValidIdentifier(table) {
		return RelationOptionsResult{}, fmt.Errorf("invalid table name: %s", table)
	}
	if !isValidIdentifier(valueColumn) {
		return RelationOptionsResult{}, fmt.Errorf("invalid value column: %s", valueColumn)
	}
	columns := uniqueStrings(append([]string{valueColumn}, cfg.LabelColumns...))
	for _, col := range columns {
		if !isValidIdentifier(col) {
			return RelationOptionsResult{}, fmt.Errorf("invalid column name: %s", col)
		}
	}

	var total int64
	countQuery := db.WithContext(ctx).Table(table)
	if query != "" {
		searchPattern := "%" + query + "%"
		whereParts := []string{}
		args := []interface{}{}
		for _, col := range cfg.SearchColumns {
			if !isValidIdentifier(col) {
				continue
			}
			whereParts = append(whereParts, fmt.Sprintf("LOWER(%s) LIKE LOWER(?)", col))
			args = append(args, searchPattern)
		}
		if len(whereParts) > 0 {
			countQuery = countQuery.Where(strings.Join(whereParts, " OR "), args...)
		}
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return RelationOptionsResult{}, err
	}

	var results []map[string]interface{}
	selectQuery := db.WithContext(ctx).Table(table).Select(columns)
	if query != "" {
		searchPattern := "%" + query + "%"
		whereParts := []string{}
		args := []interface{}{}
		for _, col := range cfg.SearchColumns {
			if !isValidIdentifier(col) {
				continue
			}
			whereParts = append(whereParts, fmt.Sprintf("LOWER(%s) LIKE LOWER(?)", col))
			args = append(args, searchPattern)
		}
		if len(whereParts) > 0 {
			selectQuery = selectQuery.Where(strings.Join(whereParts, " OR "), args...)
		}
	}
	for _, col := range cfg.OrderColumns {
		if !isValidIdentifier(col) {
			continue
		}
		selectQuery = selectQuery.Order(fmt.Sprintf("%s ASC", col))
	}
	selectQuery = selectQuery.Limit(limit).Offset(offset)

	if err := selectQuery.Find(&results).Error; err != nil {
		return RelationOptionsResult{}, err
	}

	options := make([]RelationOption, 0, len(results))
	for _, record := range results {
		recordMap := make(map[string]string)
		for k, v := range record {
			recordMap[k] = fmt.Sprintf("%v", v)
		}
		value := strings.TrimSpace(recordMap[valueColumn])
		label := value
		if cfg.Label != nil {
			label = strings.TrimSpace(cfg.Label(recordMap))
		} else if len(cfg.LabelColumns) > 0 {
			label = DisplayLabelOrKey(recordMap[cfg.LabelColumns[0]], value)
		}
		options = append(options, RelationOption{Value: value, Label: DisplayLabelOrKey(label, value)})
	}

	return RelationOptionsResult{
		Relation: cfg.Relation.Name,
		Options:  options,
		Page:     page,
		Limit:    limit,
		Total:    int(total),
		HasMore:  offset+len(options) < int(total),
	}, nil
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
