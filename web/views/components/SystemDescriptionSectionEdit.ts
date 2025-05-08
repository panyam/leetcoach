// FILE: web/views/components/SystemDescriptionSectionEdit.ts

import { SectionContent, ISectionEditComponent, SaveSuccessCallback, CancelCallback, SystemDescriptionContent } from './types';
import { TemplateLoader } from './TemplateLoader';
import { ContentApi, SystemModelApi } from './Api'; // Import APIs
import { ToastManager } from './ToastManager';

export class SystemDescriptionSectionEdit implements ISectionEditComponent {
    public element: HTMLElement; // Root element provided by container
    private designId: string;
    private sectionId: string;
    private onSaveSuccess: SaveSuccessCallback;
    private onCancel: CancelCallback;

    private templateLoader = new TemplateLoader();
    private toastManager: ToastManager;

    // Internal State
    private initialContent: SystemDescriptionContent = '';
    private isLoading: boolean = true;
    private isSaving: boolean = false;
    private isValidating: boolean = false;

    // DOM Elements
    private textareaElement: HTMLTextAreaElement | null = null;
    private saveButton: HTMLButtonElement | null = null;
    private cancelButton: HTMLButtonElement | null = null;
    private validateButton: HTMLButtonElement | null = null;


    constructor(
        element: HTMLElement,
        designId: string,
        sectionId: string,
        callbacks: { onSaveSuccess: SaveSuccessCallback, onCancel: CancelCallback }
    ) {
        this.element = element;
        this.designId = designId;
        this.sectionId = sectionId;
        this.onSaveSuccess = callbacks.onSaveSuccess;
        this.onCancel = callbacks.onCancel;
        this.toastManager = ToastManager.getInstance();

        console.log(`SystemDescriptionSectionEdit created for ${designId}/${sectionId}`);
        this.loadTemplateAndData();
    }

    private renderError(message: string): void {
        this.element.innerHTML = `<div class="p-4 text-red-500 dark:text-red-400">${message}</div>`;
    }

    private async loadTemplateAndData(): Promise<void> {
        this.isLoading = true;
        this.element.innerHTML = `<div class="p-4 text-center text-gray-500 dark:text-gray-400 italic">Loading editor...</div>`;

        // 1. Load Template
        const success = this.templateLoader.loadInto('system_description-section-edit', this.element);
        if (!success) {
            this.renderError("Failed to load editor template structure.");
            this.isLoading = false;
            return;
        }

        // 2. Find elements
        this.textareaElement = this.element.querySelector<HTMLTextAreaElement>('#system-dsl-textarea');
        this.saveButton = this.element.querySelector<HTMLButtonElement>('.section-edit-save');
        this.cancelButton = this.element.querySelector<HTMLButtonElement>('.section-edit-cancel');
        this.validateButton = this.element.querySelector<HTMLButtonElement>('.section-validate-dsl'); // Validate button in edit controls

        if (!this.textareaElement || !this.saveButton || !this.cancelButton || !this.validateButton) {
            this.renderError("Editor structure incomplete in template.");
            this.isLoading = false;
            return;
        }

        // 3. Fetch Initial Data
        try {
            const resp = await ContentApi.contentServiceGetContent({
                designId: this.designId,
                sectionId: this.sectionId,
                name: "main",
            });
            this.initialContent = resp.contentBytes ? atob(resp.contentBytes) : '';
            this.textareaElement.value = this.initialContent; // Populate textarea
            this.isLoading = false;
            this.bindControlEvents();
            console.log(`SystemDescriptionSectionEdit loaded initial content for ${this.sectionId}`);

        } catch (error: any) {
            console.error(`Error loading initial content for ${this.sectionId}:`, error);
            this.isLoading = false;
            if (error.status === 404) {
                 this.initialContent = '';
                 this.textareaElement.value = ''; // Ensure textarea is empty
                 this.bindControlEvents();
            } else {
                 this.renderError(`Failed to load DSL content: ${error.message || 'Unknown error'}`);
            }
        }
    }

    /** Binds handlers to Save, Cancel, Validate buttons */
    private bindControlEvents(): void {
        if (this.saveButton) this.saveButton.onclick = null;
        if (this.cancelButton) this.cancelButton.onclick = null;
        if (this.validateButton) this.validateButton.onclick = null;

        if (this.saveButton) { this.saveButton.onclick = (e) => { e.stopPropagation(); this.triggerSave(); }; }
        if (this.cancelButton) { this.cancelButton.onclick = (e) => { e.stopPropagation(); this.triggerCancel(); }; }
        if (this.validateButton) { this.validateButton.onclick = (e) => { e.stopPropagation(); this.handleValidateClick(this.validateButton); }; }
    }

    /** Gets content, calls API to save, signals container on success */
    private async triggerSave(): Promise<void> {
        if (this.isSaving || !this.textareaElement) return;

        const currentContent = this.textareaElement.value;
        this.isSaving = true;
        this.setControlLoading(true); // Show loading on buttons

        try {
            const encodedContent = btoa(unescape(encodeURIComponent(currentContent)));
            await ContentApi.contentServiceSetContent({
                designId: this.designId,
                sectionId: this.sectionId,
                name: "main", // Assuming DSL is 'main'
                contentBytes: encodedContent,
            });

            this.initialContent = currentContent; // Update baseline
            this.isSaving = false;
            this.toastManager.showToast("Saved", "System Model saved.", "success", 1500);
            // Don't hide loading, container will destroy
            await this.onSaveSuccess(); // Signal container

        } catch (error: any) {
             console.error(`Error saving SystemDescriptionSectionEdit ${this.sectionId}:`, error);
             this.toastManager.showToast("Save Error", `Failed to save: ${error.message || 'Server error'}`, "error");
             this.isSaving = false;
             this.setControlLoading(false); // Restore buttons on error
        }
    }

    /** Signals the container to cancel */
    private triggerCancel(): void {
        if (this.isSaving) return;
        this.onCancel();
    }

    /** Handles validation specific to the edit mode */
    private async handleValidateClick(buttonElement: HTMLButtonElement | null): Promise<void> {
        if (this.isValidating || !this.textareaElement) return;
        this.isValidating = true;
        this.showLoadingOnButton(buttonElement, true, 'Validating...');

        // IMPORTANT: Validate the *current* content in the textarea, not the saved version
        const contentToValidate = this.textareaElement.value;

        // We need to save temporarily or send content to validation endpoint.
        // Option 1: Save then validate (simpler API, but saves potentially invalid code)
        // Option 2: Modify validation API to accept content directly (cleaner, but API change)

        // Let's proceed with Option 1 for now, assuming validation happens on saved content.
        // User should save before validating in edit mode with current API.
        // OR we call the existing validation API which reads the saved file.
        console.warn("Validation in edit mode currently validates the *last saved* version.");
        this.toastManager.showToast("Info", "Validating last saved version...", "info", 2000);


        try {
            const response = await SystemModelApi.systemModelServiceValidateDSL({
                designId: this.designId,
                sectionId: this.sectionId,
                body: {},
            });
             if (response.isValid) {
                 this.toastManager.showToast("Validation Success", "Last saved DSL is valid.", "success", 2000);
             } else {
                 const errorMessages = response.errors?.map(err => `L${err.line}:C${err.column}: ${err.message}`) || ["Unknown validation error."];
                 this.toastManager.showToast("Validation Failed", errorMessages.slice(0,3).join('\n'), "error", 6000);
                 console.error("DSL Validation Errors:", response.errors);
             }
        } catch (error: any) {
            console.error(`Error validating DSL (edit mode) for ${this.sectionId}:`, error);
            this.toastManager.showToast("Validation Error", `Request failed: ${error.message || 'Server error'}`, "error");
        } finally {
            this.isValidating = false;
            this.showLoadingOnButton(buttonElement, false);
        }
    }


    /** Focuses the textarea */
    public focus(): void {
        // Use timeout for potential rendering delays
        setTimeout(() => this.textareaElement?.focus(), 50);
    }

    /** Cleans up */
    destroy(): void {
        console.log(`Destroying SystemDescriptionSectionEdit for ${this.sectionId}`);
        if (this.saveButton) this.saveButton.onclick = null;
        if (this.cancelButton) this.cancelButton.onclick = null;
        if (this.validateButton) this.validateButton.onclick = null;
        this.textareaElement = null; // Clear reference
        this.element.innerHTML = '';
    }

    /** Manages loading state for Save/Cancel/Validate buttons */
    private setControlLoading(isLoading: boolean): void {
        if (this.saveButton) this.saveButton.disabled = isLoading || this.isValidating;
        if (this.cancelButton) this.cancelButton.disabled = isLoading || this.isValidating;
        if (this.validateButton) this.validateButton.disabled = isLoading || this.isValidating; // Disable validate during save/validate

        // Add/remove visual indicator (e.g., to save button)
        const btn = this.saveButton; // Or validateButton depending on action
        if (btn) {
            const originalText = btn.dataset.originalText || 'Save'; // Adjust default text
             if (isLoading) {
                if (!btn.dataset.originalText) btn.dataset.originalText = btn.textContent || 'Save';
                const actionText = this.isSaving ? 'Saving...' : (this.isValidating ? 'Validating...' : 'Processing...');
                btn.innerHTML = `<svg class="animate-spin -ml-1 mr-2 h-4 w-4 inline-block" ...></svg> ${actionText}`;
             } else {
                 btn.innerHTML = originalText;
             }
        }
    }

    // Optional: Handle theme change if needed (e.g., for textarea styling)
     public handleThemeChange(): void {
         // Textarea styling might adapt automatically via Tailwind dark: classes.
         // No specific action needed here unless direct style manipulation is done.
     }

     // Copy showLoadingOnButton helper from SystemDescriptionView or SectionContainer if needed here too
     private showLoadingOnButton(button: HTMLButtonElement | null, isLoading: boolean, loadingText: string = "Processing...") {
        // ... implementation ...
        if (!button) return;
         const originalContent = button.dataset.originalContent || button.innerHTML;
         if (!button.dataset.originalContent && !isLoading) {
              button.dataset.originalContent = originalContent;
         } else if (!button.dataset.originalContent && isLoading) {
            button.dataset.originalContent = button.innerHTML;
         }

         if (isLoading) {
              button.disabled = true;
              button.innerHTML = `<svg class="animate-spin -ml-1 mr-2 h-4 w-4 inline-block" ...></svg> ${loadingText}`;
         } else {
              button.disabled = false;
              button.innerHTML = button.dataset.originalContent || 'Action';
         }
     }

}
