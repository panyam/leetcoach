// FILE: web/views/components/SystemDescriptionSection.ts

import { ContentApi, SystemModelApi } from './Api';
import { BaseSection } from './BaseSection';
import { SectionData, SectionContent, SectionCallbacks, SystemDescriptionContent } from './types';
import { V1ValidateDSLResponse } from './apiclient'; // <-- Import response type

export class SystemDescriptionSection extends BaseSection {
    // Simple code icon (like a semicolon in braces)
    protected static readonly ICON_SVG = `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-full h-full"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75h-10.5l-1.125 1.125c-.59.59-.188 1.625.563 1.625h12.75c.75 0 1.153-1.035.563-1.625L17.25 6.75zm0 0V4.5m0 2.25V9m-10.5 0V4.5m0 4.5V9m1.125 5.625h8.25M10.875 14.625V18m2.25-3.375V18" /> <path stroke-linecap="round" d="M12 16.5h.01" /> </svg>`;

    private dslContent: SystemDescriptionContent = ''; // Store the DSL string
    private textAreaElement: HTMLTextAreaElement | null = null; // <-- Reference to textarea

    constructor(data: SectionData, element: HTMLElement, callbacks: SectionCallbacks = {}) {
        super(data, element, callbacks);
        // No special initialization needed here for Phase 1
        this._bindViewModeActions();
    }

    // Override _bindEvents to ensure edit mode validation button is also captured if needed
    protected override _bindEvents(): void {
        super._bindEvents(); // Call base class binding

        // Add specific handler for validate in edit mode if base delegation isn't enough
        this.element.addEventListener('click', (event: MouseEvent) => {
             const target = event.target as HTMLElement;
             const validateButtonEdit = target.closest('.section-edit-form .section-validate-dsl');
             if (this.mode === 'edit' && validateButtonEdit) {
                 console.log("Validate button clicked in EDIT mode");
                 this.handleValidateClick(validateButtonEdit as HTMLButtonElement); // Pass button for loading state
             }
        });
        // Note: BaseSection's delegation might already cover this if buttons have same class
    }


    // Bind actions specific to view mode action bar
    private _bindViewModeActions(): void {
        this.element.addEventListener('click', (event: MouseEvent) => {
            if (this.mode !== 'view') return; // Only act in view mode

            const target = event.target as HTMLElement;
            const validateButton = target.closest('.view-actions .section-validate-dsl');
            // Add listeners for generate/analyze buttons here when implemented

            if (validateButton) {
                 console.log("Validate button clicked in VIEW mode");
                 this.handleValidateClick(validateButton as HTMLButtonElement);
            }
        });
    }

    protected getSectionTypeTitle(): string {
      return "System Model"; // User-facing title
    }

    protected getSectionIconSvg(): string {
      return SystemDescriptionSection.ICON_SVG;
    }

    protected populateViewContent(): void {
        const viewContent = this._findElement<HTMLDivElement>('.section-view-content');
        const codeBlock = viewContent?.querySelector('code'); // Target the <code> block
        if (codeBlock) {
            const initialContent = this.dslContent.trim().length > 0
                ? this.dslContent
                : '// No DSL content yet. Click Edit to start.';
            codeBlock.textContent = initialContent; // Set text content for <pre><code>
            // Apply syntax highlighting here in Phase 2 (e.g., using Prism.js)
            // Example: Prism.highlightElement(codeBlock);
        } else {
            console.warn(`View code block area not found for system_description section ${this.data.id}`);
            if(viewContent) viewContent.innerHTML = `<p class="text-gray-400 italic">Error: View structure incorrect.</p>`;
        }
        // Re-bind view mode actions AFTER populating view content ensures buttons exist
        this._bindViewModeActions();
    }

    protected populateEditContent(): void {
        this.textAreaElement = this._findElement<HTMLTextAreaElement>('#system-dsl-textarea');
        if (this.textAreaElement) {
             this.textAreaElement.value = this.dslContent;
             this.textAreaElement.focus();
             console.log(`Textarea initialized for section ${this.data.id}`);
        } else {
            console.error(`Edit content textarea not found for system_description section ${this.data.id}`);
            const editContainer = this._findElement('.section-edit-form');
            if(editContainer) editContainer.innerHTML = `<p class="text-red-500 p-4">Error: Textarea element missing.</p>`;
        }
    }

    protected getContentFromEditMode(): SystemDescriptionContent {
        if (this.textAreaElement) {
            return this.textAreaElement.value || '';
        }
        console.warn(`Textarea element not available when getting content for section ${this.data.id}.`);
        return this.dslContent; // Fallback
    }

    protected override async refreshContentFromServer() {
        this.dslContent = ""; // Reset content
        try {
            // Assuming DSL is stored in 'main' content for now
            const resp = await ContentApi.contentServiceGetContent({
              designId: this.designId,
              sectionId: this.sectionId,
              name: "main", // Or "dsl" if decided
            });
            // Content is base64 encoded
            this.dslContent = resp.contentBytes ? atob(resp.contentBytes) : "";
             console.log(`Refreshed DSL content for ${this.sectionId}, length: ${this.dslContent.length}`);
        } catch (err: any) {
            if (err.status !== 404) { // Don't log error for not found (means new section)
              console.error(`Error loading DSL content for section ${this.sectionId}: `, err);
            } else {
               console.log(`No existing DSL content found for new section ${this.sectionId}`);
            }
             this.dslContent = ""; // Ensure it's empty on error/not found
        }
    }

    public async handleSaveClick(): Promise<void> {
        if (this.mode !== 'edit') return; // Only save if in edit mode
        const currentContent = this.getContentFromEditMode();
        console.log(`Saving System Description section ${this.data.id}.`);
        this.isLoading = true;
        const saveButton = this._findElement<HTMLButtonElement>('.section-edit-save');
        this.showLoadingOnButton(saveButton, true, 'Saving...');

        try {
            const encodedContent = btoa(unescape(encodeURIComponent(currentContent)));

            await ContentApi.contentServiceSetContent({
                designId: this.designId,
                sectionId: this.sectionId,
                name: "main", // Or "dsl"
                contentBytes: encodedContent,
            });

            this.dslContent = currentContent;
            // this.updateLastSavedTimestamp(); // Assume BaseSection handles this maybe?
            this.toastManager.showToast("Saved", "System Model saved successfully.", "success", 1500);
            this.switchToViewMode(false);

        } catch (error: any) {
            console.error(`Error saving System Description section ${this.data.id}:`, error);
            this.toastManager.showToast("Save Error", `Failed to save: ${error.message || 'Server error'}`, "error");
        } finally {
            this.isLoading = false;
            this.showLoadingOnButton(saveButton, false);
        }
    }

    // Clean up Monaco Editor instance
    protected override cleanupEditMode(): void {
        this.textAreaElement = null; // Clear reference
        console.log(`Cleaned up edit mode for textarea section ${this.data.id}`);
    }

    // Implement abstract methods from BaseSection
    protected resizeContentForFullscreen(isEntering: boolean): void {
        console.log(`SystemDescriptionSection ${this.data.id}: Resizing content for fullscreen=${isEntering}.`);
    }

    protected override updateInternalContent(newContent: SectionContent): void {
        if (typeof newContent === 'string') {
            this.dslContent = newContent;
        } else {
             console.warn(`SystemDescriptionSection ${this.data.id}: Received non-string content during updateInternalContent`, newContent);
        }
    }

     // Handle theme changes for the editor
     public override handleThemeChange(): void {
         console.log(`SystemDescriptionSection ${this.data.id}: Handling theme change.`);
     }

    // --- Validation Logic ---
    private async handleValidateClick(buttonElement: HTMLButtonElement | null): Promise<void> {
        console.log(`Validate DSL clicked for section ${this.sectionId}`);
        if (this.isLoading) return; // Prevent concurrent validation

        this.isLoading = true;
        this.showLoadingOnButton(buttonElement, true);

        try {
            const response = await SystemModelApi.systemModelServiceValidateDSL({
                designId: this.designId,
                sectionId: this.sectionId,
                body: {} // Body might be needed depending on gateway generation
            });

            if (response.isValid) {
                this.toastManager.showToast("Validation Success", "DSL syntax is valid.", "success", 2000);
            } else {
                const errorMessages = response.errors?.map(err =>
                    `L${err.line}:C${err.column}: ${err.message}`
                ) || ["Unknown validation error."];
                // Show first few errors in toast, maybe log all to console
                const toastMessage = errorMessages.slice(0, 3).join('\n');
                this.toastManager.showToast("Validation Failed", toastMessage, "error", 5000);
                console.error("DSL Validation Errors:", response.errors);
            }

        } catch (error: any) {
            console.error(`Error validating DSL for section ${this.sectionId}:`, error);
            const errorMsg = error.message || (error.response ? await error.response.text() : 'Server error');
            this.toastManager.showToast("Validation Error", `Failed: ${errorMsg}`, "error");
        } finally {
            this.isLoading = false;
            this.showLoadingOnButton(buttonElement, false);
        }
    }

    // Helper to show/hide loading state on a button
    private showLoadingOnButton(button: HTMLButtonElement | null, isLoading: boolean, loadingText: string = "Processing...") {
         if (!button) return;
         const originalContent = button.dataset.originalContent || button.innerHTML; // Store original content if not already stored
         if (!button.dataset.originalContent) {
             button.dataset.originalContent = originalContent;
         }

         if (isLoading) {
             button.disabled = true;
             button.innerHTML = `<svg class="animate-spin -ml-1 mr-2 h-4 w-4 inline-block" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"> <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle> <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path> </svg> ${loadingText}`;
         } else {
             button.disabled = false;
             button.innerHTML = originalContent; // Restore original content
         }
     }

}
