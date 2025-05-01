# SDL Examples Package Summary (`sdl/examples`)

**Purpose:**

This package contains example system models built using the SDL library's **Go API**. These examples demonstrate how to use the `sdl/core` primitives and `sdl/components` models directly in Go code to model scenarios and analyze performance. They serve as integration tests and usage guides for the library's Go interface.

**Key Examples:**

1.  **`bitly/`:**
    *   Models a simplified URL shortening service.
    *   Uses `IDGenerator`, `components.Cache`, `components.HashIndex` (via `DatabaseComponent`), `BitlyService`.
    *   Demonstrates composing cache/DB operations via Go calls.
    *   Uses `core.Analyze` for testing.

2.  **`gpucaller/`:**
    *   Models an AppServer -> GPU pool interaction with batching.
    *   Uses `components.Batcher`, `components.ResourcePool` (stateless M/M/c), custom `GpuBatchProcessor`, `AppServer`.
    *   Includes parameter sweeping tests using `core.Analyze`.

3.  **`notifier/`:**
    *   Models a notification fan-out system (synchronous save + asynchronous delivery).
    *   Uses `components.HashIndex`, `components.LSMTree`, custom stores (`MessageStore`, `InboxStore`), `AsyncProcessor`, CDC delay simulation.
    *   Highlights fan-out modeling challenges (currently uses approximation).

**Current Status:**

*   Provides concrete usage examples for the **Go API**.
*   Tests use `core.Analyze` for verification.
*   Serves as integration tests for `core` and `components`.
*   **Not yet updated** to reflect potential API changes needed for Model V4 dual-track returns from component methods (if direct Go calls are modified instead of using VM wrappers).
*   Could be rewritten using the DSL once the parser and VM are functional.
