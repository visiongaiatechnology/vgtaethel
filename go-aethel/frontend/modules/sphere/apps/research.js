// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/apps/research.js
// Purpose: Research Desk, Evidence Collection, Provenance Tracking, and Briefing Synthesis

import { state } from '../../state.js';
import { contextBus } from '../context_bus.js';

let projectsList = [];
let activeProject = null;
let projectItems = [];

/**
 * Render Research Desk inside a window body
 * @param {HTMLElement} body
 */
export async function renderResearchApp(body) {
    if (!body) return;
    body.replaceChildren();

    const container = document.createElement('div');
    container.className = 'sphere-research-container font-mono';
    container.style.cssText = 'display:flex; flex-direction:column; height:100%; gap:10px; color:#fff;';

    // Topbar
    const header = document.createElement('div');
    header.style.cssText = 'display:flex; justify-content:space-between; align-items:center; border-bottom:1px solid rgba(138,43,226,0.25); padding-bottom:8px;';

    const projSelect = document.createElement('select');
    projSelect.id = 'sphere-research-select';
    projSelect.style.cssText = 'background:rgba(0,0,0,0.4); border:1px solid rgba(138,43,226,0.3); color:#fff; padding:4px 8px; border-radius:4px; font-size:11px; font-family:var(--font-mono); outline:none;';

    const rightActions = document.createElement('div');
    rightActions.style.cssText = 'display:flex; gap:6px;';

    const btnBriefing = document.createElement('button');
    btnBriefing.type = 'button';
    btnBriefing.className = 'cyber-button';
    btnBriefing.style.cssText = 'font-size:10px; padding:4px 10px; background:rgba(0,240,255,0.12); border:1px solid var(--vgt-cyan); color:var(--vgt-cyan); cursor:pointer; width:auto;';
    btnBriefing.textContent = '📄 BRIEFING GENERIEREN';
    btnBriefing.addEventListener('click', () => generateAndSendBriefing());

    const btnNewClip = document.createElement('button');
    btnNewClip.type = 'button';
    btnNewClip.className = 'cyber-button';
    btnNewClip.style.cssText = 'font-size:10px; padding:4px 10px; background:rgba(138,43,226,0.2); border:1px solid var(--vgt-purple); color:var(--vgt-purple); cursor:pointer; width:auto;';
    btnNewClip.textContent = '+ AUSZUG SICHERN';
    btnNewClip.addEventListener('click', () => openNewClipModal());

    rightActions.append(btnBriefing, btnNewClip);
    header.append(projSelect, rightActions);

    // Main Clips List
    const clipsList = document.createElement('div');
    clipsList.className = 'sphere-research-clips';
    clipsList.style.cssText = 'flex:1; overflow-y:auto; display:flex; flex-direction:column; gap:10px; padding-right:4px;';

    container.append(header, clipsList);
    body.appendChild(container);

    async function loadProjects() {
        try {
            const res = await fetch(`${state.API_BASE}/v1/sphere/research`);
            if (res.ok) {
                const data = await res.json();
                projectsList = data.projects || [];
                if (projectsList.length === 0) {
                    await createDefaultSampleProject();
                    return;
                }
                activeProject = projectsList[0];
                updateProjectSelect();
                await loadProjectClips();
            }
        } catch (err) {
            console.error('Failed to load research projects', err);
        }
    }

    function updateProjectSelect() {
        projSelect.replaceChildren();
        projectsList.forEach(p => {
            const opt = document.createElement('option');
            opt.value = p.id;
            opt.textContent = `${p.title} (${p.item_count || 0} Quellen)`;
            if (activeProject && p.id === activeProject.id) opt.selected = true;
            projSelect.appendChild(opt);
        });
    }

    projSelect.addEventListener('change', async () => {
        activeProject = projectsList.find(p => p.id === projSelect.value);
        await loadProjectClips();
    });

    async function loadProjectClips() {
        clipsList.replaceChildren();
        if (!activeProject) return;

        try {
            const res = await fetch(`${state.API_BASE}/v1/sphere/research/clips?project_id=${encodeURIComponent(activeProject.id)}`);
            if (res.ok) {
                const data = await res.json();
                projectItems = data.items || [];
                renderClips();
            }
        } catch (err) {
            console.error('Failed to load research clips', err);
        }
    }

    function renderClips() {
        clipsList.replaceChildren();
        if (projectItems.length === 0) {
            const empty = document.createElement('div');
            empty.style.cssText = 'text-align:center; padding:40px 10px; color:var(--vgt-text-dim); font-size:11px;';
            empty.textContent = 'Noch keine Recherche-Auszüge gesichert. Klicke auf "+ AUSZUG SICHERN" oder nutze Browser / Send-To.';
            clipsList.appendChild(empty);
            return;
        }

        projectItems.forEach(item => {
            const card = document.createElement('article');
            card.className = 'glass-card';
            card.style.cssText = 'background:linear-gradient(135deg, rgba(138,43,226,0.06), rgba(0,0,0,0.4)); border:1px solid rgba(138,43,226,0.25); border-radius:8px; padding:12px; display:flex; flex-direction:column; gap:6px;';

            const topRow = document.createElement('div');
            topRow.style.cssText = 'display:flex; justify-content:space-between; align-items:center;';

            const title = document.createElement('strong');
            title.style.cssText = 'font-size:12px; color:#fff;';
            title.textContent = item.title;

            const provBadge = document.createElement('span');
            provBadge.style.cssText = 'font-size:8px; background:rgba(138,43,226,0.2); color:var(--vgt-purple); padding:1px 6px; border-radius:4px; border:1px solid var(--vgt-purple);';
            provBadge.textContent = item.provenance || 'Aethel Research';

            topRow.append(title, provBadge);

            const quote = document.createElement('blockquote');
            quote.style.cssText = 'margin:4px 0; padding-left:8px; border-left:2px solid var(--vgt-cyan); color:rgba(255,255,255,0.85); font-size:11px; line-height:1.4; max-height:80px; overflow-y:auto;';
            quote.textContent = item.extracted_text;

            const footer = document.createElement('div');
            footer.style.cssText = 'display:flex; justify-content:space-between; align-items:center; font-size:9px; color:var(--vgt-text-dim); margin-top:2px;';

            const srcLink = document.createElement('span');
            srcLink.textContent = item.source_url ? `Quelle: ${item.source_title || item.source_url}` : 'Manuelle Notiz';

            const btnSend = document.createElement('button');
            btnSend.type = 'button';
            btnSend.style.cssText = 'background:none; border:none; color:var(--vgt-cyan); font-size:9px; cursor:pointer; padding:2px 4px;';
            btnSend.textContent = '➔ SEND TO WRITER';
            btnSend.addEventListener('click', () => {
                contextBus.openSendToDialog({
                    sourceApp: 'research',
                    title: item.title,
                    content: item.extracted_text,
                    metadata: { url: item.source_url }
                });
            });

            footer.append(srcLink, btnSend);
            card.append(topRow, quote, footer);
            clipsList.appendChild(card);
        });
    }

    async function generateAndSendBriefing() {
        if (!activeProject) return;
        btnBriefing.disabled = true;
        btnBriefing.textContent = '⏳ SYNTHETISIERE...';

        try {
            const res = await fetch(`${state.API_BASE}/v1/sphere/research/briefing?project_id=${encodeURIComponent(activeProject.id)}`);
            if (res.ok) {
                const data = await res.json();
                const briefingMD = data.briefing || '';
                
                // Open Send To Dialog targeting Writer
                contextBus.openSendToDialog({
                    sourceApp: 'research',
                    targetApp: 'writer',
                    title: `Briefing: ${activeProject.title}`,
                    content: briefingMD
                });
            }
        } catch (err) {
            console.error('Briefing generation failed', err);
        } finally {
            btnBriefing.disabled = false;
            btnBriefing.textContent = '📄 BRIEFING GENERIEREN';
        }
    }

    async function createDefaultSampleProject() {
        const defaultProj = {
            title: 'KI-Sicherheitsarchitektur & Guard Kernel',
            description: 'Forschung zu Zero-Trust Sandboxing, Least-Privilege und Memory Safety',
            tags: ['security', 'ai', 'kernel']
        };
        try {
            const res = await fetch(`${state.API_BASE}/v1/sphere/research`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(defaultProj)
            });
            if (res.ok) {
                const created = await res.json();
                projectsList = [created];
                activeProject = created;
                updateProjectSelect();
                renderClips();
            }
        } catch (e) {
            console.error('Failed to create sample research project', e);
        }
    }

    function openNewClipModal() {
        let dialog = document.getElementById('sphere-new-clip-dialog');
        if (dialog) dialog.remove();

        dialog = document.createElement('div');
        dialog.id = 'sphere-new-clip-dialog';
        dialog.style.cssText = 'position:fixed; inset:0; z-index:10080; background:rgba(3,6,15,0.85); backdrop-filter:blur(10px); display:flex; justify-content:center; align-items:center; font-family:var(--font-mono);';

        const modal = document.createElement('div');
        modal.className = 'glass-card';
        modal.style.cssText = 'width:90%; max-width:480px; background:linear-gradient(145deg, rgba(16,8,32,0.98), rgba(4,6,14,0.99)); border:1px solid var(--vgt-purple); border-radius:12px; padding:20px; box-shadow:0 0 40px rgba(138,43,226,0.25); display:flex; flex-direction:column; gap:12px; color:#fff;';

        modal.innerHTML = `
            <div style="display:flex; justify-content:space-between; align-items:center; border-bottom:1px solid rgba(138,43,226,0.2); padding-bottom:10px;">
                <strong style="color:var(--vgt-purple); font-size:13px;">🔬 RECHERCHE-AUSZUG SICHERN</strong>
                <button id="clip-modal-close" style="background:none; border:none; color:var(--vgt-text-dim); font-size:16px; cursor:pointer;">✕</button>
            </div>
            <div style="display:flex; flex-direction:column; gap:10px; font-size:11px;">
                <div>
                    <label style="display:block; color:var(--vgt-text-dim); margin-bottom:4px;">TITEL DES AUSZUGS</label>
                    <input id="clip-input-title" type="text" placeholder="z.B. Zero-Trust Sandbox Isolation" style="width:100%; background:rgba(0,0,0,0.4); border:1px solid rgba(255,255,255,0.1); color:#fff; padding:6px 10px; border-radius:4px; font-family:var(--font-mono); outline:none;" />
                </div>
                <div>
                    <label style="display:block; color:var(--vgt-text-dim); margin-bottom:4px;">QUELLEN-URL (OPTIONAL)</label>
                    <input id="clip-input-url" type="text" placeholder="https://..." style="width:100%; background:rgba(0,0,0,0.4); border:1px solid rgba(255,255,255,0.1); color:#fff; padding:6px 10px; border-radius:4px; font-family:var(--font-mono); outline:none;" />
                </div>
                <div>
                    <label style="display:block; color:var(--vgt-text-dim); margin-bottom:4px;">ZITAT / BELEIS-TEXT</label>
                    <textarea id="clip-input-text" rows="4" placeholder="Wichtiges Zitat oder Datenpunkt..." style="width:100%; background:rgba(0,0,0,0.4); border:1px solid rgba(255,255,255,0.1); color:#fff; padding:6px 10px; border-radius:4px; font-family:var(--font-mono); outline:none;"></textarea>
                </div>
            </div>
            <div style="display:flex; justify-content:flex-end; gap:8px; margin-top:8px;">
                <button id="clip-modal-cancel" class="cyber-button" style="font-size:10px; padding:6px 12px; background:rgba(255,255,255,0.05); border:1px solid rgba(255,255,255,0.1); color:#fff; cursor:pointer; width:auto;">ABBRECHEN</button>
                <button id="clip-modal-save" class="cyber-button" style="font-size:10px; padding:6px 14px; background:rgba(138,43,226,0.2); border:1px solid var(--vgt-purple); color:var(--vgt-purple); cursor:pointer; width:auto;">AUSZUG SPEICHERN</button>
            </div>
        `;

        dialog.appendChild(modal);
        document.body.appendChild(dialog);

        dialog.querySelector('#clip-modal-close').addEventListener('click', () => dialog.remove());
        dialog.querySelector('#clip-modal-cancel').addEventListener('click', () => dialog.remove());

        dialog.querySelector('#clip-modal-save').addEventListener('click', async () => {
            const title = dialog.querySelector('#clip-input-title').value.trim();
            const url = dialog.querySelector('#clip-input-url').value.trim();
            const text = dialog.querySelector('#clip-input-text').value.trim();

            if (!title || !text) {
                alert('Bitte Titel und Text eingeben.');
                return;
            }

            try {
                const res = await fetch(`${state.API_BASE}/v1/sphere/research/clips`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        project_id: activeProject ? activeProject.id : '',
                        title: title,
                        source_url: url,
                        extracted_text: text,
                        provenance: 'Aethel Research Desk'
                    })
                });
                if (res.ok) {
                    dialog.remove();
                    await loadProjectClips();
                }
            } catch (e) {
                alert('Fehler: ' + e.message);
            }
        });
    }

    void loadProjects();
}
