import { SectionContent, ISectionViewComponent } from './types';
import { ContentApi } from './Api'; // Import API

export class TextSectionView implements ISectionViewComponent {
    public element: HTMLElement; // The root element (div) managed by this component
    private designId: string;
    private sectionId: string;
    private isLoading: boolean = true;
    private content: string = ''; // Initialize as empty

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

        console.log(`TextSectionView created for ${designId}/${sectionId}`);
        this.renderLoading();
        this.loadDataAndRender(); // Load data asynchronously
    }

    /** Fetches content from the API and then renders */
    private async loadDataAndRender(): Promise<void> {
        try {
            const resp = await ContentApi.contentServiceGetContent({
                designId: this.designId,
                sectionId: this.sectionId,
                name: "main", // Assuming text content is 'main'
            });
            this.content = resp.contentBytes ? atob(resp.contentBytes) : '';
            console.log(`TextSectionView loaded content for ${this.sectionId}, length: ${this.content.length}`);
            this.isLoading = false;
            this.renderContent(); // Render the actual content

        } catch (error: any) {
             console.error(`Error loading content for TextSectionView ${this.sectionId}:`, error);
             this.isLoading = false;
             if (error.response.status === 404) {
                 // Content doesn't exist (new section), render placeholder
                 this.content = '';
                 this.renderContent();
             } else {
                 // Other fetch error
                 this.renderError(`Failed to load content: ${error.message || 'Unknown error'}`);
             }
        }
    }

    /** Renders the current content (or placeholder) into the element */
    private renderContent(): void {
        const displayHtml = this.content.trim().length > 0
            ? this.content
            : '<p class="text-gray-400 dark:text-gray-500 italic p-4 text-center">Click to add content...</p>'; // Placeholder for empty

        // Add styling classes if desired
        this.element.className = 'prose dark:prose-invert max-w-none p-1'; // Basic styling
        this.element.innerHTML = displayHtml;
    }

    /** Renders a loading state */
    private renderLoading(): void {
        this.element.className = 'p-4 text-center text-gray-500 dark:text-gray-400 italic'; // Reset classes
        this.element.innerHTML = `Loading content...`;
    }

    /** Renders an error state */
    private renderError(message: string): void {
        this.element.className = 'p-4 text-red-500 dark:text-red-400 text-center'; // Reset classes
        this.element.innerHTML = `Error: ${message}`;
    }

    /** Cleans up the component */
    destroy(): void {
        console.log(`TextSectionView destroyed for ${this.sectionId}`);
        // No specific listeners or instances to clean up for basic view
        this.element.innerHTML = '';
    }
}
