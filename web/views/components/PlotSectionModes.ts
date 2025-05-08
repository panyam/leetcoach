
import { ISectionViewComponent, ISectionEditComponent, SectionContent, SaveSuccessCallback, CancelCallback } from './types';

export class PlotSectionView implements ISectionViewComponent {
    element: HTMLElement;
    constructor(element: HTMLElement, content: SectionContent) { this.element = element; this.element.innerHTML = `<div class="p-2 italic text-gray-400">[Placeholder] Text View: ${content}</div>`; }
    destroy(): void {}
}

export class PlotSectionEdit implements ISectionEditComponent {
    element: HTMLElement;
    constructor(element: HTMLElement, content: SectionContent, callbacks: { onSave: SaveSuccessCallback, onCancel: CancelCallback }) { this.element = element; this.element.innerHTML = `<div class="p-2 italic text-gray-400">[Placeholder] Text Edit: ${content}</div> <button onclick="callbacks.onSave('saved')">Save</button><button onclick="callbacks.onCancel()">Cancel</button>`; }
    destroy(): void {}
}

