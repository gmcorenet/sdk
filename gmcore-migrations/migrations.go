package gmcore_migrations

import (
	"fmt"
	"time"
)

type Migration interface {
	Up() error
	Down() error
	GetName() string
}

type MigrationFile struct {
	Version    string
	Name       string
	UpSQL      []string
	DownSQL    []string
	ExecutedAt *time.Time
}

func NewMigrationFile(version, name string) *MigrationFile {
	return &MigrationFile{
		Version: version,
		Name:    name,
		UpSQL:   make([]string, 0),
		DownSQL: make([]string, 0),
	}
}

func (m *MigrationFile) AddUp(sql string) {
	m.UpSQL = append(m.UpSQL, sql)
}

func (m *MigrationFile) AddDown(sql string) {
	m.DownSQL = append(m.DownSQL, sql)
}

func (m *MigrationFile) Up() error {
	return fmt.Errorf("Up() requires a database executor: no database connection configured in MigrationFile")
}

func (m *MigrationFile) Down() error {
	return fmt.Errorf("Down() requires a database executor: no database connection configured in MigrationFile")
}

func (m *MigrationFile) GetName() string {
	return m.Name
}

type Version struct {
	Version    string
	Name       string
	ExecutedAt time.Time
}

type MigrationManager struct {
	migrations     map[string]Migration
	executedMigrations []Version
}

func NewMigrationManager() *MigrationManager {
	return &MigrationManager{
		migrations:          make(map[string]Migration),
		executedMigrations:  make([]Version, 0),
	}
}

func (m *MigrationManager) RegisterMigration(migration Migration) {
	m.migrations[migration.GetName()] = migration
}

func (m *MigrationManager) GetMigration(name string) Migration {
	return m.migrations[name]
}

func (m *MigrationManager) GetAllMigrations() []Migration {
	migrations := make([]Migration, 0)
	for _, mig := range m.migrations {
		migrations = append(migrations, mig)
	}
	return migrations
}

func (m *MigrationManager) GetPendingMigrations() []Migration {
	pending := make([]Migration, 0)
	for _, mig := range m.migrations {
		if !m.isExecuted(mig.GetName()) {
			pending = append(pending, mig)
		}
	}
	return pending
}

func (m *MigrationManager) isExecuted(name string) bool {
	for _, v := range m.executedMigrations {
		if v.Name == name {
			return true
		}
	}
	return false
}

func (m *MigrationManager) MarkExecuted(name, version string) {
	m.executedMigrations = append(m.executedMigrations, Version{
		Version:    version,
		Name:       name,
		ExecutedAt: time.Now(),
	})
}

func (m *MigrationManager) UnmarkExecuted(name string) {
	for i, v := range m.executedMigrations {
		if v.Name == name {
			m.executedMigrations = append(m.executedMigrations[:i], m.executedMigrations[i+1:]...)
			return
		}
	}
}

type Executor struct {
	manager *MigrationManager
}

func NewExecutor(manager *MigrationManager) *Executor {
	return &Executor{manager: manager}
}

func (e *Executor) ExecuteUp(migration Migration) error {
	if err := migration.Up(); err != nil {
		return fmt.Errorf("failed to execute up for %s: %w", migration.GetName(), err)
	}
	e.manager.MarkExecuted(migration.GetName(), "")
	return nil
}

func (e *Executor) ExecuteDown(migration Migration) error {
	if err := migration.Down(); err != nil {
		return fmt.Errorf("failed to execute down for %s: %w", migration.GetName(), err)
	}
	e.manager.UnmarkExecuted(migration.GetName())
	return nil
}

func GenerateVersion() string {
	return time.Now().Format("20060102150405")
}

func GenerateMigrationSQL(name string, upSQL, downSQL []string) string {
	sql := fmt.Sprintf(`-- Migration: %s
-- Version: %s

`, name, GenerateVersion())

sql += "-- UP\n"
for _, s := range upSQL {
	sql += s + ";\n"
}

sql += "\n-- DOWN\n"
for _, s := range downSQL {
	sql += s + ";\n"
}

return sql
}
