import { state } from './state.js';
import * as api from './api.js';
import { requestRunApproval } from './approval_dialog.js';

function statusClass(status) {
    if (status === 'running') return 'status-running';
    if (status === 'paused' || status === 'waiting_approval') return 'status-blocked';
    if (status === 'failed' || status === 'cancelled') return 'status-rejected';
    if (status === 'queued') return 'status-pending';
    return 'status-completed';
}

function makeButton(label, action, runID) {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = `cyber-button font-mono run-action-button run-action-${action}`;
    button.textContent = label;
    button.addEventListener('click', async () => {
        button.disabled = true;
        try {
            await api.runAction(runID, action);
            await fetchTasksQueue();
        } catch (error) {
            window.showAethelToast?.(`Run-Aktion fehlgeschlagen: ${error.message}`, 'error');
        } finally {
            button.disabled = false;
        }
    });
    return button;
}

function makeApprovalButton(runID) {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'cyber-button font-mono run-action-button run-action-approval';
    button.textContent = 'AKTION PRÜFEN';
    button.addEventListener('click', async () => {
        button.disabled = true;
        try {
            const challenge = await api.runAction(runID, 'approval', {});
            const approved = await requestRunApproval(challenge);
            if (!approved) return;
            await api.runAction(runID, 'approval', { approval_token: challenge.approval_token });
            await fetchTasksQueue();
        } catch (error) {
            window.showAethelToast?.(`Freigabe fehlgeschlagen: ${error.message}`, 'error');
        } finally {
            button.disabled = false;
        }
    });
    return button;
}

function buildRunCard(run) {
    const card = document.createElement('article');
    card.className = 'glass-card run-center-card';

    const header = document.createElement('div');
    header.className = 'run-center-header';
    const identity = document.createElement('div');
    const title = document.createElement('strong');
    title.textContent = run.objective || 'Untitled run';
    const meta = document.createElement('div');
    meta.className = 'run-center-meta';
    meta.textContent = `${run.id} · ${run.profile_id} · ${run.model_id || 'model auto'}`;
    identity.append(title, meta);
    const badge = document.createElement('span');
    badge.className = `tool-status-badge ${statusClass(run.status)}`;
    badge.textContent = String(run.status || 'unknown').toUpperCase();
    header.append(identity, badge);

    const verified = Array.isArray(run.steps) ? run.steps.filter(step => step.status === 'verified').length : 0;
    const total = Array.isArray(run.steps) ? run.steps.length : 0;
    const metrics = document.createElement('div');
    metrics.className = 'run-center-metrics';
    const metricValues = [
        ['FORTSCHRITT', `${verified}/${total}`],
        ['TOOLS', String(run.tool_calls || 0)],
        ['KOSTEN', `$${Number(run.spent_usd || 0).toFixed(4)} / $${Number(run.cost_budget_usd || 0).toFixed(2)}`],
        ['AKTUALISIERT', run.updated_at ? new Date(run.updated_at).toLocaleTimeString() : '—']
    ];
    for (const [label, value] of metricValues) {
        const metric = document.createElement('div');
        const metricLabel = document.createElement('span');
        metricLabel.textContent = label;
        const metricValue = document.createElement('strong');
        metricValue.textContent = value;
        metric.append(metricLabel, metricValue);
        metrics.appendChild(metric);
    }

	const continuity = document.createElement('section');
	continuity.className = 'run-continuity';
	const continuityHeader = document.createElement('div');
	continuityHeader.className = 'run-continuity-header';
	const continuityTitle = document.createElement('strong');
	continuityTitle.textContent = 'COGNITIVE CONTINUITY';
	const continuityPhase = document.createElement('span');
	const cognitiveState = run.continuity || {};
	continuityPhase.textContent = `${String(cognitiveState.tier || 'legacy').toUpperCase()} · ${String(cognitiveState.phase || run.status || 'unknown').toUpperCase()}`;
	continuityHeader.append(continuityTitle, continuityPhase);

	const tokenBudget = cognitiveState.token_budget || {};
	const usedTokens = Number(tokenBudget.input_tokens || 0) + Number(tokenBudget.output_tokens || 0);
	const maxTokens = Number(tokenBudget.max_total_tokens || 0);
	const tokenTrack = document.createElement('div');
	tokenTrack.className = 'run-token-track';
	const tokenFill = document.createElement('span');
	const tokenPercent = maxTokens > 0 ? Math.min(100, (usedTokens / maxTokens) * 100) : 0;
	tokenFill.style.width = `${tokenPercent.toFixed(2)}%`;
	tokenTrack.appendChild(tokenFill);

	const continuityMeta = document.createElement('div');
	continuityMeta.className = 'run-continuity-meta';
	const expectedEffects = Array.isArray(cognitiveState.expected_effects) ? cognitiveState.expected_effects : [];
	const evidence = Array.isArray(cognitiveState.evidence) ? cognitiveState.evidence : [];
	continuityMeta.textContent = `TOKENS ${usedTokens.toLocaleString()} / ${maxTokens.toLocaleString()} · EVIDENZ ${evidence.filter(item => item.verified).length}/${evidence.length} · EFFEKTE ${expectedEffects.length}`;
	continuity.append(continuityHeader, tokenTrack, continuityMeta);
	if (cognitiveState.reflection) {
		const reflection = document.createElement('p');
		reflection.className = 'run-continuity-reflection';
		reflection.textContent = cognitiveState.reflection;
		continuity.appendChild(reflection);
	}

    const steps = document.createElement('div');
    steps.className = 'run-center-steps';
    for (const step of run.steps || []) {
        const row = document.createElement('div');
        row.className = `run-step run-step-${step.status || 'pending'}`;
        const marker = document.createElement('span');
        marker.className = 'run-step-marker';
        marker.textContent = step.status === 'verified' ? '✓' : step.status === 'running' ? '›' : step.status === 'failed' ? '×' : '·';
        const text = document.createElement('span');
        text.textContent = step.title || step.kind;
        row.append(marker, text);
        if (step.evidence_after) {
            const evidence = document.createElement('code');
            evidence.className = 'run-step-evidence';
            evidence.textContent = step.evidence_changed ? 'VISUAL CHANGE VERIFIED' : 'VISUAL EVIDENCE CAPTURED';
            evidence.title = `Vorher: ${step.evidence_before || '—'}\nNachher: ${step.evidence_after}`;
            row.appendChild(evidence);
        }
        steps.appendChild(row);
    }

    const trace = document.createElement('details');
    trace.className = 'run-center-trace';
    const summary = document.createElement('summary');
    summary.textContent = `TRACE · ${(run.trace || []).length} EVENTS`;
    trace.appendChild(summary);
    for (const event of (run.trace || []).slice(-12).reverse()) {
        const line = document.createElement('div');
        line.className = 'run-trace-line';
        const stamp = event.timestamp ? new Date(event.timestamp).toLocaleTimeString() : '--:--';
        line.textContent = `[${stamp}] ${String(event.event || '').toUpperCase()} · ${event.detail || ''}`;
        trace.appendChild(line);
    }

    const actions = document.createElement('div');
    actions.className = 'run-center-actions';
    if (run.status === 'queued') actions.appendChild(makeButton('START', 'start', run.id));
    if (run.status === 'running') {
        actions.append(makeButton('NÄCHSTER SCHRITT', 'advance', run.id), makeButton('PAUSE', 'pause', run.id), makeButton('ABBRECHEN', 'cancel', run.id));
    }
    if (run.status === 'paused') actions.append(makeButton('FORTSETZEN', 'resume', run.id), makeButton('ABBRECHEN', 'cancel', run.id));
    if (run.status === 'waiting_approval') {
        const waiting = document.createElement('span');
        waiting.className = 'run-waiting-label';
        waiting.textContent = `FREIGABE AUSSTEHEND · ${run.approval_step_id || 'TOOL'}`;
        actions.append(waiting, makeApprovalButton(run.id), makeButton('PAUSE', 'pause', run.id), makeButton('ABBRECHEN', 'cancel', run.id));
    }

    card.append(header, metrics, continuity, steps, trace, actions);
    return card;
}

export async function fetchTasksQueue() {
    const container = document.getElementById('tasks-list-container');
    if (!container) return;
    try {
        const payload = await api.getRuns();
        const runs = Array.isArray(payload.runs) ? payload.runs : [];
        container.replaceChildren();
        if (runs.length === 0) {
            const empty = document.createElement('div');
            empty.className = 'run-center-empty';
            empty.textContent = 'Noch keine persistenten Agent Runs vorhanden.';
            container.appendChild(empty);
        } else {
            container.append(...runs.map(buildRunCard));
        }
    } catch (error) {
        container.replaceChildren();
        const failure = document.createElement('div');
        failure.className = 'run-center-empty';
        failure.textContent = `Run Center nicht erreichbar: ${error.message}`;
        container.appendChild(failure);
    }
    await fetchArtifacts();
}

export async function fetchArtifacts() {
    const container = document.getElementById('artifacts-list-container');
    if (!container) return;
    try {
        const payload = await api.getArtifacts();
        const artifacts = Array.isArray(payload.artifacts) ? payload.artifacts : [];
        container.replaceChildren();
        if (artifacts.length === 0) {
            const empty = document.createElement('div');
            empty.className = 'run-center-empty';
            empty.textContent = 'Keine Datei-Snapshots (Artefakte) vorhanden.';
            container.appendChild(empty);
            return;
        }
        for (const art of artifacts) {
            const card = document.createElement('div');
            card.className = 'artifact-card';

            const info = document.createElement('div');
            info.className = 'artifact-info';
            
            const fileTitle = document.createElement('strong');
            fileTitle.textContent = art.path.split(/[/\\]/).pop() || art.path;
            fileTitle.className = 'artifact-title';
            
            const filePath = document.createElement('span');
            filePath.textContent = art.path;
            filePath.className = 'artifact-path';
            
            const meta = document.createElement('span');
            meta.className = 'artifact-meta';
            const dateStr = art.created_at ? new Date(art.created_at).toLocaleString() : '—';
            meta.textContent = `${art.size_bytes} Bytes · ${dateStr} · ID: ${art.id}`;

            info.append(fileTitle, filePath, meta);

            const btnRestore = document.createElement('button');
            btnRestore.type = 'button';
            btnRestore.className = 'cyber-button font-mono artifact-restore-button';
            btnRestore.textContent = 'UNDO';
            btnRestore.addEventListener('click', async () => {
                btnRestore.disabled = true;
                const confirm = window.confirm(`Datei '${fileTitle.textContent}' wirklich auf den Snapshot vom ${dateStr} zurücksetzen?`);
                if (!confirm) {
                    btnRestore.disabled = false;
                    return;
                }
                try {
                    const res = await fetch(`${state.API_BASE}/v1/tools/execute`, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                            name: "fs_restore_snapshot",
                            args: { snapshot_id: art.id }
                        })
                    });
                    if (!res.ok) throw new Error(await res.text());
                    const resData = await res.json();
                    if (resData.status === "success") {
                        window.showAethelToast?.('Datei erfolgreich auf Snapshot zurückgesetzt.', 'success');
                        await fetchArtifacts();
                    } else {
                        throw new Error(resData.error || 'Fehler beim Zurücksetzen.');
                    }
                } catch (e) {
                    window.showAethelToast?.(`Undo fehlgeschlagen: ${e.message}`, 'error');
                } finally {
                    btnRestore.disabled = false;
                }
            });

            card.append(info, btnRestore);
            container.appendChild(card);
        }
    } catch (e) {
        console.error('Failed to fetch artifacts', e);
    }
}

export async function addTaskItem() {
    const titleInput = document.getElementById('task-add-text');
    const objectiveInput = document.getElementById('task-add-objective');
    const profileInput = document.getElementById('task-add-schedule');
    const budgetInput = document.getElementById('task-add-interval');
    const modelInput = document.getElementById('task-add-capabilities');
    const startInput = document.getElementById('task-add-risk');
    const title = titleInput?.value.trim() || '';
    const objective = objectiveInput?.value.trim() || title;
    if (!objective) {
        window.showAethelToast?.('Ein Run benötigt ein konkretes Ziel.', 'error');
        return;
    }
    try {
        const run = await api.createRun({
            objective: title && title !== objective ? `${title}: ${objective}` : objective,
            profile_id: profileInput?.value || 'developer',
            model_id: modelInput?.value.trim() || state.currentModel,
            mode: 'chat_agent',
            system_prompt: 'Arbeite als persistenter Aethel-Agent. Plane verifizierbare Schritte, nutze nur zulässige Tools, warte auf erforderliche Freigaben und liefere einen überprüfbaren Abschlussbericht.',
            agent_messages: [{ role: 'user', content: objective }],
            max_agent_turns: 24,
            cost_budget_usd: Number(budgetInput?.value || 2),
            steps: [
                { kind: 'plan', title: 'Ziel und Arbeitsplan validieren' },
                { kind: 'report', title: 'Verifizierten Abschlussbericht erzeugen' }
            ]
        });
        if (startInput?.value === 'start') await api.runAction(run.id, 'start');
        if (titleInput) titleInput.value = '';
        if (objectiveInput) objectiveInput.value = '';
        await fetchTasksQueue();
        await fetchKernelTasks();
    } catch (error) {
        window.showAethelToast?.(`Run konnte nicht erstellt werden: ${error.message}`, 'error');
    }
}

export async function fetchKernelTasks() {
    const container = document.getElementById('task-checklist-container');
    if (!container) return;
    try {
        const payload = await api.getRuns();
        const active = (payload.runs || []).filter(run => !['completed', 'failed', 'cancelled'].includes(run.status)).slice(0, 5);
        container.replaceChildren();
        if (active.length === 0) {
            const empty = document.createElement('div');
            empty.className = 'run-center-empty compact';
            empty.textContent = 'Keine aktiven Runs.';
            container.appendChild(empty);
            return;
        }
        for (const run of active) {
            const item = document.createElement('div');
            item.className = 'task-item';
            const dot = document.createElement('div');
            dot.className = 'task-checkbox font-mono';
            const label = document.createElement('span');
            label.className = 'task-item-label';
            label.textContent = `${run.status.toUpperCase()} · ${run.objective}`;
            item.append(dot, label);
            container.appendChild(item);
        }
    } catch (error) {
        console.error('Failed to fetch runs', error);
    }
}

export function setupTasksUIEvents() {
    document.getElementById('task-btn-add')?.addEventListener('click', addTaskItem);
}
