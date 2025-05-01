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
func (v *VM) evalAssignmentStmt(stmt *AssignmentStmt) error {
	// Evaluate the RHS expression
	resultState, err := v.Eval(stmt.Value)
	if err != nil {
		// Check for ReturnValue signal propagation
		var retVal *ReturnValue
		if errors.As(err, &retVal) {
			// A return occurred during RHS evaluation. Need to propagate.
			// The actual result is in retVal.Value (*VarState)
			stateToStore := retVal.Value
			if stateToStore != nil {
				v.env.Set(stmt.Variable.Name, stateToStore.ValueOutcome, stateToStore.LatencyOutcome)
				v.push(stateToStore) // Push the state before returning signal
			}
			return err // Return the state AND the signal error
		}
		// Otherwise, it's a real evaluation error
		return fmt.Errorf("error evaluating assignment value for '%s': %w", stmt.Variable.Name, err)
	}

	// Eval succeeded, resultState holds the VarState
	if resultState == nil {
		// This shouldn't happen if Eval returns nil state only on error
		return fmt.Errorf("internal error: Eval returned nil VarState without error for assignment '%s'", stmt.Variable.Name)
	}

	// Store the value and latency outcomes from the VarState in the environment
	v.env.Set(stmt.Variable.Name, resultState.ValueOutcome, resultState.LatencyOutcome)
	v.push(resultState)

	// Return the evaluated state (as assignments can be part of sequences)
	return nil
}

// evalExprStmt evaluates the expression and pushes its result state onto the stack.
// Returns only an error (including ReturnValue signal).
func (v *VM) evalExprStmt(stmt *ExprStmt) error {
	// Evaluate the expression
	resultState, err := v.Eval(stmt.Expression)
	if err != nil {
		// Check for ReturnValue signal propagation
		var retVal *ReturnValue
		if errors.As(err, &retVal) {
			if retVal.Value != nil {
				v.push(retVal.Value)
			} // Push state before returning signal
			return err // Propagate signal
		}
		return fmt.Errorf("error evaluating expression statement: %w", err) // Real error
	}

	if resultState == nil {
		return fmt.Errorf("internal error: Eval returned nil state without error for expression statement")
	}

	// Push the result onto the stack
	v.push(resultState)
	return nil // Success
}

// evalReturnStmt evaluates the return expression, wraps the resulting VarState
// in a ReturnValue error signal, and returns both.
func (v *VM) evalReturnStmt(stmt *ReturnStmt) error {
	// Evaluate the return value expression
	resultState, err := v.Eval(stmt.ReturnValue)
	if err != nil {
		// Check for return propagation from within the expression
		var retVal *ReturnValue
		if errors.As(err, &retVal) {
			if retVal.Value != nil {
				v.push(retVal.Value)
			} // Push nested return value
			return err // Propagate the nested signal
		}
		return fmt.Errorf("error evaluating return value: %w", err) // Real error
	}

	if resultState == nil {
		return fmt.Errorf("internal error: Eval returned nil state without error for return value")
	}

	// Push the result state onto the stack
	v.push(resultState)

	// Wrap the result state in our special error type to signal return
	returnSignal := &ReturnValue{Value: resultState, Err: ErrReturnSignal}

	// Return the signal wrapper error
	return returnSignal
}

// Helper function to create a default identity state (e.g., for empty blocks or start)
// We need a concrete value type, bool=true seems reasonable for a neutral start.
func createIdentityState() *VarState {
	return &VarState{
		ValueOutcome:   identityValueOutcome(), // Uses helper returning Outcomes[bool]{1.0=>true}
		LatencyOutcome: zeroLatencyOutcome(),   // Uses helper returning Outcomes[Duration]{1.0=>0}
	}
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
	if initialContext != nil {
		v.push(initialContext)
	} else {
		// Push identity state if no context provided?
		v.push(createIdentityState())
	}

	// fmt.Printf("DEBUG evalBlockStmt Start: Initial Stack Len=%d, Top=%v\n", len(v.stack), v.peek()) // Debug

	for i, stmt := range block.Statements {
		// Remember stack depth before evaluating statement
		// stackDepthBeforeStmt := len(v.stack)

		// Evaluate the statement. It might push a result. It returns error/signal.
		evalErr := error(nil)
		switch s := stmt.(type) {
		// Call specific eval*Stmt funcs (which now return only error)
		case *AssignmentStmt:
			evalErr = v.evalAssignmentStmt(s)
		case *ExprStmt:
			evalErr = v.evalExprStmt(s)
		case *ReturnStmt:
			evalErr = v.evalReturnStmt(s)
		case *IfStmt:
			evalErr = v.evalIfStmt(s) // If also needs update
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
				// A return occurred. The return value is on TOP of the stack.
				// Pop everything else pushed *during this block's execution*
				// leaving ONLY the return value.
				if len(v.stack) > stackStartLen {
					// Pop items until only the one pushed by the return sequence remains
					// The state returned by the stmt is already wrapped in retVal
					// We need to ensure the stack *only* contains that state.
					// Pop the item pushed by the return eval itself.
					returnedStateFromStack, popErr := v.pop()
					if popErr != nil || returnedStateFromStack != retVal.Value {
						// This indicates a major inconsistency
						v.stack = v.stack[:stackStartLen] // Attempt cleanup
						return nil, fmt.Errorf("internal error: stack state inconsistent after return signal")
					}
				} else {
					// Return occurred without pushing? Should not happen.
					v.stack = v.stack[:stackStartLen] // Attempt cleanup
					return nil, fmt.Errorf("internal error: return signal occurred but stack state is too shallow")
				}

				// Clean stack below the return value (if any were pushed before return)
				v.stack = v.stack[:stackStartLen]
				// Push the actual return value back
				v.push(retVal.Value)

				// Propagate signal
				return retVal.Value, evalErr
			}
			// Otherwise, it's a real error. Clean up stack.
			v.stack = v.stack[:stackStartLen]
			return nil, fmt.Errorf("error executing statement %d (%T): %w", i, stmt, evalErr)
		}

		// --- Implicit Sequential Composition via Stack ---
		// After a statement executes successfully AND pushed a result,
		// check if we need to combine it with the previous result.
		// Combine if stack > initialDepth + 1 (meaning at least two results from this block are present)
		// Let's adjust: Combine if stack has >= 2 items *total* (simpler logic)
		// --- This needs refinement --- Combine only the results relevant to this block's sequence.
		// How many items did the statement push? Assume 1 for now (Assignment, ExprStmt, If).
		// If stack depth is now > stackStartLen + 1, it means we have the result from the *previous*
		// iteration AND the result from the *current* iteration. Combine them.

		if len(v.stack) >= stackStartLen+2 { // We have previous result + current result
			// fmt.Printf("DEBUG evalBlockStmt Combine Triggered: Stack Len=%d\n", len(v.stack)) // Debug
			stateB, _ := v.pop() // Current statement result
			stateA, _ := v.pop() // Previous statement result (or initial context)

			combinedState, combineErr := v.combineOutcomesAndReduce(stateA, stateB)
			if combineErr != nil {
				v.stack = v.stack[:stackStartLen] // Clean up
				return nil, fmt.Errorf("implicit AND failed after statement %d (%T): %w", i, stmt, combineErr)
			}
			v.push(combinedState) // Push combined result back
		} else if len(v.stack) == stackStartLen {
			// Statement didn't push a result (e.g., maybe a future 'break' or 'continue'?)
			// Or internal error. For now, assume statements relevant to sequence push one result.
			fmt.Printf("Warning: Statement %T completed without pushing result to stack?\n", stmt)
		}

		// fmt.Printf("DEBUG evalBlockStmt End Iter %d (%T): Stack Len=%d, Top=%v\n", i, stmt, len(v.stack), v.peek()) // Debug

	} // End loop over statements

	// Block finished normally. The final result is the single item left on the stack
	// relative to where this block started.
	finalStackDepth := len(v.stack)
	if finalStackDepth != stackStartLen+1 {
		// If stack depth is wrong, something went wrong with push/pop/combine logic
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
