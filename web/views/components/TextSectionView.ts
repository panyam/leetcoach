import { SectionContent, ISectionViewComponent } from './types';
import { ContentApi } from './Api';
import { TemplateLoader } from './TemplateLoader';

// If using NPM for TUI Editor:
import Editor from '@toast-ui/editor'; // Viewer is part of the main Editor class
import '@toast-ui/editor/dist/toastui-editor-viewer.css'; // Import viewer CSS if separate
import '@toast-ui/editor/dist/toastui-editor.css'; // Or main CSS if it includes viewer styles
import codeSyntaxHighlight from '@toast-ui/editor-plugin-code-syntax-highlight'; // For viewer too
import 'prismjs/themes/prism.css';
import Prism from 'prismjs';

export class TextSectionView implements ISectionViewComponent {
    public element: HTMLElement;
    private designId: string;
    private sectionId: string;
    private isLoading: boolean = true;
    private markdownContent: string = '';

    private viewerInstance: any | null = null; // Instance of TUI Editor Viewer
    private viewerContainerElement: HTMLDivElement | null = null; // Div where viewer will mount
    private templateLoader = new TemplateLoader();

    /**
     * Creates an instance of TextSectionView.
     * @param element The container element provided by SectionContainer.
     * @param designId The ID of the parent design.
     * @param sectionId The ID of this section.
     */
    constructor(element: HTMLElement, designId: string, sectionId: string) {
        this.element = element;
        this.designId = designId;
        this.sectionId = sectionId;
        this.loadTemplateAndData();
    }

    /** Fetches content from the API and then renders */
    private async loadTemplateAndData(): Promise<void> {
        this.isLoading = true;
        // Load the template structure for the view mode
        // This template should now just have a single div for the viewer
        const success = this.templateLoader.loadInto('text-section-view', this.element);
        if (!success) {
            this.renderError("Failed to load view template structure.");
            this.isLoading = false;
            return;
        }

        this.viewerContainerElement = this.element.querySelector<HTMLDivElement>('#toast-viewer-container'); // Use new ID
        if (!this.viewerContainerElement) {
            this.renderError("Viewer container element not found in template.");
            this.isLoading = false;
            return;
        }
        this.renderLoading(); // Show loading in the viewer container

        try {
            const resp = await ContentApi.contentServiceGetContent({
                designId: this.designId,
                sectionId: this.sectionId,
                name: "main",
            });
            this.markdownContent = resp.contentBytes ? atob(resp.contentBytes) : ''; // This is raw Markdown
            console.log(`TextSectionView loaded content for ${this.sectionId}, length: ${this.markdownContent.length}`);
            this.isLoading = false;
            this.initializeViewer(); // Initialize viewer with fetched content

        } catch (error: any) {
            console.error(`Error loading Markdown for TextSectionView ${this.sectionId}:`, error);
            this.isLoading = false;
            if (error.response.status === 404) {
                this.markdownContent = '';
                this.initializeViewer();
            } else {
                this.renderError(`Failed to load content: ${error.message || 'Unknown error'}`);
            }
        }
    }

    /** Renders the fetched Markdown as HTML */
    private initializeViewer(): void {
        if (!this.viewerContainerElement) return;

        // Destroy previous instance if any
        if (this.viewerInstance) {
            try { this.viewerInstance.destroy(); } catch(e) {console.error(e);}
            this.viewerInstance = null;
        }
        this.viewerContainerElement.innerHTML = ''; // Clear before mounting

        const isDarkMode = document.documentElement.classList.contains('dark');

        try {
            this.viewerInstance = Editor.factory({
                el: this.viewerContainerElement,
                viewer: true,
                initialValue: this.markdownContent || '<p class="text-gray-400 dark:text-gray-500 italic p-4 text-center">Click to add content...</p>',
                height: 'auto', // Let it auto-adjust height
                theme: isDarkMode ? 'dark' : 'default',
                usageStatistics: false,
                // plugins: [codeSyntaxHighlight] // For NPM
                plugins: [[codeSyntaxHighlight, { highlighter: Prism }]],
            });
            console.log(`Toast UI Viewer initialized for TextSectionView ${this.sectionId}`);
        } catch (error) {
            console.error("Error initializing Toast UI Viewer:", error);
            this.viewerContainerElement.innerHTML = `<p class="text-red-500 p-4">Failed to initialize Markdown viewer.</p><pre>${this.markdownContent}</pre>`;
        }
    }

    /** Renders a loading state */
    private renderLoading(): void {
        if (this.viewerContainerElement) {
            this.viewerContainerElement.innerHTML = `<p class="text-gray-500 dark:text-gray-400 italic p-4 text-center">Loading content...</p>`;
        } else {
            this.element.innerHTML = `<p class="text-gray-500 dark:text-gray-400 italic p-4 text-center">Loading...</p>`;
        }
    }

    /** Renders an error state */
    private renderError(message: string): void {
        if (this.viewerContainerElement) {
            this.viewerContainerElement.innerHTML = `<p class="text-red-500 dark:text-red-400 p-4 text-center">Error: ${message}</p>`;
        } else {
            this.element.innerHTML = `<p class="text-red-500 dark:text-red-400 p-4">${message}</p>`;
        }
    }

    /** Cleans up the component */
    destroy(): void {
        console.log(`TextSectionView (TUI Viewer) destroyed for ${this.sectionId}`);
        if (this.viewerInstance) {
            try { this.viewerInstance.destroy(); } catch (e) { console.error(e); }
            this.viewerInstance = null;
        }
        this.element.innerHTML = '';
    }

    // Handle theme change for the viewer
    public handleThemeChange(): void {
        if (this.viewerInstance && this.viewerContainerElement) {
            console.log("TextSectionView (TUI Viewer) handling theme change...");
            // Re-initialize the viewer with the current content and new theme
            // TUI Viewer doesn't have a direct setTheme method usually, so re-init is common.
            this.initializeViewer();
        }
    }
}
