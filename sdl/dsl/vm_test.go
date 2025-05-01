// sdl/dsl/vm_test.go
package dsl

import (
	"errors"
	"testing"

	"github.com/panyam/leetcoach/sdl/core"
	// "fmt" // For debugging
)

// Helper to check VarState contents
func assertVarState(t *testing.T, state *VarState, expectedValType string, expectedLatVal float64) {
	t.Helper()
	if state == nil {
		t.Errorf("Expected non-nil VarState, got nil")
		return
	}

	// Check Value Type (using string representation for simplicity)
	valTypeStr := "nil"
	if state.ValueOutcome != nil {
		// Use helper func from vm.go (assuming it's accessible or copied)
		valTypeStr = getOutcomeTypeString(state.ValueOutcome)
	}
	if valTypeStr != expectedValType {
		t.Errorf("Expected ValueOutcome type %s, got %s", expectedValType, valTypeStr)
	}

	// Check Latency Value (should be deterministic zero for literals)
	latOutcome, ok := state.LatencyOutcome.(*core.Outcomes[core.Duration])
	if !ok {
		t.Errorf("Expected LatencyOutcome type *core.Outcomes[core.Duration], got %T", state.LatencyOutcome)
		return
	}
	latVal, ok := latOutcome.GetValue()
	if !ok {
		t.Errorf("Expected deterministic latency outcome, but got multiple buckets")
		return
	}
	// Use core.ApproxEq for float comparison
	if !core.ApproxEq(latVal, expectedLatVal, 1e-12) {
		t.Errorf("Expected latency value %.12f, got %.12f", expectedLatVal, latVal)
	}
}

// --- Test VM Initialization ---
func TestVM_NewVM(t *testing.T) {
	v := NewVM(20)
	if v == nil {
		t.Fatal("NewVM returned nil")
	}
	if v.stack == nil || len(v.stack) != 0 {
		t.Errorf("VM stack not initialized correctly")
	}
	if v.env == nil || len(v.env.store) != 0 {
		t.Error("VM environment not initialized correctly")
	}
	if v.internalFuncs == nil { // No internal funcs registered by default yet
		t.Error("VM internalFuncs map is nil")
	}
	// Check if default duration reducer is registered
	key := ReducerKey{LeftType: "*core.Outcomes[core.Duration]", RightType: "*core.Outcomes[core.Duration]"}
	if _, found := v.sequentialReducers[key]; !found {
		t.Error("Default Duration+Duration reducer not registered")
	}

	if v.maxOutcomeLen != 20 {
		t.Errorf("Expected maxOutcomeLen 20, got %d", v.maxOutcomeLen)
	}
}

// --- Test Literals (Phase 1 Updated) ---

func TestVM_Eval_LiteralInt(t *testing.T) {
	v := NewVM(10)
	literal := &LiteralExpr{Kind: "INT", Value: "42"}

	resultState, err := v.Eval(literal) // Eval returns state directly
	if err != nil {
		t.Fatalf("Eval(LiteralInt) failed: %v", err)
	}

	// Assert VarState contents
	assertVarState(t, resultState, "*core.Outcomes[int64]", 0.0)

	// Check the actual value
	outcome, _ := resultState.ValueOutcome.(*core.Outcomes[int64])
	val, ok := outcome.GetValue()
	if !ok || val != 42 {
		t.Errorf("Expected literal value 42, got %d (ok=%t)", val, ok)
	}
}

func TestVM_Eval_LiteralFloat(t *testing.T) {
	v := NewVM(10)
	literal := &LiteralExpr{Kind: "FLOAT", Value: "3.14"}

	resultState, err := v.Eval(literal)
	if err != nil {
		t.Fatalf("Eval(LiteralFloat) failed: %v", err)
	}

	assertVarState(t, resultState, "*core.Outcomes[float64]", 0.0)

	outcome, _ := resultState.ValueOutcome.(*core.Outcomes[float64])
	val, ok := outcome.GetValue()
	if !ok || val != 3.14 {
		t.Errorf("Expected literal value 3.14, got %f (ok=%t)", val, ok)
	}
}

func TestVM_Eval_LiteralString(t *testing.T) {
	v := NewVM(10)
	literal := &LiteralExpr{Kind: "STRING", Value: "hello"}

	resultState, err := v.Eval(literal)
	if err != nil {
		t.Fatalf("Eval(LiteralString) failed: %v", err)
	}

	assertVarState(t, resultState, "*core.Outcomes[string]", 0.0)

	outcome, _ := resultState.ValueOutcome.(*core.Outcomes[string])
	val, ok := outcome.GetValue()
	if !ok || val != "hello" {
		t.Errorf("Expected literal value 'hello', got '%s' (ok=%t)", val, ok)
	}
}

func TestVM_Eval_LiteralBool(t *testing.T) {
	v := NewVM(10)
	literal := &LiteralExpr{Kind: "BOOL", Value: "true"}

	resultState, err := v.Eval(literal)
	if err != nil {
		t.Fatalf("Eval(LiteralBool) failed: %v", err)
	}

	assertVarState(t, resultState, "*core.Outcomes[bool]", 0.0)

	outcome, _ := resultState.ValueOutcome.(*core.Outcomes[bool])
	val, ok := outcome.GetValue()
	if !ok || val != true {
		t.Errorf("Expected literal value true, got %v (ok=%t)", val, ok)
	}
}

// --- Test Environment & Identifier (Phase 1 Updated) ---

func TestEnvironment_SetGet_VarState(t *testing.T) {
	env := NewEnvironment()
	valOutcome := (&core.Outcomes[int64]{}).Add(1.0, 123)
	latOutcome := (&core.Outcomes[core.Duration]{}).Add(1.0, core.Millis(5))

	env.Set("myVar", valOutcome, latOutcome) // Use new Set signature

	state, ok := env.Get("myVar") // Get returns *VarState
	if !ok {
		t.Fatal("Failed to get 'myVar' from environment")
	}
	if state == nil {
		t.Fatal("Got nil VarState from environment")
	}
	if state.ValueOutcome != valOutcome {
		t.Error("ValueOutcome mismatch in retrieved VarState")
	}
	if state.LatencyOutcome != latOutcome {
		t.Error("LatencyOutcome mismatch in retrieved VarState")
	}

	_, ok = env.Get("notFound")
	if ok {
		t.Error("Expected 'notFound' to not be found")
	}
}

func TestVM_Eval_Identifier(t *testing.T) {
	v := NewVM(10)
	// Setup environment
	testVal := (&core.Outcomes[int64]{}).Add(1.0, 99)
	testLat := (&core.Outcomes[core.Duration]{}).Add(1.0, core.Millis(1))
	v.Env().Set("myVar", testVal, testLat)

	ident := &IdentifierExpr{Name: "myVar"}
	resultState, err := v.Eval(ident) // Eval returns state directly
	if err != nil {
		t.Fatalf("Eval(IdentifierExpr) failed: %v", err)
	}

	// Check the returned VarState
	if resultState == nil {
		t.Fatal("Eval(IdentifierExpr) returned nil VarState")
	}
	if resultState.ValueOutcome != testVal {
		t.Errorf("Expected identifier ValueOutcome mismatch")
	}
	if resultState.LatencyOutcome != testLat {
		t.Errorf("Expected identifier LatencyOutcome mismatch")
	}
}

func TestVM_Eval_Identifier_NotFound(t *testing.T) {
	v := NewVM(10)
	ident := &IdentifierExpr{Name: "missingVar"}

	resultState, err := v.Eval(ident) // Eval returns state directly
	if err == nil {
		t.Fatal("Eval(IdentifierExpr) should have failed for missing variable")
	}
	if resultState != nil {
		t.Errorf("Expected nil VarState on error, got %v", resultState)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

// --- Test Stack (Phase 1 Updated) ---

func TestVM_Stack_VarState(t *testing.T) {
	v := NewVM(10)

	_, err := v.pop()
	if !errors.Is(err, ErrStackUnderflow) {
		t.Errorf("Expected ErrStackUnderflow on empty pop, got %v", err)
	}

	state1 := &VarState{ValueOutcome: (&core.Outcomes[int]{}).Add(1.0, 1)}
	state2 := &VarState{ValueOutcome: (&core.Outcomes[string]{}).Add(1.0, "two")}

	v.push(state1)
	v.push(state2)

	if len(v.stack) != 2 {
		t.Fatalf("Expected stack length 2, got %d", len(v.stack))
	}

	top, ok := v.peek()
	if !ok || top != state2 {
		t.Errorf("Peek failed: expected state2, got %v (%t)", top, ok)
	}
	if len(v.stack) != 2 { // Peek should not change length
		t.Errorf("Stack length changed after peek: expected 2, got %d", len(v.stack))
	}

	pop2, err2 := v.pop()
	if err2 != nil {
		t.Fatalf("Error popping state2: %v", err2)
	}
	if pop2 != state2 {
		t.Errorf("Expected popped value state2, got %v", pop2)
	}
	if len(v.stack) != 1 {
		t.Errorf("Expected stack length 1 after pop, got %d", len(v.stack))
	}

	pop1, err1 := v.pop()
	if err1 != nil {
		t.Fatalf("Error popping state1: %v", err1)
	}
	if pop1 != state1 {
		t.Errorf("Expected popped value state1, got %v", pop1)
	}
	if len(v.stack) != 0 {
		t.Errorf("Expected stack length 0 after popping all, got %d", len(v.stack))
	}

	_, errEmpty := v.pop()
	if !errors.Is(errEmpty, ErrStackUnderflow) {
		t.Errorf("Expected ErrStackUnderflow on final empty pop, got %v", errEmpty)
	}
}

// --- Test Combine Helper (Internal - added for Phase 1) ---
func TestVM_CombineOutcomesAndReduce_Basic(t *testing.T) {
	v := NewVM(10) // Keep maxLen reasonable

	// State A: Value=bool(true), Latency=10ms
	valA := (&core.Outcomes[bool]{}).Add(1.0, true)
	latA := (&core.Outcomes[core.Duration]{}).Add(1.0, core.Millis(10))
	stateA := &VarState{ValueOutcome: valA, LatencyOutcome: latA}

	// State B: Value=bool(true), Latency=20ms
	valB := (&core.Outcomes[bool]{}).Add(1.0, true)
	latB := (&core.Outcomes[core.Duration]{}).Add(1.0, core.Millis(20))
	stateB := &VarState{ValueOutcome: valB, LatencyOutcome: latB}

	combinedState, err := v.combineOutcomesAndReduce(stateA, stateB)
	if err != nil {
		t.Fatalf("combineOutcomesAndReduce failed: %v", err)
	}

	// Expected Value: Should be the outcome from state B
	if combinedState.ValueOutcome != valB {
		t.Error("Expected combined ValueOutcome to be the same object as state B's ValueOutcome")
	}

	// Expected Latency: Should be combined latency (~30ms)
	expectedLatVal := core.Millis(10) + core.Millis(20)
	assertVarState(t, combinedState, "*core.Outcomes[bool]", expectedLatVal) // Latency check is inside assertVarState

	// Check the value itself
	combValOutcome, _ := combinedState.ValueOutcome.(*core.Outcomes[bool])
	combVal, ok := combValOutcome.GetValue()
	if !ok || combVal != true {
		t.Errorf("Expected combined value true, got %v (ok=%t)", combVal, ok)
	}
}

func TestVM_CombineOutcomesAndReduce_NilLatency(t *testing.T) {
	v := NewVM(10)

	// State A: Value=bool(true), Latency=10ms
	valA := (&core.Outcomes[bool]{}).Add(1.0, true)
	latA := (&core.Outcomes[core.Duration]{}).Add(1.0, core.Millis(10))
	stateA := &VarState{ValueOutcome: valA, LatencyOutcome: latA}

	// State B: Value=bool(true), Latency=nil (implicitly zero)
	valB := (&core.Outcomes[bool]{}).Add(1.0, true)
	stateB := &VarState{ValueOutcome: valB, LatencyOutcome: nil} // Nil latency

	combinedState, err := v.combineOutcomesAndReduce(stateA, stateB)
	if err != nil {
		t.Fatalf("combineOutcomesAndReduce failed: %v", err)
	}

	// Expected Value: state B's value
	if combinedState.ValueOutcome != valB {
		t.Error("Expected combined ValueOutcome to be state B's")
	}

	// Expected Latency: Should be latency from state A (~10ms)
	expectedLatVal := core.Millis(10)
	assertVarState(t, combinedState, "*core.Outcomes[bool]", expectedLatVal)
}

// TODO: Add combine test where reduction is triggered (e.g., latency track has > maxLen buckets). Requires a way to create complex latency outcomes easily for testing.

// --- Test Environment Scoping (No changes needed from previous version) ---
func TestEnvironment_Scoping(t *testing.T) {
	global := NewEnvironment()
	global.Set("globalVar", (&core.Outcomes[string]{}).Add(1.0, "hello"), zeroLatencyOutcome())

	local := NewEnclosedEnvironment(global)
	local.Set("localVar", (&core.Outcomes[int64]{}).Add(1.0, 123), zeroLatencyOutcome())

	// Get local
	stateLocal, okLocal := local.Get("localVar")
	if !okLocal || stateLocal.ValueOutcome.(*core.Outcomes[int64]).Buckets[0].Value != 123 {
		t.Errorf("Failed to get local var 'localVar'")
	}

	// Get global from local
	stateGlobal, okGlobal := local.Get("globalVar")
	if !okGlobal || stateGlobal.ValueOutcome.(*core.Outcomes[string]).Buckets[0].Value != "hello" {
		t.Errorf("Failed to get global var 'globalVar' from local scope")
	}

	// Try to get local from global (should fail)
	_, okLocalFromGlobal := global.Get("localVar")
	if okLocalFromGlobal {
		t.Error("Should not find local var in global scope")
	}

	// Shadowing
	local.Set("globalVar", (&core.Outcomes[int64]{}).Add(1.0, 999), zeroLatencyOutcome())
	stateShadowed, okShadowed := local.Get("globalVar")
	if !okShadowed || stateShadowed.ValueOutcome.(*core.Outcomes[int64]).Buckets[0].Value != 999 {
		t.Errorf("Failed to get shadowed var 'globalVar' from local scope")
	}

	// Original global should be unchanged
	stateOrigGlobal, okOrigGlobal := global.Get("globalVar")
	if !okOrigGlobal || stateOrigGlobal.ValueOutcome.(*core.Outcomes[string]).Buckets[0].Value != "hello" {
		t.Errorf("Global var was unexpectedly changed by local scope")
	}
}
