// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/apps/files.js
// Purpose: Vault Explorer & Workspace File Manager with Snapshots & Send-To

import { state } from '../../state.js';
import { contextBus } from '../context_bus.js';

/**
 * Render Vault Explorer inside a window body
 * @param {HTMLElement} body
 */
export async function renderFilesApp(body) {
    if (!body) return;
    body.replaceChildren();

    const container = document.createElement('div');
    container.className = 'sphere-files-container font-mono';
    container.style.cssText = 'display:flex; flex-direction:column; height:100%; gap:8px; color:#fff;';

    // Topbar
    const header = document.createElement('div');
    header.style.cssText = 'display:flex; justify-content:space-between; align-items:center; border-bottom:1px solid rgba(0,240,255,0.2); padding-bottom:8px;';

    const breadcrumbs = document.createElement('div');
    breadcrumbs.style.cssText = 'font-size:11px; color:var(--vgt-cyan);';
    breadcrumbs.textContent = '📁 VGT_WORKSPACE // ROOT';

    const refreshBtn = document.createElement('button');
    refreshBtn.type = 'button';
    refreshBtn.className = 'cyber-button';
    refreshBtn.style.cssText = 'font-size:10px; padding:3px 8px; background:rgba(0,240,255,0.1); border:1px solid var(--vgt-cyan); color:var(--vgt-cyan); cursor:pointer; width:auto;';
    refreshBtn.textContent = '🔄 AKTUALISIEREN';

    header.append(breadcrumbs, refreshBtn);

    // List area
    const listArea = document.createElement('div');
    listArea.className = 'sphere-files-list';
    listArea.style.cssText = 'flex:1; overflow-y:auto; display:flex; flex-direction:column; gap:4px; padding-right:4px;';

    container.append(header, listArea);
    body.appendChild(container);

    const loadWorkspace = async () => {
        listArea.replaceChildren();
        const loading = document.createElement('span');
        loading.style.cssText = 'font-size:11px; color:var(--vgt-text-dim); padding:10px;';
        loading.textContent = 'Lade Workspace-Index...';
        listArea.appendChild(loading);

        try {
            const response = await fetch(`${state.API_BASE}/v1/sphere/workspace`);
            if (!response.ok) throw new Error(`Workspace ${response.status}`);
            const payload = await response.json();
            const entries = payload.entries || [];

            listArea.replaceChildren();
            if (entries.length === 0) {
                const empty = document.createElement('span');
                empty.style.cssText = 'font-size:11px; color:var(--vgt-text-dim); text-align:center; padding:20px;';
                empty.textContent = 'Noch keine sichtbaren Workspace-Dateien.';
                listArea.appendChild(empty);
                return;
            }

            for (const entry of entries) {
                const row = document.createElement('div');
                row.style.cssText = 'display:flex; justify-content:space-between; align-items:center; padding:6px 10px; background:rgba(0,0,0,0.3); border:1px solid rgba(255,255,255,0.05); border-radius:6px; font-size:11px; cursor:pointer; transition:all 0.15s;';
                
                const isDir = entry.kind === 'folder';
                const left = document.createElement('span');
                left.style.cssText = 'display:flex; align-items:center; gap:6px;';
                left.innerHTML = `<span style="color:${isDir ? 'var(--vgt-orange)' : 'var(--vgt-cyan)'};">${isDir ? '▸ 📁' : '· 📄'}</span> <span>${entry.path}</span>`;

                const meta = document.createElement('small');
                meta.style.cssText = 'font-size:9px; color:var(--vgt-text-dim);';
                meta.textContent = isDir ? 'ORDNER' : `${Number(entry.size || 0).toLocaleString('de-DE')} B`;

                row.append(left, meta);

                row.addEventListener('mouseenter', () => {
                    row.style.background = 'rgba(0,240,255,0.08)';
                    row.style.borderColor = 'rgba(0,240,255,0.2)';
                });
                row.addEventListener('mouseleave', () => {
                    row.style.background = 'rgba(0,0,0,0.3)';
                    row.style.borderColor = 'rgba(255,255,255,0.05)';
                });

                row.addEventListener('click', () => {
                    if (!isDir) {
                        contextBus.openSendToDialog({
                            sourceApp: 'files',
                            title: entry.path,
                            content: `Datei: ${entry.path} (${entry.size} Bytes)`
                        });
                    }
                });

                listArea.appendChild(row);
            }
        } catch (error) {
            listArea.replaceChildren();
            const failure = document.createElement('span');
            failure.style.cssText = 'font-size:11px; color:var(--vgt-red); padding:10px;';
            failure.textContent = `Workspace nicht verfügbar: ${error.message}`;
            listArea.appendChild(failure);
        }
    };

    refreshBtn.addEventListener('click', () => void loadWorkspace());
    void loadWorkspace();
}
