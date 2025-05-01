package dsl

import (
	"fmt"

	"github.com/panyam/leetcoach/sdl/core"
)

// registerDefaultReducers populates the registry with standard combinations.
func (v *VM) registerDefaultReducers() {
	// --- AccessResult Reducers ---
	accRes := &core.Outcomes[core.AccessResult]{} // Example instance
	dur := &core.Outcomes[core.Duration]{}        // Example instance
	boolean := &core.Outcomes[bool]{}             // Example instance

	// AccessResult + AccessResult
	// The registered function needs to handle the any conversion and call core.And
	_ = v.RegisterSequentialReducer(accRes, accRes, func(a, b any) any {
		// Type assertion happens here inside the registered func
		return core.And(a.(*core.Outcomes[core.AccessResult]), b.(*core.Outcomes[core.AccessResult]), core.AndAccessResults)
	})
	// AccessResult + Duration
	_ = v.RegisterSequentialReducer(accRes, dur, func(a, b any) any {
		reducer := func(vA core.AccessResult, vB core.Duration) core.AccessResult {
			return core.AccessResult{Success: vA.Success, Latency: vA.Latency + vB}
		}
		return core.And(a.(*core.Outcomes[core.AccessResult]), b.(*core.Outcomes[core.Duration]), reducer)
	})
	// Duration + AccessResult
	_ = v.RegisterSequentialReducer(dur, accRes, func(a, b any) any {
		reducer := func(vA core.Duration, vB core.AccessResult) core.AccessResult {
			return core.AccessResult{Success: vB.Success, Latency: vA + vB.Latency}
		}
		return core.And(a.(*core.Outcomes[core.Duration]), b.(*core.Outcomes[core.AccessResult]), reducer)
	})

	// --- Duration Reducers ---
	// Duration + Duration
	_ = v.RegisterSequentialReducer(dur, dur, func(a, b any) any {
		latA, okA := a.(*core.Outcomes[core.Duration])
		latB, okB := b.(*core.Outcomes[core.Duration])
		if !okA || !okB {
			// Handle error or panic - this indicates VM called with wrong types
			panic(fmt.Sprintf("Duration reducer called with incompatible types: %T, %T", a, b))
		}
		reducer := func(vA, vB core.Duration) core.Duration { return vA + vB }
		// Ensure core.And handles nil inputs gracefully or check here
		if latA == nil {
			return latB
		} // If one is nil, return the other
		if latB == nil {
			return latA
		}
		return core.And(latA, latB, reducer)
	})

	// --- Bool Reducers ---
	// Bool + AccessResult
	_ = v.RegisterSequentialReducer(boolean, accRes, func(a, b any) any {
		reducer := func(vA bool, vB core.AccessResult) core.AccessResult { return vB } // Bool acts as filter via weights
		return core.And(a.(*core.Outcomes[bool]), b.(*core.Outcomes[core.AccessResult]), reducer)
	})
	// Bool + Bool
	_ = v.RegisterSequentialReducer(boolean, boolean, func(a, b any) any {
		reducer := func(vA, vB bool) bool { return vA && vB } // Example: logical AND
		return core.And(a.(*core.Outcomes[bool]), b.(*core.Outcomes[bool]), reducer)
	})

	// Add more combinations as needed (e.g., involving RangedResult)
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
