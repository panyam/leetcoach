package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings" // Import strings package

	protos "github.com/panyam/leetcoach/gen/go/leetcoach/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	// *** IMPORT YOUR DSL PARSER PACKAGE HERE ***
	// Example:
	// dslparser "github.com/your_username/your_dsl_parser"
)

// SystemModelService implements the gRPC service for DSL interactions.
type SystemModelService struct {
	protos.UnimplementedSystemModelServiceServer
	BaseService
	contentSvc *ContentService // Dependency
	// Add parser instance if it has state, otherwise call package functions
	// parser *dslparser.Parser
}

// NewSystemModelService creates a new instance.
func NewSystemModelService(contentSvc *ContentService /*, parser *dslparser.Parser */) *SystemModelService {
	if contentSvc == nil {
		slog.Error("NewSystemModelService created with nil ContentService")
		panic("NewSystemModelService requires a non-nil ContentService")
	}
	return &SystemModelService{
		contentSvc: contentSvc,
		// parser: parser,
	}
}

// ValidateDSL parses the DSL content and returns validation status.
func (s *SystemModelService) ValidateDSL(ctx context.Context, req *protos.ValidateDSLRequest) (*protos.ValidateDSLResponse, error) {
	designId := req.GetDesignId()
	sectionId := req.GetSectionId()
	if designId == "" || sectionId == "" {
		return nil, status.Error(codes.InvalidArgument, "Design ID and Section ID are required")
	}
	slog.Info("Handling ValidateDSL request", "designId", designId, "sectionId", sectionId)

	// Optional: Permission check (can user view this design?)
	// _, err := s.EnsureLoggedIn(ctx) // Or check specific design view permissions
	// if err != nil {
	// 	return nil, err
	// }

	// Fetch DSL content using ContentService
	// Assuming DSL is stored in "main" for now
	dslContentBytes, err := s.contentSvc.GetContentBytes(ctx, designId, sectionId, "main")
	if err != nil {
		if errors.Is(err, ErrNoSuchEntity) {
			slog.Warn("DSL content not found for validation", "designId", designId, "sectionId", sectionId)
			// Treat non-existent content as valid? Or invalid? Let's say valid (empty).
			return &protos.ValidateDSLResponse{IsValid: true}, nil
		}
		slog.Error("Failed to get DSL content for validation", "designId", designId, "sectionId", sectionId, "error", err)
		return nil, status.Error(codes.Internal, "Failed to retrieve DSL content")
	}
	dslContent := string(dslContentBytes)

	// *** CALL YOUR DSL PARSER HERE ***
	// Replace this with the actual call to your parser package/function.
	// The parser should ideally return a specific error type that includes
	// line, column, and message information.

	// Example placeholder using a dummy function:
	parseErrors := s.dummyParseDSL(dslContent) // Replace with your actual parser call

	// **********************************

	if len(parseErrors) == 0 {
		slog.Info("DSL validation successful", "designId", designId, "sectionId", sectionId)
		return &protos.ValidateDSLResponse{IsValid: true}, nil
	} else {
		slog.Warn("DSL validation failed", "designId", designId, "sectionId", sectionId, "error_count", len(parseErrors))
		// Convert your parser errors to the proto format
		protoErrors := make([]*protos.ValidationError, len(parseErrors))
		for i, pErr := range parseErrors {
			protoErrors[i] = &protos.ValidationError{
				Line:    int32(pErr.Line),   // Adapt based on your error struct
				Column:  int32(pErr.Column), // Adapt based on your error struct
				Message: pErr.Message,       // Adapt based on your error struct
			}
		}
		return &protos.ValidateDSLResponse{IsValid: false, Errors: protoErrors}, nil
	}
}

// --- Dummy Parser Logic (Replace with your actual parser call) ---

// Dummy error structure mimicking parser output
type dummyParserError struct {
	Line    int
	Column  int
	Message string
}

// Dummy parser function - replace with your real parser integration
func (s *SystemModelService) dummyParseDSL(content string) []dummyParserError {
	var errors []dummyParserError
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.Contains(trimmedLine, "ERROR") { // Simple trigger for dummy error
			errors = append(errors, dummyParserError{
				Line:    i + 1,
				Column:  strings.Index(line, "ERROR") + 1,
				Message: fmt.Sprintf("Found forbidden word 'ERROR'"),
			})
		}
		// Add more sophisticated dummy checks if needed
	}
	// Simulate finding an unclosed brace
	if strings.Count(content, "{") > strings.Count(content, "}") {
		errors = append(errors, dummyParserError{
			Line:    len(lines), // Approximate line
			Column:  1,
			Message: "Syntax Error: Unclosed '{' detected.",
		})
	}
	return errors
}

// --- End Dummy Parser Logic ---
