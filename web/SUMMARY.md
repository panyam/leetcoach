# ./web/ Summary

**Purpose:**

This folder contains the Go code responsible for the web server layer of the LeetCoach application. It handles incoming HTTP requests, serves static assets, hosts the gRPC-Gateway acting as the API endpoint, manages user sessions and authentication flows, and orchestrates server-side rendering of initial page views using Go templates.

**Key Files/Sub-folders:**

*   `server.go`: Defines and starts the main HTTP web server (`http.Server`). Includes basic middleware for logging and CORS (for development). It instantiates the `LCApp`.
*   `app.go`: Defines the main web application struct `LCApp`. Initializes core web components: `scs` for session management, `oneauth` for authentication handlers (local, Google, GitHub), `LCApi` (gRPC gateway wrapper), and `LCViews` (template rendering engine). Connects `oneauth` to the user storage logic (`web/user.go`). Sets up main routing.
*   `api.go`: Defines `LCApi`. Configures and runs the `grpc-gateway` reverse proxy, mapping HTTP/JSON requests to backend gRPC services defined in `services/`. Includes crucial middleware to extract the logged-in user ID from the session (`scs`) and inject it into the outgoing gRPC metadata, allowing backend services to identify the user. Implements custom error handling for gateway responses.
*   `user.go`: Implements the `oneauth.UserStore` interface (`LCAuthUser`, `GetUserByID`, `EnsureAuthUser`, `ValidateUsernamePassword`). This acts as the bridge between the `oneauth` library and the backend `AuthService` (accessed via `ClientMgr`), allowing `oneauth` to retrieve and verify user information. Includes mock user logic for development.
*   `views/`: Contains the logic for server-side rendering (SSR). See `./web/views/SUMMARY.md`.
*   `static/`: Contains static assets served directly (CSS, JS bundles, images). See `./web/static/SUMMARY.md`.

**Key Concepts/Responsibilities:**

*   **HTTP Server:** Listens for and handles incoming HTTP requests.
*   **API Gateway:** Hosts the `grpc-gateway` which translates REST/JSON API calls (`/api/v1/...`) into gRPC calls to the backend `services/`.
*   **Authentication Handling:** Integrates `oneauth` to manage login/signup routes (`/auth/...`) and OAuth callbacks.
*   **Session Management:** Uses `scs` to manage user sessions via cookies (or other stores).
*   **Middleware:** Applies middleware for logging, CORS, and injecting user authentication context into API calls.
*   **Routing:** Directs requests to the API gateway, static file server, or the SSR view renderer.
*   **SSR Orchestration:** Initializes the `LCViews` component responsible for rendering Go HTML templates.
