// sdl/dsl/eval_stmt.go
package dsl

import (
	"errors"
	"fmt"
)

// For identity outcome

// --- Statement Evaluators ---
// evalAssignmentStmt evaluates the RHS expression, stores the resulting
// Outcome object in the environment under the variable name, and leaves the
// result on the stack (for potential use in implicit AND).
func (v *VM) evalAssignmentStmt(stmt *AssignmentStmt) (*VarState, error) {
	resultState, err := v.Eval(stmt.Value) // Eval returns the state
	if err != nil {
		var retVal *ReturnValue
		if errors.As(err, &retVal) {
			// Assign before propagating return
			stateToStore := retVal.Value
			if stateToStore != nil {
				v.env.Set(stmt.Variable.Name, stateToStore.ValueOutcome, stateToStore.LatencyOutcome)
				// Don't push here if returning
			}
			return stateToStore, err // Return state + signal
		}
		return nil, fmt.Errorf("error evaluating assignment value for '%s': %w", stmt.Variable.Name, err)
	}
	if resultState == nil {
		return nil, fmt.Errorf("internal error: Eval returned nil state without error for assignment '%s'", stmt.Variable.Name)
	}

	v.env.Set(stmt.Variable.Name, resultState.ValueOutcome, resultState.LatencyOutcome)
	v.push(resultState)     // Push for sequence composition
	return resultState, nil // Return the state as well
}

// evalExprStmt evaluates the expression, pushes its result state onto the stack,
// and returns the state. Handles ReturnValue signal.
func (v *VM) evalExprStmt(stmt *ExprStmt) (*VarState, error) {
	resultState, err := v.Eval(stmt.Expression)
	if err != nil {
		var retVal *ReturnValue
		if errors.As(err, &retVal) {
			// Don't push if returning
			return retVal.Value, err // Return state + signal
		}
		return nil, fmt.Errorf("error evaluating expression statement: %w", err)
	}
	if resultState == nil {
		return nil, fmt.Errorf("internal error: Eval returned nil state without error for expression statement")
	}

	v.push(resultState)     // Push for sequence composition
	return resultState, nil // Return the state
}

// evalReturnStmt evaluates the return expression, pushes the result state onto the stack,
// and returns the state along with the ReturnValue signal wrapper error.
func (v *VM) evalReturnStmt(stmt *ReturnStmt) (*VarState, error) {
	resultState, err := v.Eval(stmt.ReturnValue)
	if err != nil {
		var retVal *ReturnValue
		if errors.As(err, &retVal) {
			// Return nested state/signal directly
			return retVal.Value, err
		}
		return nil, fmt.Errorf("error evaluating return value: %w", err)
	}
	if resultState == nil {
		return nil, fmt.Errorf("internal error: Eval returned nil state without error for return value")
	}

	v.push(resultState) // Push the return value state
	returnSignal := &ReturnValue{Value: resultState, Err: ErrReturnSignal}
	return resultState, returnSignal // Return state AND signal
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

// evalBlockStmt executes a sequence of statements within a given environment.
// It handles the implicit sequential 'And' composition using the dual tracks.
// Returns the VarState outcome of the *last* effectively executed statement/expression.
// If a ReturnStmt is encountered, it returns the returned VarState and the ReturnValue error.
func (v *VM) evalBlockStmt(block *BlockStmt, env *Environment, initialContext *VarState) (*VarState, error) {
	// Initialize stack for this block's execution? Or assume caller manages stack?
	// Let's assume the block operates on the current stack. Caller ensures clean state if needed.

	// Push initial context onto the stack if provided (acts as first operand)
	stackStartLen := len(v.stack) // Remember stack depth at start

	// Defer stack cleanup to ensure it happens even on errors/returns within the block
	defer func() {
		// Ensure stack is restored to original depth unless a value was intentionally left
		// This cleanup might be too aggressive if nested calls modify stack in expected ways.
		// Let's refine: Clean up only items pushed *by this block's loop* if an error occurs before the end.
		// For now, the logic relies on correct push/pop within the loop and final pop.
		// If an error other than ReturnValue occurs, the stack might be left in an inconsistent state.
		// More robust error recovery could be added later.
		// fmt.Printf("DEBUG evalBlockStmt End: Final Stack Len=%d (Started at %d)\n", len(v.stack), stackStartLen) // Debug
	}()

	if initialContext != nil {
		v.push(initialContext)
	} else {
		// Push identity state if no context provided?
		v.push(createIdentityState())
	}

	// fmt.Printf("DEBUG evalBlockStmt Start: Initial Stack Len=%d, Top=%v\n", len(v.stack), v.peek()) // Debug

	for i, stmt := range block.Statements {
		// --- Evaluate the statement ---
		// eval*Stmt functions return error only (incl. ReturnValue signal)
		// and push their result state onto the stack if applicable to the sequence.
		evalErr := error(nil)
		// stackLenBeforeStmt := len(v.stack) // Track stack changes per statement

		switch s := stmt.(type) {
		// Call specific eval*Stmt funcs (which now return only error)
		case *AssignmentStmt:
			_, evalErr = v.evalAssignmentStmt(s)
		case *ExprStmt:
			_, evalErr = v.evalExprStmt(s)
		case *ReturnStmt:
			_, evalErr = v.evalReturnStmt(s)
		case *IfStmt:
			_, evalErr = v.evalIfStmt(s) // If also needs update
		// case *DelayStmt: // Example if delay is a statement
		//  evalErr = v.evalDelayStmt(s) // This would likely pop, combine latency, push back
		default:
			// Clean up stack before returning error
			v.stack = v.stack[:stackStartLen]
			return nil, fmt.Errorf("unsupported statement type in block: %T", s)
		}

		// --- Error Handling ---
		if evalErr != nil {
			var retVal *ReturnValue
			if errors.As(evalErr, &retVal) {
				// A return occurred. The return value IS ALREADY ON TOP of the stack
				// (pushed by evalReturnStmt or propagated).
				// We need to remove everything else pushed by this block execution below it.
				stackLenAfterReturnPush := len(v.stack)

				if stackLenAfterReturnPush > stackStartLen+1 {
					// Remove intermediate items below the return value
					returnValue := v.stack[stackLenAfterReturnPush-1] // Keep the top one
					v.stack = v.stack[:stackStartLen]                 // Reset to start
					v.push(returnValue)                               // Push the actual return value back
				} else if stackLenAfterReturnPush <= stackStartLen {
					// Should not happen - return must push its value
					v.stack = v.stack[:stackStartLen] // Attempt cleanup
					return nil, fmt.Errorf("internal error: stack state inconsistent after return signal (%d <= %d)", stackLenAfterReturnPush, stackStartLen)
				}
				// Propagate the signal error, the final state is retVal.Value
				return retVal.Value, evalErr
			}
			// Otherwise, it's a real error. Clean up stack from this block.
			v.stack = v.stack[:stackStartLen]
			return nil, fmt.Errorf("error executing statement %d (%T): %w", i, stmt, evalErr)
		}
		// --- Implicit Sequential Composition via Stack ---
		// Check if the statement actually pushed a result expected for the sequence.
		// For now, assume AssignmentStmt, ExprStmt, IfStmt push one result state.
		// If stack depth increased, and we now have >= 2 items *relative to the block start*, combine them.
		if len(v.stack) >= stackStartLen+2 {
			// Pop the top two states (result of current stmt B, result of previous sequence A)
			stateB, _ := v.pop()
			stateA, _ := v.pop()

			// Combine A then B
			combinedState, combineErr := v.combineOutcomesAndReduce(stateA, stateB)
			if combineErr != nil {
				v.stack = v.stack[:stackStartLen] // Clean up stack
				return nil, fmt.Errorf("implicit AND failed after statement %d (%T): %w", i, stmt, combineErr)
			}
			v.push(combinedState) // Push combined result back
		}
		// fmt.Printf("DEBUG evalBlockStmt Iter %d (%T): Stack Len=%d\n", i, stmt, len(v.stack)) // Debug

	} // End loop over statements
	// Block finished normally. The final result is the single item left on the stack
	// relative to where this block started.
	finalStackDepth := len(v.stack)
	if finalStackDepth != stackStartLen+1 {
		v.stack = v.stack[:stackStartLen] // Attempt cleanup
		return nil, fmt.Errorf("internal error: block ended with unexpected stack depth %d (expected %d). Stack: %s", finalStackDepth, stackStartLen+1, v.stackString())
	}

	finalState, _ := v.pop() // Pop the final result for returning
	return finalState, nil   // Return final state, nil error
}

/*


// evalIfStmt handles conditional execution.
// It evaluates the condition, splits the relevant outcome distribution,
// evaluates the 'then' and 'else' branches with the split distributions,
// and appends the results.
func (v *VM) evalIfStmt(stmt *IfStmt) error {

	// 1. Evaluate the Condition Expression
	_, err := v.Eval(stmt.Condition)
	if err != nil {
		return fmt.Errorf("error evaluating if condition: %w", err)
	}
	conditionOutcomeRaw, err := v.pop() // Pop the result of the condition expression
	if err != nil {
		return fmt.Errorf("stack error getting if condition result: %w", err)
	}

	// 2. Determine Predicate and Split the Distribution
	// The condition itself might be complex (e.g., `a.Success`). We need the *outcome*
	// that the condition depends on (e.g., the outcome stored in `a`).
	// For now, assume the condition evaluation *directly* yields the outcome to be split.
	// Example: If `Condition` is `myVar.Success`, `Eval(Condition)` should push `myVar`'s outcome.
	// This requires `evalMemberAccess` to handle `.Success` specially or assumes
	// the DSL structure enforces `if someOutcome { ... }` where someOutcome is bool.
	// --- Let's refine: Assume Eval(stmt.Condition) pushes the relevant OUTCOME to split ---
	// This implies `evalMemberAccess` for `.Success` should push the receiver's outcome.
	// Now that evalMemberAccess is implemented, this assumption should hold.
	// For *this phase*, let's assume `conditionOutcomeRaw` IS the outcome distribution to split.

	splitPredicate, err := selectSplitPredicate(stmt.Condition, conditionOutcomeRaw)
	if err != nil {
		return err // Unsupported condition type
	}

	// Perform the split using type assertion (needs refinement for more types)
	var thenInputOutcome, elseInputOutcome any // Use any for split results

	switch condOutcome := conditionOutcomeRaw.(type) {
	case *core.Outcomes[core.AccessResult]:
		// Need to wrap the generic predicate
		specificPredicate := func(v core.AccessResult) bool {
			return splitPredicate(v) // Call the generic predicate
		}
		thenInputOutcome, elseInputOutcome = condOutcome.Split(specificPredicate)
	case *core.Outcomes[bool]:
		// Wrap the generic predicate
		specificPredicate := func(v bool) bool { // Expects bool
			return splitPredicate(v) // Call generic func(v any) bool
		}
		thenInputOutcome, elseInputOutcome = condOutcome.Split(specificPredicate)
	// Add cases for other splittable types (e.g., RangedResult if needed)
	default:
		return fmt.Errorf("%w: cannot split outcome of type %T based on condition %q", ErrConditionSplitFailed, conditionOutcomeRaw, stmt.Condition.String())
	}

	// 3. Evaluate Then Branch
	// Need to execute the Then block, providing `thenInputOutcome` as its starting context.
	// How to pass this context? The implicit 'And' in evalBlockStmt needs modification.
	// --- Simplification for now: ---
	// Assume the block implicitly starts with the split outcome. Push it onto the stack before calling evalBlockStmt.
	// The block's first operation will implicitly 'And' with it.
	var thenBranchResultOutcome any // This will hold the result *returned by* evalBlockStmt
	var thenErr error
	// Check if the split result is valid before proceeding
	// --- Use OutcomeContainer interface to check Len ---
	thenOutcomeContainer, thenIsContainer := thenInputOutcome.(core.OutcomeContainer)
	if !thenIsContainer && thenInputOutcome != nil {
		return fmt.Errorf("internal error: 'then' split result (%T) does not implement OutcomeContainer", thenInputOutcome)
	}
	if thenIsContainer && thenOutcomeContainer != nil && thenOutcomeContainer.Len() > 0 { // Check validity and length via interface
		// --- Pass context as argument, DON'T PUSH ---
		thenBranchResultOutcome, thenErr = v.evalBlockStmt(stmt.Then, v.env, thenInputOutcome) // Pass context
		if thenErr != nil && !errors.Is(thenErr, ErrReturnSignal) {                            // Handle errors (ignore Return signal here)
			return fmt.Errorf("error in 'then' branch: %w", thenErr)
		}
		if errors.Is(thenErr, ErrReturnSignal) {
			// If the 'then' branch returned, we need to propagate that
			// The actual return value is in thenBranchResultOutcome
			return &ReturnValue{Value: thenBranchResultOutcome} // Re-wrap
		}
		// Result is returned directly by evalBlockStmt, pop it if successful
		// No, evalBlockStmt doesn't leave result on stack, it returns it.
	} else {
		// If the 'then' path has zero probability, result is nil (or an empty outcome?)
		thenBranchResultOutcome = nil // Or create empty outcome of expected type?
	}

	// 4. Evaluate Else Branch (if it exists)
	var elseBranchResultOutcome any
	elseOutcomeContainer, elseIsContainer := elseInputOutcome.(core.OutcomeContainer)
	if !elseIsContainer && elseInputOutcome != nil {
		return fmt.Errorf("internal error: 'else' split result (%T) does not implement OutcomeContainer", elseInputOutcome)
	}
	var elseErr error
	if stmt.Else != nil && elseIsContainer && elseOutcomeContainer != nil && elseOutcomeContainer.Len() > 0 { // Check validity and length
		// --- Pass context as argument, DON'T PUSH ---
		elseBranchResultOutcome, elseErr = v.evalBlockStmt(stmt.Else, v.env, elseInputOutcome)
		// NOTE: evalBlockStmt should NOT leave its result on the stack. It returns it.
		if elseErr != nil && !errors.Is(elseErr, ErrReturnSignal) {
			return fmt.Errorf("error in 'else' branch: %w", elseErr)
		}
		if errors.Is(elseErr, ErrReturnSignal) {
			// Propagate return from 'else'
			return &ReturnValue{Value: elseBranchResultOutcome}
		}
		// Result is returned by evalBlockStmt
	} else {
		elseBranchResultOutcome = nil
	}

	// 5. Combine Results using Append
	var finalCombinedOutcome any

	// --- DEBUG ---
	fmt.Printf("DEBUG evalIfStmt Combine: ThenResult=%T, ElseResult=%T\n", thenBranchResultOutcome, elseBranchResultOutcome)

	// Use type assertions and core.Append
	switch tb := thenBranchResultOutcome.(type) {
	case *core.Outcomes[core.AccessResult]:
		eb, ok := elseBranchResultOutcome.(*core.Outcomes[core.AccessResult])
		if elseBranchResultOutcome == nil || (ok && eb != nil) { // Allow nil else or matching type
			// Need to handle case where tb itself might be nil if path had 0 prob
			if tb == nil {
				tb = &core.Outcomes[core.AccessResult]{And: core.AndAccessResults}
			} // Empty if no then path
			finalCombinedOutcome = tb.Append(eb) // Append handles nil input
		} else {
			return fmt.Errorf("%w: cannot combine then (%T) and else (%T)", ErrTypeMismatch, tb, elseBranchResultOutcome)
		}
	case *core.Outcomes[core.Duration]:
		eb, ok := elseBranchResultOutcome.(*core.Outcomes[core.Duration])
		if elseBranchResultOutcome == nil || (ok && eb != nil) {
			if tb == nil {
				tb = &core.Outcomes[core.Duration]{And: func(a, b core.Duration) core.Duration { return a + b }}
			}
			finalCombinedOutcome = tb.Append(eb)
		} else {
			return fmt.Errorf("%w: cannot combine then (%T) and else (%T)", ErrTypeMismatch, tb, elseBranchResultOutcome)
		}
	// Add cases for other types (bool, int, float, string...) if blocks can return them
	case nil: // Only else branch executed (or neither)
		// Simply use the else branch result (which might also be nil)
		finalCombinedOutcome = elseBranchResultOutcome // Handles nil/nil -> nil
	default:
		return fmt.Errorf("%w: cannot combine branches with type %T", ErrUnsupportedType, tb)
	}

	// --- Sanity Check: Ensure stack is empty before pushing final result ---
	if len(v.stack) != 0 {
		fmt.Printf("WARNING: Stack not empty before final push in evalIfStmt. Len=%d. Contents: %s\n", len(v.stack), v.stackString())
		// Optionally clear it? This might hide the real bug.
		// v.ClearStack()
	}

	// 6. Push the final combined result onto the stack
	// If both branches were nil/empty, finalCombinedOutcome might be nil. Push identity?
	if finalCombinedOutcome == nil {
		// Push a default identity outcome? This seems problematic.
		// If an if/else results in no possible outcomes, maybe that's okay?
		// Let's push nil for now, caller needs to handle.
		fmt.Println("Warning: IfStmt resulted in nil combined outcome.")
		v.push(nil) // Or push an empty outcome of a default type?
	} else {
		v.push(finalCombinedOutcome)
	}

	// TODO: Apply reduction to the final combined outcome? Yes.
	// Need to pop, check type, trim, push back.
	if finalCombinedOutcome != nil {
		finalPopped, _ := v.pop() // Pop the combined result we just pushed
		reductionNeeded := false
		// Check length based on type
		switch fc := finalPopped.(type) {
		case *core.Outcomes[core.AccessResult]:
			reductionNeeded = fc.Len() > v.maxOutcomeLen
			// Add other types if they need reduction
		}

		if reductionNeeded {
			switch fc := finalPopped.(type) {
			case *core.Outcomes[core.AccessResult]:
				trimmerFuncGen := core.TrimToSize(v.maxOutcomeLen+50, v.maxOutcomeLen)
				successes, failures := fc.Split(core.AccessResult.IsSuccess)
				trimmedSuccesses := trimmerFuncGen(successes)
				trimmedFailures := trimmerFuncGen(failures)
				finalTrimmed := (&core.Outcomes[core.AccessResult]{And: fc.And}).Append(trimmedSuccesses, trimmedFailures)
				v.push(finalTrimmed) // Push trimmed result
				// fmt.Printf("Applied TrimToSize after IF, len %d -> %d\n", finalLen, finalTrimmed.Len()) // Debug log
			// Add other trimmable types
			default:
				v.push(finalPopped) // Push original back if no trimmer
			}
		} else {
			v.push(finalPopped) // Push original back if no reduction needed
		}
	}

	return nil
}
*/
