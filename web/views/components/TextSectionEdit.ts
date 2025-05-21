
import { SectionContent, ISectionEditComponent, SaveSuccessCallback, CancelCallback, TextContent } from './types';
import { TemplateLoader } from './TemplateLoader';
import { ContentApi } from './Api';
import { ToastManager } from './ToastManager';

import Editor from '@toast-ui/editor';
import '@toast-ui/editor/dist/toastui-editor.css'; // Import CSS
import codeSyntaxHighlight from '@toast-ui/editor-plugin-code-syntax-highlight';
import 'prismjs/themes/prism.css'; // Prism CSS for the plugin
import Prism from 'prismjs'; // Import Prism to make it available for the plugin

export class TextSectionEdit implements ISectionEditComponent {
    public element: HTMLElement;
    private designId: string;
    private sectionId: string;
    private onSaveSuccess: SaveSuccessCallback;
    private onCancel: CancelCallback;

    private templateLoader = new TemplateLoader();
    private toastManager: ToastManager;

    private editorInstance: any | null = null; // Instance of Toast UI Editor
    private editorContainerElement: HTMLDivElement | null = null; // Div where editor will be mounted

    private initialContent: TextContent = ''; // Stores raw Markdown
    private isLoading: boolean = true;
    private isSaving: boolean = false;

    private saveButton: HTMLButtonElement | null = null;
    private cancelButton: HTMLButtonElement | null = null;

    /**
     * Creates an instance of TextSectionEdit.
     * @param element The container element provided by SectionContainer.
     * @param initialContent The starting HTML content for the editor.
     * @param callbacks Object containing onSave and onCancel callbacks.
     */
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

        this.loadTemplateAndData();
    }

    // Editor components are NOT ok to exit fullscreen just like that
    public shouldExitFullscreen(): boolean {
      return false;
    }

    /** Loads template HTML, fetches initial data, then initializes editor */
    private async loadTemplateAndData(): Promise<void> {
        this.isLoading = true;
        this.element.innerHTML = `<div class="p-4 text-center text-gray-500 dark:text-gray-400 italic">Loading editor...</div>`;

        // 1. Load Template - this template now just needs a single div for the editor to mount
        const success = this.templateLoader.loadInto('text-section-edit', this.element);
        if (!success) {
            console.error("Failed to load text-section-edit template.");
            this.renderError("Could not load editor structure.");
            this.isLoading = false;
            return;
        }
        this.editorContainerElement = this.element.querySelector<HTMLDivElement>('#toast-editor-container'); // Use new ID
        this.saveButton = this.element.querySelector<HTMLButtonElement>('.section-edit-save');
        this.cancelButton = this.element.querySelector<HTMLButtonElement>('.section-edit-cancel');

        if (!this.editorContainerElement || !this.saveButton || !this.cancelButton) {
            this.renderError("Editor structure incomplete (missing editor container or controls).");
            this.isLoading = false;
            return;
        }

        // 2. Fetch Initial Data
        try {
            const resp = await ContentApi.contentServiceGetContent({
                designId: this.designId,
                sectionId: this.sectionId,
                name: "main", // Assuming text content is 'main'
            });
            this.initialContent = resp.contentBytes ? atob(resp.contentBytes) : '';
            this.isLoading = false;
            this.initializeEditor(); // Initialize after data is fetched
            this.bindControlEvents();

        } catch (error: any) {
            console.error(`Error loading initial content for TextSectionEdit ${this.sectionId}:`, error);
            this.isLoading = false;
            if (error.response.status === 404) {
                 this.initialContent = '';
                 this.initializeEditor();
                 this.bindControlEvents();
            } else {
                 this.renderError(`Failed to load initial content: ${error.message || 'Unknown error'}`);
            }
        }
    }

    /** Renders an error message within the component's element */
    private renderError(message: string): void {
        this.element.innerHTML = `<div class="p-4 text-red-500 dark:text-red-400">${message}</div>`;
    }


    /** Binds click events to the Save and Cancel buttons */
    private bindControlEvents(): void {
        if (this.saveButton) this.saveButton.onclick = null;
        if (this.cancelButton) this.cancelButton.onclick = null;

        if (this.saveButton) { this.saveButton.onclick = (e) => { e.stopPropagation(); this.triggerSave(); }; }
        if (this.cancelButton) { this.cancelButton.onclick = (e) => { e.stopPropagation(); this.triggerCancel(); }; }
    }

    /** Initializes the Toast UI Editor instance */
    private initializeEditor(): void {
        if (!this.editorContainerElement) return;

        // Destroy previous instance if any
        if (this.editorInstance) {
            try { this.editorInstance.destroy(); } catch(e) { console.error("Error destroying previous TUI Editor:", e);}
            this.editorInstance = null;
        }

        const isDarkMode = document.documentElement.classList.contains('dark');

        try {
            this.editorInstance = new Editor({
                el: this.editorContainerElement,
                initialValue: this.initialContent,
                initialEditType: 'markdown', // Or 'wysiwyg'
                previewStyle: 'vertical', // Or 'tab'
                height: '400px', // Adjust as needed
                theme: isDarkMode ? 'dark' : 'default',
                usageStatistics: false, // Disable statistics
                // plugins: [codeSyntaxHighlight] // If using NPM and plugin
                // If using CDN for plugin:
                plugins: [[codeSyntaxHighlight, { highlighter: Prism }]],
                // Ensure 'Prism' is globally available if using the CDN plugin for codeSyntaxHighlight
            });
            console.log(`Toast UI Editor initialized for TextSectionEdit ${this.sectionId}`);
        } catch (error) {
            console.error("Error initializing Toast UI Editor:", error);
            this.editorContainerElement.innerHTML = `<p class="text-red-500 p-4">Failed to initialize Markdown editor.</p>`;
        }
    }

    /** Handles the save action */
    private async triggerSave(): Promise<void> {
        if (this.isSaving || !this.editorInstance) {
            console.warn("Save already in progress or editor not ready.");
            return;
        }

        const currentMarkdown = this.editorInstance.getMarkdown(); // Get raw Markdown
        this.isSaving = true;
        this.setControlLoading(true);

        try {
            const encodedContent = btoa(unescape(encodeURIComponent(currentMarkdown)));
            console.log(`TextSectionEdit (TUI Markdown) saving content for ${this.sectionId}...`);

            await ContentApi.contentServiceSetContent({
                designId: this.designId,
                sectionId: this.sectionId,
                name: "main", // Still "main", but now it's Markdown
                contentBytes: encodedContent,
            });

            this.initialContent = currentMarkdown; // Update baseline
            this.isSaving = false;
            await this.onSaveSuccess();

        } catch (error: any) {
            console.error(`Error saving content for TextSectionEdit ${this.sectionId}:`, error);
            this.toastManager.showToast("Save Error", `Failed to save: ${error.message || 'Server error'}`, "error");
            this.isSaving = false;
            this.setControlLoading(false);
        }
    }

    /** Handles the cancel action */
    private triggerCancel(): void {
        if (this.isSaving) return; // Don't cancel while saving
        console.log(`TextSectionEdit cancelling for ${this.sectionId}`);
        this.onCancel();
    }


    /** Focuses the editor */
    public focus(): void {
       setTimeout(() => this.editorInstance?.focus(), 50);
    }

    /** Cleans up the editor instance */
    destroy(): void {
        console.log(`Destroying TextSectionEdit (TUI Markdown) for ${this.sectionId}`);
        if (this.editorInstance) {
            try { this.editorInstance.destroy(); } catch(e) { console.error(e); }
            this.editorInstance = null;
        }
        if (this.saveButton) this.saveButton.onclick = null;
        if (this.cancelButton) this.cancelButton.onclick = null;
        this.element.innerHTML = '';
    }

    /** Helper to manage loading state on Save/Cancel buttons */
    private setControlLoading(isLoading: boolean): void {
        if (this.saveButton) this.saveButton.disabled = isLoading;
        if (this.cancelButton) this.cancelButton.disabled = isLoading;
        // Add visual loading indicator to save button if desired
        if (this.saveButton) {
            const originalText = this.saveButton.dataset.originalText || 'Save';
            if (isLoading) {
                if (!this.saveButton.dataset.originalText) this.saveButton.dataset.originalText = this.saveButton.textContent || 'Save';
                this.saveButton.innerHTML = `<svg class="animate-spin -ml-1 mr-2 h-4 w-4 inline-block" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"> <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle> <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path> </svg> Saving...`;
            } else {
                this.saveButton.innerHTML = originalText;
            }
        }
    }

     // Handle theme change if the editor is active
     public handleThemeChange(): void {
        if (this.editorInstance) {
            console.log("TextSectionEdit (TUI Markdown) handling theme change...");
            const isDarkMode = document.documentElement.classList.contains('dark');
            // Toast UI Editor usually requires re-initialization for theme change
            // or has a specific method. Let's try re-init.
            const currentMarkdown = this.editorInstance.getMarkdown();
            this.initialContent = currentMarkdown;
            this.initializeEditor(); // This will destroy and recreate
            // Note: Re-binding controls might be needed if initializeEditor clears the whole `this.element`
            this.bindControlEvents();
        }
     }
}
