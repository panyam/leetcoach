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

	// Expected result: Value=30, Latency=0
	runCodeAndCheckResult(t, chunk, int64(30), 0.0)
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
	vm.RegisterInternalHandler(HandlerDurationMS,
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

func TestExecuteChunk_Locals(t *testing.T) {
	compiler := NewCompiler()
	// Compile: { let localA = 10; let localB = 20; localA + localB; }
	// Note: Relies on Add handler being implemented later.
	block := &BlockStmt{NodeInfo: NodeInfo{StartPos: 1},
		Statements: []Stmt{
			&LetStmt{NodeInfo: NodeInfo{StartPos: 1}, // Line 1
				Variable: &IdentifierExpr{Name: "localA"},
				Value:    &LiteralExpr{Kind: "INT", Value: "10"},
			},
			&LetStmt{NodeInfo: NodeInfo{StartPos: 2}, // Line 2
				Variable: &IdentifierExpr{Name: "localB"},
				Value:    &LiteralExpr{Kind: "INT", Value: "20"},
			},
			&ExprStmt{NodeInfo: NodeInfo{StartPos: 3}, // Line 3 - result of block
				Expression: &BinaryExpr{
					Left:     &IdentifierExpr{Name: "localA"},
					Operator: "+",
					Right:    &IdentifierExpr{Name: "localB"},
				},
			},
		},
	}
	chunk, err := compiler.Compile(block)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}
	chunk.Disassemble()

	// Expected Bytecode (Approximate - indices might vary slightly):
	// 0000    1 OpPushConst      0 'State{V:10, L:0ms}'  // Value for localA
	// 0001    1 OpPop                              // Let doesn't leave value (removed this pop implicitly)
	// 0002    2 OpPushConst      1 'State{V:20, L:0ms}'  // Value for localB
	// 0003    2 OpPop                              // Let doesn't leave value (removed this pop implicitly)
	// --- After LetStmt changes: ---
	// 0000    1 OpPushConst      0 'State{V:10, L:0ms}' // Value for localA (remains on stack as local slot 0)
	// 0001    2 OpPushConst      1 'State{V:20, L:0ms}' // Value for localB (remains on stack as local slot 1)
	// 0002    3 OpGetLocal       0                  // Push localA (slot 0)
	// 0003    3 OpGetLocal       1                  // Push localB (slot 1)
	// 0004    3 OpCallFunc       0 (HandlerAdd)     // Pop A, Pop B, Add, Push Result
	// 0005    3 OpPop                              // Pop localB from scope exit
	// 0006    3 OpPop                              // Pop localA from scope exit
	// 0007    3 OpReturn                           // Return result of Add

	// Need HandlerAdd implemented
	t.Skip("Skipping Locals test until Add handler and scope OpPop emission are fully verified")

	// Run and check (Expected result: Value=30, Latency=0)
	// runCodeAndCheckResult(t, chunk, int64(30), 0.0)
}

// --- OpCall Test ---
type MockCallable struct {
	Name string
}

// Simple method returning a fixed VarState
func (mc *MockCallable) Echo(arg *VarState) (*VarState, error) {
	fmt.Printf("MockCallable.Echo called with arg: %s\n", formatConstant(arg))
	// Return a new state: Value=true, Latency=5ms + arg latency
	resLat := (&core.Outcomes[core.Duration]{}).Add(1.0, core.Millis(5))
	combinedLatState, _ := (&VM{}).combineOutcomesAndReduce(arg, &VarState{LatencyOutcome: resLat}) // Use combine helper logic

	return &VarState{
		ValueOutcome:   (&core.Outcomes[bool]{}).Add(1.0, true),
		LatencyOutcome: combinedLatState.LatencyOutcome,
	}, nil
}

func TestExecuteChunk_OpCall(t *testing.T) {
	compiler := NewCompiler()
	// Compile: { let mc = MockCallable{}; mc.Echo(10); } // Ignore 'let' for now
	node := &CallExpr{
		Function: &MemberAccessExpr{Receiver: ident("mc"), Member: ident("Echo")}, // Assume 'mc' holds MockCallable
		Args:     []Expr{litInt(10)},
	}
	chunk, _ := compiler.Compile(node)
	chunk.Disassemble()

	t.Skip("Skipping OpCall test until VM execution logic is implemented")
}

func TestExecuteChunk_If_SimpleBool(t *testing.T) {
	compiler := NewCompiler()
	// AST for:
	// {
	//   cond = Internal.GetBool(0.8, DurationMS(1.0), DurationMS(2.0));
	//   res = 0; // This 'let' is unused in the final result path, remove for test clarity
	//   if cond {
	//     Internal.GetAccess(1.0, DurationMS(10.0), DurationMS(0.0)); // Then
	//   } else {
	//     Internal.GetAccess(0.0, DurationMS(0.0), DurationMS(5.0));  // Else
	//   }
	//   // Implicit return of the if result
	// }
	block := &BlockStmt{NodeInfo: NodeInfo{StartPos: 1},
		Statements: []Stmt{
			&LetStmt{NodeInfo: NodeInfo{StartPos: 1}, Variable: ident("cond"), Value: internalCall("GetBool", litFloat(0.8),
				internalCall("DurationMS", litFloat(1.0)), internalCall("DurationMS", litFloat(2.0)))},
			// &LetStmt{Variable: ident("res").(*IdentifierExpr), Value: litInt(0)}, // Remove unused let
			&IfStmt{NodeInfo: NodeInfo{StartPos: 2},
				Condition: ident("cond"), // Use variable holding bool outcome
				Then: &BlockStmt{NodeInfo: NodeInfo{StartPos: 3}, Statements: []Stmt{
					// Return value of block is this expression
					&ExprStmt{NodeInfo: NodeInfo{StartPos: 3}, Expression: internalCall("GetAccess", litFloat(1.0),
						internalCall("DurationMS", litFloat(10.0)), internalCall("DurationMS", litFloat(0.0)))},
				}},
				Else: &BlockStmt{NodeInfo: NodeInfo{StartPos: 5}, Statements: []Stmt{
					// Return value of block is this expression
					&ExprStmt{NodeInfo: NodeInfo{StartPos: 5}, Expression: internalCall("GetAccess", litFloat(0.0),
						internalCall("DurationMS", litFloat(0.0)), internalCall("DurationMS", litFloat(5.0)))},
				}},
			},
			// Implicit Return at end of block
		},
	}

	chunk, err := compiler.Compile(block)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}
	chunk.Disassemble()

	// Need SplitLatenciesBasedOnValue implemented in core!
	t.Skip("Skipping If SimpleBool test until core.SplitLatenciesBasedOnValue is verified")

	vm := NewVM(15)
	// registerMockInternalFuncs(vm) // Ensure GetBool, GetAccess, DurationMS are registered

	finalState, err := vm.ExecuteChunk(chunk)
	if err != nil {
		t.Fatalf("VM execution failed: %v", err)
	}

	// Expected results (as calculated before)
	// Value Avail = 0.8
	// Latency Mean = 10.2ms
	valOutcome, okV := finalState.ValueOutcome.(*core.Outcomes[core.AccessResult])
	latOutcome, okL := finalState.LatencyOutcome.(*core.Outcomes[core.Duration])
	if !okV || !okL {
		t.Fatalf("Final state has wrong types: V=%T, L=%T", finalState.ValueOutcome, finalState.LatencyOutcome)
	}

	actualAvail := core.Availability(valOutcome)
	expectedAvail := 0.8
	if !core.ApproxEq(actualAvail, expectedAvail, 1e-9) {
		t.Errorf("Expected final Availability %.4f, got %.4f", expectedAvail, actualAvail)
	}

	actualMeanDur := meanDurationLatency(latOutcome)
	expectedMeanDur := core.Millis(10.2)
	if !core.ApproxEq(actualMeanDur, expectedMeanDur, core.Millis(0.01)) {
		t.Errorf("Expected final Mean Duration %.6fs, got %.6fs", expectedMeanDur, actualMeanDur)
	}
	t.Logf("IfStmt Bool Test: Avail=%.4f, MeanDur=%.6f", actualAvail, actualMeanDur)

}
