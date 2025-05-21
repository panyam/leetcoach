# LeetCoach - Next Steps & Roadmap

This document outlines outstanding tasks, refactoring work, and future feature implementation steps for the LeetCoach project.

**I. Complete Current Refactoring (Composition Pattern)**

1.  **Refactor `PlotSection` to Composition:**
    *   Create `PlotSectionView.ts` and `PlotSectionEdit.ts`.
    *   `PlotSectionView`: Fetch JSON config (from `content.main`), display as preformatted text or placeholder.
    *   `PlotSectionEdit`: Load template with `<textarea>` for JSON config. Fetch/Save JSON config via `ContentApi`. Handle `onSaveSuccess`/`onCancel`.
    *   Update `BaseSection.ts` (`renderModeComponent`) to instantiate these.
    *   Delete old `PlotSection.ts` and placeholder.
    *   **Verify:** Plot sections work with the new pattern (editing JSON, saving, viewing).

**II. System Description Language (DSL) Feature - Core Functionality**

1.  **DSL Editor - Monaco Integration (Replacing Textarea):**
    *   **Task:** Integrate Monaco Editor into `SystemDescriptionEdit.ts`.
    *   **Details:**
        *   Add Monaco Editor library (npm recommend).
        *   Replace `<textarea>` with Monaco Editor instance.
        *   Configure basic options (theme, word wrap, minimap).
        *   Ensure content loading/saving works with Monaco's API (`getValue`, `setValue`).
        *   Handle theme changes for Monaco.
    *   **Verify:** DSL editing experience is improved with a proper code editor.

2.  **DSL Syntax Highlighting (Monaco & View Mode):**
    *   **Task:** Implement syntax highlighting for the DSL.
    *   **Details:**
        *   **Monaco:** Define and register a Monaco Language configuration based on the DSL grammar.
        *   **View Mode:** Choose a client-side library (Prism.js or use Monaco in read-only mode). If Prism, define a language grammar for it. Apply to the `<code>` block in `SystemDescriptionView.ts`.
    *   **Verify:** DSL code is highlighted correctly in both edit and view modes.

3.  **Backend - Real DSL Parser Integration:**
    *   **Task:** Replace `dummyParseDSL` in `services/system_model_service.go` with your actual Go DSL parser.
    *   **Details:** Ensure parser errors are mapped to `ValidationError` (line, column, message).
    *   **Verify:** `ValidateDSL` API provides accurate syntax errors.

4.  **Backend - Diagram Generation (Component Connections):**
    *   **Task:** Implement logic in `SystemModelService.GenerateDiagram` to parse the DSL and generate a static component connection diagram.
    *   **Details:**
        *   Use the real parser to get an AST.
        *   Traverse AST for `component`, `uses`, `system`/`instance` declarations.
        *   Generate diagram data (SVG directly, or DOT/Mermaid to be converted/rendered).
        *   Return this data in `GenerateDiagramResponse`.
    *   **Verify:** The "Generate/Refresh Diagram" button in `SystemDescriptionView` displays a meaningful diagram based on the DSL.

**III. System Description Language (DSL) Feature - Analysis & Advanced Diagrams**

1.  **Backend - Basic Analysis Engine:**
    *   **Task:** Implement stub `SystemModelService.RunAnalysis`. Create initial simulator for `delay` and sequential method calls from an `analyze` block.
    *   **Details:** Parse DSL, identify `analyze` block, simulate specified method call with fixed timings. Aggregate simple metrics (e.g., total time).
    *   **Frontend:** Wire "Run Analysis" button in `SystemDescriptionView` to call API and display basic text/tabular results in `#analysis-results-container`.
2.  **Advanced Simulation:**
    *   Implement probabilistic `Outcome`s.
    *   Implement `distribute`.
    *   Implement `go`/`wait` (concurrency).
3.  **`expect` Block Evaluation:** Compare simulation results against `expect` conditions.
4.  **Frontend Plotting:** Integrate a plotting library (Chart.js, etc.) in `SystemDescriptionView` to visualize analysis results (histograms, P-latencies).
5.  **Dynamic/Multiple Diagrams:**
    *   Add `diagram_type` selector to `SystemDescriptionView`.
    *   Enhance `GenerateDiagram` backend to support different diagram types (e.g., sequence diagrams for specific methods).

**IV. UI/UX Enhancements**

1.  **System Model Layout Toggle:** Implement user-selectable layouts (Vertical vs. Horizontal Split) for `SystemDescriptionView`.
2.  **Editor Error Annotations:** For DSL, display validation errors as markers in Monaco.
3.  **Mermaid Rendering:** If Mermaid is chosen for diagrams, integrate its JS library.
4.  **Dark Mode Polish:** Ensure all components, including third-party editors (Toast UI, Monaco, Excalidraw) and syntax highlighting (Prism), consistently and correctly adapt to theme changes without visual glitches.

**V. LLM Feature Enhancements**

1.  **Contextual Prompts:** Improve `LlmService` to gather more context.
2.  **DSL & LLM:** Explore LLM assistance for writing/reviewing DSL code.
3.  **Fine-tune `SuggestSections` Flow.**

**VI. General Maintenance & Polish**

1.  **Comprehensive Testing:** Unit, integration tests for backend (especially DSL parser/simulator).
2.  **Error Handling & User Feedback:** Standardize and improve.
3.  **Dependencies & Build Process:** Keep up-to-date.
4.  **Documentation.**

**Suggested Immediate Focus Order:**

1.  **Finish `PlotSection` Refactoring (I.1).**
2.  **Monaco Editor for DSL (II.1).**
3.  **Real DSL Parser Backend (II.3).**
4.  **DSL Syntax Highlighting (II.2).**
5.  **Basic Diagram Generation (II.4).**
