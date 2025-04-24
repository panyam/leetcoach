// ./web/views/components/converters.ts

import { SectionData, SectionType } from './types';
import {
    V1Section, V1SectionType,
    // Import the UpdateSectionRequest type for the metadata update object
    ConsolidateSectionUpdatesIntoOneRPCUsingPATCHAndFieldMask as ApiSectionMetadataUpdateObject
} from './apiclient'; // Adjust path as needed

// --- Type Mapping ---

export function mapApiSectionTypeToFrontend(apiType?: V1SectionType): SectionType {
    switch (apiType) {
        case V1SectionType.SectionTypeText: return 'text';
        case V1SectionType.SectionTypeDrawing: return 'drawing';
        case V1SectionType.SectionTypePlot: return 'plot';
        case V1SectionType.SectionTypeUnspecified:
        default:
             console.warn(`Received unspecified or unknown section type: ${apiType}, defaulting to 'text'.`);
             return 'text';
    }
}

export function mapFrontendSectionTypeToApi(type: SectionType): V1SectionType {
    switch (type) {
        case 'text': return V1SectionType.SectionTypeText;
        case 'drawing': return V1SectionType.SectionTypeDrawing;
        case 'plot': return V1SectionType.SectionTypePlot;
        default:
             console.error(`Cannot map unknown frontend section type: ${type}`);
             return V1SectionType.SectionTypeUnspecified; // Indicate error
    }
}

// --- Content Mapping (REMOVED) ---
// REMOVED: extractContentFromApiSection
// REMOVED: mapFrontendContentToApiUpdate

// --- Full Conversion ---

/**
 * Converts a V1Section proto (metadata only) to the frontend SectionData structure.
 * Initializes content to null, as it needs to be fetched separately.
 */
export function convertApiSectionToSectionData(apiSection: V1Section): SectionData {
    if (!apiSection?.id) {
        throw new Error("Cannot convert API section without an ID");
    }
    const frontendType = mapApiSectionTypeToFrontend(apiSection.type);
    return {
        id: apiSection.id,
        designId: apiSection.designId || "", // Get designId from proto
        type: frontendType,
        title: apiSection.title || `Untitled ${frontendType} Section`,
        order: apiSection.order || 0,
        content: null, // Initialize content as null - needs separate fetch
    };
}

/**
 * Creates the request object for updating Section *metadata* (currently only title)
 * using the DesignService.UpdateSection endpoint.
 * @param updates - An object containing the fields to update (currently only title).
 * @returns The request body object for the PATCH request.
 */
export function createApiSectionMetadataUpdateObject(
    updates: { title?: string }
): ApiSectionMetadataUpdateObject {
     // Type assertion is safe here as we construct the object according to the expected structure
     const apiUpdate: Partial<ApiSectionMetadataUpdateObject> = {
        section: {}, // Initialize the nested section object
        updateMask: "" // Initialize updateMask
     };

     let maskPaths: string[] = [];

     if (updates.title !== undefined) {
        // Ensure section object exists before assigning title
        apiUpdate.section = apiUpdate.section || {};
        apiUpdate.section.title = updates.title;
        maskPaths.push("title"); // Add "title" to the mask
     }

     // Assign the joined mask paths
     apiUpdate.updateMask = maskPaths.join(',');

     // Return the fully typed object
     return apiUpdate as ApiSectionMetadataUpdateObject;
}
