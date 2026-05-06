package gmcore_serializer

import (
	"testing"
)

func TestSerializer_JSON(t *testing.T) {
	s := NewSerializer()

	data := map[string]interface{}{
		"name": "test",
		"age":  42,
	}

	bytes, err := s.Serialize(data, "json")
	if err != nil {
		t.Fatalf("JSON serialization failed: %v", err)
	}

	var result map[string]interface{}
	if err := s.Deserialize(bytes, &result, "json"); err != nil {
		t.Fatalf("JSON deserialization failed: %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("expected name=test, got %v", result["name"])
	}
}

func TestSerializer_XML(t *testing.T) {
	s := NewSerializer()

	type Person struct {
		Name string `xml:"name"`
		Age  int    `xml:"age"`
	}

	data := Person{Name: "test", Age: 42}

	bytes, err := s.Serialize(data, "xml")
	if err != nil {
		t.Fatalf("XML serialization failed: %v", err)
	}

	var result Person
	if err := s.Deserialize(bytes, &result, "xml"); err != nil {
		t.Fatalf("XML deserialization failed: %v", err)
	}

	if result.Name != "test" {
		t.Errorf("expected name=test, got %v", result.Name)
	}
}

func TestSerializer_YAML(t *testing.T) {
	s := NewSerializer()

	data := map[string]interface{}{
		"name": "test",
		"age":  42,
	}

	bytes, err := s.Serialize(data, "yaml")
	if err != nil {
		t.Fatalf("YAML serialization failed: %v", err)
	}

	var result map[string]interface{}
	if err := s.Deserialize(bytes, &result, "yaml"); err != nil {
		t.Fatalf("YAML deserialization failed: %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("expected name=test, got %v", result["name"])
	}
}

func TestSerializer_UnsupportedFormat(t *testing.T) {
	s := NewSerializer()

	_, err := s.Serialize(map[string]string{"key": "value"}, "invalid")
	if err == nil {
		t.Error("expected error for unsupported format")
	}

	err = s.Deserialize([]byte("data"), nil, "invalid")
	if err == nil {
		t.Error("expected error for unsupported format in Deserialize")
	}
}

func TestJSONSerializer(t *testing.T) {
	s := NewJSONSerializer()

	data := map[string]interface{}{
		"key": "value",
	}

	bytes, err := s.Serialize(data)
	if err != nil {
		t.Fatalf("JSONSerializer.Serialize failed: %v", err)
	}

	var result map[string]interface{}
	if err := s.Deserialize(bytes, &result); err != nil {
		t.Fatalf("JSONSerializer.Deserialize failed: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("expected key=value, got %v", result["key"])
	}
}

func TestXMLSerializer(t *testing.T) {
	s := NewXMLSerializer()

	type Item struct {
		Value string `xml:"value"`
	}

	data := Item{Value: "test"}

	bytes, err := s.Serialize(data)
	if err != nil {
		t.Fatalf("XMLSerializer.Serialize failed: %v", err)
	}

	var result Item
	if err := s.Deserialize(bytes, &result); err != nil {
		t.Fatalf("XMLSerializer.Deserialize failed: %v", err)
	}

	if result.Value != "test" {
		t.Errorf("expected value=test, got %v", result.Value)
	}
}

func TestObjectNormalizer(t *testing.T) {
	n := NewObjectNormalizer()

	type Person struct {
		Name string
		Age  int
	}

	data := Person{Name: "John", Age: 30}
	normalized, err := n.Normalize(data)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	if normalized["Name"] != "John" {
		t.Errorf("expected Name=John, got %v", normalized["Name"])
	}

	var result Person
	err = n.Denormalize(normalized, &result)
	if err != nil {
		t.Fatalf("Denormalize failed: %v", err)
	}

	if result.Name != "John" || result.Age != 30 {
		t.Errorf("expected John/30, got %v/%v", result.Name, result.Age)
	}
}

func TestArrayNormalizer(t *testing.T) {
	n := NewArrayNormalizer()

	data := []string{"a", "b", "c"}
	normalized, err := n.Normalize(data)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	if len(normalized) != 3 {
		t.Errorf("expected 3 elements, got %d", len(normalized))
	}
}

func TestSerializer_SerializeToString(t *testing.T) {
	s := NewSerializer()

	data := map[string]string{"key": "value"}
	str, err := s.SerializeToString(data, "json")
	if err != nil {
		t.Fatalf("SerializeToString failed: %v", err)
	}

	if str == "" {
		t.Error("expected non-empty string")
	}
}