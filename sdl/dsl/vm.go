// sdl/dsl/vm.go
package dsl

import (
	"errors" // Import errors package
	"fmt"

	"github.com/panyam/leetcoach/sdl/core"
	// We will need core later, but not for phase 0 structure
	// "github.com/panyam/leetcoach/sdl/core"
)

var (
	ErrNotImplemented       = errors.New("evaluation for this node type not implemented")
	ErrNotFound             = errors.New("identifier not found")
	ErrInternalFuncNotFound = errors.New("internal function not found")
	ErrReturnSignal         = errors.New("return signal")
	ErrInvalidOperand       = errors.New("invalid operand type for operation")
	ErrUnsupportedOperation = errors.New("unsupported operation")
	// Add more specific errors as needed
)

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

type OperatorFunc func(vm *VM) error // Takes VM to access stack

// VM holds the state for evaluating an AST.
type VM struct {
	// --- Existing fields ---
	// stack              []*VarState // Use []*VarState
	// env                *Environment
	// internalFuncs      map[string]InternalFunction
	// maxOutcomeLen      int
	// sequentialReducers map[ReducerKey]core.ReducerFunc[any, any, any]
	// durationReducer    core.DurationReducerFunc

	// --- Bytecode Execution Fields ---
	instructions []Instruction // Current instructions being executed
	constants    []any         // Constant pool for the current chunk

	ip    int         // Instruction Pointer: index into chunk.Code
	stack []*VarState // Reuse stack, stores *VarState now
	sp    int         // stack pointer: points *just above* the top element

	// Environment, Internal funcs, Reducers, Configs remain the same
	env                *Environment
	internalHandlers   map[int]InternalFunction
	maxOutcomeLen      int
	sequentialReducers map[ReducerKey]core.ReducerFunc[any, any, any]
	durationReducer    core.DurationReducerFunc

	// Global vars
	globals map[string]*VarState

	// --- Compiler/Scope Info (Maybe needed later) ---
	// compiler *Compiler // If needed for nested function compilation
	// locals []*VarState // Or manage locals via stack offsets
}

const VmStackCapacity = 256

// NewVM creates a new vm instance.
func NewVM(maxBuckets int) *VM {
	if maxBuckets <= 0 {
		maxBuckets = 15 // Default if invalid
	}
	vm := &VM{
		stack:              make([]*VarState, VmStackCapacity),
		env:                NewEnvironment(), // Start with a global environment
		internalHandlers:   make(map[int]InternalFunction),
		sequentialReducers: make(map[ReducerKey]core.ReducerFunc[any, any, any]),
		maxOutcomeLen:      maxBuckets,
		durationReducer:    core.ReduceDurationsAdaptive, // <-- Set default reducer
		ip:                 0,                            // Initialize IP
		globals:            map[string]*VarState{},
	}
	vm.registerDefaultReducers() // Register standard reducers
	vm.registerDefaultHandlers() // <-- Register operator functions
	return vm
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

func (v *VM) RegisterInternalHandler(name int, fn InternalFunction) {
	v.internalHandlers[name] = fn
}

// --- Evaluation (Stub) ---

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
	// --- Combine Values - Propagate Combined Probability ---
	valA := stateA.SafeValueOutcome()
	valB := stateB.SafeValueOutcome()

	// If valA or valB is nil, the result is likely just the non-nil one? Or nil?
	// If only concerned with AccessResult sequence for now:
	finalValOutcome := valB // Start with valB

	// accValA, okA := valA.(*core.Outcomes[core.AccessResult])
	// accValB, okB := valB.(*core.Outcomes[core.AccessResult])

	if okA && okB { // Only attempt weight adjustment if both are AccessResult
		// PROBLEM: We don't have a generic way to get combined SUCCESS probability easily.
		// core.And applied to the *latency* track calculates the correct combined weights per path.
		// Can we use the combined *latency* outcome's weights to reconstruct the value outcome?
		// This assumes a direct correlation between latency buckets and value buckets after And,
		// which *should* hold if the reducers maintain order/structure.

		// Create a new AccessResult outcome based on valB's *values* but combinedLat's *weights*.
		// This is complex and assumes bucket alignment.
		// newValOutcome := &core.Outcomes[core.AccessResult]{}
		// combinedBuckets := combinedLatOutcome.Buckets // Buckets from And(latA, latB)

		// How to map combined latency bucket back to value bucket from B?
		// This approach seems too complex and fragile.

		// --- Simpler Idea: Adjust final availability? No, need full distribution. ---

		// --- Revert for now: Stick to "Take Last Value" ---
		// Acknowledge this is inaccurate for combined probability.
		// We need a better core composition mechanism if this is required.
		finalValOutcome = valB
		fmt.Println("Warning: Using 'Take Last Value' for sequential composition - combined probability not fully propagated to value track.")

	} else if valA == nil && valB != nil {
		finalValOutcome = valB
	} else if valA != nil && valB == nil {
		finalValOutcome = valA // If B is nil, result is A? Or should sequence fail? Assume A.
	} else {
		// Both nil or incompatible types - result is nil? Or identity?
		finalValOutcome = nil // Default to nil if unsure
	}

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

// --- Stack Operations ---

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

// ExecuteChunk runs the bytecode in the given chunk.
func (v *VM) ExecuteChunk(chunk *Chunk) (*VarState, error) {
	if chunk == nil {
		return nil, fmt.Errorf("cannot execute nil chunk")
	}
	if len(chunk.Instructions) == 0 {
		return createIdentityState(), nil
	} // Return identity for empty

	v.instructions = chunk.Instructions
	v.constants = chunk.Constants
	v.ip = 0
	v.sp = 0 // Reset stack pointer

	for {
		if v.ip >= len(v.instructions) {
			return nil, fmt.Errorf("bytecode execution finished without OpReturn")
		}

		instr := v.instructions[v.ip]
		v.ip++ // Advance IP

		// --- Debug Output (Optional) ---
		//fmt.Printf("[%04d] ", v.ip-1)
		//chunk.disassembleInstruction(v.ip - 1) // Disassemble current instruction
		//fmt.Printf("       STACK BEFORE(sp=%d): %s\n", v.sp, v.stackString())
		// --- End Debug ---

		switch instr.Opcode {
		case OpPushConst:
			constIndex := instr.Operand
			if constIndex < 0 || constIndex >= len(v.constants) {
				return nil, v.runtimeError("invalid constant index %d", constIndex)
			}
			constantValue := v.constants[constIndex]
			state, ok := constantValue.(*VarState)
			if !ok {
				return nil, v.runtimeError("constant pool item %d is not *VarState (%T)", constIndex, constantValue)
			}
			if err := v.push(state); err != nil {
				return nil, err
			}

		case OpPop:
			if _, err := v.pop(); err != nil {
				return nil, err
			}

		case OpDefineGlobal:
			nameIndex := instr.Operand
			if nameIndex < 0 || nameIndex >= len(v.constants) {
				return nil, v.runtimeError("invalid global name index %d", nameIndex)
			}
			name, ok := v.constants[nameIndex].(string)
			if !ok {
				return nil, v.runtimeError("global name constant %d is not string", nameIndex)
			}
			// Pop the value from the stack to define the global
			state, err := v.pop()
			if err != nil {
				return nil, v.runtimeError("stack underflow defining global '%s': %v", name, err)
			}
			v.globals[name] = state

		case OpGetGlobal:
			nameIndex := instr.Operand
			if nameIndex < 0 || nameIndex >= len(v.constants) {
				return nil, v.runtimeError("invalid global name index %d", nameIndex)
			}
			name, ok := v.constants[nameIndex].(string)
			if !ok {
				return nil, v.runtimeError("global name constant %d is not string", nameIndex)
			}
			state, found := v.globals[name]
			if !found {
				return nil, v.runtimeError("undefined global variable '%s'", name)
			}
			if err := v.push(state); err != nil {
				return nil, err
			}

		case OpSetGlobal:
			nameIndex := instr.Operand
			if nameIndex < 0 || nameIndex >= len(v.constants) {
				return nil, v.runtimeError("invalid global name index %d", nameIndex)
			}
			name, ok := v.constants[nameIndex].(string)
			if !ok {
				return nil, v.runtimeError("global name constant %d is not string", nameIndex)
			}
			// Peek value from stack (assignment doesn't consume it from sequence)
			state, err := v.peek(0)
			if err != nil {
				return nil, v.runtimeError("stack underflow setting global '%s': %v", name, err)
			}
			if _, found := v.globals[name]; !found {
				// Allow setting new globals? For now, require definition first.
				return nil, v.runtimeError("undefined global variable '%s' (cannot set)", name)
			}
			v.globals[name] = state

		// --- Add OpGetLocal, OpSetLocal later when function calls/scopes are implemented ---

		case OpJump:
			offset := instr.Operand
			v.ip += offset // Adjust IP relative to *next* instruction

		case OpJumpIfFalsePop:
			offset := instr.Operand
			conditionState, err := v.pop()
			if err != nil {
				return nil, err
			}
			isFalsey, err := v.isStateFalsey(conditionState) // Use helper
			if err != nil {
				return nil, err
			} // Error if check fails (e.g., probabilistic bool)
			if isFalsey {
				v.ip += offset
			}

		case OpCallFunc:
			handlerID := instr.Operand
			handler, found := v.internalHandlers[handlerID]
			if !found {
				return nil, v.runtimeError("invalid internal handler ID %d", handlerID)
			}
			// Handler pops args, pushes result(s). It returns error or nil.
			// Passing nil args - handler uses vm stack directly.
			// Handlers should return a *VarState representation of their result
			// *only* if they are signaling a RETURN, otherwise they modify stack
			// and return (nil, nil) for success.
			_, err := handler(v, nil)
			if err != nil {
				var retVal *ReturnValue
				if errors.As(err, &retVal) {
					// Handler signaled a function return. The value is already on stack.
					// We need to terminate *this* execution loop and return that value.
					finalState, popErr := v.pop()
					if popErr != nil {
						return nil, v.runtimeError("stack error after handler return: %v", popErr)
					}
					return finalState, nil // Return the state popped by the handler
				}
				// Handler encountered a runtime error
				return nil, v.runtimeError("handler %d failed: %v", handlerID, err)
			}
			// Handler succeeded, stack is modified. Continue loop.

		case OpCall:
			// TODO: Implement function/method calls
			return nil, v.runtimeError("OpCall not implemented")

		case OpReturn:
			var resultState *VarState
			var err error
			if v.sp == 0 {
				resultState = createIdentityState()
			} else {
				resultState, err = v.pop()
				if err != nil {
					return nil, err
				}
			}
			// fmt.Printf("DEBUG: OpReturn executed, returning state: %s\n", formatConstant(resultState))
			return resultState, nil // Signal successful completion with result

		case OpIllegal:
			fallthrough
		default:
			return nil, v.runtimeError("unknown or illegal opcode %d (%s)", instr.Opcode, instr.Opcode)
		}

		// --- Debug Output (Optional) ---
		//fmt.Printf("       STACK AFTER (sp=%d): %s\n", v.sp, v.stackString())
		// --- End Debug ---
	}
}

// runtimeError formats an error message including the current line number.
func (v *VM) runtimeError(format string, args ...any) error {
	line := -1
	// Use line of the instruction *just executed* (ip was already incremented)
	instrIndex := v.ip - 1
	if v.instructions != nil && instrIndex >= 0 && instrIndex < len(v.instructions) {
		line = v.instructions[instrIndex].Line
	}
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("[line %d] runtime error: %s", line, msg)
}

// isStateFalsey checks if a VarState represents a falsey value (currently bool false).
// CRITICAL: This is only safe for *deterministic* booleans with OpJumpIfFalsePop.
// Probabilistic conditions MUST use OpCallFunc -> vm_handle_if.
func (v *VM) isStateFalsey(state *VarState) (bool, error) {
	if state == nil {
		return true, nil
	} // Treat nil state as falsey

	// Must be a deterministic boolean outcome
	if boolOutcome, ok := state.ValueOutcome.(*core.Outcomes[bool]); ok {
		val, isDet := boolOutcome.GetValue()
		if isDet {
			return !val, nil // Falsey if deterministic false
		}
		return false, v.runtimeError("cannot use probabilistic boolean in simple conditional jump")
	}
	// Other types are considered "truthy"
	return false, nil
}
