package gmcore_serializer

import (
	"encoding/json"
	"encoding/xml"
	"fmt"

	"gopkg.in/yaml.v3"
)

type SerializerInterface interface {
	Serialize(data interface{}, format string) ([]byte, error)
	Deserialize(data []byte, v interface{}, format string) error
}

type Serializer struct{}

func NewSerializer() *Serializer {
	return &Serializer{}
}

func (s *Serializer) Serialize(data interface{}, format string) ([]byte, error) {
	switch format {
	case "json":
		return json.Marshal(data)
	case "xml":
		return xml.Marshal(data)
	case "yaml":
		return yaml.Marshal(data)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func (s *Serializer) Deserialize(data []byte, v interface{}, format string) error {
	switch format {
	case "json":
		return json.Unmarshal(data, v)
	case "xml":
		return xml.Unmarshal(data, v)
	case "yaml":
		return yaml.Unmarshal(data, v)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func (s *Serializer) SerializeToString(data interface{}, format string) (string, error) {
	bytes, err := s.Serialize(data, format)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

type JSONSerializer struct{}

func NewJSONSerializer() *JSONSerializer {
	return &JSONSerializer{}
}

func (s *JSONSerializer) Serialize(data interface{}) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}

func (s *JSONSerializer) Deserialize(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

type XMLSerializer struct{}

func NewXMLSerializer() *XMLSerializer {
	return &XMLSerializer{}
}

func (s *XMLSerializer) Serialize(data interface{}) ([]byte, error) {
	return xml.MarshalIndent(data, "", "  ")
}

func (s *XMLSerializer) Deserialize(data []byte, v interface{}) error {
	return xml.Unmarshal(data, v)
}

type NormalizerInterface interface {
	Normalize(data interface{}) (map[string]interface{}, error)
	Denormalize(data map[string]interface{}, v interface{}) error
}

type ObjectNormalizer struct{}

func NewObjectNormalizer() *ObjectNormalizer {
	return &ObjectNormalizer{}
}

func (n *ObjectNormalizer) Normalize(data interface{}) (map[string]interface{}, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(bytes, &result)
	return result, err
}

func (n *ObjectNormalizer) Denormalize(data map[string]interface{}, v interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, v)
}

type ArrayNormalizer struct{}

func NewArrayNormalizer() *ArrayNormalizer {
	return &ArrayNormalizer{}
}

func (n *ArrayNormalizer) Normalize(data interface{}) ([]interface{}, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var result []interface{}
	err = json.Unmarshal(bytes, &result)
	return result, err
}
