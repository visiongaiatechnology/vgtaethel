// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/apps/live.js
// Purpose: Live Flow Timeline (Agent Turns, Traces & Steps)

import { state } from '../../state.js';

let livePollTimer = null;

export function renderLiveApp(body) {
    if (!body) return;
    body.replaceChildren();

    const container = document.createElement('div');
    container.className = 'font-mono';
    container.style.cssText = 'display:flex; flex-direction:column; gap:8px; height:100%; color:#fff;';

    const kicker = document.createElement('div');
    kicker.style.cssText = 'font-size:10px; color:var(--vgt-purple); letter-spacing:0.05em; font-weight:bold;';
    kicker.textContent = 'LIVE RUN // PLAN, TOOLS & EVIDENCE TIMELINE';

    const status = document.createElement('div');
    status.style.cssText = 'font-size:11px; color:var(--vgt-cyan); padding:4px 0; border-bottom:1px solid rgba(255,255,255,0.06);';

    const timeline = document.createElement('div');
    timeline.style.cssText = 'flex:1; overflow-y:auto; display:flex; flex-direction:column; gap:6px; padding-right:4px;';

    container.append(kicker, status, timeline);
    body.appendChild(container);

    const render = async () => {
        try {
            const response = await fetch(`${state.API_BASE}/v1/runs`);
            if (!response.ok) throw new Error(`Run Center ${response.status}`);
            const payload = await response.json();
            const runs = payload.runs || [];
            const run = runs.find(item => ['running', 'waiting_approval', 'paused'].includes(item.status)) || runs[0];
            timeline.replaceChildren();

            if (!run) {
                status.textContent = 'Kein aktiver Run. Starte eine Aufgabe im Chat oder Planner.';
                return;
            }
            status.textContent = `${String(run.status || 'unknown').toUpperCase()} · ${run.objective || 'Ohne Ziel'} · Turn ${run.agent_turn || 0}/${run.max_agent_turns || 0}`;

            const entries = [
                ...(run.trace || []).slice(-10).map(trace => ({ kind: 'trace', label: trace.event, detail: trace.detail, at: trace.timestamp })),
                ...(run.steps || []).slice(-8).map(step => ({ kind: 'step', label: `${step.kind || 'step'} · ${step.status || 'pending'}`, detail: step.title || step.result || step.error || '', at: step.finished_at || step.started_at }))
            ];
            entries.sort((a, b) => String(a.at || '').localeCompare(String(b.at || '')));

            for (const entry of entries.slice(-14)) {
                const row = document.createElement('article');
                row.style.cssText = 'padding:6px 10px; background:rgba(0,0,0,0.3); border:1px solid rgba(255,255,255,0.05); border-radius:4px; font-size:10px; display:flex; flex-direction:column; gap:2px;';

                const label = document.createElement('strong');
                label.style.color = entry.kind === 'step' ? 'var(--vgt-cyan)' : 'var(--vgt-purple)';
                label.textContent = entry.label || 'UPDATE';

                const detail = document.createElement('span');
                detail.style.color = 'rgba(255,255,255,0.8)';
                detail.textContent = entry.detail || 'Status aktualisiert.';

                row.append(label, detail);
                timeline.append(row);
            }
        } catch (error) {
            status.textContent = `Live Flow nicht verfügbar: ${error.message}`;
        }
    };

    void render();
    if (livePollTimer) clearInterval(livePollTimer);
    livePollTimer = setInterval(() => {
        const windowEl = body.closest('.sphere-window');
        if (!windowEl || windowEl.classList.contains('hidden')) return;
        void render();
    }, 2200);
}
