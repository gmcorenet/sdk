package gmcore_expression

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type ExpressionLanguage interface {
	Evaluate(expression string, context map[string]interface{}) (interface{}, error)
	Compile(expression string) (Expression, error)
}

type Expression interface {
	Evaluate(ctx map[string]interface{}) (interface{}, error)
}

type expression struct {
	expr  string
	funcs map[string]func(args ...interface{}) interface{}
}

func New() *expression {
	return &expression{
		funcs: make(map[string]func(args ...interface{}) interface{}),
	}
}

func (e *expression) RegisterFunction(name string, fn func(args ...interface{}) interface{}) {
	e.funcs[name] = fn
}

func (e *expression) Evaluate(expr string, ctx map[string]interface{}) (interface{}, error) {
	compiled, err := e.Compile(expr)
	if err != nil {
		return nil, err
	}
	return compiled.Evaluate(ctx)
}

func (e *expression) Compile(expr string) (Expression, error) {
	return &compiledExpression{expr: expr, funcs: e.funcs}, nil
}

type compiledExpression struct {
	expr  string
	funcs map[string]func(args ...interface{}) interface{}
}

func (e *compiledExpression) Evaluate(ctx map[string]interface{}) (interface{}, error) {
	expr := strings.TrimSpace(e.expr)
	if expr == "" {
		return nil, nil
	}

	expr = e.processStringLiterals(expr)

	expr = e.resolveVariables(expr, ctx)

	expr = e.evaluateFunctions(expr, ctx)

	result, err := e.evaluateOperators(expr)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (e *compiledExpression) processStringLiterals(expr string) string {
	re := regexp.MustCompile(`"(.*?)"`)
	return re.ReplaceAllStringFunc(expr, func(match string) string {
		content := match[1 : len(match)-1]
		return "STR_LITERAL:" + content
	})
}

func (e *compiledExpression) resolveVariables(expr string, ctx map[string]interface{}) string {
	re := regexp.MustCompile(`\$(\w+)`)
	return re.ReplaceAllStringFunc(expr, func(match string) string {
		varName := match[1:]
		if val, ok := ctx[varName]; ok {
			return fmt.Sprintf("VAL:%v", val)
		}
		return "VAL:nil"
	})
}

func (e *compiledExpression) evaluateFunctions(expr string, ctx map[string]interface{}) string {
	for name, fn := range e.funcs {
		pattern := regexp.MustCompile(fmt.Sprintf(`%s\(([^)]*)\)`, regexp.QuoteMeta(name)))
		expr = pattern.ReplaceAllStringFunc(expr, func(match string) string {
			argsStr := match[len(name)+1 : len(match)-1]
			args := e.parseArgs(argsStr, ctx)
			result := fn(args...)
			return fmt.Sprintf("FUNC_RESULT:%v", result)
		})
	}
	return expr
}

func (e *compiledExpression) parseArgs(argsStr string, ctx map[string]interface{}) []interface{} {
	if strings.TrimSpace(argsStr) == "" {
		return nil
	}
	parts := strings.Split(argsStr, ",")
	args := make([]interface{}, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "VAL:") {
			valStr := part[4:]
			if valStr == "nil" {
				args = append(args, nil)
			} else {
				args = append(args, valStr)
			}
		} else if strings.HasPrefix(part, "STR_LITERAL:") {
			args = append(args, part[12:])
		} else if strings.HasPrefix(part, "FUNC_RESULT:") {
			args = append(args, part[12:])
		} else {
			args = append(args, part)
		}
	}
	return args
}

func (e *compiledExpression) evaluateOperators(expr string) (interface{}, error) {
	expr = strings.TrimSpace(expr)
	if strings.HasPrefix(expr, "STR_LITERAL:") {
		return expr[13:], nil
	}
	if strings.HasPrefix(expr, "VAL:") {
		valStr := expr[4:]
		if valStr == "nil" {
			return nil, nil
		}
		if n, err := strconv.Atoi(valStr); err == nil {
			return n, nil
		}
		if f, err := strconv.ParseFloat(valStr, 64); err == nil {
			return f, nil
		}
		if valStr == "true" {
			return true, nil
		}
		if valStr == "false" {
			return false, nil
		}
		return valStr, nil
	}
	if strings.HasPrefix(expr, "FUNC_RESULT:") {
		valStr := expr[12:]
		if n, err := strconv.Atoi(valStr); err == nil {
			return n, nil
		}
		if f, err := strconv.ParseFloat(valStr, 64); err == nil {
			return f, nil
		}
		if valStr == "true" {
			return true, nil
		}
		if valStr == "false" {
			return false, nil
		}
		return valStr, nil
	}

	expr = e.evaluateComparison(expr, "||")
	expr = e.evaluateComparison(expr, "&&")
	expr = e.evaluateComparison(expr, "==")
	expr = e.evaluateComparison(expr, "!=")
	expr = e.evaluateComparison(expr, ">=")
	expr = e.evaluateComparison(expr, "<=")
	expr = e.evaluateComparison(expr, ">")
	expr = e.evaluateComparison(expr, "<")
	expr = e.evaluateArithmetic(expr, "+")
	expr = e.evaluateArithmetic(expr, "-")
	expr = e.evaluateArithmetic(expr, "*")
	expr = e.evaluateArithmetic(expr, "/")

	return e.parseFinalValue(expr)
}

func (e *compiledExpression) evaluateComparison(expr string, op string) string {
	parts := strings.Split(expr, op)
	if len(parts) != 2 {
		return expr
	}
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])
	leftVal, _ := e.parseFinalValue(left)
	rightVal, _ := e.parseFinalValue(right)
	var result bool
	switch op {
	case "==":
		result = reflect.DeepEqual(leftVal, rightVal)
	case "!=":
		result = !reflect.DeepEqual(leftVal, rightVal)
	case ">", ">=", "<", "<=":
		lNum, lOk := toFloat(leftVal)
		rNum, rOk := toFloat(rightVal)
		if lOk && rOk {
			switch op {
			case ">":
				result = lNum > rNum
			case ">=":
				result = lNum >= rNum
			case "<":
				result = lNum < rNum
			case "<=":
				result = lNum <= rNum
			}
		}
	case "&&":
		result = toBool(leftVal) && toBool(rightVal)
	case "||":
		result = toBool(leftVal) || toBool(rightVal)
	}
	return fmt.Sprintf("VAL:%v", result)
}

func (e *compiledExpression) evaluateArithmetic(expr string, op string) string {
	parts := strings.Split(expr, op)
	if len(parts) != 2 {
		return expr
	}
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])
	leftVal, leftOk := toFloat(fromVal(left))
	rightVal, rightOk := toFloat(fromVal(right))
	if !leftOk || !rightOk {
		return expr
	}
	var result float64
	switch op {
	case "+":
		result = leftVal + rightVal
	case "-":
		result = leftVal - rightVal
	case "*":
		result = leftVal * rightVal
	case "/":
		if rightVal == 0 {
			return expr
		}
		result = leftVal / rightVal
	}
	if result == float64(int(result)) {
		return fmt.Sprintf("VAL:%d", int(result))
	}
	return fmt.Sprintf("VAL:%v", result)
}

func (e *compiledExpression) parseFinalValue(expr string) (interface{}, error) {
	expr = strings.TrimSpace(expr)
	if strings.HasPrefix(expr, "VAL:") {
		return fromVal(expr), nil
	}
	if strings.HasPrefix(expr, "FUNC_RESULT:") {
		return fromVal(expr), nil
	}
	if n, err := strconv.Atoi(expr); err == nil {
		return n, nil
	}
	if f, err := strconv.ParseFloat(expr, 64); err == nil {
		return f, nil
	}
	if expr == "true" {
		return true, nil
	}
	if expr == "false" {
		return false, nil
	}
	return expr, nil
}

func fromVal(s string) interface{} {
	if strings.HasPrefix(s, "VAL:") {
		s = s[4:]
	}
	if strings.HasPrefix(s, "FUNC_RESULT:") {
		s = s[12:]
	}
	if s == "nil" || s == "" {
		return nil
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	return s
}

func toFloat(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

func toBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val == "true" || val == "1"
	case int:
		return val != 0
	case float64:
		return val != 0
	}
	return false
}

type Evaluator struct {
	functions map[string]func(args ...interface{}) interface{}
}

func NewEvaluator() *Evaluator {
	return &Evaluator{functions: make(map[string]func(args ...interface{}) interface{})}
}

func (e *Evaluator) AddFunction(name string, fn func(args ...interface{}) interface{}) {
	e.functions[name] = fn
}

func (e *Evaluator) Evaluate(expr string, ctx map[string]interface{}) (interface{}, error) {
	if fn, ok := e.functions[expr]; ok {
		return fn(), nil
	}

	if val, ok := ctx[expr]; ok {
		return val, nil
	}

	return nil, fmt.Errorf("unknown expression: %s", expr)
}

func SimpleEvaluate(expr string, variables map[string]interface{}) (interface{}, error) {
	eval := NewEvaluator()
	return eval.Evaluate(expr, variables)
}
