package dsl

import (
	"fmt"

	"github.com/panyam/leetcoach/sdl/core" // For wrapping literals
)

// Compiler transforms AST into bytecode Chunk.
type Compiler struct {
	chunk *Chunk // The chunk being built
	// Add scope/symbol table management later if needed for locals
}

// NewCompiler creates a new compiler instance.
func NewCompiler() *Compiler {
	return &Compiler{}
}

// Placeholder map - ideally shared with VM handler registration
var internalFuncNameMap = map[string]int{
	// Add mappings here... e.g. "DurationMS": HandlerDurationMS,
}

// Compile is the main entry point for compiling an AST node.
func (c *Compiler) Compile(node Node) (*Chunk, error) {
	c.chunk = NewChunk(fmt.Sprintf("Script<%T>", node)) // Add basic name

	if err := c.compileNode(node); err != nil {
		return nil, err
	}

	// Always end script compilation with OpReturn
	c.emitOpCode(OpReturn, 0, node.End()) // Operand ignored for OpReturn

	return c.chunk, nil
}

// compileNode dispatches compilation based on node type.
func (c *Compiler) compileNode(node Node) error {
	// line := node.Pos()
	switch n := node.(type) {
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
		return nil

	// --- Other Declarations (Ignore for now, handle in separate compilation phases) ---
	case *ComponentDecl, *SystemDecl, *InstanceDecl, *AnalyzeDecl, *EnumDecl:
		// These define structure, not executable code directly in this pass
		return nil
	// Add cases for ComponentDecl, SystemDecl etc. later

	default:
		return fmt.Errorf("compiler not implemented for node type %T", node)
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

// compileIdentifier emits OpGetGlobal/OpGetLocal
func (c *Compiler) compileIdentifier(expr *IdentifierExpr) error {
	// TODO: Differentiate local/global lookup
	nameIndex := c.chunk.AddConstant(expr.Name)
	c.emitOpCode(OpGetGlobal, nameIndex, expr.Pos())
	return nil
}

// compileAndExpr compiles Left, Right, then OpCallFunc(HandlerAndSeq)
func (c *Compiler) compileAndExpr(expr *AndExpr) error {
	line := expr.Pos()
	if err := c.compileNode(expr.Left); err != nil {
		return err
	}
	if err := c.compileNode(expr.Right); err != nil {
		return err
	}
	c.emitOpCode(OpCallFunc, HandlerAndSeq, line)
	return nil
}

// compileInternalCallExpr compiles Args, then OpCallFunc(Handler)
func (c *Compiler) compileInternalCallExpr(expr *InternalCallExpr) error {
	line := expr.Pos()
	// Compile arguments first
	for _, arg := range expr.Args {
		if err := c.compileNode(arg); err != nil {
			return err
		}
	}
	// TODO: Map FuncName to HandlerID (needs registry in compiler?)
	// For now, assume a direct mapping exists or use a placeholder
	handlerID, ok := internalFuncNameMap[expr.FuncName] // Example lookup
	if !ok {
		return fmt.Errorf("unknown internal function: %s", expr.FuncName)
	}
	c.emitOpCode(OpCallFunc, handlerID, line)
	return nil
}

// compileCallExpr compiles Function, Args, then OpCall(ArgCount)
func (c *Compiler) compileCallExpr(expr *CallExpr) error {
	line := expr.Pos()
	// Compile the function expression itself (e.g., identifier, member access)
	if err := c.compileNode(expr.Function); err != nil {
		return err
	}
	// Compile arguments
	argCount := len(expr.Args)
	for _, arg := range expr.Args {
		if err := c.compileNode(arg); err != nil {
			return err
		}
	}
	// Emit call instruction
	if argCount > 255 {
		return fmt.Errorf("too many arguments in call (>255)")
	}
	c.emitOpCode(OpCall, argCount, line)
	return nil
}

// compileMemberAccessExpr compiles Receiver, then OpGetMember(FieldNameIndex)?
// Or handle member access within OpCall if it's always a method call?
// Let's assume for now member access is primarily for method calls,
// so we just compile the receiver. The OpCall logic will handle the '.Member'.
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
	for _, stmt := range block.Statements {
		if err := c.compileNode(stmt); err != nil {
			return err
		}
	}
	// Result of the block is implicitly the result of the last statement?
	// The VM handles the stack. No specific block end opcode needed unless managing scopes.
	return nil
}

// compileLetStmt compiles Value, then OpDefineGlobal/OpSetLocal
func (c *Compiler) compileLetStmt(stmt *LetStmt) error {
	line := stmt.Pos()
	if err := c.compileNode(stmt.Value); err != nil {
		return err
	}
	// TODO: Use OpSetLocal if in local scope
	nameIndex := c.chunk.AddConstant(stmt.Variable.Name)
	if nameIndex > 65535 {
		return fmt.Errorf("constant pool limit exceeded")
	}
	c.emitOpCode(OpDefineGlobal, nameIndex, line) // Assumes DefineGlobal consumes value
	return nil
}

// compileExprStmt compiles Expression, then OpPop
func (c *Compiler) compileExprStmt(stmt *ExprStmt) error {
	line := stmt.Pos()
	if err := c.compileNode(stmt.Expression); err != nil {
		return err
	}
	c.emitOpCode(OpPop, 0, line) // Pop the result
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
	c.emitOpCode(OpCallFunc, HandlerDelay, line)
	// Delay result is implicitly popped by the handler? Or should we pop here?
	// Let handler manage stack. Delay modifies the *next* sequence combine.
	// No, Delay should combine with current state. Let handler handle stack.
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
	// Pop the condition result before executing 'then'? OpJumpIfFalsePop does this.
	if err := c.compileNode(stmt.Then); err != nil {
		return err
	}

	// 4. Emit Jump with placeholder offset to skip 'else'
	jumpToEnd := c.emitJump(OpJump, stmt.Then.End()) // Jump after then block

	// 5. Patch the jump-to-else offset
	// Offset = Current position - position after jump instr
	elseTargetOffset := len(c.chunk.Instructions) - (jumpToElse + 1)
	c.patchJump(jumpToElse, elseTargetOffset)

	// 6. Compile Else block (if exists)
	if stmt.Else != nil {
		// Pop the result of the 'then' block if it wasn't used? No, If should leave result on stack.
		if err := c.compileNode(stmt.Else); err != nil {
			return err
		}
	}
	// If no else block, the implicit result is the condition value? No, if should evaluate to something.
	// Maybe push a default value if no else? Or rely on OpJumpIfFalsePop having popped condition.
	// Current approach: If condition is false, stack has condition popped, then jump executes.

	// 7. Patch the jump-to-end offset
	endTargetOffset := len(c.chunk.Instructions) - (jumpToEnd + 1)
	c.patchJump(jumpToEnd, endTargetOffset)

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
	c.emitOpCode(OpPushConst, index, line)
}

// emitJump writes a jump instruction with a placeholder offset. Returns index to patch.
func (c *Compiler) emitJump(op OpCode, line int) int {
	operandPlaceholder := 9999                         // Placeholder offset
	return c.chunk.Write(op, operandPlaceholder, line) // Write instruction, return its index
}

// patchJump updates the operand of a previously written jump instruction.
func (c *Compiler) patchJump(jumpInstructionIndex int, targetOffset int) {
	if targetOffset > 65535 { // Check if offset fits in operand (e.g., 16-bit)
		panic("Jump offset too large")
	}
	if jumpInstructionIndex < 0 || jumpInstructionIndex >= len(c.chunk.Instructions) {
		panic(fmt.Sprintf("Invalid jump instruction index %d to patch", jumpInstructionIndex))
	}
	c.chunk.Instructions[jumpInstructionIndex].Operand = targetOffset
}
