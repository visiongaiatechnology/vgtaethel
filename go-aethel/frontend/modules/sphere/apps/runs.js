// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/apps/runs.js
// Purpose: Persistent Agent Runs Monitoring in Sphere Run Desk

import { state } from '../../state.js';

export async function renderRunsApp(body) {
    if (!body) return;
    body.replaceChildren();

    const container = document.createElement('div');
    container.className = 'font-mono';
    container.style.cssText = 'display:flex; flex-direction:column; gap:8px; height:100%; color:#fff;';

    const kicker = document.createElement('div');
    kicker.style.cssText = 'font-size:10px; color:var(--vgt-purple); letter-spacing:0.05em; font-weight:bold;';
    kicker.textContent = 'PERSISTENTE AGENT RUNS // RUN DESK';

    const list = document.createElement('div');
    list.style.cssText = 'flex:1; overflow-y:auto; display:flex; flex-direction:column; gap:6px; padding-right:4px;';

    container.append(kicker, list);
    body.appendChild(container);

    try {
        const response = await fetch(`${state.API_BASE}/v1/runs`);
        if (!response.ok) throw new Error(`Run Center ${response.status}`);
        const payload = await response.json();
        const runs = (payload.runs || []).slice(0, 10);

        if (runs.length === 0) {
            const empty = document.createElement('span');
            empty.style.cssText = 'font-size:11px; color:var(--vgt-text-dim); padding:10px;';
            empty.textContent = 'Keine aktiven oder gespeicherten Runs.';
            list.append(empty);
            return;
        }

        for (const run of runs) {
            const row = document.createElement('div');
            row.style.cssText = 'display:flex; justify-content:space-between; align-items:center; padding:8px 10px; background:rgba(0,0,0,0.3); border:1px solid rgba(255,255,255,0.06); border-radius:6px; font-size:11px;';

            const objective = document.createElement('span');
            objective.style.cssText = 'color:#fff; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; max-width:280px;';
            objective.textContent = run.objective || 'Ohne Ziel';

            const status = document.createElement('strong');
            status.textContent = String(run.status || 'unknown').toUpperCase();
            status.style.cssText = `font-size:9px; padding:2px 6px; border-radius:4px; ${run.status === 'completed' ? 'color:var(--vgt-green); background:rgba(57,255,20,0.1);' : 'color:var(--vgt-cyan); background:rgba(0,240,255,0.1);'}`;

            row.append(objective, status);
            list.append(row);
        }
    } catch (error) {
        const failure = document.createElement('span');
        failure.style.cssText = 'font-size:11px; color:var(--vgt-red); padding:10px;';
        failure.textContent = `Run Desk nicht verfügbar: ${error.message}`;
        list.append(failure);
    }
}
