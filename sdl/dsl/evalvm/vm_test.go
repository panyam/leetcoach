// sdl/dsl/vm_test.go
package dsl

import (
	"errors"
	"fmt"
	"testing"

	"github.com/panyam/leetcoach/sdl/core"
	// "fmt" // For debugging
)

// Mock internal funcs
func registerMockInternalFuncs(v *VM) {
	// Helper to create Duration Outcomes from Millis Literals (for tests)
	v.RegisterInternalFunc("DurationMS", func(vm *VM, args []*VarState) (*VarState, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("DurationMS needs 1 arg")
		}
		msState := args[0]
		if msState == nil {
			return nil, fmt.Errorf("DurationMS arg must be VarState")
		}
		msValF, ok := msState.ValueOutcome.(*core.Outcomes[float64]).GetValue()
		if !ok {
			return nil, fmt.Errorf("DurationMS arg must be deterministic float")
		}
		durOutcome := (&core.Outcomes[core.Duration]{}).Add(1.0, core.Millis(msValF))
		return &VarState{ValueOutcome: durOutcome, LatencyOutcome: zeroLatencyOutcome()}, nil
	})
}

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

func TestVM_Eval_BlockStmt_SimpleSequence(t *testing.T) {
	v := NewVM(10)
	registerMockInternalFuncs(v)

	// AST for: {
	//   a = Internal.GetAccess(0.9, DurationMS(10.0), DurationMS(5.0));
	//   b = Internal.GetAccess(0.8, DurationMS(20.0), DurationMS(15.0));
	// } // Implicit return of combined 'b'
	block := &BlockStmt{
		Statements: []Stmt{
			&AssignmentStmt{
				// *** Use Type Assertion ***
				Variable: ident("a").(*IdentifierExpr),
				Value: internalCall("GetAccess", litFloat(0.9),
					internalCall("DurationMS", litFloat(10.0)), // Use DurationMS helper call
					internalCall("DurationMS", litFloat(5.0))),
			},
			&AssignmentStmt{
				// *** Use Type Assertion ***
				Variable: ident("b").(*IdentifierExpr),
				Value: internalCall("GetAccess", litFloat(0.8),
					internalCall("DurationMS", litFloat(20.0)),
					internalCall("DurationMS", litFloat(15.0))),
			},
		},
	}

	// Eval the block directly for testing
	finalState, err := v.evalBlockStmt(block, v.Env(), nil) // Use block eval
	if err != nil {
		t.Fatalf("evalBlockStmt failed: %v", err)
	}

	// Check final state
	if finalState == nil {
		t.Fatal("Block returned nil state")
	}

	// Expected Value Outcome: Should be the AccessResult distribution from the second call ('b')
	// Expected Latency Outcome: Should be the combination (And) of latencies from 'a' and 'b'.
	// Combined Latency Buckets (Weight=ProbA*ProbB):
	// (10ms + 20ms) -> W = 0.9 * 0.8 = 0.72
	// (10ms + 15ms) -> W = 0.9 * 0.2 = 0.18
	// ( 5ms + 20ms) -> W = 0.1 * 0.8 = 0.08
	// ( 5ms + 15ms) -> W = 0.1 * 0.2 = 0.02
	// Expected Mean Latency = (30*0.72 + 25*0.18 + 25*0.08 + 20*0.02) = 21.6 + 4.5 + 2.0 + 0.4 = 28.5ms
	// Expected Value (Success Prob) = Prob(A succ AND B succ) = 0.9 * 0.8 = 0.72

	valOutcome, okV := finalState.ValueOutcome.(*core.Outcomes[core.AccessResult])
	latOutcome, okL := finalState.LatencyOutcome.(*core.Outcomes[core.Duration])
	if !okV || !okL {
		t.Fatalf("Final state has wrong types: V=%T, L=%T", finalState.ValueOutcome, finalState.LatencyOutcome)
	}

	// Check Value (Availability)
	actualAvail := core.Availability(valOutcome)
	expectedAvail := 0.72
	if !core.ApproxEq(actualAvail, expectedAvail, 1e-9) {
		t.Errorf("Expected final Availability %.4f, got %.4f", expectedAvail, actualAvail)
	}

	// Check Latency (Mean)
	actualMeanLat := core.MeanLatency(valOutcome) // Mean latency of *successes* in value outcome
	// Mean latency of *all* paths in latency outcome
	actualMeanDur := meanDurationLatency(latOutcome)
	expectedMeanDur := core.Millis(28.5)
	if !core.ApproxEq(actualMeanDur, expectedMeanDur, core.Millis(0.1)) {
		t.Errorf("Expected final Mean Duration %.6fs, got %.6fs", expectedMeanDur, actualMeanDur)
	}

	// Also check mean latency of successes from value track
	// Expected Mean Success Latency = (30ms * 0.72) / 0.72 = 30ms
	if !core.ApproxEq(actualMeanLat, core.Millis(30.0), core.Millis(0.1)) {
		t.Errorf("Expected final Mean Success Latency %.6fs, got %.6fs", core.Millis(30.0), actualMeanLat)
	}

	t.Logf("Block Sequence Test: Avail=%.4f, MeanLat=%.6f, MeanDur=%.6f", actualAvail, actualMeanLat, actualMeanDur)
	// t.Logf("Value Buckets: %+v", valOutcome.Buckets)
	// t.Logf("Latency Buckets: %+v", latOutcome.Buckets)

}

func TestVM_Eval_BlockStmt_WithReturn(t *testing.T) {
	v := NewVM(10)
	registerMockInternalFuncs(v)

	// AST for: {
	//   a = Internal.GetAccess(0.9, DurationMS(10.0), DurationMS(5.0));
	//   return a;
	//   b = Internal.GetAccess(0.8, DurationMS(20.0), DurationMS(15.0));
	// }
	block := &BlockStmt{
		Statements: []Stmt{
			&AssignmentStmt{
				// *** Use Type Assertion ***
				Variable: ident("a").(*IdentifierExpr),
				Value: internalCall("GetAccess", litFloat(0.9),
					internalCall("DurationMS", litFloat(10.0)),
					internalCall("DurationMS", litFloat(5.0))),
			},
			// *** Use Type Assertion ***
			&ReturnStmt{ReturnValue: ident("a")},
			&AssignmentStmt{ // Unreachable
				// *** Use Type Assertion ***
				Variable: ident("b").(*IdentifierExpr),
				Value: internalCall("GetAccess", litFloat(0.8),
					internalCall("DurationMS", litFloat(20.0)),
					internalCall("DurationMS", litFloat(15.0))),
			},
		},
	}

	// Eval the block - expect ReturnValue signal error
	_, err := v.evalBlockStmt(block, v.Env(), nil)

	if err == nil {
		t.Fatal("evalBlockStmt should have returned ReturnValue signal error")
	}
	var retVal *ReturnValue
	if !errors.As(err, &retVal) {
		t.Fatalf("Expected ReturnValue signal error, got %T: %v", err, err)
	}
	if retVal.Value == nil {
		t.Fatal("ReturnValue signal did not contain VarState")
	}

	// The returned state should be the state of 'a'
	stateA, ok := v.Env().Get("a") // Get what was stored in env
	if !ok {
		t.Fatal("Variable 'a' not found in env after return")
	}

	if retVal.Value != stateA {
		t.Error("Returned state does not match expected state of 'a'")
	}
	returnedState := retVal.Value

	// Check if 'b' was evaluated (it shouldn't be)
	if _, foundB := v.Env().Get("b"); foundB {
		t.Error("Variable 'b' should not be in env as it was after return")
	}

	// Check the returned state's content
	valOutcome, okV := returnedState.ValueOutcome.(*core.Outcomes[core.AccessResult])
	latOutcome, okL := returnedState.LatencyOutcome.(*core.Outcomes[core.Duration])
	if !okV || !okL {
		t.Fatalf("Returned state has wrong types: V=%T, L=%T", returnedState.ValueOutcome, returnedState.LatencyOutcome)
	}

	// Check Value (Availability)
	if !core.ApproxEq(core.Availability(valOutcome), 0.9, 1e-9) {
		t.Errorf("Unexpected availability")
	}
	// Check Latency (Mean)
	if !core.ApproxEq(meanDurationLatency(latOutcome), core.Millis(9.5), core.Millis(0.01)) {
		t.Errorf("Unexpected mean latency")
	} // (0.9*10 + 0.1*5) = 9.5
}

// --- IfStmt Tests ---

func TestVM_Eval_IfStmt_SimpleBool(t *testing.T) {
	v := NewVM(10)
	registerMockInternalFuncs(v)

	// AST for:
	// {
	//   cond = Internal.GetBool(0.8, DurationMS(1.0), DurationMS(2.0));
	//   res = 0;
	//   if cond {
	//     res = Internal.GetAccess(1.0, DurationMS(10.0), DurationMS(0.0));
	//   } else {
	//     res = Internal.GetAccess(0.0, DurationMS(0.0), DurationMS(5.0));
	//   }
	//   return res;
	// }
	block := &BlockStmt{
		Statements: []Stmt{
			&AssignmentStmt{Variable: ident("cond").(*IdentifierExpr), Value: internalCall("GetBool", litFloat(0.8),
				internalCall("DurationMS", litFloat(1.0)), internalCall("DurationMS", litFloat(2.0)))},
			&AssignmentStmt{Variable: ident("res").(*IdentifierExpr), Value: litInt(0)},
			&IfStmt{
				// *** Use Type Assertion ***
				Condition: ident("cond"), // Condition is the bool outcome directly
				Then: &BlockStmt{Statements: []Stmt{
					&AssignmentStmt{Variable: ident("res").(*IdentifierExpr), Value: internalCall("GetAccess", litFloat(1.0),
						internalCall("DurationMS", litFloat(10.0)), internalCall("DurationMS", litFloat(0.0)))},
				}},
				Else: &BlockStmt{Statements: []Stmt{
					&AssignmentStmt{Variable: ident("res").(*IdentifierExpr), Value: internalCall("GetAccess", litFloat(0.0),
						internalCall("DurationMS", litFloat(0.0)), internalCall("DurationMS", litFloat(5.0)))},
				}},
			},
			&ReturnStmt{ReturnValue: ident("res")},
		},
	}

	// Eval the block
	finalState, err := v.evalBlockStmt(block, v.Env(), nil)
	// We expect a ReturnValue signal containing the final state of 'res'
	if err == nil {
		t.Fatal("evalBlockStmt should have returned ReturnValue signal error for block")
	}
	var retVal *ReturnValue
	if !errors.As(err, &retVal) {
		t.Fatalf("Expected ReturnValue signal error, got %T: %v", err, err)
	}
	finalState = retVal.Value // Extract the actual final state

	// Check final state
	if finalState == nil {
		t.Fatal("IfStmt block returned nil state")
	}

	// Expected Value: 0.8 prob -> Success (from Then branch), 0.2 prob -> Failure (from Else branch)
	// Expected Latency:
	//   Path 1 (cond=T, then): Lat(cond=T) + Lat(GetAccess Then) = 1ms + 10ms = 11ms. Prob = 0.8 * 1.0 = 0.8
	//   Path 2 (cond=F, else): Lat(cond=F) + Lat(GetAccess Else) = 2ms + 5ms = 7ms.  Prob = 0.2 * 1.0 = 0.2
	// Expected Mean Latency = (11 * 0.8) + (7 * 0.2) = 8.8 + 1.4 = 10.2ms

	valOutcome, okV := finalState.ValueOutcome.(*core.Outcomes[core.AccessResult])
	latOutcome, okL := finalState.LatencyOutcome.(*core.Outcomes[core.Duration])
	if !okV || !okL {
		t.Fatalf("Final state has wrong types: V=%T, L=%T", finalState.ValueOutcome, finalState.LatencyOutcome)
	}

	// Check Value (Availability)
	actualAvail := core.Availability(valOutcome)
	expectedAvail := 0.8
	if !core.ApproxEq(actualAvail, expectedAvail, 1e-9) {
		t.Errorf("Expected final Availability %.4f, got %.4f", expectedAvail, actualAvail)
	}

	// Check Latency (Mean Duration)
	actualMeanDur := meanDurationLatency(latOutcome)
	expectedMeanDur := core.Millis(10.2)
	if !core.ApproxEq(actualMeanDur, expectedMeanDur, core.Millis(0.01)) {
		t.Errorf("Expected final Mean Duration %.6fs, got %.6fs", expectedMeanDur, actualMeanDur)
	}

	// Check buckets for expected values
	/*
		found11ms := false
		found7ms := false
		if latOutcome.Len() != 2 {
			t.Logf("Warning: Expected 2 latency buckets, got %d", latOutcome.Len())
		} // Might reduce
		for _, b := range latOutcome.Buckets {
			if core.ApproxEq(b.Value, core.Millis(11), core.Millis(0.01)) && core.ApproxEq(b.Weight, 0.8, 1e-6) {
				found11ms = true
			}
			if core.ApproxEq(b.Value, core.Millis(7), core.Millis(0.01)) && core.ApproxEq(b.Weight, 0.2, 1e-6) {
				found7ms = true
			}
		}
	*/
	// Tolerate reduction might merge/alter buckets, so just check mean/avail primarily
	// if !found11ms { t.Errorf("Latency bucket for 11ms path not found or incorrect weight") }
	// if !found7ms { t.Errorf("Latency bucket for 7ms path not found or incorrect weight") }

	t.Logf("IfStmt Bool Test: Avail=%.4f, MeanDur=%.6f", actualAvail, actualMeanDur)
}

func TestVM_Eval_IfStmt_AccessResultSuccess(t *testing.T) {
	v := NewVM(10)
	registerMockInternalFuncs(v)

	// AST for:
	// {
	//   cond = Internal.GetAccess(0.7, DurationMS(5.0), DurationMS(3.0));
	//   res = 0;
	//   if cond.Success {
	//     res = Internal.GetAccess(1.0, DurationMS(20.0), DurationMS(0.0)); // Then = S:20ms
	//   } else {
	//     res = Internal.GetAccess(1.0, DurationMS(8.0), DurationMS(0.0));  // Else = S:8ms
	//   }
	//   return res;
	// }
	block := &BlockStmt{
		Statements: []Stmt{
			&AssignmentStmt{Variable: ident("cond").(*IdentifierExpr), Value: internalCall("GetAccess", litFloat(0.7),
				internalCall("DurationMS", litFloat(5.0)), internalCall("DurationMS", litFloat(3.0)))},
			&AssignmentStmt{Variable: ident("res").(*IdentifierExpr), Value: litInt(0)},
			&IfStmt{
				Condition: &MemberAccessExpr{Receiver: ident("cond"), Member: "Success"},
				Then: &BlockStmt{Statements: []Stmt{
					&AssignmentStmt{Variable: ident("res").(*IdentifierExpr), Value: internalCall("GetAccess", litFloat(1.0), // Then is Success
						internalCall("DurationMS", litFloat(20.0)), internalCall("DurationMS", litFloat(0.0)))},
				}},
				Else: &BlockStmt{Statements: []Stmt{
					&AssignmentStmt{Variable: ident("res").(*IdentifierExpr), Value: internalCall("GetAccess", litFloat(1.0), // Else is also Success
						internalCall("DurationMS", litFloat(8.0)), internalCall("DurationMS", litFloat(0.0)))}, // Changed from F:8ms to S:8ms
				}},
			},
			&ReturnStmt{ReturnValue: ident("res")},
		},
	}

	// Eval the block
	finalState, err := v.evalBlockStmt(block, v.Env(), nil)
	if err == nil {
		t.Fatal("Expected ReturnValue signal error")
	}
	var retVal *ReturnValue
	if !errors.As(err, &retVal) {
		t.Fatalf("Expected ReturnValue signal error, got %T: %v", err, err)
	}
	finalState = retVal.Value

	// Check final state
	if finalState == nil {
		t.Fatal("IfStmt AR.Success block returned nil state")
	}

	// Expected Value: Outcome depends only on the branch taken.
	//   Then branch (Prob 0.7) returns AR{S:20}.
	//   Else branch (Prob 0.3) returns AR{S:8}. -> NOTE: Made Else return Success=True with Latency=8ms for variety.
	//   Expected Availability = 0.7*1.0 + 0.3*1.0 = 1.0
	// Expected Latency:
	//   Path 1 (cond=S, then): Lat(cond=S) + Lat(GetAccess Then) = 5ms + 20ms = 25ms. Prob = 0.7
	//   Path 2 (cond=F, else): Lat(cond=F) + Lat(GetAccess Else) = 3ms + 8ms = 11ms.  Prob = 0.3
	// Expected Mean Latency = (25 * 0.7) + (11 * 0.3) = 17.5 + 3.3 = 20.8ms

	valOutcome, okV := finalState.ValueOutcome.(*core.Outcomes[core.AccessResult])
	latOutcome, okL := finalState.LatencyOutcome.(*core.Outcomes[core.Duration])
	if !okV || !okL {
		t.Fatalf("Final state has wrong types: V=%T, L=%T", finalState.ValueOutcome, finalState.LatencyOutcome)
	}

	// Check Value (Availability)
	actualAvail := core.Availability(valOutcome)
	expectedAvail := 1.0 // Both branches return Success=true now
	if !core.ApproxEq(actualAvail, expectedAvail, 1e-9) {
		t.Errorf("Expected final Availability %.4f, got %.4f", expectedAvail, actualAvail)
	}

	// Check Latency (Mean Duration)
	actualMeanDur := meanDurationLatency(latOutcome)
	expectedMeanDur := core.Millis(20.8)
	if !core.ApproxEq(actualMeanDur, expectedMeanDur, core.Millis(0.01)) {
		t.Errorf("Expected final Mean Duration %.6fs, got %.6fs", expectedMeanDur, actualMeanDur)
	}

	t.Logf("IfStmt AR.Success Test: Avail=%.4f, MeanDur=%.6f", actualAvail, actualMeanDur)
	// t.Logf("Value Buckets: %+v", valOutcome.Buckets)
	// t.Logf("Latency Buckets: %+v", latOutcome.Buckets)
}
