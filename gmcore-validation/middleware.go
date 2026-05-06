package gmcore_validation

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gmcorenet/framework/container"
	"github.com/gmcorenet/framework/router"
	"github.com/gmcorenet/framework/routing"
)

func init() {
	routing.RegisterMiddlewareProvider(func(ctr *container.Container, r *router.Router) (func(http.Handler) http.Handler, bool) {
		return NewValidationMiddleware(ctr), true
	})
}

type ValidationAnnotation struct {
	Rules string
}

func NewValidationMiddleware(ctr *container.Container) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			controller, action := parseValidationPath(r.URL.Path)
			key := "validate." + controller + "." + action

			raw, err := ctr.Get(key)
			if err != nil || raw == nil {
				next.ServeHTTP(w, r)
				return
			}

			rulesStr := extractRules(raw)
			if rulesStr == "" {
				next.ServeHTTP(w, r)
				return
			}

			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			schema := parseRulesString(rulesStr)
			if len(schema) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeValidationErrors(w, Errors{"_": []string{"failed to read request body"}})
				return
			}

			values := make(map[string]interface{})
			if len(body) > 0 {
				if err := json.Unmarshal(body, &values); err != nil {
					writeValidationErrors(w, Errors{"_": []string{"invalid JSON body"}})
					return
				}
			}

			validationErrors := Validate(values, schema)
			if validationErrors != nil && validationErrors.HasAny() {
				writeValidationErrors(w, validationErrors)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func parseValidationPath(path string) (controller, action string) {
	path = strings.Trim(path, "/")
	if path == "" {
		return "index", "index"
	}
	parts := strings.Split(path, "/")
	controller = parts[0]
	if controller == "" {
		controller = "index"
	}
	if len(parts) == 1 {
		action = "index"
	} else {
		action = parts[1]
	}
	if action == "" {
		action = "index"
	}
	return strings.ToLower(controller), strings.ToLower(action)
}

func extractRules(raw interface{}) string {
	switch v := raw.(type) {
	case string:
		return v
	case *ValidationAnnotation:
		return v.Rules
	case ValidationAnnotation:
		return v.Rules
	default:
		return ""
	}
}

func parseRulesString(rulesStr string) Schema {
	schema := make(Schema)
	for _, fieldDef := range splitFields(rulesStr) {
		fieldDef = strings.TrimSpace(fieldDef)
		if fieldDef == "" {
			continue
		}
		colonIdx := strings.Index(fieldDef, ":")
		if colonIdx < 0 {
			continue
		}
		fieldName := strings.TrimSpace(fieldDef[:colonIdx])
		rulesPart := strings.TrimSpace(fieldDef[colonIdx+1:])
		if fieldName == "" || rulesPart == "" {
			continue
		}
		rules := parseRules(rulesPart)
		if len(rules) > 0 {
			schema[fieldName] = rules
		}
	}
	return schema
}

func splitFields(rulesStr string) []string {
	fields := make([]string, 0)
	current := ""
	depth := 0
	for _, ch := range rulesStr {
		switch ch {
		case '(':
			depth++
			current += string(ch)
		case ')':
			depth--
			current += string(ch)
		case ',':
			if depth == 0 {
				fields = append(fields, current)
				current = ""
			} else {
				current += string(ch)
			}
		default:
			current += string(ch)
		}
	}
	if current != "" {
		fields = append(fields, current)
	}
	return fields
}

func parseRules(rulesPart string) []Rule {
	rules := make([]Rule, 0)
	tokens := splitRules(rulesPart)
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		rule := parseSingleRule(token)
		if rule != nil {
			rules = append(rules, rule)
		}
	}
	return rules
}

func splitRules(rulesPart string) []string {
	tokens := make([]string, 0)
	current := ""
	depth := 0
	for _, ch := range rulesPart {
		switch ch {
		case '(':
			depth++
			current += string(ch)
		case ')':
			depth--
			current += string(ch)
		case '|':
			if depth == 0 {
				tokens = append(tokens, current)
				current = ""
			} else {
				current += string(ch)
			}
		default:
			current += string(ch)
		}
	}
	if current != "" {
		tokens = append(tokens, current)
	}
	return tokens
}

func parseSingleRule(token string) Rule {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}

	if token == "required" {
		return Required()
	}
	if token == "optional" {
		return Optional()
	}
	if token == "notBlank" || token == "not_blank" || token == "NotBlank" {
		return NotBlank()
	}
	if token == "notNull" || token == "not_null" || token == "NotNull" {
		return NotNull()
	}
	if token == "blank" || token == "Blank" {
		return Blank()
	}
	if token == "null" || token == "isNull" || token == "is_null" || token == "IsNull" {
		return IsNull()
	}
	if token == "true" || token == "isTrue" || token == "is_true" || token == "IsTrue" {
		return IsTrue()
	}
	if token == "false" || token == "isFalse" || token == "is_false" || token == "IsFalse" {
		return IsFalse()
	}
	if token == "email" || token == "Email" {
		return Email()
	}
	if token == "url" || token == "Url" {
		return Url()
	}
	if token == "uri" || token == "Uri" {
		return Uri()
	}
	if token == "uuid" || token == "Uuid" {
		return Uuid()
	}
	if token == "ip" || token == "Ip" {
		return Ip()
	}
	if token == "ipv4" || token == "Ipv4" {
		return Ipv4()
	}
	if token == "ipv6" || token == "Ipv6" {
		return Ipv6()
	}
	if token == "json" || token == "Json" {
		return Json()
	}
	if token == "slug" || token == "Slug" {
		return Slug()
	}
	if token == "alpha" || token == "Alpha" {
		return Alpha()
	}
	if token == "alnum" || token == "Alnum" {
		return Alnum()
	}
	if token == "digit" || token == "Digit" {
		return Digit()
	}
	if token == "alphaDigit" || token == "alpha_digit" || token == "AlphaDigit" {
		return AlphaDigit()
	}
	if token == "lowercase" || token == "Lowercase" {
		return Lowercase()
	}
	if token == "uppercase" || token == "Uppercase" {
		return Uppercase()
	}
	if token == "countryCode" || token == "country_code" || token == "CountryCode" {
		return CountryCode()
	}
	if token == "languageCode" || token == "language_code" || token == "LanguageCode" {
		return LanguageCode()
	}
	if token == "localeCode" || token == "locale_code" || token == "LocaleCode" {
		return LocaleCode()
	}
	if token == "currency" || token == "Currency" {
		return Currency()
	}
	if token == "bic" || token == "Bic" {
		return Bic()
	}
	if token == "iban" || token == "Iban" {
		return Iban()
	}
	if token == "luhn" || token == "Luhn" {
		return Luhn()
	}
	if token == "isbn" || token == "Isbn" {
		return Isbn()
	}
	if token == "issn" || token == "Issn" {
		return Issn()
	}
	if token == "hostname" || token == "Hostname" {
		return Hostname()
	}
	if token == "host" || token == "Host" {
		return Host()
	}
	if token == "protocol" || token == "Protocol" {
		return Protocol()
	}
	if token == "timezone" || token == "Timezone" {
		return Timezone()
	}
	if token == "date" || token == "Date" {
		return Date()
	}
	if token == "time" || token == "Time" {
		return Time()
	}
	if token == "datetime" || token == "dateTime" || token == "date_time" || token == "DateTime" {
		return DateTime()
	}

	if strings.HasPrefix(token, "type=") {
		val := strings.TrimPrefix(token, "type=")
		return Type(strings.TrimSpace(val))
	}
	if strings.HasPrefix(token, "Type=") {
		val := strings.TrimPrefix(token, "Type=")
		return Type(strings.TrimSpace(val))
	}

	if strings.HasPrefix(token, "email(") && strings.HasSuffix(token, ")") {
		return Email()
	}

	if strings.HasPrefix(token, "minLength=") || strings.HasPrefix(token, "min_length=") || strings.HasPrefix(token, "MinLength=") {
		val := extractValue(token)
		if n, err := strconv.Atoi(val); err == nil {
			return MinLength(n)
		}
	}
	if strings.HasPrefix(token, "maxLength=") || strings.HasPrefix(token, "max_length=") || strings.HasPrefix(token, "MaxLength=") {
		val := extractValue(token)
		if n, err := strconv.Atoi(val); err == nil {
			return MaxLength(n)
		}
	}
	if strings.HasPrefix(token, "length=") || strings.HasPrefix(token, "Length=") {
		val := extractValue(token)
		parts := strings.Split(val, "-")
		if len(parts) == 2 {
			min, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
			max, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			return Length(min, max)
		}
	}
	if strings.HasPrefix(token, "lengthExact=") || strings.HasPrefix(token, "length_exact=") || strings.HasPrefix(token, "LengthExact=") {
		val := extractValue(token)
		if n, err := strconv.Atoi(val); err == nil {
			return LengthExact(n)
		}
	}

	if strings.HasPrefix(token, "min=") || strings.HasPrefix(token, "Min=") {
		val := extractValue(token)
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return Min(f)
		}
	}
	if strings.HasPrefix(token, "max=") || strings.HasPrefix(token, "Max=") {
		val := extractValue(token)
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return Max(f)
		}
	}
	if strings.HasPrefix(token, "range=") || strings.HasPrefix(token, "Range=") {
		val := extractValue(token)
		parts := strings.Split(val, "-")
		if len(parts) == 2 {
			min, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			max, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			return Range(min, max)
		}
	}

	if strings.HasPrefix(token, "oneof=") || strings.HasPrefix(token, "OneOf=") {
		val := extractValue(token)
		opts := strings.Split(val, "|")
		return In(opts...)
	}
	if strings.HasPrefix(token, "notIn=") || strings.HasPrefix(token, "not_in=") || strings.HasPrefix(token, "NotIn=") {
		val := extractValue(token)
		opts := strings.Split(val, "|")
		return NotIn(opts...)
	}

	if strings.HasPrefix(token, "regex=") || strings.HasPrefix(token, "Regex=") {
		val := extractValue(token)
		return Regex(val)
	}
	if strings.HasPrefix(token, "notRegex=") || strings.HasPrefix(token, "not_regex=") || strings.HasPrefix(token, "NotRegex=") {
		val := extractValue(token)
		return NotRegex(val)
	}

	if strings.HasPrefix(token, "cardScheme=") || strings.HasPrefix(token, "card_scheme=") || strings.HasPrefix(token, "CardScheme=") {
		val := extractValue(token)
		schemes := strings.Split(val, "|")
		return CardScheme(schemes...)
	}

	if strings.HasPrefix(token, "equalTo=") || strings.HasPrefix(token, "equal_to=") || strings.HasPrefix(token, "EqualTo=") {
		val := extractValue(token)
		return EqualTo(strings.TrimSpace(val))
	}
	if strings.HasPrefix(token, "notEqualTo=") || strings.HasPrefix(token, "not_equal_to=") || strings.HasPrefix(token, "NotEqualTo=") {
		val := extractValue(token)
		return NotEqualTo(strings.TrimSpace(val))
	}

	if strings.HasPrefix(token, "lessThan=") || strings.HasPrefix(token, "less_than=") || strings.HasPrefix(token, "LessThan=") {
		val := extractValue(token)
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return LessThan(f)
		}
	}
	if strings.HasPrefix(token, "lessThanOrEqual=") || strings.HasPrefix(token, "less_than_or_equal=") || strings.HasPrefix(token, "LessThanOrEqual=") {
		val := extractValue(token)
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return LessThanOrEqual(f)
		}
	}
	if strings.HasPrefix(token, "greaterThan=") || strings.HasPrefix(token, "greater_than=") || strings.HasPrefix(token, "GreaterThan=") {
		val := extractValue(token)
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return GreaterThan(f)
		}
	}
	if strings.HasPrefix(token, "greaterThanOrEqual=") || strings.HasPrefix(token, "greater_than_or_equal=") || strings.HasPrefix(token, "GreaterThanOrEqual=") {
		val := extractValue(token)
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return GreaterThanOrEqual(f)
		}
	}

	if strings.HasPrefix(token, "count=") || strings.HasPrefix(token, "Count=") {
		val := extractValue(token)
		parts := strings.Split(val, "-")
		if len(parts) == 2 {
			min, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
			max, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			return Count(min, max)
		}
	}
	if strings.HasPrefix(token, "countMin=") || strings.HasPrefix(token, "count_min=") || strings.HasPrefix(token, "CountMin=") {
		val := extractValue(token)
		if n, err := strconv.Atoi(val); err == nil {
			return CountMin(n)
		}
	}
	if strings.HasPrefix(token, "countMax=") || strings.HasPrefix(token, "count_max=") || strings.HasPrefix(token, "CountMax=") {
		val := extractValue(token)
		if n, err := strconv.Atoi(val); err == nil {
			return CountMax(n)
		}
	}

	if strings.HasPrefix(token, "passwordStrength=") || strings.HasPrefix(token, "password_strength=") || strings.HasPrefix(token, "PasswordStrength=") {
		val := extractValue(token)
		if n, err := strconv.Atoi(val); err == nil {
			return PasswordStrength(n)
		}
	}

	if strings.HasPrefix(token, "file(") && strings.HasSuffix(token, ")") {
		return parseFileRule(token)
	}
	if strings.HasPrefix(token, "image(") && strings.HasSuffix(token, ")") {
		return parseImageRule(token)
	}

	return nil
}

func extractValue(token string) string {
	eqIdx := strings.Index(token, "=")
	if eqIdx < 0 {
		return ""
	}
	return strings.TrimSpace(token[eqIdx+1:])
}

func parseFileRule(token string) Rule {
	inner := token[5 : len(token)-1]
	var maxSize int64
	var mimeTypes []string
	var extensions []string
	parts := strings.Split(inner, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "maxSize=") || strings.HasPrefix(part, "max_size=") {
			val := extractValue(part)
			if n, err := strconv.ParseInt(val, 10, 64); err == nil {
				maxSize = n
			}
		} else if strings.HasPrefix(part, "mimeTypes=") || strings.HasPrefix(part, "mime_types=") {
			val := extractValue(part)
			mimeTypes = strings.Split(val, "|")
		} else if strings.HasPrefix(part, "extensions=") {
			val := extractValue(part)
			extensions = strings.Split(val, "|")
		}
	}
	return File(maxSize, mimeTypes, extensions)
}

func parseImageRule(token string) Rule {
	inner := token[6 : len(token)-1]
	var maxSize int64
	minWidth := 0
	maxWidth := 0
	minHeight := 0
	maxHeight := 0
	parts := strings.Split(inner, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "maxSize=") || strings.HasPrefix(part, "max_size=") {
			val := extractValue(part)
			if n, err := strconv.ParseInt(val, 10, 64); err == nil {
				maxSize = n
			}
		} else if strings.HasPrefix(part, "minWidth=") || strings.HasPrefix(part, "min_width=") {
			val := extractValue(part)
			if n, err := strconv.Atoi(val); err == nil {
				minWidth = n
			}
		} else if strings.HasPrefix(part, "maxWidth=") || strings.HasPrefix(part, "max_width=") {
			val := extractValue(part)
			if n, err := strconv.Atoi(val); err == nil {
				maxWidth = n
			}
		} else if strings.HasPrefix(part, "minHeight=") || strings.HasPrefix(part, "min_height=") {
			val := extractValue(part)
			if n, err := strconv.Atoi(val); err == nil {
				minHeight = n
			}
		} else if strings.HasPrefix(part, "maxHeight=") || strings.HasPrefix(part, "max_height=") {
			val := extractValue(part)
			if n, err := strconv.Atoi(val); err == nil {
				maxHeight = n
			}
		}
	}
	return Image(maxSize, minWidth, maxWidth, minHeight, maxHeight)
}

func writeValidationErrors(w http.ResponseWriter, errors Errors) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    422,
		"errors":  errors,
		"message": "Validation failed",
	})
}
