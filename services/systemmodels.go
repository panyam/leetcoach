// FILE: services/system_model_service.go
package services

import (
	"context"
	"errors"
	"log/slog"

	// Assuming your parser package is here:
	// "github.com/your_repo/your_dsl_parser" // Replace with your actual parser import path

	protos "github.com/panyam/leetcoach/gen/go/leetcoach/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Define an interface matching your parser function's signature.
// Adjust this based on your parser's return type (e.g., if it returns a custom error struct).
type DSLParserFunc func(dslContent string) error

// Placeholder for your actual parser function.
// Replace this with a call to your real parser.
func placeholderParser(dslContent string) error {
	slog.Warn("Using placeholder DSL parser. Implement actual parsing logic.")
	// Simulate success or failure based on content for testing
	if dslContent == "error" {
		// Simulate a basic error - replace with actual error extraction
		return errors.New("line 5: column 10: Syntax error: unexpected token 'invalid'")
	}
	return nil // Simulate success
}

// SystemModelService implements the gRPC service for DSL interactions.
type SystemModelService struct {
	protos.UnimplementedSystemModelServiceServer
	BaseService
	contentSvc *ContentService // Dependency needed to get DSL content
	// Add your parser function here if it's not a global function
	parseDSL DSLParserFunc
}

// NewSystemModelService creates a new instance.
func NewSystemModelService(contentSvc *ContentService /*, dslParser YourParserType */) *SystemModelService {
	if contentSvc == nil {
		slog.Error("NewSystemModelService created with nil ContentService")
		panic("NewSystemModelService requires a non-nil ContentService")
	}
	return &SystemModelService{
		contentSvc: contentSvc,
		// Initialize your parser here if needed
		parseDSL: placeholderParser, // Use the placeholder for now
		// parseDSL: your_real_parser.Parse, // Example with a real parser
	}
}

// ValidateDSL implements the RPC method.
func (s *SystemModelService) ValidateDSL(ctx context.Context, req *protos.ValidateDSLRequest) (*protos.ValidateDSLResponse, error) {
	designId := req.GetDesignId()
	sectionId := req.GetSectionId()
	slog.Info("Handling ValidateDSL", "design_id", designId, "section_id", sectionId)

	if designId == "" || sectionId == "" {
		return nil, status.Error(codes.InvalidArgument, "Design ID and Section ID must be provided")
	}

	// TODO: Add permission check using BaseService.EnsureLoggedIn if needed

	// Get DSL content using ContentService
	// Assuming DSL is stored in "main" content name for system_description type
	dslBytes, err := s.contentSvc.GetContentBytes(ctx, designId, sectionId, "main")
	if err != nil {
		if errors.Is(err, ErrNoSuchEntity) {
			slog.Warn("DSL content not found for validation", "design_id", designId, "section_id", sectionId)
			// Treat as valid? Or invalid? Let's say valid empty content.
			return &protos.ValidateDSLResponse{IsValid: true}, nil
		}
		slog.Error("Failed to get DSL content for validation", "design_id", designId, "section_id", sectionId, "error", err)
		return nil, status.Error(codes.Internal, "Failed to retrieve DSL content")
	}
	dslContent := string(dslBytes)

	// Parse the DSL content
	parseErr := s.parseDSL(dslContent)

	if parseErr == nil {
		slog.Info("DSL Validation Successful", "design_id", designId, "section_id", sectionId)
		return &protos.ValidateDSLResponse{IsValid: true}, nil
	} else {
		slog.Warn("DSL Validation Failed", "design_id", designId, "section_id", sectionId, "error", parseErr)
		// --- Basic Error Parsing (Customize based on your parser's error format) ---
		validationErrors := []*protos.ValidationError{}
		// Example: Try to parse "line X: col Y: message" format
		// You'll need a more robust way to handle your specific parser errors.
		// Consider defining a custom error type in your parser package.
		errorMsg := parseErr.Error()
		// Placeholder extraction - replace with actual logic
		validationErrors = append(validationErrors, &protos.ValidationError{
			Line:    0, // Extract if possible
			Column:  0, // Extract if possible
			Message: errorMsg,
		})
		// --- End Basic Error Parsing ---

		return &protos.ValidateDSLResponse{
			IsValid: false,
			Errors:  validationErrors,
		}, nil // Return nil error for gRPC, validation failure is in the response
	}
}
