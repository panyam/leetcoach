# SDL Components Package Summary (`sdl/components`)

**Purpose:**

This package provides concrete Go implementations of common distributed system components, built upon the abstractions defined in `sdl/core`. These components model the performance characteristics (latency, availability) of specific building blocks like disks, caches, queues, etc. They serve as the executable backend for both direct Go API usage and the DSL.

**Key Concepts & Components:**

1.  **Foundation:** Relies on `sdl/core` for `Outcomes`, `Duration`, composition, reduction, etc. Uses type aliases from `aliases.go`.
2.  **Component Models (Go Structs):**
    *   Implementations for `Disk`, `Cache`, `Queue` (M/M/c/K), `MM1Queue`, `ResourcePool`, `BTreeIndex`, `HashIndex`, `LSMTree`, `NetworkLink`, `Batcher`, `BitmapIndex`, `HeapFile`, `SortedFile`.
    *   Provide `Init()` or `New...()` constructors.
    *   Operation methods (`Read`, `Write`, `Acquire`, `Submit`, etc.) return `*core.Outcomes[V]`. **Note:** Under the target Model V4 DSL, these methods will need adaptation or VM wrappers to align with the DSL's expectation of processing dual value/latency tracks. Currently, they mostly return `*core.Outcomes[AccessResult]` or `*core.Outcomes[Duration]`.
    *   Internal logic often involves composing core primitives (e.g., `Disk.Read()`).

3.  **Stateless Analytical Models:**
    *   `ResourcePool` (`resourcepool.go`) and `Queue` (`queue.go`, `mm1queue.go`) use stateless M/M/c(/K) analytical models, predicting performance based on configured rates (`lambda`, `Ts`, `c`, `K`).

4.  **Declarative Layer (`decl/` sub-package):**
    *   Contains parallel Go structs (e.g., `decl.Disk`) holding configuration as `dsl.Expr`.
    *   Provides methods (`Read()`, `Find()`, etc.) that **generate Abstract Syntax Trees (ASTs)** (`sdl/dsl/ast.go`), representing operation logic declaratively.
    *   This layer separates system *definition* from VM *execution*.

5.  **Testing:**
    *   Component tests (`*_test.go`) use the `core.Analyze` primitive to verify the Go API behavior against performance `Expectation`s.

**Current Status:**

*   Provides a comprehensive suite of standard system components with performance models based on `sdl/core`.
*   Key flow components use stateless analytical models.
*   The `decl` sub-package provides AST generation for the DSL front-end.
*   Component models are well-tested via the Go API using `core.Analyze`.
*   **Pending Refactoring/VM Adaptation:** Component operation methods need review/adaptation for the dual-track `(value, latency)` processing model expected by the Model V4 VM design. The current return types (`*core.Outcomes[V]`) need to be handled appropriately by the VM/Driver (likely via wrappers during component registration in the VM).
