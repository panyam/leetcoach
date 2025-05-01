package dsl

import (
	"fmt"
	"testing"

	"github.com/panyam/leetcoach/sdl/core"
)

// Helper function to run bytecode and check the final result state
func runCodeAndCheckResult(t *testing.T, chunk *Chunk, expectedVal interface{}, expectedLatMs float64) {
	t.Helper()
	vm := NewVM(15) // Use larger max buckets for tests unless checking reduction
	finalState, err := vm.ExecuteChunk(chunk)
	if err != nil {
		chunk.Disassemble() // Disassemble on error
		t.Fatalf("VM execution failed: %v", err)
	}

	if finalState == nil {
		t.Fatalf("VM execution returned nil state unexpectedly")
	}

	// --- Check Value Outcome ---
	var actualVal interface{}
	var ok bool
	// Define expected outcome type string for better error messages
	expectedTypeStr := "nil"

	switch expectedVal.(type) {
	case int64:
		expectedTypeStr = "*core.Outcomes[int64]"
		if vo, typeOK := finalState.ValueOutcome.(*core.Outcomes[int64]); typeOK {
			actualVal, ok = vo.GetValue()
			if !ok {
				t.Errorf("Expected final value outcome to be deterministic int64")
			}
		} else {
			t.Errorf("Value type mismatch: expected %s, got %T", expectedTypeStr, finalState.ValueOutcome)
		}
	case float64:
		expectedTypeStr = "*core.Outcomes[float64]"
		if vo, typeOK := finalState.ValueOutcome.(*core.Outcomes[float64]); typeOK {
			actualVal, ok = vo.GetValue()
			if !ok {
				t.Errorf("Expected final value outcome to be deterministic float64")
			}
		} else {
			t.Errorf("Value type mismatch: expected %s, got %T", expectedTypeStr, finalState.ValueOutcome)
		}
	case string:
		expectedTypeStr = "*core.Outcomes[string]"
		if vo, typeOK := finalState.ValueOutcome.(*core.Outcomes[string]); typeOK {
			actualVal, ok = vo.GetValue()
			if !ok {
				t.Errorf("Expected final value outcome to be deterministic string")
			}
		} else {
			t.Errorf("Value type mismatch: expected %s, got %T", expectedTypeStr, finalState.ValueOutcome)
		}
	case bool:
		expectedTypeStr = "*core.Outcomes[bool]"
		if vo, typeOK := finalState.ValueOutcome.(*core.Outcomes[bool]); typeOK {
			actualVal, ok = vo.GetValue()
			if !ok {
				t.Errorf("Expected final value outcome to be deterministic bool")
			}
		} else {
			t.Errorf("Value type mismatch: expected %s, got %T", expectedTypeStr, finalState.ValueOutcome)
		}
	default:
		t.Fatalf("Unsupported expected value type in test: %T", expectedVal)
	}

	// Check if types matched and value extraction succeeded before comparing
	if !t.Failed() && actualVal != expectedVal {
		t.Errorf("Value mismatch: expected %v (%s), got %v ", expectedVal, expectedTypeStr, actualVal)
	}

	// --- Check Latency Outcome ---
	latOutcome, okL := finalState.LatencyOutcome.(*core.Outcomes[core.Duration])
	if !okL {
		t.Fatalf("LatencyOutcome is not *core.Outcomes[core.Duration], got %T", finalState.LatencyOutcome)
	}
	actualLat, okLatDet := latOutcome.GetValue()
	if !okLatDet {
		t.Errorf("Expected final latency outcome to be deterministic")
	}
	expectedLat := core.Millis(expectedLatMs)
	// Increase tolerance slightly for potential float inaccuracies in complex tests later
	if !core.ApproxEq(actualLat, expectedLat, 1e-9) {
		t.Errorf("Latency mismatch: expected %.9f, got %.9f", expectedLat, actualLat)
	}
}

func TestExecuteChunk_ConstantReturn(t *testing.T) {
	compiler := NewCompiler()
	chunk, err := compiler.Compile(&LiteralExpr{Kind: "INT", Value: "123"})
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}
	chunk.Disassemble()
	runCodeAndCheckResult(t, chunk, int64(123), 0.0)
}

func TestExecuteChunk_ConstantBoolReturn(t *testing.T) {
	compiler := NewCompiler()
	chunk, err := compiler.Compile(&LiteralExpr{Kind: "BOOL", Value: "true"})
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}
	chunk.Disassemble()
	runCodeAndCheckResult(t, chunk, true, 0.0)
}

func TestExecuteChunk_BinaryAdd(t *testing.T) {
	compiler := NewCompiler()
	// Compile: 10 + 20
	node := &BinaryExpr{
		Left:     &LiteralExpr{Kind: "INT", Value: "10"},
		Operator: "+",
		Right:    &LiteralExpr{Kind: "INT", Value: "20"},
	}
	chunk, err := compiler.Compile(node)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}
	chunk.Disassemble()

	// Need to implement the vm_handle_add function in vm.go first!
	// This test will fail until vm_handle_add is implemented.
	t.Skip("Skipping Binary Add test until vm_handle_add handler is implemented") // Skip for now

	// Expected result: Value=30, Latency=0
	// runCodeAndCheckResult(t, chunk, int64(30), 0.0)
}

func TestExecuteChunk_LetAndGetGlobal(t *testing.T) {
	compiler := NewCompiler()
	// Compile: let x = 5; x;
	block := &BlockStmt{
		Statements: []Stmt{
			&LetStmt{
				Variable: &IdentifierExpr{Name: "x"},
				Value:    &LiteralExpr{Kind: "INT", Value: "5"},
			},
			&ExprStmt{ // Use ExprStmt to push x onto stack before final return
				Expression: &IdentifierExpr{Name: "x"},
			},
		},
	}
	chunk, err := compiler.Compile(block) // Compiler adds OpReturn implicitly for block
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}
	chunk.Disassemble()

	// Run and check
	runCodeAndCheckResult(t, chunk, int64(5), 0.0)
}

func TestExecuteChunk_SimpleSequence(t *testing.T) {
	compiler := NewCompiler()
	// Compile: { delay 10ms; delay 20ms; }
	// Use internal calls to create duration states for delay
	dur10 := internalCall("DurationMS", litFloat(10.0))
	dur20 := internalCall("DurationMS", litFloat(20.0))

	block := &BlockStmt{
		Statements: []Stmt{
			// Need to register DurationMS handler in VM tests
			&DelayStmt{Duration: dur10},
			&DelayStmt{Duration: dur20},
		},
	}
	chunk, err := compiler.Compile(block) // Implicit return of final state
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}
	chunk.Disassemble()

	// Register DurationMS handler for the test VM
	vm := NewVM(15)
	vm.RegisterInternalHandler(0, // Assume DurationMS gets ID 0
		func(vm *VM, args []*VarState) (*VarState, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("DurationMS needs 1 arg")
			}
			msState := args[0]
			if msState == nil {
				return nil, fmt.Errorf("DurationMS arg must be VarState")
			}
			msValF, ok := msState.ValueOutcome.(*core.Outcomes[float64]).GetValue()
			if !ok {
				return nil, fmt.Errorf("DurationMS arg must be deterministic float")
			}
			durOutcome := (&core.Outcomes[core.Duration]{}).Add(1.0, core.Millis(msValF))
			state := &VarState{ValueOutcome: durOutcome, LatencyOutcome: zeroLatencyOutcome()}
			if err := vm.push(state); err != nil {
				return nil, err
			} // Push result for delay
			return nil, nil // Success
		})

	// This test needs HandlerDelay and HandlerAndSeq implemented
	t.Skip("Skipping Simple Sequence test until Delay/AndSeq handlers are implemented")

	// Expected result: Value=true (from identity), Latency=30ms
	// finalState, err := vm.ExecuteChunk(chunk)
	// ... assertions ...
	// runCodeAndCheckResult(t, chunk, true, 30.0)
}
