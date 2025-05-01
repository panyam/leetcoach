package dsl

// sdl/dsl/eval_stmt.go (Latest evalIfStmt and helpers)

import (
	"errors"
	"fmt"
	"reflect"

	// Still needed for getOutcomeTypeString if we keep that helper
	"github.com/panyam/leetcoach/sdl/core"
)

var (
	ErrInvalidConditionType = errors.New("if condition must evaluate to *core.Outcomes[bool] or access .Success field")
	ErrConditionSplitFailed = errors.New("failed to split outcome based on condition")
)

// selectSplitPredicate returns the predicate for bool outcomes.
// Error if condition didn't evaluate to Outcomes[bool].
func selectSplitPredicate(conditionValueOutcome any) (func(v bool) bool, error) {
	if _, ok := conditionValueOutcome.(*core.Outcomes[bool]); ok {
		// The predicate just returns the boolean value itself
		return func(v bool) bool {
			return v
		}, nil
	}
	return nil, fmt.Errorf("%w: condition evaluated to type %T, expected *core.Outcomes[bool]",
		ErrInvalidConditionType, conditionValueOutcome)
}

// evalIfStmt handles conditional execution using dual tracks.
// It evaluates the condition (expecting Outcomes[bool]), splits value and latency tracks,
// evaluates the appropriate branches, combines the results, applies reduction,
// and returns the final combined state and error (including ReturnValue signal).
func (v *VM) evalIfStmt(stmt *IfStmt) (*VarState, error) {
	// 1. Evaluate the Condition Expression
	conditionState, err := v.Eval(stmt.Condition)
	if err != nil {
		var retVal *ReturnValue // Check for early return during condition eval
		if errors.As(err, &retVal) {
			return retVal.Value, err
		}
		return nil, fmt.Errorf("error evaluating if condition: %w", err)
	}
	if conditionState == nil || conditionState.ValueOutcome == nil {
		return nil, fmt.Errorf("internal error: nil state or value outcome from condition evaluation")
	}

	// 2. Get the Boolean Predicate
	// Relies *only* on the type of the value outcome.
	boolPredicate, err := selectSplitPredicate(conditionState.ValueOutcome)
	if err != nil {
		return nil, err // Condition didn't yield Outcomes[bool]
	}
	boolOutcome, _ := conditionState.ValueOutcome.(*core.Outcomes[bool]) // Cast is safe here

	// 3. Split Value Track (Outcomes[bool])
	thenValOutcomeAny, elseValOutcomeAny := boolOutcome.Split(boolPredicate)
	thenValOutcome := thenValOutcomeAny
	elseValOutcome := elseValOutcomeAny

	// 4. Split Latency Track based on Value Predicate
	latOutcome, ok := conditionState.LatencyOutcome.(*core.Outcomes[core.Duration])
	if !ok || latOutcome == nil {
		latOutcome = zeroLatencyOutcome() // Use zero if missing
	}
	thenLatOutcome, elseLatOutcome, splitErr := SplitLatenciesBasedOnValue(latOutcome, boolOutcome, boolPredicate)
	if splitErr != nil {
		return nil, fmt.Errorf("error splitting latency based on value for if: %w", splitErr)
	}

	// 5. Prepare contexts for branch evaluation
	thenCtxState := &VarState{ValueOutcome: thenValOutcome, LatencyOutcome: thenLatOutcome}
	elseCtxState := &VarState{ValueOutcome: elseValOutcome, LatencyOutcome: elseLatOutcome}

	// 6. Determine which branches have probability weight
	thenHasWeight := outcomeHasWeight(thenCtxState.ValueOutcome)
	elseHasWeight := outcomeHasWeight(elseCtxState.ValueOutcome)

	var finalThenResultState *VarState = nil // Initialize result states
	var finalElseResultState *VarState = nil
	var branchErr error

	// 7. Evaluate Then Branch (if it has weight)
	if thenHasWeight && stmt.Then != nil { // Also check if Then block exists
		finalThenResultState, branchErr = v.evalBlockStmt(stmt.Then, v.env, thenCtxState)
		if branchErr != nil {
			var retVal *ReturnValue
			if errors.As(branchErr, &retVal) {
				return retVal.Value, branchErr
			} // Propagate return
			return nil, fmt.Errorf("error in 'then' branch: %w", branchErr)
		}
	} else if thenHasWeight && stmt.Then == nil {
		// Condition was true, but no 'then' block - result is the 'then' context
		finalThenResultState = thenCtxState
	}

	// 8. Evaluate Else Branch (if it exists and has weight)
	if stmt.Else != nil && elseHasWeight {
		finalElseResultState, branchErr = v.evalBlockStmt(stmt.Else, v.env, elseCtxState)
		if branchErr != nil {
			var retVal *ReturnValue
			if errors.As(branchErr, &retVal) {
				return retVal.Value, branchErr
			} // Propagate return
			return nil, fmt.Errorf("error in 'else' branch: %w", branchErr)
		}
	} else if stmt.Else == nil && elseHasWeight {
		// No Else block, result for this path is the original 'else' context
		finalElseResultState = elseCtxState
	}

	// 9. Combine Results from Branches using Append
	var combinedValOutcome any
	var combinedLatOutcome any

	// Combine Value Outcomes (use type switch on *one* of the results, assuming types match if both non-nil)
	refValOutcome := finalThenResultState.SafeValueOutcome()
	if refValOutcome == nil {
		refValOutcome = finalElseResultState.SafeValueOutcome()
	} // Use else if then was nil

	switch refValOutcome.(type) {
	case nil: // Both branches resulted in nil value outcome
		combinedValOutcome = nil
	case *core.Outcomes[bool]:
		tv := finalThenResultState.SafeValueOutcome().(*core.Outcomes[bool])
		ev := finalElseResultState.SafeValueOutcome().(*core.Outcomes[bool])
		if tv == nil {
			tv = &core.Outcomes[bool]{}
		} // Need non-nil receiver for Append
		combinedValOutcome = tv.Append(ev) // Append handles nil ev
	case *core.Outcomes[core.AccessResult]: // If branches can return AccessResult (e.g., from calls)
		tv := finalThenResultState.SafeValueOutcome().(*core.Outcomes[core.AccessResult])
		ev := finalElseResultState.SafeValueOutcome().(*core.Outcomes[core.AccessResult])
		if tv == nil {
			tv = &core.Outcomes[core.AccessResult]{}
		}
		combinedValOutcome = tv.Append(ev)
	// Add other discrete types as needed
	default:
		// This occurs if branches return incompatible types or unsupported types
		// We need to know the expected return type of the 'if' expression if used as value
		// For now, error if types seem incompatible or unknown during append.
		if finalThenResultState.SafeValueOutcome() != nil && finalElseResultState.SafeValueOutcome() != nil &&
			reflect.TypeOf(finalThenResultState.SafeValueOutcome()) != reflect.TypeOf(finalElseResultState.SafeValueOutcome()) {
			return nil, fmt.Errorf("if branch value type mismatch: then=%T, else=%T", finalThenResultState.SafeValueOutcome(), finalElseResultState.SafeValueOutcome())
		}
		return nil, fmt.Errorf("unsupported type %T in if branch value combination", finalThenResultState.SafeValueOutcome())
	}

	// Combine Latency Outcomes (always Duration)
	lat1, _ := finalThenResultState.SafeLatencyOutcome().(*core.Outcomes[core.Duration])
	lat2, _ := finalElseResultState.SafeLatencyOutcome().(*core.Outcomes[core.Duration])
	if lat1 == nil {
		lat1 = zeroLatencyOutcome()
	} // Use zero if nil
	if lat2 == nil {
		lat2 = zeroLatencyOutcome()
	}
	// Need non-nil receiver for Append
	tempLat := lat1.Copy() // Copy lat1 to avoid modifying it if it was context
	combinedLatOutcome = tempLat.Append(lat2)

	// Create combined state
	finalCombinedState := &VarState{
		ValueOutcome:   combinedValOutcome,
		LatencyOutcome: combinedLatOutcome,
	}

	// Handle case where neither branch produced output
	if finalCombinedState.ValueOutcome == nil && finalCombinedState.LatencyOutcome.(*core.Outcomes[core.Duration]).Len() == 0 {
		finalCombinedState = createIdentityState()
	} else if finalCombinedState.ValueOutcome != nil && finalCombinedState.LatencyOutcome == nil {
		// Should not happen if we initialize lat1/lat2 correctly above
		finalCombinedState.LatencyOutcome = zeroLatencyOutcome()
	}

	// 10. Apply Reduction
	reducedState, reduceErr := v.reduceStateIfNeeded(finalCombinedState)
	if reduceErr != nil {
		return nil, fmt.Errorf("error reducing combined IfStmt result: %w", reduceErr)
	}

	// 11. Return the final combined state
	return reducedState, nil
}
