package gmcorevalidation

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	gmerr "github.com/gmcorenet/gmcore-error"
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

func (e Errors) All(field string) []string {
	return e[field]
}

func (e Errors) Get(field string) []string {
	return e[field]
}

func (e Errors) Has(field string) bool {
	return len(e[field]) > 0
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
type OptionalRule struct{}
type NotBlankRule struct{}
type NotNullRule struct{}
type BlankRule struct{}
type IsNullRule struct{}
type IsTrueRule struct{}
type IsFalseRule struct{}
type TypeRule struct{ Expected string }

type EmailRule struct{}
type UrlRule struct{}
type UriRule struct{}
type UuidRule struct{}
type IpRule struct{}
type Ipv4Rule struct{}
type Ipv6Rule struct{}
type JsonRule struct{}
type SlugRule struct{}

type LengthRule struct{ Min, Max int }
type MinLengthRule struct{ Value int }
type MaxLengthRule struct{ Value int }
type LengthExactRule struct{ Value int }

type RangeRule struct{ Min, Max float64 }
type MinRule struct{ Value float64 }
type MaxRule struct{ Value float64 }

type EqualToRule struct{ Field string }
type NotEqualToRule struct{ Field string }
type IdenticalToRule struct{ Field string }
type NotIdenticalToRule struct{ Field string }

type LessThanRule struct{ Value float64 }
type LessThanOrEqualRule struct{ Value float64 }
type GreaterThanRule struct{ Value float64 }
type GreaterThanOrEqualRule struct{ Value float64 }

type InRule struct{ Values []string }
type NotInRule struct{ Values []string }

type RegexRule struct{ Pattern *regexp.Regexp }
type NotRegexRule struct{ Pattern *regexp.Regexp }

type CountRule struct{ Min, Max int }
type CountMinRule struct{ Value int }
type CountMaxRule struct{ Value int }

type AlphaRule struct{}
type AlnumRule struct{}
type DigitRule struct{}
type AlphaDigitRule struct{}

type LowercaseRule struct{}
type UppercaseRule struct{}

type CountryCodeRule struct{ Type string }
type LanguageCodeRule struct{}
type LocaleCodeRule struct{}
type CurrencyRule struct{}
type BicRule struct{}
type IbanRule struct{}

type CardSchemeRule struct{ Schemes []string }
type LuhnRule struct{}

type IsbnRule struct{}
type IssnRule struct{}

type HostnameRule struct{}
type HostRule struct{}
type ProtocolRule struct{}

type FileRule struct {
	MaxSize    int64
	MimeTypes  []string
	Extensions []string
}
type ImageRule struct {
	MaxSize   int64
	MinWidth  int
	MaxWidth  int
	MinHeight int
	MaxHeight int
}
type MimeTypeRule struct{ Types []string }
type ExtensionRule struct{ Extensions []string }

type PasswordStrengthRule struct{ MinStrength int }

type TimezoneRule struct{}
type DateRule struct{}
type TimeRule struct{}
type DateTimeRule struct{}

type WhenRule struct {
	Field   string
	Value   interface{}
	Rules   []Rule
	ElseRules []Rule
}

var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
var urlPattern = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
var ipv4Pattern = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
var ipv6Pattern = regexp.MustCompile(`^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`)
var uriPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.\-]*:`)
var alphanumPattern = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
var alphaPattern = regexp.MustCompile(`^[a-zA-Z]+$`)
var digitPattern = regexp.MustCompile(`^[0-9]+$`)

func Required() Rule { return RequiredRule{} }
func Optional() Rule { return OptionalRule{} }
func NotBlank() Rule { return NotBlankRule{} }
func NotNull() Rule { return NotNullRule{} }
func Blank() Rule { return BlankRule{} }
func IsNull() Rule { return IsNullRule{} }
func IsTrue() Rule { return IsTrueRule{} }
func IsFalse() Rule { return IsFalseRule{} }
func Type(expected string) Rule { return TypeRule{Expected: expected} }

func Email() Rule { return EmailRule{} }
func Url() Rule { return UrlRule{} }
func Uri() Rule { return UriRule{} }
func Uuid() Rule { return UuidRule{} }
func Ip() Rule { return IpRule{} }
func Ipv4() Rule { return Ipv4Rule{} }
func Ipv6() Rule { return Ipv6Rule{} }
func Json() Rule { return JsonRule{} }
func Slug() Rule { return SlugRule{} }

func Length(min, max int) Rule { return LengthRule{Min: min, Max: max} }
func MinLength(value int) Rule { return MinLengthRule{Value: value} }
func MaxLength(value int) Rule { return MaxLengthRule{Value: value} }
func LengthExact(value int) Rule { return LengthExactRule{Value: value} }

func Range(min, max float64) Rule { return RangeRule{Min: min, Max: max} }
func Min(value float64) Rule { return MinRule{Value: value} }
func Max(value float64) Rule { return MaxRule{Value: value} }

func EqualTo(field string) Rule { return EqualToRule{Field: field} }
func NotEqualTo(field string) Rule { return NotEqualToRule{Field: field} }
func IdenticalTo(field string) Rule { return IdenticalToRule{Field: field} }
func NotIdenticalTo(field string) Rule { return NotIdenticalToRule{Field: field} }

func LessThan(value float64) Rule { return LessThanRule{Value: value} }
func LessThanOrEqual(value float64) Rule { return LessThanOrEqualRule{Value: value} }
func GreaterThan(value float64) Rule { return GreaterThanRule{Value: value} }
func GreaterThanOrEqual(value float64) Rule { return GreaterThanOrEqualRule{Value: value} }

func In(values ...string) Rule { return InRule{Values: values} }
func NotIn(values ...string) Rule { return NotInRule{Values: values} }

func Regex(pattern string) Rule {
	compiled, _ := regexp.Compile(pattern)
	return RegexRule{Pattern: compiled}
}
func NotRegex(pattern string) Rule {
	compiled, _ := regexp.Compile(pattern)
	return NotRegexRule{Pattern: compiled}
}

func Count(min, max int) Rule { return CountRule{Min: min, Max: max} }
func CountMin(value int) Rule { return CountMinRule{Value: value} }
func CountMax(value int) Rule { return CountMaxRule{Value: value} }

func Alpha() Rule { return AlphaRule{} }
func Alnum() Rule { return AlnumRule{} }
func Digit() Rule { return DigitRule{} }
func AlphaDigit() Rule { return AlphaDigitRule{} }
func Lowercase() Rule { return LowercaseRule{} }
func Uppercase() Rule { return UppercaseRule{} }

func CountryCode() Rule { return CountryCodeRule{Type: "alpha2"} }
func LanguageCode() Rule { return LanguageCodeRule{} }
func LocaleCode() Rule { return LocaleCodeRule{} }
func Currency() Rule { return CurrencyRule{} }
func Bic() Rule { return BicRule{} }
func Iban() Rule { return IbanRule{} }

func CardScheme(schemes ...string) Rule { return CardSchemeRule{Schemes: schemes} }
func Luhn() Rule { return LuhnRule{} }

func Isbn() Rule { return IsbnRule{} }
func Issn() Rule { return IssnRule{} }

func Hostname() Rule { return HostnameRule{} }
func Host() Rule { return HostRule{} }
func Protocol() Rule { return ProtocolRule{} }

func File(maxSize int64, mimeTypes, extensions []string) Rule {
	return FileRule{MaxSize: maxSize, MimeTypes: mimeTypes, Extensions: extensions}
}
func Image(maxSize int64, minWidth, maxWidth, minHeight, maxHeight int) Rule {
	return ImageRule{MaxSize: maxSize, MinWidth: minWidth, MaxWidth: maxWidth, MinHeight: minHeight, MaxHeight: maxHeight}
}

func PasswordStrength(minStrength int) Rule { return PasswordStrengthRule{MinStrength: minStrength} }

func Timezone() Rule { return TimezoneRule{} }
func Date() Rule { return DateRule{} }
func Time() Rule { return TimeRule{} }
func DateTime() Rule { return DateTimeRule{} }

func When(field string, value interface{}, rules []Rule, elseRules []Rule) Rule {
	return WhenRule{Field: field, Value: value, Rules: rules, ElseRules: elseRules}
}

func (r RequiredRule) Validate(field string, value interface{}) string {
	if value == nil || isEmptyValue(value) {
		return field + ": this field is required"
	}
	return ""
}

func (r OptionalRule) Validate(field string, value interface{}) string {
	return ""
}

func (r NotBlankRule) Validate(field string, value interface{}) string {
	str := strings.TrimSpace(fmt.Sprint(value))
	if str == "" {
		return field + ": this field cannot be blank"
	}
	return ""
}

func (r NotNullRule) Validate(field string, value interface{}) string {
	if value == nil {
		return field + ": this field cannot be null"
	}
	return ""
}

func (r BlankRule) Validate(field string, value interface{}) string {
	str := strings.TrimSpace(fmt.Sprint(value))
	if str != "" {
		return field + ": this field must be blank"
	}
	return ""
}

func (r IsNullRule) Validate(field string, value interface{}) string {
	if value != nil {
		return field + ": this field must be null"
	}
	return ""
}

func (r IsTrueRule) Validate(field string, value interface{}) string {
	if !isTruthy(value) {
		return field + ": this field must be true"
	}
	return ""
}

func (r IsFalseRule) Validate(field string, value interface{}) string {
	if isTruthy(value) {
		return field + ": this field must be false"
	}
	return ""
}

func (r TypeRule) Validate(field string, value interface{}) string {
	if value == nil {
		return ""
	}
	switch r.Expected {
	case "string":
		_, ok := value.(string)
		if !ok {
			return field + ": expected string"
		}
	case "int", "integer":
		switch value.(type) {
		case int, int8, int16, int32, int64:
		default:
			return field + ": expected integer"
		}
	case "float", "number":
		switch value.(type) {
		case float32, float64, int, int8, int16, int32, int64:
		default:
			return field + ": expected number"
		}
	case "bool", "boolean":
		switch value.(type) {
		case bool:
		default:
			return field + ": expected boolean"
		}
	case "array", "slice":
		rv := reflect.ValueOf(value)
		if rv.Kind() != reflect.Array && rv.Kind() != reflect.Slice {
			return field + ": expected array"
		}
	case "map":
		rv := reflect.ValueOf(value)
		if rv.Kind() != reflect.Map {
			return field + ": expected map"
		}
	}
	return ""
}

func (r EmailRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	if !emailPattern.MatchString(fmt.Sprint(value)) {
		return field + ": invalid email address"
	}
	return ""
}

func (r UrlRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	if !urlPattern.MatchString(fmt.Sprint(value)) {
		return field + ": invalid URL"
	}
	return ""
}

func (r UuidRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	if !uuidPattern.MatchString(fmt.Sprint(value)) {
		return field + ": invalid UUID"
	}
	return ""
}

func (r IpRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := fmt.Sprint(value)
	if !ipv4Pattern.MatchString(v) && !ipv6Pattern.MatchString(v) {
		return field + ": invalid IP address"
	}
	return ""
}

func (r Ipv4Rule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := fmt.Sprint(value)
	if !ipv4Pattern.MatchString(v) {
		return field + ": invalid IPv4 address"
	}
	return ""
}

func (r Ipv6Rule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := fmt.Sprint(value)
	if !ipv6Pattern.MatchString(v) {
		return field + ": invalid IPv6 address"
	}
	return ""
}

func (r UriRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := fmt.Sprint(value)
	if !uriPattern.MatchString(v) {
		return field + ": invalid URI"
	}
	return ""
}

func (r JsonRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := strings.TrimSpace(fmt.Sprint(value))
	var js interface{}
	if err := json.Unmarshal([]byte(v), &js); err != nil {
		return field + ": invalid JSON"
	}
	return ""
}

func (r SlugRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	if !slugPattern.MatchString(fmt.Sprint(value)) {
		return field + ": invalid slug (only lowercase letters, numbers and hyphens)"
	}
	return ""
}

func (r AlphaRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := fmt.Sprint(value)
	for _, r := range v {
		if !unicode.IsLetter(r) {
			return field + ": must contain only letters"
		}
	}
	return ""
}

func (r AlnumRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	if !alphanumPattern.MatchString(fmt.Sprint(value)) {
		return field + ": must contain only letters and numbers"
	}
	return ""
}

func (r DigitRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	if !digitPattern.MatchString(fmt.Sprint(value)) {
		return field + ": must contain only digits"
	}
	return ""
}

func (r LengthRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	length := int64(0)
	switch v := value.(type) {
	case string:
		length = int64(len(v))
	case []interface{}, map[string]interface{}:
		rv := reflect.ValueOf(value)
		length = int64(rv.Len())
	default:
		return ""
	}
	if r.Min > 0 && length < int64(r.Min) {
		return fmt.Sprintf("%s: length must be at least %d", field, r.Min)
	}
	if r.Max > 0 && length > int64(r.Max) {
		return fmt.Sprintf("%s: length must be at most %d", field, r.Max)
	}
	return ""
}

func (r MinLengthRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	length := int64(len(fmt.Sprint(value)))
	if length < int64(r.Value) {
		return fmt.Sprintf("%s: length must be at least %d", field, r.Value)
	}
	return ""
}

func (r MaxLengthRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	length := int64(len(fmt.Sprint(value)))
	if length > int64(r.Value) {
		return fmt.Sprintf("%s: length must be at most %d", field, r.Value)
	}
	return ""
}

func (r RangeRule) Validate(field string, value interface{}) string {
	num, ok := toFloat64(value)
	if !ok || r.isEmpty(value) {
		return ""
	}
	if r.Min != 0 && num < r.Min {
		return fmt.Sprintf("%s: value must be at least %v", field, r.Min)
	}
	if r.Max != 0 && num > r.Max {
		return fmt.Sprintf("%s: value must be at most %v", field, r.Max)
	}
	return ""
}

func (r MinRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	num, ok := toFloat64(value)
	if !ok {
		return ""
	}
	if num < r.Value {
		return fmt.Sprintf("%s: value must be at least %v", field, r.Value)
	}
	return ""
}

func (r MaxRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	num, ok := toFloat64(value)
	if !ok {
		return ""
	}
	if num > r.Value {
		return fmt.Sprintf("%s: value must be at most %v", field, r.Value)
	}
	return ""
}

func (r InRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := fmt.Sprint(value)
	for _, allowed := range r.Values {
		if v == allowed {
			return ""
		}
	}
	return field + ": value must be one of: " + strings.Join(r.Values, ", ")
}

func (r NotInRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := fmt.Sprint(value)
	for _, banned := range r.Values {
		if v == banned {
			return field + ": value cannot be one of: " + strings.Join(r.Values, ", ")
		}
	}
	return ""
}

func (r RegexRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) || r.Pattern == nil {
		return ""
	}
	if !r.Pattern.MatchString(fmt.Sprint(value)) {
		return field + ": invalid format"
	}
	return ""
}

func (r CountRule) Validate(field string, value interface{}) string {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() || (rv.Kind() != reflect.Array && rv.Kind() != reflect.Slice) {
		return ""
	}
	length := rv.Len()
	if r.Min > 0 && length < r.Min {
		return fmt.Sprintf("%s: count must be at least %d", field, r.Min)
	}
	if r.Max > 0 && length > r.Max {
		return fmt.Sprintf("%s: count must be at most %d", field, r.Max)
	}
	return ""
}

func (r BicRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := strings.ToUpper(fmt.Sprint(value))
	if len(v) != 8 && len(v) != 11 {
		return field + ": invalid BIC/SWIFT code"
	}
	bicPattern := regexp.MustCompile(`^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$`)
	if !bicPattern.MatchString(v) {
		return field + ": invalid BIC/SWIFT code"
	}
	return ""
}

func (r IbanRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(fmt.Sprint(value), " ", ""), "-", ""))
	if len(v) < 15 || len(v) > 34 {
		return field + ": invalid IBAN"
	}
	return ""
}

func (r CardSchemeRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := strings.ReplaceAll(strings.ReplaceAll(fmt.Sprint(value), " ", ""), "-", "")
	for _, scheme := range r.Schemes {
		switch strings.ToLower(scheme) {
		case "visa":
			if len(v) >= 2 && v[:2] == "4" && (len(v) == 13 || len(v) == 16) {
				return ""
			}
		case "mastercard":
			mcPrefixes := []string{"51", "52", "53", "54", "55", "2221-2720"}
			for _, prefix := range mcPrefixes {
				if len(v) >= 2 && strings.HasPrefix(v, prefix) && len(v) == 16 {
					return ""
				}
			}
		case "amex":
			if len(v) == 15 && (strings.HasPrefix(v, "34") || strings.HasPrefix(v, "37")) {
				return ""
			}
		case "discover":
			if len(v) == 16 && strings.HasPrefix(v, "6011") {
				return ""
			}
		}
	}
	return field + ": invalid card number for accepted schemes"
}

func (r LuhnRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := strings.ReplaceAll(fmt.Sprint(value), " ", "")
	if !digitPattern.MatchString(v) {
		return field + ": invalid card number"
	}
	sum := 0
	alt := false
	for i := len(v) - 1; i >= 0; i-- {
		n := int(v[i] - '0')
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	if sum%10 != 0 {
		return field + ": invalid card number (failed Luhn check)"
	}
	return ""
}

func (r CountryCodeRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := strings.ToUpper(fmt.Sprint(value))
	switch r.Type {
	case "alpha2":
		if len(v) != 2 {
			return field + ": invalid country code (alpha-2)"
		}
	case "alpha3":
		if len(v) != 3 {
			return field + ": invalid country code (alpha-3)"
		}
	}
	return ""
}

func (r IsbnRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := strings.ReplaceAll(strings.ReplaceAll(fmt.Sprint(value), "-", ""), " ", "")
	if len(v) == 13 {
		return r.validateIsbn13(v, field)
	} else if len(v) == 10 {
		return r.validateIsbn10(v, field)
	}
	return field + ": invalid ISBN"
}

func (r IsbnRule) validateIsbn13(v, field string) string {
	sum := 0
	for i, c := range v {
		d := int(c - '0')
		if i%2 == 0 {
			d *= 1
		} else {
			d *= 3
		}
		sum += d
	}
	if sum%10 == 0 {
		return ""
	}
	return field + ": invalid ISBN"
}

func (r IsbnRule) validateIsbn10(v, field string) string {
	sum := 0
	for i, c := range v {
		d := int(c - '0')
		if c == 'X' {
			d = 10
		}
		sum += d * (10 - i)
	}
	if sum%11 == 0 {
		return ""
	}
	return field + ": invalid ISBN"
}

func (r IssnRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := strings.ReplaceAll(strings.ReplaceAll(fmt.Sprint(value), "-", ""), " ", "")
	if len(v) == 8 {
		return r.validateIssn8(v, field)
	}
	return field + ": invalid ISSN"
}

func (r IssnRule) validateIssn8(v, field string) string {
	sum := 0
	for i, c := range v {
		d := int(c - '0')
		if c == 'X' {
			d = 10
		}
		sum += d * (8 - i)
	}
	if sum%11 == 0 {
		return ""
	}
	return field + ": invalid ISSN"
}

func (r PasswordStrengthRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := fmt.Sprint(value)
	strength := 0
	if len(v) >= 8 {
		strength++
	}
	if len(v) >= 12 {
		strength++
	}
	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSpecial := false
	for _, c := range v {
		switch {
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsDigit(c):
			hasDigit = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}
	if hasLower && hasUpper {
		strength++
	}
	if hasDigit {
		strength++
	}
	if hasSpecial {
		strength++
	}
	if strength < r.MinStrength {
		return field + ": password too weak"
	}
	return ""
}

func (r TimezoneRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := fmt.Sprint(value)
	_, err := time.LoadLocation(v)
	if err != nil {
		return field + ": invalid timezone"
	}
	return ""
}

func (r DateRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := fmt.Sprint(value)
	formats := []string{time.RFC3339, "2006-01-02", "01/02/2006", "02-01-2006"}
	for _, format := range formats {
		if _, err := time.Parse(format, v); err == nil {
			return ""
		}
	}
	return field + ": invalid date"
}

func (r TimeRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := fmt.Sprint(value)
	formats := []string{time.RFC3339, "15:04:05", "3:04 PM", "15:04"}
	for _, format := range formats {
		if _, err := time.Parse(format, v); err == nil {
			return ""
		}
	}
	return field + ": invalid time"
}

func (r WhenRule) Validate(field string, value interface{}) string {
	conditionValue := getFieldValue(field)
	if fmt.Sprint(conditionValue) == fmt.Sprint(r.Value) {
		for _, rule := range r.Rules {
			if err := rule.Validate(field, value); err != "" {
				return err
			}
		}
	} else if r.ElseRules != nil {
		for _, rule := range r.ElseRules {
			if err := rule.Validate(field, value); err != "" {
				return err
			}
		}
	}
	return ""
}

func (r FileRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	return ""
}

func (r ImageRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	return ""
}

func (r LowercaseRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := fmt.Sprint(value)
	if v != strings.ToLower(v) {
		return field + ": must be lowercase"
	}
	return ""
}

func (r UppercaseRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := fmt.Sprint(value)
	if v != strings.ToUpper(v) {
		return field + ": must be uppercase"
	}
	return ""
}

func (r AlphaDigitRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := fmt.Sprint(value)
	for _, c := range v {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
			return field + ": must contain only letters and numbers"
		}
	}
	return ""
}

func (r LessThanRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	num, ok := toFloat64(value)
	if !ok {
		return ""
	}
	if num >= r.Value {
		return fmt.Sprintf("%s: must be less than %v", field, r.Value)
	}
	return ""
}

func (r LessThanOrEqualRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	num, ok := toFloat64(value)
	if !ok {
		return ""
	}
	if num > r.Value {
		return fmt.Sprintf("%s: must be less than or equal to %v", field, r.Value)
	}
	return ""
}

func (r GreaterThanRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	num, ok := toFloat64(value)
	if !ok {
		return ""
	}
	if num <= r.Value {
		return fmt.Sprintf("%s: must be greater than %v", field, r.Value)
	}
	return ""
}

func (r GreaterThanOrEqualRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	num, ok := toFloat64(value)
	if !ok {
		return ""
	}
	if num < r.Value {
		return fmt.Sprintf("%s: must be greater than or equal to %v", field, r.Value)
	}
	return ""
}

func (r EqualToRule) Validate(field string, value interface{}) string {
	return ""
}

func (r NotEqualToRule) Validate(field string, value interface{}) string {
	return ""
}

func (r IdenticalToRule) Validate(field string, value interface{}) string {
	return ""
}

func (r NotIdenticalToRule) Validate(field string, value interface{}) string {
	return ""
}

func (r LengthExactRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	length := int64(len(fmt.Sprint(value)))
	if length != int64(r.Value) {
		return fmt.Sprintf("%s: length must be exactly %d", field, r.Value)
	}
	return ""
}

func (r CountMinRule) Validate(field string, value interface{}) string {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() || (rv.Kind() != reflect.Array && rv.Kind() != reflect.Slice) {
		return ""
	}
	if rv.Len() < r.Value {
		return fmt.Sprintf("%s: count must be at least %d", field, r.Value)
	}
	return ""
}

func (r CountMaxRule) Validate(field string, value interface{}) string {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() || (rv.Kind() != reflect.Array && rv.Kind() != reflect.Slice) {
		return ""
	}
	if rv.Len() > r.Value {
		return fmt.Sprintf("%s: count must be at most %d", field, r.Value)
	}
	return ""
}

func (r NotRegexRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) || r.Pattern == nil {
		return ""
	}
	if r.Pattern.MatchString(fmt.Sprint(value)) {
		return field + ": must not match pattern"
	}
	return ""
}

func (r LanguageCodeRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := strings.ToLower(fmt.Sprint(value))
	if len(v) != 2 && len(v) != 3 {
		return field + ": invalid language code"
	}
	return ""
}

func (r LocaleCodeRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := strings.ReplaceAll(strings.ReplaceAll(fmt.Sprint(value), "_", "-"), " ", "")
	parts := strings.Split(v, "-")
	if len(parts) < 1 || len(parts) > 2 {
		return field + ": invalid locale code"
	}
	return ""
}

func (r CurrencyRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := strings.ToUpper(fmt.Sprint(value))
	if len(v) != 3 {
		return field + ": invalid currency code (ISO 4217)"
	}
	return ""
}

func (r HostnameRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := fmt.Sprint(value)
	hostnamePattern := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	if !hostnamePattern.MatchString(v) && v != "localhost" {
		return field + ": invalid hostname"
	}
	return ""
}

func (r HostRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	return ""
}

func (r ProtocolRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := strings.ToLower(fmt.Sprint(value))
	if v != "http" && v != "https" && v != "ftp" && v != "sftp" {
		return field + ": invalid protocol"
	}
	return ""
}

func (r DateTimeRule) Validate(field string, value interface{}) string {
	if r.isEmpty(value) {
		return ""
	}
	v := fmt.Sprint(value)
	_, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return field + ": invalid datetime (expected RFC3339)"
	}
	return ""
}

func isEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	case int, int8, int16, int32, int64:
		return v == 0
	case float32, float64:
		return v == 0
	case bool:
		return !v
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
		return rv.Len() == 0
	case reflect.Ptr, reflect.Interface:
		if rv.IsNil() {
			return true
		}
	}
	return false
}

func isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case int, int8, int16, int32, int64:
		return v != 0
	case float32, float64:
		return v != 0
	case string:
		return v != ""
	}
	return true
}

func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func getFieldValue(field string) interface{} {
	return nil
}

func (r *RequiredRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *OptionalRule) isEmpty(v interface{}) bool { return false }
func (r *NotBlankRule) isEmpty(v interface{}) bool { return false }
func (r *NotNullRule) isEmpty(v interface{}) bool { return v == nil }
func (r *BlankRule) isEmpty(v interface{}) bool { return false }
func (r *IsNullRule) isEmpty(v interface{}) bool { return v != nil }
func (r *IsTrueRule) isEmpty(v interface{}) bool { return false }
func (r *IsFalseRule) isEmpty(v interface{}) bool { return false }
func (r *TypeRule) isEmpty(v interface{}) bool { return false }
func (r *EmailRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *UrlRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *UriRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *UuidRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *IpRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *Ipv4Rule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *Ipv6Rule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *JsonRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *SlugRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *LengthRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *MinLengthRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *MaxLengthRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *LengthExactRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *RangeRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *MinRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *MaxRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *InRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *NotInRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *RegexRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *NotRegexRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *CountRule) isEmpty(v interface{}) bool { return false }
func (r *CountMinRule) isEmpty(v interface{}) bool { return false }
func (r *CountMaxRule) isEmpty(v interface{}) bool { return false }
func (r *AlphaRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *AlnumRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *DigitRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *AlphaDigitRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *LowercaseRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *UppercaseRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *CountryCodeRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *LanguageCodeRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *LocaleCodeRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *CurrencyRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *BicRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *IbanRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *CardSchemeRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *LuhnRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *IsbnRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *IssnRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *HostnameRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *HostRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *ProtocolRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *PasswordStrengthRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *TimezoneRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *DateRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *TimeRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *DateTimeRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *FileRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *ImageRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *EqualToRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *NotEqualToRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *IdenticalToRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *NotIdenticalToRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *LessThanRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *LessThanOrEqualRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *GreaterThanRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *GreaterThanOrEqualRule) isEmpty(v interface{}) bool { return isEmptyValue(v) }
func (r *WhenRule) isEmpty(v interface{}) bool { return false }
