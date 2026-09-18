// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/context_bus.js
// Purpose: Central Pub/Sub Context Bus, Focused Entity State, and Universal SEND TO... Dispatcher

import { state } from '../state.js';
import { createSendToPayload, ObjectType } from './object_model.js';

class ContextBus {
    constructor() {
        this.listeners = new Map();
        this.activeContext = {
            activeDesktop: 'PERSONAL',
            focusedApp: null,
            focusedWindowId: null,
            activeEntity: null,
            selectedText: '',
            activeTrip: null,
            activeDocument: null,
            activeBrowserTab: null,
            activeResearchProject: null
        };
    }

    /**
     * Subscribe to a context event
     * @param {string} event
     * @param {Function} callback
     * @returns {Function} unsubscribe function
     */
    on(event, callback) {
        if (!this.listeners.has(event)) {
            this.listeners.set(event, new Set());
        }
        this.listeners.get(event).add(callback);
        return () => this.off(event, callback);
    }

    /**
     * Unsubscribe from an event
     * @param {string} event
     * @param {Function} callback
     */
    off(event, callback) {
        if (this.listeners.has(event)) {
            this.listeners.get(event).delete(callback);
        }
    }

    /**
     * Emit an event to all subscribers
     * @param {string} event
     * @param {*} data
     */
    emit(event, data) {
        if (this.listeners.has(event)) {
            for (const callback of this.listeners.get(event)) {
                try {
                    callback(data);
                } catch (err) {
                    console.error(`[ContextBus] Error in listener for ${event}:`, err);
                }
            }
        }
    }

    /**
     * Update active context state and broadcast change
     * @param {Object} partialContext
     */
    updateContext(partialContext) {
        Object.assign(this.activeContext, partialContext);
        this.emit('context:changed', this.activeContext);
    }

    /**
     * Get current snapshot of context
     * @returns {Object}
     */
    getContext() {
        return { ...this.activeContext };
    }

    /**
     * Open the Universal SEND TO... Dialog
     * @param {Object} item - { sourceApp, title, content, objectType, metadata }
     */
    openSendToDialog(item) {
        let existing = document.getElementById('sphere-sendto-modal');
        if (existing) existing.remove();

        const modal = document.createElement('div');
        modal.id = 'sphere-sendto-modal';
        modal.className = 'sphere-modal-backdrop';
        modal.style.cssText = 'position:fixed; inset:0; z-index:10050; background:rgba(3,6,15,0.88); backdrop-filter:blur(12px); display:flex; justify-content:center; align-items:center; font-family:var(--font-mono);';

        const card = document.createElement('div');
        card.className = 'sphere-sendto-card glass-card';
        card.style.cssText = 'width:90%; max-width:540px; background:linear-gradient(145deg, rgba(8,16,32,0.98), rgba(4,8,18,0.99)); border:1px solid var(--vgt-cyan); border-radius:12px; padding:22px; box-shadow:0 0 50px rgba(0,240,255,0.25); display:flex; flex-direction:column; gap:14px; color:#fff;';

        const header = document.createElement('div');
        header.style.cssText = 'display:flex; justify-content:space-between; align-items:center; border-bottom:1px solid rgba(0,240,255,0.2); padding-bottom:12px;';
        
        const titleText = document.createElement('strong');
        titleText.style.cssText = 'font-size:13px; color:var(--vgt-cyan); letter-spacing:0.05em; display:flex; align-items:center; gap:8px;';
        titleText.textContent = `SEND TO // ${item.sourceApp ? item.sourceApp.toUpperCase() : 'SPHERE'} ➔ ZIEL-APP`;

        const closeBtn = document.createElement('button');
        closeBtn.type = 'button';
        closeBtn.style.cssText = 'background:none; border:none; color:var(--vgt-text-dim); font-size:16px; cursor:pointer; padding:4px;';
        closeBtn.textContent = '✕';
        closeBtn.addEventListener('click', () => modal.remove());

        header.append(titleText, closeBtn);

        const previewBox = document.createElement('div');
        previewBox.style.cssText = 'background:rgba(0,0,0,0.4); border:1px solid rgba(255,255,255,0.08); border-radius:8px; padding:12px; font-size:11px;';
        
        const previewTitle = document.createElement('strong');
        previewTitle.style.cssText = 'display:block; color:#fff; margin-bottom:4px;';
        previewTitle.textContent = item.title || 'Ohne Titel';

        const previewSnippet = document.createElement('p');
        previewSnippet.style.cssText = 'margin:0; color:rgba(255,255,255,0.7); max-height:80px; overflow-y:auto; line-height:1.4;';
        previewSnippet.textContent = (item.content || '').replace(/<[^>]*>/g, '').slice(0, 300) + '...';

        previewBox.append(previewTitle, previewSnippet);

        const targetsGrid = document.createElement('div');
        targetsGrid.style.cssText = 'display:grid; grid-template-columns:1fr 1fr; gap:10px; margin-top:6px;';

        const targets = [
            { id: 'writer', label: '📄 AETHEL WRITER', desc: 'Als Dokument oder Anhang einfügen', accent: 'cyan' },
            { id: 'research', label: '🔬 RESEARCH DESK', desc: 'Als Quelle & Beweisstück erfassen', accent: 'purple' },
            { id: 'trip', label: '🗺 TRIP PLANNER', desc: 'Zu aktiver Reise / Itinerary hinzufügen', accent: 'orange' },
            { id: 'task', label: '⏰ MASTER PLANNER', desc: 'In Aufgabe oder Meilenstein umwandeln', accent: 'cyan' }
        ];

        targets.forEach(t => {
            const btn = document.createElement('button');
            btn.type = 'button';
            btn.className = 'sphere-sendto-target-btn cyber-button';
            btn.style.cssText = `display:flex; flex-direction:column; align-items:flex-start; gap:4px; padding:12px; background:rgba(255,255,255,0.03); border:1px solid rgba(0,240,255,0.25); border-radius:8px; cursor:pointer; text-align:left; transition:all 0.2s;`;
            
            const btnTitle = document.createElement('strong');
            btnTitle.style.cssText = 'font-size:11px; color:#fff;';
            btnTitle.textContent = t.label;

            const btnDesc = document.createElement('small');
            btnDesc.style.cssText = 'font-size:9px; color:var(--vgt-text-dim);';
            btnDesc.textContent = t.desc;

            btn.append(btnTitle, btnDesc);

            btn.addEventListener('mouseenter', () => {
                btn.style.background = 'rgba(0,240,255,0.12)';
                btn.style.borderColor = 'var(--vgt-cyan)';
            });
            btn.addEventListener('mouseleave', () => {
                btn.style.background = 'rgba(255,255,255,0.03)';
                btn.style.borderColor = 'rgba(0,240,255,0.25)';
            });

            btn.addEventListener('click', async () => {
                btn.disabled = true;
                btnTitle.textContent = '⏳ ÜBERTRAGE...';

                try {
                    const payload = createSendToPayload(
                        item.sourceApp || 'sphere',
                        t.id,
                        item.title,
                        item.content,
                        item.objectType || ObjectType.DOCUMENT,
                        item.metadata || {}
                    );

                    const response = await fetch(`${state.API_BASE}/v1/sphere/send_to`, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify(payload)
                    });

                    if (!response.ok) throw new Error(`HTTP ${response.status}`);
                    const result = await response.json();

                    modal.remove();
                    window.dispatchEvent(new CustomEvent('aethel:toast', {
                        detail: { message: `Erfolgreich an ${t.label} gesendet!`, type: 'success' }
                    }));

                    // Automatically open or refresh target app
                    if (window.openSphereApp) {
                        window.openSphereApp(t.id);
                    }
                    this.emit('object:transferred', { targetApp: t.id, result });
                } catch (err) {
                    btn.disabled = false;
                    btnTitle.textContent = '❌ FEHLER';
                    console.error('Send to failed:', err);
                }
            });

            targetsGrid.appendChild(btn);
        });

        card.append(header, previewBox, targetsGrid);
        modal.appendChild(card);
        document.body.appendChild(modal);

        modal.addEventListener('click', (e) => {
            if (e.target === modal) modal.remove();
        });
    }
}

export const contextBus = new ContextBus();
window.contextBus = contextBus;
