package dsl

import "github.com/panyam/leetcoach/sdl/core"

var (
	// Lazily initialize these? Or create here? Creating here is simpler.
	nilState   *VarState
	trueState  *VarState
	falseState *VarState
)

func init() {
	// Initialize predefined states used by OpNil, OpTrue, OpFalse
	nilState = &VarState{ValueOutcome: nil, LatencyOutcome: zeroLatencyOutcome()} // No specific value type for nil
	trueState = &VarState{
		ValueOutcome:   (&core.Outcomes[bool]{}).Add(1.0, true),
		LatencyOutcome: zeroLatencyOutcome(),
	}
	falseState = &VarState{
		ValueOutcome:   (&core.Outcomes[bool]{}).Add(1.0, false),
		LatencyOutcome: zeroLatencyOutcome(),
	}
}

// VarState holds the dual-track outcomes for a variable or expression result.
type VarState struct {
	ValueOutcome   any // *core.Outcomes[V] (V is discrete type)
	LatencyOutcome any // *core.Outcomes[core.Duration]
}

// createIdentityState provides a neutral starting point for composition.
// Value: bool=true (neutral for AND-like success combination)
// Latency: 0ms
func createIdentityState() *VarState {
	// Ensure these helpers return non-nil outcomes
	val := identityValueOutcome() // Outcomes[bool]{1.0=>true}
	lat := zeroLatencyOutcome()   // Outcomes[Duration]{1.0=>0}
	if val == nil || lat == nil {
		panic("internal error: failed to create identity/zero outcomes")
	}
	return &VarState{ValueOutcome: val, LatencyOutcome: lat}
}

// Add Safe Accessor helpers to VarState to handle nil pointers gracefully
func (vs *VarState) SafeValueOutcome() any {
	if vs == nil {
		return nil
	}
	return vs.ValueOutcome
}

func (vs *VarState) SafeLatencyOutcome() any {
	if vs == nil {
		return nil
	}
	return vs.LatencyOutcome
}

func createNilState() *VarState { return nilState }
func createBoolState(b bool) *VarState {
	if b {
		return trueState
	}
	return falseState
}
