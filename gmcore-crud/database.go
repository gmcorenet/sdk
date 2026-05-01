package gmcorecrud

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	gmcoreuuid "gmcore-uuid"
	"gorm.io/gorm"
)

var sqlIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func isValidIdentifier(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	return sqlIdentifierPattern.MatchString(value)
}

type ORMConfig struct {
	DB         *gorm.DB
	TableName  string
	SoftDelete bool
}

type ORMBackend struct {
	db         *gorm.DB
	tableName  string
	softDelete bool
}

func NewORMBackend(cfg ORMConfig) (*ORMBackend, error) {
	if cfg.DB == nil {
		return nil, errors.New("missing gorm DB")
	}
	tableName := strings.TrimSpace(cfg.TableName)
	if tableName == "" {
		return nil, errors.New("missing table name")
	}
	return &ORMBackend{
		db:         cfg.DB,
		tableName:  tableName,
		softDelete: cfg.SoftDelete,
	}, nil
}

func (b *ORMBackend) Kind() BackendKind {
	return BackendDatabase
}

func (b *ORMBackend) IndexQueryName() string {
	return "index"
}

func (b *ORMBackend) applySoftDeleteScope(query *gorm.DB) *gorm.DB {
	if !b.softDelete {
		return query
	}
	return query.Unscoped()
}

func (b *ORMBackend) List(ctx context.Context, cfg Config, params ListParams) ([]Record, error) {
	query := b.db.WithContext(ctx).Table(b.tableName)
	query = b.applyQuerySpec(query, cfg, params)

	var records []map[string]interface{}
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}

	result := make([]Record, len(records))
	for i, r := range records {
		result[i] = Record(r)
	}
	return result, nil
}

func (b *ORMBackend) Count(ctx context.Context, cfg Config, params ListParams) (int, error) {
	query := b.db.WithContext(ctx).Table(b.tableName)
	query = b.applyCountSpec(query, cfg, params)

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (b *ORMBackend) Get(ctx context.Context, cfg Config, key string, scope map[string]interface{}) (Record, error) {
	if err := gmcoreuuid.IsValidPrimaryKey(key, gmcoreuuid.PrimaryKeyType(cfg.PrimaryKeyType)); err != nil {
		return nil, err
	}

	query := b.db.WithContext(ctx).Table(b.tableName)
	pkField := cfg.PrimaryKey
	if !isValidIdentifier(pkField) {
		return nil, fmt.Errorf("invalid primary key field: %s", pkField)
	}

	query = b.applySoftDeleteScope(query)
	query = query.Where(fmt.Sprintf("%s = ?", pkField), key)

	var record map[string]interface{}
	if err := query.First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("not found")
		}
		return nil, err
	}
	return Record(record), nil
}

func (b *ORMBackend) Create(ctx context.Context, cfg Config, record Record, scope map[string]interface{}) (Record, error) {
	var createdRecord Record
	err := b.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(b.tableName).Create(record).Error; err != nil {
			return translateDBError(err)
		}
		createdRecord = record
		return nil
	})
	if err != nil {
		return nil, err
	}
	return createdRecord, nil
}

func (b *ORMBackend) Update(ctx context.Context, cfg Config, key string, record Record, scope map[string]interface{}) (Record, error) {
	if err := gmcoreuuid.IsValidPrimaryKey(key, gmcoreuuid.PrimaryKeyType(cfg.PrimaryKeyType)); err != nil {
		return nil, err
	}

	pkField := cfg.PrimaryKey
	if !isValidIdentifier(pkField) {
		return nil, fmt.Errorf("invalid primary key field: %s", pkField)
	}

	var updatedRecord Record
	err := b.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Table(b.tableName).Where(fmt.Sprintf("%s = ?", pkField), key).Updates(record)
		if res.Error != nil {
			return translateDBError(res.Error)
		}
		if res.RowsAffected == 0 {
			return errors.New("not found")
		}
		updatedRecord = record
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updatedRecord, nil
}

func (b *ORMBackend) Delete(ctx context.Context, cfg Config, key string, scope map[string]interface{}) error {
	if err := gmcoreuuid.IsValidPrimaryKey(key, gmcoreuuid.PrimaryKeyType(cfg.PrimaryKeyType)); err != nil {
		return err
	}

	pkField := cfg.PrimaryKey
	if !isValidIdentifier(pkField) {
		return fmt.Errorf("invalid primary key field: %s", pkField)
	}

	return b.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Table(b.tableName).Where(fmt.Sprintf("%s = ?", pkField), key)
		if !b.softDelete {
			return query.Delete(nil).Error
		}
		return query.Unscoped().Delete(nil).Error
	})
}

func (b *ORMBackend) Bulk(ctx context.Context, cfg Config, action string, keys []string, scope map[string]interface{}) error {
	if action != "delete" {
		return errors.New("unsupported bulk action")
	}

	if len(keys) == 0 {
		return nil
	}

	for _, key := range keys {
		if err := gmcoreuuid.IsValidPrimaryKey(key, gmcoreuuid.PrimaryKeyType(cfg.PrimaryKeyType)); err != nil {
			return err
		}
	}

	pkField := cfg.PrimaryKey
	if !isValidIdentifier(pkField) {
		return fmt.Errorf("invalid primary key field: %s", pkField)
	}

	return b.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Table(b.tableName).Where(fmt.Sprintf("%s IN ?", pkField), keys)
		if !b.softDelete {
			return query.Delete(nil).Error
		}
		return query.Unscoped().Delete(nil).Error
	})
}

func (b *ORMBackend) applyCountSpec(query *gorm.DB, cfg Config, params ListParams) *gorm.DB {
	if params.Search != "" {
		searchTerm := "%" + params.Search + "%"
		orConditions := []string{}
		orArgs := []interface{}{}
		for _, field := range cfg.Fields {
			if field.Searchable && isValidIdentifier(field.Name) {
				orConditions = append(orConditions, fmt.Sprintf("%s LIKE ?", field.Name))
				orArgs = append(orArgs, searchTerm)
			}
		}
		if len(orConditions) > 0 {
			query = query.Where(strings.Join(orConditions, " OR "), orArgs...)
		}
	}

	for field, value := range params.Filters {
		if value == "" || !isValidIdentifier(field) {
			continue
		}
		query = query.Where(fmt.Sprintf("%s = ?", field), value)
	}

	for _, filter := range params.ColumnFilters {
		field := filter.Field
		if !isValidIdentifier(field) {
			continue
		}
		switch filter.Operator {
		case FilterOperatorEq:
			query = query.Where(fmt.Sprintf("%s = ?", field), filter.Value)
		case FilterOperatorNeq:
			query = query.Where(fmt.Sprintf("%s != ?", field), filter.Value)
		case FilterOperatorLike:
			query = query.Where(fmt.Sprintf("%s LIKE ?", field), "%"+filter.Value+"%")
		case FilterOperatorGt:
			query = query.Where(fmt.Sprintf("%s > ?", field), filter.Value)
		case FilterOperatorGte:
			query = query.Where(fmt.Sprintf("%s >= ?", field), filter.Value)
		case FilterOperatorLt:
			query = query.Where(fmt.Sprintf("%s < ?", field), filter.Value)
		case FilterOperatorLte:
			query = query.Where(fmt.Sprintf("%s <= ?", field), filter.Value)
		case FilterOperatorIn:
			query = query.Where(fmt.Sprintf("%s IN ?", field), filter.Values)
		case FilterOperatorIsNull:
			query = query.Where(fmt.Sprintf("%s IS NULL", field))
		}
	}

	return query
}

func (b *ORMBackend) applyQuerySpec(query *gorm.DB, cfg Config, params ListParams) *gorm.DB {
	query = b.applyCountSpec(query, cfg, params)

	query = b.applySoftDeleteScope(query)

	if len(params.Select) > 0 {
		validSelect := make([]string, 0, len(params.Select))
		for _, field := range params.Select {
			if isValidIdentifier(field) {
				validSelect = append(validSelect, field)
			}
		}
		if len(validSelect) > 0 {
			query = query.Select(validSelect)
		}
	}

	if len(params.Preload) > 0 {
		for _, preload := range params.Preload {
			if isValidIdentifier(preload) {
				query = query.Preload(preload)
			}
		}
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	for _, sortItem := range params.Sort {
		desc := strings.HasPrefix(sortItem, "-")
		field := strings.TrimPrefix(sortItem, "-")
		if !isValidIdentifier(field) {
			continue
		}
		if desc {
			query = query.Order(field + " DESC")
		} else {
			query = query.Order(field)
		}
	}

	return query
}

func translateDBError(err error) error {
	if err == nil {
		return nil
	}
	errStr := err.Error()

	if strings.Contains(errStr, "UNIQUE constraint failed") ||
		strings.Contains(errStr, "Duplicate entry") ||
		strings.Contains(errStr, "unique constraint") ||
		strings.Contains(errStr, "23505") {
		return errors.New("record already exists")
	}

	if strings.Contains(errStr, "FOREIGN KEY constraint failed") ||
		strings.Contains(errStr, "foreign key constraint") ||
		strings.Contains(errStr, "23503") {
		return errors.New("record has related dependencies")
	}

	if strings.Contains(errStr, "NOT NULL constraint failed") ||
		strings.Contains(errStr, "null value") ||
		strings.Contains(errStr, "23502") {
		return errors.New("required field is missing")
	}

	return err
}
