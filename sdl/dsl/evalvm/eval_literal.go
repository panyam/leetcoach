package dsl

import (
	"fmt"
	"strconv"

	"github.com/panyam/leetcoach/sdl/core"
)

// sdl/dsl/eval_literal.go (Updated)

// evalLiteral now returns *VarState and error
func (v *VM) evalLiteral(expr *LiteralExpr) (*VarState, error) {
	var valueOutcome any
	var parseErr error

	// Create zero latency outcome for literals
	zeroLat := zeroLatencyOutcome() // Use helper

	switch expr.Kind {
	case "INT":
		val, err := strconv.ParseInt(expr.Value, 10, 64)
		if err != nil {
			parseErr = fmt.Errorf("invalid INT literal '%s': %w", expr.Value, err)
		} else {
			o := (&core.Outcomes[int64]{}).Add(1.0, val)
			valueOutcome = o
		}
	case "FLOAT":
		val, err := strconv.ParseFloat(expr.Value, 64)
		if err != nil {
			parseErr = fmt.Errorf("invalid FLOAT literal '%s': %w", expr.Value, err)
		} else {
			o := (&core.Outcomes[float64]{}).Add(1.0, val)
			valueOutcome = o
		}
	case "STRING":
		o := (&core.Outcomes[string]{}).Add(1.0, expr.Value)
		valueOutcome = o
	case "BOOL":
		val, err := strconv.ParseBool(expr.Value)
		if err != nil {
			parseErr = fmt.Errorf("invalid BOOL literal '%s': %w", expr.Value, err)
		} else {
			o := (&core.Outcomes[bool]{}).Add(1.0, val)
			valueOutcome = o
		}
	// TODO: case "DURATION":
	default:
		return nil, fmt.Errorf("unknown literal kind: %s", expr.Kind)
	}

	if parseErr != nil {
		return nil, parseErr
	}
	if valueOutcome == nil {
		return nil, fmt.Errorf("internal error: value outcome is nil after parsing literal %s", expr.Value)
	}

	// Return the VarState
	return &VarState{
		ValueOutcome:   valueOutcome,
		LatencyOutcome: zeroLat,
	}, nil
}
