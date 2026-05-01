package gmcoretemplating

import (
	"encoding/json"
	"fmt"
	"html/template"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type FilterFunc func(interface{}, ...interface{}) interface{}
type TestFunc func(interface{}, ...interface{}) bool

var (
	filterRegistryMu sync.RWMutex
	filterRegistry   = map[string]FilterFunc{}
	testRegistryMu   sync.RWMutex
	testRegistry     = map[string]TestFunc{}
)

func init() {
	RegisterFilter("default", func(value interface{}, args ...interface{}) interface{} {
		if !isEmptyValue(value) {
			return value
		}
		if len(args) == 0 {
			return ""
		}
		return args[0]
	})
	RegisterFilter("lower", func(value interface{}, _ ...interface{}) interface{} { return strings.ToLower(fmt.Sprint(value)) })
	RegisterFilter("upper", func(value interface{}, _ ...interface{}) interface{} { return strings.ToUpper(fmt.Sprint(value)) })
	RegisterFilter("title", func(value interface{}, _ ...interface{}) interface{} {
		return strings.Title(strings.TrimSpace(fmt.Sprint(value)))
	})
	RegisterFilter("trim", func(value interface{}, _ ...interface{}) interface{} { return strings.TrimSpace(fmt.Sprint(value)) })
	RegisterFilter("slug", func(value interface{}, _ ...interface{}) interface{} {
		out := strings.ToLower(strings.TrimSpace(fmt.Sprint(value)))
		replacer := strings.NewReplacer(" ", "-", "/", "-", "_", "-")
		return replacer.Replace(out)
	})
	RegisterFilter("join", func(value interface{}, args ...interface{}) interface{} {
		separator := ", "
		if len(args) > 0 {
			separator = fmt.Sprint(args[0])
		}
		items := []string{}
		for _, item := range toSlice(value) {
			items = append(items, fmt.Sprint(item))
		}
		return strings.Join(items, separator)
	})
	RegisterFilter("json", func(value interface{}, _ ...interface{}) interface{} { return mustJSON(value) })
	RegisterFilter("safe", func(value interface{}, _ ...interface{}) interface{} { return template.HTML(fmt.Sprint(value)) })
	RegisterFilter("safeHTML", func(value interface{}, _ ...interface{}) interface{} { return template.HTML(fmt.Sprint(value)) })
	RegisterFilter("length", func(value interface{}, _ ...interface{}) interface{} { return collectionLen(value) })
	RegisterFilter("replace", func(value interface{}, args ...interface{}) interface{} {
		out := fmt.Sprint(value)
		if len(args) == 1 {
			if mapping, ok := args[0].(map[string]interface{}); ok {
				for key, replacement := range mapping {
					out = strings.ReplaceAll(out, key, fmt.Sprint(replacement))
				}
				return out
			}
		}
		if len(args) >= 2 {
			return strings.ReplaceAll(out, fmt.Sprint(args[0]), fmt.Sprint(args[1]))
		}
		return out
	})

	RegisterTest("empty", func(value interface{}, _ ...interface{}) bool { return isEmptyValue(value) })
	RegisterTest("null", func(value interface{}, _ ...interface{}) bool { return value == nil })
	RegisterTest("defined", func(value interface{}, _ ...interface{}) bool { return value != nil })
	RegisterTest("iterable", func(value interface{}, _ ...interface{}) bool {
		kind := reflectValueKind(value)
		return kind == reflect.Array || kind == reflect.Slice || kind == reflect.Map
	})
	RegisterTest("odd", func(value interface{}, _ ...interface{}) bool {
		number, ok := numericValue(value)
		return ok && int(number)%2 != 0
	})
	RegisterTest("even", func(value interface{}, _ ...interface{}) bool {
		number, ok := numericValue(value)
		return ok && int(number)%2 == 0
	})
	RegisterTest("sameas", func(value interface{}, args ...interface{}) bool {
		if len(args) == 0 {
			return false
		}
		return fmt.Sprint(value) == fmt.Sprint(args[0])
	})
	RegisterTest("divisibleby", func(value interface{}, args ...interface{}) bool {
		if len(args) == 0 {
			return false
		}
		left, leftOK := numericValue(value)
		right, rightOK := numericValue(args[0])
		return leftOK && rightOK && right != 0 && int(left)%int(right) == 0
	})
}

func RegisterFilter(name string, fn FilterFunc) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" || fn == nil {
		return
	}
	filterRegistryMu.Lock()
	defer filterRegistryMu.Unlock()
	filterRegistry[name] = fn
}

func RegisterTest(name string, fn TestFunc) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" || fn == nil {
		return
	}
	testRegistryMu.Lock()
	defer testRegistryMu.Unlock()
	testRegistry[name] = fn
}

func applyFilter(name string, value interface{}, args ...interface{}) interface{} {
	filterRegistryMu.RLock()
	fn := filterRegistry[strings.ToLower(strings.TrimSpace(name))]
	filterRegistryMu.RUnlock()
	if fn == nil {
		return value
	}
	return fn(value, args...)
}

func applyTest(name string, value interface{}, args ...interface{}) bool {
	testRegistryMu.RLock()
	fn := testRegistry[strings.ToLower(strings.TrimSpace(name))]
	testRegistryMu.RUnlock()
	if fn == nil {
		return false
	}
	return fn(value, args...)
}

func evaluateTwigExpression(payload interface{}, expr string, funcs template.FuncMap) interface{} {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil
	}
	if left, right, ok := splitTopLevelWord(expr, " ?? "); ok {
		value := evaluateTwigExpression(payload, left, funcs)
		if !isEmptyValue(value) {
			return value
		}
		return evaluateTwigExpression(payload, right, funcs)
	}
	if left, right, ok := splitTopLevelWord(expr, " or "); ok {
		return truthy(evaluateTwigExpression(payload, left, funcs)) || truthy(evaluateTwigExpression(payload, right, funcs))
	}
	if left, right, ok := splitTopLevelWord(expr, " and "); ok {
		return truthy(evaluateTwigExpression(payload, left, funcs)) && truthy(evaluateTwigExpression(payload, right, funcs))
	}
	if strings.HasPrefix(expr, "not ") {
		return !truthy(evaluateTwigExpression(payload, strings.TrimSpace(strings.TrimPrefix(expr, "not ")), funcs))
	}
	if left, right, ok := splitTopLevelWord(expr, " ~ "); ok {
		return fmt.Sprint(evaluateTwigExpression(payload, left, funcs)) + fmt.Sprint(evaluateTwigExpression(payload, right, funcs))
	}
	if left, right, ok := splitTopLevelWord(expr, " in "); ok {
		return twigContains(evaluateTwigExpression(payload, right, funcs), evaluateTwigExpression(payload, left, funcs))
	}
	if left, op, right, ok := splitComparison(expr); ok {
		return compareValues(
			evaluateTwigExpression(payload, left, funcs),
			op,
			evaluateTwigExpression(payload, right, funcs),
		)
	}
	if left, testName, args, ok := splitTest(expr); ok {
		return applyTest(testName, evaluateTwigExpression(payload, left, funcs), evaluateTwigArgs(payload, args, funcs)...)
	}

	segments := splitPipeline(expr)
	value := evaluateTwigPrimary(payload, segments[0], funcs)
	for _, segment := range segments[1:] {
		name, args := parseInvocation(segment)
		if strings.TrimSpace(name) == "" {
			name = strings.TrimSpace(segment)
		}
		value = applyFilter(name, value, evaluateTwigArgs(payload, args, funcs)...)
	}
	return value
}

func evaluateTwigPrimary(payload interface{}, expr string, funcs template.FuncMap) interface{} {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil
	}
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		return evaluateTwigExpression(payload, strings.TrimSpace(expr[1:len(expr)-1]), funcs)
	}
	if quoted, ok := parseQuoted(expr); ok {
		return quoted
	}
	if number, ok := parseNumberLiteral(expr); ok {
		return number
	}
	switch strings.ToLower(expr) {
	case "true":
		return true
	case "false":
		return false
	case "null", "nil":
		return nil
	}

	if name, args := parseInvocation(expr); name != "" && len(args) >= 0 {
		if fn, ok := funcs[name]; ok {
			resolved := evaluateTwigArgs(payload, args, funcs)
			return callTemplateFunc(fn, resolved...)
		}
		if fn, ok := funcs[normalizeTwigCallableName(name)]; ok {
			resolved := evaluateTwigArgs(payload, args, funcs)
			return callTemplateFunc(fn, resolved...)
		}
	}
	return resolveTwigValue(payload, expr)
}

func evaluateTwigArgs(payload interface{}, args []string, funcs template.FuncMap) []interface{} {
	out := make([]interface{}, 0, len(args))
	for _, arg := range args {
		out = append(out, evaluateTwigExpression(payload, arg, funcs))
	}
	return out
}

func resolveTwigValue(current interface{}, expr string) interface{} {
	expr = strings.TrimSpace(expr)
	if expr == "." {
		return current
	}
	if strings.HasPrefix(expr, ".") {
		expr = strings.TrimPrefix(expr, ".")
	}
	if expr == "" {
		return current
	}
	parts := strings.Split(expr, ".")
	value := reflect.ValueOf(current)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !value.IsValid() {
			return nil
		}
		for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return nil
			}
			value = value.Elem()
		}
		switch value.Kind() {
		case reflect.Map:
			key := reflect.ValueOf(part)
			mapped := value.MapIndex(key)
			if !mapped.IsValid() && value.Type().Key().Kind() == reflect.String {
				for _, existing := range value.MapKeys() {
					if strings.EqualFold(fmt.Sprint(existing.Interface()), part) {
						mapped = value.MapIndex(existing)
						break
					}
				}
			}
			if !mapped.IsValid() {
				return nil
			}
			value = mapped
		case reflect.Struct:
			field := value.FieldByNameFunc(func(candidate string) bool {
				return strings.EqualFold(candidate, part)
			})
			if field.IsValid() {
				value = field
				continue
			}
			method := value.MethodByName(part)
			if method.IsValid() && method.Type().NumIn() == 0 && method.Type().NumOut() >= 1 {
				results := method.Call(nil)
				if len(results) > 0 {
					value = results[0]
					continue
				}
			}
			return nil
		case reflect.Slice, reflect.Array:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= value.Len() {
				return nil
			}
			value = value.Index(index)
		default:
			return nil
		}
	}
	if !value.IsValid() {
		return nil
	}
	return value.Interface()
}

func callTemplateFunc(fn interface{}, args ...interface{}) interface{} {
	defer func() {
		_ = recover()
	}()
	value := reflect.ValueOf(fn)
	if !value.IsValid() || value.Kind() != reflect.Func {
		return nil
	}
	inputs := make([]reflect.Value, 0, len(args))
	fnType := value.Type()
	for index := 0; index < fnType.NumIn(); index++ {
		if fnType.IsVariadic() && index == fnType.NumIn()-1 {
			elemType := fnType.In(index).Elem()
			for _, arg := range args[index:] {
				inputs = append(inputs, adaptValue(arg, elemType))
			}
			break
		}
		if index >= len(args) {
			inputs = append(inputs, reflect.Zero(fnType.In(index)))
			continue
		}
		inputs = append(inputs, adaptValue(args[index], fnType.In(index)))
	}
	results := value.Call(inputs)
	if len(results) == 0 {
		return nil
	}
	if len(results) == 2 && !results[1].IsNil() {
		return nil
	}
	return results[0].Interface()
}

func adaptValue(value interface{}, target reflect.Type) reflect.Value {
	if value == nil {
		return reflect.Zero(target)
	}
	current := reflect.ValueOf(value)
	if current.Type().AssignableTo(target) {
		return current
	}
	if current.Type().ConvertibleTo(target) {
		return current.Convert(target)
	}
	if target.Kind() == reflect.Interface {
		return current
	}
	return reflect.Zero(target)
}

func parseInvocation(expr string) (string, []string) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", nil
	}
	if open := strings.Index(expr, "("); open > 0 && strings.HasSuffix(expr, ")") {
		name := strings.TrimSpace(expr[:open])
		rawArgs := strings.TrimSpace(expr[open+1 : len(expr)-1])
		return name, splitTopLevelArgs(rawArgs)
	}
	parts := splitSpaceArgs(expr)
	if len(parts) > 1 {
		return parts[0], parts[1:]
	}
	return "", nil
}

func normalizeTwigCallableName(name string) string {
	return strings.ReplaceAll(strings.TrimSpace(name), ".", "__")
}

func splitPipeline(expr string) []string {
	return splitTopLevelByRune(expr, '|')
}

func splitTopLevelArgs(expr string) []string {
	return splitTopLevelByRune(expr, ',')
}

func splitTopLevelByRune(expr string, separator rune) []string {
	out := []string{}
	var current strings.Builder
	depth := 0
	inQuotes := false
	quote := rune(0)
	for _, char := range expr {
		switch {
		case inQuotes:
			current.WriteRune(char)
			if char == quote {
				inQuotes = false
			}
		case char == '"' || char == '\'':
			inQuotes = true
			quote = char
			current.WriteRune(char)
		case char == '(':
			depth++
			current.WriteRune(char)
		case char == ')':
			if depth > 0 {
				depth--
			}
			current.WriteRune(char)
		case char == separator && depth == 0:
			part := strings.TrimSpace(current.String())
			if part != "" {
				out = append(out, part)
			}
			current.Reset()
		default:
			current.WriteRune(char)
		}
	}
	if part := strings.TrimSpace(current.String()); part != "" {
		out = append(out, part)
	}
	return out
}

func splitSpaceArgs(expr string) []string {
	out := []string{}
	var current strings.Builder
	depth := 0
	inQuotes := false
	quote := rune(0)
	for _, char := range expr {
		switch {
		case inQuotes:
			current.WriteRune(char)
			if char == quote {
				inQuotes = false
			}
		case char == '"' || char == '\'':
			inQuotes = true
			quote = char
			current.WriteRune(char)
		case char == '(':
			depth++
			current.WriteRune(char)
		case char == ')':
			if depth > 0 {
				depth--
			}
			current.WriteRune(char)
		case char == ' ' && depth == 0:
			if part := strings.TrimSpace(current.String()); part != "" {
				out = append(out, part)
			}
			current.Reset()
		default:
			current.WriteRune(char)
		}
	}
	if part := strings.TrimSpace(current.String()); part != "" {
		out = append(out, part)
	}
	return out
}

func splitTopLevelWord(expr, needle string) (string, string, bool) {
	depth := 0
	inQuotes := false
	quote := rune(0)
	for i, char := range expr {
		switch {
		case inQuotes:
			if char == quote {
				inQuotes = false
			}
		case char == '"' || char == '\'':
			inQuotes = true
			quote = char
		case char == '(':
			depth++
		case char == ')':
			if depth > 0 {
				depth--
			}
		}
		if depth == 0 && strings.HasPrefix(expr[i:], needle) {
			return strings.TrimSpace(expr[:i]), strings.TrimSpace(expr[i+len(needle):]), true
		}
	}
	return "", "", false
}

func splitComparison(expr string) (string, string, string, bool) {
	for _, operator := range []string{"==", "!=", ">=", "<=", ">", "<"} {
		if left, right, ok := splitTopLevelWord(expr, " "+operator+" "); ok {
			return left, operator, right, true
		}
	}
	return "", "", "", false
}

func splitTest(expr string) (string, string, []string, bool) {
	left, right, ok := splitTopLevelWord(expr, " is ")
	if !ok {
		return "", "", nil, false
	}
	name, args := parseInvocation(right)
	if name != "" {
		return left, strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), " ", ""), args, true
	}
	return left, strings.ReplaceAll(strings.ToLower(strings.TrimSpace(right)), " ", ""), nil, true
}

func compareValues(left interface{}, op string, right interface{}) bool {
	switch op {
	case "==":
		return fmt.Sprint(left) == fmt.Sprint(right)
	case "!=":
		return fmt.Sprint(left) != fmt.Sprint(right)
	}
	leftFloat, leftOK := numericValue(left)
	rightFloat, rightOK := numericValue(right)
	if leftOK && rightOK {
		switch op {
		case ">":
			return leftFloat > rightFloat
		case ">=":
			return leftFloat >= rightFloat
		case "<":
			return leftFloat < rightFloat
		case "<=":
			return leftFloat <= rightFloat
		}
	}
	ls := fmt.Sprint(left)
	rs := fmt.Sprint(right)
	switch op {
	case ">":
		return ls > rs
	case ">=":
		return ls >= rs
	case "<":
		return ls < rs
	case "<=":
		return ls <= rs
	default:
		return false
	}
}

func numericValue(value interface{}) (float64, bool) {
	switch current := value.(type) {
	case int:
		return float64(current), true
	case int8:
		return float64(current), true
	case int16:
		return float64(current), true
	case int32:
		return float64(current), true
	case int64:
		return float64(current), true
	case uint:
		return float64(current), true
	case uint8:
		return float64(current), true
	case uint16:
		return float64(current), true
	case uint32:
		return float64(current), true
	case uint64:
		return float64(current), true
	case float32:
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

func parseQuoted(expr string) (string, bool) {
	if len(expr) < 2 {
		return "", false
	}
	if (strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"")) || (strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'")) {
		return expr[1 : len(expr)-1], true
	}
	return "", false
}

func parseNumberLiteral(expr string) (interface{}, bool) {
	if value, err := strconv.ParseInt(expr, 10, 64); err == nil {
		return value, true
	}
	if value, err := strconv.ParseFloat(expr, 64); err == nil {
		return value, true
	}
	return nil, false
}

func mustJSON(value interface{}) string {
	data, err := jsonMarshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func jsonMarshal(value interface{}) ([]byte, error) {
	type jsonMarshaler interface {
		MarshalJSON() ([]byte, error)
	}
	if custom, ok := value.(jsonMarshaler); ok {
		return custom.MarshalJSON()
	}
	return json.Marshal(value)
}

func toSlice(value interface{}) []interface{} {
	current := reflect.ValueOf(value)
	if !current.IsValid() {
		return nil
	}
	for current.Kind() == reflect.Interface || current.Kind() == reflect.Pointer {
		if current.IsNil() {
			return nil
		}
		current = current.Elem()
	}
	switch current.Kind() {
	case reflect.Array, reflect.Slice:
		out := make([]interface{}, 0, current.Len())
		for i := 0; i < current.Len(); i++ {
			out = append(out, current.Index(i).Interface())
		}
		return out
	case reflect.Map:
		keys := current.MapKeys()
		sort.SliceStable(keys, func(i, j int) bool {
			return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
		})
		out := make([]interface{}, 0, len(keys))
		for _, key := range keys {
			out = append(out, current.MapIndex(key).Interface())
		}
		return out
	default:
		return nil
	}
}

func truthy(value interface{}) bool {
	if boolean, ok := value.(bool); ok {
		return boolean
	}
	return !isEmptyValue(value)
}

func twigContains(container interface{}, needle interface{}) bool {
	switch current := container.(type) {
	case string:
		return strings.Contains(current, fmt.Sprint(needle))
	}
	value := reflect.ValueOf(container)
	if !value.IsValid() {
		return false
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return false
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		for index := 0; index < value.Len(); index++ {
			if fmt.Sprint(value.Index(index).Interface()) == fmt.Sprint(needle) {
				return true
			}
		}
	case reflect.Map:
		for _, key := range value.MapKeys() {
			if fmt.Sprint(key.Interface()) == fmt.Sprint(needle) {
				return true
			}
		}
	}
	return false
}

func reflectValueKind(value interface{}) reflect.Kind {
	current := reflect.ValueOf(value)
	if !current.IsValid() {
		return reflect.Invalid
	}
	for current.Kind() == reflect.Interface || current.Kind() == reflect.Pointer {
		if current.IsNil() {
			return reflect.Invalid
		}
		current = current.Elem()
	}
	return current.Kind()
}
