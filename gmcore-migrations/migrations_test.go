package gmcore_migrations

import (
	"testing"
	"time"
)

func TestNewMigrationManager(t *testing.T) {
	m := NewMigrationManager()
	if m == nil {
		t.Fatal("NewMigrationManager returned nil")
	}
	if m.migrations == nil {
		t.Fatal("migrations map should be initialized")
	}
	if len(m.executedMigrations) != 0 {
		t.Fatal("executedMigrations should be empty")
	}
}

func TestNewMigrationFile(t *testing.T) {
	mf := NewMigrationFile("v1", "create_users_table")
	if mf == nil {
		t.Fatal("NewMigrationFile returned nil")
	}
	if mf.Version != "v1" {
		t.Fatalf("expected version 'v1', got %s", mf.Version)
	}
	if mf.Name != "create_users_table" {
		t.Fatalf("expected name 'create_users_table', got %s", mf.Name)
	}
	if mf.UpSQL == nil {
		t.Fatal("UpSQL should be initialized")
	}
	if mf.DownSQL == nil {
		t.Fatal("DownSQL should be initialized")
	}
}

func TestMigrationFile_AddUp(t *testing.T) {
	mf := NewMigrationFile("v1", "test")
	mf.AddUp("CREATE TABLE users (id INT PRIMARY KEY)")
	mf.AddUp("CREATE INDEX idx_users_email ON users(email)")

	if len(mf.UpSQL) != 2 {
		t.Fatalf("expected 2 up statements, got %d", len(mf.UpSQL))
	}
	if mf.UpSQL[0] != "CREATE TABLE users (id INT PRIMARY KEY)" {
		t.Fatalf("unexpected first statement: %s", mf.UpSQL[0])
	}
}

func TestMigrationFile_AddDown(t *testing.T) {
	mf := NewMigrationFile("v1", "test")
	mf.AddDown("DROP TABLE users")

	if len(mf.DownSQL) != 1 {
		t.Fatalf("expected 1 down statement, got %d", len(mf.DownSQL))
	}
}

func TestMigrationFile_Up_Down_WithoutDB(t *testing.T) {
	mf := NewMigrationFile("v1", "test")
	err := mf.Up()
	if err == nil {
		t.Fatal("Up() should error without database")
	}

	err = mf.Down()
	if err == nil {
		t.Fatal("Down() should error without database")
	}
}

func TestMigrationFile_GetName(t *testing.T) {
	mf := NewMigrationFile("v2", "add_indexes")
	if mf.GetName() != "add_indexes" {
		t.Fatalf("expected 'add_indexes', got %s", mf.GetName())
	}
}

func TestMigrationManager_RegisterMigration(t *testing.T) {
	m := NewMigrationManager()
	mf := NewMigrationFile("v1", "create_users")
	m.RegisterMigration(mf)

	if m.GetMigration("create_users") != mf {
		t.Fatal("GetMigration should return the registered migration")
	}
}

func TestMigrationManager_GetMigration_Missing(t *testing.T) {
	m := NewMigrationManager()
	result := m.GetMigration("nonexistent")
	if result != nil {
		t.Fatal("GetMigration should return nil for missing migration")
	}
}

func TestMigrationManager_GetAllMigrations(t *testing.T) {
	m := NewMigrationManager()
	mf1 := NewMigrationFile("v1", "migration1")
	mf2 := NewMigrationFile("v2", "migration2")
	m.RegisterMigration(mf1)
	m.RegisterMigration(mf2)

	all := m.GetAllMigrations()
	if len(all) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(all))
	}
}

func TestMigrationManager_GetPendingMigrations(t *testing.T) {
	m := NewMigrationManager()
	mf1 := NewMigrationFile("v1", "migration1")
	mf2 := NewMigrationFile("v2", "migration2")
	m.RegisterMigration(mf1)
	m.RegisterMigration(mf2)

	pending := m.GetPendingMigrations()
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending, got %d", len(pending))
	}

	m.MarkExecuted("migration1", "v1")
	pending = m.GetPendingMigrations()
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}
}

func TestMigrationManager_MarkExecuted(t *testing.T) {
	m := NewMigrationManager()
	mf := NewMigrationFile("v1", "my_migration")
	m.RegisterMigration(mf)

	m.MarkExecuted("my_migration", "v1")
	if len(m.executedMigrations) != 1 {
		t.Fatalf("expected 1 executed migration, got %d", len(m.executedMigrations))
	}
	if m.executedMigrations[0].Name != "my_migration" {
		t.Fatalf("expected 'my_migration', got %s", m.executedMigrations[0].Name)
	}
	if m.executedMigrations[0].Version != "v1" {
		t.Fatalf("expected 'v1', got %s", m.executedMigrations[0].Version)
	}
	if m.executedMigrations[0].ExecutedAt.IsZero() {
		t.Fatal("ExecutedAt should not be zero")
	}
}

func TestMigrationManager_UnmarkExecuted(t *testing.T) {
	m := NewMigrationManager()
	mf := NewMigrationFile("v1", "test")
	m.RegisterMigration(mf)
	m.MarkExecuted("test", "v1")

	m.UnmarkExecuted("test")
	if len(m.executedMigrations) != 0 {
		t.Fatalf("expected 0 executed, got %d", len(m.executedMigrations))
	}
}

func TestMigrationManager_UnmarkExecuted_NonExistent(t *testing.T) {
	m := NewMigrationManager()
	m.UnmarkExecuted("nonexistent")
	if len(m.executedMigrations) != 0 {
		t.Fatal("UnmarkExecuted should not affect non-existent entries")
	}
}

func TestNewExecutor(t *testing.T) {
	m := NewMigrationManager()
	e := NewExecutor(m)
	if e == nil {
		t.Fatal("NewExecutor returned nil")
	}
	if e.manager != m {
		t.Fatal("executor should reference the manager")
	}
}

func TestExecutor_ExecuteUp(t *testing.T) {
	m := NewMigrationManager()
	mf := NewMigrationFile("v1", "test")
	m.RegisterMigration(mf)
	e := NewExecutor(m)

	err := e.ExecuteUp(mf)
	if err == nil {
		t.Fatal("should fail because migration has no DB")
	}

	if len(m.executedMigrations) != 0 {
		t.Fatal("should not mark executed on failure")
	}
}

func TestExecutor_ExecuteDown(t *testing.T) {
	m := NewMigrationManager()
	mf := NewMigrationFile("v1", "test")
	m.RegisterMigration(mf)
	m.MarkExecuted("test", "v1")
	e := NewExecutor(m)

	err := e.ExecuteDown(mf)
	if err == nil {
		t.Fatal("should fail because migration has no DB")
	}
}

func TestGenerateVersion(t *testing.T) {
	v := GenerateVersion()
	if len(v) != 14 {
		t.Fatalf("expected 14 chars (YYYYMMDDHHMMSS), got %d chars: %s", len(v), v)
	}
}

func TestGenerateMigrationSQL(t *testing.T) {
	sql := GenerateMigrationSQL("create_users", []string{"CREATE TABLE users (id INT)"}, []string{"DROP TABLE users"})
	if sql == "" {
		t.Fatal("GenerateMigrationSQL should not return empty string")
	}
	if !contains(sql, "create_users") {
		t.Fatal("SQL should contain migration name")
	}
	if !contains(sql, "CREATE TABLE users") {
		t.Fatal("SQL should contain up statement")
	}
	if !contains(sql, "DROP TABLE users") {
		t.Fatal("SQL should contain down statement")
	}
	if !contains(sql, "-- UP") {
		t.Fatal("SQL should contain UP marker")
	}
	if !contains(sql, "-- DOWN") {
		t.Fatal("SQL should contain DOWN marker")
	}
}

func TestVersions_struct(t *testing.T) {
	v := Version{
		Version:    "1.0.0",
		Name:       "init",
		ExecutedAt: time.Now(),
	}
	if v.Version != "1.0.0" {
		t.Fatalf("unexpected version: %s", v.Version)
	}
	if v.Name != "init" {
		t.Fatalf("unexpected name: %s", v.Name)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
