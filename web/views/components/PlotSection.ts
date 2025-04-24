// components/PlotSection.ts

import { BaseSection } from './BaseSection';
import { PlotContent, SectionData, SectionCallbacks } from './types'; // Or move interfaces

export class PlotSection extends BaseSection {

    // Placeholder for plot library instance or config data
    private plotConfig: object = {};
    private plotCanvas: HTMLCanvasElement | null = null; // If using canvas

    constructor(data: SectionData, element: HTMLElement, callbacks: SectionCallbacks = {}) {
        super(data, element, callbacks);
         // Ensure content is initialized as an object if not present
         if (typeof this.data.content !== 'object' || this.data.content === null) {
            this.data.content = { format: 'placeholder_plot', data: {} };
        }
        this.enableFullscreen();
    }

    /** Decodes raw base64 string, parses JSON into plot config object */
    private decodeContent(rawContent: any | null): object {
       const defaultPlotConfig = {};
        if (typeof rawContent === 'string') {
            try {
                const jsonString = decodeURIComponent(escape(atob(rawContent)));
                const parsed = JSON.parse(jsonString);
                return (parsed && typeof parsed === 'object') ? parsed : defaultPlotConfig;
            } catch (e) {
                console.error(`PlotSection ${this.data.id}: Failed to decode/parse content.`, e);
                return defaultPlotConfig;
            }
        }
        return defaultPlotConfig; // Return default if content is null or not string
    }

    protected populateViewContent(): void {
        const previewContainer = this.contentContainer?.querySelector('.plot-preview-container');
        if (previewContainer) {
             previewContainer.innerHTML = ''; // Clear placeholder/previous plot
             const configData = this.decodeContent(this.data.content);
             this.plotConfig = configData; // Store for potential interactions

             console.log(`Placeholder: Render plot in section ${this.data.id} using config:`, configData);
              // **Placeholder:** Render the plot using content.data and a library (e.g., Chart.js, Plotly)
             // Example:
             // this.plotCanvas = document.createElement('canvas');
             // previewContainer.appendChild(this.plotCanvas);
             // new Chart(this.plotCanvas.getContext('2d'), content.data);
            if (configData && Object.keys(configData).length > 0) {
                 previewContainer.innerHTML = `<pre class="text-xs text-gray-600 dark:text-gray-400">View Placeholder:\n${JSON.stringify(configData, null, 2)}</pre>`;
                 // Example with Canvas:
                 // this.plotCanvas = document.createElement('canvas');
                 // previewContainer.appendChild(this.plotCanvas);
                 // new Chart(this.plotCanvas.getContext('2d'), configData);
            } else {
                 previewContainer.innerHTML = `<p class="text-gray-500 dark:text-gray-400 italic">No plot data. Click 'Edit' to configure.</p>`;
            }
            this.resizeContentForFullscreen(this.isFullscreen); 
        } else {
             console.warn(`View content area not found for plot section ${this.data.id}`);
        }
    }

    protected populateEditContent(): void {
        const editorContainer = this.contentContainer?.querySelector('.plot-editor-container');
        if (editorContainer instanceof HTMLElement) {
            const currentConfig = this.decodeContent(this.data.content);
            this.plotConfig = currentConfig; // Store current config

            // **Placeholder:** Create form fields or a JSON editor
            editorContainer.innerHTML = `
                <label for="plot-json-${this.data.id}" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Plot Configuration (JSON):</label>
                <textarea id="plot-json-${this.data.id}" rows="10" class="plot-config-textarea block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:bg-gray-800 dark:border-gray-600 dark:text-white sm:text-sm font-mono">${JSON.stringify(this.plotConfig, null, 2)}</textarea>
            `;
             const textarea = editorContainer.querySelector('.plot-config-textarea') as HTMLTextAreaElement;
             if (textarea) {
                // Auto-select text on focus for easier editing
                textarea.addEventListener('focus', () => textarea.select());
             }
            console.log(`Placeholder: Initialize plot config editor in section ${this.data.id}`);
        } else {
             console.warn(`Edit content area not found for plot section ${this.data.id}`);
        }
    }

    protected bindViewModeEvents(): void {
         const editTrigger = this.contentContainer?.querySelector('.section-edit-trigger');
        if (editTrigger) {
             editTrigger.removeEventListener('click', this.handleViewClick); // Prevent multiple listeners
             editTrigger.addEventListener('click', this.handleViewClick.bind(this));
        }
         // **Placeholder:** Add listeners for plot interactions if needed
    }

    // Handler function to ensure 'this' context is correct
     private handleViewClick(): void {
        this.switchToEditMode();
    }

    protected bindEditModeEvents(): void {
        const saveButton = this.contentContainer?.querySelector('.section-edit-save');
        const cancelButton = this.contentContainer?.querySelector('.section-edit-cancel');

        if (saveButton) {
            saveButton.addEventListener('click', () => {
                this.switchToViewMode(true); // Save changes
            });
        }
        if (cancelButton) {
            cancelButton.addEventListener('click', () => {
                this.switchToViewMode(false); // Discard changes
            });
        }
        // **Placeholder:** Add listeners for config editor changes if needed
    }

    protected getContentFromEditMode(): PlotContent {
        console.log(`Placeholder: Get data from plot config editor for section ${this.data.id}`);
        let plotData: object = {};
        const textarea = this.contentContainer?.querySelector('.plot-config-textarea') as HTMLTextAreaElement | null;

        if (textarea) {
             try {
                 plotData = JSON.parse(textarea.value);
             } catch (e) {
                 console.error(`Invalid JSON in plot config for section ${this.data.id}:`, e);
                 // Optionally show an error to the user
                 alert("Error parsing plot configuration JSON. Please check the format.");
                 // Return the last known valid config or default
                 plotData = this.plotConfig;
             }
        } else {
             // Fallback if textarea not found (shouldn't happen)
             plotData = this.plotConfig;
        }


        // Return in the expected PlotContent format
        return {
            format: 'placeholder_plot', // Or the actual format used by your plot lib
            data: plotData
        };
    }

    /** Gets the current plot configuration object from the edit mode UI (textarea) */
    protected getInterpretedContentForSaving(): object {
        let plotData: object = {};
        const textarea = this.contentContainer?.querySelector('.plot-config-textarea') as HTMLTextAreaElement | null;

        if (textarea) {
             try {
                 plotData = JSON.parse(textarea.value || '{}'); // Default to empty object if textarea is empty
             } catch (e) {
                 console.error(`Invalid JSON in plot config for section ${this.data.id}:`, e);
                 alert("Error parsing plot configuration JSON. Please check the format.");
                 // Return the last known valid config or default
                 plotData = this.plotConfig || {};
             }
        } else {
             plotData = this.plotConfig || {}; // Fallback
        }
        return plotData;
    }

    /** Encodes the plot config object to base64 */
    protected encodeContentForSaving(interpretedContent: object): { contentBytes: string; contentType: string; format: string; } {
         try {
            const jsonString = JSON.stringify(interpretedContent || {});
            const contentBytes = btoa(unescape(encodeURIComponent(jsonString)));
            return {
                contentBytes: contentBytes,
                contentType: 'application/json',
                // TODO: Determine format based on library later (e.g., chartjs/json)
                format: 'placeholder_plot/json'
            };
         } catch (e) {
              console.error(`PlotSection ${this.data.id}: Failed to stringify/encode plot config.`, e);
             return { contentBytes: btoa(unescape(encodeURIComponent('{}'))), contentType: 'application/json', format: 'placeholder_plot/json' };
         }
    }

    /** No specific cleanup needed for textarea editor */
    protected cleanupEditMode(): void {
         this.plotCanvas = null; // Clear canvas ref if used
         console.log(`Cleanup: PlotSection ${this.data.id} edit mode.`);
    }

    /** Returns the content formatted for the DocumentSection type (PlotContent object) */
    protected getDocumentFormattedContent(): PlotContent {
        let configData: object;
        if (this.mode === 'edit') {
            configData = this.getInterpretedContentForSaving();
        } else {
            configData = this.decodeContent(this.data.content);
        }
        return {
            // TODO: Determine format properly later
            format: 'placeholder_plot/json',
            data: configData
        };
    }

    /** Implement resizing logic for the specific plotting library */
    protected resizeContentForFullscreen(isEntering: boolean): void {
       console.log(`PlotSection ${this.data.id}: Resizing content for fullscreen=${isEntering}.`);
       // Example: if (this.plotInstance) { this.plotInstance.resize(); }
       // Example: if (this.plotContainerElement) { Plotly.Plots.resize(this.plotContainerElement); }
        if (this.plotCanvas && this.mode === 'view') {
             // Re-render plot if using canvas and library needs explicit redraw on resize
             // This assumes populateViewContent sets up the plot instance.
             // const config = this.decodeContent(this.data.content);
             // new Chart(this.plotCanvas.getContext('2d'), config); // Re-create chart
        }
    }

     public override handleThemeChange(): void {
         // Re-render plot in view mode if needed, or update editor theme
         if (this.mode === 'view') {
            this.populateViewContent(); // Re-render plot with current theme
         } else {
            // Update editor theme if applicable (e.g., CodeMirror/Monaco theme)
            console.log(`PlotSection ${this.data.id}: Theme changed in edit mode. Update editor theme if necessary.`);
         }
     }

}
