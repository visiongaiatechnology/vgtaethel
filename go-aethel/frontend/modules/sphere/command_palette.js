// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/command_palette.js
// Purpose: Universal Command Palette (Ctrl+K / Voice / Fuzzy Search across Apps, Objects & Natural Language Actions)

import { state } from '../state.js';
import { sendMessage } from '../chat.js';
import { sphereAppDefinitions } from './app_registry.js';
import { contextBus } from './context_bus.js';

let paletteModalEl = null;

/**
 * Initialize Command Palette triggers and keyboard shortcut
 */
export function setupCommandPalette() {
    // Listen for Ctrl+K or Cmd+K
    document.addEventListener('keydown', (e) => {
        if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
            const sphereView = document.getElementById('view-sphere');
            if (sphereView && !sphereView.classList.contains('hidden')) {
                e.preventDefault();
                openCommandPalette();
            }
        }
    });

    // Wire existing bottom command bar
    const bottomBar = document.getElementById('sphere-command-form');
    const bottomInput = document.getElementById('sphere-command-input');
    if (bottomBar && bottomInput && bottomBar.dataset.vgtPaletteBound !== 'true') {
        bottomBar.dataset.vgtPaletteBound = 'true';
        bottomInput.addEventListener('focus', () => {
            // Optional: can open full palette or handle directly
        });
    }
}

/**
 * Open the Universal Command Palette Modal
 */
export function openCommandPalette() {
    if (paletteModalEl) paletteModalEl.remove();

    paletteModalEl = document.createElement('div');
    paletteModalEl.id = 'sphere-command-palette-modal';
    paletteModalEl.className = 'sphere-modal-backdrop';
    paletteModalEl.style.cssText = 'position:fixed; inset:0; z-index:10060; background:rgba(2,5,14,0.85); backdrop-filter:blur(14px); display:flex; justify-content:center; align-items:flex-start; padding-top:12vh; font-family:var(--font-mono);';

    const panel = document.createElement('div');
    panel.className = 'sphere-palette-panel glass-card';
    panel.style.cssText = 'width:90%; max-width:640px; background:linear-gradient(145deg, rgba(8,16,34,0.98), rgba(3,7,18,0.99)); border:1px solid var(--vgt-cyan); border-radius:12px; box-shadow:0 0 60px rgba(0,240,255,0.25); overflow:hidden; display:flex; flex-direction:column; color:#fff;';

    const inputRow = document.createElement('div');
    inputRow.style.cssText = 'display:flex; align-items:center; gap:10px; padding:16px 20px; border-bottom:1px solid rgba(0,240,255,0.2); background:rgba(0,0,0,0.3);';

    const promptSymbol = document.createElement('span');
    promptSymbol.style.cssText = 'color:var(--vgt-cyan); font-size:16px; font-weight:bold;';
    promptSymbol.textContent = '◈';

    const input = document.createElement('input');
    input.type = 'text';
    input.placeholder = 'Aethel befehlen oder suchen (z.B. "Reise Japan", "Neues Dokument", "Ölpreis prüfen")...';
    input.style.cssText = 'flex:1; background:none; border:none; color:#fff; font-size:14px; font-family:var(--font-mono); outline:none;';

    const escKey = document.createElement('kbd');
    escKey.style.cssText = 'font-size:10px; color:var(--vgt-text-dim); background:rgba(255,255,255,0.08); padding:3px 6px; border-radius:4px;';
    escKey.textContent = 'ESC';

    inputRow.append(promptSymbol, input, escKey);

    const list = document.createElement('div');
    list.className = 'sphere-palette-list';
    list.style.cssText = 'max-height:360px; overflow-y:auto; padding:10px; display:flex; flex-direction:column; gap:4px;';

    const footer = document.createElement('footer');
    footer.style.cssText = 'padding:10px 20px; border-top:1px solid rgba(255,255,255,0.06); font-size:10px; color:var(--vgt-text-dim); display:flex; justify-content:space-between; align-items:center; background:rgba(0,0,0,0.2);';
    footer.innerHTML = `<span>▲▼ Navigieren · ↵ Ausführen · Tab Vervollständigen</span><span style="color:var(--vgt-cyan);">AETHEL KERNEL 2.1</span>`;

    panel.append(inputRow, list, footer);
    paletteModalEl.appendChild(panel);
    document.body.appendChild(paletteModalEl);

    input.focus();

    let selectedIndex = 0;
    let currentItems = [];

    const defaultActions = [
        { id: 'app_dashboard', title: 'Personal Operations Dashboard öffnen', category: 'APP', icon: '◫', action: () => window.openSphereApp('dashboard') },
        { id: 'app_writer', title: 'Aethel Writer (Word/Notion Canvas) öffnen', category: 'APP', icon: '✦', action: () => window.openSphereApp('writer') },
        { id: 'app_browser', title: 'Aethel Browser (Multi-Tab & AI Control) öffnen', category: 'APP', icon: '◌', action: () => window.openSphereApp('browser') },
        { id: 'app_travel', title: 'Trip Planner & Global Watch Bridge öffnen', category: 'APP', icon: '🗺', action: () => window.openSphereApp('travel') },
        { id: 'app_planner', title: 'Master Planner & Meilensteine öffnen', category: 'APP', icon: '⏰', action: () => window.openSphereApp('planner') },
        { id: 'app_research', title: 'Research Desk & Quellen-Evidence öffnen', category: 'APP', icon: '🔬', action: () => window.openSphereApp('research') },
        { id: 'app_files', title: 'Vault Explorer & Dateiverwaltung öffnen', category: 'APP', icon: '▤', action: () => window.openSphereApp('files') },
        { id: 'app_markets', title: 'Live Market Engine (Öl, Krypto, Aktien) öffnen', category: 'APP', icon: '◇', action: () => window.openSphereApp('market') },
        { id: 'app_weather', title: 'Weather Satellite Pulse öffnen', category: 'APP', icon: '☼', action: () => window.openSphereApp('weather') }
    ];

    async function renderItems(query) {
        list.replaceChildren();
        currentItems = [];

        const q = (query || '').toLowerCase().trim();

        // 1. If query is present, add AI natural language prompt execution option at top
        if (q) {
            currentItems.push({
                id: 'ai_prompt',
                title: `Aethel anweisen: "${query}"`,
                category: 'AI ACTION',
                icon: '⚡',
                action: async () => {
                    paletteModalEl.remove();
                    const sharedInput = document.getElementById('user-input');
                    if (sharedInput) {
                        sharedInput.value = query;
                        await sendMessage();
                    }
                }
            });
        }

        // 2. Filter default app actions
        for (const item of defaultActions) {
            if (!q || item.title.toLowerCase().includes(q) || item.category.toLowerCase().includes(q)) {
                currentItems.push(item);
            }
        }

        // 3. Search Objects from backend if query length >= 2
        if (q.length >= 2) {
            try {
                const res = await fetch(`${state.API_BASE}/v1/sphere/search?q=${encodeURIComponent(q)}`);
                if (res.ok) {
                    const data = await res.json();
                    const results = data.results || [];
                    for (const obj of results.slice(0, 8)) {
                        currentItems.push({
                            id: `obj_${obj.id}`,
                            title: `${obj.title} (${obj.summary || obj.type})`,
                            category: obj.type.toUpperCase(),
                            icon: obj.type === 'trip' ? '🗺' : obj.type === 'document' ? '📄' : obj.type === 'plan' ? '⏰' : '🔬',
                            action: () => {
                                paletteModalEl.remove();
                                if (obj.type === 'document' && window.openSphereApp) window.openSphereApp('writer');
                                else if (obj.type === 'trip' && window.openSphereApp) window.openSphereApp('travel');
                                else if (obj.type === 'plan' && window.openSphereApp) window.openSphereApp('planner');
                                else if (obj.type === 'research_item' && window.openSphereApp) window.openSphereApp('research');
                            }
                        });
                    }
                }
            } catch (err) {
                // Ignore transient search failure
            }
        }

        selectedIndex = 0;
        currentItems.forEach((item, index) => {
            const row = document.createElement('div');
            row.className = `sphere-palette-row ${index === selectedIndex ? 'selected' : ''}`;
            row.style.cssText = `display:flex; justify-content:space-between; align-items:center; padding:8px 12px; border-radius:6px; cursor:pointer; transition:all 0.15s; ${index === selectedIndex ? 'background:rgba(0,240,255,0.15); border-left:3px solid var(--vgt-cyan);' : 'background:rgba(255,255,255,0.02);'}`;

            const left = document.createElement('div');
            left.style.cssText = 'display:flex; align-items:center; gap:10px; font-size:12px;';

            const iconSpan = document.createElement('span');
            iconSpan.style.color = 'var(--vgt-cyan)';
            iconSpan.textContent = item.icon;

            const textSpan = document.createElement('span');
            textSpan.textContent = item.title;

            left.append(iconSpan, textSpan);

            const badge = document.createElement('span');
            badge.style.cssText = 'font-size:9px; background:rgba(0,240,255,0.08); border:1px solid rgba(0,240,255,0.2); color:var(--vgt-cyan); padding:2px 6px; border-radius:4px;';
            badge.textContent = item.category;

            row.append(left, badge);

            row.addEventListener('mouseenter', () => {
                selectedIndex = index;
                updateSelection();
            });

            row.addEventListener('click', () => {
                paletteModalEl.remove();
                item.action();
            });

            list.appendChild(row);
        });
    }

    function updateSelection() {
        const rows = list.querySelectorAll('.sphere-palette-row');
        rows.forEach((row, i) => {
            const isSel = i === selectedIndex;
            row.classList.toggle('selected', isSel);
            row.style.background = isSel ? 'rgba(0,240,255,0.15)' : 'rgba(255,255,255,0.02)';
            row.style.borderLeft = isSel ? '3px solid var(--vgt-cyan)' : 'none';
        });
    }

    input.addEventListener('input', (e) => {
        void renderItems(e.target.value);
    });

    input.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            paletteModalEl.remove();
        } else if (e.key === 'ArrowDown') {
            e.preventDefault();
            if (currentItems.length > 0) {
                selectedIndex = (selectedIndex + 1) % currentItems.length;
                updateSelection();
            }
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            if (currentItems.length > 0) {
                selectedIndex = (selectedIndex - 1 + currentItems.length) % currentItems.length;
                updateSelection();
            }
        } else if (e.key === 'Enter') {
            e.preventDefault();
            if (currentItems[selectedIndex]) {
                paletteModalEl.remove();
                currentItems[selectedIndex].action();
            }
        }
    });

    paletteModalEl.addEventListener('click', (e) => {
        if (e.target === paletteModalEl) paletteModalEl.remove();
    });

    void renderItems('');
}
