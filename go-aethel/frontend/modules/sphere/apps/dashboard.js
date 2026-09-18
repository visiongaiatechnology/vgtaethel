// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/apps/dashboard.js
// Purpose: Personal Operations Dashboard with Daily Pulse, Active Trips, Tasks & Market Telemetry

import { state } from '../../state.js';

/**
 * Render Personal Operations Dashboard inside a window body
 * @param {HTMLElement} body
 */
export async function renderDashboardApp(body) {
    if (!body) return;
    body.replaceChildren();

    const container = document.createElement('div');
    container.className = 'sphere-dashboard-container font-mono';
    container.style.cssText = 'display:flex; flex-direction:column; height:100%; gap:12px; overflow-y:auto; padding-right:4px; color:#fff;';

    // 1. Welcome & Daily Greeting
    const greetingCard = document.createElement('article');
    greetingCard.className = 'glass-card';
    greetingCard.style.cssText = 'background:linear-gradient(135deg, rgba(0,240,255,0.08), rgba(0,0,0,0.5)); border:1px solid var(--vgt-cyan); border-radius:10px; padding:16px; display:flex; justify-content:space-between; align-items:center;';

    const greetText = document.createElement('div');
    const now = new Date();
    const timeStr = now.toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit' });
    const dateStr = now.toLocaleDateString('de-DE', { weekday: 'long', day: '2-digit', month: 'long', year: 'numeric' });

    greetText.innerHTML = `
        <span style="font-size:10px; color:var(--vgt-cyan); letter-spacing:0.05em; font-weight:bold;">AETHEL // PERSONAL OPERATIONS</span>
        <h2 style="margin:2px 0 0; font-size:18px; color:#fff;">Guten Tag, Operator.</h2>
        <small style="color:var(--vgt-text-dim); font-size:11px;">${dateStr} · ${timeStr} · Alle Systeme gesichert (Kernel 2.1)</small>
    `;

    const statusBadge = document.createElement('div');
    statusBadge.style.cssText = 'text-align:right; font-size:11px;';
    statusBadge.innerHTML = `
        <span style="color:var(--vgt-green); font-weight:bold; display:flex; align-items:center; gap:4px;">● OPERATIONAL</span>
        <small style="color:var(--vgt-text-dim);">Zero-Trust Active</small>
    `;

    greetingCard.append(greetText, statusBadge);
    container.appendChild(greetingCard);

    // 2. Metrics Grid (Trips, Tasks, Market, Weather)
    const grid = document.createElement('div');
    grid.style.cssText = 'display:grid; grid-template-columns:1fr 1fr; gap:10px;';

    // Card A: Active Trip
    const tripCard = document.createElement('article');
    tripCard.className = 'glass-card';
    tripCard.style.cssText = 'background:rgba(255,123,0,0.06); border:1px solid rgba(255,123,0,0.25); border-radius:8px; padding:12px; display:flex; flex-direction:column; gap:6px; cursor:pointer;';
    tripCard.innerHTML = `
        <div style="display:flex; justify-content:space-between; align-items:center;">
            <strong style="font-size:11px; color:var(--vgt-orange);">🗺 AKTIVE REISE</strong>
            <span style="font-size:8px; background:rgba(57,255,20,0.15); color:var(--vgt-green); padding:1px 4px; border-radius:3px;">LAGE: NORMAL ●</span>
        </div>
        <b style="font-size:13px; color:#fff;">Reise Japan 2027</b>
        <small style="color:var(--vgt-text-dim); font-size:10px;">Tokyo → Kyoto → Osaka · 12 Tage · €2.500 Budget</small>
    `;
    tripCard.addEventListener('click', () => window.openSphereApp('travel'));

    // Card B: Master Planner
    const planCard = document.createElement('article');
    planCard.className = 'glass-card';
    planCard.style.cssText = 'background:rgba(0,240,255,0.06); border:1px solid rgba(0,240,255,0.25); border-radius:8px; padding:12px; display:flex; flex-direction:column; gap:6px; cursor:pointer;';
    planCard.innerHTML = `
        <div style="display:flex; justify-content:space-between; align-items:center;">
            <strong style="font-size:11px; color:var(--vgt-cyan);">⏰ MASTER PLANNER</strong>
            <span style="font-size:8px; background:rgba(0,240,255,0.15); color:var(--vgt-cyan); padding:1px 4px; border-radius:3px;">60% ERREICHT</span>
        </div>
        <b style="font-size:13px; color:#fff;">Aethel Beta 4 Release</b>
        <small style="color:var(--vgt-text-dim); font-size:10px;">3 Meilensteine aktiv · 0 Blocker</small>
    `;
    planCard.addEventListener('click', () => window.openSphereApp('planner'));

    // Card C: Market Pulse
    const marketCard = document.createElement('article');
    marketCard.className = 'glass-card';
    marketCard.style.cssText = 'background:rgba(255,255,255,0.02); border:1px solid rgba(255,255,255,0.08); border-radius:8px; padding:12px; display:flex; flex-direction:column; gap:6px; cursor:pointer;';
    marketCard.innerHTML = `
        <div style="display:flex; justify-content:space-between; align-items:center;">
            <strong style="font-size:11px; color:var(--vgt-cyan);">◇ MÄRKTE // PULSE</strong>
            <small style="color:var(--vgt-text-dim); font-size:9px;">MULTI-ASSET</small>
        </div>
        <div style="display:flex; justify-content:space-between; font-size:12px;">
            <span>BTC/USD: <b style="color:var(--vgt-green);">$94.200</b></span>
            <span>Brent Öl: <b style="color:var(--vgt-cyan);">$78.40</b></span>
        </div>
    `;
    marketCard.addEventListener('click', () => window.openSphereApp('market'));

    // Card D: Weather Satellite
    const weatherCard = document.createElement('article');
    weatherCard.className = 'glass-card';
    weatherCard.style.cssText = 'background:rgba(255,255,255,0.02); border:1px solid rgba(255,255,255,0.08); border-radius:8px; padding:12px; display:flex; flex-direction:column; gap:6px; cursor:pointer;';
    weatherCard.innerHTML = `
        <div style="display:flex; justify-content:space-between; align-items:center;">
            <strong style="font-size:11px; color:var(--vgt-cyan);">☼ WETTER // SATELLIT</strong>
            <small style="color:var(--vgt-text-dim); font-size:9px;">KÖLN</small>
        </div>
        <div style="display:flex; justify-content:space-between; font-size:12px;">
            <span>Köln: <b style="color:#fff;">18.5 °C</b></span>
            <span>Tokio: <b style="color:var(--vgt-orange);">22.0 °C</b></span>
        </div>
    `;
    weatherCard.addEventListener('click', () => window.openSphereApp('weather'));

    grid.append(tripCard, planCard, marketCard, weatherCard);
    container.appendChild(grid);

    body.appendChild(container);
}
