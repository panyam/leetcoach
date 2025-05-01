# SDL Core Package Summary (`sdl/core`)

**Purpose:**

This package forms the fundamental layer of the SDL library. It provides the generic building blocks for representing probabilistic performance outcomes, composing them, calculating standard metrics, and managing complexity through distribution reduction. It is designed to be independent of specific system components and the DSL execution layer.

**Key Concepts & Components:**

1.  **`Outcomes[V any]` (`outcomes.go`):**
    *   The central data structure representing a discrete probability distribution.
    *   A slice of `Bucket[V]` structs, where each bucket has a `Weight` (probability) and a `Value` of generic type `V`.
    *   Provides core methods: `Add`, `Len`, `TotalWeight`, `Copy`, `Split` (conditional branching based on value predicate), `Partition`, `ScaleWeights`, `GetValue` (for deterministic results), `Sample` (random sampling), `Append`.
    *   *(Note: The `And` field on this struct is deprecated/unused; composition is handled externally by callers like the VM using reducers).*

2.  **Composition Operators (`outcomes.go`):**
    *   Generic functions like `And` (sequential composition), `Map` (transform outcome values). `Append` is primarily a method on `*Outcomes[V]`.
    *   `And` requires a type-specific `ReducerFunc` (e.g., `AndAccessResults`, `AddDurations`) provided externally (often via a registry in the VM) to define how inner values combine probabilistically (weights multiply, values combine via reducer).

3.  **Result Types (`simpleresult.go`, `rangedresult.go`, `utils.go`):**
    *   Defines concrete types for `V` used commonly in performance modeling.
    *   `AccessResult`: `{Success bool, Latency Duration}` - Represents success/failure with point latency. *Note: Under Model V4 DSL, this struct might be decomposed, with `Success` (bool) in the value track and `Latency` (Duration) in the latency track.*
    *   `RangedResult`: `{Success bool, MinLatency, ModeLatency, MaxLatency Duration}` - Represents outcomes with latency uncertainty.
    *   `Duration`: Type alias for `float64` representing time in seconds (`utils.go`).

4.  **`Metricable` Interface (`metrics.go`):**
    *   Interface (`IsSuccess() bool`, `GetLatency() Duration`) required by outcome types (`V`) to be used with generic metric calculations (like `AccessResult`, `RangedResult`). *Note: Under Model V4, metrics might be calculated differently based on separate value/latency tracks.*

5.  **Metrics Calculation (`metrics.go`, `metrics_test.go`):**
    *   Provides generic functions (`Availability`, `MeanLatency`, `PercentileLatency`) operating on `Outcomes[V]` where `V` implements `Metricable`. *Note: Applicability might change with Model V4's dual tracks.*

6.  **Reduction / Trimming (`reducer.go`, `duration.go`, `simpleresult.go`, `rangedresult.go`, `reduction_benchmark_test.go`):**
    *   Essential mechanisms to manage the combinatorial explosion of buckets during composition.
    *   Strategies include:
        *   **Duration:** `InterpolateDurations`, `ReduceDurationsAdaptive` (using `DurationSignificanceWeightedLatencyDelta`). Configurable via `DurationReducerFunc`.
        *   **AccessResult:** `MergeAdjacentAccessResults`, `TrimToSize` (using Merge + `ReduceAccessResultsPercentileAnchor`).
        *   **RangedResult:** `MergeOverlappingRangedResults`, `InterpolateRangedResults`, `TrimToSizeRanged`.
        *   **Generic:** `AdaptiveReduce`.

7.  **Latency Splitting Helper (`dsl/eval_ifstmt.go` currently, move to `core`):**
    *   `SplitLatenciesBasedOnValue[V any](...)`: Helper function needed to split an `Outcomes[Duration]` distribution based on a predicate applied to a corresponding (synchronized) `Outcomes[V]` distribution. Crucial for the VM's `if`/`distribute` logic in Model V4. *(Note: Currently defined in `dsl`, should be moved to `core`)*.

8.  **Distribution Helpers (`distributions.go`, `distributions_test.go`):**
    *   Utilities to create common distributions.
    *   `NewDistributionFromPercentiles`: Creates `Outcomes[AccessResult]`.

9.  **Analysis Primitive (`analyzer.go`, `analyzer_test.go`):**
    *   **Stateless `Analyze` function:** Used extensively in **Go API tests**. Takes a simulation function `func() *Outcomes[V]`, calculates metrics, checks `Expectation`s, returns `AnalysisResult[V]`.
    *   *Note: The DSL uses `analyze` blocks, which trigger VM execution culminating in metrics calculated by the VM/Driver using these core metric functions.*

10. **Converters (`converters.go`, `converters_test.go`):**
    *   Functions to convert between `Outcomes[AccessResult]` and `Outcomes[RangedResult]`.
    *   Includes `SampleWithinRange` for sampling from a triangular distribution defined by `RangedResult`.

11. **Utilities (`utils.go`):**
    *   `Duration` type alias, time unit helpers (`Millis`, etc.), `ApproxEq`.

**Current Status:**

*   The core library is mature and provides a robust foundation for probabilistic modeling via the Go API.
*   Key composition, reduction (including adaptive duration reduction), metric, and Go analysis functions are implemented and tested. Benchmarks exist for reduction.
*   Generic design allows extension.
*   Includes necessary primitives (except `SplitLatenciesBasedOnValue` which needs to be moved/implemented here) to support the dual-track logic of the target Model V4 DSL VM.
*   Ready to be driven by the DSL execution engine.
