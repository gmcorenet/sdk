package gmcorevalidation

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type Errors map[string][]string

type Rule interface {
	Validate(field string, value interface{}) string
}

type Schema map[string][]Rule

func (e Errors) Add(field, message string) {
	if strings.TrimSpace(message) == "" {
		return
	}
	e[field] = append(e[field], strings.TrimSpace(message))
}

func (e Errors) HasAny() bool {
	return len(e) > 0
}

func (e Errors) First(field string) string {
	if len(e[field]) == 0 {
		return ""
	}
	return e[field][0]
}

func (e Errors) Merge(other Errors) Errors {
	if len(other) == 0 {
		return e
	}
	if e == nil {
		e = Errors{}
	}
	for field, messages := range other {
		e[field] = append(e[field], messages...)
	}
	return e
}

func (e Errors) Error() string {
	if !e.HasAny() {
		return ""
	}
	parts := []string{}
	for field, messages := range e {
		for _, message := range messages {
			parts = append(parts, field+": "+message)
		}
	}
	return strings.Join(parts, "; ")
}

func Validate(values map[string]interface{}, schema Schema) Errors {
	errors := Errors{}
	for field, rules := range schema {
		value := values[field]
		for _, rule := range rules {
			if rule == nil {
				continue
			}
			errors.Add(field, rule.Validate(field, value))
		}
	}
	if !errors.HasAny() {
		return nil
	}
	return errors
}

type RequiredRule struct{}
type MinLengthRule struct{ Value int }
type MaxLengthRule struct{ Value int }
type MinRule struct{ Value float64 }
type MaxRule struct{ Value float64 }
type EmailRule struct{}

var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type PatternRule struct {
	Pattern *regexp.Regexp
	Label   string
}
type OneOfRule struct{ Values []string }
type MatchFieldRule struct{ Other string }

func Required() Rule           { return RequiredRule{} }
func MinLength(value int) Rule { return MinLengthRule{Value: value} }
func MaxLength(value int) Rule { return MaxLengthRule{Value: value} }
func Min(value float64) Rule   { return MinRule{Value: value} }
func Max(value float64) Rule   { return MaxRule{Value: value} }
func Email() Rule              { return EmailRule{} }
func Pattern(expr string) Rule {
	compiled, _ := regexp.Compile(strings.TrimSpace(expr))
	return PatternRule{Pattern: compiled, Label: strings.TrimSpace(expr)}
}
func OneOf(values ...string) Rule {
	return OneOfRule{Values: append([]string(nil), values...)}
}
func MatchField(other string) Rule { return MatchFieldRule{Other: strings.TrimSpace(other)} }

func (RequiredRule) Validate(field string, value interface{}) string {
	if strings.TrimSpace(fmt.Sprint(value)) == "" {
		return field + " is required"
	}
	return ""
}

func (r MinLengthRule) Validate(field string, value interface{}) string {
	if len(strings.TrimSpace(fmt.Sprint(value))) < r.Value {
		return fmt.Sprintf("%s must have at least %d characters", field, r.Value)
	}
	return ""
}

func (EmailRule) Validate(field string, value interface{}) string {
	email := strings.TrimSpace(fmt.Sprint(value))
	if email == "" {
		return ""
	}
	if !emailPattern.MatchString(email) {
		return field + " must be a valid email"
	}
	return ""
}

func (r PatternRule) Validate(field string, value interface{}) string {
	if r.Pattern == nil {
		return ""
	}
	current := strings.TrimSpace(fmt.Sprint(value))
	if current == "" {
		return ""
	}
	if !r.Pattern.MatchString(current) {
		return field + " has an invalid format"
	}
	return ""
}

func (r MaxLengthRule) Validate(field string, value interface{}) string {
	if len(strings.TrimSpace(fmt.Sprint(value))) > r.Value {
		return fmt.Sprintf("%s must have at most %d characters", field, r.Value)
	}
	return ""
}

func (r MinRule) Validate(field string, value interface{}) string {
	number, ok := numericValue(value)
	if ok && number < r.Value {
		return fmt.Sprintf("%s must be greater than or equal to %s", field, trimNumeric(r.Value))
	}
	return ""
}

func (r MaxRule) Validate(field string, value interface{}) string {
	number, ok := numericValue(value)
	if ok && number > r.Value {
		return fmt.Sprintf("%s must be less than or equal to %s", field, trimNumeric(r.Value))
	}
	return ""
}

func (r OneOfRule) Validate(field string, value interface{}) string {
	current := strings.TrimSpace(fmt.Sprint(value))
	for _, candidate := range r.Values {
		if current == strings.TrimSpace(candidate) {
			return ""
		}
	}
	return field + " must be one of the allowed values"
}

func (r MatchFieldRule) Validate(field string, value interface{}) string {
	return ""
}

type matchFieldHelper struct {
	Rule     MatchFieldRule
	Field    string
	Values   map[string]interface{}
}

func collectMatchFieldHelpers(schema Schema, values map[string]interface{}) []matchFieldHelper {
	var helpers []matchFieldHelper
	for field, rules := range schema {
		for _, rule := range rules {
			if mfr, ok := rule.(MatchFieldRule); ok {
				helpers = append(helpers, matchFieldHelper{
					Rule:   mfr,
					Field:  field,
					Values: values,
				})
			}
		}
	}
	return helpers
}

func applyMatchFieldValidation(helpers []matchFieldHelper, errors Errors) {
	for _, h := range helpers {
		fieldValue := fmt.Sprint(h.Values[h.Field])
		otherValue := fmt.Sprint(h.Values[h.Rule.Other])
		if fieldValue != otherValue {
			if errors == nil {
				errors = Errors{}
			}
			errors.Add(h.Field, h.Field+" must match "+h.Rule.Other)
		}
	}
}

func ValidateStruct(input interface{}, schema Schema) Errors {
	values := Flatten(input)
	errors := Validate(values, schema)
	helpers := collectMatchFieldHelpers(schema, values)
	applyMatchFieldValidation(helpers, errors)
	return errors
}

func ValidateTaggedStruct(input interface{}) Errors {
	schema, err := SchemaFromStruct(input)
	if err != nil {
		out := Errors{}
		out.Add("_schema", err.Error())
		return out
	}
	return ValidateStruct(input, schema)
}

func SchemaFromStruct(input interface{}) (Schema, error) {
	typ := reflect.TypeOf(input)
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ == nil || typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("validation schema source must be a struct")
	}
	schema := Schema{}
	for idx := 0; idx < typ.NumField(); idx++ {
		field := typ.Field(idx)
		if !field.IsExported() {
			continue
		}
		name := firstNonEmpty(tagName(field.Tag.Get("form"), "name"), field.Name)
		rules := ParseRules(field.Tag.Get("validate"))
		if len(rules) == 0 {
			continue
		}
		schema[name] = rules
	}
	return schema, nil
}

func ParseRules(raw string) []Rule {
	items := splitTagItems(raw)
	rules := []Rule{}
	for _, item := range items {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			key = item
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		switch key {
		case "", "-":
			continue
		case "required":
			rules = append(rules, Required())
		case "email":
			rules = append(rules, Email())
		case "minlength", "min_length":
			if number, err := strconv.Atoi(value); err == nil {
				rules = append(rules, MinLength(number))
			}
		case "maxlength", "max_length":
			if number, err := strconv.Atoi(value); err == nil {
				rules = append(rules, MaxLength(number))
			}
		case "min":
			if number, err := strconv.ParseFloat(value, 64); err == nil {
				rules = append(rules, Min(number))
			}
		case "max":
			if number, err := strconv.ParseFloat(value, 64); err == nil {
				rules = append(rules, Max(number))
			}
		case "pattern", "regex":
			rules = append(rules, Pattern(value))
		case "oneof", "one_of":
			rules = append(rules, OneOf(splitRuleValues(value)...))
		case "match", "matchfield", "match_field":
			rules = append(rules, MatchField(value))
		}
	}
	return rules
}

func Flatten(input interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	flattenInto(out, "", reflect.ValueOf(input))
	return out
}

func flattenInto(out map[string]interface{}, prefix string, value reflect.Value) {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}
	if !value.IsValid() {
		return
	}
	if value.Kind() != reflect.Struct {
		if prefix != "" {
			out[prefix] = value.Interface()
		}
		return
	}
	typ := value.Type()
	for idx := 0; idx < value.NumField(); idx++ {
		field := typ.Field(idx)
		if !field.IsExported() {
			continue
		}
		name := firstNonEmpty(tagName(field.Tag.Get("form"), "name"), field.Name)
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		child := value.Field(idx)
		if child.Kind() == reflect.Struct && field.Type.PkgPath() != "time" {
			flattenInto(out, path, child)
			continue
		}
		out[path] = child.Interface()
		out[name] = child.Interface()
	}
}

func numericValue(value interface{}) (float64, bool) {
	switch current := value.(type) {
	case int:
		return float64(current), true
	case int64:
		return float64(current), true
	case float64:
		return current, true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(current), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func trimNumeric(value float64) string {
	if float64(int64(value)) == value {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func splitTagItems(raw string) []string {
	out := []string{}
	for _, item := range strings.Split(strings.TrimSpace(raw), ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func splitRuleValues(raw string) []string {
	parts := strings.FieldsFunc(strings.TrimSpace(raw), func(r rune) bool { return r == '|' || r == ';' })
	out := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func tagName(raw string, key string) string {
	for _, item := range splitTagItems(raw) {
		left, right, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(key)) {
			return strings.Trim(strings.TrimSpace(right), `"`)
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
