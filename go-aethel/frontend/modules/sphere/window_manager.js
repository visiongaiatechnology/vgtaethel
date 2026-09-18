// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/window_manager.js
// Purpose: Unified Window Lifecycle, Dragging, 8-Way Resizing, Snapping, Focus Stacking, and Persistence

import { state } from '../state.js';
import { getAppDefinition } from './app_registry.js';
import { contextBus } from './context_bus.js';

let highestZIndex = 100;
const windowStateMap = new Map();

/**
 * Focus window and bring to top of z-stack
 * @param {HTMLElement} windowEl
 */
export function focusWindow(windowEl) {
    if (!windowEl) return;
    
    document.querySelectorAll('.sphere-window').forEach(win => {
        win.classList.remove('active-focus');
    });

    highestZIndex += 2;
    windowEl.classList.add('active-focus');
    windowEl.style.zIndex = highestZIndex;
    
    state.activeSphereWindow = windowEl.id;
    const appID = windowEl.dataset.appId || windowEl.id.replace('sphere-window-', '');
    
    contextBus.updateContext({
        focusedApp: appID,
        focusedWindowId: windowEl.id
    });
}

/**
 * Make an element draggable by its header
 * @param {HTMLElement} windowEl
 * @param {HTMLElement} handleEl
 */
export function makeDraggable(windowEl, handleEl) {
    if (!windowEl || !handleEl) return;
    let pos1 = 0, pos2 = 0, pos3 = 0, pos4 = 0;

    handleEl.addEventListener('mousedown', dragMouseDown);
    handleEl.addEventListener('touchstart', dragTouchStart, { passive: true });

    function dragMouseDown(e) {
        if (e.target.closest('button, input, textarea, select, .window-controls')) return;
        e.preventDefault();
        focusWindow(windowEl);

        pos3 = e.clientX;
        pos4 = e.clientY;

        document.addEventListener('mouseup', closeDragElement);
        document.addEventListener('mousemove', elementDrag);
    }

    function dragTouchStart(e) {
        if (e.target.closest('button, input, textarea, select, .window-controls')) return;
        focusWindow(windowEl);
        if (e.touches.length === 1) {
            pos3 = e.touches[0].clientX;
            pos4 = e.touches[0].clientY;

            document.addEventListener('touchend', closeDragElement);
            document.addEventListener('touchmove', elementTouchDrag, { passive: false });
        }
    }

    function elementDrag(e) {
        e.preventDefault();
        pos1 = pos3 - e.clientX;
        pos2 = pos4 - e.clientY;
        pos3 = e.clientX;
        pos4 = e.clientY;

        applyNewPosition(windowEl.offsetTop - pos2, windowEl.offsetLeft - pos1);
    }

    function elementTouchDrag(e) {
        if (e.touches.length === 1) {
            e.preventDefault();
            pos1 = pos3 - e.touches[0].clientX;
            pos2 = pos4 - e.touches[0].clientY;
            pos3 = e.touches[0].clientX;
            pos4 = e.touches[0].clientY;

            applyNewPosition(windowEl.offsetTop - pos2, windowEl.offsetLeft - pos1);
        }
    }

    function applyNewPosition(top, left) {
        const desktop = document.querySelector('.sphere-desktop');
        if (!desktop) return;

        const rect = desktop.getBoundingClientRect();
        const winRect = windowEl.getBoundingClientRect();

        const minTop = 0;
        const maxTop = rect.height - 40;
        const minLeft = -winRect.width + 100;
        const maxLeft = rect.width - 100;

        const finalTop = Math.max(minTop, Math.min(top, maxTop));
        const finalLeft = Math.max(minLeft, Math.min(left, maxLeft));

        windowEl.style.top = finalTop + 'px';
        windowEl.style.left = finalLeft + 'px';
    }

    function closeDragElement() {
        document.removeEventListener('mouseup', closeDragElement);
        document.removeEventListener('mousemove', elementDrag);
        document.removeEventListener('touchend', closeDragElement);
        document.removeEventListener('touchmove', elementTouchDrag);
        saveDesktopState();
    }
}

/**
 * Equip window with 8-directional edge resize handles
 * @param {HTMLElement} windowEl
 */
export function makeResizable(windowEl) {
    if (!windowEl || windowEl.dataset.resizableBound === 'true') return;
    windowEl.dataset.resizableBound = 'true';
    const edges = ['n', 'ne', 'e', 'se', 's', 'sw', 'w', 'nw'];

    for (const edge of edges) {
        const handle = document.createElement('div');
        handle.className = `sphere-resize-handle sphere-resize-${edge}`;
        handle.dataset.edge = edge;
        handle.setAttribute('aria-hidden', 'true');
        handle.addEventListener('mousedown', event => startResize(event, edge));
        windowEl.append(handle);
    }

    function startResize(event, edge) {
        if (windowEl.classList.contains('maximized') || windowEl.classList.contains('minimized')) return;
        event.preventDefault();
        event.stopPropagation();
        focusWindow(windowEl);

        const desktop = document.querySelector('.sphere-desktop');
        if (!desktop) return;

        const startRect = windowEl.getBoundingClientRect();
        const desktopRect = desktop.getBoundingClientRect();
        const startX = event.clientX;
        const startY = event.clientY;
        const startLeft = windowEl.offsetLeft;
        const startTop = windowEl.offsetTop;
        const minWidth = 320;
        const minHeight = 220;

        const onMove = moveEvent => {
            moveEvent.preventDefault();
            const deltaX = moveEvent.clientX - startX;
            const deltaY = moveEvent.clientY - startY;
            let left = startLeft;
            let top = startTop;
            let width = startRect.width;
            let height = startRect.height;

            if (edge.includes('e')) width = startRect.width + deltaX;
            if (edge.includes('s')) height = startRect.height + deltaY;
            if (edge.includes('w')) { width = startRect.width - deltaX; left = startLeft + deltaX; }
            if (edge.includes('n')) { height = startRect.height - deltaY; top = startTop + deltaY; }

            if (width < minWidth && edge.includes('w')) left = startLeft + startRect.width - minWidth;
            if (height < minHeight && edge.includes('n')) top = startTop + startRect.height - minHeight;

            width = Math.max(minWidth, Math.min(width, desktopRect.width - Math.max(0, left) - 12));
            height = Math.max(minHeight, Math.min(height, desktopRect.height - Math.max(0, top) - 12));
            left = Math.max(0, Math.min(left, desktopRect.width - minWidth));
            top = Math.max(0, Math.min(top, desktopRect.height - minHeight));

            windowEl.style.left = `${left}px`;
            windowEl.style.top = `${top}px`;
            windowEl.style.width = `${width}px`;
            windowEl.style.height = `${height}px`;
        };

        const onUp = () => {
            document.removeEventListener('mousemove', onMove);
            document.removeEventListener('mouseup', onUp);
            saveDesktopState();
        };

        document.addEventListener('mousemove', onMove);
        document.addEventListener('mouseup', onUp);
    }
}

/**
 * Create a new window for an app
 * @param {string} appID
 * @returns {HTMLElement}
 */
export function createSphereWindow(appID) {
    const definition = getAppDefinition(appID);
    const layer = document.getElementById('sphere-app-layer');
    if (!definition || !layer) return null;

    let existing = document.getElementById(`sphere-window-${appID}`);
    if (existing) return existing;

    const windowEl = document.createElement('section');
    windowEl.id = `sphere-window-${appID}`;
    windowEl.dataset.appId = appID;
    windowEl.className = `sphere-window sphere-app-window glass-card sphere-accent-${definition.accent || 'cyan'}`;
    windowEl.style.left = `${definition.left || 100}px`;
    windowEl.style.top = `${definition.top || 60}px`;
    windowEl.style.width = `${definition.width || 600}px`;
    windowEl.style.height = `${definition.height || 450}px`;

    const header = document.createElement('header');
    header.className = 'sphere-window-header';

    const titleBox = document.createElement('div');
    titleBox.style.cssText = 'display:flex; align-items:center; gap:8px;';
    
    const iconSpan = document.createElement('span');
    iconSpan.style.color = `var(--vgt-${definition.accent || 'cyan'})`;
    iconSpan.textContent = definition.icon || '◈';

    const title = document.createElement('strong');
    title.className = 'sphere-window-title font-mono';
    title.textContent = definition.title;

    titleBox.append(iconSpan, title);

    const controls = document.createElement('div');
    controls.className = 'window-controls';

    const min = document.createElement('button');
    min.type = 'button';
    min.className = 'win-btn win-btn-min';
    min.title = 'Minimieren (−)';
    min.innerHTML = `<span aria-hidden="true">−</span>`;

    const max = document.createElement('button');
    max.type = 'button';
    max.className = 'win-btn win-btn-max';
    max.title = 'Maximieren / Normalgröße (⤢)';
    max.innerHTML = `<span aria-hidden="true">⤢</span>`;

    const close = document.createElement('button');
    close.type = 'button';
    close.className = 'win-btn win-btn-close';
    close.title = 'Schließen (✕)';
    close.innerHTML = `<span aria-hidden="true">✕</span>`;

    controls.append(min, max, close);
    header.append(titleBox, controls);

    const body = document.createElement('div');
    body.className = 'sphere-app-body';

    windowEl.append(header, body);
    layer.appendChild(windowEl);

    makeDraggable(windowEl, header);
    makeResizable(windowEl);

    windowEl.addEventListener('mousedown', () => focusWindow(windowEl));
    windowEl.addEventListener('touchstart', () => focusWindow(windowEl), { passive: true });

    for (const btn of [min, max, close]) {
        btn.addEventListener('mousedown', event => event.stopPropagation());
    }

    min.addEventListener('click', event => {
        event.stopPropagation();
        windowEl.classList.add('minimized');
        saveDesktopState();
    });

    max.addEventListener('click', event => {
        event.stopPropagation();
        windowEl.classList.remove('minimized');
        windowEl.classList.toggle('maximized');
        focusWindow(windowEl);
        saveDesktopState();
    });

    close.addEventListener('click', event => {
        event.stopPropagation();
        windowEl.classList.add('hidden');
        saveDesktopState();
    });

    return windowEl;
}

/**
 * Save all open window geometries to state and backend
 */
export function saveDesktopState() {
    const windowsState = {};
    document.querySelectorAll('.sphere-window').forEach(win => {
        const appID = win.dataset.appId || win.id.replace('sphere-window-', '');
        windowsState[appID] = {
            app_id: appID,
            left: parseInt(win.style.left, 10) || 0,
            top: parseInt(win.style.top, 10) || 0,
            width: parseInt(win.style.width, 10) || win.offsetWidth,
            height: parseInt(win.style.height, 10) || win.offsetHeight,
            is_minimized: win.classList.contains('minimized'),
            is_maximized: win.classList.contains('maximized'),
            is_open: !win.classList.contains('hidden')
        };
    });

    try {
        localStorage.setItem('aethel_sphere_windows', JSON.stringify(windowsState));
    } catch (e) {
        // quota exceeded
    }
}
