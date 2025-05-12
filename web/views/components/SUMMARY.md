# ./web/views/components/ Summary

**Purpose:**

This folder contains the core client-side TypeScript logic responsible for making the LeetCoach application interactive after the initial server-side render. It manages UI state, handles user events, interacts with the backend API via the gRPC-Gateway, manipulates the DOM, and orchestrates different parts of the user interface. It notably avoids using a major frontend framework (like React, Vue, Angular) *except* for the parts required by the integrated Excalidraw component. It implements the **Composition pattern** for section rendering.

**Key Files/Components & Recent Refactoring:**

*   **Refactoring Overview:** The primary architectural pattern for sections has shifted from Inheritance (`[Type]Section extends BaseSection`) to **Composition**. `BaseSection.ts` now acts as a `SectionContainer`, managing the frame and orchestrating mode-specific components (`[Type]View.ts` and `[Type]Edit.ts`). These mode components are responsible for their own data fetching, rendering, and saving logic. This refactoring is complete for Text, Drawing, and System Description sections; Plot is pending.

*   **`BaseSection.ts` (Section Container):**
    *   Manages the outer section element, header, and universal controls (Move, Delete, LLM, Add, etc.).
    *   Holds section metadata (`SectionData`).
    *   Manages the current mode (`view` or `edit`).
    *   Instantiates and destroys the appropriate `[Type]View` or `[Type]Edit` component when the mode changes, placing it in the `.section-content` container.
    *   Provides `onSaveSuccess` and `onCancel` callbacks to Edit components.
    *   Handles universal button clicks and generic click-to-edit logic (for applicable types).
*   **View Mode Components:**
    *   `TextSectionView.ts`: Fetches Markdown content, renders it using ToastUI renderer.
    *   `DrawingSectionView.ts`: Fetches SVG previews, renders the appropriate one based on theme.
    *   `SystemDescriptionView.ts`: Fetches DSL content, displays it in a `<pre><code>` block, contains section-specific action buttons (Validate, Generate Diagram), handles API calls for these actions, displays diagram/results placeholders.
    *   `PlotSectionView.ts`: (To be refactored).
*   **Edit Mode Components:**
    *   `TextSectionEdit.ts`: Loads template, fetches initial markdown content, initializes ToastUI, handles editor content changes, binds Save/Cancel buttons, calls `ContentApi.setContent` on save, signals container via `onSaveSuccess`/`onCancel`.
    *   `DrawingSectionEdit.ts`: Loads template, fetches initial JSON, initializes Excalidraw (via `ExcalidrawWrapper.tsx`), captures changes, handles Save (generates SVGs, calls `ContentApi` multiple times), signals container. Uses React/ReactDOM for Excalidraw.
    *   `SystemDescriptionEdit.ts`: Loads template, fetches initial DSL, uses a `<textarea>` (Monaco deferred), handles Save (calls `ContentApi`), handles Cancel, handles Validate button (calls `SystemModelApi`).
    *   `PlotSectionEdit.ts`: (To be refactored).
*   **Managers & Handlers:**
    *   `SectionManager.ts`: Manages the *collection* of `BaseSection` (container) instances on the editor page. Handles adding/deleting/moving sections (by manipulating the container list and calling `DesignService` API). Orchestrates the "Suggest Sections" flow. Updates `TableOfContents`.
    *   `LlmInteractionHandler.ts`: Manages the LLM modals (`llm-dialog`, `llm-results`), interacts with `LlmApi`, handles prompt saving via `DesignApi`.
    *   `ThemeManager.ts`: Handles light/dark/system theme switching logic and updates UI elements. Notifies section containers of changes.
    *   `Modal.ts`, `ToastManager.ts`: Singleton managers for global modals and toast notifications.
    *   `TableOfContents.ts`: Manages the TOC sidebar UI and interactions.
    *   `DocumentTitle.ts`: Handles inline editing of the main design title and API calls.
    *   `FullscreenHandler.ts`: Manages entering/exiting fullscreen mode for sections that support it.
*   **Page Entry Points:**
    *   `DesignEditorPage.ts`: Initializes all managers and components for the main editor view, orchestrates initial design metadata loading.
    *   `HomePage.ts`: Handles logic for the design list page (e.g., "Create New" modal).
    *   `LoginPage.ts`: Handles logic for the login/signup form toggle.
*   **Utilities:**
    *   `Api.ts`: Configures and exports instances of the auto-generated API clients, including auth token injection.
    *   `TemplateLoader.ts`: Utility for loading HTML fragments from `TemplateRegistry.html`.
    *   `types.ts`, `interfaces.ts` (potentially): Defines core TypeScript types and the new interfaces for mode components (`ISectionModeComponent`, etc.).
    *   `converters.ts`: Helper functions for mapping between API enums/structs and frontend types.
    *   `ExcalidrawWrapper.tsx`: A simple React wrapper component specifically for integrating the `Excalidraw` component, necessary due to Excalidraw's React dependency.

**Key Concepts/Responsibilities:**

*   **Client-Side Interactivity:** Makes the application dynamic after initial SSR.
*   **Component Orchestration:** Manages the lifecycle and interaction of different UI parts.
*   **API Interaction:** Communicates with the backend Go services via the JSON API gateway.
*   **DOM Manipulation:** Updates the web page based on user actions and API responses, often using templates from `TemplateRegistry.html`.
*   **State Management:** Manages UI state (e.g., current mode, loading states) within components.
*   **Composition Pattern:** Implements the view/edit logic for sections using separate, focused components managed by a container (`BaseSection`).
