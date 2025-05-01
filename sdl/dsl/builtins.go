package dsl

import (
	"fmt"

	"github.com/panyam/leetcoach/sdl/core"
)

func (v *VM) registerDefaultInternals() {
	v.RegisterInternalFunc("GetAccess", v.vm_handle_get_access)
	v.RegisterInternalFunc("GetBool", v.vm_handle_get_bool)
}

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
	// v.RegisterInternalHandler(HandlerDurationMS, v.vm_handle_durationms)
	v.RegisterInternalHandler(HandlerGetBool, v.vm_handle_get_bool)
	v.RegisterInternalHandler(HandlerGetAccess, v.vm_handle_get_access)
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

func (v *VM) vm_handle_get_access(vm *VM, args []*VarState) (*VarState, error) {
	// Args: successProb (float Outcome), successLat (Duration Outcome), failLat (Duration Outcome)
	if len(args) != 3 {
		return nil, fmt.Errorf("GetAccess needs 3 args")
	}
	probState := args[0]
	latSState := args[1]
	latFState := args[2]
	if probState == nil || latSState == nil || latFState == nil {
		return nil, fmt.Errorf("GetBool args must be VarState")
	}

	probF, ok := probState.ValueOutcome.(*core.Outcomes[float64]).GetValue()
	if !ok {
		return nil, fmt.Errorf("GetAccess prob must be deterministic")
	}
	latS, ok := latSState.ValueOutcome.(*core.Outcomes[core.Duration]).GetValue()
	if !ok {
		return nil, fmt.Errorf("GetAccess successLat must be deterministic")
	}
	latF, ok := latFState.ValueOutcome.(*core.Outcomes[core.Duration]).GetValue()
	if !ok {
		return nil, fmt.Errorf("GetAccess failLat must be deterministic")
	}

	valOutcome := &core.Outcomes[core.AccessResult]{}
	latOutcome := &core.Outcomes[core.Duration]{}
	if probF > 1e-9 {
		ar := core.AccessResult{Success: true, Latency: latS}
		valOutcome.Add(probF, ar)
		latOutcome.Add(probF, latS)
	}
	if (1.0 - probF) > 1e-9 {
		ar := core.AccessResult{Success: false, Latency: latF}
		valOutcome.Add(1.0-probF, ar)
		latOutcome.Add(1.0-probF, latF)
	}
	return &VarState{ValueOutcome: valOutcome, LatencyOutcome: latOutcome}, nil
}

func (v *VM) vm_handle_get_bool(vm *VM, args []*VarState) (*VarState, error) {
	// Args: successProb (float Outcome), successLat (Duration Outcome), failLat (Duration Outcome)
	if len(args) != 3 {
		return nil, fmt.Errorf("GetBool needs 3 args")
	}
	probState := args[0]
	latSState := args[1]
	latFState := args[2]
	if probState == nil || latSState == nil || latFState == nil {
		return nil, fmt.Errorf("GetBool args must be VarState")
	}

	// Get deterministic values from argument outcomes
	probF, ok := probState.ValueOutcome.(*core.Outcomes[float64]).GetValue()
	if !ok {
		return nil, fmt.Errorf("GetBool prob must be deterministic")
	}
	latS, ok := latSState.ValueOutcome.(*core.Outcomes[core.Duration]).GetValue()
	if !ok {
		return nil, fmt.Errorf("GetBool successLat must be deterministic")
	}
	latF, ok := latFState.ValueOutcome.(*core.Outcomes[core.Duration]).GetValue()
	if !ok {
		return nil, fmt.Errorf("GetBool failLat must be deterministic")
	}

	valOutcome := &core.Outcomes[bool]{}
	latOutcome := &core.Outcomes[core.Duration]{}
	if probF > 1e-9 {
		valOutcome.Add(probF, true)
		latOutcome.Add(probF, latS)
	}
	if (1.0 - probF) > 1e-9 {
		valOutcome.Add(1.0-probF, false)
		latOutcome.Add(1.0-probF, latF)
	}
	return &VarState{ValueOutcome: valOutcome, LatencyOutcome: latOutcome}, nil
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
	// Stack state upon entry (relative to top):
	// - ElseChunkIndex Constant State (*VarState{Outcomes[int64]})
	// - ThenChunkIndex Constant State (*VarState{Outcomes[int64]})
	// - Condition Result State (*VarState{Outcomes[bool]})
	// - Context State (*VarState) // State *before* condition was evaluated

	// 1. Pop Operands and Condition
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

	// 2. Peek at the Context State (state before condition evaluation)
	//    This holds the latency accumulated *up to* the 'if' statement.
	contextState, err := v.peek(0) // Context is now top
	if err != nil {
		return nil, v.runtimeError("stack underflow retrieving context for 'if'")
	}

	// 3. Extract Chunk Indices
	elseChunkIndexVal, ok := elseChunkIdxState.ValueOutcome.(*core.Outcomes[int64]).GetValue()
	if !ok {
		return nil, v.runtimeError("else chunk index constant error")
	}
	thenChunkIndexVal, ok := thenChunkIdxState.ValueOutcome.(*core.Outcomes[int64]).GetValue()
	if !ok {
		return nil, v.runtimeError("then chunk index constant error")
	}

	// 4. Extract Condition Bool Outcome
	boolOutcome, ok := conditionState.ValueOutcome.(*core.Outcomes[bool])
	if !ok {
		return nil, v.runtimeError("if condition must evaluate to *core.Outcomes[bool], got %T", conditionState.ValueOutcome)
	}
	boolPredicate := func(v bool) bool { return v } // Standard predicate for bool outcomes

	// --- Split the Context Latency based on the Condition Value ---
	contextLatOutcome, _ := contextState.LatencyOutcome.(*core.Outcomes[core.Duration])
	if contextLatOutcome == nil {
		contextLatOutcome = zeroLatencyOutcome()
	}

	thenContextLat, elseContextLat, splitErr := SplitLatenciesBasedOnValue(contextLatOutcome, boolOutcome, boolPredicate)
	if splitErr != nil {
		return nil, v.runtimeError("error splitting context latency for if: %v", splitErr)
	}

	// --- Split the Condition Latency ---
	// The latency incurred *during* the condition evaluation also needs splitting.
	conditionLatOutcome, _ := conditionState.LatencyOutcome.(*core.Outcomes[core.Duration])
	if conditionLatOutcome == nil {
		conditionLatOutcome = zeroLatencyOutcome()
	}

	thenCondLat, elseCondLat, splitErr := SplitLatenciesBasedOnValue(conditionLatOutcome, boolOutcome, boolPredicate)
	if splitErr != nil {
		return nil, v.runtimeError("error splitting condition latency for if: %v", splitErr)
	}

	// 5. Prepare Initial States for Branch Execution
	// The initial state for a branch includes the split context latency *plus* the split condition latency.
	// The value outcome for the initial state is determined by the split condition outcome.

	// Combine split context latency and split condition latency for 'then' branch start
	thenStartLatState, err := v.combineOutcomesAndReduce(&VarState{LatencyOutcome: thenContextLat}, &VarState{LatencyOutcome: thenCondLat})
	if err != nil {
		return nil, v.runtimeError("failed combining then branch start latency: %v", err)
	}
	// Split value outcome for 'then' branch start
	thenValOutcome, _ := boolOutcome.Split(boolPredicate) // Split value outcome
	thenStartState := &VarState{
		ValueOutcome:   thenValOutcome,
		LatencyOutcome: thenStartLatState.LatencyOutcome,
	}

	// Combine split context latency and split condition latency for 'else' branch start
	elseStartLatState, err := v.combineOutcomesAndReduce(&VarState{LatencyOutcome: elseContextLat}, &VarState{LatencyOutcome: elseCondLat})
	if err != nil {
		return nil, v.runtimeError("failed combining else branch start latency: %v", err)
	}
	// Split value outcome for 'else' branch start
	_, elseValOutcome := boolOutcome.Split(boolPredicate)
	elseStartState := &VarState{
		ValueOutcome:   elseValOutcome,
		LatencyOutcome: elseStartLatState.LatencyOutcome,
	}

	var finalThenResultState *VarState = nil // Initialize result states
	var finalElseResultState *VarState = nil

	// 6. Execute Then Branch (if it has weight and exists)
	thenChunk := v.constants[thenChunkIndexVal].(*Chunk)                   // Assumes constant is correct type
	if thenChunk != nil && outcomeHasWeight(thenStartState.ValueOutcome) { // Check split value weight
		// Execute sub-chunk recursively. Requires ability to execute specific chunk with context.
		// Create a temporary VM or context for sub-execution?
		// Or: Modify vm state, execute, restore state? Risky.
		// --- Simplest: Execute sub-chunk, starting stack contains ONLY thenStartState ---
		vm.stack[vm.sp] = thenStartState                           // Replace context state with then start state on stack
		branchResultState, branchErr := vm.ExecuteChunk(thenChunk) // Execute compiled 'then' block
		if branchErr != nil {
			// TODO: Handle potential ReturnValue signal from branch
			return nil, v.runtimeError("error executing 'then' branch chunk: %v", branchErr)
		}
		finalThenResultState = branchResultState // Store the result
	} else if outcomeHasWeight(thenStartState.ValueOutcome) { // Path has weight but no block
		finalThenResultState = thenStartState // Result is just the starting state for this path
	}

	// 7. Execute Else Branch (if it has weight and exists)
	elseChunk := (*Chunk)(nil)
	if elseChunkIndexVal >= 0 {
		elseChunk = v.constants[elseChunkIndexVal].(*Chunk)
	}
	if elseChunk != nil && outcomeHasWeight(elseStartState.ValueOutcome) {
		vm.stack[vm.sp] = elseStartState // Replace context state with else start state
		branchResultState, branchErr := vm.ExecuteChunk(elseChunk)
		if branchErr != nil {
			// TODO: Handle ReturnValue signal
			return nil, v.runtimeError("error executing 'else' branch chunk: %v", branchErr)
		}
		finalElseResultState = branchResultState
	} else if outcomeHasWeight(elseStartState.ValueOutcome) { // Path has weight but no block
		finalElseResultState = elseStartState // Result is starting state
	}

	// 8. Combine Branch Results using appendVarStates
	combinedState, combineErr := appendVarStates(finalThenResultState, finalElseResultState)
	if combineErr != nil {
		return nil, v.runtimeError("failed to combine 'if' branch results: %v", combineErr)
	}

	// 9. Replace context state on stack with the final combined result
	_, err = v.pop() // Pop the original context state (or the state left by the last branch)
	if err != nil {
		return nil, v.runtimeError("stack error popping context/branch state after 'if'")
	}

	// 10. Apply Reduction to combined state
	finalCombinedState, reduceErr := v.reduceStateIfNeeded(combinedState)
	if reduceErr != nil {
		return nil, v.runtimeError("error reducing combined IfStmt result: %v", reduceErr)
	}

	// 11. Push final result onto stack
	if err := v.push(finalCombinedState); err != nil {
		return nil, err
	}

	return nil, nil // Success, result is on stack}
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
	reducer := func(a, b core.Duration) core.Duration { return a + b } // Standard duration reducer
	thenLat = &core.Outcomes[core.Duration]{}                          // Initialize empty
	elseLat = &core.Outcomes[core.Duration]{}                          // Initialize empty

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
		// Check if latency distribution has a corresponding bucket (paranoid check)
		if i >= len(latDistribution.Buckets) {
			// This should be caught by the initial length check, but defensively...
			err = fmt.Errorf("internal error: latency distribution missing bucket at index %d during split", i)
			return &core.Outcomes[core.Duration]{And: reducer}, &core.Outcomes[core.Duration]{And: reducer}, err
		}
		latencyBucket := latDistribution.Buckets[i] // Corresponding latency bucket

		// Apply the predicate to the value bucket
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
	// Handle nil states
	if stateA == nil && stateB == nil {
		return createIdentityState(), nil
	} // Return identity if both paths yield nothing
	if stateA == nil {
		return stateB, nil
	}
	if stateB == nil {
		return stateA, nil
	}

	// --- Combine Latencies using Append ---
	// Latency is always *core.Outcomes[core.Duration] or nil
	latA := stateA.SafeLatencyOutcome().(*core.Outcomes[core.Duration]) // Type assertion safe due to VarState invariants
	latB := stateB.SafeLatencyOutcome().(*core.Outcomes[core.Duration])
	var combinedLat *core.Outcomes[core.Duration]
	if latA == nil && latB == nil {
		combinedLat = zeroLatencyOutcome()
	} else if latA == nil {
		combinedLat = latB.Copy() // Copy B if A is nil
	} else {
		// Use A as receiver, Append handles nil B correctly
		combinedLat = latA.Copy().Append(latB)
	}

	// --- Combine Values using Append ---
	valA := stateA.SafeValueOutcome()
	valB := stateB.SafeValueOutcome()
	var combinedVal any // Resulting combined value outcome

	// Determine the type and perform append
	switch a := valA.(type) {
	case nil:
		combinedVal = valB // If A is nil, result is B (which could also be nil)
	case *core.Outcomes[bool]:
		if b, ok := valB.(*core.Outcomes[bool]); ok || valB == nil {
			combinedVal = a.Copy().Append(b) // Append handles nil b
		} else {
			goto TypeMismatchError
		}
	case *core.Outcomes[int64]:
		if b, ok := valB.(*core.Outcomes[int64]); ok || valB == nil {
			combinedVal = a.Copy().Append(b)
		} else {
			goto TypeMismatchError
		}
	// case *core.Outcomes[float64]:
	case *core.Outcomes[core.Duration]: // Should not happen as ValueOutcome, but handle defensively
		if b, ok := valB.(*core.Outcomes[core.Duration]); ok || valB == nil {
			combinedVal = a.Copy().Append(b)
		} else {
			goto TypeMismatchError
		}
	case *core.Outcomes[core.AccessResult]:
		if b, ok := valB.(*core.Outcomes[core.AccessResult]); ok || valB == nil {
			combinedVal = a.Copy().Append(b)
		} else {
			goto TypeMismatchError
		}
	// Add other supported *core.Outcomes[T] types here...
	default:
		// This case occurs if valA is non-nil but not a recognized *core.Outcomes type
		// OR if valB is non-nil and doesn't match valA's type.
		// We already checked for valB mismatch above (goto TypeMismatchError),
		// so this mainly catches unsupported types in valA.
		return nil, fmt.Errorf("unsupported value type %T encountered during appendVarStates", valA)
	}

	return &VarState{ValueOutcome: combinedVal, LatencyOutcome: combinedLat}, nil

TypeMismatchError:
	return nil, fmt.Errorf("cannot append mismatched value types: %T and %T", valA, valB)
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
