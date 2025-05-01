package dsl

import (
	"fmt"
	"reflect"

	"github.com/panyam/leetcoach/sdl/core"
)

// OpCode represents a single bytecode operation.
// Using uint8 for compactness.
type OpCode uint8

// Define initial opcodes.
const (
	OpIllegal OpCode = iota // Represents an invalid instruction

	// Stack & Constants
	OpPushConst // Operand: Constant Pool Index (int)
	OpPop       // Pop top of stack

	// Variables (Operands: Index - int)
	OpDefineGlobal // Operand: Constant Pool Index (Name string)
	OpGetGlobal    // Operand: Constant Pool Index (Name string)
	OpSetGlobal    // Operand: Constant Pool Index (Name string)
	OpGetLocal     // Operand: Local Slot Index (int)
	OpSetLocal     // Operand: Local Slot Index (int)

	// Control Flow (Operand: Jump Offset - int relative to *next* instruction)
	OpJump           // Unconditional jump
	OpJumpIfFalsePop // Pops condition VarState, jumps if falsey

	// Function/Method/Handler Calls
	OpCallFunc // Operand: Handler ID (int) -> Calls registered InternalFunction
	OpCall     // Operand: ArgCount (int) -> Calls user/component function/method on stack
	OpReturn   // Return from current function/operation/script
)

type Instruction struct {
	Opcode  OpCode
	Operand int // Generic operand
	Line    int // source line number
}

// Chunk represents compiled bytecode.
type Chunk struct {
	Name         string
	Instructions []Instruction
	Constants    []any // Constant pool (literals, function names, etc.)
}

// NewChunk creates an empty chunk.
func NewChunk(name string) *Chunk {
	return &Chunk{
		Name:         name,
		Instructions: []Instruction{},
		Constants:    []any{},
	}
}

// Write appends a byte (opcode or operand) to the chunk's code.
func (c *Chunk) Write(op OpCode, operand int, line int) int {
	offset := len(c.Instructions)
	c.Instructions = append(c.Instructions, Instruction{Opcode: op, Operand: operand, Line: line})
	return offset
}

// AddConstant adds a value to the chunk's constant pool and returns its index.
func (c *Chunk) AddConstant(value any) int {
	// Check if constant already exists (optional optimization)
	// for i, constant := range c.Constants {
	//     if constant == value { // Note: Deep equality might be needed for complex types
	//         return i
	//     }
	// }
	c.Constants = append(c.Constants, value)
	return len(c.Constants) - 1
}

// Dissassemble prints a human-readable representation of the chunk (for debugging).
func (c *Chunk) Disassemble() {
	fmt.Printf("== Dissassembly: %s ==\n", c.Name)
	if len(c.Instructions) == 0 {
		fmt.Println("(No instructions)")
	}
	for offset := 0; offset < len(c.Instructions); offset++ {
		c.disassembleInstruction(offset)
	}
	fmt.Printf("== Constants (%d) for %s ==\n", len(c.Constants), c.Name)
	for i, constant := range c.Constants {
		fmt.Printf("[%04d] %s\n", i, formatConstant(constant)) // Use helper
	}
	fmt.Printf("== End Dissassembly: %s ==\n", c.Name)
}

// disassembleInstruction decodes and prints a single instruction struct.
func (c *Chunk) disassembleInstruction(offset int) {
	instr := c.Instructions[offset]
	fmt.Printf("%04d %4d ", offset, instr.Line) // Show offset and line

	opName := "Unknown"
	operandStr := ""

	switch instr.Opcode {
	case OpPushConst:
		opName = "OpPushConst"
		if instr.Operand >= 0 && instr.Operand < len(c.Constants) {
			operandStr = fmt.Sprintf("%4d '%s'", instr.Operand, formatConstant(c.Constants[instr.Operand]))
		} else {
			operandStr = fmt.Sprintf("%4d (invalid index)", instr.Operand)
		}
	case OpPop:
		opName = "OpPop"
	case OpDefineGlobal:
		opName = "OpDefineGlobal"
		if instr.Operand >= 0 && instr.Operand < len(c.Constants) {
			operandStr = fmt.Sprintf("NameIdx:%d '%s'", instr.Operand, formatConstant(c.Constants[instr.Operand]))
		} else {
			operandStr = fmt.Sprintf("NameIdx:%d (invalid)", instr.Operand)
		}
	case OpGetGlobal:
		opName = "OpGetGlobal"
		if instr.Operand >= 0 && instr.Operand < len(c.Constants) {
			operandStr = fmt.Sprintf("NameIdx:%d '%s'", instr.Operand, formatConstant(c.Constants[instr.Operand]))
		} else {
			operandStr = fmt.Sprintf("NameIdx:%d (invalid)", instr.Operand)
		}
	case OpSetGlobal:
		opName = "OpSetGlobal"
		if instr.Operand >= 0 && instr.Operand < len(c.Constants) {
			operandStr = fmt.Sprintf("NameIdx:%d '%s'", instr.Operand, formatConstant(c.Constants[instr.Operand]))
		} else {
			operandStr = fmt.Sprintf("NameIdx:%d (invalid)", instr.Operand)
		}
	case OpGetLocal:
		opName = "OpGetLocal"
		operandStr = fmt.Sprintf("Slot:%d", instr.Operand)
	case OpSetLocal:
		opName = "OpSetLocal"
		operandStr = fmt.Sprintf("Slot:%d", instr.Operand)
	case OpJump:
		opName = "OpJump"
		target := offset + 1 + instr.Operand // Relative jump offset calculation
		operandStr = fmt.Sprintf("%d -> %d", instr.Operand, target)
	case OpJumpIfFalsePop:
		opName = "OpJumpIfFalsePop"
		target := offset + 1 + instr.Operand // Relative jump offset calculation
		operandStr = fmt.Sprintf("%d -> %d", instr.Operand, target)
	case OpCallFunc:
		opName = "OpCallFunc"
		// TODO: We need the VM to map Handler ID back to name for better disassembly
		operandStr = fmt.Sprintf("HandlerID:%d", instr.Operand)
	case OpCall:
		opName = "OpCall"
		operandStr = fmt.Sprintf("ArgCount:%d", instr.Operand)
	case OpReturn:
		opName = "OpReturn"
	case OpIllegal:
		opName = "OpIllegal"
	default:
		opName = fmt.Sprintf("Unknown (%d)", instr.Opcode)
		operandStr = fmt.Sprintf("%d", instr.Operand)
	}

	fmt.Printf("%-16s %s\n", opName, operandStr)
}

// formatConstant helper (minor tweaks for clarity)
func formatConstant(value any) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	case *VarState:
		valStr := "nil"
		if v.ValueOutcome != nil {
			var detVal any
			var isDet bool
			// Expanded type checks for deterministic values
			switch vo := v.ValueOutcome.(type) {
			case *core.Outcomes[int64]:
				detVal, isDet = vo.GetValue()
			case *core.Outcomes[string]:
				detVal, isDet = vo.GetValue()
			case *core.Outcomes[bool]:
				detVal, isDet = vo.GetValue()
			// case *core.Outcomes[float64]: detVal, isDet = vo.GetValue()
			case *core.Outcomes[core.Duration]:
				detVal, isDet = vo.GetValue()
			case *core.Outcomes[core.AccessResult]:
				valStr = "*O[AR]" // Abbreviation
			default:
				valStr = getOutcomeTypeString(v.ValueOutcome)
			}
			if isDet {
				valStr = fmt.Sprintf("%v", detVal)
			}
		}

		latStr := "0ms"
		if v.LatencyOutcome != nil {
			if lo, ok := v.LatencyOutcome.(*core.Outcomes[core.Duration]); ok {
				if latVal, isDet := lo.GetValue(); isDet {
					if latVal == 0 {
						// latStr = "0ms" // Keep default
					} else {
						latStr = fmt.Sprintf("%.4fms", latVal*1000.0)
					}
				} else {
					latStr = "*O[Dur]" // Abbreviation
				}
			} else {
				latStr = getOutcomeTypeString(v.LatencyOutcome)
			}
		}
		return fmt.Sprintf("State{V:%s, L:%s}", valStr, latStr)

	case int, int64, float64, bool: // Handle raw literals if compiler uses them
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%T", v) // Show type for unknown constants
	}
}

// getOutcomeTypeString helper (needed by formatConstant)
func getOutcomeTypeString(outcome any) string {
	if outcome == nil {
		return "<nil>"
	}
	return reflect.TypeOf(outcome).String() // From vm.go
}

// OpCode.String() for debugging disassemble
func (op OpCode) String() string {
	switch op {
	case OpReturn:
		return "OpReturn"
	// Add other names here as they are implemented
	default:
		return fmt.Sprintf("OpCode(%d)", op)
	}
}
