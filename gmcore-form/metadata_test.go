package gmcore_form

import (
	"testing"
)

type testFormInput struct {
	Email string `form:"name=email,label=Email,required" validate:"required,email"`
	Name  string `form:"name=name,label=Name" validate:"minLength=3"`
}

func TestValidationSchemaFromStruct(t *testing.T) {
	schema, err := ValidationSchemaFromStruct(testFormInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(schema) == 0 {
		t.Fatal("expected non-empty schema")
	}
	if len(schema["email"]) == 0 {
		t.Fatal("expected validation rules for email field")
	}
	if len(schema["name"]) == 0 {
		t.Fatal("expected validation rules for name field")
	}
}

func TestDefinitionFromStruct_ExtractsValidationTags(t *testing.T) {
	def, err := DefinitionFromStruct(testFormInput{}, DefinitionOptions{Name: "signup", Title: "Signup"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if def.Name != "signup" || def.Title != "Signup" {
		t.Fatalf("unexpected definition metadata: %+v", def)
	}
	if len(def.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(def.Fields))
	}
	if len(def.Fields[0].Validation) == 0 {
		t.Fatal("expected validation entries to be extracted from tags")
	}
}
