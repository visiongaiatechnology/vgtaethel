// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/sidecar.js
// Purpose: Context-Aware Aethel Sidecar Assistant, Contextual Suggestions & Actions

import { state } from '../state.js';
import { sendMessage } from '../chat.js';
import { contextBus } from './context_bus.js';

let sidecarContainer = null;
let sidecarContentEl = null;
let isSidecarCollapsed = false;

/**
 * Setup and attach the Aethel Sidecar panel
 */
export function setupAethelSidecar() {
    const desktop = document.querySelector('.sphere-desktop');
    if (!desktop || document.getElementById('sphere-sidecar-drawer')) return;

    sidecarContainer = document.createElement('aside');
    sidecarContainer.id = 'sphere-sidecar-drawer';
    sidecarContainer.className = 'sphere-sidecar-drawer glass-card font-mono';
    sidecarContainer.style.cssText = 'position:absolute; right:14px; top:14px; bottom:84px; width:300px; z-index:85; background:rgba(6,12,24,0.92); backdrop-filter:blur(20px); border:1px solid rgba(0,240,255,0.25); border-radius:14px; display:flex; flex-direction:column; box-shadow:0 0 40px rgba(0,0,0,0.6); transition:all 0.3s cubic-bezier(0.16, 1, 0.3, 1); pointer-events:auto;';

    // Header
    const header = document.createElement('header');
    header.style.cssText = 'display:flex; justify-content:space-between; align-items:center; padding:12px 14px; border-bottom:1px solid rgba(0,240,255,0.15); background:rgba(0,0,0,0.3);';

    const brand = document.createElement('div');
    brand.style.cssText = 'display:flex; align-items:center; gap:8px;';

    const dot = document.createElement('span');
    dot.style.cssText = 'width:8px; height:8px; border-radius:50%; background:var(--vgt-cyan); box-shadow:0 0 8px var(--vgt-cyan);';

    const title = document.createElement('strong');
    title.style.cssText = 'font-size:11px; color:#fff; letter-spacing:0.05em;';
    title.textContent = 'AETHEL SIDECAR';

    brand.append(dot, title);

    const toggleBtn = document.createElement('button');
    toggleBtn.type = 'button';
    toggleBtn.className = 'sphere-sidecar-toggle';
    toggleBtn.style.cssText = 'background:none; border:none; color:var(--vgt-text-dim); font-size:12px; cursor:pointer; padding:2px 6px;';
    toggleBtn.textContent = '▶';
    toggleBtn.title = 'Sidecar ein-/ausklappen';

    toggleBtn.addEventListener('click', () => {
        isSidecarCollapsed = !isSidecarCollapsed;
        if (isSidecarCollapsed) {
            sidecarContainer.style.width = '44px';
            sidecarContentEl.style.display = 'none';
            toggleBtn.textContent = '◀';
        } else {
            sidecarContainer.style.width = '300px';
            sidecarContentEl.style.display = 'flex';
            toggleBtn.textContent = '▶';
        }
    });

    header.append(brand, toggleBtn);

    // Scrollable Content
    sidecarContentEl = document.createElement('div');
    sidecarContentEl.className = 'sphere-sidecar-body';
    sidecarContentEl.style.cssText = 'flex:1; overflow-y:auto; padding:12px; display:flex; flex-direction:column; gap:12px; font-size:11px; color:#fff;';

    sidecarContainer.append(header, sidecarContentEl);
    desktop.appendChild(sidecarContainer);

    // Subscribe to context changes
    contextBus.on('context:changed', (ctx) => {
        updateSidecarContent(ctx);
    });

    // Initial render
    updateSidecarContent(contextBus.getContext());
}

/**
 * Render contextually aware widgets based on active app and entity
 * @param {Object} ctx
 */
function updateSidecarContent(ctx) {
    if (!sidecarContentEl) return;
    sidecarContentEl.replaceChildren();

    const focusedApp = ctx.focusedApp || 'general';

    // 1. Context Badge Card
    const ctxCard = document.createElement('article');
    ctxCard.style.cssText = 'background:rgba(0,240,255,0.06); border:1px solid rgba(0,240,255,0.2); border-radius:8px; padding:10px; display:flex; flex-direction:column; gap:4px;';
    
    const kicker = document.createElement('small');
    kicker.style.cssText = 'font-size:9px; color:var(--vgt-cyan); letter-spacing:0.05em; font-weight:bold;';
    kicker.textContent = 'AKTIVER KONTEXT';

    const focusTitle = document.createElement('strong');
    focusTitle.style.cssText = 'font-size:12px; color:#fff;';
    focusTitle.textContent = getAppTitle(focusedApp);

    ctxCard.append(kicker, focusTitle);
    sidecarContentEl.appendChild(ctxCard);

    // 2. Contextual Quick Actions based on focused app
    const actionsCard = document.createElement('article');
    actionsCard.style.cssText = 'display:flex; flex-direction:column; gap:6px;';

    const actionsHeader = document.createElement('span');
    actionsHeader.style.cssText = 'font-size:10px; color:var(--vgt-text-dim); text-transform:uppercase;';
    actionsHeader.textContent = 'Kontextuelle KI-Aktionen';
    actionsCard.appendChild(actionsHeader);

    const prompts = getContextualPrompts(focusedApp, ctx);
    prompts.forEach(p => {
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'cyber-button';
        btn.style.cssText = 'font-size:10px; padding:6px 10px; text-align:left; background:rgba(255,255,255,0.03); border:1px solid rgba(0,240,255,0.2); color:#fff; border-radius:6px; cursor:pointer; width:100%; display:flex; align-items:center; gap:6px;';
        
        const symbol = document.createElement('span');
        symbol.style.color = 'var(--vgt-cyan)';
        symbol.textContent = '⚡';

        const label = document.createElement('span');
        label.textContent = p.label;

        btn.append(symbol, label);

        btn.addEventListener('click', async () => {
            btn.disabled = true;
            const sharedInput = document.getElementById('user-input');
            if (sharedInput) {
                sharedInput.value = p.prompt;
                await sendMessage();
            }
            btn.disabled = false;
        });

        actionsCard.appendChild(btn);
    });

    sidecarContentEl.appendChild(actionsCard);

    // 3. Global Watch Live Feed Bridge widget
    const gwCard = document.createElement('article');
    gwCard.style.cssText = 'background:rgba(255,123,0,0.06); border:1px solid rgba(255,123,0,0.25); border-radius:8px; padding:10px; display:flex; flex-direction:column; gap:6px; margin-top:auto;';
    
    const gwHeader = document.createElement('div');
    gwHeader.style.cssText = 'display:flex; justify-content:space-between; align-items:center;';

    const gwKicker = document.createElement('small');
    gwKicker.style.cssText = 'font-size:9px; color:var(--vgt-orange); font-weight:bold;';
    gwKicker.textContent = 'GLOBAL WATCH // SENTINEL';

    const gwStatus = document.createElement('span');
    gwStatus.style.cssText = 'font-size:8px; background:rgba(57,255,20,0.15); color:var(--vgt-green); padding:1px 4px; border-radius:3px;';
    gwStatus.textContent = 'LIVE ●';

    gwHeader.append(gwKicker, gwStatus);

    const gwText = document.createElement('p');
    gwText.style.cssText = 'margin:0; font-size:10px; color:rgba(255,255,255,0.8); line-height:1.4;';
    gwText.textContent = 'Laufende Überwachung aktiver Reisedestinationen und Risikosignale aktiv.';

    gwCard.append(gwHeader, gwText);
    sidecarContentEl.appendChild(gwCard);
}

function getAppTitle(appID) {
    const titles = {
        writer: 'AETHEL WRITER (Dokument)',
        browser: 'AETHEL BROWSER (Webseite)',
        travel: 'TRIP PLANNER (Reise & Lage)',
        planner: 'MASTER PLANNER (Ziele)',
        research: 'RESEARCH DESK (Evidence)',
        files: 'VAULT EXPLORER (Dateien)',
        dashboard: 'PERSONAL DASHBOARD',
        mail: 'SPHERE MAIL',
        market: 'MARKET ENGINE'
    };
    return titles[appID] || 'AETHEL SPHERE DESKTOP';
}

function getContextualPrompts(appID, ctx) {
    switch (appID) {
    case 'writer':
        return [
            { label: 'Schlage Verbesserungen vor (Track Changes)', prompt: 'Lies das aktuelle Writer-Dokument und schlage Verbesserungen als Track-Changes-Diff vor.' },
            { label: 'Erstelle Executive Summary', prompt: 'Fasse den Inhalt des Writer-Dokuments in drei prägnanten Kernaussagen zusammen.' },
            { label: 'Inhalt als Forschungs-Evidence erfassen', prompt: 'Extrahiere die wichtigsten Thesen des Dokuments in den Research Desk.' }
        ];
    case 'browser':
        return [
            { label: 'Seite zusammenfassen & in Writer übernehmen', prompt: 'Lies den aktuellen Browserinhalt und übertrage die Kernaussagen in den Writer.' },
            { label: 'Forschungszitat mit URL sichern', prompt: 'Sichere das wesentliche Zitat dieser Seite mit Quellenangabe im Research Desk.' },
            { label: 'Reiseoptionen auf Seite prüfen', prompt: 'Prüfe ob diese Seite passende Hotels oder Flüge für aktive Reisen enthält.' }
        ];
    case 'travel':
        return [
            { label: 'Prüfe Global Watch Risikolage', prompt: 'Gleiche alle Stationen meiner aktiven Reise mit aktuellen Global Watch Risiken ab.' },
            { label: 'Optimiere Tagesablauf (Itinerary)', prompt: 'Schlage eine optimierte Zeiteinteilung für die Reisetage vor.' },
            { label: 'Reisebudget analysieren', prompt: 'Prüfe das Reisebudget und liste Einsparpotenziale auf.' }
        ];
    case 'planner':
        return [
            { label: 'Erstelle Meilenstein-Checkliste', prompt: 'Strukturiere das aktuelle Ziel in Meilensteine und Aufgaben mit Abhängigkeiten.' },
            { label: 'Blocker & Risiken identifizieren', prompt: 'Analysiere offene Aufgaben und identifiziere Abhängigkeits-Blocker.' }
        ];
    case 'research':
        return [
            { label: 'Generiere umfassendes Briefing', prompt: 'Synthetisiere alle gesammelten Recherche-Auszüge zu einem Markdown-Lagebericht.' },
            { label: 'Widersprüche in Quellen prüfen', prompt: 'Analysiere gesammelte Quellen auf Widersprüche und Zuverlässigkeit.' }
        ];
    default:
        return [
            { label: 'Heutigen Tagesüberblick erstellen', prompt: 'Erstelle eine strukturierte Tagesübersicht mit Aufgaben, Terminen und Marktlage.' },
            { label: 'Global Watch Lagebericht anfordern', prompt: 'Erstelle ein kurzes weltweites Lagebriefing der wichtigsten Signale.' }
        ];
    }
}
