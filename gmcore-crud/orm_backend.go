package gmcore_crud

import (
	"context"
	"errors"
	"fmt"
	"strings"

	gmcore_orm "github.com/gmcorenet/sdk/gmcore-orm"
	gmcore_uid "github.com/gmcorenet/sdk/gmcore-uid"
)

type ORMConfig struct {
	DB         *gmcore_orm.DB
	TableName  string
	SoftDelete bool
}

type ORMBackend struct {
	orm        *gmcore_orm.DB
	tableName  string
	softDelete bool
}

func NewORMBackend(cfg ORMConfig) (*ORMBackend, error) {
	if cfg.DB == nil {
		return nil, errors.New("missing orm DB")
	}
	tableName := strings.TrimSpace(cfg.TableName)
	if tableName == "" {
		return nil, errors.New("missing table name")
	}
	return &ORMBackend{
		orm:        cfg.DB,
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

func (b *ORMBackend) List(ctx context.Context, cfg Config, params ListParams) ([]Record, error) {
	qb := b.orm.Query(&genericEntity{tableName: b.tableName})
	qb = b.applyQuerySpec(qb, cfg, params)

	var records []map[string]interface{}
	if err := qb.Find(ctx, &records); err != nil {
		return nil, err
	}

	result := make([]Record, len(records))
	for i, r := range records {
		result[i] = Record(r)
	}
	return result, nil
}

func (b *ORMBackend) Count(ctx context.Context, cfg Config, params ListParams) (int, error) {
	qb := b.orm.Query(&genericEntity{tableName: b.tableName})
	b.applyCountSpec(qb, cfg, params)

	count, err := qb.Count(ctx)
	return int(count), err
}

func (b *ORMBackend) Get(ctx context.Context, cfg Config, key string, scope map[string]interface{}) (Record, error) {
	if err := gmcore_uid.IsValidPrimaryKey(key, gmcore_uid.PrimaryKeyType(cfg.PrimaryKeyType)); err != nil {
		return nil, err
	}

	pkField := cfg.PrimaryKey
	if !isValidIdentifier(pkField) {
		return nil, fmt.Errorf("invalid primary key field: %s", pkField)
	}

	qb := b.orm.Query(&genericEntity{tableName: b.tableName})
	qb = qb.Where(pkField+" = ?", key)

	var record map[string]interface{}
	if err := qb.First(ctx, &record); err != nil {
		return nil, errors.New("not found")
	}
	return Record(record), nil
}

func (b *ORMBackend) Create(ctx context.Context, cfg Config, record Record, scope map[string]interface{}) (Record, error) {
	var createdRecord Record
	err := b.orm.TransactionWithORMCallback(func(tx *gmcore_orm.DB) error {
		qb := tx.Query(&genericEntity{tableName: b.tableName})
		return qb.Create(ctx, record)
	})
	if err != nil {
		return nil, translateDBError(err)
	}
	createdRecord = record
	return createdRecord, nil
}

func (b *ORMBackend) Update(ctx context.Context, cfg Config, key string, record Record, scope map[string]interface{}) (Record, error) {
	if err := gmcore_uid.IsValidPrimaryKey(key, gmcore_uid.PrimaryKeyType(cfg.PrimaryKeyType)); err != nil {
		return nil, err
	}

	pkField := cfg.PrimaryKey
	if !isValidIdentifier(pkField) {
		return nil, fmt.Errorf("invalid primary key field: %s", pkField)
	}

	qb := b.orm.Query(&genericEntity{tableName: b.tableName})
	qb = qb.Where(pkField+" = ?", key)

	err := qb.Update(ctx, map[string]interface{}(record))
	if err != nil {
		return nil, translateDBError(err)
	}

	return record, nil
}

func (b *ORMBackend) Delete(ctx context.Context, cfg Config, key string, scope map[string]interface{}) error {
	if err := gmcore_uid.IsValidPrimaryKey(key, gmcore_uid.PrimaryKeyType(cfg.PrimaryKeyType)); err != nil {
		return err
	}

	pkField := cfg.PrimaryKey
	if !isValidIdentifier(pkField) {
		return fmt.Errorf("invalid primary key field: %s", pkField)
	}

	qb := b.orm.Query(&genericEntity{tableName: b.tableName})
	qb = qb.Where(pkField+" = ?", key)

	return qb.Delete(ctx)
}

func (b *ORMBackend) Bulk(ctx context.Context, cfg Config, action string, keys []string, scope map[string]interface{}) error {
	if action != "delete" {
		return errors.New("unsupported bulk action")
	}

	if len(keys) == 0 {
		return nil
	}

	for _, key := range keys {
		if err := gmcore_uid.IsValidPrimaryKey(key, gmcore_uid.PrimaryKeyType(cfg.PrimaryKeyType)); err != nil {
			return err
		}
	}

	pkField := cfg.PrimaryKey
	if !isValidIdentifier(pkField) {
		return fmt.Errorf("invalid primary key field: %s", pkField)
	}

	qb := b.orm.Query(&genericEntity{tableName: b.tableName})
	qb = qb.Where(pkField+" IN ?", keys)

	return qb.Delete(ctx)
}

func (b *ORMBackend) applyQuerySpec(qb gmcore_orm.QueryBuilder, cfg Config, params ListParams) gmcore_orm.QueryBuilder {
	b.applyCountSpec(qb, cfg, params)

	if params.Limit > 0 {
		qb = qb.Limit(params.Limit)
	}
	if params.Offset > 0 {
		qb = qb.Offset(params.Offset)
	}

	for _, sortItem := range params.Sort {
		desc := strings.HasPrefix(sortItem, "-")
		field := strings.TrimPrefix(sortItem, "-")
		if !isValidIdentifier(field) {
			continue
		}
		if desc {
			qb = qb.OrderBy(field + " DESC")
		} else {
			qb = qb.OrderBy(field)
		}
	}

	return qb
}

func (b *ORMBackend) applyCountSpec(qb gmcore_orm.QueryBuilder, cfg Config, params ListParams) {
	if params.Search != "" {
		searchTerm := "%" + params.Search + "%"
		orConditions := []string{}
		orArgs := []interface{}{}
		for _, field := range cfg.Fields {
			if field.Searchable && isValidIdentifier(field.Name) {
				orConditions = append(orConditions, field.Name+" LIKE ?")
				orArgs = append(orArgs, searchTerm)
			}
		}
		if len(orConditions) > 0 {
			qb = qb.Where(strings.Join(orConditions, " OR "), orArgs...)
		}
	}

	for field, value := range params.Filters {
		if value == "" || !isValidIdentifier(field) {
			continue
		}
		qb = qb.Where(field+" = ?", value)
	}

	for _, filter := range params.ColumnFilters {
		field := filter.Field
		if !isValidIdentifier(field) {
			continue
		}
		switch filter.Operator {
		case FilterOperatorEq:
			qb = qb.Where(field+" = ?", filter.Value)
		case FilterOperatorNeq:
			qb = qb.Where(field+" != ?", filter.Value)
		case FilterOperatorLike:
			qb = qb.Where(field+" LIKE ?", "%"+filter.Value+"%")
		case FilterOperatorGt:
			qb = qb.Where(field+" > ?", filter.Value)
		case FilterOperatorGte:
			qb = qb.Where(field+" >= ?", filter.Value)
		case FilterOperatorLt:
			qb = qb.Where(field+" < ?", filter.Value)
		case FilterOperatorLte:
			qb = qb.Where(field+" <= ?", filter.Value)
		case FilterOperatorIn:
			qb = qb.Where(field+" IN ?", filter.Values)
		case FilterOperatorIsNull:
			qb = qb.Where(field + " IS NULL")
		}
	}
}

type genericEntity struct {
	tableName string
}

func (e *genericEntity) TableName() string {
	return e.tableName
}

func (e *genericEntity) GetID() interface{} {
	return nil
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
