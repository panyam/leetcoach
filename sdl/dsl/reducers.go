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
