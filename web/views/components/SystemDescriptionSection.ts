// FILE: web/views/components/SystemDescriptionSection.ts

import { ContentApi } from './Api';
import { BaseSection } from './BaseSection';
import { SectionData, SectionContent, SectionCallbacks, SystemDescriptionContent } from './types';

// Assume Monaco Editor types are available globally or imported appropriately
// If using CDN, declare the 'monaco' global variable:
declare const monaco: any;
// Or import if using npm package:
// import * as monaco from 'monaco-editor';

export class SystemDescriptionSection extends BaseSection {
    // Simple code icon (like a semicolon in braces)
    protected static readonly ICON_SVG = `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-full h-full"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75h-10.5l-1.125 1.125c-.59.59-.188 1.625.563 1.625h12.75c.75 0 1.153-1.035.563-1.625L17.25 6.75zm0 0V4.5m0 2.25V9m-10.5 0V4.5m0 4.5V9m1.125 5.625h8.25M10.875 14.625V18m2.25-3.375V18" /> <path stroke-linecap="round" d="M12 16.5h.01" /> </svg>`;

    private editorInstance: any | null = null; // Monaco Editor instance type
    private dslContent: SystemDescriptionContent = ''; // Store the DSL string

    constructor(data: SectionData, element: HTMLElement, callbacks: SectionCallbacks = {}) {
        super(data, element, callbacks);
        // No special initialization needed here for Phase 1
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
        const editorTarget = this._findElement<HTMLDivElement>('.system-dsl-editor-target');
        if (!editorTarget) {
            console.error(`Edit content target area not found for system_description section ${this.data.id}`);
            return;
        }

        // Basic check if Monaco is available (replace with more robust check if needed)
        if (typeof monaco === 'undefined' || typeof monaco.editor === 'undefined') {
           console.error("Monaco Editor is not available. Make sure it's loaded.");
           editorTarget.innerHTML = `<p class="text-red-500 p-4">Error: Code editor library (Monaco) not loaded.</p>`;
           return;
        }

        // Ensure previous instance is destroyed
        this.cleanupEditMode();

        const isDarkMode = document.documentElement.classList.contains('dark');

        try {
            this.editorInstance = monaco.editor.create(editorTarget, {
                value: this.dslContent,
                language: 'plaintext', // Use 'plaintext' for Phase 1, custom language in Phase 2
                theme: isDarkMode ? 'vs-dark' : 'vs',
                automaticLayout: true, // Adjusts editor size on container resize
                minimap: { enabled: false },
                wordWrap: 'on', // Or 'off' based on preference
                fontSize: 13,
                scrollBeyondLastLine: false,
            });
            this.editorInstance.focus();
            console.log(`Monaco Editor initialized for section ${this.data.id}`);
        } catch(error) {
             console.error(`Error initializing Monaco Editor for section ${this.data.id}:`, error);
             editorTarget.innerHTML = `<p class="text-red-500 p-4">Error initializing code editor.</p>`;
             this.editorInstance = null; // Ensure instance is null on error
        }
    }

    protected getContentFromEditMode(): SystemDescriptionContent {
        if (this.editorInstance && typeof this.editorInstance.getValue === 'function') {
            try {
                return this.editorInstance.getValue() || '';
            } catch (error) {
                 console.error(`Error getting content from Monaco Editor for section ${this.data.id}:`, error);
                 return this.dslContent; // Fallback to stored content
            }
        }
        console.warn(`Monaco Editor instance not available when getting content for section ${this.data.id}.`);
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
        if (this.editorInstance && typeof this.editorInstance.dispose === 'function') {
            console.log(`Disposing Monaco Editor instance for section ${this.data.id}`);
            try {
                this.editorInstance.dispose();
            } catch (error) {
                 console.error(`Error disposing Monaco editor for ${this.data.id}:`, error);
            } finally {
                this.editorInstance = null;
            }
        }
    }

    // Implement abstract methods from BaseSection
    protected resizeContentForFullscreen(isEntering: boolean): void {
        console.log(`SystemDescriptionSection ${this.data.id}: Resizing content for fullscreen=${isEntering}.`);
        // Monaco Editor with automaticLayout should handle most cases.
        // We might force a layout recalculation if needed:
        if (this.editorInstance && typeof this.editorInstance.layout === 'function') {
             // Brief delay might help ensure container dimensions are settled
             setTimeout(() => this.editorInstance.layout(), 50);
        }
    }

    protected override updateInternalContent(newContent: SectionContent): void {
        if (typeof newContent === 'string') {
            this.dslContent = newContent;
             // If editor is active, update it
             if (this.mode === 'edit' && this.editorInstance && typeof this.editorInstance.setValue === 'function') {
                try {
                    this.editorInstance.setValue(this.dslContent);
                } catch(e) { console.error("Error setting Monaco value:", e); }
            }
        } else {
             console.warn(`SystemDescriptionSection ${this.data.id}: Received non-string content during updateInternalContent`, newContent);
        }
    }

     // Handle theme changes for the editor
     public override handleThemeChange(): void {
         console.log(`SystemDescriptionSection ${this.data.id}: Handling theme change.`);
         if (this.mode === 'edit' && this.editorInstance && typeof monaco !== 'undefined') {
            const isDarkMode = document.documentElement.classList.contains('dark');
            try {
                monaco.editor.setTheme(isDarkMode ? 'vs-dark' : 'vs');
                console.log(`Monaco theme updated for ${this.data.id}`);
            } catch(e) { console.error("Error setting Monaco theme:", e); }
         }
     }
}
