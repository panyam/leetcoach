# SDL Project - Next Steps (Bytecode VM Focus)

**Overall Goal:** Implement the core functionality of the DSL execution engine (VM + Compiler) based on the **Hybrid Bytecode execution model** and targeting the **Model V4 DSL syntax** (Implicit Dual Track, Discrete Values, Explicit Delay).

**Phase 1: VM Handler Implementation & Core Logic (Current Focus)**

*Goal: Flesh out the VM's Go handlers (`vm_handle_*`) called by `OpCallFunc` to correctly implement arithmetic, boolean logic, sequence combination, and explicit delay for the dual-track `VarState`.*

1.  **Implement Arithmetic Handlers (`vm_handle_add`, `_subtract`, `_multiply`, `_divide`, `_negate`):**
    *   **Task:** In `dsl/vm.go`, implement the logic within these handlers. They need to:
        *   Pop the required number of `*VarState` operands from the stack.
        *   Perform type checking (e.g., expect `*Outcomes[int64]` or `*Outcomes[float64]`). Handle type errors.
        *   Ensure operands are deterministic (`GetValue()`). Handle errors if not (for now).
        *   Perform the arithmetic operation on the deterministic values.
        *   Combine the latency tracks of the operands using `combineOutcomesAndReduce` (or just its latency part).
        *   Create a new `VarState` with the computed value outcome (wrapped) and the combined latency outcome.
        *   Push the resulting `VarState` back onto the stack.
    *   **Testing:** Add specific tests in `dsl/vm_exec_test.go` for each arithmetic operation, checking both value and combined latency. Unskip `TestExecuteChunk_BinaryAdd`.

2.  **Implement Boolean/Comparison Handlers (`vm_handle_not`, `_equal`, `_notEqual`, `_greater`, `_less`, etc., `_and`, `_or`):**
    *   **Task:** Similar to arithmetic handlers, implement the logic: Pop operands, check types (bool, or comparable types), perform deterministic comparison/logic, combine latency tracks, create and push result `VarState` (`*Outcomes[bool]` value).
    *   **Testing:** Add tests for boolean logic and comparisons.

3.  **Refine `vm_handle_and_seq` and `vm_handle_delay`:**
    *   **Task:** Ensure these handlers correctly use `combineOutcomesAndReduce` (or alternatives if value propagation changes) for the dual tracks. Verify latency combination logic.
    *   **Testing:** Unskip and ensure `TestExecuteChunk_SimpleSequence` passes (once `DurationMS` handler registration is sorted in tests). Add more sequence tests.

**Phase 2: Probabilistic Control Flow & Core Helper**

*Goal: Implement the complex logic for `if` and `distribute`, requiring the core latency splitting helper.*

1.  **Implement `core.SplitLatenciesBasedOnValue`:** **(Critical Blocker)**
    *   **Task:** Implement this helper function in `sdl/core`. Design it to efficiently split `Outcomes[Duration]` based on a predicate applied to a synchronized `Outcomes[V]` without reflection if feasible (may require helpers on `Outcomes`).
    *   **Testing:** Add thorough unit tests in `sdl/core`.

2.  **Implement `vm_handle_if`:**
    *   **Task:** This Go helper function (called by `OpCallFunc HandlerIf`, likely needing the jump offsets passed somehow or read from subsequent instructions) needs to:
        *   Pop the condition `*VarState` (must be `*Outcomes[bool]`).
        *   Use `core.SplitLatenciesBasedOnValue` (once available) and `Outcomes[bool].Split` to get `thenVal`, `elseVal`, `thenLat`, `elseLat` outcomes.
        *   Create `thenContext` and `elseContext` VarStates.
        *   Recursively call `vm.ExecuteChunk` (or similar) for the `then` bytecode block (using `thenContext`) and the `else` bytecode block (using `elseContext`). These recursive calls return the final `*VarState` for each branch.
        *   Combine the `thenResultState` and `elseResultState` using weighted `Append` logic for both value and latency tracks.
        *   Push the final combined `VarState`.
    *   **Testing:** Add complex `if` tests in `vm_exec_test.go`, verifying probabilistic combination of results and latency.

3.  **Implement `vm_handle_distribute`:**
    *   **Task:** Similar complexity to `vm_handle_if`. Needs to handle multiple probability cases, split context accordingly, execute branches, and combine results.
    *   **Testing:** Add `distribute` tests.

**Phase 3: Function/Method Calls & Scoping**

*Goal: Enable calling component methods and user-defined operations, introducing local scopes.*

1.  **Implement Local Variables & Scoping:**
    *   **Task:** Add support for local variables. This might involve using stack slots (`OpGetLocal`, `OpSetLocal`) and managing scope entry/exit during block compilation and execution. Update `Compiler` and `VM` execution loop.
    *   **Testing:** Add tests involving nested blocks and local variable shadowing.
2.  **Implement `OpCall` / `OpReturn`:**
    *   **Task:** Implement the VM logic for `OpCall`. This involves looking up the function/method, setting up a new call frame (potentially on the stack or a separate call stack), passing arguments, and transferring control (`ip`). `OpReturn` needs to restore the previous frame and `ip`.
    *   **Testing:** Add tests for simple function calls (once functions can be defined).
3.  **Component Method Adaptation / Wrappers:**
    *   **Task:** Decide on and implement the strategy (modify components or use VM wrappers) to make component methods compatible with the VM's dual-track `VarState` expectation.
    *   **Testing:** Add tests calling actual component methods (e.g., `Disk.Read`) via `OpCall`.

**Phase 4: Compiler Completeness & Parser**

*Goal: Compile the full AST and parse DSL files.*

1.  **Complete Compiler:** Implement compilation logic for all remaining AST nodes (`DistributeStmt`, `ParallelStmt`, `RepeatStmt`, `SwitchExpr`, etc.) emitting the appropriate bytecode and `OpCallFunc` / `OpCallVM_*` instructions.
2.  **Implement Parser (`dsl/parser.go`):** Use `participle` or similar to parse DSL text based on `dsl/GRAMMAR.md` into the `ast.go` structures.
3.  **Update Driver:** Modify `RunDSL` to accept file paths/strings, invoke the parser, then the compiler, then `vm.ExecuteChunk`.
4.  **Examples & Docs:** Rewrite an example using DSL, write initial DSL documentation.

**Phase 5: Refinements**

*   Refine `combineOutcomesAndReduce` for accurate value propagation if needed.
*   Implement `expect` clause evaluation in the Driver/VM.
*   Improve error handling and reporting (source mapping from bytecode).
*   Performance optimizations.
