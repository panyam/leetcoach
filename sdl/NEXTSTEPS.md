# SDL Project - Next Steps

**Overall Goal:** Implement the core functionality of the DSL execution engine (VM + Bytecode Compiler) based on the **Hybrid execution model** and targeting the **Model V4 DSL syntax** (Implicit Dual Track, Discrete Values, Explicit Delay).

**Phase 1: Core VM Refactoring & Primitives (Partially Complete -> Continue)**

*Goal: Ensure the VM correctly handles the dual-track state, basic expressions, sequence, and the core Model V4 primitives (`delay`, `distribute`).*

1.  **Implement `core.SplitLatenciesBasedOnValue`:** **(Critical Blocker)** Implement this helper in `sdl/core` to correctly split `Outcomes[Duration]` based on a predicate applied to a synchronized `Outcomes[V]` (using iteration, not reflection if possible by adding helpers to Outcomes). Add thorough tests.
2.  **Test `evalIfStmt` / `evalBlockStmt`:** Complete testing in `dsl/vm_test.go` now that prerequisites are clearer. Verify dual-track combination and splitting logic.
3.  **Implement `evalDelayStmt`:** Add `DelayStmt` to `ast.go`. Implement the evaluator in `dsl/eval_stmt.go`. It should:
    *   Evaluate the duration expression.
    *   Pop the current state from the stack (if block sequence).
    *   Combine latencies (`And`).
    *   Push new state (original value, combined latency).
    *   Add tests.
4.  **Implement `evalDistributeStmt`:** Add `DistributeStmt`, `DistributeCase`, `DefaultCase` to `ast.go`. Implement the evaluator (`evalDistributeStmt`) in `dsl/dsl_controlflow.go` (new file suggested).
    *   Evaluate probabilities.
    *   Split incoming context (`value` + `latency`) based on probabilities.
    *   Recursively call `evalBlockStmt` for each branch with its split context.
    *   Combine results from branches using weighted `Append` logic (needs careful implementation for both tracks).
    *   Push the final combined state.
    *   Add tests for probabilistic branching and result combination.

**Phase 2: Function/Method Calls & Component Interaction**

*Goal: Enable calling component methods and user-defined operations within the DSL.*

1.  **Refactor Component Methods / Add VM Wrappers:** Decide on strategy:
    *   Option A: Modify Go methods in `sdl/components` (e.g., `Disk.Read`) to return `(valueOutcome any, latencyOutcome any, error)`. Requires changes across `components`.
    *   Option B: Keep component methods returning `*core.Outcomes[V]`. Create wrappers *within the VM/Driver* during component registration that translate the `*Outcomes[V]` result into the dual-track `*VarState` expected by the VM. This isolates changes. (Recommendation: Option B seems less intrusive).
2.  **Implement `evalCallExpr`:**
    *   Handle resolving function/method names (global built-ins, component methods).
    *   Evaluate arguments (must be deterministic).
    *   Invoke the Go method/function (potentially via the VM wrapper from step 1).
    *   Handle the returned `(value, latency)` tuple (or the wrapped `VarState`) correctly.
    *   Handle `ReturnValue` signal propagation.
    *   Add tests calling actual component methods (like `Disk.Read`).
3.  **Implement `OperationDef` Evaluation:** Allow user-defined operations within components to be called. This involves setting up a new environment frame, passing arguments, executing the operation's `BlockStmt` bytecode/AST, and handling the `ReturnStmt`.

**Phase 3: Basic Bytecode Compiler & Hybrid VM Loop**

*Goal: Implement a simple compiler and update the VM to execute basic bytecode, delegating complex flow to Go helpers.*

1.  **Define Bytecode Opcodes:** Finalize the initial set of opcodes (PUSH_CONST, LOAD/STORE_VAR, basic Arithmetic/Boolean ops, CALL_METHOD, RET, AND_SEQ, DELAY_OP, CALL_VM_IF, CALL_VM_DISTRIBUTE, etc.).
2.  **Implement Compiler (`dsl/compiler.go`):**
    *   Input: AST (`*dsl.File` or specific nodes).
    *   Output: Bytecode sequence (`[]byte` or `[]Instruction`), Constant Pool.
    *   Start by compiling simple expressions (literals, identifiers, binary/unary ops) and statements (`LetStmt`, `ExprStmt`, `ReturnStmt`, `DelayStmt`).
    *   Emit `CALL_VM_*` opcodes for `if`/`distribute`.
    *   Handle basic environment/scope mapping (variable slots).
3.  **Implement Bytecode VM Loop (`dsl/vm_exec.go`):**
    *   Takes bytecode and constant pool.
    *   Fetches, decodes, executes instructions.
    *   Manages instruction pointer (IP) and stack (holding `*VarState`).
    *   Implement execution logic for initial opcodes.
    *   When `CALL_VM_*` opcode is hit, call the corresponding Go `eval*` function (passing necessary context/bytecode offsets). These Go functions might recursively call the bytecode loop for sub-blocks.
4.  **Update Driver:** Modify `RunDSL` to compile AST to bytecode and then execute using the bytecode VM loop.
5.  **Tests:** Add tests specifically for the compiler (AST -> Bytecode) and the bytecode execution loop (Bytecode -> Correct Stack/Env State).

**Phase 4: Parser Implementation & Integration**

*Goal: Allow defining systems directly in DSL files.*

1.  **Implement Parser (`dsl/parser.go`):** Use `participle` or similar to parse DSL text based on the defined grammar into the AST (`ast.go`).
2.  **Integrate Parser:** Update `RunDSL` (or add entry point) to accept file paths/strings, parse them, and execute the resulting AST (via compiler/VM).
3.  **Examples & Docs:** Rewrite an example using DSL, write initial DSL documentation.

**Phase 5: Advanced Features & Refinements**

*   Implement remaining evaluators/opcodes (`Parallel`, `Repeat`, `Fanout`, `Switch`, `Filter`).
*   Implement `expect` clause evaluation.
*   Refine error handling and reporting (source positions).
*   Add more built-in functions.
*   Performance optimizations (bytecode, VM).


**Suggested Next Steps Order:**

1.  **Implement `core.SplitLatenciesBasedOnValue`** + tests. **(Highest Priority)**
2.  **Test `evalIfStmt` / `evalBlockStmt`** thoroughly using manual ASTs.
3.  **Implement `evalDelayStmt`** + tests.
4.  **Implement `evalDistributeStmt`** + tests.
5.  **Refactor/Wrap Component Methods & Implement `evalCallExpr`**.
6.  Start **Phase 3 (Basic Bytecode Compiler & Hybrid VM Loop)** for expressions and simple statements.

