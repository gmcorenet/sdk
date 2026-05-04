package gmcore_seed

import (
	"context"
)

type Field struct {
	Name     string
	Column   string
	Type     string
	Primary  bool
	Required bool
	Writable bool
}

type Schema struct {
	Name           string
	TableName      string
	PrimaryKey     string
	PrimaryKeyType string
	Fields         []Field
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
