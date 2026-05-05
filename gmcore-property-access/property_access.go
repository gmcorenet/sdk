package gmcore_property_access

// Package gmcore_property_access provides dynamic property access to structs and maps
// using dot-notation paths with support for nested properties, array indices, and map keys.
//
// Examples:
//
//	accessor := New()
//	// Get nested value: user.Address.City
//	val, _ := accessor.GetValue(user, "Address.City")
//	// Get array element: users[0].Name
//	val, _ := accessor.GetValue(data, "Users[0].Name")
//	// Set value
//	accessor.SetValue(user, "Address.City", "NYC")

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var indexPattern = regexp.MustCompile(`\[(\d+)\]`)

// PropertyAccessor provides dynamic property access to structs and maps.
type PropertyAccessor struct{}

// New creates a new PropertyAccessor.
func New() *PropertyAccessor {
	return &PropertyAccessor{}
}

// GetValue retrieves a property value from an object using a property path.
// Supports struct fields, map keys, and array indices.
// Example paths: "Name", "Address.City", "Users[0].Name", "data[0][1]"
func (p *PropertyAccessor) GetValue(obj interface{}, path string) (interface{}, error) {
	if path == "" {
		return nil, fmt.Errorf("property path cannot be empty")
	}

	parts := p.parsePath(path)
	current := reflect.ValueOf(obj)

	for i, part := range parts {
		if !current.IsValid() {
			return nil, fmt.Errorf("property path %q: reached nil at part %d", path, i)
		}

		current = p.resolveNext(current, part)
		if !current.IsValid() {
			return nil, fmt.Errorf("property path %q: reached nil at part %d", path, i)
		}
	}

	return current.Interface(), nil
}

// SetValue sets a property value on an object using a property path.
// Supports struct fields, map keys, and array indices.
func (p *PropertyAccessor) SetValue(obj interface{}, path string, value interface{}) error {
	if path == "" {
		return fmt.Errorf("property path cannot be empty")
	}

	parts := p.parsePath(path)
	if len(parts) == 0 {
		return fmt.Errorf("property path cannot be empty")
	}

	if len(parts) == 1 {
		return p.setValueAtRoot(obj, parts[0], value)
	}

	return p.setValueAtPath(obj, parts, value)
}

func (p *PropertyAccessor) setValueAtPath(obj interface{}, parts []string, value interface{}) error {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("object must be a pointer for nested path setting")
	}

	v = v.Elem()
	if !v.IsValid() {
		return fmt.Errorf("invalid object")
	}

	for i := 0; i < len(parts)-1; i++ {
		part := p.normalizeFieldName(parts[i])

		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return fmt.Errorf("cannot traverse nil pointer at path %q", parts[i])
			}
			v = v.Elem()
		}

		switch v.Kind() {
		case reflect.Struct:
			field := v.FieldByName(part)
			if !field.IsValid() {
				return fmt.Errorf("field %q not found at path part %d", part, i)
			}
			v = field
		case reflect.Map:
			key := reflect.ValueOf(part)
			elem := v.MapIndex(key)
			if !elem.IsValid() {
				return fmt.Errorf("key %q not found in map at path part %d", part, i)
			}
			v = elem
		case reflect.Slice, reflect.Array:
			idx, err := p.parseIndex(part)
			if err != nil {
				return fmt.Errorf("invalid index %q at path part %d: %w", part, i, err)
			}
			if idx < 0 || idx >= v.Len() {
				return fmt.Errorf("index %d out of bounds at path part %d", idx, i)
			}
			v = v.Index(idx)
		default:
			return fmt.Errorf("cannot traverse into %s at path part %d", v.Kind(), i)
		}
	}

	lastPart := p.normalizeFieldName(parts[len(parts)-1])

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return fmt.Errorf("cannot set value on nil pointer")
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		field := v.FieldByName(lastPart)
		if !field.IsValid() {
			return fmt.Errorf("field %q not found in struct", lastPart)
		}
		if !field.CanSet() {
			return fmt.Errorf("field %q is not settable", lastPart)
		}
		field.Set(reflect.ValueOf(value))
	case reflect.Map:
		v.SetMapIndex(reflect.ValueOf(lastPart), reflect.ValueOf(value))
	case reflect.Slice, reflect.Array:
		idx, err := p.parseIndex(lastPart)
		if err != nil {
			return fmt.Errorf("invalid index %q: %w", lastPart, err)
		}
		if idx < 0 || idx >= v.Len() {
			return fmt.Errorf("index %d out of bounds for slice of length %d", idx, v.Len())
		}
		elem := v.Index(idx)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		if !elem.CanSet() {
			return fmt.Errorf("element at index %d is not settable", idx)
		}
		elem.Set(reflect.ValueOf(value))
	default:
		return fmt.Errorf("unsupported type for setting value: %s", v.Kind())
	}

	return nil
}

func (p *PropertyAccessor) setValueAtRoot(obj interface{}, part string, value interface{}) error {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("cannot set value on non-pointer")
	}

	fieldName := p.normalizeFieldName(part)
	v = v.Elem()

	if v.Kind() == reflect.Struct {
		field := v.FieldByName(fieldName)
		if !field.IsValid() {
			return fmt.Errorf("field %q not found in struct", fieldName)
		}
		if !field.CanSet() {
			return fmt.Errorf("field %q is not settable", fieldName)
		}
		field.Set(reflect.ValueOf(value))
		return nil
	}

	if v.Kind() == reflect.Map {
		key := reflect.ValueOf(fieldName)
		v.SetMapIndex(key, reflect.ValueOf(value))
		return nil
	}

	return fmt.Errorf("unsupported root type: %s", v.Kind())
}

func (p *PropertyAccessor) setValueAt(obj interface{}, part string, value interface{}) error {
	v := reflect.ValueOf(obj)
	part = p.normalizeFieldName(part)

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return fmt.Errorf("cannot set value on nil pointer")
		}
		v = v.Elem()
	}

	if !v.IsValid() {
		return fmt.Errorf("cannot set value on nil")
	}

	switch v.Kind() {
	case reflect.Struct:
		field := v.FieldByName(part)
		if !field.IsValid() {
			return fmt.Errorf("field %q not found in struct", part)
		}
		if !field.CanSet() {
			return fmt.Errorf("field %q is not settable", part)
		}
		field.Set(reflect.ValueOf(value))

	case reflect.Map:
		key := reflect.ValueOf(part)
		v.SetMapIndex(key, reflect.ValueOf(value))

	case reflect.Slice, reflect.Array:
		idx, err := p.parseIndex(part)
		if err != nil {
			return fmt.Errorf("invalid array index %q: %w", part, err)
		}
		if idx < 0 || idx >= v.Len() {
			return fmt.Errorf("index %d out of bounds for slice of length %d", idx, v.Len())
		}
		elem := v.Index(idx)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		if !elem.CanSet() {
			return fmt.Errorf("element at index %d is not settable", idx)
		}
		elem.Set(reflect.ValueOf(value))

	default:
		return fmt.Errorf("unsupported type for setting value: %s", v.Kind())
	}

	return nil
}

func (p *PropertyAccessor) resolveNext(v reflect.Value, part string) reflect.Value {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		fieldName := p.normalizeFieldName(part)
		field := v.FieldByName(fieldName)
		if field.IsValid() && field.CanInterface() {
			return field
		}
		return reflect.Value{}

	case reflect.Map:
		key := reflect.ValueOf(part)
		elem := v.MapIndex(key)
		if !elem.IsValid() {
			return reflect.Value{}
		}
		return elem

	case reflect.Slice, reflect.Array:
		idx, err := p.parseIndex(part)
		if err != nil {
			return reflect.Value{}
		}
		if idx < 0 || idx >= v.Len() {
			return reflect.Value{}
		}
		return v.Index(idx)

	case reflect.Interface:
		if v.IsNil() {
			return reflect.Value{}
		}
		return p.resolveNext(v.Elem(), part)
	}

	return reflect.Value{}
}

func (p *PropertyAccessor) parsePath(path string) []string {
	var parts []string
	var current strings.Builder
	for i := 0; i < len(path); i++ {
		c := path[i]
		switch c {
		case '.':
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		case '[':
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			end := strings.Index(path[i:], "]")
			if end == -1 {
				parts = append(parts, path[i:])
				i = len(path)
			} else {
				parts = append(parts, path[i:i+end+1])
				i += end
			}
		default:
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func (p *PropertyAccessor) parseIndex(part string) (int, error) {
	part = strings.Trim(part, "[]")
	if part == "" {
		return 0, fmt.Errorf("empty index")
	}
	idx, err := strconv.Atoi(part)
	if err != nil {
		return 0, fmt.Errorf("invalid index format %q: %w", part, err)
	}
	return idx, nil
}

func (p *PropertyAccessor) normalizeFieldName(name string) string {
	name = strings.Trim(name, "[]")
	name = strings.Split(name, "[")[0]
	return name
}

// IsReadable checks if a property path can be read from an object.
func (p *PropertyAccessor) IsReadable(obj interface{}, path string) bool {
	_, err := p.GetValue(obj, path)
	return err == nil
}

// IsWritable checks if a property path can be written to an object.
func (p *PropertyAccessor) IsWritable(obj interface{}, path string) bool {
	if path == "" {
		return false
	}

	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr {
		return false
	}

	parts := p.parsePath(path)
	if len(parts) == 0 {
		return false
	}

	current := v.Elem()
	for _, part := range parts[:len(parts)-1] {
		if !current.IsValid() {
			return false
		}
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return false
			}
			current = current.Elem()
		}
		current = p.resolveNext(current, part)
		if !current.IsValid() {
			return false
		}
	}

	if !current.IsValid() {
		return false
	}
	if current.Kind() == reflect.Ptr {
		current = current.Elem()
	}

	lastPart := parts[len(parts)-1]
	lastPart = p.normalizeFieldName(lastPart)

	switch current.Kind() {
	case reflect.Struct:
		field := current.FieldByName(lastPart)
		return field.IsValid() && field.CanSet()
	case reflect.Map:
		return true
	case reflect.Slice, reflect.Array:
		_, err := p.parseIndex(lastPart)
		return err == nil
	}
	return false
}
