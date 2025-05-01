package dsl

import (
	"fmt"
)

// evalIdentifier looks up the identifier's name in the current environment
// and returns the found VarState.
func (v *VM) evalIdentifier(expr *IdentifierExpr) (*VarState, error) {
	state, ok := v.env.Get(expr.Name) // Get returns *VarState now
	if !ok {
		return nil, fmt.Errorf("%w: '%s'", ErrNotFound, expr.Name)
	}
	if state == nil {
		// Should not happen if Set prevents nil VarState
		return nil, fmt.Errorf("internal error: found nil VarState for identifier '%s'", expr.Name)
	}
	// Return the state containing both value and latency outcomes
	return state, nil
}
