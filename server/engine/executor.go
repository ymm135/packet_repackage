package engine

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Action represents a modification action
type Action struct {
	Field string `json:"field"` // Field name to modify
	Op    string `json:"op"`    // Operation: set, add, sub, mul, div, shell
	Value string `json:"value"` // Value or shell command
}

// ExecuteActions executes all actions on the packet context
func ExecuteActions(actionsJSON string, ctx *PacketContext) error {
	if strings.TrimSpace(actionsJSON) == "" {
		return nil
	}

	var actions []Action
	err := json.Unmarshal([]byte(actionsJSON), &actions)
	if err != nil {
		return fmt.Errorf("failed to parse actions: %w", err)
	}

	for _, action := range actions {
		err = executeAction(action, ctx)
		if err != nil {
			return fmt.Errorf("failed to execute action on %s: %w", action.Field, err)
		}
	}

	return nil
}

func executeAction(action Action, ctx *PacketContext) error {
	currentValue := ctx.Fields[action.Field]

	switch action.Op {
	case "set":
		// Direct value assignment
		ctx.Fields[action.Field] = action.Value
		
	case "add", "sub", "mul", "div":
		// Arithmetic operations
		result, err := performArithmetic(currentValue, action.Value, action.Op)
		if err != nil {
			return err
		}
		ctx.Fields[action.Field] = result
		
	case "shell":
		// Execute shell command and use output
		result, err := executeShellCommand(action.Value)
		if err != nil {
			return err
		}
		ctx.Fields[action.Field] = strings.TrimSpace(result)
		
	default:
		return fmt.Errorf("unknown operation: %s", action.Op)
	}

	return nil
}

func performArithmetic(currentValue interface{}, valueStr string, op string) (interface{}, error) {
	// Try to convert current value to int64
	var current int64
	switch v := currentValue.(type) {
	case int64:
		current = v
	case int:
		current = int64(v)
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert current value to number: %v", currentValue)
		}
		current = parsed
	default:
		return nil, fmt.Errorf("unsupported type for arithmetic: %T", currentValue)
	}

	// Parse the operand value
	operand, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid numeric value: %s", valueStr)
	}

	// Perform operation
	switch op {
	case "add":
		return current + operand, nil
	case "sub":
		return current - operand, nil
	case "mul":
		return current * operand, nil
	case "div":
		if operand == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return current / operand, nil
	default:
		return nil, fmt.Errorf("unknown arithmetic operation: %s", op)
	}
}

func executeShellCommand(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("shell command failed: %w", err)
	}
	return string(output), nil
}
