// sdl/dsl/compiler.go
package dsl

import (
	"fmt"

	"github.com/panyam/leetcoach/sdl/core"
)

// --- Handler IDs (Must match VM registration) ---
const (
	HandlerAdd = iota // ... (copy all IDs from vm.go) ...
	HandlerSubtract
	HandlerMultiply
	HandlerDivide
	HandlerNegate
	HandlerNot
	HandlerEqual
	HandlerNotEqual
	HandlerGreater
	HandlerGreaterEqual
	HandlerLess
	HandlerLessEqual
	HandlerAnd    // Logical AND
	HandlerOr     // Logical OR
	HandlerAndSeq // Sequential Composition
	HandlerDelay
	HandlerIf
	HandlerDurationMS // Added Handler ID
	HandlerDistribute
)

// Compiler transforms AST into bytecode Chunk.
type Compiler struct {
	chunk *Chunk
}

// NewCompiler creates a new compiler instance.
func NewCompiler() *Compiler {
	return &Compiler{}
}

// Compile entry point - adjust final OpReturn logic
func (c *Compiler) Compile(node Node) (*Chunk, error) {
	c.chunk = NewChunk(fmt.Sprintf("Script<%T>", node)) // Add basic name

	if err := c.compileNode(node); err != nil {
		return nil, err
	}

	// Always end script compilation with OpReturn *if needed*
	lastInstrIndex := -1
	if len(c.chunk.Instructions) > 0 {
		lastInstrIndex = len(c.chunk.Instructions) - 1
	}

	if lastInstrIndex < 0 || c.chunk.Instructions[lastInstrIndex].Opcode != OpReturn {
		// Add OpReturn if the last instruction wasn't already one.
		// Determine line number from the last instruction or the node itself.
		line := 0
		if lastInstrIndex >= 0 {
			line = c.chunk.Instructions[lastInstrIndex].Line
		} else if node != nil {
			line = node.End() // Fallback to node end position
		}
		c.emitOpCode(OpReturn, 0, line)
	}

	return c.chunk, nil
}

// compileNode dispatches compilation based on node type.
func (c *Compiler) compileNode(node Node) error {
	// line := node.Pos()

	switch n := node.(type) {
	// --- Expression Nodes ---
	case *LiteralExpr:
		return c.compileLiteral(n)
	case *IdentifierExpr:
		return c.compileIdentifier(n)
	case *BinaryExpr:
		return c.compileBinaryExpr(n)
	case *UnaryExpr:
		return c.compileUnaryExpr(n)
	case *AndExpr:
		return c.compileAndExpr(n) // Use specific func
	case *InternalCallExpr:
		return c.compileInternalCallExpr(n)
	case *CallExpr:
		return c.compileCallExpr(n) // Add this
	case *MemberAccessExpr:
		return c.compileMemberAccessExpr(n) // Add this

	// --- Statement Nodes ---
	case *BlockStmt:
		return c.compileBlockStmt(n)
	case *LetStmt:
		return c.compileLetStmt(n)
	case *ExprStmt:
		return c.compileExprStmt(n)
	case *ReturnStmt:
		return c.compileReturnStmt(n)
	case *DelayStmt:
		return c.compileDelayStmt(n)
	case *IfStmt:
		return c.compileIfStmt(n) // Add this

	// --- Top Level Nodes ---
	case *File:
		for _, decl := range n.Declarations {
			if err := c.compileNode(decl); err != nil {
				return err
			}
		}
		// Add implicit return after compiling file declarations? Maybe not needed.
		return nil

	// --- Other Declarations (Ignore for now, handle in separate compilation phases) ---
	case *ComponentDecl, *SystemDecl, *InstanceDecl, *AnalyzeDecl, *EnumDecl:
		// These define structure, not executable code directly in this pass
		return nil

	default:
		return fmt.Errorf("compiler cannot handle node type %T yet", node)
	}
}

// compileLiteral remains the same (emits OpPushConst)
func (c *Compiler) compileLiteral(expr *LiteralExpr) error {
	parsedVal, err := ParseLiteralValue(expr)
	if err != nil {
		return err
	}
	var valueOutcome any
	switch v := parsedVal.(type) {
	case int64:
		valueOutcome = (&core.Outcomes[int64]{}).Add(1.0, v)
	case float64:
		valueOutcome = (&core.Outcomes[float64]{}).Add(1.0, v)
	case string:
		valueOutcome = (&core.Outcomes[string]{}).Add(1.0, v)
	case bool:
		valueOutcome = (&core.Outcomes[bool]{}).Add(1.0, v)
	default:
		return fmt.Errorf("cannot create VarState for literal type %T", parsedVal)
	}
	state := &VarState{ValueOutcome: valueOutcome, LatencyOutcome: zeroLatencyOutcome()}
	constIndex := c.chunk.AddConstant(state)
	c.emitOpCode(OpPushConst, constIndex, expr.Pos())
	return nil
}

// compileIdentifier emits OpGetGlobal/OpGetLocal
func (c *Compiler) compileIdentifier(expr *IdentifierExpr) error {
	// TODO: Differentiate local/global lookup
	nameIndex := c.chunk.AddConstant(expr.Name)
	c.emitOpCode(OpGetGlobal, nameIndex, expr.Pos())
	return nil
}

// compileBinaryExpr compiles Left, Right, then OpCallFunc(Handler)
func (c *Compiler) compileBinaryExpr(expr *BinaryExpr) error {
	line := expr.Pos()
	if err := c.compileNode(expr.Left); err != nil {
		return err
	}
	if err := c.compileNode(expr.Right); err != nil {
		return err
	}

	var handlerID int
	switch expr.Operator {
	case "+":
		handlerID = HandlerAdd
	case "-":
		handlerID = HandlerSubtract
	case "*":
		handlerID = HandlerMultiply
	case "/":
		handlerID = HandlerDivide
	case "==":
		handlerID = HandlerEqual
	case "!=":
		handlerID = HandlerNotEqual
	case ">":
		handlerID = HandlerGreater
	case ">=":
		handlerID = HandlerGreaterEqual
	case "<":
		handlerID = HandlerLess
	case "<=":
		handlerID = HandlerLessEqual
	case "&&":
		handlerID = HandlerAnd
	case "||":
		handlerID = HandlerOr
	default:
		return fmt.Errorf("unsupported binary operator: %s", expr.Operator)
	}
	c.emitOpCode(OpCallFunc, handlerID, line)
	return nil
}

// compileUnaryExpr compiles Operand, then OpCallFunc(Handler)
func (c *Compiler) compileUnaryExpr(expr *UnaryExpr) error {
	line := expr.Pos()
	if err := c.compileNode(expr.Right); err != nil {
		return err
	}

	var handlerID int
	switch expr.Operator {
	case "-":
		handlerID = HandlerNegate
	case "!":
		handlerID = HandlerNot
	default:
		return fmt.Errorf("unsupported unary operator: %s", expr.Operator)
	}
	c.emitOpCode(OpCallFunc, handlerID, line)
	return nil
}

// compileAndExpr compiles Left, Right, then OpCallFunc(HandlerAndSeq)
func (c *Compiler) compileAndExpr(expr *AndExpr) error {
	line := expr.Pos()
	if err := c.compileNode(expr.Left); err != nil {
		return err
	}
	// Result of left is on stack
	// Compile right - its result will be on stack above left's
	if err := c.compileNode(expr.Right); err != nil {
		return err
	}
	// Call the sequence handler - it expects Left then Right state on stack
	c.emitOpCode(OpCallFunc, HandlerAndSeq, line)
	return nil
}

// compileInternalCallExpr compiles Args, then OpCallFunc(Handler)
func (c *Compiler) compileInternalCallExpr(expr *InternalCallExpr) error {
	line := expr.Pos()
	// Compile arguments first, pushing their results onto the stack
	// argCount := len(expr.Args)
	for _, arg := range expr.Args {
		if err := c.compileNode(arg); err != nil {
			return err
		}
	}
	// Map FuncName to HandlerID
	handlerID, ok := internalFuncNameMap[expr.FuncName]
	if !ok {
		return fmt.Errorf("unknown internal function: %s", expr.FuncName)
	}
	// Check arity? Handler function will do that.
	// Emit OpCallFunc - handler will pop 'argCount' arguments
	c.emitOpCode(OpCallFunc, handlerID, line)
	// Handler must push exactly one result state
	return nil
}

// Placeholder map - ideally shared with VM handler registration
var internalFuncNameMap = map[string]int{
	"DurationMS": HandlerDurationMS,
	// Add other internal function mappings here...
	// e.g., "GetDiskReadProfile": HandlerGetDiskReadProfile,
}

// compileCallExpr compiles Function, Args, then OpCall(ArgCount)
func (c *Compiler) compileCallExpr(expr *CallExpr) error {
	line := expr.Pos()
	// Compile the function expression itself (e.g., identifier, member access)
	// This should leave the function/method object (or identifier for it) on the stack.
	if err := c.compileNode(expr.Function); err != nil {
		return err
	}
	// Compile arguments, pushing results onto the stack
	argCount := len(expr.Args)
	for _, arg := range expr.Args {
		if err := c.compileNode(arg); err != nil {
			return err
		}
	}
	// Emit call instruction - VM handles finding the function and executing
	if argCount > 255 {
		return fmt.Errorf("too many arguments in call (>255)")
	}
	c.emitOpCode(OpCall, argCount, line)
	return nil
}

// compileMemberAccessExpr compiles Receiver, then OpGetMember(FieldNameIndex)?
// Or handle member access within OpCall if it's always a method call?
// Let's assume for now member access is primarily for method calls,
// so we just compile the receiver. The subsequent OpCall will use this.
func (c *Compiler) compileMemberAccessExpr(expr *MemberAccessExpr) error {
	// Compile the receiver expression. The subsequent OpCall will use this.
	if err := c.compileNode(expr.Receiver); err != nil {
		return err
	}
	// We might need to push the member name (as a constant) for OpCall later?
	// Or OpCall assumes the function object already knows its name.
	// Let's assume OpCall handles it for now.
	return nil
}

// compileBlockStmt compiles the sequence of statements.
func (c *Compiler) compileBlockStmt(block *BlockStmt) error {
	// TODO: Handle scope enter/exit for locals
	for i, stmt := range block.Statements {
		isLast := (i == len(block.Statements)-1)
		err := c.compileNode(stmt)
		if err != nil {
			return err
		}

		// Pop result of statement if it's not the last one and not a defining statement
		// Needed because ExprStmt leaves its value on stack, but we only want the *last* one for the block result.
		// Let/Define statements consume the value themselves. Return exits.
		// If/Distribute leave their combined result.
		if !isLast {
			switch stmt.(type) {
			case *ExprStmt, *IfStmt, *DistributeStmt: // Add other result-leaving stmts
				c.emitOpCode(OpPop, 0, stmt.End())
			case *LetStmt, *ReturnStmt:
				// These don't leave a value intended for the *next* sequential combination
			}
		}
	}
	// Result of the block is implicitly the result left on the stack by the last statement.
	// The VM handles the stack. No specific block end opcode needed unless managing scopes.
	return nil
}

// compileLetStmt compiles Value, then OpDefineGlobal/OpSetLocal
func (c *Compiler) compileLetStmt(stmt *LetStmt) error {
	line := stmt.Pos()
	// Compile the value expression first
	if err := c.compileNode(stmt.Value); err != nil {
		return err
	}
	// Now emit instruction to store the value (which is on top of stack)
	// TODO: Use OpSetLocal if in local scope
	nameIndex := c.chunk.AddConstant(stmt.Variable.Name)
	// Check index limits maybe? Assume ok for now.
	c.emitOpCode(OpDefineGlobal, nameIndex, line) // Assumes DefineGlobal consumes value from stack
	// LetStmt itself doesn't leave a value on the stack for sequence
	return nil
}

// compileExprStmt compiles Expression. Value is left on stack.
func (c *Compiler) compileExprStmt(stmt *ExprStmt) error {
	if err := c.compileNode(stmt.Expression); err != nil {
		return err
	}
	// Value is left on stack. If not last stmt in block, BlockStmt compile adds OpPop.
	return nil
}

// compileReturnStmt compiles ReturnValue, then OpReturn
func (c *Compiler) compileReturnStmt(stmt *ReturnStmt) error {
	line := stmt.Pos()
	if err := c.compileNode(stmt.ReturnValue); err != nil {
		return err
	}
	c.emitOpCode(OpReturn, 0, line)
	return nil
}

// compileDelayStmt compiles Duration, then OpCallFunc(HandlerDelay)
func (c *Compiler) compileDelayStmt(stmt *DelayStmt) error {
	line := stmt.Pos()
	if err := c.compileNode(stmt.Duration); err != nil {
		return err
	}
	// Delay needs the *current* implicit state before the duration.
	// This is tricky. Maybe Delay needs to be an expression form `delay(dur, expr)`?
	// For now, assume OpCallFunc(HandlerDelay) pops the duration and implicitly uses the
	// state *before* the duration was pushed. This requires careful VM handler logic.
	c.emitOpCode(OpCallFunc, HandlerDelay, line)
	// Handler must leave the combined state on the stack.
	return nil
}

// compileIfStmt compiles Condition, OpJumpIfFalsePop, Then block, OpJump (over else), Else block.
func (c *Compiler) compileIfStmt(stmt *IfStmt) error {
	line := stmt.Pos()
	// 1. Compile Condition
	if err := c.compileNode(stmt.Condition); err != nil {
		return err
	}

	// 2. Emit JumpIfFalsePop with placeholder offset for 'else'
	jumpToElse := c.emitJump(OpJumpIfFalsePop, line) // Returns index of jump instr

	// 3. Compile Then block
	// OpJumpIfFalsePop pops the condition value implicitly.
	// The result of the Then block will be left on the stack.
	if err := c.compileNode(stmt.Then); err != nil {
		return err
	}

	// 4. Emit Jump with placeholder offset to skip 'else'
	jumpToEnd := c.emitJump(OpJump, stmt.Then.End()) // Jump after then block

	// 5. Patch the jump-to-else offset
	elseTargetOffset := len(c.chunk.Instructions) - (jumpToElse + 1) // Offset from *next* instruction
	c.patchJump(jumpToElse, elseTargetOffset)

	// 6. Compile Else block (if exists)
	if stmt.Else != nil {
		// If we reached here (condition was false), the stack should just contain
		// the state *before* the condition was evaluated.
		// The 'else' block execution will combine with that implicit state.
		// The result of the Else block will replace the pre-condition state on the stack.
		if err := c.compileNode(stmt.Else); err != nil {
			return err
		}
	} else {
		// If no 'else' block, what should be the result if condition was false?
		// Push the identity state? Or just leave stack as it was after condition pop?
		// Let's push identity for now. VM side needs to handle this case in `combine`.
		c.emitConstant(createIdentityState(), stmt.End())
	}

	// 7. Patch the jump-to-end offset
	endTargetOffset := len(c.chunk.Instructions) - (jumpToEnd + 1)
	c.patchJump(jumpToEnd, endTargetOffset)
	// The final result (either from Then or Else/Identity) is left on the stack.
	return nil
}

// --- Bytecode Emission Helpers ---

// emitOpCode adds an instruction with an opcode and operand.
func (c *Compiler) emitOpCode(op OpCode, operand int, line int) {
	c.chunk.Write(op, operand, line)
}

// emitConstant adds the value to the constant pool and emits OpPushConst + index.
func (c *Compiler) emitConstant(value interface{}, line int) {
	index := c.chunk.AddConstant(value)
	if index > 65535 { // Example limit for 16-bit operand
		panic("Constant pool limit (65536) exceeded")
	}
	// TODO: Emit OpPushConst with 1 or 2 byte operand based on index size
	if index > 255 {
		panic("OpPushConst with 2-byte operand not implemented yet")
	}
	c.emitOpCode(OpPushConst, index, line)
}

// emitJump writes a jump instruction with a placeholder offset. Returns index to patch.
func (c *Compiler) emitJump(op OpCode, line int) int {
	operandPlaceholder := 9999 // Placeholder offset
	// TODO: Emit jump operand (e.g., 2 bytes)
	if op != OpJump && op != OpJumpIfFalsePop {
		panic(fmt.Sprintf("emitJump called with non-jump opcode %s", op))
	}
	offset := c.chunk.Write(op, operandPlaceholder, line) // Write instruction, return its index
	return offset

}

// patchJump updates the operand of a previously written jump instruction.
func (c *Compiler) patchJump(jumpInstructionIndex int, targetOffset int) {
	// TODO: Update jump operand (e.g., 2 bytes)
	if targetOffset < 0 {
		// Should not happen if jumps are always forward for if/else
		panic(fmt.Sprintf("Negative jump offset calculated: %d", targetOffset))
	}
	if targetOffset > 65535 { // Check if offset fits in operand (e.g., 16-bit)
		panic(fmt.Sprintf("Jump offset %d too large (> 65535)", targetOffset))
	}
	if jumpInstructionIndex < 0 || jumpInstructionIndex >= len(c.chunk.Instructions) {
		panic(fmt.Sprintf("Invalid jump instruction index %d to patch", jumpInstructionIndex))
	}
	c.chunk.Instructions[jumpInstructionIndex].Operand = targetOffset
}
