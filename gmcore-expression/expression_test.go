package gmcore_expression

import (
	"testing"
)

func TestExpression_Variables(t *testing.T) {
	expr := New()

	result, err := expr.Evaluate("$name", map[string]interface{}{
		"name": "John",
	})
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != "John" {
		t.Errorf("expected John, got %v", result)
	}
}

func TestExpression_Addition(t *testing.T) {
	expr := New()

	result, err := expr.Evaluate("2 + 3", nil)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != 5 {
		t.Errorf("expected 5, got %v", result)
	}
}

func TestExpression_Subtraction(t *testing.T) {
	expr := New()

	result, err := expr.Evaluate("10 - 4", nil)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != 6 {
		t.Errorf("expected 6, got %v", result)
	}
}

func TestExpression_Multiplication(t *testing.T) {
	expr := New()

	result, err := expr.Evaluate("3 * 4", nil)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != 12 {
		t.Errorf("expected 12, got %v", result)
	}
}

func TestExpression_Division(t *testing.T) {
	expr := New()

	result, err := expr.Evaluate("15 / 3", nil)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != 5 {
		t.Errorf("expected 5, got %v", result)
	}
}

func TestExpression_ComparisonGreaterThan(t *testing.T) {
	expr := New()

	result, err := expr.Evaluate("5 > 3", nil)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestExpression_ComparisonLessThan(t *testing.T) {
	expr := New()

	result, err := expr.Evaluate("5 < 3", nil)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestExpression_LogicalAnd(t *testing.T) {
	expr := New()

	result, err := expr.Evaluate("true && true", nil)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestExpression_LogicalOr(t *testing.T) {
	expr := New()

	result, err := expr.Evaluate("true || false", nil)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestExpression_Empty(t *testing.T) {
	expr := New()

	result, err := expr.Evaluate("", nil)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestExpression_MissingVariable(t *testing.T) {
	expr := New()

	result, err := expr.Evaluate("$undefined", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for undefined variable, got %v", result)
	}
}

func TestExpression_BooleanEquality(t *testing.T) {
	expr := New()

	result, err := expr.Evaluate("true == false", nil)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}