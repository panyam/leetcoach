package dsl

import (
	"errors"
	"fmt"
)

// evalInternalCall evaluates the arguments, looks up the function, calls it,
// and returns the resulting VarState and error.
func (v *VM) evalInternalCall(expr *InternalCallExpr) (*VarState, error) {
	// 1. Evaluate arguments recursively -> results are returned directly by Eval
	evaluatedArgStates := make([]*VarState, len(expr.Args)) // <-- Slice of *VarState

	for i, argExpr := range expr.Args {
		argState, err := v.Eval(argExpr)
		if err != nil {
			var retVal *ReturnValue // Handle return signal
			if errors.As(err, &retVal) {
				return retVal.Value, err
			}
			return nil, fmt.Errorf("error evaluating arg %d for %s: %w", i, expr.FuncName, err)
		}
		if argState == nil {
			return nil, fmt.Errorf("internal error: nil state from arg %d evaluation for %s", i, expr.FuncName)
		}
		evaluatedArgStates[i] = argState // Store the *VarState directly
	}

	// 2. Look up the internal function
	internalFn, ok := v.internalFuncs[expr.FuncName]
	if !ok {
		return nil, fmt.Errorf("%w: '%s'", ErrInternalFuncNotFound, expr.FuncName)
	}

	// 3. Call the internal function
	// *** Pass the slice of *VarState directly ***
	resultState, err := internalFn(v, evaluatedArgStates)
	if err != nil {
		// Check for return signal propagation from internal function
		var retVal *ReturnValue
		if errors.As(err, &retVal) {
			return retVal.Value, err
		}
		// Handle real error
		return nil, fmt.Errorf("error executing internal function '%s': %w", expr.FuncName, err)
	}

	// 4. Check and return the result (should be *VarState)
	if resultState == nil {
		// Internal function returned nil state without error? Problematic.
		return nil, fmt.Errorf("internal error: function '%s' returned nil *VarState without error", expr.FuncName)
	}

	return resultState, nil
}
