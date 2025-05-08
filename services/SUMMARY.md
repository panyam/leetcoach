# ./services/ Summary

**Purpose:**

This folder contains the Go implementation of the backend gRPC services that define the core business logic and data manipulation capabilities of LeetCoach. It acts as the layer between the API gateway/frontend and the underlying data persistence mechanisms.

**Key Files/Sub-folders:**

*   `server.go`: Initializes and starts the main gRPC server. Instantiates service implementations, the `DesignStore`, the `LLMClient`, and registers all gRPC services (Design, Content, Tag, LLM, SystemModel, Admin).
*   `designs_store.go`: **Crucial component**. Encapsulates *all* filesystem interactions for designs, sections, content, and prompts located under the configured data path (e.g., `./data/designs/`). Provides methods for reading/writing metadata and content files, path generation, and deletion. It does *not* contain business logic, only filesystem operations.
*   `designs.go`: Implements the `DesignService`. Handles creation, retrieval, update, deletion of *design metadata* and manages the list/order of sections *within* a design (`design.json`). It uses `DesignStore` for all filesystem I/O related to design/section metadata files and prompt files. It orchestrates section addition/deletion/movement by manipulating the `SectionIds` list in the design metadata and interacting with `DesignStore`.
*   `content.go`: Implements the `ContentService`. Handles getting and setting the raw byte content for specific named files within a section's directory (e.g., `content.main`, `content.light.svg`, `content.dsl`). It relies on `DesignStore` (via `DesignService` or direct injection if refactored) for path generation but performs the actual file reads/writes.
*   `llm_service.go`: Implements the `LlmService`. Orchestrates calls to the LLM via the `LLMClient`. It fetches necessary context (like section/design metadata, prompts, or content) using `DesignStore` and `ContentService`, formats prompts, interacts with the `LLMClient`, and parses responses. Handles generation, review, suggestion, and default prompt generation logic.
*   `system_model_service.go`: Implements the `SystemModelService`. Handles DSL-specific actions like validation (`ValidateDSL`) and diagram generation (`GenerateDiagram` - currently stubbed). Fetches DSL content via `ContentService` and will contain the parser and generator logic.
*   `tags.go`: Implements the `TagService`. Manages tag data stored in Google Cloud Datastore via `ClientMgr`.
*   `auth.go`: Implements the `AuthService`. Handles user authentication logic, user/identity/channel creation and retrieval from Datastore via `ClientMgr`.
*   `admin.go`: Implements the `AdminService` (likely for development/testing purposes).
*   `models.go`: Defines the core Go data structures (structs like `Design`, `Section`, `User`, `Tag`, `Identity`, `Channel`, etc.) used within the backend services, including helper types like `StringMapField`. These structs represent the data as stored or manipulated internally. `Section` includes transient fields for prompts populated during reads.
*   `converters.go`: Contains helper functions to convert between the internal Go structs (`models.go`) and the Protobuf message types (`proto/leetcoach/v1/*.proto`).
*   `clientmgr.go`: Manages connections to external services, primarily Google Cloud Datastore. Provides typed wrappers (`DataStore[T]`) for interacting with specific Datastore kinds.
*   `llm/`: Sub-package containing the `LLMClient` interface and its implementations (currently OpenAI).
*   `base.go`, `gcds.go`, `idgen.go`, `constants.go`: Base service utilities, Datastore interaction helpers, ID generation logic (filesystem based), and shared constants.
*   `*_test.go`: Unit tests for the services and `DesignStore`.

**Key Concepts/Responsibilities:**

*   **gRPC Service Implementation:** Provides the concrete logic for the RPC methods defined in the `.proto` files.
*   **Business Logic:** Contains rules for creating/updating designs, adding/moving sections, handling auth flows, etc.
*   **Data Persistence Abstraction:** Uses `DesignStore` for filesystem operations and `ClientMgr` for Datastore operations, decoupling service logic from raw persistence details.
*   **LLM Orchestration:** Formats prompts, interacts with the LLM client, and processes results for various features.
*   **DSL Processing:** (Future) Will contain the parser, simulator, and diagram generator for the System Description language.
