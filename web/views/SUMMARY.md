# ./web/views/ Summary

**Purpose:**

This folder contains the Go code specifically responsible for the Server-Side Rendering (SSR) aspect of the LeetCoach web application. It defines the data structures needed for each page template, implements the logic to fetch that data from backend services, and manages the template rendering pipeline.

**Key Files/Sub-folders:**

*   `main.go`:
    *   Defines core view context (`ViewContext`), interface (`View`), and renderer function (`ViewRenderer`).
    *   Initializes the `templar` template engine.
    *   Loads HTML templates from `./web/views/templates/`.
    *   Defines shared template functions available to all templates (e.g., `UserInfo`, `Ago`).
*   `views.go`: Sets up the main HTTP routes for different pages (e.g., `/`, `/login`, `/designs/{id}/edit`). Maps these routes to specific `View` struct implementations and their corresponding template files.
*   `HomePage.go`, `DesignEditorPage.go`, `DesignViewerPage.go`, `LoginPage.go`, `GenericPage.go`, etc.:
    *   Define Go structs that represent the data context required for a specific page or reusable component (like `Header`).
    *   Implement the `View` interface, primarily the `Load` method.
    *   The `Load` method contains the logic to:
        *   Extract necessary information from the HTTP request (path parameters, query strings, user session).
        *   Make calls to backend gRPC services (via the `ClientMgr`) to fetch the required data (e.g., fetching design details for the editor page, fetching a list of designs for the home page).
        *   Populate the fields of the view struct with the fetched data.
*   `Header.go`, `DesignList.go`, `Paginator.go`: Define structs and `Load` logic for reusable page components rendered server-side.
*   `utils.go`: Contains helper functions used during server-side rendering (e.g., `randomDesignName`).
*   `templates/`: Contains the actual Go HTML template files. See `./web/views/templates/SUMMARY.md`.
*   `components/`: Contains the frontend TypeScript components responsible for client-side interactivity. See `./web/views/components/SUMMARY.md`.

**Key Concepts/Responsibilities:**

*   **SSR Data Fetching:** Provides the server-side logic to gather all necessary data *before* rendering the initial HTML page.
*   **Page Context Structs:** Defines the shape of the data passed into the Go HTML templates.
*   **Template Engine Integration:** Manages the `templar` engine, template loading, and function maps.
*   **Route-to-View Mapping:** Connects specific URL paths to the Go code responsible for preparing and rendering that page.
