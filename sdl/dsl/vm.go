// sdl/dsl/vm.go
package dsl

import (
	"errors" // Import errors package
	"fmt"
	"reflect"

	"github.com/panyam/leetcoach/sdl/core"
	// We will need core later, but not for phase 0 structure
	// "github.com/panyam/leetcoach/sdl/core"
)

var (
	ErrStackUnderflow       = errors.New("vm stack underflow")
	ErrNotImplemented       = errors.New("evaluation for this node type not implemented")
	ErrNotFound             = errors.New("identifier not found")
	ErrInternalFuncNotFound = errors.New("internal function not found")
	ErrReturnSignal         = errors.New("return signal")
	// Add more specific errors as needed
)

// ReturnValue wraps the actual return value when ErrReturnSignal is returned.
type ReturnValue struct {
	Value *VarState // Holds the VarState returned
	Err   error     // Embed the signal error
}

// Implement the error interface for ReturnValue
func (rv *ReturnValue) Error() string {
	return fmt.Sprintf("ReturnSignal(%+v)", rv.Value) // Show VarState content
}

// Unwrap allows errors.Is(err, ErrReturnSignal) to work
func (rv *ReturnValue) Unwrap() error {
	return ErrReturnSignal
}

// InternalFunction defines the signature for built-in functions callable by the VM.
// It receives the vm instance (for potential access to env/stack) and evaluated arguments.
type InternalFunction func(vm *VM, args []*VarState) (*VarState, error)

// --- Reducer Registry ---

// ReducerKey identifies a pair of types for dispatching reducer functions.
// Using reflect.Type directly might be problematic with generics. Let's try string representation first.
type ReducerKey struct {
	LeftType  string // e.g., "*core.Outcomes[core.AccessResult]"
	RightType string
}

// VM holds the state for evaluating an AST.
type VM struct {
	stack              []*VarState
	env                *Environment
	internalFuncs      map[string]InternalFunction
	maxOutcomeLen      int
	sequentialReducers map[ReducerKey]core.ReducerFunc[any, any, any] // Registry for AND
	durationReducer    core.DurationReducerFunc                       // <-- Add configuration field
}

// NewVM creates a new vm instance.
func NewVM(maxBuckets int) *VM {
	if maxBuckets <= 0 {
		maxBuckets = 15 // Default if invalid
	}
	interp := &VM{
		stack:              make([]*VarState, 0, 100), // Initial capacity
		env:                NewEnvironment(),          // Start with a global environment
		internalFuncs:      make(map[string]InternalFunction),
		sequentialReducers: make(map[ReducerKey]core.ReducerFunc[any, any, any]),
		maxOutcomeLen:      maxBuckets,
		durationReducer:    core.ReduceDurationsAdaptive, // <-- Set default reducer
	}
	interp.registerDefaultReducers() // Register standard reducers
	return interp
}

// SetDurationReducer allows changing the reduction strategy.
func (v *VM) SetDurationReducer(reducer core.DurationReducerFunc) {
	if reducer != nil {
		v.durationReducer = reducer
	} else {
		// Maybe default back or error? Default back seems safer.
		v.durationReducer = core.ReduceDurationsAdaptive
	}
}

// Env returns the current environment the vm is using.
func (v *VM) Env() *Environment {
	return v.env
}

// Helper to provide access to VM environment during setup
func (v *VM) SetEnv(e *Environment) {
	v.env = e
}

// Helper to access outer environment
func (e *Environment) Outer() *Environment {
	return e.outer
}

// --- Internal Function Registry ---

func (v *VM) RegisterInternalFunc(name string, fn InternalFunction) {
	v.internalFuncs[name] = fn
}

// --- Evaluation (Stub) ---

// Eval is the main entry point for evaluating an AST node.
// It uses a type switch to delegate to specific eval methods.
// The result of an evaluation is typically left on the vm's stack.
// Returns the final result (often the top of the stack after full eval) and any error.
func (v *VM) Eval(node Node) (*VarState, error) {
	// fmt.Printf("Eval entry: %T - %s\n", node, node) // Debug entry
	var resultState *VarState
	var err error
	switch n := node.(type) {
	// We'll add cases here in subsequent phases
	case *LiteralExpr:
		resultState, err = v.evalLiteral(n)
	case *IdentifierExpr:
		resultState, err = v.evalIdentifier(n)
	case *InternalCallExpr:
		resultState, err = v.evalInternalCall(n)
	case *AndExpr:
		resultState, err = v.evalAndExpr(n)

	// --- Statement Nodes ---
	case *BlockStmt:
		// When Eval is called directly on a BlockStmt (e.g., top level),
		// there is no initial context from an outer structure like IfStmt.
		// evalBlockStmt now returns the result directly, doesn't leave on stack implicitly
		resultState, err = v.evalBlockStmt(n, v.env, nil) // Pass nil context
	case *AssignmentStmt:
		resultState, err = v.evalAssignmentStmt(n)
	case *IfStmt: // <-- Will be implemented now
		resultState, err = v.evalIfStmt(n)
	case *ReturnStmt:
		resultState, err = v.evalReturnStmt(n) // Return signals via special error/value
	case *MemberAccessExpr:
		resultState, err = v.evalMemberAccessExpr(n) // <-- Call the actual implementation
	case *ExprStmt:
		resultState, err = v.evalExprStmt(n)
		/*
			case *CallExpr:
				resultState, err = v.evalCallExpr(n)
			case *RepeatExpr:
				resultState, err = v.evalRepeatExpr(n)
			// ... other cases ...
			case *DistributeExpr: // Placeholder for distribute expression
				// resultState, err = v.evalDistributeExpr(n) // Implement this
				return nil, fmt.Errorf("%w: DistributeExpr", ErrNotImplemented)
			case *DelayStmt: // Placeholder for delay statement
				// resultState, err = v.evalDelayStmt(n) // Implement this
				return nil, fmt.Errorf("%w: DelayStmt", ErrNotImplemented)
			// Add placeholders for ParallelExpr, FanoutExpr, etc. if they are in ast.go
			case *ParallelExpr:
				return nil, fmt.Errorf("%w: ParallelExpr", ErrNotImplemented)
			case *FanoutExpr:
				return nil, fmt.Errorf("%w: FanoutExpr", ErrNotImplemented)
			case *FilterExpr:
				return nil, fmt.Errorf("%w: FilterExpr", ErrNotImplemented)
			case *SwitchExpr:
				return nil, fmt.Errorf("%w: SwitchExpr", ErrNotImplemented)
		*/
	default:
		return nil, fmt.Errorf("Eval not implemented for node type %T", node)
	}
	// Error occurred during evaluation
	if err != nil {
		// Check for return signal - propagate it with the resultState extracted above if needed
		if _, ok := err.(*ReturnValue); ok {
			// Caller needs to check for this specific error type
			return resultState, err
		}
		// Otherwise, return nil state and the actual error
		return nil, err
	}

	// Return the resulting state
	return resultState, nil
}

func (v *VM) GetFinalResult() (*VarState, error) {
	if len(v.stack) == 0 {
		return nil, fmt.Errorf("cannot get final result: stack is empty")
	}
	if len(v.stack) > 1 {
		// This might be okay if intermediate Eval calls left things? Needs review.
		// Let's keep the strict check for now.
		return nil, fmt.Errorf("cannot get final result: stack contains multiple items (%d). Contents: %s", len(v.stack), v.stackString())
	}
	state := v.stack[0]
	if state == nil {
		panic(fmt.Sprintf("internal error: final stack item is not VarState (%T)", v.stack[0]))
	}
	return state, nil
}

// Add stubs for other eval functions as needed...

// --- Combination & Reduction Helper ---

// Helper to create a zero latency outcome
func zeroLatencyOutcome() *core.Outcomes[core.Duration] {
	o := (&core.Outcomes[core.Duration]{}).Add(1.0, 0.0)
	// Should have the correct Adder func registered
	// o.And = func(a, b core.Duration) core.Duration { return a + b }
	return o
}

// Helper to create an identity value outcome (e.g., for delay)
// We need a concrete type, maybe bool=true?
func identityValueOutcome() *core.Outcomes[bool] {
	o := (&core.Outcomes[bool]{}).Add(1.0, true)
	// o.And = func(a, b bool) bool { return a && b } // Example
	return o
}

// combineOutcomesAndReduce takes two VarStates, combines their latency tracks,
// uses the value track from the *second* state, and applies reduction.
// Returns the final combined VarState.
func (v *VM) combineOutcomesAndReduce(stateA, stateB *VarState) (*VarState, error) {
	if stateA == nil || stateB == nil {
		// Handle cases where one input might be nil (e.g., start of block)
		if stateA == nil && stateB == nil {
			return nil, nil
		} // Or return identity? Requires thought.
		if stateA == nil {
			return stateB, nil
		}
		if stateB == nil {
			return stateA, nil
		}
		// This logic might need refinement depending on how nil context is handled.
	}

	var combinedLatOutcome *core.Outcomes[core.Duration] // Expecting Duration outcome

	// --- Combine Latencies ---
	latA := stateA.LatencyOutcome
	latB := stateB.LatencyOutcome

	// Handle nil latency outcomes (treat as zero latency)
	if latA == nil {
		latA = zeroLatencyOutcome()
	}
	if latB == nil {
		latB = zeroLatencyOutcome()
	}

	// Check types before combining
	latAOutcome, okA := latA.(*core.Outcomes[core.Duration])
	latBOutcome, okB := latB.(*core.Outcomes[core.Duration])
	if !okA || !okB {
		return nil, fmt.Errorf("internal error: latency tracks are not *core.Outcomes[core.Duration] (A: %T, B: %T)", latA, latB)
	}

	// Use Reducer Registry for Duration + Duration
	key := ReducerKey{
		LeftType:  getOutcomeTypeString(latAOutcome), // Should be Outcomes[Duration]
		RightType: getOutcomeTypeString(latBOutcome), // Should be Outcomes[Duration]
	}
	registeredReducer, found := v.sequentialReducers[key]
	if !found {
		// Should have been registered in NewVM
		return nil, fmt.Errorf("internal error: no reducer found for Duration + Duration")
	}

	// Perform the combination using the registered reducer
	combinedLatRaw := registeredReducer(latAOutcome, latBOutcome)
	var ok bool
	combinedLatOutcome, ok = combinedLatRaw.(*core.Outcomes[core.Duration])
	if !ok {
		// Reducer returned wrong type?
		return nil, fmt.Errorf("internal error: duration reducer returned unexpected type %T", combinedLatRaw)
	}

	// --- Value Track (Take Last) ---
	// The final value distribution is simply the one from stateB.
	finalValOutcome := stateB.ValueOutcome

	// --- Create Combined State ---
	// Initially use the combined latency and the value from B
	combinedState := &VarState{
		ValueOutcome:   finalValOutcome,
		LatencyOutcome: combinedLatOutcome,
	}

	// --- Apply Reduction (to Latency Track) ---
	// Reduction primarily applies to the latency track as it grows complex.
	// Value track complexity is managed by discrete types / `distribute`.
	reductionNeeded := false
	if combinedLatOutcome != nil && combinedLatOutcome.Len() > v.maxOutcomeLen {
		reductionNeeded = true
	}

	if reductionNeeded {
		// Currently, only know how to trim AccessResult/RangedResult easily.
		// Need a way to trim Duration outcomes. Interpolation might be best.
		// Placeholder: If we had core.InterpolateDurations(o, maxLen)
		// trimmedLat := core.InterpolateDurations(combinedLatOutcome, v.maxOutcomeLen)
		// combinedState.LatencyOutcome = trimmedLat
		// For now, let's skip trimming Duration until a method exists.
		// fmt.Printf("Warning: Latency outcome length (%d) exceeded limit (%d), but no Duration trimmer implemented.\n", combinedLatOutcome.Len(), v.maxOutcomeLen)

		// --- If Value is AccessResult, trim it too? ---
		// This depends: Should reduction be applied synchronously with latency?
		// Let's apply it if the corresponding type supports it.
		if arVal, ok := combinedState.ValueOutcome.(*core.Outcomes[core.AccessResult]); ok {
			if arVal.Len() > v.maxOutcomeLen {
				trimmerFuncGen := core.TrimToSize(v.maxOutcomeLen+50, v.maxOutcomeLen)
				successes, failures := arVal.Split(core.AccessResult.IsSuccess)
				trimmedSuccesses := trimmerFuncGen(successes)
				trimmedFailures := trimmerFuncGen(failures)
				finalTrimmedVal := (&core.Outcomes[core.AccessResult]{And: arVal.And}).Append(trimmedSuccesses, trimmedFailures)
				combinedState.ValueOutcome = finalTrimmedVal
			}
		}
		// Add RangedResult trimming if needed
	}

	return combinedState, nil
}

// Update NeedsReduction Helper
func (v *VM) needsReduction(state *VarState) bool {
	if state == nil {
		return false
	}
	// Primarily check latency track for complexity
	if latOutcome, ok := state.LatencyOutcome.(*core.Outcomes[core.Duration]); ok {
		if latOutcome.Len() > v.maxOutcomeLen {
			return true
		}
	}
	// Optionally check value track if specific types need trimming
	switch valOutcome := state.ValueOutcome.(type) {
	case *core.Outcomes[core.AccessResult]:
		return valOutcome.Len() > v.maxOutcomeLen
	case *core.Outcomes[core.RangedResult]:
		return valOutcome.Len() > v.maxOutcomeLen
	}
	return false
}

// --- Reducer Registry ---

func getOutcomeTypeString(outcome any) string {
	// Use reflection to get a string representation like "*core.Outcomes[core.AccessResult]"
	// This is somewhat fragile but avoids complex generic type reflection for now.
	if outcome == nil {
		return "<nil>"
	}
	return reflect.TypeOf(outcome).String()
}

func (v *VM) RegisterSequentialReducer(leftExample, rightExample any, reducer core.ReducerFunc[any, any, any]) error {
	if reducer == nil {
		return fmt.Errorf("reducer function cannot be nil")
	}
	// Use type strings as keys for simplicity
	key := ReducerKey{
		LeftType:  getOutcomeTypeString(leftExample),
		RightType: getOutcomeTypeString(rightExample),
	}
	if key.LeftType == "<nil>" || key.RightType == "<nil>" {
		return fmt.Errorf("cannot register reducer with nil example types")
	}

	// fmt.Printf("DEBUG: Registering Reducer for Key: %+v\n", key) // Debug registration
	v.sequentialReducers[key] = reducer
	return nil
}

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

// --- Stack Operations ---

// push adds a VarState to the stack.
func (v *VM) push(state *VarState) {
	if state == nil {
		// Avoid pushing nil VarState, maybe push a default state? Or panic?
		// Let's panic for now, indicates an error in eval logic.
		panic("attempting to push nil VarState onto VM stack")
	}
	v.stack = append(v.stack, state)
}

// pop removes and returns the top VarState from the stack.
func (v *VM) pop() (*VarState, error) {
	if len(v.stack) == 0 {
		return nil, ErrStackUnderflow
	}
	lastIndex := len(v.stack) - 1
	state := v.stack[lastIndex]
	v.stack = v.stack[:lastIndex]
	return state, nil
}

// peek returns the top VarState without removing it.
func (v *VM) peek() (*VarState, bool) {
	if len(v.stack) == 0 {
		return nil, false
	}
	state := v.stack[len(v.stack)-1]
	return state, true
}

// stackString provides a string representation of the stack for debugging
func (v *VM) stackString() string {
	// Modify to show VarState info if useful
	L := len(v.stack)
	items := make([]string, L)
	for idx, state := range v.stack {
		// Provide a more informative string for VarState
		valTypeStr := "nil"
		if state.ValueOutcome != nil {
			valTypeStr = getOutcomeTypeString(state.ValueOutcome)
		}
		latTypeStr := "nil"
		if state.LatencyOutcome != nil {
			latTypeStr = getOutcomeTypeString(state.LatencyOutcome)
		}
		items[idx] = fmt.Sprintf("State{V:%s, L:%s}", valTypeStr, latTypeStr)
	}
	return fmt.Sprintf("Stack(len=%d): %v", L, items)
}

// ClearStack resets the stack.
func (v *VM) ClearStack() {
	v.stack = v.stack[:0] // No change needed here
}
