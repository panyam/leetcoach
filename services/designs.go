// FILE: ./services/designs.go
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	protos "github.com/panyam/leetcoach/gen/go/leetcoach/v1"
	// Add if using SectionMetadataToProto helper:
	// tspb "google.golang.org/protobuf/types/known/timestamppb"
)

// --- DesignService struct holds configuration and state ---
type DesignService struct {
	protos.UnimplementedDesignServiceServer
	BaseService
	clients *ClientMgr // Keep for potential auth needs

	idgen IDGen

	// Configuration / State moved from globals
	basePath string
	mutexMap sync.Map // Mutex map keyed by design ID
}

// --- NewDesignService Constructor ---
func NewDesignService(clients *ClientMgr, basePath string) *DesignService {
	if basePath == "" {
		basePath = defaultDesignsBasePath
	}
	out := &DesignService{
		clients:  clients,
		basePath: basePath,
	}
	out.idgen = IDGen{
		NextIDFunc: (&SimpleIDGen{}).NextID,
		GetID:      func(kind, id string) (*GenID, error) { return out.getDesignId(id) },
	}
	if err := ensureDir(out.basePath); err != nil {
		log.Fatalf("Could not create base designs directory '%s': %v", out.basePath, err)
	}
	return out
}

// --- Filesystem Path Helpers ---
func (s *DesignService) getDesignId(id string) (g *GenID, err error) {
	designPath := s.getDesignPath(id)
	_, err = os.Stat(designPath)
	if err == nil {
		g = &GenID{Id: id}
		return
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	slog.Error("Error checking design path existence", "id", id, "path", designPath, "error", err)
	return nil, err
}

func (s *DesignService) getDesignMutex(designId string) *sync.Mutex {
	mutex, _ := s.mutexMap.LoadOrStore(designId, &sync.Mutex{})
	return mutex.(*sync.Mutex)
}

func (s *DesignService) getDesignPath(designId string) string {
	return filepath.Join(s.basePath, designId)
}

func (s *DesignService) getDesignMetadataPath(designId string) string {
	return filepath.Join(s.getDesignPath(designId), "design.json")
}

func (s *DesignService) getSectionsBasePath(designId string) string {
	return filepath.Join(s.getDesignPath(designId), "sections")
}

func (s *DesignService) getSectionPath(designId, sectionId string) string {
	return filepath.Join(s.getSectionsBasePath(designId), sectionId)
}

func (s *DesignService) getSectionMetadataPath(designId, sectionId string) string {
	return filepath.Join(s.getSectionPath(designId, sectionId), "main.json")
}

func (s *DesignService) getContentPath(designId, sectionId, contentName string) string {
	safeName := strings.ReplaceAll(contentName, "..", "")
	safeName = strings.ReplaceAll(safeName, "/", "")
	safeName = strings.ReplaceAll(safeName, "\\", "")
	if safeName == "" {
		safeName = "default"
	}
	return filepath.Join(s.getSectionPath(designId, sectionId), fmt.Sprintf("content.%s", safeName))
}

// --- Internal Read/Write Helpers ---

// Helper to read and unmarshal design metadata (design.json).
func (s *DesignService) readDesignMetadata(designId string) (*Design, error) {
	metadataPath := s.getDesignMetadataPath(designId)
	jsonData, err := os.ReadFile(metadataPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, status.Errorf(codes.NotFound, "Design '%s' not found", designId)
		}
		slog.Error("Failed to read design metadata", "designId", designId, "path", metadataPath, "error", err)
		return nil, status.Error(codes.Internal, "Failed to read design metadata")
	}
	var metadata Design
	if err = json.Unmarshal(jsonData, &metadata); err != nil {
		slog.Error("Failed to unmarshal design metadata", "designId", designId, "path", metadataPath, "error", err)
		return nil, status.Error(codes.DataLoss, "Failed to parse design metadata")
	}
	return &metadata, nil
}

// Helper to marshal and write design metadata (design.json).
func (s *DesignService) writeDesignMetadata(designId string, metadata *Design) error {
	metadataPath := s.getDesignMetadataPath(designId)
	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal design metadata for writing", "designId", designId, "error", err)
		return status.Error(codes.Internal, "Failed to serialize design metadata")
	}
	if err = os.WriteFile(metadataPath, jsonData, 0644); err != nil {
		slog.Error("Failed to write design metadata file", "designId", designId, "path", metadataPath, "error", err)
		return status.Error(codes.Internal, "Failed to save design metadata")
	}
	return nil
}

// Helper to read and unmarshal section metadata (main.json)
func (s *DesignService) readSectionMetadata(designId, sectionId string) (*Section, error) {
	metadataPath := s.getSectionMetadataPath(designId, sectionId)
	jsonData, err := os.ReadFile(metadataPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoSuchEntity // Use custom error for consistency
		}
		return nil, fmt.Errorf("failed to read section metadata file %s: %w", metadataPath, err)
	}
	var section Section
	if err := json.Unmarshal(jsonData, &section); err != nil {
		return nil, fmt.Errorf("failed to unmarshal section metadata from %s: %w", metadataPath, err)
	}
	return &section, nil
}

// Helper to marshal and write section metadata (main.json).
func (s *DesignService) writeSectionMetadata(designId, sectionId string, sectionData *Section) error {
	metadataPath := s.getSectionMetadataPath(designId, sectionId)
	jsonData, err := json.MarshalIndent(sectionData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal section metadata %s/%s: %w", designId, sectionId, err)
	}
	if err = os.WriteFile(metadataPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write section metadata file %s: %w", metadataPath, err)
	}
	slog.Debug("Successfully wrote section metadata file", "path", metadataPath)
	return nil
}

// --- DesignService RPC Implementations ---

func (s *DesignService) CreateDesign(ctx context.Context, req *protos.CreateDesignRequest) (resp *protos.CreateDesignResponse, err error) {
	slog.Info("CreateDesign Request", "req", req)
	designProto := req.Design
	if designProto == nil {
		return nil, status.Error(codes.InvalidArgument, "Design payload cannot be nil")
	}

	ownerId, err := s.EnsureLoggedIn(ctx)
	if ENFORCE_LOGIN && err != nil {
		slog.Error("Login enforcement failed", "error", err)
		return nil, err
	}
	if !ENFORCE_LOGIN && ownerId == "" {
		ownerId = "dev_user"
	}

	designId := designProto.Id
	if designId == "" {
		slog.Debug("Design ID not provided, attempting auto-generation")
		genid, err := s.idgen.NextID("design", ownerId, time.Unix(0, 0))
		if genid == nil || err != nil {
			slog.Error("Failed to generate a unique design ID", "error", err)
			return nil, status.Error(codes.Internal, "Failed to generate unique design ID")
		}
		designId = genid.Id
		slog.Info("Generated unique Design ID", "id", designId)
	} else {
		slog.Debug("Using client-provided Design ID", "id", designId)
	}

	if strings.TrimSpace(designProto.Name) == "" {
		return nil, status.Error(codes.InvalidArgument, "Design name cannot be empty")
	}

	mutex := s.getDesignMutex(designId)
	mutex.Lock()
	defer mutex.Unlock()
	slog.Debug("Acquired mutex", "designId", designId)

	designPath := s.getDesignPath(designId)
	metadataPath := s.getDesignMetadataPath(designId)

	// Check if design *metadata* specifically exists
	if _, err := os.Stat(metadataPath); err == nil {
		slog.Warn("Design metadata already exists", "designId", designId, "path", metadataPath)
		return nil, status.Error(codes.AlreadyExists, fmt.Sprintf("Design with id '%s' already exists", designId))
	} else if !errors.Is(err, os.ErrNotExist) {
		// Error stating the metadata file (permissions?)
		slog.Error("Error checking design metadata path", "designId", designId, "path", metadataPath, "error", err)
		return nil, status.Error(codes.Internal, "Failed to check design metadata existence")
	}

	// Ensure base directory and sections subdir exist
	sectionsPath := s.getSectionsBasePath(designId)
	if err := ensureDir(sectionsPath); err != nil {
		return nil, status.Error(codes.Internal, "Failed to create design directories")
	}
	slog.Debug("Ensured directories exist", "designPath", designPath, "sectionsPath", sectionsPath)

	now := time.Now()
	metadata := Design{
		BaseModel: BaseModel{
			CreatedAt: now,
			UpdatedAt: now,
		},
		Id:          designId,
		OwnerId:     ownerId,
		Name:        designProto.Name,
		Description: designProto.Description,
		Visibility:  designProto.Visibility,
		VisibleTo:   designProto.VisibleTo,
		SectionIds:  []string{}, // Initialize empty
	}
	if metadata.Visibility == "" {
		metadata.Visibility = "private"
	}

	// Write the design metadata using the helper
	if err = s.writeDesignMetadata(designId, &metadata); err != nil {
		return nil, err // writeDesignMetadata already returns status error
	}
	slog.Info("Successfully created design metadata", "designId", designId, "path", metadataPath)

	createdDesignProto := DesignToProto(&metadata) // Static utility

	resp = &protos.CreateDesignResponse{
		Design: createdDesignProto,
	}
	return resp, nil
}

func (s *DesignService) GetDesign(ctx context.Context, req *protos.GetDesignRequest) (resp *protos.GetDesignResponse, err error) {
	slog.Info("GetDesign Request", "req", req)
	designId := req.Id
	if designId == "" {
		return nil, status.Error(codes.InvalidArgument, "Design ID cannot be empty")
	}

	metadata, err := s.readDesignMetadata(designId)
	if err != nil {
		// readDesignMetadata handles NotFound and Internal errors
		return nil, err
	}

	// TODO: Permission Checks

	designProto := DesignToProto(metadata) // Static utility
	resp = &protos.GetDesignResponse{
		Design: designProto,
	}

	// --- Handle include_section_metadata ---
	if req.IncludeSectionMetadata {
		slog.Debug("Including section metadata", "designId", designId)
		sectionMetadataProtos := make([]*protos.Section, 0, len(metadata.SectionIds))
		for i, secId := range metadata.SectionIds {
			sectionMeta, err := s.readSectionMetadata(designId, secId)
			if err != nil {
				slog.Warn("Could not read section metadata during GetDesign", "designId", designId, "sectionId", secId, "error", err)
				// Skip this section if its metadata is unreadable
				continue
			}
			// Convert sectionMeta (service struct) to proto.Section (metadata only)
			sectionProto := SectionMetadataToProto(sectionMeta) // Use the updated converter
			sectionProto.Order = uint32(i)                      // Add order based on index
			sectionMetadataProtos = append(sectionMetadataProtos, sectionProto)
		}
		resp.SectionsMetadata = sectionMetadataProtos
		slog.Debug("Finished fetching section metadata", "designId", designId, "count", len(resp.SectionsMetadata))
	}
	// ----------------------------------------

	slog.Info("Successfully retrieved design", "designId", designId)
	return resp, nil
}

func (s *DesignService) GetSection(ctx context.Context, req *protos.GetSectionRequest) (resp *protos.Section, err error) {
	designId := req.DesignId
	sectionId := req.SectionId
	if designId == "" || sectionId == "" {
		return nil, status.Error(codes.InvalidArgument, "Design ID and Section ID must be provided")
	}
	slog.Info("GetSection Request", "designId", designId, "sectionId", sectionId)

	// 1. Check design exists (and maybe permissions later)
	// Reading section metadata implicitly checks if section dir exists.
	// Optional: Explicitly check design metadata first if needed for permissions.
	// designMeta, err := s.readDesignMetadata(designId)
	// if err != nil {
	// 	return nil, err
	// }
	// // TODO: Permission Check based on designMeta

	// 2. Read section metadata (main.json)
	sectionData, err := s.readSectionMetadata(designId, sectionId)
	if err != nil {
		if errors.Is(err, ErrNoSuchEntity) {
			slog.Warn("Section metadata (main.json) not found", "designId", designId, "sectionId", sectionId)
			return nil, status.Errorf(codes.NotFound, "Section with id '%s' not found in design '%s'", sectionId, designId)
		}
		slog.Error("Failed to read section metadata", "designId", designId, "sectionId", sectionId, "error", err)
		return nil, status.Error(codes.Internal, "Failed to read section metadata")
	}

	// 3. Find order (if needed for response proto - the proto requires it)
	designMeta, err := s.readDesignMetadata(designId) // Need to read design meta for order
	if err != nil {
		slog.Error("Failed to read design metadata to determine section order", "designId", designId, "sectionId", sectionId, "error", err)
		// Return section without order, or return error? Let's return without order for now.
		resp = SectionMetadataToProto(sectionData)
	} else {
		order := uint32(0) // Default order
		found := false
		for i, id := range designMeta.SectionIds {
			if id == sectionId {
				order = uint32(i)
				found = true
				break
			}
		}
		if !found {
			slog.Warn("Section ID found in file system but not in design metadata's SectionIds list", "designId", designId, "sectionId", sectionId)
			// Section exists but isn't properly linked, still return metadata but maybe log warning.
		}
		resp = SectionMetadataToProto(sectionData)
		resp.Order = order // Set the order
	}

	slog.Info("Successfully retrieved section metadata", "designId", designId, "sectionId", sectionId)
	return resp, nil
}

func (s *DesignService) ListDesigns(ctx context.Context, req *protos.ListDesignsRequest) (resp *protos.ListDesignsResponse, err error) {
	// This implementation remains largely the same as it only reads design.json
	// files from the top level of each design directory.
	slog.Info("ListDesigns Request", "req", req)

	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		slog.Error("Failed to read designs directory", "path", s.basePath, "error", err)
		return nil, status.Error(codes.Internal, "Failed to list designs")
	}

	var allMetadata []*Design
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		designId := entry.Name()
		// Use the helper to read metadata, handling errors internally
		metadata, err := s.readDesignMetadata(designId)
		if err != nil {
			// readDesignMetadata logs errors, just continue if one is unreadable
			slog.Warn("Skipping design due to metadata read error during list", "designId", designId, "error", err)
			continue
		}
		// Optional: Check if metadata.Id matches directory name for sanity
		if metadata.Id != designId {
			slog.Warn("Mismatch between directory name and metadata Id", "dirName", designId, "metadataId", metadata.Id)
			continue
		}
		allMetadata = append(allMetadata, metadata)
	}
	slog.Debug("Collected metadata for designs", "count", len(allMetadata))

	// Filtering logic
	var filteredMetadata []*Design
	loggedInUserId, _ := s.EnsureLoggedIn(ctx) // Ignore error if not enforcing login
	for _, meta := range allMetadata {
		isOwner := (!ENFORCE_LOGIN && meta.OwnerId == "dev_user") || (ENFORCE_LOGIN && loggedInUserId == meta.OwnerId)
		isVisiblePublic := meta.Visibility == "public"

		// Apply filters
		if req.OwnerId != "" && meta.OwnerId != req.OwnerId {
			continue // Filter by owner
		}
		if req.LimitToPublic && !isVisiblePublic {
			continue // Filter by public visibility
		}
		if !isVisiblePublic && !isOwner {
			// If not public and not owner, filter out (unless specific owner was requested)
			if req.OwnerId == "" { // Don't show private designs of others if listing all
				// continue // This logic might need refinement based on exact requirements
			}
		}
		filteredMetadata = append(filteredMetadata, meta)
	}
	slog.Debug("Filtered metadata for designs", "count", len(filteredMetadata))

	// Sorting logic (remains the same)
	sort.Slice(filteredMetadata, func(i, j int) bool {
		m1 := filteredMetadata[i]
		m2 := filteredMetadata[j]
		switch req.OrderBy {
		case "recent", "":
			return m1.UpdatedAt.After(m2.UpdatedAt)
		case "name":
			return strings.ToLower(m1.Name) < strings.ToLower(m2.Name)
		case "created":
			return m1.CreatedAt.After(m2.CreatedAt)
		default:
			return m1.UpdatedAt.After(m2.UpdatedAt)
		}
	})
	slog.Debug("Sorted metadata")

	// Pagination logic (remains the same)
	totalResults := len(filteredMetadata)
	start := 0
	end := totalResults
	pageSize := 20 // Default page size
	hasMore := false

	if req.Pagination != nil {
		if req.Pagination.PageSize > 0 {
			pageSize = int(req.Pagination.PageSize)
		}
		start = max(int(req.Pagination.PageOffset), 0)
		if start >= totalResults {
			filteredMetadata = []*Design{} // Out of bounds
			end = start                    // Should be start?
		} else {
			end = start + pageSize
			if end >= totalResults {
				end = totalResults // Clamp to end
			} else {
				hasMore = true // More pages exist
			}
			filteredMetadata = filteredMetadata[start:end] // Slice the results
		}
	}
	slog.Debug("Applied pagination", "offset", start, "limit", pageSize, "actualCount", len(filteredMetadata), "hasMore", hasMore)

	finalProtos := make([]*protos.Design, len(filteredMetadata))
	for i, meta := range filteredMetadata {
		finalProtos[i] = DesignToProto(meta) // Use converter
	}

	resp = &protos.ListDesignsResponse{
		Designs: finalProtos,
		Pagination: &protos.PaginationResponse{
			HasMore:      hasMore,
			TotalResults: int32(totalResults),
			// NextPageKey/Offset not implemented for simple FS listing
		},
	}

	slog.Info("Successfully listed designs", "request", req, "responseCount", len(resp.Designs))
	return resp, nil
}

func (s *DesignService) AddSection(ctx context.Context, req *protos.AddSectionRequest) (resp *protos.Section, err error) {
	sectionProto := req.Section
	if sectionProto == nil {
		return nil, status.Error(codes.InvalidArgument, "Section payload cannot be nil")
	}
	designId := sectionProto.GetDesignId() // Get designId from the section proto

	if designId == "" {
		return nil, status.Error(codes.InvalidArgument, "Design ID must be provided within the section payload")
	}
	if sectionProto.Type == protos.SectionType_SECTION_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "Section type must be specified")
	}
	slog.Info("AddSection Request", "designId", designId, "type", sectionProto.Type, "relativeTo", req.RelativeSectionId, "position", req.Position)

	ownerId, err := s.EnsureLoggedIn(ctx)
	if ENFORCE_LOGIN && err != nil {
		return nil, err
	}

	mutex := s.getDesignMutex(designId)
	mutex.Lock()
	defer mutex.Unlock()
	slog.Debug("Acquired mutex for AddSection", "designId", designId)

	// 1. Read design metadata & check permissions
	metadata, err := s.readDesignMetadata(designId)
	if err != nil {
		return nil, err // Handles NotFound etc.
	}
	if ENFORCE_LOGIN && ownerId != metadata.OwnerId {
		slog.Warn("Permission denied for AddSection", "designId", designId, "user", ownerId)
		return nil, status.Error(codes.PermissionDenied, "User cannot add sections to this design")
	}

	// 2. Generate new Section ID
	genid, err := s.idgen.NextID("section", ownerId, time.Unix(0, 0))
	if err != nil || genid == nil {
		slog.Error("Failed to generate unique section ID", "designId", designId, "error", err)
		return nil, status.Error(codes.Internal, "Failed to generate section ID")
	}
	newSectionId := genid.Id
	slog.Info("Generated new section ID", "designId", designId, "newSectionId", newSectionId)

	// 3. Prepare new Section metadata struct (NO CONTENT)
	now := time.Now()
	newSectionMetadata := SectionMetadataFromProto(sectionProto) // Use metadata converter
	newSectionMetadata.Id = newSectionId
	newSectionMetadata.DesignId = designId // Ensure design ID is set
	newSectionMetadata.CreatedAt = now
	newSectionMetadata.UpdatedAt = now
	// Content-related fields (ContentType, Format) from proto are ignored here,
	// they should be set via ContentService.SetContent later if needed.

	// 4. Determine insertion index
	insertIndex := len(metadata.SectionIds)
	if req.RelativeSectionId != "" {
		found := false
		for i, id := range metadata.SectionIds {
			if id == req.RelativeSectionId {
				if req.Position == protos.PositionType_POSITION_TYPE_BEFORE {
					insertIndex = i
				} else { // AFTER or unspecified defaults to AFTER
					insertIndex = i + 1
				}
				found = true
				break
			}
		}
		if !found {
			slog.Warn("RelativeSectionId not found during AddSection, appending to end", "designId", designId, "relativeId", req.RelativeSectionId)
			insertIndex = len(metadata.SectionIds)
		}
	} else if req.Position != protos.PositionType_POSITION_TYPE_END {
		slog.Warn("No RelativeSectionId provided but position is not END, appending to end", "designId", designId, "position", req.Position)
		insertIndex = len(metadata.SectionIds)
	} // If position is END, insertIndex remains len(...)

	// 5. Create section directory and write metadata file (main.json)
	sectionPath := s.getSectionPath(designId, newSectionId)
	if err = ensureDir(sectionPath); err != nil {
		slog.Error("Failed to create section directory", "path", sectionPath, "error", err)
		return nil, status.Error(codes.Internal, "Failed to create section directory")
	}
	if err = s.writeSectionMetadata(designId, newSectionId, newSectionMetadata); err != nil {
		// Attempt cleanup of directory? Maybe not critical here.
		return nil, err // writeSectionMetadata returns status error
	}
	slog.Info("Successfully wrote new section metadata file", "path", s.getSectionMetadataPath(designId, newSectionId))

	// 6. Update and write design metadata
	metadata.SectionIds = append(metadata.SectionIds[:insertIndex], append([]string{newSectionId}, metadata.SectionIds[insertIndex:]...)...)
	metadata.UpdatedAt = now // Update design timestamp
	if err = s.writeDesignMetadata(designId, metadata); err != nil {
		// Attempt to clean up orphaned section directory+metadata file?
		slog.Error("Failed to write updated design metadata after adding section ID", "designId", designId, "error", err)
		_ = os.RemoveAll(sectionPath) // Best effort cleanup
		return nil, status.Error(codes.Internal, "Failed to update design metadata after adding section")
	}
	slog.Info("Successfully updated design metadata with new section ID", "designId", designId)

	// 7. Prepare and return response proto (metadata only, with order)
	resp = SectionMetadataToProto(newSectionMetadata)
	resp.Order = uint32(insertIndex) // Add the calculated order
	return resp, nil
}

func (s *DesignService) UpdateSection(ctx context.Context, req *protos.UpdateSectionRequest) (resp *protos.Section, err error) {
	sectionUpdatesProto := req.Section
	if sectionUpdatesProto == nil {
		return nil, status.Error(codes.InvalidArgument, "Section update payload cannot be nil")
	}
	designId := sectionUpdatesProto.DesignId
	sectionId := sectionUpdatesProto.Id
	updateMask := req.UpdateMask

	if designId == "" || sectionId == "" {
		return nil, status.Error(codes.InvalidArgument, "Design ID and Section ID must be provided in section payload")
	}
	if updateMask == nil || len(updateMask.Paths) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Update mask must be provided for section update")
	}
	slog.Info("UpdateSection Request", "designId", designId, "sectionId", sectionId, "mask", updateMask.Paths)

	ownerId, err := s.EnsureLoggedIn(ctx)
	if ENFORCE_LOGIN && err != nil {
		return nil, err
	}

	mutex := s.getDesignMutex(designId)
	mutex.Lock()
	defer mutex.Unlock()
	slog.Debug("Acquired mutex for UpdateSection", "designId", designId, "sectionId", sectionId)

	// 1. Check design exists and user has permission
	designMeta, err := s.readDesignMetadata(designId)
	if err != nil {
		return nil, err
	}
	if ENFORCE_LOGIN && ownerId != designMeta.OwnerId {
		slog.Warn("Permission denied for UpdateSection", "designId", designId, "user", ownerId)
		return nil, status.Error(codes.PermissionDenied, "User cannot update sections in this design")
	}

	// 2. Check section exists within the design's list and get order
	sectionOrder := uint32(0)
	found := false
	for i, id := range designMeta.SectionIds {
		if id == sectionId {
			sectionOrder = uint32(i)
			found = true
			break
		}
	}
	if !found {
		slog.Warn("Section ID not found in design metadata during update", "designId", designId, "sectionId", sectionId)
		return nil, status.Errorf(codes.NotFound, "Section '%s' not found in design '%s'", sectionId, designId)
	}

	// 3. Read existing section metadata (main.json)
	sectionData, err := s.readSectionMetadata(designId, sectionId)
	if err != nil {
		if errors.Is(err, ErrNoSuchEntity) {
			slog.Error("Inconsistency: Section ID in design metadata but main.json not found", "designId", designId, "sectionId", sectionId)
			return nil, status.Errorf(codes.Internal, "Section '%s' metadata file not found despite being listed in design '%s'", sectionId, designId)
		}
		slog.Error("Failed to read section metadata for update", "designId", designId, "sectionId", sectionId, "error", err)
		return nil, status.Error(codes.Internal, "Failed to read section metadata for update")
	}

	// 4. Apply updates *only* to metadata fields based on mask
	updated := false
	for _, path := range updateMask.Paths {
		switch path {
		case "title":
			newVal := sectionUpdatesProto.GetTitle()
			if sectionData.Title != newVal {
				if strings.TrimSpace(newVal) == "" {
					slog.Warn("Attempted to update section title to empty, ignoring", "sectionId", sectionId)
					continue // Or return InvalidArgument error
				}
				sectionData.Title = newVal
				updated = true
			}
		// Ignore content paths ("text_content", "drawing_content", "plot_content")
		// Ignore format/contentType for now unless explicitly handled
		default:
			slog.Warn("Unsupported path in update mask for UpdateSection", "path", path)
			// Return InvalidArgument error?
		}
	}

	// 5. Write updated section metadata (main.json) if changed
	if updated {
		slog.Info("Section metadata changed, saving...", "designId", designId, "sectionId", sectionId)
		sectionData.UpdatedAt = time.Now()
		if err = s.writeSectionMetadata(designId, sectionId, sectionData); err != nil {
			return nil, err // writeSectionMetadata returns status error
		}

		// 6. Update design metadata timestamp (only if section metadata was updated)
		designMeta.UpdatedAt = sectionData.UpdatedAt
		if err = s.writeDesignMetadata(designId, designMeta); err != nil {
			// Log error, but proceed as section metadata *was* saved.
			slog.Error("Failed to update design metadata timestamp after section update", "designId", designId, "error", err)
		}
	} else {
		slog.Info("No changes detected for section metadata based on update mask", "designId", designId, "sectionId", sectionId)
	}

	// 7. Return the potentially updated section metadata
	resp = SectionMetadataToProto(sectionData)
	resp.Order = sectionOrder // Include the order
	return resp, nil
}

func (s *DesignService) DeleteSection(ctx context.Context, req *protos.DeleteSectionRequest) (resp *protos.DeleteSectionResponse, err error) {
	designId := req.DesignId
	sectionId := req.SectionId

	if designId == "" || sectionId == "" {
		return nil, status.Error(codes.InvalidArgument, "Design ID and Section ID must be provided")
	}
	slog.Info("DeleteSection Request", "designId", designId, "sectionId", sectionId)

	ownerId, err := s.EnsureLoggedIn(ctx)
	if ENFORCE_LOGIN && err != nil {
		return nil, err
	}

	mutex := s.getDesignMutex(designId)
	mutex.Lock()
	defer mutex.Unlock()
	slog.Debug("Acquired mutex for DeleteSection", "designId", designId, "sectionId", sectionId)

	// 1. Read design metadata & check permissions
	metadata, err := s.readDesignMetadata(designId)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			slog.Warn("Design not found during DeleteSection", "designId", designId)
			return &protos.DeleteSectionResponse{}, nil // Design gone, section implicitly gone
		}
		return nil, err // Other read errors
	}
	if ENFORCE_LOGIN && ownerId != metadata.OwnerId {
		slog.Warn("Permission denied for DeleteSection", "designId", designId, "user", ownerId)
		return nil, status.Error(codes.PermissionDenied, "User cannot delete sections from this design")
	}

	// 2. Find and remove section ID from metadata list
	found := false
	originalLen := len(metadata.SectionIds)
	newSectionIds := make([]string, 0, originalLen-1)
	for _, id := range metadata.SectionIds {
		if id == sectionId {
			found = true
		} else {
			newSectionIds = append(newSectionIds, id)
		}
	}

	if !found {
		slog.Warn("Section ID not found in metadata during DeleteSection (idempotent)", "designId", designId, "sectionId", sectionId)
		// Section doesn't exist in the list, check if the directory exists and remove it anyway
		sectionPath := s.getSectionPath(designId, sectionId)
		if err := os.RemoveAll(sectionPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Error("Failed to remove potentially orphaned section directory", "path", sectionPath, "error", err)
			// Don't fail the request, just log.
		}
		return &protos.DeleteSectionResponse{}, nil
	}

	// 3. Update and write metadata *first* if ID was removed
	if len(newSectionIds) < originalLen {
		metadata.SectionIds = newSectionIds
		metadata.UpdatedAt = time.Now()
		if err = s.writeDesignMetadata(designId, metadata); err != nil {
			slog.Error("Failed to write updated design metadata after removing section ID", "designId", designId, "error", err)
			// Don't delete the section directory if metadata update failed
			return nil, status.Error(codes.Internal, "Failed to update design metadata before deleting section")
		}
		slog.Info("Successfully updated design metadata removing section ID", "designId", designId, "sectionId", sectionId)
	}

	// 4. Delete section *directory*
	sectionPath := s.getSectionPath(designId, sectionId)
	err = os.RemoveAll(sectionPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		// Log the error but consider the operation successful as metadata was updated
		slog.Error("Failed to delete section directory, but metadata was updated", "designId", designId, "sectionPath", sectionPath, "error", err)
	} else {
		slog.Info("Successfully deleted section directory (or it was already gone)", "path", sectionPath)
	}

	return &protos.DeleteSectionResponse{}, nil
}

func (s *DesignService) MoveSection(ctx context.Context, req *protos.MoveSectionRequest) (resp *protos.MoveSectionResponse, err error) {
	designId := req.DesignId
	sectionIdToMove := req.SectionId
	relativeSectionId := req.RelativeSectionId
	position := req.Position

	if designId == "" || sectionIdToMove == "" {
		return nil, status.Error(codes.InvalidArgument, "Design ID and Section ID to move must be provided")
	}
	if position == protos.PositionType_POSITION_TYPE_UNSPECIFIED || position == protos.PositionType_POSITION_TYPE_END {
		return nil, status.Error(codes.InvalidArgument, "Position must be BEFORE or AFTER for move operation")
	}
	if relativeSectionId == "" {
		return nil, status.Error(codes.InvalidArgument, "Relative Section ID must be provided for BEFORE/AFTER position")
	}
	if sectionIdToMove == relativeSectionId {
		slog.Info("MoveSection requested for same source and relative ID, no change needed.", "designId", designId, "sectionId", sectionIdToMove)
		return &protos.MoveSectionResponse{}, nil // Moving relative to itself is a no-op
	}
	slog.Info("MoveSection Request", "designId", designId, "sectionId", sectionIdToMove, "relativeTo", relativeSectionId, "position", position)

	ownerId, err := s.EnsureLoggedIn(ctx)
	if ENFORCE_LOGIN && err != nil {
		return nil, err
	}

	mutex := s.getDesignMutex(designId)
	mutex.Lock()
	defer mutex.Unlock()
	slog.Debug("Acquired mutex for MoveSection", "designId", designId, "sectionId", sectionIdToMove)

	metadata, err := s.readDesignMetadata(designId)
	if err != nil {
		return nil, err
	}
	if ENFORCE_LOGIN && ownerId != metadata.OwnerId {
		slog.Warn("Permission denied for MoveSection", "designId", designId, "user", ownerId)
		return nil, status.Error(codes.PermissionDenied, "User cannot move sections in this design")
	}

	// --- Refined Index Finding and Slice Manipulation ---
	currentIndex := -1
	relativeIndex := -1
	originalIds := metadata.SectionIds // Keep original for comparison later

	for i, id := range originalIds {
		if id == sectionIdToMove {
			currentIndex = i
		}
		if id == relativeSectionId {
			relativeIndex = i
		}
	}

	if currentIndex == -1 {
		return nil, status.Errorf(codes.NotFound, "Section to move ('%s') not found in design '%s'", sectionIdToMove, designId)
	}
	if relativeIndex == -1 {
		return nil, status.Errorf(codes.NotFound, "Relative section ('%s') not found in design '%s'", relativeSectionId, designId)
	}

	// 1. Create a new slice *without* the item being moved.
	tempSectionIds := make([]string, 0, len(originalIds)-1)
	itemToMove := originalIds[currentIndex]
	for i, id := range originalIds {
		if i != currentIndex {
			tempSectionIds = append(tempSectionIds, id)
		}
	}

	// 2. Find the index of the relative item *in the temporary slice*.
	newRelativeIndex := -1
	for i, id := range tempSectionIds {
		if id == relativeSectionId {
			newRelativeIndex = i
			break
		}
	}
	// This check should ideally not fail if relativeIndex was valid before, but safeguard.
	if newRelativeIndex == -1 {
		slog.Error("Internal error: Relative section ID disappeared after removing item to move", "designId", designId, "relativeId", relativeSectionId)
		return nil, status.Error(codes.Internal, "Error processing move operation: relative ID lost")
	}

	// 3. Calculate the insertion index in the temporary slice.
	insertIndex := 0
	if position == protos.PositionType_POSITION_TYPE_BEFORE {
		insertIndex = newRelativeIndex
	} else { // AFTER
		insertIndex = newRelativeIndex + 1
	}
	// Note: insertIndex can range from 0 to len(tempSectionIds) inclusive.

	// 4. Perform the insertion to create the final ordered slice.
	finalSectionIds := make([]string, 0, len(originalIds))
	finalSectionIds = append(finalSectionIds, tempSectionIds[:insertIndex]...)
	finalSectionIds = append(finalSectionIds, itemToMove)
	finalSectionIds = append(finalSectionIds, tempSectionIds[insertIndex:]...)

	// 5. Check if the order actually changed compared to the original.
	orderChanged := false
	if len(finalSectionIds) != len(originalIds) { // Sanity check, lengths should match
		orderChanged = true
		slog.Warn("Length mismatch after move, should not happen", "designId", designId)
	} else {
		for i := range finalSectionIds {
			if finalSectionIds[i] != originalIds[i] {
				orderChanged = true
				break
			}
		}
	}

	if !orderChanged {
		slog.Info("MoveSection resulted in no effective change to order.", "designId", designId, "sectionId", sectionIdToMove)
		return &protos.MoveSectionResponse{}, nil // No-op
	}

	// 6. Update and write metadata.
	metadata.SectionIds = finalSectionIds // Assign the newly calculated slice
	metadata.UpdatedAt = time.Now()
	if err = s.writeDesignMetadata(designId, metadata); err != nil { // Call the write helper
		slog.Error("Failed to write updated design metadata after moving section", "designId", designId, "error", err)
		return nil, status.Error(codes.Internal, "Failed to update design metadata after move")
	}
	slog.Info("Successfully updated design metadata after moving section", "designId", designId, "sectionId", sectionIdToMove, "newOrder", finalSectionIds)

	return &protos.MoveSectionResponse{}, nil
}

func (s *DesignService) UpdateDesign(ctx context.Context, req *protos.UpdateDesignRequest) (resp *protos.UpdateDesignResponse, err error) {
	// Implementation remains the same as it only modifies design.json
	slog.Info("UpdateDesign Request", "req", req)
	designId := req.Design.GetId()
	updateMask := req.UpdateMask

	if designId == "" {
		return nil, status.Error(codes.InvalidArgument, "Design ID cannot be empty in update request")
	}
	if req.Design == nil {
		return nil, status.Error(codes.InvalidArgument, "Design payload cannot be nil in update request")
	}
	if updateMask == nil || len(updateMask.Paths) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Update mask must be provided with at least one field path")
	}

	validPaths := map[string]bool{
		"name":        true,
		"description": true,
		"visibility":  true,
		"visible_to":  true,
		// "section_ids" is typically NOT updated via UpdateDesign, use Move/Add/DeleteSection
	}
	protoToJsonPathMap := map[string]string{
		"name":        "Name",
		"description": "Description",
		"visibility":  "Visibility",
		"visible_to":  "VisibleTo",
	}

	for _, path := range updateMask.Paths {
		// Allow paths like "design.name" or just "name" from FieldMask helper
		effectivePath := strings.TrimPrefix(path, "design.")
		if !validPaths[effectivePath] {
			slog.Warn("Invalid path in update mask for UpdateDesign", "path", path)
			return nil, status.Errorf(codes.InvalidArgument, "Update mask contains invalid or unsupported path for UpdateDesign: %s", path)
		}
	}

	mutex := s.getDesignMutex(designId)
	mutex.Lock()
	defer mutex.Unlock()
	slog.Debug("Acquired mutex for UpdateDesign", "designId", designId)

	metadata, err := s.readDesignMetadata(designId)
	if err != nil {
		return nil, err // Handles NotFound etc.
	}

	loggedInUserId, err := s.EnsureLoggedIn(ctx)
	if ENFORCE_LOGIN {
		if err != nil {
			return nil, err
		}
		if loggedInUserId != metadata.OwnerId {
			slog.Warn("Permission denied for UpdateDesign", "designId", designId, "loggedInUser", loggedInUserId, "owner", metadata.OwnerId)
			return nil, status.Error(codes.PermissionDenied, "User does not have permission to update this design")
		}
	}

	updated := false
	for _, path := range updateMask.Paths {
		effectivePath := strings.TrimPrefix(path, "design.")
		jsonField := protoToJsonPathMap[effectivePath] // Assumes map contains all validPaths

		switch jsonField {
		case "Name":
			newValue := req.Design.Name
			if strings.TrimSpace(newValue) == "" {
				return nil, status.Error(codes.InvalidArgument, "Design name cannot be updated to empty")
			}
			if metadata.Name != newValue {
				metadata.Name = newValue
				updated = true
			}
		case "Description":
			newValue := req.Design.Description
			if metadata.Description != newValue {
				metadata.Description = newValue
				updated = true
			}
		case "Visibility":
			newValue := req.Design.Visibility
			if metadata.Visibility != newValue {
				metadata.Visibility = newValue
				updated = true
			}
		case "VisibleTo":
			newValue := req.Design.VisibleTo
			// Make a copy and sort for comparison to handle order differences
			currentSorted := make([]string, len(metadata.VisibleTo))
			copy(currentSorted, metadata.VisibleTo)
			sort.Strings(currentSorted)
			newSorted := make([]string, len(newValue))
			copy(newSorted, newValue)
			sort.Strings(newSorted)

			if !equalStringSlices(currentSorted, newSorted) {
				metadata.VisibleTo = newValue // Store the original unsorted value from request
				updated = true
			}
		default:
			slog.Warn("Encountered unhandled valid path during UpdateDesign apply", "path", path)
		}
	}

	if updated {
		metadata.UpdatedAt = time.Now()
		slog.Info("Design metadata changed, attempting save", "designId", designId)
		if err = s.writeDesignMetadata(designId, metadata); err != nil {
			return nil, err
		}
		slog.Info("Successfully updated design metadata file", "designId", designId)
	} else {
		slog.Info("No changes detected for design based on update mask", "designId", designId)
	}

	updatedProto := DesignToProto(metadata)
	resp = &protos.UpdateDesignResponse{
		Design: updatedProto,
	}
	return resp, nil
}

func (s *DesignService) DeleteDesign(ctx context.Context, req *protos.DeleteDesignRequest) (resp *protos.DeleteDesignResponse, err error) {
	// Implementation remains the same as it operates on the design directory level
	slog.Info("DeleteDesign Request", "req", req)
	designId := req.Id

	if designId == "" {
		return nil, status.Error(codes.InvalidArgument, "Design ID cannot be empty for delete request")
	}

	mutex := s.getDesignMutex(designId)
	mutex.Lock()
	defer mutex.Unlock()
	slog.Debug("Acquired mutex for DeleteDesign", "designId", designId)

	// Check permissions *before* deleting
	metadata, err := s.readDesignMetadata(designId)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			slog.Warn("Design not found for delete (idempotent)", "designId", designId)
			return &protos.DeleteDesignResponse{}, nil // Already deleted
		}
		return nil, err // Other read errors
	}

	loggedInUserId, err := s.EnsureLoggedIn(ctx)
	if ENFORCE_LOGIN {
		if err != nil {
			return nil, err
		}
		if loggedInUserId != metadata.OwnerId {
			slog.Warn("Permission denied for DeleteDesign", "designId", designId, "loggedInUser", loggedInUserId, "owner", metadata.OwnerId)
			return nil, status.Error(codes.PermissionDenied, "User does not have permission to delete this design")
		}
	}

	designPath := s.getDesignPath(designId)
	slog.Info("Attempting to delete design directory", "designId", designId, "path", designPath)
	err = os.RemoveAll(designPath)
	if err != nil {
		// Check if it was already gone
		if _, statErr := os.Stat(designPath); errors.Is(statErr, os.ErrNotExist) {
			slog.Warn("Design directory already gone during delete", "designId", designId, "path", designPath)
			return &protos.DeleteDesignResponse{}, nil
		}
		// Other deletion error
		slog.Error("Failed to delete design directory", "designId", designId, "path", designPath, "error", err)
		return nil, status.Error(codes.Internal, "Failed to delete design")
	}

	slog.Info("Successfully deleted design", "designId", designId, "path", designPath)

	resp = &protos.DeleteDesignResponse{}
	return resp, nil
}

func (s *DesignService) GetDesigns(ctx context.Context, req *protos.GetDesignsRequest) (resp *protos.GetDesignsResponse, err error) {
	slog.Warn("GetDesigns called, which is inefficient for filesystem backend")
	return nil, status.Error(codes.Unimplemented, "GetDesigns is not efficiently implemented for the filesystem backend. Use GetDesign for individual IDs.")
}

// --- Utility Functions ---

func ensureDir(path string) error {
	err := os.MkdirAll(path, 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		slog.Error("Failed to create directory", "path", path, "error", err)
		return err // Return the error to the caller
	}
	return nil
}

// equalStringSlices checks if two sorted string slices are equal.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
