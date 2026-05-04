package gmcore_seed

import (
	"testing"
	"time"
)

func TestFakeRecordGeneratesCorrectValues(t *testing.T) {
	schema := Schema{
		Name:           "users",
		TableName:      "fake_users",
		PrimaryKey:     "id",
		PrimaryKeyType: "uuid",
		Fields: []Field{
			{Name: "id", Column: "id", Type: "uuid", Primary: true, Required: true},
			{Name: "email", Column: "email", Type: "email", Required: true, Writable: true},
			{Name: "roles", Column: "roles", Type: "json", Writable: true},
			{Name: "created_at", Column: "created_at", Type: "datetime", Writable: true},
		},
	}

	record, err := FakeRecord(schema, 4, Options{Now: func() time.Time {
		return time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)
	}})
	if err != nil {
		t.Fatal(err)
	}

	if record["email"] != "user000005@example.test" {
		t.Fatalf("unexpected email: %#v", record["email"])
	}

	id, ok := record["id"].(string)
	if !ok || id == "" {
		t.Fatalf("expected non-empty uuid string id: %#v", record["id"])
	}

	if record["roles"] != `["ROLE_USER"]` {
		t.Fatalf("unexpected roles: %#v", record["roles"])
	}

	if record["created_at"] == nil {
		t.Fatalf("expected created_at to be set")
	}
}

func TestFakeRecordSkipsAutoIncrementPrimary(t *testing.T) {
	schema := Schema{
		Name:           "items",
		TableName:      "items",
		PrimaryKey:     "id",
		PrimaryKeyType: "int",
		Fields: []Field{
			{Name: "id", Column: "id", Type: "int", Primary: true, AutoIncrement: true},
			{Name: "name", Column: "name", Type: "string", Writable: true},
		},
	}

	record, err := FakeRecord(schema, 0, Options{})
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := record["id"]; ok {
		t.Fatalf("expected id to be skipped for auto-increment primary key")
	}

	if record["name"] == nil || record["name"] == "" {
		t.Fatalf("expected name to be set")
	}
}

func TestFakeRecordWithOverrides(t *testing.T) {
	schema := Schema{
		Name:           "users",
		TableName:      "users",
		PrimaryKeyType: "uuid",
		Fields: []Field{
			{Name: "id", Column: "id", Type: "uuid", Primary: true, Required: true},
			{Name: "email", Column: "email", Type: "email", Required: true, Writable: true},
		},
	}

	record, err := FakeRecord(schema, 1, Options{
		Overrides: map[string]interface{}{
			"email": "custom@example.test",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if record["email"] != "custom@example.test" {
		t.Fatalf("expected override email, got: %#v", record["email"])
	}
}

func TestSchemaNormalization(t *testing.T) {
	schema := Schema{
		Name: "users",
		Fields: []Field{
			{Name: "ID", Column: "", Type: "int"},
		},
	}

	normalized := schema.Normalized()

	if normalized.TableName != "users" {
		t.Fatalf("expected table name to be 'users', got: %s", normalized.TableName)
	}

	if normalized.Fields[0].Column != "ID" {
		t.Fatalf("expected column to be 'ID', got: %s", normalized.Fields[0].Column)
	}
}
