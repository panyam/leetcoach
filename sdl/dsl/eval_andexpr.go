package dsl

import (
	"errors"
	"fmt"
	// Needed for more generic type handling later, maybe basic switch now
)

var (
	ErrTypeMismatch    = errors.New("type mismatch in operation")
	ErrUnsupportedType = errors.New("unsupported type for operation")
)

// evalAndExpr evaluates Left, then Right. Implicitly combines their results
// which are assumed to be pushed onto the stack by their respective Eval calls
// if they were statement expressions. If AndExpr is used purely as an expression,
// Eval calls won't modify the stack directly, but return states.
// Let's stick to the signature: Eval returns state, caller pushes if needed.
// evalAndExpr will evaluate, combine, and return the combined state.
func (v *VM) evalAndExpr(expr *AndExpr) (*VarState, error) {
	// 1. Evaluate Left -> returns state directly
	leftState, err := v.Eval(expr.Left)
	if err != nil {
		var retVal *ReturnValue // Check for early return
		if errors.As(err, &retVal) {
			return retVal.Value, err
		}
		return nil, fmt.Errorf("error evaluating left side of AND: %w", err)
	}
	if leftState == nil {
		return nil, fmt.Errorf("internal error: Eval returned nil state for left side of AND without error")
	}

	// 2. Evaluate Right -> returns state directly
	rightState, err := v.Eval(expr.Right)
	if err != nil {
		var retVal *ReturnValue // Check for early return
		if errors.As(err, &retVal) {
			return retVal.Value, err
		}
		return nil, fmt.Errorf("error evaluating right side of AND: %w", err)
	}
	if rightState == nil {
		return nil, fmt.Errorf("internal error: Eval returned nil state for right side of AND without error")
	}

	// 3. Combine the returned states using the helper
	combinedState, err := v.combineOutcomesAndReduce(leftState, rightState)
	if err != nil {
		return nil, fmt.Errorf("error combining results in AND expression: %w", err)
	}

	// 4. Return the final combined state
	// The caller (e.g., evalBlockStmt for an ExprStmt, or evalAssignmentStmt)
	// will push this onto the stack if needed.
	return combinedState, nil
}
