package engine

import (
	"fmt"
	"packet-repackage/models"
	"regexp"
	"strings"
)

// EvaluateCondition evaluates a condition expression against packet context
// Supports: field == "value", field != "value", &&, ||, !, ()
func EvaluateCondition(condition string, ctx *PacketContext, fields []models.Field) (bool, error) {
	if strings.TrimSpace(condition) == "" {
		return true, nil
	}

	// Build field map for lookup
	fieldMap := make(map[string]models.Field)
	for _, f := range fields {
		fieldMap[f.Name] = f
	}

	// Parse and evaluate expression tree
	return evaluateExpression(condition, ctx, fieldMap)
}

func evaluateExpression(expr string, ctx *PacketContext, fieldMap map[string]models.Field) (bool, error) {
	expr = strings.TrimSpace(expr)

	// Handle OR operations (lowest precedence)
	orParts := splitByOperator(expr, "||")
	if len(orParts) > 1 {
		for _, part := range orParts {
			result, err := evaluateExpression(part, ctx, fieldMap)
			if err != nil {
				return false, err
			}
			if result {
				return true, nil
			}
		}
		return false, nil
	}

	// Handle AND operations
	andParts := splitByOperator(expr, "&&")
	if len(andParts) > 1 {
		for _, part := range andParts {
			result, err := evaluateExpression(part, ctx, fieldMap)
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil
			}
		}
		return true, nil
	}

	// Handle NOT operations
	if strings.HasPrefix(expr, "!") {
		result, err := evaluateExpression(expr[1:], ctx, fieldMap)
		if err != nil {
			return false, err
		}
		return !result, nil
	}

	// Handle parentheses
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		return evaluateExpression(expr[1:len(expr)-1], ctx, fieldMap)
	}

	// Handle comparison operations
	return evaluateComparison(expr, ctx, fieldMap)
}

func evaluateComparison(expr string, ctx *PacketContext, fieldMap map[string]models.Field) (bool, error) {
	expr = strings.TrimSpace(expr)

	// Match pattern: fieldName == "value" or fieldName != "value"
	eqRegex := regexp.MustCompile(`(\w+)\s*==\s*"([^"]*)"`)
	neRegex := regexp.MustCompile(`(\w+)\s*!=\s*"([^"]*)"`)

	if matches := eqRegex.FindStringSubmatch(expr); matches != nil {
		fieldName := matches[1]
		expectedValue := matches[2]
		return compareField(fieldName, expectedValue, true, ctx, fieldMap)
	}

	if matches := neRegex.FindStringSubmatch(expr); matches != nil {
		fieldName := matches[1]
		expectedValue := matches[2]
		return compareField(fieldName, expectedValue, false, ctx, fieldMap)
	}

	return false, fmt.Errorf("invalid comparison expression: %s", expr)
}

func compareField(fieldName, expectedValue string, equality bool, ctx *PacketContext, fieldMap map[string]models.Field) (bool, error) {
	field, exists := fieldMap[fieldName]
	if !exists {
		return false, fmt.Errorf("field not found: %s", fieldName)
	}

	actualValue := ctx.Fields[fieldName]
	result := CompareFieldValue(actualValue, expectedValue, field.Type)

	if equality {
		return result, nil
	}
	return !result, nil
}

// splitByOperator splits expression by operator while respecting parentheses
func splitByOperator(expr string, operator string) []string {
	var parts []string
	var current strings.Builder
	depth := 0
	i := 0

	for i < len(expr) {
		if expr[i] == '(' {
			depth++
			current.WriteByte(expr[i])
			i++
		} else if expr[i] == ')' {
			depth--
			current.WriteByte(expr[i])
			i++
		} else if depth == 0 && i+len(operator) <= len(expr) && expr[i:i+len(operator)] == operator {
			parts = append(parts, current.String())
			current.Reset()
			i += len(operator)
		} else {
			current.WriteByte(expr[i])
			i++
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	if len(parts) == 0 {
		return []string{expr}
	}

	return parts
}
