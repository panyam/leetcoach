import { SectionContent, ISectionViewComponent } from './types';
import { ContentApi, SystemModelApi } from './Api'; // Import APIs
import { TemplateLoader } from './TemplateLoader';
import { ToastManager } from './ToastManager';
import { V1GenerateDiagramResponse } from './apiclient'; // Import relevant response types if needed later

export class SystemDescriptionSectionView implements ISectionViewComponent {
    public element: HTMLElement; // The root element (div) provided by the container
    private designId: string;
    private sectionId: string;
    private isLoading: boolean = true;
    private dslContent: string = '';

    // DOM Elements within the view template
    private codeBlockElement: HTMLElement | null = null;
    private diagramContainer: HTMLDivElement | null = null;
    private analysisContainer: HTMLDivElement | null = null;
    private validateButton: HTMLButtonElement | null = null;
    private generateDiagramButton: HTMLButtonElement | null = null;
    private runAnalysisButton: HTMLButtonElement | null = null; // Placeholder for now

    // Utilities
    private templateLoader = new TemplateLoader();
    private toastManager: ToastManager;

    // Loading states for actions
    private isValidating: boolean = false;
    private isGeneratingDiagram: boolean = false;
    // private isRunningAnalysis: boolean = false; // Future

    constructor(element: HTMLElement, designId: string, sectionId: string) {
        this.element = element;
        this.designId = designId;
        this.sectionId = sectionId;
        this.toastManager = ToastManager.getInstance();

        console.log(`SystemDescriptionSectionView created for ${designId}/${sectionId}`);
        this.loadTemplateAndData();
    }

    // Viewer components are ok to exit fullscreen just like that
    public shouldExitFullscreen(): boolean {
      return true;
    }

    private async loadTemplateAndData(): Promise<void> {
        this.isLoading = true;
        // 1. Load Template Structure
        const success = this.templateLoader.loadInto('system_description-section-view', this.element);
        if (!success) {
            this.renderError("Failed to load view template structure.");
            this.isLoading = false;
            return;
        }

        // 2. Find internal elements
        this.codeBlockElement = this.element.querySelector<HTMLElement>('code.language-dsl');
        this.diagramContainer = this.element.querySelector<HTMLDivElement>('#generated-diagram-container');
        this.analysisContainer = this.element.querySelector<HTMLDivElement>('#analysis-results-container'); // Find analysis placeholder
        this.validateButton = this.element.querySelector<HTMLButtonElement>('.section-validate-dsl');
        this.generateDiagramButton = this.element.querySelector<HTMLButtonElement>('.section-generate-diagram');
        this.runAnalysisButton = this.element.querySelector<HTMLButtonElement>('.section-run-analysis');

        if (!this.codeBlockElement || !this.diagramContainer || !this.analysisContainer || !this.validateButton || !this.generateDiagramButton /*|| !this.runAnalysisButton*/) {
            this.renderError("View template structure incomplete (missing code, diagram, analysis, or action buttons).");
            this.isLoading = false;
            return;
        }

        this.renderLoading(); // Show loading in main element or specific areas

        // 3. Fetch Initial DSL Data
        try {
            const resp = await ContentApi.contentServiceGetContent({
                designId: this.designId,
                sectionId: this.sectionId,
                name: "main", // Assuming DSL is 'main'
            });
            this.dslContent = resp.contentBytes ? atob(resp.contentBytes) : '';
            this.isLoading = false;
            this.renderContent(); // Render fetched DSL
            this.bindSpecificViewActions(); // Bind buttons now that they exist

        } catch (error: any) {
            console.error(`Error loading DSL content for ${this.sectionId}:`, error);
            this.isLoading = false;
            if (error.response.status === 404) {
                this.dslContent = '';
                this.renderContent(); // Render empty placeholder
                this.bindSpecificViewActions(); // Bind buttons even if content is empty
            } else {
                this.renderError(`Failed to load DSL content: ${error.message || 'Unknown error'}`);
            }
        }
    }

    /** Renders the current DSL content into the code block */
    private renderContent(): void {
        if (!this.codeBlockElement || !this.diagramContainer || !this.analysisContainer) return; // Guard

        const displayContent = this.dslContent.trim().length > 0
            ? this.dslContent
            : '// No DSL content yet. Edit section to add code.';
        this.codeBlockElement.textContent = displayContent;

        // Set initial state for diagram/analysis areas
        this.diagramContainer.innerHTML = `<p class="text-gray-500 dark:text-gray-400 italic p-4 text-center">Click "Generate/Refresh Diagram" to view.</p>`;
        this.analysisContainer.innerHTML = `<p class="text-gray-500 dark:text-gray-400 italic p-4 text-center">Run analysis to see results.</p>`;
        this.analysisContainer.classList.add('hidden'); // Keep analysis hidden initially

        // TODO: Add syntax highlighting here later (Prism.js)
        // if (typeof Prism !== 'undefined') {
        //     Prism.highlightElement(this.codeBlockElement);
        // }
    }

    private renderLoading(): void {
        // Could show loading in code block or entire element
        if (this.codeBlockElement) this.codeBlockElement.textContent = 'Loading DSL...';
        if (this.diagramContainer) this.diagramContainer.innerHTML = ''; // Clear diagram area
        if (this.analysisContainer) this.analysisContainer.innerHTML = ''; // Clear analysis area
    }

    private renderError(message: string): void {
        this.element.innerHTML = `<div class="p-4 text-red-500 dark:text-red-400">${message}</div>`;
    }

    /** Binds handlers to the Validate, Generate Diagram buttons */
    private bindSpecificViewActions(): void {
        // Remove potentially old listeners before adding new ones
        if (this.validateButton) this.validateButton.onclick = null;
        if (this.generateDiagramButton) this.generateDiagramButton.onclick = null;
        // if (this.runAnalysisButton) this.runAnalysisButton.onclick = null;

        if (this.validateButton) {
            this.validateButton.onclick = (e) => { e.stopPropagation(); this.handleValidateClick(this.validateButton); };
        }
        if (this.generateDiagramButton) {
            this.generateDiagramButton.onclick = (e) => { e.stopPropagation(); this.handleGenerateDiagramClick(this.generateDiagramButton); };
        }
        // Bind run analysis later...
    }

    // --- Action Handlers ---

    private async handleValidateClick(buttonElement: HTMLButtonElement | null): Promise<void> {
        if (this.isValidating) return;
        this.isValidating = true;
        this.showLoadingOnButton(buttonElement, true, 'Validating...');

        try {
            const response = await SystemModelApi.systemModelServiceValidateDSL({
                designId: this.designId,
                sectionId: this.sectionId,
                body: {},
            });
             if (response.isValid) {
                this.toastManager.showToast("Validation Success", "DSL syntax is valid.", "success", 2000);
            } else {
                const errorMessages = response.errors?.map(err => `L${err.line}:C${err.column}: ${err.message}`) || ["Unknown validation error."];
                const toastMessage = errorMessages.slice(0, 3).join('\n'); // Limit toast length
                this.toastManager.showToast("Validation Failed", toastMessage, "error", 6000);
                console.error("DSL Validation Errors:", response.errors);
            }
        } catch (error: any) {
            console.error(`Error validating DSL for section ${this.sectionId}:`, error);
            this.toastManager.showToast("Validation Error", `Request failed: ${error.message || 'Server error'}`, "error");
        } finally {
            this.isValidating = false;
            this.showLoadingOnButton(buttonElement, false);
        }
    }

    private async handleGenerateDiagramClick(buttonElement: HTMLButtonElement | null): Promise<void> {
        if (this.isGeneratingDiagram || !this.diagramContainer) return;
        this.isGeneratingDiagram = true;
        this.showLoadingOnButton(buttonElement, true, 'Generating...');
        this.diagramContainer.innerHTML = `<p class="text-gray-500 dark:text-gray-400 italic p-4 text-center">Generating diagram...</p>`; // Show loading in diagram area

        try {
            const response = await SystemModelApi.systemModelServiceGenerateDiagram({
                designId: this.designId,
                sectionId: this.sectionId,
                body: {},
            });

            if (response.diagramContent) {
                 if (response.diagramFormat === 'svg') {
                     // WARNING: Ensure SVG from backend is sanitized if necessary!
                     this.diagramContainer.innerHTML = response.diagramContent;
                     this.toastManager.showToast("Diagram Generated", "Diagram updated.", "success", 1500);
                 } else {
                     // Handle other formats (Mermaid, etc.) later or show raw content
                     this.diagramContainer.innerHTML = `<pre class="text-xs p-2">Format: ${response.diagramFormat}\n\n${response.diagramContent}</pre>`;
                     this.toastManager.showToast("Diagram Generated", `Received ${response.diagramFormat} (preview TBD).`, "info");
                 }
            } else {
                this.diagramContainer.innerHTML = `<p class="text-yellow-500 p-4 text-center">Diagram generation returned empty content.</p>`;
                this.toastManager.showToast("Warning", "Diagram generation was empty.", "warning");
            }
        } catch (error: any) {
             console.error(`Error generating diagram for ${this.sectionId}:`, error);
             this.diagramContainer.innerHTML = `<p class="text-red-500 p-4 text-center">Error generating diagram: ${error.message || 'Server error'}</p>`;
             this.toastManager.showToast("Diagram Error", `Generation failed: ${error.message || 'Server error'}`, "error");
        } finally {
            this.isGeneratingDiagram = false;
            this.showLoadingOnButton(buttonElement, false);
        }
    }


    /** Cleans up the component */
    destroy(): void {
        console.log(`SystemDescriptionSectionView destroyed for ${this.sectionId}`);
        // Remove listeners
        if (this.validateButton) this.validateButton.onclick = null;
        if (this.generateDiagramButton) this.generateDiagramButton.onclick = null;
        // if (this.runAnalysisButton) this.runAnalysisButton.onclick = null;
        this.element.innerHTML = '';
    }

    // Helper for button loading state (copied from previous implementation)
    private showLoadingOnButton(button: HTMLButtonElement | null, isLoading: boolean, loadingText: string = "Processing...") {
       if (!button) return;
        const originalContent = button.dataset.originalContent || button.innerHTML;
        if (!button.dataset.originalContent && !isLoading) {
             button.dataset.originalContent = originalContent;
        } else if (!button.dataset.originalContent && isLoading) {
           button.dataset.originalContent = button.innerHTML;
        }

        if (isLoading) {
             button.disabled = true;
             button.innerHTML = `<svg class="animate-spin -ml-1 mr-2 h-4 w-4 inline-block" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"> <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle> <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path> </svg> ${loadingText}`;
        } else {
             button.disabled = false;
             button.innerHTML = button.dataset.originalContent || 'Action'; // Restore
        }
     }

     // Handle theme change (e.g., if syntax highlighting is added later)
     public handleThemeChange(): void {
         console.log(`SystemDescriptionSectionView ${this.sectionId} received theme change`);
         // Potentially re-apply syntax highlighting if needed
         // if (this.codeBlockElement && typeof Prism !== 'undefined') {
         //     Prism.highlightElement(this.codeBlockElement);
         // }
     }

}
