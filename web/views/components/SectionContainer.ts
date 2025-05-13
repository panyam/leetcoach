import { Modal } from './Modal';
import {
    SectionData, SectionType, SectionContent, SectionCallbacks,
    ISectionModeComponent, ISectionViewComponent, ISectionEditComponent, // Import interfaces
    SaveSuccessCallback, CancelCallback // Import callbacks
} from './types';
import { V1Section } from './apiclient'; // Import API Section type
import { ContentApi } from './Api'; // Need this if Container handles saving
import { TemplateLoader } from './TemplateLoader'; // Import the converter
import { ToastManager } from './ToastManager'; // Import the converter
import { LlmInteractionHandler } from './LlmInteractionHandler'; // Import the new handler
import { FullscreenHandler } from './FullscreenHandler'; // Import the Fullscreen handler
// --- Placeholder Imports for Mode Components (Will be replaced in Phase 3) ---
// These are just placeholders for now to illustrate the structure.
// We will create the actual TextSectionView/Edit in Phase 2.
import { TextSectionView } from './TextSectionView';
import { TextSectionEdit } from './TextSectionEdit';
import { DrawingSectionView } from './DrawingSectionView';
import { DrawingSectionEdit } from './DrawingSectionEdit';
import { SystemDescriptionSectionView } from './SystemDescriptionSectionView';
import { SystemDescriptionSectionEdit } from './SystemDescriptionSectionEdit';
import { PlotSectionView, PlotSectionEdit } from './PlotSectionModes';
// --- End Placeholder Imports ---

/** Treat SectionContainer as the Section Container now
 */
export class SectionContainer {
    protected data: SectionData;
    protected element: HTMLElement;
    protected callbacks: SectionCallbacks;
    protected mode: 'view' | 'edit' = 'view';
    protected isLoading: boolean = false; // Keep track of general loading state

    // Common DOM elements
    protected sectionHeaderElement: HTMLElement | null;
    protected contentContainer: HTMLElement | null; // Container for MODE components
    protected titleElement: HTMLElement | null;
    protected typeIconElement: HTMLElement | null;
    protected deleteButton: HTMLElement | null;
    protected moveUpButton: HTMLElement | null;
    protected moveDownButton: HTMLElement | null;
    protected settingsButton: HTMLElement | null;
    protected addBeforeButton: HTMLElement | null;
    protected addAfterButton: HTMLElement | null;
    protected fullscreenButton: HTMLElement | null; // Moved from subclasses
    protected exitFullscreenButton: HTMLElement | null; // Found after view load

    protected modal: Modal;
    protected toastManager: ToastManager; // Add toast manager
    protected templateLoader = new TemplateLoader()
    protected llmInteractionHandler: LlmInteractionHandler; // Add handler instance
    protected fullscreenHandler: FullscreenHandler | null = null; // Instance for fullscreen logic

    // --- NEW: State for Composition ---
    /** Holds the instance of the currently active View or Edit component. */
    private activeModeComponent: ISectionModeComponent | null = null;
    // ---------------------------------

    // Helper method to find elements within the section's root element
    protected _findElement<T extends HTMLElement>(selector: string, required: boolean = true): T | null {
        const element = this.element.querySelector<T>(selector);
        if (!element && required) {
            console.warn(`SectionContainer ${this.sectionId}: Required element with selector "${selector}" not found.`);
        }
        return element;
    }

    constructor(data: SectionData, element: HTMLElement, callbacks: SectionCallbacks = {}) {
        this.data = data;
        this.element = element;
        this.callbacks = callbacks;
        this.modal = Modal.getInstance();
        this.toastManager = ToastManager.getInstance();
        this.llmInteractionHandler = new LlmInteractionHandler(this.modal, this.toastManager);

        // Find common frame elements
        this.sectionHeaderElement = this._findElement('.section-header');
        this.contentContainer = this._findElement('.section-content'); // This is where modes render
        this.titleElement = this._findElement('.section-title');
        this.typeIconElement = this._findElement('.section-type-icon');
        // ... find universal buttons ...
        this.deleteButton = this._findElement('.section-delete', false);
        this.moveUpButton = this._findElement('.section-move-up', false);
        this.moveDownButton = this._findElement('.section-move-down', false);
        this.settingsButton = this._findElement('.section-settings', false);
        this.addBeforeButton = this._findElement('.section-add-before', false);
        this.addAfterButton = this._findElement('.section-add-after', false);
        this.fullscreenButton = this._findElement('.section-fullscreen', false);
        this.exitFullscreenButton = this._findElement('.section-exit-fullscreen', false);


        if (!this.contentContainer) {
            console.error(`Section content container not found for section ID: ${this.data.id}. Cannot proceed.`);
            // Maybe throw error or mark section as invalid?
            return;
        }

        // Initial setup
        this.updateDisplayTitle();
        this.updateTypeIcon();
        this._bindUniversalEvents(); // Bind events for universal controls

        // Load initial content and render initial view mode
        // NOTE: loadContent is now responsible for fetching *and* setting this.currentContent
        this.renderModeComponent('view');

        (this.element as any).componentInstance = this; // Keep for potential external access
    }

    public get sectionId(): string { return this.data.id }
    public get designId(): string { return this.data.designId }

    /** Returns the section's ID */
    public getId(): string {
        return this.data.id;
    }

     /** Returns the section's current order */
     public getOrder(): number {
        return this.data.order;
    }

    /** Updates the displayed title in the section header */
    protected updateDisplayTitle(): void {
        if (this.titleElement) {
            this.titleElement.textContent = this.data.title;
        }
    }

    /** Updates the type icon in the section header */
    protected updateTypeIcon(): void {
        if (this.typeIconElement) {
            this.typeIconElement.innerHTML = this.getSectionIconSvg();
            this.typeIconElement.setAttribute('title', this.getSectionTypeTitle())
        }
    }

    /** Updates the displayed section number */
    public updateNumber(number: number): void {
        this.data.order = number;
        const numberElement = this.element.querySelector('.section-number');
        if (numberElement) {
            numberElement.textContent = `${number}.`;
        }
    }

    /** Returns the (type) title for this section. */
    protected getSectionTypeTitle(): string {
      return "Section"
    }


    /** Returns the svg content to show for this section. */
    protected getSectionIconSvg(): string {
      return ""
    }

    /** Opens the section settings modal (Placeholder) */
    protected openSettings(): void {
        console.log('Section settings clicked for', this.data.id);
        alert(`Settings for section: ${this.data.title} (ID: ${this.data.id})`);
    }

    /**
     * Returns the callback function to apply LLM results to this specific section.
     * Base implementation returns undefined. Subclasses override this.
     */
    protected getApplyCallback(): ((generatedText: string) => void) | undefined {
        return undefined; // Base sections don't know how to apply text
    }

    /** Returns a copy of the section's data */
    public getData(): SectionData {
        // Return a copy to prevent direct modification
        return JSON.parse(JSON.stringify(this.data));
    }

    /** Enables the fullscreen button functionality for this section */
    public enableFullscreen(): void {
         if (this.fullscreenButton && this.exitFullscreenButton && this.contentContainer) {
            this.fullscreenButton.classList.remove('hidden'); // Make the button visible

            // Instantiate the handler if not already done
            if (!this.fullscreenHandler) {
                 this.fullscreenHandler = new FullscreenHandler(
                     { // Pass required elements
                         sectionElement: this.element,
                         contentContainer: this.contentContainer,
                         fullscreenButton: this.fullscreenButton,
                         exitFullscreenButton: this.exitFullscreenButton,
                         moveUpButton: this.moveUpButton,
                         moveDownButton: this.moveDownButton,
                         addBeforeButton: this.addBeforeButton,
                         addAfterButton: this.addAfterButton,
                         settingsButton: this.settingsButton,
                         deleteButton: this.deleteButton,
                     },
                     this.resizeContentForFullscreen.bind(this) // Pass the resize callback
                 );
            }
             console.log(`Fullscreen enabled for section ${this.data.id}`);
         } else {
             console.warn(`Attempted to enable fullscreen, but required elements (fullscreen/exit button or content container) not found for section ${this.data.id}`);
         }
    }

    /** Removes the section's root element from the DOM */
    public removeElement(): void {
      this.element.remove();
    }

    public destroy(): void {
        this.activeModeComponent?.destroy(); // Destroy active mode component
        this.activeModeComponent = null;
        this.fullscreenHandler?.destroy();
        // Remove container's own listeners if any were added directly
    }

    /** Binds events for universal controls handled by the container itself. */
    protected _bindUniversalEvents(): void {
        // Title editing
        if (this.titleElement) {
            this.titleElement.removeEventListener('click', this.startTitleEdit);
            this.titleElement.addEventListener('click', this.startTitleEdit.bind(this));
        }

        // Use event delegation for UNIVERSAL buttons within the main element
        this.element.onclick = (event: MouseEvent) => {
            const target = event.target as HTMLElement;

            // Check for UNIVERSAL action buttons in the header/frame
            const universalActionButton = target.closest<HTMLButtonElement>(
                '.section-delete, .section-move-up, .section-move-down, .section-settings, .section-ai, ' +
                '.section-add-before, .section-add-after, .section-edit-trigger'
                // Note: Save/Cancel are NOT handled here - they belong to the Edit component
                // Note: Validate/Generate etc are NOT handled here - they belong to the Mode component
            );

            if (universalActionButton) {
                console.log(`Universal Action button clicked:`, universalActionButton.classList);
                event.stopPropagation(); // Prevent triggering general view click

                if (universalActionButton.matches('.section-delete')) { this.callbacks.onDelete?.(this.data.id); }
                else if (universalActionButton.matches('.section-move-up')) { this.callbacks.onMoveUp?.(this.data.id); }
                else if (universalActionButton.matches('.section-move-down')) { this.callbacks.onMoveDown?.(this.data.id); }
                else if (universalActionButton.matches('.section-settings')) { this.openSettings(); }
                else if (universalActionButton.matches('.section-ai')) { this.llmInteractionHandler.showLlmDialog(this.data, this.getApplyCallback()); }
                else if (universalActionButton.matches('.section-add-before')) { this.callbacks.onAddSectionRequest?.(this.data.id, 'before'); }
                else if (universalActionButton.matches('.section-add-after')) { this.callbacks.onAddSectionRequest?.(this.data.id, 'after'); }
                else if (universalActionButton.matches('.section-edit-trigger') && this.mode === 'view') {
                    // This button explicitly triggers the switch
                    this.switchToEditMode();
                }
                return; // Universal action handled
            }

            // Check for general view click to trigger edit (if applicable)
            // This logic remains crucial for sections where clicking content edits (like Text)
            // We might make this behavior configurable per section type later.
            if (this.mode === 'view' /* && this.data.type === 'text' */) { // Example: Only allow for text
                const clickedOnInteractive = target.closest('button, a, input, textarea, select');
                const contentContainer = target.closest('.section-content'); // Check click is within the content area
                const clickedSpecificAction = target.closest('.view-actions button'); // Check if it's a view-mode specific button

                if (contentContainer && !clickedOnInteractive && !clickedSpecificAction) {
                    console.log("General view content click detected, switching to edit.");
                    this.switchToEditMode();
                }
            }
        };
    }


    // --- Refactored Mode Switching Logic ---

    /** Switches the displayed component to the specified mode. */
    private renderModeComponent(targetMode: 'view' | 'edit'): void {
        if (!this.contentContainer) return; // Guard

        // 1. Destroy existing component (if any)
        console.log(`Destroying previous mode component for ${this.sectionId} (if any)`);
        this.activeModeComponent?.destroy();
        this.activeModeComponent = null;
        this.contentContainer.innerHTML = ''; // Clear container

        // 2. Determine which component class to use
        let ComponentClass: any = null;
        let editCallbacks: { onSaveSuccess: SaveSuccessCallback, onCancel: CancelCallback } | null = null;
        const sectionType = this.data.type;

        // Define callbacks *once* outside the switch
        if (targetMode === 'edit') {
            editCallbacks = {
                onSaveSuccess: this.handleSaveSuccess.bind(this), // Renamed handler
                onCancel: this.handleModeCancel.bind(this),
            };
        }

        // Select Component Class
        if (targetMode === 'view') {
            switch (sectionType) {
                case 'text': ComponentClass = TextSectionView; break;
                case 'drawing': ComponentClass = DrawingSectionView; break;
                case 'plot': ComponentClass = PlotSectionView; break; // Placeholder
                case 'system_description': ComponentClass = SystemDescriptionSectionView; break; // *** USE ACTUAL ***
                default: console.error(`Unsupported view: ${sectionType}`);
            }
        } else { // edit mode
            switch (sectionType) {
                case 'text': ComponentClass = TextSectionEdit; break;
                case 'drawing': ComponentClass = DrawingSectionEdit; break;
                case 'plot': ComponentClass = PlotSectionEdit; break; // Placeholder
                case 'system_description': ComponentClass = SystemDescriptionSectionEdit; break; // *** USE ACTUAL ***
                default: console.error(`Unsupported edit: ${sectionType}`);
            }
        }

        if (!ComponentClass) {
             console.error(`No matching mode component found for type: ${sectionType}, mode: ${targetMode}`);
             this.renderErrorState(`Unsupported component: ${sectionType}/${targetMode}`);
             this.isLoading = false;
             return;
        }

        // 3. Create container element for the mode component (remains same)
        const modeElement = document.createElement('div');
        modeElement.className = `section-mode-content section-${sectionType}-${targetMode}-mode`;
        modeElement.style.height = '100%';
        // Append immediately so component constructor can use it
        this.contentContainer.appendChild(modeElement);

        // 4. Instantiate the new component
        // The component constructor is now expected to handle its own data fetching.
        try {
            console.log(`Instantiating ${targetMode} component for ${sectionType} section ${this.sectionId}`);
            if (targetMode === 'edit' && editCallbacks) {
                 // Edit components now expect element, designId, sectionId, and callbacks
                 this.activeModeComponent = new ComponentClass(
                     modeElement,
                     this.designId, // Pass ID
                     this.sectionId, // Pass ID
                     editCallbacks    // Pass save/cancel handlers
                 );
            } else {
                 // View components expect element, designId, sectionId
                 this.activeModeComponent = new ComponentClass(
                     modeElement,
                     this.designId, // Pass ID
                     this.sectionId // Pass ID
                 );
            }
            if (sectionType != "text") this.enableFullscreen();

            this.mode = targetMode;
             // Optional focus for edit mode
             if (targetMode === 'edit' && typeof (this.activeModeComponent as ISectionEditComponent).focus === 'function') {
                 setTimeout(() => (this.activeModeComponent as ISectionEditComponent).focus?.(), 50);
             }
             // Note: Loading state (isLoading) should ideally be cleared by the mode component
             // itself once it finishes loading its data. If not, clear it here after a delay.
             // For simplicity now, let's clear it here.
             this.isLoading = false;
             // contentContainer should be populated by the component now, no need to clear loading state here.

        } catch (error) {
             console.error(`Error instantiating ${targetMode} component for ${this.sectionId}:`, error);
             this.renderErrorState(`Error rendering ${targetMode} mode.`);
             this.activeModeComponent = null;
             this.isLoading = false;
        }
    }

    /** Switches display to Edit mode */
    public switchToEditMode(): void {
        if (this.mode === 'edit' || this.isLoading) return;
        console.log(`Requesting switch to edit mode for ${this.sectionId}`);
        this.renderModeComponent('edit');
        if (this.data.editInFSMode) {
            setTimeout(() => this.fullscreenHandler?.enter(), 50)
        }
    }

    /** Switches display to View mode */
    public switchToViewMode(): void {
        // Note: Saving is handled via handleModeSave callback *before* this is typically called
        if (this.mode === 'view' || this.isLoading) return;
        console.log(`Requesting switch to view mode for ${this.sectionId}`);
        // If switching from edit due to Cancel, activeModeComponent.destroy() will be called here
        this.renderModeComponent('view');
    }

    /** Handles the save action triggered by the Edit component */
    private async handleSaveSuccess(/* newContent: SectionContent*/): Promise<void> {
        console.log(`Container received save SUCCESS signal for ${this.sectionId}`);
        // The Edit component already saved. We just need to switch mode.
        this.renderModeComponent('view');
        // We could potentially trigger a state update higher up if needed (e.g., update last saved time)
        // this.updateLastSavedTimestamp(); // Assume SectionContainer handles this maybe?
    }

    /** Handles the cancel action triggered by the Edit component */
    private handleModeCancel(): void {
        console.log(`Received cancel request from edit component for ${this.sectionId}`);
        // this.switchToViewMode(); // Just switch back, discarding changes implicitly
        this.renderModeComponent('view'); // Discard changes by simply rendering view again
    }

    /** Renders a loading state within the content container */
    protected renderLoadingState(): void {
        if (this.contentContainer) {
             this.contentContainer.innerHTML = `<div class="p-4 text-center text-gray-500 dark:text-gray-400 italic">Loading...</div>`;
        }
    }

    /** Renders an error state within the content container */
    protected renderErrorState(errorMessage: string): void {
         if (this.contentContainer) {
              this.contentContainer.innerHTML = `<div class="p-4 text-red-500 dark:text-red-400 text-center">Error: ${errorMessage}</div>`;
         }
    }


    /** Handles resizing the section's specific content (e.g., canvas, plot) when entering/exiting fullscreen or on window resize. */
    // resizeContentForFullscreen needs to delegate to the active component
    protected resizeContentForFullscreen(isEntering: boolean): void {
        console.log(`Container ${this.sectionId}: Resizing trigger. Fullscreen: ${isEntering}`);
        if (typeof (this.activeModeComponent as any)?.resizeContentForFullscreen === 'function') {
            (this.activeModeComponent as any).resizeContentForFullscreen(isEntering);
        }
    }

    /**
     * Called when the application theme (light/dark) changes.
     * Subclasses can override this method to react to the theme change,
     * e.g., by re-rendering components or adjusting styles.
     * The default implementation does nothing.
     */
    public handleThemeChange(): void {
        console.log(`Theme change notified in container ${this.sectionId}`);
        if (typeof (this.activeModeComponent as any)?.handleThemeChange === 'function') {
            (this.activeModeComponent as any).handleThemeChange();
        }
    }

    /** Handles the logic for editing the section title */
    protected startTitleEdit(e: Event): void {
        e.preventDefault();
        e.stopPropagation();

        if (!this.titleElement) return;

        const currentTitle = this.data.title; // Store original title for cancel
        const input = document.createElement('input');
        input.type = 'text';
        // Apply similar styling as the title for consistency, adjust as needed
        input.className = 'text-lg font-medium border border-gray-300 dark:border-gray-600 dark:bg-gray-700 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 dark:text-white px-1 py-0.5';
        input.value = currentTitle;
        input.style.minWidth = '200px'; // Prevent input from being too small

        // Temporarily replace title span with input
        const titleSpan = this.titleElement; // Assuming titleElement is the span
        const parent = titleSpan.parentNode;
        if (!parent) return;

        parent.replaceChild(input, titleSpan);
        input.focus();
        input.select();

        const cleanup = () => {
            input.removeEventListener('blur', handleSave);
            input.removeEventListener('keydown', handleKeyDown);
            // Ensure the original title span is back if the input is removed
            if (!input.parentNode) { // If already removed by blur save/cancel
               return;
            }
             parent.replaceChild(titleSpan, input);
             titleSpan.textContent = this.data.title; // Ensure display is up-to-date
        };

        const handleSave = () => {
            const newTitle = input.value.trim();
            if (newTitle && newTitle !== this.data.title) {
                this.data.title = newTitle;
                this.callbacks.onTitleChange?.(this.data.id, newTitle);
            }
            // Restore title display (even if unchanged, removes input)
            parent.replaceChild(titleSpan, input);
            titleSpan.textContent = this.data.title; // Update displayed text
            cleanup();
        };

        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                input.blur(); // Trigger save
            } else if (e.key === 'Escape') {
                e.preventDefault();
                // Restore original title visually and remove input
                parent.replaceChild(titleSpan, input);
                titleSpan.textContent = currentTitle; // Revert display
                this.data.title = currentTitle; // Revert data if needed (though not saved yet)
                cleanup();
            }
        };

        input.addEventListener('blur', handleSave, { once: true }); // Auto-save on blur
        input.addEventListener('keydown', handleKeyDown);
    }
}
