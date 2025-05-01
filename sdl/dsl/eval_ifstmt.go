package dsl

// sdl/dsl/eval_stmt.go (Latest evalIfStmt and helpers)

import (
	"errors"
	"fmt"

	// Still needed for getOutcomeTypeString if we keep that helper
	"github.com/panyam/leetcoach/sdl/core"
)

var (
	ErrInvalidConditionType = errors.New("if condition must evaluate to *core.Outcomes[bool] or access .Success field")
	ErrConditionSplitFailed = errors.New("failed to split outcome based on condition")
)

// outcomeHasWeight checks if an outcome object (expected *core.Outcomes[T]) is non-nil and has weight > 0
// putting it here temporarily
var ErrInvalidMemberAccess = errors.New("invalid member access")

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

// selectSplitPredicate dynamically creates the predicate function func(v any) bool
// based on the Condition expression.
func selectSplitPredicate(conditionExpr Expr, conditionValueOutcome any) (func(v any) bool, error) {
	// Case 1: Condition is direct boolean outcome
	if _, ok := conditionValueOutcome.(*core.Outcomes[bool]); ok {
		return func(v any) bool {
			b, _ := v.(bool) // Assume type assertion works
			return b
		}, nil
	}

	// Case 2: Condition is accessing '.Success' (e.g., myVar.Success)
	if memExpr, ok := conditionExpr.(*MemberAccessExpr); ok && memExpr.Member == "Success" {
		// Check the type of the outcome associated with the receiver
		switch conditionValueOutcome.(type) {
		case *core.Outcomes[core.AccessResult]:
			return func(v any) bool {
				ar, ok := v.(core.AccessResult)
				return ok && ar.Success // Check type assertion success
			}, nil
		case *core.Outcomes[core.RangedResult]:
			return func(v any) bool {
				rr, ok := v.(core.RangedResult)
				return ok && rr.Success // Check type assertion success
			}, nil
		// Add other types that have a '.Success' field if necessary
		default:
			return nil, fmt.Errorf("%w: cannot access '.Success' on value outcome type %T", ErrInvalidMemberAccess, conditionValueOutcome)
		}
	}

	// TODO: Handle other conditions? e.g., == checking discrete values?
	// Example: if myVar == MyEnumValue.CaseA
	// Requires evaluating the RHS, getting its deterministic value,
	// and creating a predicate `func(v any) bool { return v == rhsValue }`

	// If neither matches, the condition isn't supported yet
	return nil, fmt.Errorf("%w: expression %q (%T outcome) not supported", ErrInvalidConditionType, conditionExpr.String(), conditionValueOutcome)
}

// splitDualTrack splits both value and latency tracks using type assertions and core helpers.
// Returns (thenVal, thenLat), (elseVal, elseLat), error
func splitDualTrack(vm *VM, conditionExpr Expr, conditionState *VarState) (any, any, any, any, error) {
	if conditionState == nil || conditionState.ValueOutcome == nil {
		zeroLat := zeroLatencyOutcome()
		return nil, zeroLat, nil, zeroLat, nil
	}

	// 1. Select the predicate based on the condition expression and value outcome type
	splitPredicateAny, err := selectSplitPredicate(conditionExpr, conditionState.ValueOutcome)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// 2. Split the Value Outcome using type assertion and the existing Split method
	var thenValOutcome, elseValOutcome any

	switch valOutcome := conditionState.ValueOutcome.(type) {
	case *core.Outcomes[bool]:
		predicate := func(v bool) bool { return splitPredicateAny(v) }
		thenValOutcome, elseValOutcome = valOutcome.Split(predicate)
	case *core.Outcomes[core.AccessResult]:
		predicate := func(v core.AccessResult) bool { return splitPredicateAny(v) }
		thenValOutcome, elseValOutcome = valOutcome.Split(predicate)
	case *core.Outcomes[core.RangedResult]: // Added RangedResult
		predicate := func(v core.RangedResult) bool { return splitPredicateAny(v) }
		thenValOutcome, elseValOutcome = valOutcome.Split(predicate)
	// TODO: Add cases for other discrete types (enums, etc.)
	default:
		return nil, nil, nil, nil, fmt.Errorf("%w: cannot split value outcome of type %T for condition %q",
			ErrConditionSplitFailed, conditionState.ValueOutcome, conditionExpr.String())
	}

	// 3. Split the Latency Outcome using the core helper
	latOutcome, ok := conditionState.LatencyOutcome.(*core.Outcomes[core.Duration])
	if !ok || latOutcome == nil {
		zeroLat := zeroLatencyOutcome()
		return thenValOutcome, zeroLat, elseValOutcome, zeroLat, nil
	}

	var thenLat, elseLat *core.Outcomes[core.Duration]
	var splitErr error

	// Call the core helper, passing the correctly typed value outcome and predicate
	switch valueTypedOutcome := conditionState.ValueOutcome.(type) {
	case *core.Outcomes[bool]:
		predicate := func(v bool) bool { return splitPredicateAny(v) }
		thenLat, elseLat, splitErr = SplitLatenciesBasedOnValue(latOutcome, valueTypedOutcome, predicate)
	case *core.Outcomes[core.AccessResult]:
		predicate := func(v core.AccessResult) bool { return splitPredicateAny(v) }
		thenLat, elseLat, splitErr = SplitLatenciesBasedOnValue(latOutcome, valueTypedOutcome, predicate)
	case *core.Outcomes[core.RangedResult]:
		predicate := func(v core.RangedResult) bool { return splitPredicateAny(v) }
		thenLat, elseLat, splitErr = SplitLatenciesBasedOnValue(latOutcome, valueTypedOutcome, predicate)
	// Add other cases matching value split
	default:
		return nil, nil, nil, nil, fmt.Errorf("internal logic error: unexpected value type %T during latency split", conditionState.ValueOutcome)
	}

	if splitErr != nil {
		return nil, nil, nil, nil, fmt.Errorf("error splitting latency based on value: %w", splitErr)
	}

	return thenValOutcome, thenLat, elseValOutcome, elseLat, nil
}

// evalIfStmt handles conditional execution using dual tracks and stack accumulation.
// Returns error only (including ReturnValue signal). Pushes final result onto stack.
func (v *VM) evalIfStmt(stmt *IfStmt) error {
	// 1. Evaluate the Condition Expression -> yields the state whose value is tested
	conditionState, err := v.Eval(stmt.Condition)
	if err != nil {
		var retVal *ReturnValue // Check for early return during condition eval
		if errors.As(err, &retVal) {
			return err
		} // Propagate signal
		return fmt.Errorf("error evaluating if condition: %w", err)
	}
	if conditionState == nil {
		return fmt.Errorf("internal error: nil state from condition evaluation")
	}

	// 2. Split the condition's dual track based on the predicate
	thenValCtx, thenLatCtx, elseValCtx, elseLatCtx, splitErr := splitDualTrack(v, stmt.Condition, conditionState)
	if splitErr != nil {
		return fmt.Errorf("failed to split distributions for if: %w", splitErr)
	}

	// --- Prepare contexts for branch evaluation ---
	thenCtxState := &VarState{ValueOutcome: thenValCtx, LatencyOutcome: thenLatCtx}
	elseCtxState := &VarState{ValueOutcome: elseValCtx, LatencyOutcome: elseLatCtx}

	// --- Determine which branches have probability ---
	thenHasWeight := outcomeHasWeight(thenCtxState.ValueOutcome)
	elseHasWeight := outcomeHasWeight(elseCtxState.ValueOutcome)

	var finalThenResultState *VarState
	var finalElseResultState *VarState
	var branchErr error // Tracks errors from branches

	// 3. Evaluate Then Branch (if it has weight)
	if thenHasWeight {
		finalThenResultState, branchErr = v.evalBlockStmt(stmt.Then, v.env, thenCtxState)
		if branchErr != nil {
			var retVal *ReturnValue
			if errors.As(branchErr, &retVal) {
				v.push(retVal.Value) // Push the returned state
				return branchErr     // Propagate signal
			}
			return fmt.Errorf("error in 'then' branch: %w", branchErr)
		}
	} else {
		finalThenResultState = nil // No result for this path
	}

	// 4. Evaluate Else Branch (if it exists and has weight)
	if stmt.Else != nil && elseHasWeight {
		finalElseResultState, branchErr = v.evalBlockStmt(stmt.Else, v.env, elseCtxState)
		if branchErr != nil {
			var retVal *ReturnValue
			if errors.As(branchErr, &retVal) {
				v.push(retVal.Value) // Push the returned state
				return branchErr     // Propagate signal
			}
			return fmt.Errorf("error in 'else' branch: %w", branchErr)
		}
	} else if stmt.Else == nil && elseHasWeight {
		// No Else block, result is the original 'else' context state
		finalElseResultState = elseCtxState
	} else {
		finalElseResultState = nil // No result for this path
	}

	// 5. Combine Results using Append
	var finalValOutcome any
	var finalLatOutcome any

	// Get Value Outcomes to Append
	var thenVal, elseVal any
	if finalThenResultState != nil {
		thenVal = finalThenResultState.ValueOutcome
	}
	if finalElseResultState != nil {
		elseVal = finalElseResultState.ValueOutcome
	}
	// Use type switch for append (replaces core.AppendOutcomes)
	switch tv := thenVal.(type) {
	case nil:
		finalValOutcome = elseVal // Handles elseVal potentially being nil too
	case *core.Outcomes[bool]:
		ev, ok := elseVal.(*core.Outcomes[bool])
		if elseVal == nil || ok {
			if tv == nil {
				tv = &core.Outcomes[bool]{}
			}
			finalValOutcome = tv.Append(ev)
		} else {
			return fmt.Errorf("if branch value type mismatch: then=%T, else=%T", thenVal, elseVal)
		}
	case *core.Outcomes[core.AccessResult]:
		ev, ok := elseVal.(*core.Outcomes[core.AccessResult])
		if elseVal == nil || ok {
			if tv == nil {
				tv = &core.Outcomes[core.AccessResult]{ /* Set And? */ }
			}
			finalValOutcome = tv.Append(ev)
		} else {
			return fmt.Errorf("if branch value type mismatch: then=%T, else=%T", thenVal, elseVal)
		}
	// Add other discrete types if needed
	default:
		return fmt.Errorf("unsupported type %T in if branch value combination", thenVal)
	}

	// Get Latency Outcomes to Append
	var thenLat, elseLat any
	if finalThenResultState != nil {
		thenLat = finalThenResultState.LatencyOutcome
	}
	if finalElseResultState != nil {
		elseLat = finalElseResultState.LatencyOutcome
	}
	// Latency is always Outcomes[Duration] or nil
	lat1, ok1 := thenLat.(*core.Outcomes[core.Duration])
	lat2, ok2 := elseLat.(*core.Outcomes[core.Duration])
	if (!ok1 && thenLat != nil) || (!ok2 && elseLat != nil) {
		return fmt.Errorf("internal error: non-duration outcome in latency track (%T, %T)", thenLat, elseLat)
	}
	if lat1 == nil {
		lat1 = &core.Outcomes[core.Duration]{}
	} // Ensure non-nil receiver for Append
	finalLatOutcome = lat1.Append(lat2) // Append handles nil lat2

	// Final combined state
	finalCombinedState := &VarState{
		ValueOutcome:   finalValOutcome,
		LatencyOutcome: finalLatOutcome,
	}

	// Ensure latency is valid if value exists
	if finalCombinedState.ValueOutcome != nil && finalCombinedState.LatencyOutcome == nil {
		finalCombinedState.LatencyOutcome = zeroLatencyOutcome()
	}
	// If everything resulted in nil (e.g., empty branches), create identity state
	if finalCombinedState.ValueOutcome == nil && finalCombinedState.LatencyOutcome == nil {
		finalCombinedState = createIdentityState()
	}

	// 6. Apply Reduction
	reducedState, reduceErr := v.reduceStateIfNeeded(finalCombinedState)
	if reduceErr != nil {
		return fmt.Errorf("error reducing combined IfStmt result: %w", reduceErr)
	}

	// 7. Push the final result onto the stack
	v.push(reducedState)

	return nil // Success
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
