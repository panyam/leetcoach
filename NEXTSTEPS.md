# LeetCoach - Next Steps & Roadmap

This document outlines outstanding tasks, refactoring work, and future feature implementation steps for the LeetCoach project.

**I. Immediate Tasks (Finish Current Refactoring)**

1.  **Refactor `PlotSection` to Composition:**
    *   Create `PlotSectionView.ts` and `PlotSectionEdit.ts`.
    *   Move view logic (placeholder rendering for now) to `PlotSectionView`. Fetch data internally.
    *   Move edit logic (JSON textarea) to `PlotSectionEdit`. Implement internal save/load via `ContentApi`. Handle `onSaveSuccess`/`onCancel`.
    *   Update `BaseSection.ts` (`renderModeComponent`) to instantiate the new Plot components.
    *   Delete old `PlotSection.ts` and placeholder files.
    *   **Verify:** Ensure Plot sections work correctly with the new pattern alongside other refactored sections.

**II. System Description Language (DSL) Feature Implementation**

*(Phased approach)*

1.  **Phase 2 Polish - Editor & Highlighting:**
    *   **Task:** Replace the `<textarea>` in `SystemDescriptionEdit.ts` with the Monaco Editor. Replace the `<pre><code>` view with Prism.js (or use Monaco in read-only mode).
    *   **Details:**
        *   Integrate Monaco Editor library (npm install or CDN).
        *   Configure Monaco instance in `SystemDescriptionEdit.ts` (theme, basic options).
        *   Define and register a Monaco Language configuration based on the DSL grammar for syntax highlighting in the editor.
        *   Integrate Prism.js (or alternative) for view mode highlighting, defining a custom language grammar for it. Apply to the `<code>` block in `SystemDescriptionView.ts`.
    *   **Verify:** Syntax highlighting works correctly in both edit and view modes.

2.  **Phase 2 Polish - Real Backend Validation:**
    *   **Task:** Replace the `dummyParseDSL` function in `services/system_model_service.go` with calls to your actual Go DSL parser.
    *   **Details:**
        *   Ensure your parser returns errors that can be reasonably mapped to the `ValidationError` protobuf message (line, column, message).
        *   Update the `ValidateDSL` gRPC method to use the real parser and return accurate error details.
    *   **Verify:** Clicking "Validate DSL" provides meaningful, accurate errors based on the real parser output.

3.  **Phase 3 - Diagram Generation (Component Connections):**
    *   **Task:** Implement the backend logic to generate a static component connection diagram from the DSL and display it.
    *   **Details:**
        *   Enhance the backend parser/create an AST visitor to extract `component`, `uses`, and `system`/`instance` information.
        *   Implement logic in `SystemModelService.GenerateDiagram` to traverse this structure and generate diagram data (e.g., Graphviz DOT format).
        *   Convert DOT to SVG using a Go library or shell command (if acceptable). Alternatively, generate Mermaid syntax string.
        *   Return the SVG string or Mermaid string in `GenerateDiagramResponse`.
        *   Enhance `SystemDescriptionView.ts` (`handleGenerateDiagramClick`) to correctly render the received SVG (inject into `diagramContainer.innerHTML`) or use the Mermaid JS library to render Mermaid syntax.
    *   **Verify:** DSL defining components and `uses` clauses generates a corresponding visual diagram in the view mode after clicking "Generate/Refresh Diagram".

4.  **Phase 4 - Basic Analysis Engine & Results:**
    *   **Task:** Implement a minimal simulation engine and display basic results.
    *   **Details:**
        *   Backend: Implement `SystemModelService.RunAnalysis` (stub first).
        *   Backend: Create a basic simulator package. Handle `delay` (fixed time), simple method calls between instances defined in `system`. Ignore probability/concurrency initially.
        *   Backend: Aggregate simple metrics (e.g., total execution time).
        *   Frontend: Add "Run Analysis" button logic in `SystemDescriptionView.ts`. Call the API.
        *   Frontend: Display returned numeric/text results in the `#analysis-results-container` using simple HTML tables or text.
    *   **Verify:** Running analysis on a simple DSL produces basic numeric output displayed in the UI.

5.  **Phase 5+ - Advanced DSL Features:**
    *   Implement probabilistic `Outcome` simulation in the backend engine.
    *   Implement `distribute` keyword logic.
    *   Implement `go` / `wait` for concurrent simulation (more complex).
    *   Implement `expect` block evaluation against simulation results.
    *   Add frontend plotting (Chart.js, Plotly) for analysis results.
    *   Implement generation of different diagram types (e.g., sequence diagrams based on method calls) selectable via UI.

**III. UI/UX Enhancements**

1.  **System Model Layout Toggle:** Implement the user-selectable layout toggle ([Vertical Stack] vs. [Horizontal: Code+Diagram | Plots Below]) within the `SystemDescriptionView` component.
2.  **Editor Error Annotations:** Enhance `ValidateDSL` frontend handling to display errors directly as markers/squiggles within the Monaco editor (requires Monaco API knowledge).
3.  **Loading Indicators:** Improve loading state feedback during validation, diagram generation, and analysis runs.
4.  **Mermaid Rendering:** If Mermaid is chosen for diagrams, integrate the Mermaid JS library and ensure proper rendering/updates.
5.  **Refine Empty States:** Improve placeholder messages for empty DSL, diagrams, analysis.

**IV. LLM Feature Enhancements**

1.  **Contextual Prompts:** Enhance `LlmService` to gather context (other section titles/summaries) to improve prompt quality for generation/review.
2.  **DSL Generation/Review:** Add LLM capabilities to *suggest* or *review* the DSL code itself.
3.  **Suggest Next Sections:** Implement the `SuggestSections` LLM flow properly within the `SectionManager`'s add section workflow.

**V. General Maintenance & Other**

1.  **Testing:** Add more comprehensive unit and integration tests, especially for the DSL parser, simulator, and API services. Add frontend tests if a framework is adopted later.
2.  **Error Handling:** Improve consistency and user-friendliness of error messages across the application.
3.  **Dependencies:** Update dependencies periodically.
4.  **Deployment:** Define build and deployment processes (Dockerfile, Cloud Run/GKE configuration).
5.  **Documentation:** Keep READMEs and summaries up-to-date. Add more detailed developer documentation if needed.

**Suggested Order:**

1.  Finish PlotSection Refactoring (Item I.1).
2.  Implement Real Backend Validation (Item II.2).
3.  Implement Monaco Editor & Basic Highlighting (Item II.1).
4.  Implement Basic Diagram Generation & Display (Item II.3).
5.  Implement Basic Analysis Engine & Display (Item II.4).
6.  Implement UI Layout Toggle (Item III.1).
7.  Tackle Advanced DSL Features / LLM Enhancements / UX Polish iteratively.
