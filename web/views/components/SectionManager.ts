// components/SectionManager.ts

import { Modal } from './Modal';
import { BaseSection } from './BaseSection';
import { TextSection } from './TextSection';
import { DrawingSection } from './DrawingSection';
import { PlotSection } from './PlotSection';
import { DocumentSection, SectionType, TextContent, DrawingContent, PlotContent, SectionData, SectionCallbacks } from './types';
import { TocItemInfo, TableOfContents } from './TableOfContents';
import { V1Section, V1PositionType, V1GetContentResponse /* other models if needed */ } from './apiclient'; // Import V1Section and V1PositionType
import { DesignApi, ContentApi  } from './Api'; // Import API client
import { convertApiSectionToSectionData, createApiSectionMetadataUpdateObject } from './converters'; // Import the converter
import { mapFrontendSectionTypeToApi } from './converters'; // Import the update object creator


/**
 * Manages document sections using the BaseSection hierarchy.
 * Collaborates with TableOfContents component for UI updates.
 */
export class SectionManager {
    private designId: string = "";
    private sections: Map<string, BaseSection> = new Map();
    private sectionData: SectionData[] = [];
    private nextSectionId: number = 1;
    private sectionsContainer: HTMLElement | null;
    private emptyStateEl: HTMLElement | null;
    private fabAddSectionBtn: HTMLElement | null;
    private sectionTemplate: HTMLElement | null;
    private modal: Modal;

    // Reference to the external TOC component - set via method/constructor
    private tocComponent: TableOfContents | null = null;

    // Store section IDs and metadata fetched from the API before loading content
    private initialSectionIds: string[] = [];
    private initialSectionsMetadata: V1Section[] | null = null; // Optional metadata

    // Predefined title suggestions (keep as before)
    private static readonly TEXT_TITLES: string[] = [
      "Requirements", "Functional Requirements", "Non-Functional Requirements",
      "API Design", "Data Model", "High-Level Design", "Detailed Design",
      "Assumptions", "Security Considerations", "Deployment Strategy",
      "Monitoring & Alerting", "Future Considerations", "Capacity Planning",
      "System Interfaces", "User Scenarios"
    ];
    private static readonly DRAWING_TITLES: string[] = [
        "Architecture Overview", "System Components", "Data Flow Diagram",
        "Sequence Diagram", "Network Topology", "Component Interactions",
        "Deployment View", "API Interactions", "Database Schema",
        "High-Level Architecture", "User Flow"
    ];
    private static readonly PLOT_TITLES: string[] = [
        "Scalability Analysis", "Latency vs Throughput", "QPS Estimates",
        "Storage Projections", "Cost Analysis", "Performance Metrics",
        "Resource Utilization", "Traffic Estimation", "Data Growth",
        "Benchmark Results"
    ];


    private static getRandomTitle(type: SectionType): string {
      const titles = type === 'drawing' ? this.DRAWING_TITLES : type === 'plot' ? this.PLOT_TITLES : this.TEXT_TITLES;
      const randomIndex = Math.floor(Math.random() * titles.length);
      return titles[randomIndex] || `New ${type.charAt(0).toUpperCase() + type.slice(1)} Section`; // Fallback
    }


    constructor(public currentDesignId: string | null) {
        this.sectionsContainer = document.getElementById('sections-container');
        this.emptyStateEl = document.getElementById('empty-state');
        this.fabAddSectionBtn = document.getElementById('fab-add-section-btn'); // Find the new area
        this.sectionTemplate = document.getElementById('section-template');
        this.modal = Modal.getInstance();

        if (!this.sectionsContainer || !this.emptyStateEl || !this.fabAddSectionBtn || !this.sectionTemplate) {
            console.error("SectionManager: Could not find all required DOM elements. Check IDs.");
        }
        if (this.sectionTemplate) {
           this.sectionTemplate.classList.add('hidden');
        }

        this.bindEvents();
        this.handleEmptyState(); // Initial check
    }

    /** Sets the TableOfContents component instance for communication */
    public setTocComponent(toc: TableOfContents): void {
        this.tocComponent = toc;
        // Update TOC immediately when it's set, in case sections loaded before TOC was ready
        this.triggerTocUpdate();
    }

    /**
     * NEW: Sets the design ID for API calls.
     */
    public setDesignId(designId: string): void {
        this.currentDesignId = designId;
        console.log(`SectionManager: Design ID set to ${designId}`);
    }

    /**
     * NEW: Stores the initial section IDs and metadata received from the API.
     * This data will be used later to load the actual section content.
     */
    public setInitialSectionInfo(ids: string[], metadata?: V1Section[] | null): void {
        console.log("SectionManager: Received initial section info", { count: ids.length, hasMetadata: !!metadata });
        this.initialSectionIds = ids || [];
        this.initialSectionsMetadata = metadata || null;
        // Do NOT load sections here yet. That's Step 1.2.
        // Do NOT call handleEmptyState here yet, as sectionData is still empty.
        this.clearAllSections();
    }


    /** Bind event listeners (only for section creation/type selection now) */
    private bindEvents(): void {
        // Bind add section buttons originating from OUTSIDE the TOC
        // Note: The TOC's own 'Add Section' button is handled by TableOfContents component
        document.addEventListener('click', (e: MouseEvent) => {
            const target = e.target as HTMLElement;
            // Only listen for buttons NOT inside the TOC sidebar
            const addSectionBtn = target.closest('#create-first-section'); // Only listen for the empty state button here

            // Add buttons within sections are now handled by BaseSection -> onAddSectionRequest callback
            if (addSectionBtn) {
                e.preventDefault();
                let insertAfterId: string | null = null;
                // The #create-first-section button always adds to the end
                this.openSectionTypeSelector(null, 'after');
            }
        });


         // Bind the FAB "Add Section" button
         if (this.fabAddSectionBtn) { // Use the class property directly
            this.fabAddSectionBtn.addEventListener('click', () => {
                // Find the ID of the *current* last section
                const lastSection = this.sectionData.length > 0
                    ? this.sectionData.sort((a, b) => a.order - b.order)[this.sectionData.length - 1]
                    : null;
                const lastSectionId = lastSection ? lastSection.id : null;
                // Open selector to add AFTER the last section (or to the end if list is empty, though button should be hidden then)
                this.openSectionTypeSelector(lastSectionId, 'after');
            });
         }
    }

    /** Open section type selector modal */
    public openSectionTypeSelector(relativeToId: string | null, position: 'before' | 'after' = 'after'): void {
        this.modal.show('section-type-selector', {
            relativeToId: relativeToId || '',
            position: position
        });
        console.log('SectionManager opening section type selector modal', { relativeToId, position });
    }

    /**
     * Handles the user's choice from the section type selector modal.
     * Calls AddSection for metadata, then SetContent for initial (empty) content.
     */
    public async handleSectionTypeSelection(
        type: SectionType,
        relativeToId: string | null = null,
        position: 'before' | 'after' = 'after'
    ): Promise<void> {
         if (!this.currentDesignId) {
            console.error("Cannot add section: Design ID not set in SectionManager.");
            // toastManager.showToast("Error", "Cannot add section: Design ID missing.", "error");
            return;
        }
        console.log(`User requested to add a '${type}' section ${position} ${relativeToId || 'the end'}`);

        let apiPosition: V1PositionType;
        if (!relativeToId) {
            apiPosition = V1PositionType.PositionTypeEnd;
        } else {
            apiPosition = position === 'before' ? V1PositionType.PositionTypeBefore : V1PositionType.PositionTypeAfter;
        }

        // Prepare minimal section metadata for AddSection API
        const addSectionPayload: Partial<V1Section> = { // Use Partial<V1Section> for clarity
             type: mapFrontendSectionTypeToApi(type),
             title: SectionManager.getRandomTitle(type),
             designId: this.currentDesignId, // Include designId here
             // NO content fields here anymore
        };

        let createdApiSection: V1Section | null = null;

        try {
            console.log("Calling designServiceAddSection API...");
            // The API now returns Section metadata only
            createdApiSection = await DesignApi.designServiceAddSection({
                sectionDesignId: this.currentDesignId, // Route parameter
                body: { // Body for AddSectionRequest
                    section: addSectionPayload,
                    relativeSectionId: relativeToId || undefined,
                    position: apiPosition,
                }
            });
            console.log("API Success: designServiceAddSection returned:", createdApiSection);

            // Convert API metadata response to frontend SectionData (content will be null)
            const newSectionData = convertApiSectionToSectionData(createdApiSection);

            // --- Step 2: Set Initial (Empty) Content via ContentService ---
            try {
                console.log(`Calling contentServiceSetContent for new section ${newSectionData.id}...`);
                const { initialContentBytes, contentType, format } = this.getInitialContentInfo(type);
                const contentName = 'main'; // Define the content name

                await ContentApi.contentServiceSetContent({
                    designId: this.currentDesignId,
                    sectionId: newSectionData.id,
                    contentName: 'main', // Use the name parameter in the URL path
                    body: {
                        content: { // Metadata about the content being set
                            type: contentType,
                            format: format,
                        },
                        contentBytes: initialContentBytes, // Base64 encoded empty content
                        updateMask: 'content_bytes,content.type,content.format' // Update bytes and metadata
                    }
                });
                console.log(`API Success: contentServiceSetContent for section ${newSectionData.id}`);
                // Update the local SectionData content field AFTER setting it via API
                newSectionData.content = initialContentBytes;

            } catch (contentError) {
                 console.error(`API Error setting initial content for section ${newSectionData.id}:`, contentError);
                 // Section metadata was created, but content failed. Add section locally anyway?
                 // For now, add it with null content. User can try editing later.
                 newSectionData.content = null;
                 // toastManager.showToast("Warning", `Section added, but failed to set initial content for "${newSectionData.title}".`, "warning");
            }
            // ---------------------------------------------------------

            // Create the section locally using the data (content might be null or initial bytes)
            this.createSectionInternal(newSectionData, true); // Pass true indicating user action

        } catch (error) {
            console.error("API Error adding section metadata:", error);
             // If AddSection itself failed, don't proceed.
             // toastManager.showToast("Error Adding Section", `Failed to add section metadata: ${error.message || 'Server Error'}`, "error");
             // If createdApiSection is not null here, it means AddSection succeeded but SetContent failed earlier.
             // We might want to attempt deleting the created section metadata for consistency? Or leave it.
        }
    }

    /**
     * Internal method to create the section instance, DOM elements, and update state.
     * Called either during initial load or after a successful addSection API call.
     * @param sectionData - The complete data for the section (metadata + potentially raw content).
     * @param isUserAction - Flag to differentiate between load and user add for reordering/TOC updates.
     */
    private createSectionInternal(
        sectionData: SectionData,
        isUserAction: boolean = false
    ): BaseSection | null {
        console.log("createSectionInternal called with data:", sectionData, "isUserAction:", isUserAction);

        const sectionId = sectionData.id;
        const type = sectionData.type;

        if (!this.sectionTemplate || !this.sectionsContainer) {
            console.error("Missing section template or container in createSectionInternal.");
            return null;
        }
        const sectionEl = this.sectionTemplate.cloneNode(true) as HTMLElement;
        sectionEl.classList.remove('hidden');
        sectionEl.id = sectionId;
        sectionEl.dataset.sectionId = sectionId;
        sectionEl.dataset.sectionType = type;

        this.sectionsContainer.appendChild(sectionEl); // Append first, reorder later if needed

        let sectionInstance: BaseSection;
        const callbacks: SectionCallbacks = {
            onDelete: this.deleteSection.bind(this),
            onMoveUp: this.moveSectionUp.bind(this),
            onMoveDown: this.moveSectionDown.bind(this),
            onTitleChange: this.updateSectionTitle.bind(this),
            // REMOVE onContentChange - Saving is handled by BaseSection->ContentService
            onAddSectionRequest: this.openSectionTypeSelector.bind(this)
        };

        // Instantiate the correct subclass. It receives SectionData, which now might
        // have raw (base64) content or null in the .content field.
        switch (type) {
            case 'text':
                sectionInstance = new TextSection(sectionData, sectionEl, callbacks);
                break;
            case 'drawing':
                sectionInstance = new DrawingSection(sectionData, sectionEl, callbacks);
                break;
            case 'plot':
                sectionInstance = new PlotSection(sectionData, sectionEl, callbacks);
                break;
            default:
                console.warn(`Unhandled section type "${type}" during creation. Defaulting to Text.`);
                sectionData.type = 'text';
                sectionInstance = new TextSection(sectionData, sectionEl, callbacks);
        }

        this.sections.set(sectionId, sectionInstance);
        // Find existing data if any and update, otherwise push
        const existingIndex = this.sectionData.findIndex(s => s.id === sectionId);
        if (existingIndex > -1) {
            this.sectionData[existingIndex] = sectionData;
        } else {
            this.sectionData.push(sectionData);
        }


        if (isUserAction) {
            // We added a new section, need to sort, renumber, reorder DOM, update TOC
            this.normalizeOrdersAndRenumber(); // Sorts sectionData, updates order prop, updates DOM numbers
            this.reorderSectionsInDOM();       // Rearranges DOM elements to match sectionData order
            this.triggerTocUpdate();           // Updates the Table of Contents display
            this.handleEmptyState();           // Hides/shows empty state message
        }
        // If !isUserAction (initial load), these updates are called once in loadSectionsFromData

        return sectionInstance;
    }

    /** Returns appropriate initial content bytes and metadata for a section type */
    private getInitialContentInfo(type: SectionType): { initialContentBytes: string, contentType: string, format: string } {
        let rawContent: string | object;
        let contentType: string;
        let format: string;

        switch (type) {
            case 'text':
                rawContent = ''; // Empty paragraph or just empty string
                contentType = 'text/html';
                format = 'html';
                break;
            case 'drawing':
                // Empty Excalidraw scene
                rawContent = { elements: [], appState: {} };
                contentType = 'application/json';
                format = 'excalidraw/json'; // Specific format
                break;
            case 'plot':
                rawContent = {}; // Empty JSON object for plot config
                contentType = 'application/json';
                format = 'placeholder_plot/json'; // Placeholder format
                break;
            default:
                rawContent = '';
                contentType = 'text/plain';
                format = 'unknown';
        }

        // Convert rawContent to base64 string for contentBytes
        const contentString = typeof rawContent === 'string' ? rawContent : JSON.stringify(rawContent);
        const initialContentBytes = btoa(unescape(encodeURIComponent(contentString))); // String -> UTF8 bytes -> Base64

        return { initialContentBytes, contentType, format };
    }
    /** Returns appropriate default content for a section type */
    private getDefaultContent(type: SectionType): SectionData['content'] {
        switch (type) {
            case 'text': return '';
            case 'drawing': return { format: 'placeholder_drawing', data: {} };
            case 'plot': return { format: 'placeholder_plot', data: {} };
            default: return '';
        }
    }


    /**
     * Fetches metadata and then content for multiple section IDs and then loads them.
     */
    public async loadSectionContentsByIds(designId: string, sectionIds: string[]): Promise<void> {
        if (sectionIds.length === 0) {
            console.log("SectionManager: No section IDs provided to load.");
            this.handleEmptyState();
            return;
        }
        console.log(`SectionManager: Fetching metadata and content for ${sectionIds.length} sections...`);
        this.designId = designId; // Ensure designId is set for the manager

        const sectionDataPromises = sectionIds.map(async (id, index) => {
            try {
                // Step 1: Fetch Section Metadata
                const apiSectionMeta: V1Section = await DesignApi.designServiceGetSection({ designId: designId, sectionId: id });
                const sectionData = convertApiSectionToSectionData(apiSectionMeta); // Creates SectionData with content = null

                try {
                    // Step 2: Fetch Main Content ("main" assumed name)
                    // The response contains content metadata and the bytes (as base64 string)
                    const contentResp: V1GetContentResponse = await ContentApi.contentServiceGetContent({
                        designId: designId,
                        sectionId: id,
                        name: 'main' // Assuming primary content is named 'main'
                    });

                    // Store the raw fetched content (likely base64 string) into the sectionData.
                    // The section subclass (TextSection, DrawingSection) will handle decoding/parsing.
                    sectionData.content = contentResp.contentBytes || null; // Store base64 string or null

                    // Optional: Store content metadata if needed later
                    // sectionData.contentType = contentResp.content?.type;
                    // sectionData.contentFormat = contentResp.content?.format;

                    console.log(`SectionManager: Successfully fetched content for section ${id}`);


                } catch (contentError: any) {
                     // Handle cases where content might not exist (e.g., newly added section)
                     // Check if the error is a 404 specifically for content
                    if (contentError instanceof Response && contentError.status === 404) {
                         console.warn(`SectionManager: Content 'main' not found for section ${id}. Initializing content as null.`);
                         sectionData.content = null; // Explicitly null if not found
                    } else {
                        // Log other content fetch errors but potentially continue
                        console.error(`SectionManager: Failed to fetch content 'main' for section ${id}:`, contentError);
                        sectionData.content = null; // Set to null on other errors too? Or mark as error state? Null for now.
                    }
                }
                // Ensure order is set correctly based on initial fetch order
                sectionData.order = index + 1; // Set order based on the input sectionIds array index

                return sectionData; // Return the SectionData with content potentially populated

            } catch (metaError) {
                console.error(`SectionManager: Failed to fetch or convert section metadata for ${id}:`, metaError);
                return null; // Return null for failed metadata fetches
            }
        });

        const results = await Promise.all(sectionDataPromises);

        // Filter out null results (failed metadata loads)
        const loadedSectionsData = results.filter(data => data !== null) as SectionData[];
        // Data is already ordered based on the initial sectionIds array due to map index

        console.log(`SectionManager: Successfully processed ${loadedSectionsData.length} sections.`);
        this.loadSectionsFromData(loadedSectionsData); // Render using the updated data
    }


    /**
     * Loads multiple sections from provided data (typically from API processing).
     * Now receives SectionData potentially including raw 'content'.
     */
    public loadSectionsFromData(sectionsData: SectionData[]): void {
        if (!this.sectionsContainer) return;
        console.log("SectionManager: Loading sections from data:", sectionsData.length);
        this.clearAllSections();
        let maxIdNum = 0;

        // Ensure data is sorted by the 'order' property assigned during fetch
        // sectionsData.sort((a, b) => a.order - b.order);

        sectionsData.forEach(data => {
            const createdSection = this.createSectionInternal(data, false); // isUserAction = false
            if (createdSection) {
                // Existing ID number logic (unchanged)
                const idNumMatch = data.id.match(/\d+$/);
                if (idNumMatch) {
                    maxIdNum = Math.max(maxIdNum, parseInt(idNumMatch[0], 10));
                }
            }
        });

        this.nextSectionId = maxIdNum + 1;
        // normalizeOrdersAndRenumber might not be strictly needed if order is set correctly during load
        // but let's keep it for consistency for now.
        this.normalizeOrdersAndRenumber();
        this.reorderSectionsInDOM(); // Ensure DOM matches data order
        this.triggerTocUpdate();
        this.handleEmptyState();
        console.log("SectionManager: Sections loaded and rendered from data.");
    }

    /** Clears all sections and resets state */
     private clearAllSections(): void {
         // if (this.sectionsContainer) this.sectionsContainer.innerHTML = '';
         if (this.sectionsContainer) {
             // Remove only the actual section elements, not the empty state container
             const sectionElements = this.sectionsContainer.querySelectorAll('[data-section-id]');
             sectionElements.forEach(el => el.remove());
             console.log("SectionManager: Cleared section elements.");
         }

         this.sections.clear();
         this.sectionData = [];
         this.nextSectionId = 1;
         this.triggerTocUpdate(); // Update TOC to reflect emptiness
         this.handleEmptyState();
     }


    /** Normalize order values and update numbering */
     private normalizeOrdersAndRenumber(): void {
         this.sectionData.sort((a, b) => a.order - b.order);
         this.sectionData.forEach((sectionDataItem, index) => {
             sectionDataItem.order = index + 1;
             this.sections.get(sectionDataItem.id)?.updateNumber(index + 1);
         });
     }


    /** Delete a section */
    private deleteSection(sectionId: string): void {
        if (!this.currentDesignId) {
            console.error("Cannot delete section: Design ID not set.");
            // toastManager.showToast("Error", "Cannot delete section: Design ID missing.", "error");
            return;
        }
        if (!confirm('Are you sure you want to delete this section?')) return;


        console.log(`Attempting to delete section ${sectionId} from design ${this.currentDesignId}`);
        // Show loading indicator?

        // --- API Call FIRST ---
        const sectionInstance = this.sections.get(sectionId);
        if (!sectionInstance) {
            console.warn(`Section instance ${sectionId} not found during delete.`);
            return; // Should not happen if UI is consistent
        }

        DesignApi.designServiceDeleteSection({
            designId: this.currentDesignId,
            sectionId: sectionId
        }).then(() => {
            console.log(`API: Successfully deleted section ${sectionId}`);
            // --- Local Removal on API Success ---
            sectionInstance.removeElement(); // Use the public method
            this.sectionData = this.sectionData.filter(s => s.id !== sectionId);
            this.sections.delete(sectionId);
            this.normalizeOrdersAndRenumber();
            this.reorderSectionsInDOM(); // Ensure DOM order reflects data
            this.triggerTocUpdate();
            this.handleEmptyState();
            // toastManager.showToast("Section Deleted", "Section removed successfully.", "success", 2000);
        }).catch(error => {
            console.error(`API Error deleting section ${sectionId}:`, error);
            // toastManager.showToast("Delete Error", `Failed to delete section: ${error.message || 'Server Error'}`, "error");
            // Do NOT remove locally if API failed
        }).finally(() => {
            // Hide loading indicator?
        });
    }

    /** Common logic for moving sections via API */
    private async moveSectionApi(sectionId: string, moveUp: boolean): Promise<void> {
        if (!this.currentDesignId) {
             console.error("Cannot move section: Design ID not set.");
            // toastManager.showToast("Error", "Cannot move section: Design ID missing.", "error");
             return;
         }
        const currentIndex = this.sectionData.findIndex(s => s.id === sectionId);
        if (moveUp && currentIndex <= 0) return; // Cannot move first item up
        if (!moveUp && currentIndex >= this.sectionData.length - 1) return; // Cannot move last item down

        const targetIndex = moveUp ? currentIndex - 1 : currentIndex + 1;
        if (targetIndex < 0 || targetIndex >= this.sectionData.length) {
            console.warn("Calculated invalid target index for move:", { sectionId, moveUp, currentIndex, targetIndex });
            return; // Should be caught by above checks, but safeguard
        }

        const relativeSectionId = this.sectionData[targetIndex].id;
        // If moving up, we position BEFORE the item currently at the target index.
        // If moving down, we position AFTER the item currently at the target index.
        const position = moveUp ? V1PositionType.PositionTypeBefore : V1PositionType.PositionTypeAfter;

        console.log(`Attempting to move section ${sectionId} ${position} section ${relativeSectionId} in design ${this.currentDesignId}`);
        // Show loading indicator?

        try {
            await DesignApi.designServiceMoveSection({
                designId: this.currentDesignId,
                sectionId: sectionId, // Section being moved
                body: {
                    relativeSectionId: relativeSectionId,
                    position: position,
                }
            });
            console.log(`API: Successfully moved section ${sectionId}`);

            // --- Local Reorder on API Success ---
            // Adjust local order slightly to force resorting, then normalize
            const currentOrder = this.sectionData[currentIndex].order;
            const targetOrder = this.sectionData[targetIndex].order;
            // Give it a temporary order between the target and its neighbor
            this.sectionData[currentIndex].order = moveUp ? targetOrder - 0.5 : targetOrder + 0.5;

            this.reorderAndRenumber(); // This normalizes orders, updates DOM numbers, reorders DOM, triggers TOC
            // toastManager.showToast("Section Moved", "Section reordered successfully.", "success", 1500);

        } catch (error) {
            console.error(`API Error moving section ${sectionId}:`, error);
            // toastManager.showToast("Move Error", `Failed to move section: ${error.message || 'Server Error'}`, "error");
            // Do NOT reorder locally if API failed
        } finally {
             // Hide loading indicator?
        }
    }

    /** Move a section up */
    private moveSectionUp(sectionId: string): void {
        this.moveSectionApi(sectionId, true);
    }


    /** Move a section down */
    private moveSectionDown(sectionId: string): void {
        this.moveSectionApi(sectionId, false);
    }


    /** Reorders DOM, renumbers, and updates TOC */
    private reorderAndRenumber(): void {
         this.normalizeOrdersAndRenumber();
         this.reorderSectionsInDOM();
         this.triggerTocUpdate(); // Update TOC after reordering
    }


    /** Reorder section elements in the DOM based on the sorted sectionData */
    private reorderSectionsInDOM(): void {
        if (!this.sectionsContainer) return;
        this.sectionData.forEach(data => {
            const sectionEl = document.getElementById(data.id);
            if (sectionEl) this.sectionsContainer!.appendChild(sectionEl);
        });
    }

    /** Update section title METADATA only */
    private updateSectionTitle(sectionId: string, newTitle: string): void {
        if (!this.currentDesignId) {
            console.error("Cannot update section title: Design ID not set.");
            return;
        }
        const sectionData = this.sectionData.find(s => s.id === sectionId);
        if (sectionData) {
            sectionData.title = newTitle;
            this.triggerTocUpdate(); // Update TOC when title changes

            // --- API Call to update Section METADATA ---
            console.log(`Calling API to update title for section ${sectionId} in design ${this.currentDesignId}`);
            // Use the *metadata* update object creator
            const updatePayload = createApiSectionMetadataUpdateObject({ title: newTitle });

            DesignApi.designServiceUpdateSection({
                sectionDesignId: this.currentDesignId, // Route param 1
                sectionId: sectionId,                  // Route param 2
                body: updatePayload,                   // Body contains section with title and mask="title"
            }).then(updatedSectionMetadata => {
                console.log(`API: Successfully updated title for section ${sectionId}`, updatedSectionMetadata);
                // Optional: Update local data with server response metadata if needed (e.g., updated timestamp)
                // sectionData.title = updatedSectionMetadata.title; // Already set locally
                // toastManager.showToast("Title Saved", `Section "${newTitle}" title saved.`, "success", 1500);
            }).catch(error => {
                console.error(`API Error updating title for section ${sectionId}:`, error);
                // toastManager.showToast("Save Error", `Failed to save title for section "${newTitle}".`, "error");
                // TODO: Consider reverting UI?
            });
        }
    }

    /** Creates the simplified data structure needed by the TOC component */
    private getTocItemsInfo(): TocItemInfo[] {
        return this.sectionData
            .sort((a, b) => a.order - b.order) // Ensure sorted before mapping
            .map(section => ({
                id: section.id,
                title: section.title,
                order: section.order
            }));
    }


    /** Calls the update method on the TOC component, if available */
    private triggerTocUpdate(): void {
        if (this.tocComponent) {
            this.tocComponent.update(this.getTocItemsInfo());
            console.log("TOC update triggered.");
        } else {
            console.warn("Cannot trigger TOC update: TOC component not set.");
        }
    }


    /** Handle empty state visibility */
    public handleEmptyState(): void {
        if (!this.emptyStateEl || !this.fabAddSectionBtn) return;
        if (this.sectionData.length === 0) {
            this.emptyStateEl.classList.remove('hidden');
            this.fabAddSectionBtn.classList.add('hidden'); // Hide final button when empty
        } else {
            this.emptyStateEl.classList.add('hidden');
            this.fabAddSectionBtn.classList.remove('hidden'); // Show final button when not empty
        }
    }

    /** Get all section data formatted for the document model (INTERPRETED state) */
    public getDocumentSections(): DocumentSection[] {
        // This needs to get the *interpreted* content from each section instance,
        // not the raw content stored in sectionData.
        return this.sectionData
            .sort((a, b) => a.order - b.order)
            .map(sectionDataItem => {
                const sectionInstance = this.sections.get(sectionDataItem.id);
                if (sectionInstance) {
                    // getDocumentData() in BaseSection/subclasses should return the structured content
                    return sectionInstance.getDocumentData();
                }
                // Fallback if instance not found (should be rare)
                 console.warn(`Section instance not found for ID: ${sectionDataItem.id} during getDocumentSections.`);
                 // Cannot easily reconstruct interpreted content here, return metadata focus
                 return {
                     id: sectionDataItem.id,
                     title: sectionDataItem.title,
                     order: sectionDataItem.order,
                     type: sectionDataItem.type,
                     // Content representation here is problematic without the instance
                     content: sectionDataItem.content // Might be raw base64 or null
                 } as any; // Type assertion needed as content doesn't match DocumentSection types
            });
    }

    /** Notifies all managed sections that the application theme has changed. */
    public notifySectionsOfThemeChange(): void {
        console.log("SectionManager: Notifying all sections of theme change...");
        this.sections.forEach(section => {
            try {
                section.handleThemeChange();
            } catch (error) {
                 console.error(`Error handling theme change for section ${section.getId()}:`, error);
            }
        });
    }
}
