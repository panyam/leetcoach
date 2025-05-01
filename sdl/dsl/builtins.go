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
		return nil, fmt.Errorf("add failed during combination: %w", err)
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
	// Let's assume operands aren't passed via args, but read from instruction stream if needed.
	// Complex logic delegated to eval_ifstmt.go (needs adapting to bytecode context)
	// For now:
	return nil, v.runtimeError("if handler not implemented")
}

func (v *VM) vm_handle_distribute(vm *VM, args []*VarState) (*VarState, error) {
	// Complex logic delegated to eval_distribute.go (needs adapting)
	// For now:
	return nil, v.runtimeError("distribute handler not implemented")
}
