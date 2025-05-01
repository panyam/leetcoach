package dsl

import (
	"errors"
)

var (
	ErrStackUnderflow = errors.New("vm stack underflow")
	ErrStackOverflow  = errors.New("vm stack overflow")
)

// push adds a VarState to the stack.
func (v *VM) push(state *VarState) error {
	if v.sp >= VmStackCapacity {
		return v.runtimeError("%v: max stack size %d reached", ErrStackOverflow, len(v.stack))
	}
	if state == nil {
		panic("internal error: attempting to push nil VarState onto VM stack")
	}
	v.stack[v.sp] = state
	v.sp++
	return nil
}

// pop removes and returns the top VarState from the stack.
func (v *VM) pop() (*VarState, error) {
	if v.sp == 0 {
		return nil, v.runtimeError("%v", ErrStackUnderflow) // Add line info
	}
	v.sp--
	state := v.stack[v.sp]
	// v.stack[v.sp] = nil // Optional: Help GC
	return state, nil
}

// peek returns the item offset positions down from the top *without* removing it.
// peek(0) returns the top element. peek(1) returns the next element down.
func (v *VM) peek(offset int) (*VarState, error) {
	peekIndex := v.sp - 1 - offset
	if peekIndex < 0 || peekIndex >= v.sp { // Check bounds relative to sp
		return nil, v.runtimeError("%v (offset %d, sp %d)", ErrStackUnderflow, offset, v.sp)
	}
	return v.stack[peekIndex], nil
}

// stackString provides a string representation of the stack for debugging
/*
func (v *VM) stackString() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("[SP=%d:", v.sp))
	for i := range v.sp {
		builder.WriteString(" ")
		builder.WriteString(formatConstant(v.stack[i]))
	}
	builder.WriteString(" ]")
	return builder.String()
}
*/

// ClearStack resets the stack.
func (v *VM) ClearStack() {
	v.stack = v.stack[:0] // No change needed here
}
