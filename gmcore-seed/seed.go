package gmcore_seed

import (
	"context"
	"fmt"
	"time"
)

type Field struct {
	Name          string
	Column        string
	Type          string
	Primary       bool
	Required      bool
	Writable      bool
	AutoIncrement bool
}

type Schema struct {
	Name           string
	TableName      string
	PrimaryKey     string
	PrimaryKeyType string
	Fields         []Field
}

type Options struct {
	Now        func() time.Time
	Overrides  map[string]interface{}
}

func FakeRecord(schema Schema, index int, opts Options) (map[string]interface{}, error) {
	record := make(map[string]interface{})

	if opts.Now == nil {
		opts.Now = time.Now
	}

	for _, field := range schema.Fields {
		if field.Primary && field.AutoIncrement {
			continue
		}
		if field.Name == schema.PrimaryKey && schema.PrimaryKeyType == "uuid" {
			record[field.Name] = fmt.Sprintf("00000000-0000-4000-8000-00000000%04d", index)
			continue
		}
		if field.Name == schema.PrimaryKey && schema.PrimaryKeyType == "int" {
			record[field.Name] = index + 1
			continue
		}
		if override, ok := opts.Overrides[field.Name]; ok {
			record[field.Name] = override
			continue
		}
		switch field.Type {
		case "email":
			record[field.Name] = fmt.Sprintf("user%06d@example.test", index+1)
		case "string":
			record[field.Name] = fmt.Sprintf("string_%d", index)
		case "int":
			record[field.Name] = index + 1
		case "datetime", "timestamp":
			record[field.Name] = opts.Now().Format(time.RFC3339)
		case "json":
			record[field.Name] = `["ROLE_USER"]`
		case "bool":
			record[field.Name] = index%2 == 0
		default:
			record[field.Name] = nil
		}
	}

	return record, nil
}

func (s Schema) Normalized() Schema {
	normalized := s
	if normalized.TableName == "" {
		normalized.TableName = normalized.Name
	}
	for i := range normalized.Fields {
		if normalized.Fields[i].Column == "" {
			normalized.Fields[i].Column = normalized.Fields[i].Name
		}
	}
	return normalized
}

type Seeder interface {
	Seed(ctx context.Context) error
}

type SeederManager struct {
	seeders map[string]Seeder
}

func NewManager() *SeederManager {
	return &SeederManager{seeders: make(map[string]Seeder)}
}

func (m *SeederManager) Register(name string, seeder Seeder) {
	m.seeders[name] = seeder
}

func (m *SeederManager) Get(name string) Seeder {
	return m.seeders[name]
}

func (m *SeederManager) SeedAll(ctx context.Context) error {
	for _, s := range m.seeders {
		if err := s.Seed(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (m *SeederManager) SeedOne(ctx context.Context, name string) error {
	if s, ok := m.seeders[name]; ok {
		return s.Seed(ctx)
	}
	return nil
}

type SimpleSeeder struct {
	name    string
	seedFn  func(ctx context.Context) error
}

func NewSimpleSeeder(name string, seedFn func(ctx context.Context) error) *SimpleSeeder {
	return &SimpleSeeder{name: name, seedFn: seedFn}
}

func (s *SimpleSeeder) Seed(ctx context.Context) error {
	return s.seedFn(ctx)
}

func (s *SimpleSeeder) GetName() string {
	return s.name
}

type FakerSeeder struct {
	name   string
	seeds  []func(ctx context.Context) error
}

func NewFakerSeeder(name string) *FakerSeeder {
	return &FakerSeeder{name: name, seeds: make([]func(ctx context.Context) error, 0)}
}

func (s *FakerSeeder) AddSeed(fn func(ctx context.Context) error) {
	s.seeds = append(s.seeds, fn)
}

func (s *FakerSeeder) Seed(ctx context.Context) error {
	for _, fn := range s.seeds {
		if err := fn(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *FakerSeeder) GetName() string {
	return s.name
}

type DatabaseSeeder struct {
	seeds map[string]func(ctx context.Context) error
}

func NewDatabaseSeeder() *DatabaseSeeder {
	return &DatabaseSeeder{seeds: make(map[string]func(ctx context.Context) error)}
}

func (s *DatabaseSeeder) Add(name string, fn func(ctx context.Context) error) {
	s.seeds[name] = fn
}

func (s *DatabaseSeeder) Seed(ctx context.Context) error {
	for _, fn := range s.seeds {
		if err := fn(ctx); err != nil {
			return err
		}
	}
	return nil
}
