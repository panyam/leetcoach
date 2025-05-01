# SDL Project Summary & Onboarding Guide

**1. Vision & Goal:**

*   **Purpose:** The SDL (System Design Language) library models and analyzes the performance characteristics (latency, availability) of distributed system components using **analytical models** and **probabilistic composition**. It prioritizes **speed** for interactive "what-if" analysis over detailed Discrete Event Simulation (DES), focusing primarily on **steady-state** behavior.
*   **Use Cases:** Enable rapid analysis of system designs, bottleneck identification, SLO evaluation, performance exploration under different configurations, and potentially educational tools for system design reasoning.

**2. Overall Architecture:**

*   **Probabilistic Core (`sdl/core`):** The foundation, providing generic `Outcomes[V]` distributions, composition operators (`And`, `Map`, `Split`, `Append`), complexity management via reduction (`Trim...`, `Merge...`, etc.), metric calculations (`Availability`, `PercentileLatency`, etc.), and a standardized `Analyze` primitive for testing the Go API.
*   **Analytical Components (`sdl/components`):** Concrete Go models of system building blocks (Disk, Cache, Queue, Pool, LSM, BTree, etc.) using `sdl/core`. Key components (`ResourcePool`, `Queue`) use stateless analytical formulas (M/M/c, M/M/c/K).
*   **Declarative Layer (`sdl/components/decl`):** Defines component structures and methods (`Read()`, `Find()`, etc.) that generate Abstract Syntax Trees (ASTs) representing operation logic, separating definition from execution.
*   **DSL Execution Engine (`sdl/dsl`):** Contains the AST definition (`ast.go`), Bytecode (`bytecode.go`), Compiler (`compiler.go`), VM (`vm.go`), Environment (`environment.go`), and Driver (`driver.go`).
    *   **Target Model (V4):** Implicit dual-track (Value + Latency) state, discrete value sets, explicit `delay`, probabilistic `distribute`.
    *   **Execution Strategy:** Hybrid Bytecode/Go Helpers. AST is compiled to bytecode. The VM executes bytecode, calling registered Go handlers (`vm.vm_handle_*`) for complex operations (arithmetic, sequence, delay, if, distribute).
*   **Examples (`sdl/examples`):** Demonstrates usage of the **Go API** to build and analyze systems (`bitly`, `gpucaller`, `notifier`).

**3. Code Structure & Package Overview:**

*   **`sdl/` (Root)**
    *   `go.mod`, `go.sum`: Go module definitions.
    *   `Makefile`: Build/test/watch commands.
    *   `SUMMARY.md`: (This document) Project overview.
    *   `NEXTSTEPS.md`: Outstanding tasks and future work.
    *   `dsl/SYNTAX.md`, `dsl/GRAMMAR.md`, `DSL.md`: DSL design notes.
*   **`sdl/core/`**: Fundamental, generic types and operations. (See `sdl/core/SUMMARY.md`).
*   **`sdl/components/`**: Concrete system building blocks using `sdl/core`. Includes `decl/` sub-package for AST generation. (See `sdl/components/SUMMARY.md`).
*   **`sdl/dsl/`**: DSL AST, Bytecode, Compiler, VM, Driver. (See `sdl/dsl/SUMMARY.md`).
*   **`sdl/examples/`**: Examples using the Go API. (See `sdl/examples/SUMMARY.md`).

**4. Current Status & Recent Work Summary:**

*   The core analytical library (`sdl/core`, `components`) is functional for modelling steady-state performance via probabilistic composition using the **Go API**. Standardized testing via `core.Analyze` is implemented.
*   The declarative layer (`sdl/decl`) generates ASTs for component operations.
*   The **DSL execution engine (`sdl/dsl`)** is under development using the **Hybrid Bytecode** strategy targeting **Model V4**:
    *   AST definitions are in place.
    *   Bytecode format (`Instruction`, `Chunk`) and opcodes (using `OpCallFunc`) defined.
    *   Basic compiler emits bytecode for literals, simple expressions (`+`, `-`, `!`, etc. via `OpCallFunc`), globals (`Let`/`Get`), `AndExpr`, `ExprStmt`, `ReturnStmt`, `IfStmt` (with jumps).
    *   VM executes basic bytecode (`OpPushConst`, `OpPop`, `OpDefine/Get/SetGlobal`, `OpJump*`, `OpReturn`, `OpCallFunc`).
    *   VM includes stubs/basic implementations for `vm_handle_*` handlers (Add, AndSeq, Delay, If, Distribute).
    *   VM state uses dual-track `VarState` struct.
    *   Driver needs update for bytecode flow.
*   **Recent Work:** Shifted from direct AST evaluation to the Hybrid Bytecode VM structure. Implemented basic compiler and VM loop structures with simplified opcodes and `OpCallFunc` handlers.

**5. Known Limitations (Explicitly Acknowledged):**

*   **Analytical Steady-State:** Models average performance, not transient bursts/dynamics. Accuracy depends on analytical models (e.g., M/M/c assumptions).
*   **Stateless Components:** `ResourcePool`/`Queue` rely purely on configured rates.
*   **Profile Accuracy:** Depends on realistic input `Outcomes` distributions (especially for leaf components like `Disk` or work profiles like `gpuwork`).
*   **VM Incomplete:** Requires full implementation of `vm_handle_*` functions (especially for `If`, `Distribute`, probabilistic ops), `OpCall` for user/component methods, local variable scopes. `combineOutcomesAndReduce` uses simplified "Take Last Value". `SplitLatenciesBasedOnValue` helper needed in `core`.
*   **Compiler Incomplete:** Needs to handle all AST nodes, local scopes, potentially function definitions.
*   **Driver:** Needs update for bytecode compilation/execution flow.
*   **DSL Parser Not Implemented:** Currently relying on manually constructed ASTs or ASTs generated by `decl`.
*   **Component Methods:** Go API methods need adaptation or VM wrappers for Model V4 dual-track compatibility.

**6. Next Steps:**

*   Focus on implementing the `vm_handle_*` functions in the VM to handle core operations and control flow correctly for the dual-track `VarState`.
*   Implement the `core.SplitLatenciesBasedOnValue` helper.
*   Implement `OpCall` and associated VM logic for calling component methods.
*   Refine compiler and VM for local scopes.
*   Update Driver.
*   Implement Parser.
*   See `NEXTSTEPS.md` for a detailed breakdown.
