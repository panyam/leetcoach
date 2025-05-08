// FILE: web/views/components/SystemDescriptionSection.ts

import { ContentApi, SystemModelApi } from './Api';
import { BaseSection } from './BaseSection';
import { SectionData, SectionContent, SectionCallbacks, SystemDescriptionContent } from './types';
import { V1ValidateDSLResponse, V1GenerateDiagramResponse } from './apiclient'; // <-- Import response type

export class SystemDescriptionSection extends BaseSection {
    // Simple code icon (like a semicolon in braces)
    protected static readonly ICON_SVG = `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-full h-full"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75h-10.5l-1.125 1.125c-.59.59-.188 1.625.563 1.625h12.75c.75 0 1.153-1.035.563-1.625L17.25 6.75zm0 0V4.5m0 2.25V9m-10.5 0V4.5m0 4.5V9m1.125 5.625h8.25M10.875 14.625V18m2.25-3.375V18" /> <path stroke-linecap="round" d="M12 16.5h.01" /> </svg>`;

    private dslContent: SystemDescriptionContent = ''; // Store the DSL string
    private textAreaElement: HTMLTextAreaElement | null = null; // <-- Reference to textarea

    // New elements for Diagram display
    private diagramContainer: HTMLDivElement | null = null;
    private isDiagramLoading: boolean = false; // State for diagram loading

    constructor(data: SectionData, element: HTMLElement, callbacks: SectionCallbacks = {}) {
        super(data, element, callbacks);
        // No special initialization needed here for Phase 1
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


    protected getSectionTypeTitle(): string {
      return "System Model"; // User-facing title
    }

    protected getSectionIconSvg(): string {
      return SystemDescriptionSection.ICON_SVG;
    }

    protected populateViewContent(): void {
        this.diagramContainer = this._findElement<HTMLDivElement>('#generated-diagram-container');
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

        // Initialize diagram container state (e.g., show placeholder text)
        this.updateDiagramDisplay(null, false); // Show initial placeholder

        // **Bind specific view actions AFTER content is populated**
        this.bindSpecificViewActions();
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
        // Find and potentially disable diagram container/buttons in edit mode if needed
        this.diagramContainer = this._findElement<HTMLDivElement>('#generated-diagram-container');
        this.diagramContainer?.classList.add('hidden'); // Hide diagram area in edit mode

        // **Bind specific edit actions AFTER content is populated**
        this.bindSpecificEditActions();
    }

    /** Binds actions specific to the view mode UI */
    protected bindSpecificViewActions(): void {
        console.log(`Binding specific view actions for ${this.sectionId}`);
        const validateButton = this._findElement<HTMLButtonElement>('.view-actions .section-validate-dsl', false);
        const generateDiagramButton = this._findElement<HTMLButtonElement>('.view-actions .section-generate-diagram', false);
        // const runAnalysisButton = this._findElement<HTMLButtonElement>('.view-actions .section-run-analysis', false); // Future

        if (validateButton) {
            validateButton.onclick = (e) => {
                e.stopPropagation(); // *** Crucial: Stop event from bubbling ***
                this.handleValidateClick(validateButton);
            };
        } else { console.warn(`View Validate button not found for ${this.sectionId}`); }

        if (generateDiagramButton) {
            generateDiagramButton.onclick = (e) => {
                e.stopPropagation(); // *** Crucial: Stop event from bubbling ***
                this.handleGenerateDiagramClick(generateDiagramButton);
            };
        } else { console.warn(`View Generate Diagram button not found for ${this.sectionId}`); }

        // Bind run analysis later...
    }

    /** Binds actions specific to the edit mode UI */
    protected bindSpecificEditActions(): void {
        console.log(`Binding specific edit actions for ${this.sectionId}`);
        const validateButton = this._findElement<HTMLButtonElement>('.edit-controls .section-validate-dsl', false);
        // Note: Save/Cancel are likely handled by BaseSection's delegated handler,
        // but could be bound here too if needed for specific subclass logic.

        if (validateButton) {
            validateButton.onclick = (e) => {
                e.stopPropagation(); // *** Crucial: Stop event from bubbling ***
                this.handleValidateClick(validateButton);
            };
        } else { console.warn(`Edit Validate button not found for ${this.sectionId}`); }
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
        this.showLoadingOnButton(buttonElement, true, 'Validating...');

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


    // --- NEW: Diagram Generation Logic ---
    private async handleGenerateDiagramClick(buttonElement: HTMLButtonElement | null): Promise<void> {
        console.log(`Generate Diagram clicked for section ${this.sectionId}`);
        if (this.isDiagramLoading) return; // Prevent concurrent generation

        this.isDiagramLoading = true;
        this.showLoadingOnButton(buttonElement, true, 'Generating...');
        this.updateDiagramDisplay('<p class="text-gray-500 dark:text-gray-400 italic">Generating diagram...</p>', true); // Show loading state in diagram area

        try {
            const response: V1GenerateDiagramResponse = await SystemModelApi.systemModelServiceGenerateDiagram({
                designId: this.designId,
                sectionId: this.sectionId,
                body: {} // Empty body for now
            });

            console.log("Received diagram response:", response.diagramFormat); // Log format

            if (response.diagramContent) {
                if (response.diagramFormat === 'svg') {
                    this.updateDiagramDisplay(response.diagramContent, false); // Display SVG
                    this.toastManager.showToast("Diagram Generated", "Diagram updated successfully.", "success", 1500);
                } else if (response.diagramFormat === 'mermaid') {
                    // TODO: Add Mermaid rendering logic later
                    console.warn("Mermaid diagram format received but rendering not implemented yet.");
                    this.updateDiagramDisplay(`<pre class="text-xs">${response.diagramContent}</pre>`, false); // Show code for now
                    this.toastManager.showToast("Diagram Generated", "Mermaid code generated (preview TBD).", "info", 2000);
                } else {
                    console.warn("Received unsupported diagram format:", response.diagramFormat);
                    this.updateDiagramDisplay(`<p class="text-yellow-500">Received unsupported diagram format: ${response.diagramFormat}</p>`, false);
                    this.toastManager.showToast("Warning", `Received diagram in unsupported format: ${response.diagramFormat}`, "warning");
                }
            } else {
                this.updateDiagramDisplay('<p class="text-yellow-500">Diagram generation returned empty content.</p>', false);
                this.toastManager.showToast("Warning", "Diagram generation resulted in empty content.", "warning");
            }

        } catch (error: any) {
            console.error(`Error generating diagram for section ${this.sectionId}:`, error);
            const errorMsg = error.message || (error.response ? await error.response.text() : 'Server error');
            this.toastManager.showToast("Diagram Error", `Failed: ${errorMsg}`, "error");
            this.updateDiagramDisplay(`<p class="text-red-500">Error generating diagram: ${errorMsg}</p>`, false); // Show error in diagram area
        } finally {
            this.isDiagramLoading = false;
            this.showLoadingOnButton(buttonElement, false); // Restore button
        }
    }

    // --- NEW: Helper to update diagram display area ---
    private updateDiagramDisplay(content: string | null, isLoading: boolean): void {
         if (!this.diagramContainer) {
             // Try to find it again in case populateViewContent hasn't run yet or was cleared
             this.diagramContainer = this._findElement<HTMLDivElement>('#generated-diagram-container');
             if (!this.diagramContainer) {
                console.warn(`Cannot update diagram display: container not found for ${this.sectionId}`);
                return;
             }
         }

        // Always ensure container is visible if we're trying to update it (unless in edit mode)
        if (this.mode === 'view') {
            this.diagramContainer.classList.remove('hidden');
        }

        if (isLoading) {
             this.diagramContainer.innerHTML = `<div class="p-4 text-center text-gray-500 dark:text-gray-400 italic">Loading diagram...</div>`;
        } else if (content) {
             // Directly set innerHTML. Assumes content is safe (e.g., sanitized SVG from backend or trusted Mermaid output).
             // BE CAREFUL with unsanitized content here due to potential XSS.
             this.diagramContainer.innerHTML = content;
             // If it's Mermaid, trigger rendering (add in Phase 3)
             // if (format === 'mermaid') { mermaid.run(...); }
        } else {
             // Show placeholder if content is null/empty
             this.diagramContainer.innerHTML = `<p class="text-gray-500 dark:text-gray-400 italic p-4 text-center">Click "Generate/Refresh Diagram" to view.</p>`;
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
