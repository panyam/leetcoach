import { SectionContent, ISectionEditComponent, SaveSuccessCallback, CancelCallback, DrawingContent, ExcalidrawSceneData } from './types';
import { TemplateLoader } from './TemplateLoader';
import { ContentApi } from './Api'; // Import API
import { ToastManager } from './ToastManager'; // For feedback

// Excalidraw integration
import React from 'react';
import ReactDOM from 'react-dom/client';
import { ExcalidrawWrapper } from './ExcalidrawWrapper';
import { convertToExcalidrawElements, exportToSvg } from "@excalidraw/excalidraw"; // Need export

// Excalidraw API type
import type { ExcalidrawImperativeAPI } from '@excalidraw/excalidraw/types/types';

export class DrawingSectionEdit implements ISectionEditComponent {
    public element: HTMLElement;
    private designId: string;
    private sectionId: string;
    private onSaveSuccess: SaveSuccessCallback;
    private onCancel: CancelCallback;

    private templateLoader = new TemplateLoader();
    private toastManager: ToastManager;
    private reactRoot: ReactDOM.Root | null = null;
    private excalidrawAPI: ExcalidrawImperativeAPI | null = null;
    private editorContainerElement: HTMLDivElement | null = null;

    // State
    private initialContent: DrawingContent | null = null; // Loaded from server
    private currentContent: DrawingContent | null = null; // Live state from editor
    private isLoading: boolean = true;
    private isSaving: boolean = false;

    // DOM Elements
    private saveButton: HTMLButtonElement | null = null;
    private cancelButton: HTMLButtonElement | null = null;

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

        console.log(`DrawingSectionEdit created for ${designId}/${sectionId}`);
        this.loadTemplateAndData();
    }

    // Editor components are NOT ok to exit fullscreen just like that
    public shouldExitFullscreen(): boolean {
      return false;
    }

    private renderError(message: string): void {
        this.element.innerHTML = `<div class="p-4 text-red-500 dark:text-red-400">${message}</div>`;
    }

    private async loadTemplateAndData(): Promise<void> {
        this.isLoading = true;
        this.element.innerHTML = `<div class="p-4 text-center text-gray-500 dark:text-gray-400 italic">Loading drawing editor...</div>`;

        // 1. Load Template
        const success = this.templateLoader.loadInto('drawing-section-edit', this.element);
        if (!success) {
            this.renderError("Failed to load editor template structure.");
            this.isLoading = false;
            return;
        }
        this.editorContainerElement = this.element.querySelector<HTMLDivElement>('.drawing-editor-container');
        this.saveButton = this.element.querySelector<HTMLButtonElement>('.section-edit-save');
        this.cancelButton = this.element.querySelector<HTMLButtonElement>('.section-edit-cancel');

        if (!this.editorContainerElement || !this.saveButton || !this.cancelButton) {
            this.renderError("Editor structure incomplete in template.");
            this.isLoading = false;
            return;
        }

        // 2. Fetch Initial Data (main JSON only)
        try {
            const resp = await ContentApi.contentServiceGetContent({
                designId: this.designId,
                sectionId: this.sectionId,
                name: "main",
            });
            if (resp.contentBytes) {
                const jsonStr = atob(resp.contentBytes);
                this.initialContent = JSON.parse(jsonStr) as DrawingContent;
            } else {
                this.initialContent = this.normalizeContent(null); // Use default empty
            }
            this.currentContent = this.initialContent; // Initialize current state
            console.log(`DrawingSectionEdit loaded initial JSON for ${this.sectionId}`);

            // 3. Initialize Editor
            this.initializeEditor(this.editorContainerElement);
            this.bindControlEvents();
            this.isLoading = false;

        } catch (error: any) {
            console.error(`Error loading initial drawing content for ${this.sectionId}:`, error);
            this.isLoading = false;
            if (error.response.status === 404) {
                 this.initialContent = this.normalizeContent(null);
                 this.currentContent = this.initialContent;
                 this.initializeEditor(this.editorContainerElement); // Init with empty
                 this.bindControlEvents();
            } else {
                 this.renderError(`Failed to load drawing data: ${error.message || 'Unknown error'}`);
            }
        }
    }

    private normalizeContent(content: SectionContent): DrawingContent {
        if (content && typeof content === 'object' && 'format' in content && content.format === 'excalidraw/json') {
            return content as DrawingContent;
        }
        // Default empty drawing content
        return { format: 'excalidraw/json', data: { elements: [], appState: {} } };
    }

    private initializeEditor(targetElement: HTMLDivElement): void {
         if (this.reactRoot) {
             console.warn("DrawingSectionEdit: Attempting to initialize editor, but React root already exists. Unmounting first.");
             this.reactRoot.unmount();
             this.reactRoot = null;
             this.excalidrawAPI = null;
         }

        this.reactRoot = ReactDOM.createRoot(targetElement);
        const sceneData = (this.initialContent?.data || { elements: [], appState: {} }) as ExcalidrawSceneData;
        const initialElements = sceneData?.elements || [] // convertToExcalidrawElements([...(sceneData?.elements || [])]); // Use spread to ensure mutable array if needed
        let initialAppState = sceneData?.appState || {};
        initialAppState = { ...initialAppState, collaborators: [] }; // Ensure collaborators is empty array

        const isDarkMode = document.documentElement.classList.contains('dark');

        this.reactRoot.render(
            // Using the wrapper component now
            React.createElement(ExcalidrawWrapper, {
                theme: isDarkMode ? 'dark' : 'light',
                initialData: {
                     elements: initialElements,
                     appState: initialAppState,
                },
                excalidrawAPI: (api: ExcalidrawImperativeAPI) => {
                     this.excalidrawAPI = api;
                     // Optionally center content after API is ready
                     // api?.scrollToContent(initialElements, { fitToContent: true });
                 },
                onChange: this.handleExcalidrawChange.bind(this)
            })
        );
        console.log("Excalidraw editor initialized via wrapper.");
    }

    public resizeContentForFullscreen(isEntering: boolean) {
        const sceneData = (this.currentContent?.data || { elements: [], appState: {} }) as ExcalidrawSceneData;
        const initialElements = sceneData?.elements || [] // convertToExcalidrawElements([...(sceneData?.elements || [])]); // Use spread to ensure mutable array if needed
        this.excalidrawAPI?.scrollToContent(initialElements, { fitToContent: true})
    }

    private handleExcalidrawChange(elements: ReadonlyArray<any>, appState: any /* AppState */, files: any /* BinaryFiles */): void {
        // Update internal state immediately on change
        this.currentContent = { format: 'excalidraw/json', data: { elements, appState } };
        // console.log(`Excalidraw changed. Elements: ${elements.length}`);
        // Debounced auto-save could be triggered here if desired
    }


    private bindControlEvents(): void {
        // Remove potential old listeners
        if (this.saveButton) this.saveButton.onclick = null;
        if (this.cancelButton) this.cancelButton.onclick = null;
        // Bind new ones
        if (this.saveButton) { this.saveButton.onclick = (e) => { e.stopPropagation(); this.triggerSave(); }; }
        if (this.cancelButton) { this.cancelButton.onclick = (e) => { e.stopPropagation(); this.triggerCancel(); }; }
    }
    /** Gets current content, generates SVGs, calls ContentApi multiple times, then calls onSaveSuccess */
    private async triggerSave(): Promise<void> {
        if (this.isSaving || !this.currentContent) {
            console.warn("Save already in progress or no content.");
            return;
        }

        this.isSaving = true;
        this.setControlLoading(true);
        this.toastManager.showToast("Saving...", "Saving drawing and generating previews...", "info");

        const contentToSave = this.currentContent;
        const sceneData = contentToSave.data as ExcalidrawSceneData;

        try {
            // 1. Generate SVGs
            const svgOptions = { exportBackground: false, exportPadding: 10 }; // Consistent options
            const lightSvgPromise = exportToSvg({ elements: sceneData.elements, appState: { ...sceneData.appState, ...svgOptions, exportWithDarkMode: false } } as any);
            const darkSvgPromise = exportToSvg({ elements: sceneData.elements, appState: { ...sceneData.appState, ...svgOptions, exportWithDarkMode: true } } as any);

            const [lightSvgResult, darkSvgResult] = await Promise.allSettled([lightSvgPromise, darkSvgPromise]);

            let lightSvgString = '';
            let darkSvgString = '';
            if (lightSvgResult.status === 'fulfilled') lightSvgString = lightSvgResult.value.outerHTML; else console.error("Light SVG export failed:", lightSvgResult.reason);
            if (darkSvgResult.status === 'fulfilled') darkSvgString = darkSvgResult.value.outerHTML; else console.error("Dark SVG export failed:", darkSvgResult.reason);

            // 2. Prepare content for API calls
            const mainContentJson = JSON.stringify(contentToSave);
            const mainContentBytes = btoa(unescape(encodeURIComponent(mainContentJson)));
            const lightSvgBytes = btoa(unescape(encodeURIComponent(lightSvgString)));
            const darkSvgBytes = btoa(unescape(encodeURIComponent(darkSvgString)));

            // 3. Call APIs concurrently
            const savePromises = [
                ContentApi.contentServiceSetContent({ designId: this.designId, sectionId: this.sectionId, name: "main", contentBytes: mainContentBytes }),
                ContentApi.contentServiceSetContent({ designId: this.designId, sectionId: this.sectionId, name: "light.svg", contentBytes: lightSvgBytes }),
                ContentApi.contentServiceSetContent({ designId: this.designId, sectionId: this.sectionId, name: "dark.svg", contentBytes: darkSvgBytes })
            ];

            const results = await Promise.allSettled(savePromises);

            // Check for failures
            const failures = results.filter(r => r.status === 'rejected');
            if (failures.length > 0) {
                console.error("One or more save operations failed:", failures);
                throw new Error(`Failed to save all drawing parts (${failures.length} failures).`);
            }

            console.log(`DrawingSectionEdit save successful for ${this.sectionId}.`);
            this.initialContent = contentToSave; // Update baseline on successful save
            this.isSaving = false;
            this.toastManager.showToast("Saved", "Drawing saved successfully.", "success", 1500);
            // Don't hide loading here, container will destroy this component
            await this.onSaveSuccess(); // Signal container AFTER all saves are successful

        } catch (error: any) {
             console.error(`Error saving DrawingSectionEdit ${this.sectionId}:`, error);
             this.toastManager.showToast("Save Error", `Failed to save drawing: ${error.message || 'Unknown error'}`, "error");
             this.isSaving = false;
             this.setControlLoading(false); // Restore buttons on error
        }
    }

    private triggerCancel(): void {
        if (this.isSaving) return;
        console.log(`DrawingSectionEdit cancelling for ${this.sectionId}`);
        this.onCancel();
    }

    destroy(): void {
        console.log(`Destroying DrawingSectionEdit for ${this.sectionId}`);
        if (this.reactRoot) {
            try { this.reactRoot.unmount(); } catch (e) { console.error(e); }
            this.reactRoot = null; this.excalidrawAPI = null;
        }
        if (this.saveButton) this.saveButton.onclick = null;
        if (this.cancelButton) this.cancelButton.onclick = null;
        this.element.innerHTML = '';
    }

    public handleThemeChange(): void {
        if (this.editorContainerElement && !this.isLoading) { // Avoid re-render during initial load
            console.log("DrawingSectionEdit handling theme change...");
            // Preserve current content before re-rendering
            const contentBeforeChange = this.currentContent;
            this.initialContent = contentBeforeChange; // Use current as initial for re-render
            this.initializeEditor(this.editorContainerElement); // Re-initializes with new theme
            this.bindControlEvents();
        }
    }

     private setControlLoading(isLoading: boolean): void {
        // ... (implement button loading state similar to TextSectionEdit) ...
         if (this.saveButton) this.saveButton.disabled = isLoading;
         if (this.cancelButton) this.cancelButton.disabled = isLoading;
         // Add visual indicator...
    }

    // focus() - might not be directly applicable
    public focus(): void { this.editorContainerElement?.focus(); }
}
