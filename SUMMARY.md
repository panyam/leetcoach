# LeetCoach Project Summary

**1. Overview:**

LeetCoach is a web application designed to assist users in preparing for system design interviews. Its primary function is enabling the creation, editing, and viewing of multi-section "Design" documents. A key feature is the integration of LLM capabilities for content generation, review, and suggesting design elements. The application is undergoing a significant refactoring of its frontend section components towards a Composition pattern.

**2. Architecture & Technologies:**

*   **Backend:** Go, gRPC, gRPC-Gateway
*   **Frontend:** Go Templates (SSR), TypeScript (Vanilla JS patterns for UI logic, *except* for Excalidraw's React dependency), Tailwind CSS
*   **API:** RESTful JSON API exposed via gRPC-Gateway, consumed by the frontend TypeScript.
*   **Persistence:**
    *   **Filesystem:** (`./data/designs/`) Stores core design/section metadata, content files, and LLM prompts. Managed via `services/designs_store.go`.
    *   **Google Cloud Datastore:** Stores Users, Tags, Auth entities. Managed via `services/clientmgr.go` and specific service logic (e.g., `services/auth.go`, `services/tags.go`).
*   **LLM:** Interface (`services/llm/client.go`) with an OpenAI implementation. `LlmService` orchestrates interactions.
*   **Authentication:** `oneauth` library handling local & OAuth (Google/GitHub), integrated with backend `AuthService`.
*   **Code Generation:** Protobuf generates Go service interfaces/structs and the TypeScript API client.

**3. Core Features:**

*   **Design Documents:** Create, view, edit, list, delete designs.
*   **Sections:** Add, delete, reorder sections within a design.
*   **Section Types:**
    *   **Text:** Rich text editing (TinyMCE).
    *   **Drawing:** Diagramming (Excalidraw).
    *   **Plot:** Basic JSON config display/edit (placeholder for future plotting).
    *   **System Description:** DSL code editing (`<textarea>` currently), validation, diagram generation (stubbed).
*   **LLM Integration:**
    *   Suggest new sections.
    *   Generate/Review content for Text sections using filesystem prompts.
    *   Generate default prompts for sections.
    *   Validate DSL for System Description sections.
    *   Generate diagrams (stubbed) for System Description sections.
*   **User Authentication:** Login/Signup via email/password or OAuth providers.

**4. Key Recent Development (Refactoring):**

The frontend architecture for handling different section types has been refactored from an Inheritance model (`[Type]Section extends SectionContainer`) to a **Composition model**:

*   `SectionContainer.ts` now acts as a **`SectionContainer`**. It manages the section frame, universal controls, metadata (`SectionData`), and the current mode (`view`/`edit`).
*   Specific **`[Type]SectionView.ts`** and **`[Type]SectionEdit.ts`** components are created for each section type (Text, Drawing, System Description completed; Plot pending).
*   These mode-specific components are instantiated by the `SectionContainer` based on the current mode and section type.
*   **Mode components are responsible for:**
    *   Rendering their specific UI (view display or editor/controls).
    *   Fetching their own data using `ContentApi`.
    *   Handling their own save logic (calling `ContentApi`) within the Edit component.
    *   Communicating success/cancel back to the container via callbacks (`onSaveSuccess`, `onCancel`).
*   This refactoring aims for better separation of concerns, simpler components, and improved maintainability.

**5. Folder Structure Summary:**

*   `./`: Root application entry point (`main.go`), top-level config (`.env`), core app orchestration (`app.go`).
*   `./proto/`: Protobuf definitions for gRPC services and messages.
*   `./services/`: Backend Go implementation of gRPC services, business logic, persistence layer interaction (`DesignStore`, `ClientMgr`).
*   `./web/`: Go web server, API gateway setup, auth handlers, SSR view logic entry point.
*   `./web/views/`: Go structs and logic for server-side rendering data preparation.
*   `./web/views/templates/`: Go HTML templates for pages and client-side UI components (`TemplateRegistry.html`).
*   `./web/views/components/`: Core frontend TypeScript components (Section Container, Mode Components, Managers, API setup).
*   `./data/`: (Implicit) Root for filesystem-based design/section persistence.

**6. Current State:**

*   The core application features (designs, sections, auth, basic LLM) are functional.
*   The Composition refactor for frontend sections is complete for `Text`, `Drawing`, and `SystemDescription` types. `PlotSection` still needs refactoring.
*   The `SystemDescription` section type allows DSL editing via `<textarea>`, saving/loading, backend validation (stubbed parser), and diagram generation (stubbed backend returning mock SVG).
*   Monaco Editor / Prism.js integration for the DSL section is deferred.
*   Real DSL parsing and diagram generation logic are not yet implemented in the backend.
*   Analysis features for the DSL are not started.

