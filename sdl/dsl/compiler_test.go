package dsl

import (
	"testing"
)

// Helper to compile and check bytecode/constants
// Updated to work with []Instruction
func compileAndCheck(t *testing.T, node Node, expectedInstructions []Instruction, expectedConstants []interface{}) *Chunk {
	t.Helper()
	compiler := NewCompiler()
	chunk, err := compiler.Compile(node)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Check Instruction count
	if len(chunk.Instructions) != len(expectedInstructions) {
		chunk.Disassemble() // Takes no args now
		t.Fatalf("Instruction count mismatch: expected %d, got %d", len(expectedInstructions), len(chunk.Instructions))
	}

	// Check Instructions (Opcode and Operand)
	for i, expectedInstr := range expectedInstructions {
		actualInstr := chunk.Instructions[i]
		if actualInstr.Opcode != expectedInstr.Opcode {
			chunk.Disassemble()
			t.Fatalf("Opcode mismatch at index %d: expected %s, got %s",
				i, expectedInstr.Opcode, actualInstr.Opcode) // Use OpCode stringer
		}
		// Only compare operands if expected operand is meaningful (e.g., not for OpReturn)
		// This simple check might need refinement depending on how operands are used.
		// For now, just compare them directly if the opcode expects one.
		switch expectedInstr.Opcode {
		case OpPushConst, OpDefineGlobal, OpGetGlobal, OpSetGlobal, OpGetLocal, OpSetLocal, OpJump, OpJumpIfFalsePop, OpCallFunc, OpCall:
			if actualInstr.Operand != expectedInstr.Operand {
				chunk.Disassemble()
				t.Fatalf("Operand mismatch at index %d for %s: expected %d, got %d",
					i, expectedInstr.Opcode, expectedInstr.Operand, actualInstr.Operand)
			}
		}
		// We don't compare Line numbers here, assume compiler sets them correctly.
	}

	// Check Constants count
	if len(chunk.Constants) != len(expectedConstants) {
		chunk.Disassemble()
		t.Fatalf("Constant count mismatch: expected %d, got %d", len(expectedConstants), len(chunk.Constants))
	}

	// Deeper constant check (optional, type and value)
	for i, expectedConst := range expectedConstants {
		// Allow nil placeholder in expectedConstants for simple count check
		if expectedConst == nil {
			continue
		}
		actualConst := chunk.Constants[i]
		// Add more specific checks if needed, e.g., comparing VarState values
		// For now, just check if it's a VarState if expectedConst was non-nil
		if _, ok := actualConst.(*VarState); !ok {
			t.Errorf("Constant %d is not *VarState, got %T", i, actualConst)
		}
		// Note: comparing complex expectedConst requires more setup
	}

	return chunk
}

// TODO: Add tests for Float, String literals
// TODO: Add test for compiling a File node (should be empty or handle contained decls later)

func TestCompile_LiteralInt(t *testing.T) {
	node := &LiteralExpr{NodeInfo: NodeInfo{StartPos: 1}, Kind: "INT", Value: "123"} // Add line number
	// Expected: Instruction{OpPushConst, 0, 1}, Instruction{OpReturn, 0, 1}
	// Constant[0]: VarState{Outcomes[int64]{1.0=>123}, Outcomes[Duration]{1.0=>0}}
	compileAndCheck(t, node,
		[]Instruction{
			{Opcode: OpPushConst, Operand: 0, Line: 1}, // Operand is const index
			{Opcode: OpReturn, Operand: 0, Line: 1},    // Operand ignored
		},
		[]interface{}{ /* placeholder for VarState */ nil},
	)
}

func TestCompile_LiteralBool(t *testing.T) {
	node := &LiteralExpr{NodeInfo: NodeInfo{StartPos: 1}, Kind: "BOOL", Value: "false"}
	// Expected: Instruction{OpPushConst, 0, 1}, Instruction{OpReturn, 0, 1}
	// Constant[0]: VarState{Outcomes[bool]{1.0=>false}, Outcomes[Duration]{1.0=>0}}
	compileAndCheck(t, node,
		[]Instruction{
			{Opcode: OpPushConst, Operand: 0, Line: 1},
			{Opcode: OpReturn, Operand: 0, Line: 1},
		},
		[]interface{}{ /* placeholder for VarState */ nil},
	)
}
