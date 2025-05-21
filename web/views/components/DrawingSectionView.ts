
// FILE: web/views/components/DrawingSectionView.ts
import { SectionContent, ISectionViewComponent, DrawingContent, ExcalidrawSceneData } from './types';
import { ContentApi } from './Api'; // Import API
import { TemplateLoader } from './TemplateLoader'; // To load the template structure

export class DrawingSectionView implements ISectionViewComponent {
    public element: HTMLElement;
    private designId: string;
    private sectionId: string;
    private isLoading: boolean = true;

    // Store fetched content and previews
    private drawingContent: DrawingContent | null = null; // Main JSON data (optional to load in view)
    private lightPreviewSvg: string | null = null;
    private darkPreviewSvg: string | null = null;

    // DOM element references within the template
    private previewContainer: HTMLDivElement | null = null;
    private editTriggerButton: HTMLButtonElement | null = null; // For consistency, though handled by SectionContainer

    private templateLoader = new TemplateLoader();

    constructor(element: HTMLElement, designId: string, sectionId: string) {
        this.element = element;
        this.designId = designId;
        this.sectionId = sectionId;

        console.log(`DrawingSectionView created for ${designId}/${sectionId}`);
        this.loadTemplateAndData();
    }

    // Viewer components are ok to exit fullscreen just like that
    public shouldExitFullscreen(): boolean {
      return true;
    }

    private async loadTemplateAndData(): Promise<void> {
        this.isLoading = true;
        // Load the base template structure first
        const success = this.templateLoader.loadInto('drawing-section-view', this.element);
        if (!success) {
            this.renderError("Failed to load view template structure.");
            this.isLoading = false;
            return;
        }

        // Find the container within the loaded template
        this.previewContainer = this.element.querySelector<HTMLDivElement>('.drawing-preview-container');
        this.editTriggerButton = this.element.querySelector<HTMLButtonElement>('.section-edit-trigger'); // May not be needed if Base handles it

        if (!this.previewContainer) {
             this.renderError("Preview container missing in template.");
             this.isLoading = false;
             return;
        }

        this.renderLoading(); // Show loading inside preview container

        // Fetch SVG previews (prioritize these for display)
        try {
            const [lightResp, darkResp] = await Promise.allSettled([
                ContentApi.contentServiceGetContent({ designId: this.designId, sectionId: this.sectionId, name: "light.svg" }),
                ContentApi.contentServiceGetContent({ designId: this.designId, sectionId: this.sectionId, name: "dark.svg" })
            ]);

            if (lightResp.status === 'fulfilled' && lightResp.value.contentBytes) {
                this.lightPreviewSvg = atob(lightResp.value.contentBytes);
            } else if (lightResp.status === 'rejected' && lightResp.reason.status !== 404) {
                console.error("Error loading light SVG:", lightResp.reason);
            }

            if (darkResp.status === 'fulfilled' && darkResp.value.contentBytes) {
                this.darkPreviewSvg = atob(darkResp.value.contentBytes);
            } else if (darkResp.status === 'rejected' && darkResp.reason.status !== 404) {
                 console.error("Error loading dark SVG:", darkResp.reason);
            }

             // Optionally load main JSON data if needed for fallback display
             // const mainResp = await ContentApi.contentServiceGetContent({ designId: this.designId, sectionId: this.sectionId, name: "main" });
             // if (mainResp.contentBytes) this.drawingContent = JSON.parse(atob(mainResp.contentBytes));

            this.isLoading = false;
            this.renderContent(); // Render the appropriate preview

            // Add listener for theme changes to potentially swap SVG
            // Using a more specific listener if possible, or rely on container delegation
            document.documentElement.addEventListener('theme-changed', this.handleThemeChange);


        } catch (error: any) {
            // This catch might not be reached if Promise.allSettled is used,
            // handle errors based on settled results instead.
             console.error(`Error loading drawing previews for ${this.sectionId}:`, error);
             this.isLoading = false;
             this.renderError(`Failed to load previews: ${error.message || 'Unknown error'}`);
        }
    }


    /** Renders the appropriate SVG preview or a fallback */
    private renderContent(): void {
        if (!this.previewContainer) return;

        const isDarkMode = document.documentElement.classList.contains('dark');
        const svgToDisplay = isDarkMode ? this.darkPreviewSvg : this.lightPreviewSvg;

        if (svgToDisplay) {
            // Directly inject SVG string. Ensure backend provides sanitized SVG.
            this.previewContainer.innerHTML = svgToDisplay;
        } else {
             // Fallback if SVGs are missing
             // Could try rendering from this.drawingContent if it was loaded,
             // but that requires Excalidraw's export logic here, which is complex for view-only.
             this.previewContainer.innerHTML = `<p class="text-gray-500 dark:text-gray-400 italic p-4 text-center">Drawing preview not available. Click 'Edit' to view/create.</p>`;
        }
    }

    private renderLoading(): void {
         if (this.previewContainer) {
            this.previewContainer.innerHTML = `<p class="text-gray-500 dark:text-gray-400 italic p-4 text-center">Loading preview...</p>`;
         }
    }

    private renderError(message: string): void {
         if (this.previewContainer) {
             this.previewContainer.innerHTML = `<p class="text-red-500 dark:text-red-400 p-4 text-center">Error: ${message}</p>`;
         } else {
             // Fallback to main element if preview container wasn't found
             this.element.innerHTML = `<p class="text-red-500 dark:text-red-400 p-4">${message}</p>`;
         }
    }

    // Method to handle theme change event
    private handleThemeChange = () => { // Use arrow function for correct 'this'
        console.log(`DrawingSectionView ${this.sectionId} detected theme change.`);
        // Re-render content to potentially swap SVG
        if (!this.isLoading) {
             this.renderContent();
        }
    }

    destroy(): void {
        console.log(`DrawingSectionView destroyed for ${this.sectionId}`);
        // Remove theme change listener
        document.documentElement.removeEventListener('theme-changed', this.handleThemeChange);
        this.element.innerHTML = '';
    }
}
