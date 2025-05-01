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

// outcomeHasWeight checks if an outcome object (expected *core.Outcomes[T]) is non-nil and has weight > 0

func outcomeHasWeight(outcome any) bool {
	if outcome == nil {
		return false
	}
	// Use the interface to check TotalWeight
	if oc, ok := outcome.(core.OutcomeContainer); ok {
		return oc.TotalWeight() > 1e-12 // Use a small epsilon
	}
	// If it's not an OutcomeContainer, log a warning but assume no weight?
	fmt.Printf("Warning: outcomeHasWeight received non-OutcomeContainer type: %T\n", outcome)
	return false
}

// SplitLatenciesBasedOnValue splits a duration distribution based on a predicate
// applied to a corresponding value distribution, assuming they have synchronized buckets.
// Returns the split latency distributions (thenLat, elseLat).
func SplitLatenciesBasedOnValue[V any](
	latDistribution *core.Outcomes[core.Duration],
	valueDistribution *core.Outcomes[V],
	valuePredicate func(v V) bool,
) (thenLat *core.Outcomes[core.Duration], elseLat *core.Outcomes[core.Duration], err error) {

	thenLat = &core.Outcomes[core.Duration]{} // Initialize empty
	elseLat = &core.Outcomes[core.Duration]{} // Initialize empty

	if latDistribution == nil || valueDistribution == nil {
		// If either is nil, cannot perform split, return empty outcomes
		// Note: If valueDistribution is nil, the predicate can't be applied.
		// If latDistribution is nil, the result is empty latency splits.
		return thenLat, elseLat, nil
	}

	// Critical Assumption: Buckets are synchronized by index.
	if len(latDistribution.Buckets) != len(valueDistribution.Buckets) {
		// This indicates a fundamental issue in how the distributions were created or managed.
		err = fmt.Errorf("cannot split latency: value (len %d) and latency (len %d) distributions have different bucket counts",
			len(valueDistribution.Buckets), len(latDistribution.Buckets))
		return nil, nil, err // Return nil outcomes on error
	}

	for i := range valueDistribution.Buckets {
		valueBucket := valueDistribution.Buckets[i]
		latencyBucket := latDistribution.Buckets[i]

		// Sanity check weights (should match, but allow for float nuances)
		// if !ApproxEq(valueBucket.Weight, latencyBucket.Weight, 1e-9) {
		//     err = fmt.Errorf("bucket weight mismatch at index %d during latency split", i)
		//     return nil, nil, err
		// }

		if valuePredicate(valueBucket.Value) {
			// Use latencyBucket's weight and value
			thenLat.Buckets = append(thenLat.Buckets, latencyBucket)
		} else {
			elseLat.Buckets = append(elseLat.Buckets, latencyBucket)
		}
	}

	return thenLat, elseLat, nil
}

// reduceStateIfNeeded applies trimming if the state exceeds limits.
// Checks latency and known value types (AccessResult, RangedResult).
func (v *VM) reduceStateIfNeeded(state *VarState) (*VarState, error) {
	if state == nil {
		return nil, nil
	}
	reductionApplied := false
	newState := &VarState{ // Start with original references
		ValueOutcome:   state.ValueOutcome,
		LatencyOutcome: state.LatencyOutcome,
	}
	maxLen := v.maxOutcomeLen // Use configured max length

	// --- Reduce Latency Track ---
	if latOutcome, ok := state.LatencyOutcome.(*core.Outcomes[core.Duration]); ok && latOutcome != nil {
		if latOutcome.Len() > maxLen {
			// *** Use the configured reducer ***
			if v.durationReducer == nil {
				// Fallback or error if not set? Should be set by NewVM.
				fmt.Printf("Warning: VM durationReducer not set, defaulting to Adaptive.\n")
				newState.LatencyOutcome = core.ReduceDurationsAdaptive(latOutcome, maxLen)
			} else {
				newState.LatencyOutcome = v.durationReducer(latOutcome, maxLen)
			}
			reductionApplied = true
			// fmt.Printf("DEBUG: Reducing latency track (len %d > %d) using %T\n",
			//      latOutcome.Len(), maxLen, v.durationReducer) // Log which reducer was used
		}
	}

	// --- Reduce Value Track (for types that support it) ---
	triggerLen := maxLen + 50 // Use a higher trigger length for reduction
	switch valOutcome := state.ValueOutcome.(type) {
	case *core.Outcomes[core.AccessResult]:
		if valOutcome != nil && valOutcome.Len() > maxLen { // Check length against maxLen directly
			// fmt.Printf("DEBUG: Reducing AccessResult value track (len %d > %d)\n", valOutcome.Len(), maxLen) // Debug
			trimmerFuncGen := core.TrimToSize(triggerLen, maxLen) // TrimToSize includes merge+interpolate/anchor
			successes, failures := valOutcome.Split(core.AccessResult.IsSuccess)
			trimmedSuccesses := trimmerFuncGen(successes)
			trimmedFailures := trimmerFuncGen(failures)
			tempOutcome := &core.Outcomes[core.AccessResult]{}
			finalTrimmedVal := tempOutcome.Append(trimmedSuccesses, trimmedFailures)
			newState.ValueOutcome = finalTrimmedVal
			reductionApplied = true
		}
	case *core.Outcomes[core.RangedResult]:
		if valOutcome != nil && valOutcome.Len() > maxLen { // Check length against maxLen
			// fmt.Printf("DEBUG: Reducing RangedResult value track (len %d > %d)\n", valOutcome.Len(), maxLen) // Debug
			trimmerFuncGen := core.TrimToSizeRanged(triggerLen, maxLen, 0.1) // TrimToSizeRanged includes merge+interpolate
			successes, failures := valOutcome.Split(core.RangedResult.IsSuccess)
			trimmedSuccesses := trimmerFuncGen(successes)
			trimmedFailures := trimmerFuncGen(failures)
			tempOutcome := &core.Outcomes[core.RangedResult]{}
			finalTrimmedVal := tempOutcome.Append(trimmedSuccesses, trimmedFailures)
			newState.ValueOutcome = finalTrimmedVal
			reductionApplied = true
		}
		// Add other reducible types if needed
	}

	_ = reductionApplied // Keep variable, might be useful later

	return newState, nil
}
