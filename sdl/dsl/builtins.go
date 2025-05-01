package dsl

import (
	"fmt"

	"github.com/panyam/leetcoach/sdl/core"
)

// registerDefaultHandlers - called by NewVM to setup handlers for common ops.
func (v *VM) registerDefaultHandlers() {
	// Register stubs for now, pointing to specific handler funcs
	v.RegisterInternalHandler(HandlerAdd, v.vm_handle_add)
	v.RegisterInternalHandler(HandlerSubtract, v.vm_handle_unimplemented_binary)
	v.RegisterInternalHandler(HandlerMultiply, v.vm_handle_unimplemented_binary)
	v.RegisterInternalHandler(HandlerDivide, v.vm_handle_unimplemented_binary)
	v.RegisterInternalHandler(HandlerNegate, v.vm_handle_unimplemented_unary)
	v.RegisterInternalHandler(HandlerNot, v.vm_handle_unimplemented_unary)
	v.RegisterInternalHandler(HandlerEqual, v.vm_handle_unimplemented_binary)
	v.RegisterInternalHandler(HandlerNotEqual, v.vm_handle_unimplemented_binary)
	v.RegisterInternalHandler(HandlerGreater, v.vm_handle_unimplemented_binary)
	v.RegisterInternalHandler(HandlerGreaterEqual, v.vm_handle_unimplemented_binary)
	v.RegisterInternalHandler(HandlerLess, v.vm_handle_unimplemented_binary)
	v.RegisterInternalHandler(HandlerLessEqual, v.vm_handle_unimplemented_binary)
	v.RegisterInternalHandler(HandlerAnd, v.vm_handle_unimplemented_binary)
	v.RegisterInternalHandler(HandlerOr, v.vm_handle_unimplemented_binary)
	v.RegisterInternalHandler(HandlerAndSeq, v.vm_handle_and_seq)
	v.RegisterInternalHandler(HandlerDelay, v.vm_handle_delay)
	v.RegisterInternalHandler(HandlerIf, v.vm_handle_if)
	v.RegisterInternalHandler(HandlerDistribute, v.vm_handle_distribute)
}

func (v *VM) vm_handle_add(vm *VM, args []*VarState) (*VarState, error) {
	// Binary Op: Pop Right, Pop Left
	rightState, err := v.pop()
	if err != nil {
		return nil, err
	}
	leftState, err := v.pop()
	if err != nil {
		return nil, err
	}

	// TODO: Perform actual type checking and probabilistic ADDITION

	// --- Perform Value Addition (Deterministic Example) ---
	var resultValOutcome any
	var combinedErr error

	// Check types and perform operation (for int64 + int64 -> int64)
	leftValInt, okL := leftState.ValueOutcome.(*core.Outcomes[int64])
	rightValInt, okR := rightState.ValueOutcome.(*core.Outcomes[int64])

	if okL && okR {
		// Ensure deterministic
		lVal, lOK := leftValInt.GetValue()
		rVal, rOK := rightValInt.GetValue()
		if lOK && rOK {
			resultVal := lVal + rVal
			resultValOutcome = (&core.Outcomes[int64]{}).Add(1.0, resultVal)
		} else {
			combinedErr = v.runtimeError("cannot add non-deterministic int64 outcomes yet")
		}
	} else {
		// TODO: Handle other type combinations (float+float, etc.)
		combinedErr = v.runtimeError("unsupported operand types for '+': %T and %T",
			leftState.ValueOutcome, rightState.ValueOutcome)
	}
	if combinedErr != nil {
		return nil, combinedErr // Return error if value combination failed
	}

	combinedState, err := v.combineOutcomesAndReduce(leftState, rightState)
	if err != nil {
		return nil, fmt.Errorf("add failed during combination: %v", err)
	}

	// --- Push combined state with NEW value outcome ---
	finalState := &VarState{ValueOutcome: resultValOutcome, LatencyOutcome: combinedState.LatencyOutcome}
	if err := v.push(finalState); err != nil {
		return nil, err
	}

	return nil, nil // Success, result is on stack
}

func (v *VM) vm_handle_unimplemented_binary(vm *VM, args []*VarState) (*VarState, error) {
	return nil, v.runtimeError("binary handler not implemented")
}
func (v *VM) vm_handle_unimplemented_unary(vm *VM, args []*VarState) (*VarState, error) {
	return nil, v.runtimeError("unary handler not implemented")
}

// vm_handle_and_seq calls the standard combine function
func (v *VM) vm_handle_and_seq(vm *VM, args []*VarState) (*VarState, error) {
	stateB, err := v.pop()
	if err != nil {
		return nil, err
	}
	stateA, err := v.pop()
	if err != nil {
		return nil, err
	}

	combinedState, combineErr := v.combineOutcomesAndReduce(stateA, stateB)
	if combineErr != nil {
		return nil, v.runtimeError("and_seq combination failed: %v", combineErr)
	}

	if err := v.push(combinedState); err != nil {
		return nil, err
	}
	return nil, nil // Success
}

func (v *VM) vm_handle_delay(vm *VM, args []*VarState) (*VarState, error) {
	durationState, err := v.pop()
	if err != nil {
		return nil, err
	}
	currentState, err := v.pop()
	if err != nil {
		return nil, err
	}

	durationOutcome, ok := durationState.ValueOutcome.(*core.Outcomes[core.Duration])
	if !ok {
		return nil, v.runtimeError("delay operand must be Outcomes[Duration], got %T", durationState.ValueOutcome)
	}
	// IMPORTANT: Delay can be probabilistic! Combine probabilistically.
	// We need a way to combine currentState.Latency + durationOutcome (probabilistic)

	// --- Probabilistic Delay Combination ---
	// Create a dummy value state for the delay duration
	delayValState := &VarState{
		ValueOutcome:   identityValueOutcome(), // Use neutral value
		LatencyOutcome: durationOutcome,        // Use the probabilistic duration
	}

	// Combine Current State THEN Delay State
	combinedState, combineErr := v.combineOutcomesAndReduce(currentState, delayValState)
	if combineErr != nil {
		return nil, v.runtimeError("delay combination failed: %v", combineErr)
	}

	if err := v.push(combinedState); err != nil {
		return nil, err
	}
	return nil, nil // Success
}

func (v *VM) vm_handle_if(vm *VM, args []*VarState) (*VarState, error) {
	// Args might contain jump offsets passed via operands? Or vm looks ahead?
	// Complex logic delegated to eval_ifstmt.go (needs adapting to bytecode context)

	// Stack order (from OpCallFunc): ElseChunkIdxConst, ThenChunkIdxConst, ConditionState
	elseChunkIdxState, err := v.pop()
	if err != nil {
		return nil, err
	}
	thenChunkIdxState, err := v.pop()
	if err != nil {
		return nil, err
	}
	conditionState, err := v.pop()
	if err != nil {
		return nil, err
	}

	// Get the actual chunk indices from the constants
	_ /*elseChunkIndex*/, ok := elseChunkIdxState.ValueOutcome.(*core.Outcomes[int64]).GetValue()
	if !ok {
		return nil, v.runtimeError("else chunk index must be deterministic int")
	}
	_ /*thenChunkIndex*/, ok = thenChunkIdxState.ValueOutcome.(*core.Outcomes[int64]).GetValue()
	if !ok {
		return nil, v.runtimeError("then chunk index must be deterministic int")
	}

	// --- Get Condition Outcome & Predicate ---
	boolOutcome, ok := conditionState.ValueOutcome.(*core.Outcomes[bool])
	if !ok {
		return nil, v.runtimeError("if condition must evaluate to *core.Outcomes[bool], got %T", conditionState.ValueOutcome)
	}
	boolPredicate := func(v bool) bool { return v } // Simple predicate for bool outcomes

	// --- Split Current Context (State BEFORE condition was evaluated) ---
	// PROBLEM: The state *before* the condition is not easily available on the stack.
	// The standard sequential model combines state.
	//
	// REVISED APPROACH for IF Handler:
	// 1. The stack *before* OpCallFunc(HandlerIf) contains:
	//    [..., PreConditionState, ConditionResultState, ThenChunkIdxState, ElseChunkIdxState] ?? No, compiler emits condition then indices.
	//    Stack: [..., ConditionState, ThenChunkIdxState, ElseChunkIdxState]
	//
	// 2. We need the state *before* the condition was evaluated. This state should be
	//    available deeper in the stack (e.g., peek(3) if stack was [PreCond, Cond, ThenIdx, ElseIdx]).
	//    Let's assume `peek(0)` holds ElseChunkIdx, `peek(1)` ThenChunkIdx, `peek(2)` ConditionState.
	//    Where is the PreConditionState? The implicit sequence handling consumes it.
	//
	// ALTERNATIVE VM DESIGN for IF/DISTRIBUTE:
	// - Handlers *don't* pop the triggering input (like ConditionState).
	// - They operate on the stack, potentially pushing intermediate results or contexts.
	// - They return control to the VM loop, possibly modifying the IP to jump to sub-chunks.
	// This seems overly complex compared to recursive evaluation.
	//
	// Let's stick to recursive execution, but the Go handler needs the context.
	// HOW? Maybe `OpCallFunc` should implicitly pass the state *below* its direct operands?
	// Or, the handler *pops* operands, *peeks* at the context below, executes, *pushes* result.

	// --- Let's try peeking ---
	// Stack: [..., ContextState, ConditionState, ThenChunkIdxState, ElseChunkIdxState] <-- Top
	// Pop indices first
	// elseChunkIdxState, err = v.pop(); if err != nil { return nil, err }
	// thenChunkIdxState, err = v.pop(); if err != nil { return nil, err }
	// conditionState, err = v.pop(); if err != nil { return nil, err }
	// Now peek at the context state below
	contextState, err := v.peek(0) // Peek at what's now the top
	if err != nil {
		return nil, v.runtimeError("stack underflow retrieving context for 'if'")
	}

	// --- Split Value Track (Condition's Bool Outcome) ---
	/*thenValOutcomeAny, elseValOutcomeAny := */
	boolOutcome.Split(boolPredicate)

	// --- Split Latency Track (Context's Latency Outcome) ---
	// **Requires core.SplitLatenciesBasedOnValue or equivalent**
	contextLatOutcome, _ := contextState.LatencyOutcome.(*core.Outcomes[core.Duration])
	if contextLatOutcome == nil {
		contextLatOutcome = zeroLatencyOutcome()
	}

	// *** PLACEHOLDER for core function ***
	_, _ /*thenLatOutcome, elseLatOutcome, */, splitErr := SplitLatenciesBasedOnValue(contextLatOutcome, boolOutcome, boolPredicate)
	if splitErr != nil {
		return nil, v.runtimeError("error splitting context latency for if: %v", splitErr)
	}
	// *** END PLACEHOLDER ***

	// Create split contexts (Value comes from split Condition, Latency from split Context)
	// This seems wrong. The latency *of the condition evaluation* needs to be added too.
	// TODO: Revisit split logic - needs condition latency + context latency combined first?

	// --- Execute Branches Recursively (Simplified for now) ---
	// This needs to execute sub-chunks. Pass split context?
	// For now, placeholder:
	fmt.Println("Warning: vm_handle_if execution logic is a placeholder.")
	thenResultState := createIdentityState() // Placeholder
	elseResultState := createIdentityState() // Placeholder

	// Combine results (placeholder)
	// Need proper Append logic for VarState or its components
	combinedLat := (&core.Outcomes[core.Duration]{}).Append(thenResultState.LatencyOutcome.(*core.Outcomes[core.Duration]), elseResultState.LatencyOutcome.(*core.Outcomes[core.Duration]))
	combinedVal := (&core.Outcomes[bool]{}).Append(thenResultState.ValueOutcome.(*core.Outcomes[bool]), elseResultState.ValueOutcome.(*core.Outcomes[bool])) // Assume bool for now

	// --- Crucial: Replace the context state with the final combined result ---
	_, err = v.pop() // Pop the original context state
	if err != nil {
		return nil, v.runtimeError("stack error popping context state after 'if'")
	}

	finalState := &VarState{ValueOutcome: combinedVal, LatencyOutcome: combinedLat}

	// Apply reduction before pushing
	finalState, err = v.reduceStateIfNeeded(finalState)
	if err != nil {
		return nil, err
	}

	if err := v.push(finalState); err != nil {
		return nil, err
	}

	// For now:
	return nil, v.runtimeError("if handler not implemented")
}

func (v *VM) vm_handle_distribute(vm *VM, args []*VarState) (*VarState, error) {
	// Complex logic delegated to eval_distribute.go (needs adapting)
	// For now:
	return nil, v.runtimeError("distribute handler not implemented")
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

	// Handle nil inputs gracefully
	if valueDistribution == nil {
		// Cannot apply predicate if value dist is nil. Return empty splits.
		// Or should we return the original latDistribution in both? Ambiguous. Empty seems safer.
		return thenLat, elseLat, nil
	}
	if latDistribution == nil {
		// No latency to split, return empty splits.
		return thenLat, elseLat, nil
	}

	// Critical Assumption: Buckets are synchronized by index.
	if len(latDistribution.Buckets) != len(valueDistribution.Buckets) {
		err = fmt.Errorf("cannot split latency: value (len %d) and latency (len %d) distributions have different bucket counts",
			len(valueDistribution.Buckets), len(latDistribution.Buckets))
		// Return empty outcomes on error
		return &core.Outcomes[core.Duration]{}, &core.Outcomes[core.Duration]{}, err
	}

	// Iterate and distribute latency buckets based on value predicate
	for i := range valueDistribution.Buckets {
		valueBucket := valueDistribution.Buckets[i]
		latencyBucket := latDistribution.Buckets[i] // Corresponding latency bucket

		// Optional: Weight check (can be sensitive to float precision)
		// if !core.ApproxEq(valueBucket.Weight, latencyBucket.Weight, 1e-9) {
		// 	err = fmt.Errorf("bucket weight mismatch at index %d during latency split (V:%.9f L:%.9f)", i, valueBucket.Weight, latencyBucket.Weight)
		// 	return &core.Outcomes[core.Duration]{}, &core.Outcomes[core.Duration]{}, err
		// }

		if valuePredicate(valueBucket.Value) {
			thenLat.Buckets = append(thenLat.Buckets, latencyBucket)
		} else {
			elseLat.Buckets = append(elseLat.Buckets, latencyBucket)
		}
	}

	// Optional: Renormalize weights in split outcomes? Only needed if input wasn't normalized.
	// Assume input distributions are normalized for now.

	return thenLat, elseLat, nil
}

// --- Also need VarState.Append method or helper ---
// Helper to append VarStates (simplistic version)
func appendVarStates(stateA, stateB *VarState) (*VarState, error) {
	if stateA == nil {
		return stateB, nil
	}
	if stateB == nil {
		return stateA, nil
	}

	// Combine Latencies
	latA := stateA.SafeLatencyOutcome().(*core.Outcomes[core.Duration])
	latB := stateB.SafeLatencyOutcome().(*core.Outcomes[core.Duration])
	if latA == nil {
		latA = zeroLatencyOutcome()
	}
	if latB == nil {
		latB = zeroLatencyOutcome()
	}
	combinedLat := latA.Copy().Append(latB) // Append creates new slice internally

	// Combine Values (requires type matching)
	var combinedVal any
	valA := stateA.SafeValueOutcome()
	valB := stateB.SafeValueOutcome()

	if valA == nil {
		combinedVal = valB
		goto CombineDone
	}
	if valB == nil {
		combinedVal = valA
		goto CombineDone
	}

	// TODO: Use reflection or type switch to call appropriate Append
	switch vA := valA.(type) {
	case *core.Outcomes[bool]:
		if vB, ok := valB.(*core.Outcomes[bool]); ok {
			combinedVal = vA.Copy().Append(vB)
		} else {
			goto TypeMismatch
		}
	case *core.Outcomes[int64]:
		if vB, ok := valB.(*core.Outcomes[int64]); ok {
			combinedVal = vA.Copy().Append(vB)
		} else {
			goto TypeMismatch
		}
	// Add other supported discrete types (float64, string, AccessResult etc.)
	default:
		return nil, fmt.Errorf("unsupported value type %T for append", valA)
	}
	goto CombineDone

TypeMismatch:
	return nil, fmt.Errorf("cannot append mismatched value types: %T and %T", valA, valB)

CombineDone:
	return &VarState{ValueOutcome: combinedVal, LatencyOutcome: combinedLat}, nil
}
