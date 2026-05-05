package gmcore_property_access

import (
	"testing"
)

type Address struct {
	City    string
	Country string
}

type User struct {
	Name    string
	Address Address
	Emails  []string
}

func TestGetValueSimple(t *testing.T) {
	user := &User{Name: "John"}

	p := New()
	val, err := p.GetValue(user, "Name")
	if err != nil {
		t.Fatalf("get Name failed: %v", err)
	}
	if val != "John" {
		t.Errorf("expected John, got %v", val)
	}
}

func TestGetValueNested(t *testing.T) {
	user := &User{
		Name: "John",
		Address: Address{
			City: "NYC",
		},
	}

	p := New()

	val, err := p.GetValue(user, "Address.City")
	if err != nil {
		t.Fatalf("get Address.City failed: %v", err)
	}
	if val != "NYC" {
		t.Errorf("expected NYC, got %v", val)
	}
}

func TestGetValueArray(t *testing.T) {
	user := &User{
		Name:   "John",
		Emails: []string{"john@example.com", "j@work.com"},
	}

	p := New()

	val, err := p.GetValue(user, "Emails[0]")
	if err != nil {
		t.Fatalf("get Emails[0] failed: %v", err)
	}
	if val != "john@example.com" {
		t.Errorf("expected john@example.com, got %v", val)
	}
}

func TestGetValueMap(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "John",
			"age":  30,
		},
	}

	p := New()

	val, err := p.GetValue(data, "user.name")
	if err != nil {
		t.Fatalf("get user.name failed: %v", err)
	}
	if val != "John" {
		t.Errorf("expected John, got %v", val)
	}
}

func TestSetValueSimple(t *testing.T) {
	user := &User{Name: "John"}

	p := New()
	err := p.SetValue(user, "Name", "Jane")
	if err != nil {
		t.Fatalf("set Name failed: %v", err)
	}

	if user.Name != "Jane" {
		t.Errorf("expected Jane, got %s", user.Name)
	}
}

func TestSetValueNested(t *testing.T) {
	user := &User{
		Name:    "John",
		Address: Address{City: "NYC"},
	}

	p := New()
	err := p.SetValue(user, "Address.City", "LA")
	if err != nil {
		t.Fatalf("set Address.City failed: %v", err)
	}

	if user.Address.City != "LA" {
		t.Errorf("expected LA, got %s", user.Address.City)
	}
}

func TestSetValueArray(t *testing.T) {
	user := &User{
		Emails: []string{"old@example.com"},
	}

	p := New()
	err := p.SetValue(user, "Emails[0]", "new@example.com")
	if err != nil {
		t.Fatalf("set Emails[0] failed: %v", err)
	}

	if user.Emails[0] != "new@example.com" {
		t.Errorf("expected new@example.com, got %s", user.Emails[0])
	}
}

func TestIsReadable(t *testing.T) {
	user := &User{Name: "John"}
	p := New()

	if !p.IsReadable(user, "Name") {
		t.Error("Name should be readable")
	}
	if p.IsReadable(user, "NonExistent") {
		t.Error("NonExistent should not be readable")
	}
}

func TestIsWritable(t *testing.T) {
	user := &User{Name: "John"}
	p := New()

	if !p.IsWritable(user, "Name") {
		t.Error("Name should be writable")
	}
}

func TestGetValueNonPointer(t *testing.T) {
	user := User{Name: "John"}
	p := New()

	_, err := p.GetValue(user, "Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetValueInvalidPath(t *testing.T) {
	user := &User{Name: "John"}
	p := New()

	_, err := p.GetValue(user, "")
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestGetValueInvalidIndex(t *testing.T) {
	user := &User{Emails: []string{}}
	p := New()

	_, err := p.GetValue(user, "Emails[5]")
	if err == nil {
		t.Error("expected error for out of bounds index")
	}
}

func TestGetValueMapKeyNotFound(t *testing.T) {
	data := map[string]interface{}{}
	p := New()

	_, err := p.GetValue(data, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent key")
	}
}
