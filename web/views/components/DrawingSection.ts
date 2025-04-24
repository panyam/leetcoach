// components/DrawingSection.ts

import { BaseSection } from './BaseSection';
import { SectionData, SectionCallbacks, DrawingContent, ExcalidrawSceneData } from './types';

// --- Excalidraw Integration ---
import React from 'react';
import ReactDOM from 'react-dom/client'; // Use client for React 18+
import { Excalidraw, convertToExcalidrawElements } from '@excalidraw/excalidraw';
// Optional: Import types if needed for stricter typing
// import { ExcalidrawElement, ExcalidrawImperativeAPI } from '@excalidraw/excalidraw/types/element/types';
// import { AppState } from '@excalidraw/excalidraw/types/types';

// Type alias for Excalidraw's API handle
type ExcalidrawApi = any; // Use ExcalidrawImperativeAPI if types are imported

export class DrawingSection extends BaseSection {

    // Placeholder for drawing library instance
    // Store React root and Excalidraw API instance
    private reactRoot: ReactDOM.Root | null = null;
    private excalidrawAPI: ExcalidrawApi | null = null;
    private currentDrawingData: ExcalidrawSceneData | null = null; // Store current state for saving
    private editorContainerElement: HTMLDivElement | null = null; // Store container reference

    constructor(data: SectionData, element: HTMLElement, callbacks: SectionCallbacks = {}) {
        super(data, element, callbacks);
         // Ensure content is initialized as an object if not present
         if (typeof this.data.content !== 'object' || this.data.content === null) {
            this.data.content = { format: 'excalidraw/json', data: { elements: [], appState: {} } };
        }
        this.enableFullscreen();
    }

    /** Decodes raw base64 string, parses JSON into ExcalidrawSceneData */
    private decodeContent(rawContent: any | null): ExcalidrawSceneData {
        const defaultScene: ExcalidrawSceneData = { elements: [], appState: {} };
        if (typeof rawContent === 'string') {
            try {
                const jsonString = decodeURIComponent(escape(atob(rawContent)));
                const parsed = JSON.parse(jsonString);
                // Basic validation: check if it looks like scene data
                if (parsed && typeof parsed === 'object' && Array.isArray(parsed.elements) && typeof parsed.appState === 'object') {
                     // Ensure elements are converted if necessary (might be redundant if saved correctly)
                     parsed.elements = convertToExcalidrawElements(parsed.elements);
                    return parsed as ExcalidrawSceneData;
                } else {
                    console.warn(`DrawingSection ${this.data.id}: Decoded content is not valid SceneData.`, parsed);
                    return defaultScene;
                }
            } catch (e) {
                console.error(`DrawingSection ${this.data.id}: Failed to decode/parse content.`, e);
                return defaultScene;
            }
        }
        return defaultScene; // Return default if content is null or not string
    }

    protected populateViewContent(): void {
        const previewContainer = this.contentContainer?.querySelector('.drawing-preview-container');
        if (previewContainer) {
            const content = this.data.content as DrawingContent;
            const sceneData = this.decodeContent(this.data.content);
            // TODO: Implement proper SVG export/render for view mode later
             if (sceneData?.elements?.length > 0) {
                 previewContainer.innerHTML = `<pre class="text-xs text-gray-600 dark:text-gray-400">View Placeholder: ${sceneData.elements.length} elements</pre>`; // Simple placeholder
                 // Replace with actual SVG rendering:
                 // const svg = await ExcalidrawLib.exportToSvg({ elements: sceneData.elements, appState: sceneData.appState });
                 // previewContainer.innerHTML = ''; // Clear placeholder
                 // previewContainer.appendChild(svg);
            } else {
                 previewContainer.innerHTML = `<p class="text-gray-500 dark:text-gray-400 italic">No drawing data. Click 'Edit' to start.</p>`;
            }
        } else {
             console.warn(`View content area not found for drawing section ${this.data.id}`);
        }
    }

    protected populateEditContent(): void {
        this.editorContainerElement = this.contentContainer?.querySelector('.drawing-editor-container') as HTMLDivElement;
        if (this.editorContainerElement instanceof HTMLElement) {
            this.editorContainerElement.innerHTML = ''; // Clear placeholder
            console.log(`Initializing Excalidraw in section ${this.data.id}`);
            this.cleanupEditMode(); // Ensure any previous instance is gone

            // Ensure React root doesn't already exist (safety check)
            if (this.reactRoot) {
                console.warn(`DrawingSection ${this.data.id}: Attempting to populate edit content, but React root already exists. Unmounting first.`);
                this.reactRoot.unmount();
                this.reactRoot = null;
                this.excalidrawAPI = null;
            }

            this.reactRoot = ReactDOM.createRoot(this.editorContainerElement);
            const initialSceneData = this.decodeContent(this.data.content); // Decode raw content
            this.currentDrawingData = initialSceneData; // Set initial edit state

            // Determine theme
            const isDarkMode = document.documentElement.classList.contains('dark');

            this.reactRoot.render(
                React.createElement(Excalidraw, {
                    excalidrawAPI: (api: ExcalidrawApi) => { this.excalidrawAPI = api; }, // Store API handle
                    initialData: {
                        elements: initialSceneData.elements,
                        appState: initialSceneData.appState
                    },
                    onChange: (elements, appState) => {
                        // Update internal state used by getInterpretedContentForSaving
                        this.currentDrawingData = { elements, appState };
                    },
                    theme: isDarkMode ? 'dark' : 'light',
                })
            );
        } else {
             console.warn(`Edit content area not found for drawing section ${this.data.id}`);
        }
    }

    protected bindViewModeEvents(): void {
        const editTrigger = this.contentContainer?.querySelector('.section-edit-trigger');
        if (editTrigger) {
             editTrigger.removeEventListener('click', this.handleViewClick); // Prevent multiple listeners
             editTrigger.addEventListener('click', this.handleViewClick.bind(this));
        }
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
                // No explicit discard needed, just switch mode without saving
                this.switchToViewMode(false); // Discard changes
            });
        }
    }

    /** Handles changes from Excalidraw - could be used for auto-save or marking dirty */
    private handleExcalidrawChange(elements: ReadonlyArray<any>, appState: any): void {
        // Store the latest data internally. Debounce saving or mark as dirty here.
        this.currentDrawingData = { elements, appState };
        // console.log(`Excalidraw changed in section ${this.data.id}. Elements: ${elements.length}`);
        // Example: Trigger debounced save after inactivity
        // this.debouncedSave();
    }

    protected getContentFromEditMode(): DrawingContent {
        console.log(`Getting data from Excalidraw instance for section ${this.data.id}`);

        let drawingData: ExcalidrawSceneData = { elements: [], appState: {} };

        // Try to get data from the stored state updated by onChange
        if (this.currentDrawingData) {
            drawingData = this.currentDrawingData;
        }
        // Fallback: Try to get directly from API if onChange didn't fire recently (less ideal)
        else if (this.excalidrawAPI) {
            drawingData = {
                 elements: this.excalidrawAPI.getSceneElements(),
                 appState: this.excalidrawAPI.getAppState(),
            };
            console.warn(`Getting Excalidraw data directly from API for ${this.data.id}. Should ideally use onChange state.`);
        } else {
             console.error(`Cannot get Excalidraw data: API instance not found for section ${this.data.id}. Returning empty.`);
             // Return the last known saved data or empty
             const savedContent = this.data.content as DrawingContent;
             if (savedContent?.format === 'excalidraw/json') {
                drawingData = savedContent.data as ExcalidrawSceneData;
             }
        }

        // Return in the expected DrawingContent format
        return {
            format: 'placeholder_drawing', // Or the actual format used by your lib
            data: drawingData
        };
    }
 
    // Override switchToViewMode to handle React unmounting
    public override async switchToViewMode(saveChanges: boolean): Promise<void> {
        if (this.mode === 'edit') {
            console.log(`DrawingSection ${this.data.id}: Unmounting React component on switch to view.`);
            if (this.reactRoot) {
                try {
                    this.reactRoot.unmount();
                } catch (error) {
                    console.error(`Error unmounting React root for section ${this.data.id}:`, error);
                } finally {
                    this.reactRoot = null;
                    this.excalidrawAPI = null;
                    this.editorContainerElement = null; // Clear container ref
                    this.currentDrawingData = null; // Clear temporary state
                }
            }
        }
        // Call the base class method *after* cleanup
        super.switchToViewMode(saveChanges);
    }

    /**
     * Handles theme changes specifically for the DrawingSection.
     * Re-renders the Excalidraw component if it's in edit mode.
     */
    public override handleThemeChange(): void {
         console.log(`DrawingSection ${this.data.id}: Handling theme change.`);
         if (this.mode !== 'edit' || !this.reactRoot || !this.editorContainerElement) {
             // Only need to act if the editor is currently active
             return;
         }

         console.log(`DrawingSection ${this.data.id}: Re-rendering Excalidraw for theme change.`);

         // Get current content before unmounting/re-rendering
         const currentContent = this.getContentFromEditMode();
         this.data.content = currentContent; // Update internal data immediately

         // Unmount the existing instance cleanly
         this.reactRoot.unmount();
         this.reactRoot = null;
         this.excalidrawAPI = null;

         // Re-populate content, which will re-mount Excalidraw with the correct theme
         this.populateEditContent();
     }

    /** Implement the abstract method from BaseSection */
    protected resizeContentForFullscreen(isEntering: boolean): void {
        console.log(`DrawingSection ${this.data.id}: Resizing content for fullscreen=${isEntering}. Excalidraw should resize automatically within container.`);
        // Excalidraw usually adapts to its container size.
        // Example: if (this.drawingEditorInstance && typeof this.drawingEditorInstance.resize === 'function') {
        //     this.drawingEditorInstance.resize(); // Call the library's specific resize/redraw method
        // }
    }
    /** Gets the current Excalidraw scene data from the internal state updated by onChange */
    protected getInterpretedContentForSaving(): ExcalidrawSceneData {
        // Prefer the state updated by onChange
        if (this.currentDrawingData) {
            return this.currentDrawingData;
        }
        // Fallback to API - less ideal as it might miss very last changes
        if (this.excalidrawAPI) {
            console.warn(`DrawingSection ${this.data.id}: Falling back to excalidrawAPI.getSceneElements/getAppState for saving.`);
            return {
                elements: this.excalidrawAPI.getSceneElements() || [],
                appState: this.excalidrawAPI.getAppState() || {},
            };
        }
        console.error(`DrawingSection ${this.data.id}: Cannot get content for saving - no state and no API.`);
        return this.decodeContent(this.data.content); // Absolute fallback
    }

    /** Encodes Excalidraw scene data object to base64 */
    protected encodeContentForSaving(interpretedContent: ExcalidrawSceneData): { contentBytes: string; contentType: string; format: string; } {
        try {
            const jsonString = JSON.stringify(interpretedContent || { elements: [], appState: {} });
            const contentBytes = btoa(unescape(encodeURIComponent(jsonString))); // JSON -> UTF8 -> Base64
            return {
                contentBytes: contentBytes,
                contentType: 'application/json',
                format: 'excalidraw/json' // Be specific
            };
        } catch (e) {
             console.error(`DrawingSection ${this.data.id}: Failed to stringify/encode scene data.`, e);
             // Return empty content to avoid saving corrupted data?
             return { contentBytes: btoa(unescape(encodeURIComponent('{"elements":[],"appState":{}}'))), contentType: 'application/json', format: 'excalidraw/json' };
        }
    }

    /** Unmounts the React component */
    protected cleanupEditMode(): void {
        if (this.reactRoot) {
            console.log(`Cleanup: Unmounting Excalidraw for ${this.data.id}`);
            try {
                this.reactRoot.unmount();
            } catch (error) {
                console.error(`Error unmounting React root for section ${this.data.id}:`, error);
            } finally {
                this.reactRoot = null;
                this.excalidrawAPI = null;
                this.currentDrawingData = null; // Clear edit state
                this.editorContainerElement = null;
            }
        }
    }

    /** Returns the content formatted for the DocumentSection type (DrawingContent object) */
    protected getDocumentFormattedContent(): DrawingContent {
        let sceneData: ExcalidrawSceneData;
        if (this.mode === 'edit') {
            sceneData = this.getInterpretedContentForSaving(); // Get current editor state
        } else {
            sceneData = this.decodeContent(this.data.content); // Decode stored raw content
        }
        return {
            format: 'excalidraw/json',
            data: sceneData
        };
    }
}
