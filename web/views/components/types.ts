// components/types.ts

export type SectionType = 'text' | 'drawing' | 'plot' | 'system_description';
export type SectionContent = TextContent | DrawingContent | PlotContent | SystemDescriptionContent | null;
export interface DocumentMetadata {
  id: string; // UUID - Placeholder for now
  schemaVersion: string;
  // createdAt: string; // ISO 8601 timestamp - Omitted for simplicity for now
  lastSavedAt: string; // ISO 8601 timestamp
}

export type TextContent = string; // HTML content

// More specific structure for Excalidraw data
export interface ExcalidrawSceneData {
    elements: ReadonlyArray<any>; // Use ExcalidrawElement type if importing Excalidraw types
    appState: any; // Use ExcalidrawAppState type if importing
}


export interface DrawingContent {
  format: "excalidraw/json" | string; // Be specific for Excalidraw
  // Use the specific scene data type or fallback to object/string
  // Use object for Excalidraw as we store { elements, appState }
  data: ExcalidrawSceneData | object | string;
}

export interface PlotContent {
  format: string; // e.g., "chartjs_config", "plotly_json", etc.
  data: object; // The configuration or data object for the plot
}

export type SystemDescriptionContent = string; // DSL content as a string

export interface BaseDocumentSection {
  id: string;
  title: string;
  order: number;
}

export interface TextDocumentSection extends BaseDocumentSection {
  type: 'text';
  content: TextContent | null;
}

export interface DrawingDocumentSection extends BaseDocumentSection {
  type: 'drawing';
  content: DrawingContent | null;
}

export interface PlotDocumentSection extends BaseDocumentSection {
  type: 'plot';
  content: PlotContent | null;
}

export interface SystemDescriptionDocumentSection extends BaseDocumentSection {
  type: 'system_description';
  content: SystemDescriptionContent | null;
}


export interface SectionData {
  id: string;
  designId: string;
  type: SectionType;
  title: string;
  // content?: TextContent | DrawingContent | PlotContent | null;
  order: number;
  getAnswerPrompt: string;
  verifyAnswerPrompt: string;
  editInFSMode?: boolean;
}

export interface SectionCallbacks {
  onDelete?: (sectionId: string) => void;
  onMoveUp?: (sectionId: string) => void;
  onMoveDown?: (sectionId: string) => void;
  onTitleChange?: (sectionId: string, newTitle: string) => void;
  // Ensure content type matches SectionData['content']
  onContentChange?: (sectionId: string, newContent: SectionContent) => void;
  // Callback for requesting section addition
  onAddSectionRequest?: (relativeToId: string, position: 'before' | 'after') => void;
}

// DocumentSection represents the interpreted state, useful for export/display.
export type DocumentSection = TextDocumentSection | DrawingDocumentSection | PlotDocumentSection | SystemDescriptionDocumentSection;

// LeetCoachDocument represents the interpreted state of the whole document.
export interface LeetCoachDocument {
  metadata: DocumentMetadata;
  title: string;
  sections: DocumentSection[];
}

/** Basic interface for any component managing a view/edit mode within a section */
export interface ISectionModeComponent {
    /** The root DOM element managed by this specific mode component. */
    element: HTMLElement;
    /** Cleans up resources like event listeners, editor instances, etc. */
    destroy(): void;

    // Called to chck if the component is ready to be exited out of fullscreen mode
    shouldExitFullscreen(): boolean
}

/** Interface for components displaying section content in view mode. */
export interface ISectionViewComponent extends ISectionModeComponent {
    // Typically, view components might not need many methods beyond destroy.
    // Data is usually passed during instantiation or via an update method if needed later.
    // updateContent?(newContent: SectionContent): void; // Optional update method if needed
}

/** Interface for components handling the editing of section content. */
export interface ISectionEditComponent extends ISectionModeComponent {
    /** Optional: Method to retrieve the current, potentially unsaved, content from the editor/form. */
    getContent?(): SectionContent;
    /** Optional: Method to programmatically focus the primary editing element. */
    focus?(): void;
}

// --- NEW: Callback types for Edit Components ---

/**
 * Callback invoked when the user initiates a save action within an Edit component.
 * The component should pass its current content.
 * The container handles the persistence and decides whether to switch modes.
 * The Promise indicates async operation (like API call).
 */
export type SaveSuccessCallback = () => Promise<void>;

/**
 * Callback invoked when the user initiates a cancel action within an Edit component.
 * The container typically handles switching back to view mode without saving.
 */
export type CancelCallback = () => void;

