# SDL DSL Package Summary (`sdl/dsl`)

**Purpose:**

This package implements the execution engine for the System Design Language (DSL). It takes compiled bytecode (derived from an Abstract Syntax Tree), executes it using a Virtual Machine (VM), orchestrates calls to `sdl/core` and `sdl/components`, and produces analysis results. It aims to provide a high-level, declarative interface, abstracting direct manipulation of probabilistic outcomes.

**Target Execution Model (Model V4 - Implicit Dual Track):**

*   **Implicit Outcomes:** The DSL syntax deals with functional types (`bool`, `enum`, `int`), not `Outcomes` directly.
*   **VM Dual Track:** The VM implicitly tracks two distributions per variable/expression result via `VarState { ValueOutcome any /* *Outcomes[V] */, LatencyOutcome any /* *Outcomes[Duration] */ }`.
*   **Discrete Values:** `V` in the value track is restricted to registered discrete types (bool, enums).
*   **Explicit Delay:** Latency is added *only* by leaf operations or explicit `delay` statements (via `OpCallFunc HandlerDelay`).
*   **Probabilistic Branching:** `distribute` statement (future) handles probabilistic control flow. `if` splits based on boolean value outcomes (via `OpCallFunc HandlerIf`).

**Hybrid Bytecode Execution Strategy:**

*   **Bytecode:** Simple expressions and statements are compiled into bytecode instructions (`bytecode.go`).
*   **VM Go Helpers:** Complex probabilistic control flow (`if`, `distribute`) and potentially other operations (`AndSeq`, `Delay`) are handled by dedicated Go functions (`vm.vm_handle_*`) within the VM, triggered by specific `OpCallFunc` bytecode instructions. These Go helpers orchestrate context splitting, recursive calls (potentially back to the bytecode loop), and result combination using `sdl/core` functions.

**Key Concepts & Components:**

1.  **Abstract Syntax Tree (AST) (`ast.go`):**
    *   Defines Go structs representing the parsed DSL (aligned with Model V4 grammar).
    *   Serves as the input to the Compiler.

2.  **Bytecode (`bytecode.go`):**
    *   Defines `OpCode` constants for VM instructions (simplified set using `OpCallFunc`).
    *   Defines `Instruction` struct (Opcode + Operand).
    *   Defines `Chunk` struct to hold instructions and constants.
    *   Includes disassembler for debugging.

3.  **Compiler (`compiler.go`):**
    *   Transforms AST into `Chunk` (bytecode + constants).
    *   Manages constant pool (currently storing `*VarState` for literals).
    *   Emits basic opcodes (`OpPushConst`, `OpDefineGlobal`, `OpGetGlobal`, `OpPop`, `OpReturn`) and `OpCallFunc` for operations/control flow.
    *   Handles basic block compilation, `LetStmt`, `ExprStmt`, `ReturnStmt`, `IfStmt` (emitting jumps and placeholders). *(Partially Implemented)*

4.  **Virtual Machine (VM) (`vm.go`):**
    *   Core execution engine running bytecode instructions from a `Chunk`.
    *   Manages execution state: `ip` (instruction pointer), `stack` (holding `*VarState`), `globals` map.
    *   **Dual Track State:** `VarState` struct used on stack and in globals.
    *   **Execution Loop (`ExecuteChunk`):** Fetches, decodes, executes bytecode instructions.
    *   **Internal Handlers:** Executes Go helper functions (`vm.vm_handle_*`) triggered by `OpCallFunc`. Registered via `RegisterInternalHandler`. Contains stubs/basic implementations for several handlers (Add, AndSeq, Delay, If, Distribute).
    *   **Composition:** Uses `combineOutcomesAndReduce` helper for sequential combination (`AndSeq` handler). Currently uses "Take Last Value" simplification.
    *   **Reduction:** Uses `reduceStateIfNeeded` (currently only checks latency track size and applies reduction via configured `durationReducer`).

5.  **Environment (`environment.go`):**
    *   Handles lexical scoping (though currently only globals are implemented in VM). Stores `*VarState`.

6.  **Driver (`driver.go`):**
    *   Provides `RunDSL` to compile and execute a system AST. *(Note: Needs update for bytecode)*
    *   Initializes VM, registers built-in components/functions.
    *   Instantiates system components based on AST.
    *   Executes `analyze` targets by calling VM.
    *   Calculates metrics using `core` functions and returns results in `AnalysisResultWrapper`.

**Current Status:**

*   AST (`ast.go`) reasonably defined for Model V4.
*   Bytecode structures (`Chunk`, `Instruction`, `OpCode`) defined. Simplified opcode set using `OpCallFunc`. Disassembler implemented.
*   Compiler (`compiler.go`) implemented for basic constructs: Literals, Globals (`Let`/`Get`), Binary/Unary Ops (emitting `OpCallFunc`), `AndExpr` (`OpCallFunc`), `ExprStmt` (`OpPop`), `ReturnStmt` (`OpReturn`), `IfStmt` (emitting `OpJumpIfFalsePop`, `OpJump`).
*   VM (`vm.go`) has core loop (`ExecuteChunk`), stack (`*VarState`), globals. Executes `OpPushConst`, `OpDefineGlobal`, `OpGetGlobal`, `OpSetGlobal`, `OpPop`, `OpJumpIfFalsePop`, `OpJump`, `OpReturn`, `OpCallFunc`.
*   VM includes `vm_handle_*` stubs/basic implementations for `Add`, `AndSeq`, `Delay`, `If`, `Distribute`. `vm_handle_add` has basic deterministic logic.
*   Driver (`driver.go`) needs update to use Compiler + `ExecuteChunk`.
*   **Key Missing Pieces:** Full implementation of `vm_handle_*` functions for complex logic (especially `If`, `Distribute`, probabilistic binary ops), local variables/scoping, method/function calls (`OpCall`), parser. `combineOutcomesAndReduce` value propagation is simplified.

**Next Steps:**

*   Implement logic within `vm_handle_*` handlers (Arithmetic, Boolean, `Delay`, `AndSeq`).
*   Implement `core.SplitLatenciesBasedOnValue` and integrate into `vm_handle_if`.
*   Implement `vm_handle_distribute`.
*   Implement locals (`OpGetLocal`, `OpSetLocal`) and call frame management (`OpCall`, `OpReturn`).
*   Update Driver to use compiler + `ExecuteChunk`.
*   Implement Parser.
