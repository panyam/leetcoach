// FILE: ./services/designs_test.go
package services

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	protos "github.com/panyam/leetcoach/gen/go/leetcoach/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	tspb "google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/go-cmp/cmp"
)

// --- Test Helpers (Modified for new structure) ---

func testContextWithUser(userId string) context.Context {
	md := metadata.Pairs("loggedinuserid", userId)
	ctx := metadata.NewIncomingContext(context.Background(), md)
	return ctx
}

func newTestDesignService(t *testing.T) (*DesignService, string) {
	t.Helper()
	tempDir := t.TempDir()
	t.Logf("Using temp directory for test service: %s", tempDir)
	// Assume ClientMgr is nil for these filesystem tests
	service := NewDesignService(nil, tempDir)
	t.Cleanup(func() {
		service.mutexMap = sync.Map{} // Clear mutex map after test
	})
	return service, tempDir
}

// Helper to directly create a design metadata file (design.json)
func createDesignDirectly(t *testing.T, basePath string, metadata Design) {
	t.Helper()
	designPath := filepath.Join(basePath, metadata.Id)
	metadataPath := filepath.Join(designPath, "design.json")
	sectionsPath := filepath.Join(designPath, "sections")

	err := os.MkdirAll(sectionsPath, 0755) // Ensure sections dir exists too
	require.NoError(t, err, "Failed to create directories for direct design creation")

	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	require.NoError(t, err, "Failed to marshal design metadata for direct creation")

	err = os.WriteFile(metadataPath, jsonData, 0644)
	require.NoError(t, err, "Failed to write design metadata file for direct creation")
}

// Helper to directly create section metadata (main.json)
func createSectionMetadataDirectly(t *testing.T, basePath, designId, sectionId string, sectionMeta Section) {
	t.Helper()
	sectionDir := filepath.Join(basePath, designId, "sections", sectionId)
	err := os.MkdirAll(sectionDir, 0755) // Ensure section dir exists
	require.NoError(t, err, "Failed to create section directory for direct metadata creation")

	metadataPath := filepath.Join(sectionDir, "main.json")
	// Ensure ID is set correctly in the struct before marshaling if it's derived
	sectionMeta.Id = sectionId
	sectionMeta.DesignId = designId
	jsonData, err := json.MarshalIndent(sectionMeta, "", "  ")
	require.NoError(t, err, "Failed to marshal section metadata for direct creation")
	err = os.WriteFile(metadataPath, jsonData, 0644)
	require.NoError(t, err, "Failed to write section metadata file (main.json)")
}

// Helper to read design metadata directly from file (design.json)
func readDesignMetadataDirectly(t *testing.T, basePath string, designId string) *Design {
	t.Helper()
	metadataPath := filepath.Join(basePath, designId, "design.json")
	jsonData, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		require.NoError(t, err, "Failed to read design metadata file for verification")
	}

	var metadata Design
	err = json.Unmarshal(jsonData, &metadata)
	require.NoError(t, err, "Failed to unmarshal design metadata for verification")
	return &metadata
}

// Helper to read section metadata directly from file (main.json)
func readSectionMetadataDirectly(t *testing.T, basePath, designId, sectionId string) *Section {
	t.Helper()
	metadataPath := filepath.Join(basePath, designId, "sections", sectionId, "main.json")
	jsonData, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		require.NoError(t, err, "Failed to read section metadata file (main.json)")
	}
	var sectionData Section
	err = json.Unmarshal(jsonData, &sectionData)
	require.NoError(t, err, "Failed to unmarshal section metadata (main.json)")
	return &sectionData
}

// --- DesignService Test Cases ---

func TestCreateDesign(t *testing.T) {
	service, tempDir := newTestDesignService(t)
	ctx := testContextWithUser("test-creator")

	t.Run("Create with Provided ID", func(t *testing.T) {
		req := &protos.CreateDesignRequest{
			Design: &protos.Design{
				Id:          "test-create-1",
				Name:        "Test Create Design 1",
				Description: "Description 1",
				Visibility:  "public",
			},
		}
		resp, err := service.CreateDesign(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Design)
		assert.Equal(t, "test-create-1", resp.Design.Id)
		assert.Equal(t, "Test Create Design 1", resp.Design.Name)
		assert.Equal(t, "Description 1", resp.Design.Description)
		assert.Equal(t, "public", resp.Design.Visibility)
		assert.Equal(t, "test-creator", resp.Design.OwnerId)
		assert.NotNil(t, resp.Design.CreatedAt)
		assert.NotNil(t, resp.Design.UpdatedAt)
		assert.Empty(t, resp.Design.SectionIds) // Should be empty on creation

		// Verify design.json
		meta := readDesignMetadataDirectly(t, tempDir, "test-create-1")
		require.NotNil(t, meta)
		assert.Equal(t, "test-create-1", meta.Id)
		assert.Equal(t, "Test Create Design 1", meta.Name)
		assert.Equal(t, "public", meta.Visibility)
		assert.Equal(t, "test-creator", meta.OwnerId)
		assert.Empty(t, meta.SectionIds)
		// Verify sections directory exists
		sectionsPath := filepath.Join(tempDir, "test-create-1", "sections")
		_, errStat := os.Stat(sectionsPath)
		assert.NoError(t, errStat, "Sections directory should be created")
		assert.True(t, !os.IsNotExist(errStat))
	})

	// ... (other CreateDesign subtests remain mostly the same) ...
	t.Run("Create with Auto-Generated ID", func(t *testing.T) {
		req := &protos.CreateDesignRequest{
			Design: &protos.Design{
				Name:       "Test Auto ID",
				Visibility: "private",
			},
		}
		resp, err := service.CreateDesign(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Design)
		assert.NotEmpty(t, resp.Design.Id)
		assert.Regexp(t, `^[a-zA-Z0-9-_]+$`, resp.Design.Id) // Allow dash/underscore if IDGen uses them
		assert.Equal(t, "Test Auto ID", resp.Design.Name)
		assert.Equal(t, "private", resp.Design.Visibility)
		assert.Equal(t, "test-creator", resp.Design.OwnerId)
		assert.Empty(t, resp.Design.SectionIds)

		generatedId := resp.Design.Id
		meta := readDesignMetadataDirectly(t, tempDir, generatedId)
		require.NotNil(t, meta)
		assert.Equal(t, generatedId, meta.Id)
		assert.Equal(t, "Test Auto ID", meta.Name)
		assert.Equal(t, "private", meta.Visibility)
		assert.Empty(t, meta.SectionIds)
		sectionsPath := filepath.Join(tempDir, generatedId, "sections")
		_, errStat := os.Stat(sectionsPath)
		assert.NoError(t, errStat)
	})

	t.Run("Fail on Empty Name", func(t *testing.T) {
		req := &protos.CreateDesignRequest{
			Design: &protos.Design{Name: "  "},
		}
		_, err := service.CreateDesign(ctx, req)
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("Fail on Already Exists (Provided Id)", func(t *testing.T) {
		createDesignDirectly(t, tempDir, Design{Id: "test-conflict", Name: "Existing"})
		req := &protos.CreateDesignRequest{
			Design: &protos.Design{Id: "test-conflict", Name: "Trying to overwrite"},
		}
		_, err := service.CreateDesign(ctx, req)
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.AlreadyExists, st.Code())
	})
}

func TestGetDesign(t *testing.T) {
	service, tempDir := newTestDesignService(t)
	ctx := testContextWithUser("test-getter")
	designId := "test-get-1"
	secIdA := "sec-meta-a"
	secIdB := "sec-meta-b"

	now := time.Now().UTC().Truncate(time.Second)
	setupMeta := Design{
		Id:          designId,
		OwnerId:     "test-owner",
		Name:        "Test Get Design",
		Description: "Get Description",
		Visibility:  "private",
		VisibleTo:   []string{"user1", "user2"},
		BaseModel: BaseModel{
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now,
		},
		SectionIds: []string{secIdA, secIdB}, // Include sections
	}
	createDesignDirectly(t, tempDir, setupMeta)
	// Create corresponding section metadata files
	createSectionMetadataDirectly(t, tempDir, designId, secIdA, Section{Type: "text", Title: "Section A Meta"})
	createSectionMetadataDirectly(t, tempDir, designId, secIdB, Section{Type: "drawing", Title: "Section B Meta"})
	// Create a section dir/file that's *not* in SectionIds to ensure it's ignored
	createSectionMetadataDirectly(t, tempDir, designId, "sec-orphan", Section{Type: "plot", Title: "Orphan Meta"})

	t.Run("Get Existing Design (No Section Meta)", func(t *testing.T) {
		req := &protos.GetDesignRequest{Id: designId}
		resp, err := service.GetDesign(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Design)
		assert.Empty(t, resp.SectionsMetadata, "SectionsMetadata should be empty when not requested")

		// Verify design fields
		expectedProto := &protos.Design{
			Id:          setupMeta.Id,
			OwnerId:     setupMeta.OwnerId,
			Name:        setupMeta.Name,
			Description: setupMeta.Description,
			Visibility:  setupMeta.Visibility,
			VisibleTo:   setupMeta.VisibleTo,
			CreatedAt:   tspb.New(setupMeta.CreatedAt),
			UpdatedAt:   tspb.New(setupMeta.UpdatedAt),
			SectionIds:  []string{secIdA, secIdB},
		}
		assert.True(t, cmp.Equal(expectedProto, resp.Design, protocmp.Transform()), cmp.Diff(expectedProto, resp.Design, protocmp.Transform()))
	})

	t.Run("Get Existing Design (Include Section Meta)", func(t *testing.T) {
		req := &protos.GetDesignRequest{Id: designId, IncludeSectionMetadata: true}
		resp, err := service.GetDesign(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Design)
		require.Len(t, resp.SectionsMetadata, 2, "Should return metadata for sections listed in SectionIds")

		// Check section metadata order and content (no content oneof)
		assert.Equal(t, secIdA, resp.SectionsMetadata[0].Id)
		assert.Equal(t, "Section A Meta", resp.SectionsMetadata[0].Title)
		assert.Equal(t, protos.SectionType_SECTION_TYPE_TEXT, resp.SectionsMetadata[0].Type)
		assert.Equal(t, uint32(0), resp.SectionsMetadata[0].Order) // Order should be set based on index
		// *** REMOVED content assertions ***

		assert.Equal(t, secIdB, resp.SectionsMetadata[1].Id)
		assert.Equal(t, "Section B Meta", resp.SectionsMetadata[1].Title)
		assert.Equal(t, protos.SectionType_SECTION_TYPE_DRAWING, resp.SectionsMetadata[1].Type)
		assert.Equal(t, uint32(1), resp.SectionsMetadata[1].Order)

		// Verify design fields are still correct
		assert.Equal(t, designId, resp.Design.Id)
		assert.Equal(t, []string{secIdA, secIdB}, resp.Design.SectionIds)
	})

	t.Run("Get Non-Existent Design", func(t *testing.T) {
		req := &protos.GetDesignRequest{Id: "does-not-exist"}
		_, err := service.GetDesign(ctx, req)
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.NotFound, st.Code())
	})
}

func TestListDesigns(t *testing.T) {
	// This test remains largely the same as it focuses on design.json
	service, tempDir := newTestDesignService(t)
	ctx := testContextWithUser("user1")

	ts := time.Now().UTC().Truncate(time.Second)
	meta1 := Design{Id: "list-aaa", OwnerId: "user1", Name: "AAA Design", Visibility: "public", BaseModel: BaseModel{UpdatedAt: ts.Add(-2 * time.Hour)}}
	meta2 := Design{Id: "list-ccc", OwnerId: "user2", Name: "CCC Design", Visibility: "private", BaseModel: BaseModel{UpdatedAt: ts}}
	meta3 := Design{Id: "list-bbb", OwnerId: "user1", Name: "BBB Design", Visibility: "public", BaseModel: BaseModel{UpdatedAt: ts.Add(-1 * time.Hour)}}
	createDesignDirectly(t, tempDir, meta1)
	createDesignDirectly(t, tempDir, meta2)
	createDesignDirectly(t, tempDir, meta3)
	// Create invalid entries to ensure they are skipped gracefully
	require.NoError(t, os.Mkdir(filepath.Join(tempDir, "list-corrupt"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "list-corrupt", "design.json"), []byte("bad"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(tempDir, "list-no-meta"), 0755)) // Dir without design.json

	t.Run("List All (Default Order - Recent)", func(t *testing.T) {
		req := &protos.ListDesignsRequest{}
		resp, err := service.ListDesigns(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Designs, 3) // Only valid designs counted
		assert.False(t, resp.Pagination.HasMore)
		// TotalResults might be tricky depending on whether we count readable files or successful unmarshals.
		// Let's assume it counts successfully processed metadata.
		assert.EqualValues(t, 3, resp.Pagination.TotalResults)
		// Order depends on UpdatedAt: CCC, BBB, AAA
		assert.Equal(t, "list-ccc", resp.Designs[0].Id)
		assert.Equal(t, "list-bbb", resp.Designs[1].Id)
		assert.Equal(t, "list-aaa", resp.Designs[2].Id)
	})

	// ... (other ListDesigns subtests for sorting, filtering, pagination remain the same) ...
	t.Run("List Sort by Name", func(t *testing.T) {
		req := &protos.ListDesignsRequest{OrderBy: "name"}
		resp, err := service.ListDesigns(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Designs, 3)
		assert.Equal(t, "list-aaa", resp.Designs[0].Id)
		assert.Equal(t, "list-bbb", resp.Designs[1].Id)
		assert.Equal(t, "list-ccc", resp.Designs[2].Id)
	})

	t.Run("List Filter by Owner", func(t *testing.T) {
		req := &protos.ListDesignsRequest{OwnerId: "user1"}
		resp, err := service.ListDesigns(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Designs, 2)
		assert.Equal(t, "list-bbb", resp.Designs[0].Id) // BBB more recent than AAA for user1
		assert.Equal(t, "list-aaa", resp.Designs[1].Id)
	})

	t.Run("List with Pagination", func(t *testing.T) {
		// Request first page of size 2, default sort (recent: CCC, BBB, AAA)
		req1 := &protos.ListDesignsRequest{Pagination: &protos.Pagination{PageSize: 2, PageOffset: 0}}
		resp1, err1 := service.ListDesigns(ctx, req1)
		require.NoError(t, err1)
		require.NotNil(t, resp1)
		assert.Len(t, resp1.Designs, 2)
		assert.True(t, resp1.Pagination.HasMore)
		assert.EqualValues(t, 3, resp1.Pagination.TotalResults)
		assert.Equal(t, "list-ccc", resp1.Designs[0].Id)
		assert.Equal(t, "list-bbb", resp1.Designs[1].Id)

		// Request second page of size 2
		req2 := &protos.ListDesignsRequest{Pagination: &protos.Pagination{PageSize: 2, PageOffset: 2}}
		resp2, err2 := service.ListDesigns(ctx, req2)
		require.NoError(t, err2)
		require.NotNil(t, resp2)
		assert.Len(t, resp2.Designs, 1)
		assert.False(t, resp2.Pagination.HasMore)
		assert.EqualValues(t, 3, resp2.Pagination.TotalResults)
		assert.Equal(t, "list-aaa", resp2.Designs[0].Id)
	})
}

func TestUpdateDesign(t *testing.T) {
	// This test remains the same as it only modifies design.json
	service, tempDir := newTestDesignService(t)
	ctx := testContextWithUser("updater")

	setupTime := time.Now().UTC().Truncate(time.Second)
	meta := Design{Id: "update-1", OwnerId: "updater", Name: "Original Name", Description: "Original Desc", Visibility: "private", BaseModel: BaseModel{CreatedAt: setupTime, UpdatedAt: setupTime}}
	createDesignDirectly(t, tempDir, meta)

	t.Run("Update Single Field (Name)", func(t *testing.T) {
		req := &protos.UpdateDesignRequest{
			Design: &protos.Design{
				Id:   "update-1", // ID must be in the design object now for path matching in gateway/proto
				Name: "Updated Name",
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
		}
		resp, err := service.UpdateDesign(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "update-1", resp.Design.Id)
		assert.Equal(t, "Updated Name", resp.Design.Name)
		assert.Equal(t, "Original Desc", resp.Design.Description) // Should not change
		assert.True(t, resp.Design.UpdatedAt.AsTime().After(setupTime))

		updatedMeta := readDesignMetadataDirectly(t, tempDir, "update-1")
		require.NotNil(t, updatedMeta)
		assert.Equal(t, "Updated Name", updatedMeta.Name)
		assert.Equal(t, "Original Desc", updatedMeta.Description)
	})

	// ... (other UpdateDesign subtests remain the same) ...
	t.Run("Update Multiple Fields (Desc, Visibility)", func(t *testing.T) {
		currentMeta := readDesignMetadataDirectly(t, tempDir, "update-1")
		require.NotNil(t, currentMeta)
		previousUpdateTime := currentMeta.UpdatedAt

		req := &protos.UpdateDesignRequest{
			Design: &protos.Design{
				Id:          "update-1",
				Name:        "Ignored Name", // Should be ignored by mask
				Description: "Updated Description",
				Visibility:  "public",
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description", "visibility"}},
		}
		resp, err := service.UpdateDesign(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "Updated Name", resp.Design.Name) // From previous step
		assert.Equal(t, "Updated Description", resp.Design.Description)
		assert.Equal(t, "public", resp.Design.Visibility)
		assert.True(t, resp.Design.UpdatedAt.AsTime().After(previousUpdateTime))

		updatedMeta := readDesignMetadataDirectly(t, tempDir, "update-1")
		require.NotNil(t, updatedMeta)
		assert.Equal(t, "Updated Name", updatedMeta.Name)
		assert.Equal(t, "Updated Description", updatedMeta.Description)
		assert.Equal(t, "public", updatedMeta.Visibility)
	})
}

func TestDeleteDesign(t *testing.T) {
	// This test remains the same as it operates on the design directory
	service, tempDir := newTestDesignService(t)
	ctx := testContextWithUser("deleter")
	designId := "delete-1"

	meta := Design{Id: designId, OwnerId: "deleter", Name: "To Be Deleted", BaseModel: BaseModel{CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	createDesignDirectly(t, tempDir, meta)

	t.Run("Delete Existing", func(t *testing.T) {
		req := &protos.DeleteDesignRequest{Id: designId}

		designPath := filepath.Join(tempDir, designId)
		_, err := os.Stat(designPath)
		require.NoError(t, err, "Design directory should exist before delete")

		resp, err := service.DeleteDesign(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		_, err = os.Stat(designPath)
		require.Error(t, err, "Design directory should not exist after delete")
		assert.True(t, os.IsNotExist(err), "Error should be os.ErrNotExist")

		// Verify GetDesign also fails
		_, getErr := service.GetDesign(ctx, &protos.GetDesignRequest{Id: designId})
		require.Error(t, getErr)
		st, ok := status.FromError(getErr)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})

	// ... (other DeleteDesign subtests remain the same) ...
	t.Run("Delete Non-Existent (Idempotency)", func(t *testing.T) {
		req := &protos.DeleteDesignRequest{Id: "does-not-exist"}
		resp, err := service.DeleteDesign(ctx, req)
		require.NoError(t, err, "Deleting non-existent should not error")
		require.NotNil(t, resp)
	})
}

// --- Section Test Cases (Modified) ---

func TestGetSection(t *testing.T) {
	service, tempDir := newTestDesignService(t)
	ctx := testContextWithUser("section-owner")
	designId := "design-get-sections"
	sectionId := "sec-get-1"

	// Setup Design (linking the section)
	createDesignDirectly(t, tempDir, Design{
		Id:         designId,
		OwnerId:    "section-owner",
		Name:       "Design Get Sections",
		SectionIds: []string{sectionId},
	})

	// Setup Section Metadata (main.json)
	now := time.Now()
	sectionMeta := Section{
		// Id/DesignId are set from context/request, not read from file typically
		Type:      "text",
		Title:     "Test Section Get Title",
		BaseModel: BaseModel{CreatedAt: now, UpdatedAt: now},
		// NO Content field here
	}
	createSectionMetadataDirectly(t, tempDir, designId, sectionId, sectionMeta)

	t.Run("Get Existing Section Metadata", func(t *testing.T) {
		req := &protos.GetSectionRequest{DesignId: designId, SectionId: sectionId}
		resp, err := service.GetSection(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, sectionId, resp.Id)
		assert.Equal(t, designId, resp.DesignId)
		assert.Equal(t, protos.SectionType_SECTION_TYPE_TEXT, resp.Type)
		assert.Equal(t, "Test Section Get Title", resp.Title)
		assert.Equal(t, uint32(0), resp.Order) // Should have order 0

		// *** REMOVED content assertions ***
	})

	t.Run("Get Non-Existent Section", func(t *testing.T) {
		req := &protos.GetSectionRequest{DesignId: designId, SectionId: "non-existent-sec"}
		_, err := service.GetSection(ctx, req)
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.NotFound, st.Code())
	})
}

func TestAddSection(t *testing.T) {
	service, tempDir := newTestDesignService(t)
	ownerId := "add-sec-owner"
	ctx := testContextWithUser(ownerId)
	designId := "design-add-sec"

	// Setup Design
	createDesignDirectly(t, tempDir, Design{
		Id:         designId,
		OwnerId:    ownerId,
		Name:       "Design for Adding Sections",
		SectionIds: []string{"sec-a"}, // Start with one section
	})
	// Create metadata for initial section
	createSectionMetadataDirectly(t, tempDir, designId, "sec-a", Section{Type: "text", Title: "Section A"})

	t.Run("Add Section to End", func(t *testing.T) {
		req := &protos.AddSectionRequest{
			Section: &protos.Section{
				DesignId: designId, // designId now part of Section proto
				Type:     protos.SectionType_SECTION_TYPE_DRAWING,
				Title:    "New Drawing Section",
				// NO Content field in request proto expected by AddSection now
			},
			Position: protos.PositionType_POSITION_TYPE_END,
		}
		resp, err := service.AddSection(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		newSectionId := resp.Id
		assert.NotEmpty(t, newSectionId)
		assert.NotEqual(t, "sec-a", newSectionId)
		assert.Equal(t, protos.SectionType_SECTION_TYPE_DRAWING, resp.Type)
		assert.Equal(t, "New Drawing Section", resp.Title)
		assert.Equal(t, uint32(1), resp.Order) // Should be order 1 (index 1 after sec-a)
		// *** REMOVED content assertion: assert.Nil(t, resp.GetDrawingContent()) *** // This method doesn't exist

		// Verify design.json updated
		meta := readDesignMetadataDirectly(t, tempDir, designId)
		require.NotNil(t, meta)
		assert.Equal(t, []string{"sec-a", newSectionId}, meta.SectionIds)

		// Verify section dir and main.json created
		sectionDir := filepath.Join(tempDir, designId, "sections", newSectionId)
		_, errStatDir := os.Stat(sectionDir)
		assert.NoError(t, errStatDir)

		secMeta := readSectionMetadataDirectly(t, tempDir, designId, newSectionId)
		require.NotNil(t, secMeta)
		assert.Equal(t, newSectionId, secMeta.Id)
		assert.Equal(t, designId, secMeta.DesignId)
		assert.Equal(t, "drawing", secMeta.Type) // Stored type string
		assert.Equal(t, "New Drawing Section", secMeta.Title)

		// Verify NO content file was created by AddSection
		contentPath := filepath.Join(sectionDir, "content.main") // Assume default name
		_, errStatContent := os.Stat(contentPath)
		assert.Error(t, errStatContent)
		assert.True(t, os.IsNotExist(errStatContent))
	})

	// ... (Add tests for AddSection Before/After) ...
	t.Run("Add Section After sec-a", func(t *testing.T) {
		req := &protos.AddSectionRequest{
			Section: &protos.Section{
				DesignId: designId,
				Type:     protos.SectionType_SECTION_TYPE_TEXT, // *** FIXED ENUM ***
				Title:    "Inserted Section B",
			},
			RelativeSectionId: "sec-a",
			Position:          protos.PositionType_POSITION_TYPE_AFTER,
		}
		resp, err := service.AddSection(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		newSectionIdB := resp.Id
		assert.NotEmpty(t, newSectionIdB)
		assert.Equal(t, uint32(1), resp.Order) // Should be inserted at index 1

		// Verify design.json: Order should be [sec-a, newSectionIdB, previous_new_section]
		meta := readDesignMetadataDirectly(t, tempDir, designId)
		require.NotNil(t, meta)
		require.Len(t, meta.SectionIds, 3)
		assert.Equal(t, "sec-a", meta.SectionIds[0])
		assert.Equal(t, newSectionIdB, meta.SectionIds[1])

		// Verify main.json for B
		secMetaB := readSectionMetadataDirectly(t, tempDir, designId, newSectionIdB)
		require.NotNil(t, secMetaB)
		assert.Equal(t, "Inserted Section B", secMetaB.Title)
		assert.Equal(t, "text", secMetaB.Type)
	})
}

func TestUpdateSection(t *testing.T) {
	service, tempDir := newTestDesignService(t)
	ownerId := "update-sec-owner"
	ctx := testContextWithUser(ownerId)
	designId := "design-update-sec"
	secId := "update-meta-1"

	// Setup Design & Section Metadata
	createDesignDirectly(t, tempDir, Design{
		Id:         designId,
		OwnerId:    ownerId,
		SectionIds: []string{secId},
	})
	createSectionMetadataDirectly(t, tempDir, designId, secId, Section{Type: "text", Title: "Original Title"})

	t.Run("Update Title Only", func(t *testing.T) {
		newTitle := "Updated Section Title"
		req := &protos.UpdateSectionRequest{
			Section: &protos.Section{
				DesignId: designId,
				Id:       secId,
				Title:    newTitle,
				// Type is not updated via this RPC in this design
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
		}
		resp, err := service.UpdateSection(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, newTitle, resp.Title)
		assert.Equal(t, protos.SectionType_SECTION_TYPE_TEXT, resp.Type) // Type shouldn't change
		// *** REMOVED content assertion: assert.Nil(t, resp.GetTextContent()) ***

		// Verify main.json updated
		secMeta := readSectionMetadataDirectly(t, tempDir, designId, secId)
		require.NotNil(t, secMeta)
		assert.Equal(t, newTitle, secMeta.Title)
		assert.Equal(t, "text", secMeta.Type) // Type unchanged
	})

	t.Run("Ignore Content Update Attempt", func(t *testing.T) {
		// This test verifies that UpdateSection *ignores* content updates
		req := &protos.UpdateSectionRequest{
			Section: &protos.Section{
				DesignId: designId,
				Id:       secId,
				Title:    "Attempt Content Update", // Include a title update too
				// Content field doesn't exist here anymore, cannot add TextContent
			},
			// Mask requests content update, which should be ignored by UpdateSection
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title", "text_content"}}, // text_content path is now invalid for UpdateSection
		}
		resp, err := service.UpdateSection(ctx, req)
		// UpdateSection *should* potentially error if mask contains invalid paths like "text_content"
		// Let's assert for InvalidArgument or check if it ignores gracefully.
		// Assuming graceful ignore for now based on switch statement in UpdateSection.
		require.NoError(t, err) // No error if it ignores the path
		require.NotNil(t, resp)
		assert.Equal(t, "Attempt Content Update", resp.Title) // Title should update
		// *** REMOVED content assertion ***

		// Verify main.json title updated
		secMeta := readSectionMetadataDirectly(t, tempDir, designId, secId)
		require.NotNil(t, secMeta)
		assert.Equal(t, "Attempt Content Update", secMeta.Title)
		// Content file should NOT have been created or modified by this call
	})

	t.Run("Update Non-Existent Section", func(t *testing.T) {
		req := &protos.UpdateSectionRequest{
			Section:    &protos.Section{DesignId: designId, Id: "does-not-exist", Title: "No"},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
		}
		_, err := service.UpdateSection(ctx, req)
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.NotFound, st.Code())
	})

	// Add tests for permission denied, invalid mask path ("type"), no mask
}

func TestDeleteSection(t *testing.T) {
	service, tempDir := newTestDesignService(t)
	ownerId := "delete-sec-owner"
	ctx := testContextWithUser(ownerId)
	designId := "design-delete-sec"
	secA := "del-a"
	secB := "del-b"
	secC := "del-c"

	// Setup Design & Section Metadata & Dirs
	createDesignDirectly(t, tempDir, Design{
		Id:         designId,
		OwnerId:    ownerId,
		SectionIds: []string{secA, secB, secC},
	})
	createSectionMetadataDirectly(t, tempDir, designId, secA, Section{Type: "text"})
	createSectionMetadataDirectly(t, tempDir, designId, secB, Section{Type: "drawing"})
	createSectionMetadataDirectly(t, tempDir, designId, secC, Section{Type: "plot"})

	t.Run("Delete Existing Section", func(t *testing.T) {
		req := &protos.DeleteSectionRequest{DesignId: designId, SectionId: secB}

		sectionBPath := filepath.Join(tempDir, designId, "sections", secB)
		_, err := os.Stat(sectionBPath)
		require.NoError(t, err, "Section B directory should exist before delete")

		_, err = service.DeleteSection(ctx, req)
		require.NoError(t, err)

		// Verify metadata updated in design.json
		meta := readDesignMetadataDirectly(t, tempDir, designId)
		require.NotNil(t, meta)
		assert.Equal(t, []string{secA, secC}, meta.SectionIds)

		// Verify section B directory deleted
		_, err = os.Stat(sectionBPath)
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))

		// Verify other section dirs still exist
		sectionAPath := filepath.Join(tempDir, designId, "sections", secA)
		_, err = os.Stat(sectionAPath)
		assert.NoError(t, err)
		sectionCPath := filepath.Join(tempDir, designId, "sections", secC)
		_, err = os.Stat(sectionCPath)
		assert.NoError(t, err)
	})

	// ... (other DeleteSection subtests remain the same) ...
	t.Run("Delete Non-Existent Section (Idempotent)", func(t *testing.T) {
		req := &protos.DeleteSectionRequest{DesignId: designId, SectionId: "non-existent"}
		_, err := service.DeleteSection(ctx, req)
		require.NoError(t, err) // Should not error

		meta := readDesignMetadataDirectly(t, tempDir, designId)
		require.NotNil(t, meta)
		assert.Equal(t, []string{secA, secC}, meta.SectionIds) // Unchanged
	})
}

func TestMoveSection(t *testing.T) {
	// This test remains the same as it only modifies design.json
	service, tempDir := newTestDesignService(t)
	ownerId := "move-sec-owner"
	ctx := testContextWithUser(ownerId)
	designId := "design-move-sec"
	secA, secB, secC, secD := "mv-a", "mv-b", "mv-c", "mv-d"

	createDesignDirectly(t, tempDir, Design{
		Id:         designId,
		OwnerId:    ownerId,
		SectionIds: []string{secA, secB, secC, secD},
	})

	t.Run("Move B After C", func(t *testing.T) {
		req := &protos.MoveSectionRequest{
			DesignId:          designId,
			SectionId:         secB, // Section being moved
			RelativeSectionId: secC,
			Position:          protos.PositionType_POSITION_TYPE_AFTER,
		}
		_, err := service.MoveSection(ctx, req)
		require.NoError(t, err)
		meta := readDesignMetadataDirectly(t, tempDir, designId)
		require.NotNil(t, meta)
		assert.Equal(t, []string{secA, secC, secB, secD}, meta.SectionIds)
	})

	t.Run("Move D Before C", func(t *testing.T) {
		// State after previous test: [A, C, B, D]
		req := &protos.MoveSectionRequest{
			DesignId:          designId,
			SectionId:         secD, // Section being moved
			RelativeSectionId: secC,
			Position:          protos.PositionType_POSITION_TYPE_BEFORE,
		}
		_, err := service.MoveSection(ctx, req)
		require.NoError(t, err)
		meta := readDesignMetadataDirectly(t, tempDir, designId)
		require.NotNil(t, meta)
		assert.Equal(t, []string{secA, secD, secC, secB}, meta.SectionIds)
	})

	// Add more move tests: move first, move last, invalid relative, no change needed etc.
	t.Run("Move A After D (Move first to last)", func(t *testing.T) {
		// State: [A, D, C, B]
		req := &protos.MoveSectionRequest{
			DesignId:          designId,
			SectionId:         secA,
			RelativeSectionId: secB, // Relative to last item
			Position:          protos.PositionType_POSITION_TYPE_AFTER,
		}
		_, err := service.MoveSection(ctx, req)
		require.NoError(t, err)
		meta := readDesignMetadataDirectly(t, tempDir, designId)
		require.NotNil(t, meta)
		assert.Equal(t, []string{secD, secC, secB, secA}, meta.SectionIds)
	})

	t.Run("Move C Before C (No Change)", func(t *testing.T) {
		// State: [D, C, B, A]
		req := &protos.MoveSectionRequest{
			DesignId:          designId,
			SectionId:         secC,
			RelativeSectionId: secC, // Relative to itself
			Position:          protos.PositionType_POSITION_TYPE_BEFORE,
		}
		_, err := service.MoveSection(ctx, req)
		require.NoError(t, err) // Should be a no-op
		meta := readDesignMetadataDirectly(t, tempDir, designId)
		require.NotNil(t, meta)
		assert.Equal(t, []string{secD, secC, secB, secA}, meta.SectionIds) // Unchanged
	})
}
