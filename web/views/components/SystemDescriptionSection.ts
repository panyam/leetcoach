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
    private validateButton: HTMLButtonElement | null = null; // <-- Reference to validate button


    constructor(data: SectionData, element: HTMLElement, callbacks: SectionCallbacks = {}) {
        super(data, element, callbacks);
        // No special initialization needed here for Phase 1
        this.validateButton = this._findElement<HTMLButtonElement>('.section-validate-dsl', false);
        this.validateButton?.classList.remove('hidden'); // Make validate button visible
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
    }

    protected populateEditContent(): void {
        this.textAreaElement = this._findElement<HTMLTextAreaElement>('#system-dsl-textarea');
        if (this.textAreaElement) {
             this.textAreaElement.value = this.dslContent;
             this.textAreaElement.focus();
             console.log(`Textarea initialized for section ${this.data.id}`);
        } else {
             console.error(`Edit content textarea not found for system_description section ${this.data.id}`);
             // Maybe render an error message in the content container
             this.contentContainer!.innerHTML = `<p class="text-red-500 p-4">Error: Textarea editor target not found.</p>`;
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
        this.isLoading = true; // Indicate loading state (optional)

        try {
            // Base64 encode the DSL string before sending
            const encodedContent = btoa(unescape(encodeURIComponent(currentContent)));

            await ContentApi.contentServiceSetContent({
                designId: this.designId,
                sectionId: this.sectionId,
                name: "main", // Or "dsl"
                contentBytes: encodedContent, // Send encoded bytes
            });

            // Update internal state *after* successful save
            this.dslContent = currentContent;
            // this.updateLastSavedTimestamp(); // Method from BaseSection assumed
            this.toastManager.showToast("Saved", "System Model saved successfully.", "success", 1500);
            this.switchToViewMode(false); // Switch to view mode without re-saving

        } catch (error: any) {
            console.error(`Error saving System Description section ${this.data.id}:`, error);
            this.toastManager.showToast("Save Error", `Failed to save: ${error.message || 'Server error'}`, "error");
            // Remain in edit mode on error
        } finally {
            this.isLoading = false;
        }
    }

    // Clean up Monaco Editor instance
    protected override cleanupEditMode(): void {
        this.textAreaElement = null; // Clear reference
        console.log(`Cleaned up edit mode for textarea section ${this.data.id}`);
    }

    // Bind the validate button click listener
    protected override _bindEvents(): void {
        super._bindEvents(); // Call base class binding

        // Add specific listener for the validate button
        if (this.validateButton) {
            this.validateButton.removeEventListener('click', this.handleValidateClick); // Prevent duplicates
            this.validateButton.addEventListener('click', this.handleValidateClick.bind(this));
        }
    }


    // --- Validation Logic ---
    private async handleValidateClick(event: MouseEvent): Promise<void> {
        event.stopPropagation(); // Prevent triggering other section clicks
        if (!this.validateButton) return;

        console.log(`Validate button clicked for section ${this.sectionId}`);
        const originalButtonText = this.validateButton.innerHTML; // Store original icon/text
        this.setLoadingOnButton(this.validateButton, true); // Show loading state

        try {
            const response = await SystemModelApi.systemModelServiceValidateDSL({
                designId: this.designId,
                sectionId: this.sectionId,
                // Body is empty for this POST request as defined in proto annotation
                body: {},
            });

            if (response.isValid) {
                this.toastManager.showToast("Validation Successful", "DSL syntax is valid.", "success", 2000);
            } else {
                // Format errors for toast message
                const errorMessages = response.errors?.map((err: any) =>
                     `L${err.line ?? '?'}:C${err.column ?? '?'} - ${err.message}` // Handle potential null line/col
                 ) || ["Unknown validation error."];
                const fullMessage = `Validation Failed:\n - ${errorMessages.join('\n - ')}`;
                 // Show multi-line error toast (ToastManager might need adjustment for this)
                this.toastManager.showToast("Validation Failed", fullMessage, "error", 8000); // Longer duration for errors
                console.warn("Validation Errors:", response.errors);
            }

        } catch (error: any) {
            console.error(`Error calling validation API for section ${this.sectionId}:`, error);
            const errorMsg = error.message || (error.response ? await error.response.text() : 'Unknown API error');
            this.toastManager.showToast("Validation Error", `Failed to validate: ${errorMsg}`, "error");
        } finally {
            this.setLoadingOnButton(this.validateButton, false, originalButtonText); // Restore button
        }
    }

    // Helper to show/hide loading state on a button
    private setLoadingOnButton(button: HTMLButtonElement | null, isLoading: boolean, defaultContent: string = ''): void {
        if (!button) return;
        if (isLoading) {
            button.disabled = true;
            button.innerHTML = `<svg class="animate-spin h-5 w-5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"> <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle> <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path> </svg>`; // Spinner SVG
        } else {
            button.disabled = false;
            button.innerHTML = defaultContent; // Restore original content (icon)
        }
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
}
