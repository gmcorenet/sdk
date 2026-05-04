package gmcore_validation

import "testing"

func TestValidateStructWithExtendedRules(t *testing.T) {
	type User struct {
		Email           string
		Age             int
		Role            string
		Password        string
		PasswordConfirm string
	}
	errors := ValidateStruct(User{
		Email:           "bad",
		Age:             14,
		Role:            "guest",
		Password:        "secret",
		PasswordConfirm: "different",
	}, Schema{
		"Email":           {Required(), Email()},
		"Age":             {Min(18)},
		"Role":            {OneOf("admin", "editor")},
		"PasswordConfirm": {MatchField("Password")},
	})
	if !errors.HasAny() || errors.First("Email") == "" || errors.First("Age") == "" || errors.First("Role") == "" || errors.First("PasswordConfirm") == "" {
		t.Fatalf("unexpected validation result: %#v", errors)
	}
}

func TestSchemaFromStructAndValidateTaggedStruct(t *testing.T) {
	type Profile struct {
		Email    string `form:"name=email" validate:"required,email"`
		Role     string `form:"name=role" validate:"oneof=admin|editor"`
		Password string `validate:"minLength=8"`
	}
	schema, err := SchemaFromStruct(Profile{})
	if err != nil {
		t.Fatal(err)
	}
	if len(schema["email"]) != 2 || len(schema["role"]) != 1 {
		t.Fatalf("unexpected derived schema: %#v", schema)
	}
	errors := ValidateTaggedStruct(Profile{Email: "bad", Role: "guest", Password: "short"})
	if errors.First("email") == "" || errors.First("role") == "" || errors.First("Password") == "" {
		t.Fatalf("unexpected tagged validation errors: %#v", errors)
	}
}
