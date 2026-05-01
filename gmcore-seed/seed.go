package gmcoreseed

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	gmcoreuuid "gmcore-uuid"
	"gorm.io/gorm"
)

type Options struct {
	StartIndex int
	Now       func() time.Time
	Overrides  map[string]interface{}
}

type Schema struct {
	Name         string
	TableName    string
	PrimaryKey   string
	PrimaryKeyType string
	Fields       []Field
}

type Field struct {
	Name        string
	Column      string
	Type        string
	Primary     bool
	AutoIncrement bool
	Writable    bool
	Nullable    bool
	Required    bool
}

func (s Schema) Normalized() Schema {
	s.TableName = strings.TrimSpace(s.TableName)
	if s.TableName == "" {
		s.TableName = s.Name
	}
	s.PrimaryKey = strings.TrimSpace(s.PrimaryKey)
	s.PrimaryKeyType = strings.TrimSpace(s.PrimaryKeyType)
	if s.PrimaryKeyType == "" {
		s.PrimaryKeyType = "int"
	}
	normalized := make([]Field, len(s.Fields))
	for i, f := range s.Fields {
		f.Column = strings.TrimSpace(f.Column)
		if f.Column == "" {
			f.Column = f.Name
		}
		normalized[i] = f
	}
	s.Fields = normalized
	return s
}

func (s Schema) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("schema name is required")
	}
	if len(s.Fields) == 0 {
		return fmt.Errorf("schema %q has no fields", s.Name)
	}
	return nil
}

type Record map[string]interface{}

func Seed(ctx context.Context, db *gorm.DB, schema Schema, count int, options Options) error {
	if db == nil {
		return fmt.Errorf("missing database")
	}
	schema = schema.Normalized()
	if err := schema.Validate(); err != nil {
		return err
	}
	if count < 0 {
		return fmt.Errorf("seed count must be positive")
	}
	for index := 0; index < count; index++ {
		record, err := FakeRecord(schema, options.StartIndex+index, options)
		if err != nil {
			return err
		}
		if err := insertRecord(ctx, db, schema, record); err != nil {
			return fmt.Errorf("seed %s row %d: %w", schema.Name, index+1, err)
		}
	}
	return nil
}

func SeedStruct[T any](ctx context.Context, db *gorm.DB, count int, options Options) error {
	if db == nil {
		return fmt.Errorf("missing database")
	}
	if count < 0 {
		return fmt.Errorf("seed count must be positive")
	}
	schema := SchemaFromStruct[T]()
	for index := 0; index < count; index++ {
		record, err := FakeRecord(schema, options.StartIndex+index, options)
		if err != nil {
			return err
		}
		var entity T
		if err := convertToStruct(record, &entity); err != nil {
			return err
		}
		if err := db.WithContext(ctx).Create(&entity).Error; err != nil {
			return fmt.Errorf("seed row %d: %w", index+1, err)
		}
	}
	return nil
}

func SchemaFromStruct[T any]() Schema {
	var entity T
	typ := reflect.TypeOf(entity)
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return Schema{}
	}
	schema := Schema{
		Name:         typ.Name(),
		PrimaryKey:   "id",
		PrimaryKeyType: "int",
	}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		fieldType := strings.ToLower(strings.TrimSpace(field.Tag.Get("gorm")))
		column := field.Name
		if idx := strings.Index(fieldType, "column:"); idx >= 0 {
			rest := fieldType[idx+7:]
			end := strings.IndexAny(rest, ";")
			if end < 0 {
				column = strings.TrimSpace(rest)
			} else {
				column = strings.TrimSpace(rest[:end])
			}
		}
		schema.Fields = append(schema.Fields, Field{
			Name:    field.Name,
			Column:  column,
			Type:    field.Type.Name(),
			Primary: strings.HasPrefix(field.Tag.Get("gorm"), "primaryKey"),
		})
	}
	return schema.Normalized()
}

func convertToStruct(record Record, target interface{}) error {
	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Pointer {
		return fmt.Errorf("target must be a pointer")
	}
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a struct")
	}
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		colName := field.Name
		if value, ok := record[colName]; ok {
			fieldVal := val.Field(i)
			if fieldVal.CanSet() {
				fieldVal.Set(reflect.ValueOf(value))
			}
		}
	}
	return nil
}

func FakeRecord(schema Schema, index int, options Options) (Record, error) {
	schema = schema.Normalized()
	now := time.Now
	if options.Now != nil {
		now = options.Now
	}
	record := Record{}
	for _, field := range schema.Fields {
		field = normalizedField(field)
		if !field.Writable && !field.Primary && !field.Required {
			continue
		}
		if field.AutoIncrement && field.Primary {
			continue
		}
		if value, ok := options.Overrides[field.Name]; ok {
			record[field.Name] = value
			continue
		}
		if value, ok := options.Overrides[field.Column]; ok {
			record[field.Name] = value
			continue
		}
		value, err := fakeFieldValue(schema, field, index, now)
		if err != nil {
			return nil, err
		}
		record[field.Name] = value
	}
	return record, nil
}

func insertRecord(ctx context.Context, db *gorm.DB, schema Schema, record Record) error {
	return db.WithContext(ctx).Table(schema.TableName).Create(record).Error
}

func fakeFieldValue(schema Schema, field Field, index int, now func() time.Time) (interface{}, error) {
	name := strings.ToLower(strings.TrimSpace(firstNonEmpty(field.Name, field.Column)))
	fieldType := strings.ToLower(strings.TrimSpace(field.Type))
	if field.Primary && schema.PrimaryKeyType == "uuid" {
		return gmcoreuuid.New(), nil
	}
	switch fieldType {
	case "uuid", "string", "text", "varchar", "char":
		return fakeString(name, index), nil
	case "email":
		return fmt.Sprintf("user%06d@example.test", index+1), nil
	case "int", "int8", "int16", "int32", "int64", "integer":
		return index + 1, nil
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return uint(index + 1), nil
	case "float", "float32", "float64", "decimal", "number":
		return math.Round(float64(index+1)*123.45) / 100, nil
	case "bool", "boolean":
		return index%2 == 0, nil
	case "time", "datetime", "timestamp", "date":
		return now().Add(-time.Duration(index) * time.Hour).UTC(), nil
	case "json", "jsonb":
		if strings.Contains(name, "role") {
			return `["ROLE_USER"]`, nil
		}
		return `[]`, nil
	default:
		if field.Nullable && !field.Required {
			return nil, nil
		}
		return fakeString(name, index), nil
	}
}

func fakeString(name string, index int) string {
	switch {
	case strings.Contains(name, "email"):
		return fmt.Sprintf("user%06d@example.test", index+1)
	case strings.Contains(name, "display") || strings.Contains(name, "name"):
		return fmt.Sprintf("Fake Name %d", index+1)
	case strings.Contains(name, "title"):
		return fmt.Sprintf("Fake Title %d", index+1)
	case strings.Contains(name, "slug"):
		return fmt.Sprintf("fake-item-%d", index+1)
	case strings.Contains(name, "locale"):
		return "es"
	case strings.Contains(name, "status"):
		return "active"
	case strings.Contains(name, "category"):
		return "general"
	case strings.Contains(name, "password"):
		return "fake-password-" + strconv.Itoa(index+1)
	case strings.Contains(name, "url"):
		return fmt.Sprintf("https://example.test/%d", index+1)
	case strings.Contains(name, "content") || strings.Contains(name, "description") || strings.Contains(name, "body"):
		return fmt.Sprintf("Fake generated content %d.", index+1)
	default:
		return fmt.Sprintf("fake-%s-%d", firstNonEmpty(name, "value"), index+1)
	}
}

func normalizedField(field Field) Field {
	if strings.TrimSpace(field.Column) == "" {
		field.Column = field.Name
	}
	return field
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
