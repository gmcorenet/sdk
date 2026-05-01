package gmcorecrud

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

const (
	RelationOnDeleteNone     = ""
	RelationOnDeleteRestrict = "restrict"
	RelationOnDeleteSetNull  = "set_null"
	RelationOnDeleteCascade  = "cascade"
	RelationOnDeleteDetach   = "detach"
)

func CascadeLocalFieldUpdate(ctx context.Context, db *gorm.DB, relation Relation, oldValue, newValue string) error {
	if db == nil {
		return fmt.Errorf("database unavailable")
	}
	if !relation.PropagateLocalValueChange {
		return nil
	}
	oldValue = strings.TrimSpace(oldValue)
	newValue = strings.TrimSpace(newValue)
	if oldValue == "" || newValue == "" || oldValue == newValue {
		return nil
	}
	switch relation.Type {
	case RelationHasMany:
		meta, err := resolveRelationMetadataByTable(db, relation)
		if err != nil {
			return err
		}
		targetTable, err := quotedIdentifier(meta.targetSchema)
		if err != nil {
			return err
		}
		foreignKey, err := quotedIdentifier(meta.targetField)
		if err != nil {
			return err
		}
		if err := db.WithContext(ctx).Exec(fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s = ?", targetTable, foreignKey, foreignKey), newValue, oldValue).Error; err != nil {
			return err
		}
		return nil
	case RelationManyToMany:
		pivotTable, err := quotedIdentifier(relation.PivotTable)
		if err != nil {
			return err
		}
		pivotLocalKey, err := quotedIdentifier(relation.PivotLocalKey)
		if err != nil {
			return err
		}
		return db.WithContext(ctx).Exec(fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s = ?", pivotTable, pivotLocalKey, pivotLocalKey), newValue, oldValue).Error
	default:
		return nil
	}
}

func AssignHasManySelection(ctx context.Context, db *gorm.DB, relation Relation, ownerValue string, selectedIDs []string) error {
	if db == nil {
		return fmt.Errorf("database unavailable")
	}
	if !relation.SyncSelectionAssign {
		return nil
	}
	if relation.Type != RelationHasMany {
		return nil
	}
	meta, err := resolveRelationMetadataByTable(db, relation)
	if err != nil {
		return err
	}
	targetTable, err := quotedIdentifier(meta.targetSchema)
	if err != nil {
		return err
	}
	targetPrimaryKey, err := quotedIdentifier(meta.primaryField)
	if err != nil {
		return err
	}
	foreignKey, err := quotedIdentifier(meta.targetField)
	if err != nil {
		return err
	}

	ownerValue = strings.TrimSpace(ownerValue)
	if ownerValue == "" {
		return nil
	}
	if err := db.WithContext(ctx).Exec(fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s = ?", targetTable, foreignKey, foreignKey), emptyFieldValue("", true), ownerValue).Error; err != nil {
		return err
	}
	if len(selectedIDs) == 0 {
		return nil
	}
	args := make([]interface{}, 0, len(selectedIDs)+1)
	args = append(args, ownerValue)
	for _, id := range selectedIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		args = append(args, id)
	}
	if len(args) == 1 {
		return nil
	}
	placeholders := make([]string, 0, len(args)-1)
	for i := 1; i < len(args); i++ {
		placeholders = append(placeholders, "?")
	}
	query := fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s IN (%s)", targetTable, foreignKey, targetPrimaryKey, strings.Join(placeholders, ","))
	return db.WithContext(ctx).Exec(query, args...).Error
}

func AssignManyToManySelection(ctx context.Context, db *gorm.DB, relation Relation, ownerValue string, selectedIDs []string) error {
	if db == nil {
		return fmt.Errorf("database unavailable")
	}
	if !relation.SyncSelectionAssign {
		return nil
	}
	if relation.Type != RelationManyToMany {
		return nil
	}
	pivotTable, err := quotedIdentifier(relation.PivotTable)
	if err != nil {
		return err
	}
	pivotLocalKey, err := quotedIdentifier(relation.PivotLocalKey)
	if err != nil {
		return err
	}
	pivotForeignKey, err := quotedIdentifier(relation.PivotForeignKey)
	if err != nil {
		return err
	}

	ownerValue = strings.TrimSpace(ownerValue)
	if ownerValue == "" {
		return nil
	}
	if err := db.WithContext(ctx).Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = ?", pivotTable, pivotLocalKey), ownerValue).Error; err != nil {
		return err
	}
	for _, current := range selectedIDs {
		current = strings.TrimSpace(current)
		if current == "" {
			continue
		}
		if err := db.WithContext(ctx).Exec(fmt.Sprintf("INSERT INTO %s (%s, %s) VALUES (?, ?)", pivotTable, pivotLocalKey, pivotForeignKey), ownerValue, current).Error; err != nil {
			return err
		}
	}
	return nil
}

func AssignRelationSelection(ctx context.Context, db *gorm.DB, relation Relation, ownerValue string, selectedIDs []string) error {
	switch relation.Type {
	case RelationHasMany:
		return AssignHasManySelection(ctx, db, relation, ownerValue, selectedIDs)
	case RelationManyToMany:
		return AssignManyToManySelection(ctx, db, relation, ownerValue, selectedIDs)
	default:
		return nil
	}
}

func RelationOwnerValue(record map[string]interface{}, relation Relation) string {
	if len(record) == 0 {
		return ""
	}
	localField := strings.TrimSpace(relation.LocalField)
	if localField != "" {
		return strings.TrimSpace(fmt.Sprint(record[localField]))
	}
	if value, ok := record["id"]; ok {
		return strings.TrimSpace(fmt.Sprint(value))
	}
	return ""
}

func ApplyDeleteRelations(ctx context.Context, db *gorm.DB, relations []Relation, record map[string]interface{}) error {
	if db == nil || len(relations) == 0 || len(record) == 0 {
		return nil
	}
	for _, relation := range relations {
		if err := applyDeleteRelation(ctx, db, relation, strings.TrimSpace(fmt.Sprint(record[relation.LocalField]))); err != nil {
			return err
		}
	}
	return nil
}

func applyDeleteRelation(ctx context.Context, db *gorm.DB, relation Relation, ownerValue string) error {
	ownerValue = strings.TrimSpace(ownerValue)
	if ownerValue == "" {
		return nil
	}
	switch relation.Type {
	case RelationHasMany:
		return applyHasManyDeleteRelation(ctx, db, relation, ownerValue)
	case RelationManyToMany:
		return applyManyToManyDeleteRelation(ctx, db, relation, ownerValue)
	default:
		return nil
	}
}

func applyHasManyDeleteRelation(ctx context.Context, db *gorm.DB, relation Relation, ownerValue string) error {
	meta, err := resolveRelationMetadataByTable(db, relation)
	if err != nil {
		return err
	}
	targetTable, err := quotedIdentifier(meta.targetSchema)
	if err != nil {
		return err
	}
	foreignKey, err := quotedIdentifier(meta.targetField)
	if err != nil {
		return err
	}

	switch strings.ToLower(strings.TrimSpace(relation.OnDelete)) {
	case RelationOnDeleteNone:
		return nil
	case RelationOnDeleteRestrict:
		var total int64
		if err := db.WithContext(ctx).Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", targetTable, foreignKey), ownerValue).Scan(&total).Error; err != nil {
			return err
		}
		if total > 0 {
			return fmt.Errorf("cannot delete record: relation %q still has %d linked records", relation.Name, total)
		}
		return nil
	case RelationOnDeleteSetNull:
		return db.WithContext(ctx).Exec(fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s = ?", targetTable, foreignKey, foreignKey), nil, ownerValue).Error
	case RelationOnDeleteCascade:
		return db.WithContext(ctx).Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = ?", targetTable, foreignKey), ownerValue).Error
	default:
		return nil
	}
}

func applyManyToManyDeleteRelation(ctx context.Context, db *gorm.DB, relation Relation, ownerValue string) error {
	pivotTable, err := quotedIdentifier(relation.PivotTable)
	if err != nil {
		return err
	}
	pivotLocalKey, err := quotedIdentifier(relation.PivotLocalKey)
	if err != nil {
		return err
	}

	switch strings.ToLower(strings.TrimSpace(relation.OnDelete)) {
	case RelationOnDeleteNone:
		return nil
	case RelationOnDeleteRestrict:
		var total int64
		if err := db.WithContext(ctx).Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", pivotTable, pivotLocalKey), ownerValue).Scan(&total).Error; err != nil {
			return err
		}
		if total > 0 {
			return fmt.Errorf("cannot delete record: relation %q still has %d linked records", relation.Name, total)
		}
		return nil
	case RelationOnDeleteDetach, RelationOnDeleteCascade:
		return db.WithContext(ctx).Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = ?", pivotTable, pivotLocalKey), ownerValue).Error
	default:
		return nil
	}
}
