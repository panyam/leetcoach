# ./proto/leetcoach/v1/ Summary

**Purpose:**

This folder contains the Protobuf (`.proto`) definition files that define the contracts for the gRPC services and the structure of the data messages exchanged between the backend services and the frontend gateway (and potentially other future clients). These definitions are the source of truth for the API structure.

**Key Files:**

*   `models.proto`: Defines common data structures used across services, such as `Design`, `Section` (including `SectionType` enum and prompt fields), `Tag`, `Pagination`, `User` (potentially simplified for API), etc.
*   `designs.proto`: Defines the `DesignService` interface, including RPCs for creating, getting, updating, deleting designs, and managing sections within a design (`AddSection`, `UpdateSection`, `DeleteSection`, `MoveSection`). Defines corresponding request/response messages.
*   `content.proto`: Defines the `ContentService` interface with RPCs for getting (`GetContent`) and setting (`SetContent`) named content blobs associated with a section.
*   `llm_service.proto`: Defines the `LlmService` interface with RPCs for LLM interactions like `SimpleLlmQuery`, `SuggestSections`, `GenerateTextContent`, `ReviewTextContent`, and `GenerateDefaultPrompts`. Includes messages like `SuggestedSection`.
*   `system_model_service.proto`: Defines the `SystemModelService` interface with RPCs for DSL processing: `ValidateDSL` and `GenerateDiagram` (stubbed response). Includes messages like `ValidateDSLRequest`, `ValidateDSLResponse`, `ValidationError`, `GenerateDiagramRequest`, `GenerateDiagramResponse`.
*   `tag_service.proto`: Defines the `TagService` interface for managing tags.
*   `admin_service.proto`: Defines the `AdminService` interface for development/admin operations.
*   `auth_service.proto`: (Potentially, if auth operations beyond the standard `oneauth` flow require direct API calls).

**Key Concepts/Responsibilities:**

*   **API Contract:** Defines the precise methods (RPCs), request parameters, and response structures for all backend services.
*   **Data Serialization:** Specifies how data is structured for transmission over the network (Protobuf binary format).
*   **Code Generation:** Serves as the input for code generators (`protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway`, `openapi-generator`) to create:
    *   Go service interfaces and message structs (`gen/go/`).
    *   Go gRPC server/client stubs (`gen/go/`).
    *   gRPC-Gateway reverse proxy code (`gen/go/`).
    *   OpenAPI v2 specification (`gen/openapiv2/`).
    *   TypeScript client library (`web/views/components/apiclient/`).
