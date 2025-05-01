// sdl/dsl/environment.go
package dsl

import (
	"fmt"

	"github.com/panyam/leetcoach/sdl/core"
)

// Environment holds the runtime values for identifiers (variables, functions, components).
// Supports basic scoping via the 'outer' environment.
type Environment struct {
	store map[string]*VarState
	outer *Environment
}

// NewEnvironment creates a fresh top-level environment.
func NewEnvironment() *Environment {
	s := make(map[string]*VarState)
	return &Environment{store: s, outer: nil}
}

// NewEnclosedEnvironment creates a new environment nested within an outer one.
// Useful for function calls or block scopes.
func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

// Get retrieves a value by name. It checks the current environment first,
// then recursively checks outer environments.
func (e *Environment) Get(name string) (*VarState, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		// Not found here, try the outer scope
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

// VarState holds the dual-track outcomes for a variable or expression result.
type VarState struct {
	ValueOutcome   any // *core.Outcomes[V] (V is discrete type)
	LatencyOutcome any // *core.Outcomes[core.Duration]
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

// Set defines or updates a VarState in the *current* environment.
// It takes separate value and latency outcomes and creates/updates the VarState.
func (e *Environment) Set(name string, valueOutcome any, latencyOutcome any) {
	// Ensure latency is always *core.Outcomes[core.Duration] or nil
	if latencyOutcome != nil {
		if _, ok := latencyOutcome.(*core.Outcomes[core.Duration]); !ok {
			// This indicates an internal error, should not happen if logic is correct
			panic(fmt.Sprintf("internal error: attempting to set non-Duration outcome (%T) as latency for '%s'", latencyOutcome, name))
		}
	}

	// Create or update the VarState
	e.store[name] = &VarState{
		ValueOutcome:   valueOutcome,
		LatencyOutcome: latencyOutcome,
	}
}

// String representation for debugging
func (e *Environment) String() string {
	keys := make([]string, 0, len(e.store))
	for k := range e.store {
		keys = append(keys, k)
	}
	return fmt.Sprintf("Env{store: %v, outer: %v}", keys, e.outer != nil)
}
