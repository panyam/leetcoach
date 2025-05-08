
// FILE: web/views/components/TextSectionEdit.ts

import { SectionContent, ISectionEditComponent, SaveSuccessCallback, CancelCallback, TextContent } from './types';
import { TemplateLoader } from './TemplateLoader'; // To load the edit template HTML
import { ContentApi } from './Api'; // Import API
import { ToastManager } from './ToastManager'; // For feedback

// Import TinyMCE types and core
import tinymce, { Editor } from 'tinymce/tinymce';
// Import necessary TinyMCE modules (theme, icons, plugins) - ensure these are imported somewhere globally or managed by your build process
import 'tinymce/themes/silver/theme';
import 'tinymce/icons/default/icons';
import 'tinymce/models/dom/model';
import 'tinymce/plugins/autolink';
import 'tinymce/plugins/lists';
import 'tinymce/plugins/link';
import 'tinymce/plugins/image';
import 'tinymce/plugins/charmap';
import 'tinymce/plugins/preview';
import 'tinymce/plugins/anchor';
import 'tinymce/plugins/searchreplace';
import 'tinymce/plugins/visualblocks';
import 'tinymce/plugins/code';
import 'tinymce/plugins/fullscreen';
import 'tinymce/plugins/insertdatetime';
import 'tinymce/plugins/media';
import 'tinymce/plugins/table';
import 'tinymce/plugins/help';
import 'tinymce/plugins/wordcount';
import 'tinymce/plugins/autoresize';


export class TextSectionEdit implements ISectionEditComponent {
    private onSaveSuccess: SaveSuccessCallback;
    private onCancel: CancelCallback;
    private editorInstance: Editor | null = null;
    private templateLoader = new TemplateLoader();
    private toastManager: ToastManager;
    private saveButton: HTMLButtonElement | null = null;
    private cancelButton: HTMLButtonElement | null = null;

    private initialContent: TextContent = ''; // Store initially loaded content
    private isLoading: boolean = true; // For initial data load
    private isSaving: boolean = false; // For save operation

    // DOM Elements within the template
    private editorTargetElement: HTMLElement | null = null;


    /**
     * Creates an instance of TextSectionEdit.
     * @param element The container element provided by SectionContainer.
     * @param initialContent The starting HTML content for the editor.
     * @param callbacks Object containing onSave and onCancel callbacks.
     */
    constructor(
        public element: HTMLElement,
        public designId: string,
        public sectionId: string,
        callbacks: { onSaveSuccess: SaveSuccessCallback, onCancel: CancelCallback }
    ) {
        this.element = element;
        this.designId = designId;
        this.sectionId = sectionId;
        this.onSaveSuccess = callbacks.onSaveSuccess;
        this.onCancel = callbacks.onCancel;
        this.toastManager = ToastManager.getInstance();

        console.log(`TextSectionEdit created for ${designId}/${sectionId}`);
        this.loadTemplateAndData(); // Load template and fetch initial data
    }

    /** Loads template HTML, fetches initial data, then initializes editor */
    private async loadTemplateAndData(): Promise<void> {
        this.isLoading = true;
        this.element.innerHTML = `<div class="p-4 text-center text-gray-500 dark:text-gray-400 italic">Loading editor...</div>`; // Initial loading message

        // 1. Load Template Structure
        const success = this.templateLoader.loadInto('text-section-edit', this.element);
        if (!success) {
            console.error("Failed to load text-section-edit template.");
            this.renderError("Could not load editor structure.");
            this.isLoading = false;
            return;
        }
        this.editorTargetElement = this.element.querySelector<HTMLElement>('.text-editor-target');
        this.saveButton = this.element.querySelector<HTMLButtonElement>('.section-edit-save');
        this.cancelButton = this.element.querySelector<HTMLButtonElement>('.section-edit-cancel');

        if (!this.editorTargetElement || !this.saveButton || !this.cancelButton) {
            console.error("Essential elements (.text-editor-target, .section-edit-save, .section-edit-cancel) not found within text-section-edit template.");
            this.renderError("Editor structure incomplete in template.");
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
            console.log(`TextSectionEdit loaded initial content for ${this.sectionId}, length: ${this.initialContent.length}`);

            // 3. Initialize Editor (only after data is loaded)
            this.initializeEditor(this.editorTargetElement); // Pass target
            this.bindControlEvents();
            this.isLoading = false;
            // Editor init handles its own placeholder/content setting

        } catch (error: any) {
            console.error(`Error loading initial content for TextSectionEdit ${this.sectionId}:`, error);
            this.isLoading = false;
            if (error.status === 404) {
                 // Content doesn't exist (new section), initialize editor with empty
                 this.initialContent = '';
                 this.initializeEditor(this.editorTargetElement);
                 this.bindControlEvents();
            } else {
                 // Other fetch error
                 this.renderError(`Failed to load initial content: ${error.message || 'Unknown error'}`);
            }
        }
    }

    /** Renders an error message within the component's element */
    private renderError(message: string): void {
        this.element.innerHTML = `<div class="p-4 text-red-500 dark:text-red-400">${message}</div>`;
    }

    /** Initializes the TinyMCE editor instance */
    private initializeEditor(targetElement: HTMLElement): void {
        const isDarkMode = document.documentElement.classList.contains('dark');
        // Assuming static assets are served correctly
        const tinyMCEPublicPath = '/static/js/gen/'; // Adjust if necessary

        // Ensure any previous instance is removed *before* initializing a new one
        if (this.editorInstance) {
            console.warn("Removing lingering TinyMCE instance before re-initialization.");
            try { tinymce.remove(this.editorInstance); } catch(e) { console.error(e); }
            this.editorInstance = null;
        }

        tinymce.init({
            target: targetElement,
            plugins: 'autolink lists link image charmap preview anchor searchreplace visualblocks code fullscreen insertdatetime media table help wordcount autoresize',
            toolbar: 'undo redo | formatselect | bold italic backcolor | alignleft aligncenter alignright alignjustify | bullist numlist outdent indent | removeformat | help',
            content_style: 'body { font-family:Helvetica,Arial,sans-serif; font-size:14px }',
            height: 300,
            menubar: false,
            statusbar: false,
            autoresize_bottom_margin: 10,
            autoresize_min_height: 200,
            skin_url: `${tinyMCEPublicPath}skins/ui/${isDarkMode ? 'oxide-dark' : 'oxide'}`,
            content_css: `${tinyMCEPublicPath}skins/content/${isDarkMode ? 'dark' : 'default'}/content.min.css`,
            license_key: 'gpl', // Or your commercial key
            setup: (editor: Editor) => {
                editor.on('init', () => {
                    editor.setContent(this.initialContent || '');
                    this.editorInstance = editor; // Store instance only on successful init
                    // Don't focus immediately, let container call focus() if needed
                    // editor.focus();
                    console.log(`TinyMCE initialized for TextSectionEdit`);
                });
                editor.on('remove', () => {
                    // Clear instance ref if editor is removed externally/during cleanup
                    if (this.editorInstance === editor) {
                        this.editorInstance = null;
                        console.log(`Cleaned up editor instance ref on 'remove' event`);
                    }
                });
                 // Optional: Add Cmd/Ctrl+Enter shortcut if desired
                editor.on('keydown', (e: KeyboardEvent) => {
                     if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                         e.preventDefault();
                         this.triggerSave(); // Call internal save handler
                     }
                });
            }
        }).catch((error: any) => {
            console.error(`Error initializing TinyMCE for TextSectionEdit:`, error);
            targetElement.innerHTML = `<p class="text-red-500">Error initializing Rich Text Editor.</p>`;
        });
    }

    /** Binds click events to the Save and Cancel buttons */
    private bindControlEvents(): void {
        if (this.saveButton) {
            this.saveButton.onclick = (e) => {
                e.stopPropagation(); // Prevent bubbling
                this.triggerSave();
            };
        }
        if (this.cancelButton) {
            this.cancelButton.onclick = (e) => {
                e.stopPropagation(); // Prevent bubbling
                this.triggerCancel();
            };
        }
    }

    /** Handles the save action */
    private async triggerSave(): Promise<void> {
        if (!this.editorInstance || typeof this.editorInstance.getContent !== 'function') {
            console.error("Cannot save: TinyMCE instance not available.");
            // Optionally show a toast error to the user
            return;
        }

        const currentContent = this.editorInstance.getContent() || '';
        // Avoid saving if content hasn't changed? Optional optimization.
        // if (currentContent === this.initialContent) {
        //     console.log("Content unchanged, skipping save.");
        //     this.onSaveSuccess(); // Signal success to container anyway to switch mode
        //     return;
        // }

        this.isSaving = true;
        this.setControlLoading(true); // Show loading on buttons

        try {
            const encodedContent = btoa(unescape(encodeURIComponent(currentContent)));
            console.log(`TextSectionEdit saving content for ${this.sectionId}...`);

            await ContentApi.contentServiceSetContent({
                designId: this.designId,
                sectionId: this.sectionId,
                name: "main", // Or "html"
                contentBytes: encodedContent,
            });

            console.log(`TextSectionEdit save successful for ${this.sectionId}.`);
            this.initialContent = currentContent; // Update initial content baseline after successful save
            this.isSaving = false;
            // Don't hide loading here, the component will be destroyed by the container
            await this.onSaveSuccess(); // Signal container that save succeeded

        } catch (error: any) {
            console.error(`Error saving content for TextSectionEdit ${this.sectionId}:`, error);
            this.toastManager.showToast("Save Error", `Failed to save: ${error.message || 'Server error'}`, "error");
            this.isSaving = false;
            this.setControlLoading(false); // Restore buttons on error
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
       setTimeout(() => { this.editorInstance?.focus(); }, 50);
    }

    /** Cleans up the editor instance */
    destroy(): void {
        console.log(`Destroying TextSectionEdit for ${this.sectionId}`);
        if (this.editorInstance) {
            try { tinymce.remove(this.editorInstance); } catch (e) { console.error(e); }
            this.editorInstance = null;
        }
        // Remove listeners
        if (this.saveButton) this.saveButton.onclick = null;
        if (this.cancelButton) this.cancelButton.onclick = null;
        // if (this.validateButton) this.validateButton.onclick = null;
        this.element.innerHTML = ''; // Clear element
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

    /** Gets the current content from the editor */
    public getContent(): SectionContent {
        if (this.editorInstance && typeof this.editorInstance.getContent === 'function') {
            try {
                return this.editorInstance.getContent() || '';
            } catch (error) {
                 console.error(`Error getting content from TinyMCE:`, error);
            }
        }
        return this.initialContent; // Fallback
    }

     // Handle theme change if the editor is active
     public handleThemeChange(): void {
         if (this.editorInstance && this.editorTargetElement) {
             console.log("TextSectionEdit handling theme change...");
             const currentContent = this.editorInstance.getContent();
             // Re-initialize
             this.destroy(); // Clean up old one first
             this.initialContent = currentContent; // Update content baseline
             this.initializeEditor(this.editorTargetElement); // Re-initialize
             this.bindControlEvents(); // Re-bind controls
         }
     }
}
