// STATUS: DIAMANT VGT SUPREME
import { state } from './state.js';

const TERMINAL = new Set(['COMPLETED', 'FAILED', 'CANCELLED']);
const codeState = {
    initialized: false,
    project: null,
    activePath: '',
    originalContent: '',
    activeSessionID: '',
    currentSession: null,
    pollGeneration: 0,
    pollRunning: false,
    conversationSequence: 0,
    completedReportedSession: '',
    scope: 'AGENT',
    diffMode: 'unified',
    activeDiffPath: '',
    activeDiff: null,
    displayedSessionID: '',
};

function element(id) { return document.getElementById(id); }

async function decodeResponse(response, label) {
    const type = response.headers.get('content-type') || '';
    const payload = type.includes('application/json') ? await response.json() : { message: (await response.text()).slice(0, 300) };
    if (!response.ok) throw new Error(payload.message || `${label} failed (${response.status})`);
    return payload;
}

export function initVGTCode() {
    if (codeState.initialized) return;
    codeState.initialized = true;
    element('vgt-code-refresh')?.addEventListener('click', refreshVGTCode);
    element('vgt-code-save')?.addEventListener('click', saveActiveFile);
    element('vgt-code-run')?.addEventListener('click', startCodeRun);
    element('vgt-code-stop')?.addEventListener('click', stopCodeRun);
    element('vgt-code-exit-focus')?.addEventListener('click', () => window.switchMode?.('core'));
    element('vgt-code-open-runs')?.addEventListener('click', () => window.switchMode?.('tasks'));
    element('vgt-code-open-project')?.addEventListener('click', openCodeProject);
    element('vgt-code-empty-open')?.addEventListener('click', openCodeProject);
    element('vgt-code-close-project')?.addEventListener('click', closeCodeProject);
    element('vgt-code-new-conversation')?.addEventListener('click', startNewCodeConversation);
    element('vgt-code-mode-toggle')?.addEventListener('click', toggleSecurityMode);
    element('vgt-code-tabs')?.addEventListener('click', event => {
        const button = event.target.closest('[data-code-tab]');
        if (button) switchCodeTab(button.dataset.codeTab);
    });
    element('vgt-code-scope')?.addEventListener('click', event => {
        const button = event.target.closest('[data-scope]');
        if (!button) return;
        codeState.scope = button.dataset.scope;
        document.querySelectorAll('#vgt-code-scope [data-scope]').forEach(item => item.classList.toggle('active', item === button));
        renderChanges(codeState.currentSession);
    });
    element('vgt-code-worker-model')?.addEventListener('change', event => localStorage.setItem('aethel_coder_worker_model', event.target.value));
    element('vgt-code-orchestrator-model')?.addEventListener('change', event => localStorage.setItem('aethel_coder_orchestrator_model', event.target.value));
    element('vgt-code-diff-unified')?.addEventListener('click', () => setDiffMode('unified'));
    element('vgt-code-diff-split')?.addEventListener('click', () => setDiffMode('split'));
    element('vgt-code-diff-prev')?.addEventListener('click', () => navigateDiff(-1));
    element('vgt-code-diff-next')?.addEventListener('click', () => navigateDiff(1));
    element('vgt-code-diff-open')?.addEventListener('click', () => openCodeFile(codeState.activeDiffPath));
    element('vgt-code-diff-ask')?.addEventListener('click', askCoderAboutDiff);
    const editor = element('vgt-code-content');
    editor?.addEventListener('input', () => { syncEditorMetrics(); updateDirtyState(); });
    editor?.addEventListener('scroll', () => { if (element('vgt-code-lines')) element('vgt-code-lines').scrollTop = editor.scrollTop; });
    editor?.addEventListener('click', syncEditorMetrics);
    editor?.addEventListener('keyup', syncEditorMetrics);
    editor?.addEventListener('keydown', event => {
        if (event.key === 'Tab') {
            event.preventDefault();
            editor.setRangeText('    ', editor.selectionStart, editor.selectionEnd, 'end');
            editor.dispatchEvent(new Event('input'));
        }
        if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
            event.preventDefault();
            saveActiveFile();
        }
    });
    element('vgt-code-objective')?.addEventListener('keydown', event => {
        if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
            event.preventDefault();
            startCodeRun();
        }
    });
    document.addEventListener('keydown', handleCoderShortcut);
    integrateAgentTeam();
    setProjectUI(false);
    hydrateCoderModels();
    loadCoderSessions();
    void refreshSecurityMode();
}

function integrateAgentTeam() {
    const team = element('view-agent');
    const main = document.querySelector('#view-code .vgt-code-main');
    if (!team || !main || team.parentElement === main) return;
    team.classList.add('vgt-code-team-pane', 'hidden');
    main.appendChild(team);
}

export function openVGTCodeTeam() {
    switchCodeTab('team');
}

export async function refreshVGTCode() {
    hydrateCoderModels();
    void refreshSecurityMode();
    await loadCoderSessions();
    if (codeState.activeSessionID) {
        await syncSession(codeState.activeSessionID, false);
        ensureSessionPolling();
    }
    const tree = element('vgt-code-tree');
    if (!tree) return;
    if (!codeState.project) {
        const message = document.createElement('p');
        message.textContent = 'Select a project to browse files.';
        tree.replaceChildren(message);
        return;
    }
    const loading = document.createElement('div');
    loading.className = 'vgt-code-tree-loading';
    loading.textContent = 'INDEXING AUTHORIZED WORKSPACE…';
    tree.replaceChildren(loading);
    try {
        const payload = await decodeResponse(await fetch(`${state.API_BASE}/v1/code/workspace/tree`, { headers: { Accept: 'application/json' } }), 'Workspace index');
        const fragment = document.createDocumentFragment();
        const count = countFiles(payload.entries || []);
        for (const entry of payload.entries || []) fragment.appendChild(renderTreeEntry(entry, 0));
        if (!count) fragment.appendChild(textBlock('vgt-code-tree-loading', 'Workspace is empty.'));
        tree.replaceChildren(fragment);
        element('vgt-code-tree-count').textContent = `${count}${payload.truncated ? '+' : ''}`;
    } catch (error) {
        tree.replaceChildren(textBlock('vgt-code-tree-error', error.message));
    }
}

function hydrateCoderModels() {
    const models = Array.isArray(state.modelRegistry) ? state.modelRegistry.filter(model => model?.id) : [];
    populateModelSelect(element('vgt-code-worker-model'), models, 'aethel_coder_worker_model', state.currentModel);
    populateModelSelect(element('vgt-code-orchestrator-model'), models.filter(model => model.supports_tools !== false), 'aethel_coder_orchestrator_model', state.orchestratorModel || state.currentModel);
}

function populateModelSelect(select, models, storageKey, fallback) {
    if (!select) return;
    const currentIDs = Array.from(select.options).map(option => option.value).filter(Boolean);
    const nextIDs = models.map(model => model.id);
    if (currentIDs.length === nextIDs.length && currentIDs.every((id, index) => id === nextIDs[index])) return;
    const remembered = localStorage.getItem(storageKey);
    const selected = nextIDs.includes(remembered) ? remembered : (nextIDs.includes(fallback) ? fallback : nextIDs[0]);
    const fragment = document.createDocumentFragment();
    for (const model of models) {
        const option = document.createElement('option');
        option.value = model.id;
        option.textContent = `${model.provider || 'Local'} · ${model.name || model.id}`;
        fragment.appendChild(option);
    }
    if (!models.length) {
        const option = document.createElement('option');
        option.disabled = true;
        option.textContent = 'No configured model';
        fragment.appendChild(option);
    }
    select.replaceChildren(fragment);
    if (selected) {
        select.value = selected;
        localStorage.setItem(storageKey, selected);
    }
}

async function openCodeProject() {
    const selector = window.go?.main?.App?.SelectCodeProject;
    if (typeof selector !== 'function') return showToast('Native project picker is unavailable.', 'error');
    try {
        const result = await selector();
        if (!result || result.status === 'cancelled') return;
        if (result.status !== 'success' || !result.path) throw new Error(result.message || 'Project selection failed.');
        codeState.project = { name: result.name || 'Project', path: result.path };
        codeState.activePath = '';
        codeState.originalContent = '';
        setProjectUI(true);
        startNewCodeConversation();
        await refreshVGTCode();
    } catch (error) { showToast(error.message, 'error'); }
}

async function closeCodeProject() {
    if (isDirty() && !window.confirm('Projekt schließen und ungespeicherte Änderungen verwerfen?')) return;
    codeState.project = null;
    codeState.activePath = '';
    const closer = window.go?.main?.App?.CloseCodeProject;
    if (typeof closer === 'function') await closer().catch(error => console.warn('Project close binding failed', error));
    setProjectUI(false);
}

function setProjectUI(active) {
    const name = active ? codeState.project.name : 'No project open';
    element('vgt-code-project-name').textContent = name;
    element('vgt-code-env-project').textContent = active ? name : 'No local project';
    element('vgt-code-access-state').textContent = active ? 'READ / WRITE' : 'NONE';
    element('vgt-code-empty').classList.toggle('hidden', active);
    element('vgt-code-chat-pane').classList.toggle('hidden', !active);
    element('vgt-code-close-project').disabled = !active;
    element('vgt-code-refresh').disabled = !active;
    element('vgt-code-run').disabled = !active;
    if (!active) {
        switchCodeTab('chat');
        element('vgt-code-active-file').textContent = 'Conversation';
        element('vgt-code-tree-count').textContent = '0';
        refreshVGTCode();
    }
}

function switchCodeTab(tab) {
    document.querySelectorAll('#vgt-code-tabs [data-code-tab]').forEach(button => button.classList.toggle('active', button.dataset.codeTab === tab));
    const hasContent = Boolean(codeState.project) || Boolean(codeState.activeSessionID) || Boolean(codeState.currentSession);
    element('vgt-code-empty').classList.toggle('hidden', hasContent || tab !== 'chat');
    element('vgt-code-chat-pane').classList.toggle('hidden', !hasContent || tab !== 'chat');
    element('vgt-code-editor-pane').classList.toggle('hidden', tab !== 'editor');
    element('vgt-code-plan-pane').classList.toggle('hidden', tab !== 'plan');
    element('vgt-code-diff-pane').classList.toggle('hidden', tab !== 'diff');
    element('view-agent')?.classList.toggle('hidden', tab !== 'team');
}

function startNewCodeConversation() {
    if (!codeState.project) return;
    codeState.conversationSequence++;
    codeState.activeSessionID = '';
    codeState.displayedSessionID = '';
    element('vgt-code-objective').value = '';
    const welcome = document.createElement('div');
    welcome.className = 'vgt-code-welcome';
    const title = document.createElement('strong');
    title.textContent = `New task in ${codeState.project.name}`;
    const detail = document.createElement('span');
    detail.textContent = 'Describe the outcome. VGT Code will inspect, implement and verify it in the background.';
    welcome.append(title, detail);
    element('vgt-code-chat-messages').replaceChildren(welcome);
    switchCodeTab('chat');
    element('vgt-code-objective').focus();
}

async function startCodeRun() {
    const input = element('vgt-code-objective');
    const request = input?.value.trim() || '';
    if (!codeState.project) return showToast('Open a project folder first.', 'error');
    if (!request) { input?.focus(); return showToast('Define an agent objective first.', 'error'); }
    if (isDirty() && !window.confirm('Die aktive Datei enthält ungespeicherte Änderungen. Agent trotzdem starten?')) return;
    const worker = element('vgt-code-worker-model')?.value || '';
    const orchestrator = element('vgt-code-orchestrator-model')?.value || '';
    if (!worker || !orchestrator) return showToast('Select both coding models first.', 'error');
    appendChatMessage('user', request);
    input.value = '';
    setRunState('STARTING');
    element('vgt-code-run').disabled = true;
    try {
        const session = await decodeResponse(await fetch(`${state.API_BASE}/v1/coder/sessions`, {
            method: 'POST', headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
            body: JSON.stringify({
                request,
                worker_model: worker,
                orchestrator_model: orchestrator,
                reasoning_effort: element('vgt-code-reasoning')?.value || 'auto',
                max_turns: Number(element('vgt-code-turns')?.value || 16),
                system_prompt: state.VGT_SYSTEM_PROTOCOL,
            }),
        }), 'Coder session');
        codeState.activeSessionID = session.session_id;
        codeState.completedReportedSession = '';
        renderSession(session);
        await loadCoderSessions();
        ensureSessionPolling(true);
    } catch (error) {
        setRunState('FAILED');
        renderReport(error.message, true);
        element('vgt-code-run').disabled = false;
    }
}

async function stopCodeRun() {
    const sessionID = codeState.activeSessionID;
    const button = element('vgt-code-stop');
    if (!sessionID || TERMINAL.has(codeState.currentSession?.status)) return;
    button.disabled = true;
    button.textContent = 'Abbruch läuft…';
    try {
        const session = await decodeResponse(await fetch(`${state.API_BASE}/v1/coder/sessions/${encodeURIComponent(sessionID)}/cancel`, {
            method: 'POST', headers: { Accept: 'application/json' },
        }), 'Coder cancel');
        renderSession(session);
        showToast('Run gestoppt. Weitere Modellschritte werden verworfen.', 'success');
    } catch (error) {
        showToast(error.message, 'error');
    } finally {
        button.disabled = false;
        button.textContent = '■ Run stoppen';
    }
}

async function loadCoderSessions() {
    try {
        const payload = await decodeResponse(await fetch(`${state.API_BASE}/v1/coder/sessions?limit=50`, { headers: { Accept: 'application/json' } }), 'Session history');
        renderSessionHistory(payload.sessions || []);
        if (!codeState.activeSessionID) {
            const active = (payload.sessions || []).find(session => !TERMINAL.has(session.status));
            if (active) {
                codeState.activeSessionID = active.session_id;
                if (!codeState.project) {
                    codeState.project = {
                        name: active.project_name || 'Workspace Project',
                        path: active.project_root || '.',
                        root: active.project_root || '.'
                    };
                    setProjectUI(true);
                }
                renderSession(active);
                ensureSessionPolling();
            }
        }
    } catch (error) { console.warn('VGT Coder history unavailable', error); }
}

function renderSessionHistory(sessions) {
    const list = element('vgt-code-conversations');
    if (!list) return;
    const fragment = document.createDocumentFragment();
    for (const session of sessions) {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'vgt-code-conversation';
        button.classList.toggle('active', session.session_id === codeState.activeSessionID);
        const title = document.createElement('strong');
        title.textContent = session.user_request || 'Coding session';
        const meta = document.createElement('small');
        meta.textContent = `${session.status} · ${session.worker_model}`;
        button.append(title, meta);
        button.addEventListener('click', async () => {
            codeState.activeSessionID = session.session_id;
            if (!codeState.project) {
                codeState.project = {
                    name: session.project_name || 'Workspace Project',
                    path: session.project_root || '.',
                    root: session.project_root || '.'
                };
                setProjectUI(true);
            }
            renderSession(session);
            switchCodeTab('chat');
            document.querySelectorAll('#vgt-code-conversations .vgt-code-conversation').forEach(b => {
                b.classList.toggle('active', b === button);
            });
            await syncSession(session.session_id, false);
            ensureSessionPolling(true);
        });
        fragment.appendChild(button);
    }
    if (!sessions.length) fragment.appendChild(textBlock('', 'No conversations yet'));
    list.replaceChildren(fragment);
}

function ensureSessionPolling(restart = false) {
    if (!codeState.activeSessionID) return;
    if (restart) codeState.pollGeneration++;
    if (codeState.pollRunning && !restart) return;
    const generation = ++codeState.pollGeneration;
    codeState.pollRunning = true;
    void pollSession(generation);
}

async function pollSession(generation) {
    while (generation === codeState.pollGeneration && codeState.activeSessionID) {
        try {
            const session = await syncSession(codeState.activeSessionID, false);
            if (!session || TERMINAL.has(session.status)) break;
        } catch (error) {
            setRunState('SYNC ERROR');
            renderReport(error.message, true);
        }
        await new Promise(resolve => window.setTimeout(resolve, document.hidden ? 2500 : 1000));
    }
    if (generation === codeState.pollGeneration) codeState.pollRunning = false;
}

async function syncSession(sessionID, focus) {
    const session = await decodeResponse(await fetch(`${state.API_BASE}/v1/coder/sessions/${encodeURIComponent(sessionID)}`, { headers: { Accept: 'application/json' } }), 'Coder session');
    renderSession(session);
    if (focus) switchCodeTab('plan');
    return session;
}

function renderSession(session) {
    codeState.currentSession = session;
	if (codeState.displayedSessionID !== session.session_id) {
		codeState.displayedSessionID = session.session_id;
		initializeSessionConversation(session);
	}
    setRunState(session.status || 'UNKNOWN');
    element('vgt-code-run-id').textContent = session.session_id || 'NO ACTIVE SESSION';
    element('vgt-code-run').disabled = !codeState.project || !TERMINAL.has(session.status);
    const active = !TERMINAL.has(session.status);
    element('vgt-code-stop')?.classList.toggle('hidden', !active);
    renderRunHealth(session);
    renderEvents(session.events || []);
    renderChanges(session);
    renderValidations(session.validation_results || []);
    renderLiveBar(session);
	renderConversationActivity(session);
    if (session.agent_summary) renderReport(session.agent_summary, session.status === 'FAILED');
    else renderReport(`${session.status} // ${session.user_request || 'Coding session'}`, session.status === 'FAILED');
    const lastEvent = (session.events || []).at(-1);
    element('vgt-code-mini-activity').textContent = `${session.status} · ${session.worker_model}\n${lastEvent?.summary || lastEvent?.title || 'Preparing execution'}`;
    if (TERMINAL.has(session.status) && codeState.completedReportedSession !== session.session_id) {
        codeState.completedReportedSession = session.session_id;
		void refreshVGTCode();
    }
}

function renderRunHealth(session) {
    const health = element('vgt-code-run-health');
    if (!health) return;
    const last = session.last_activity_at ? new Date(session.last_activity_at).getTime() : Date.now();
    const quietSeconds = Math.max(0, Math.floor((Date.now() - last) / 1000));
    const stateName = session.health || (TERMINAL.has(session.status) ? 'TERMINAL' : quietSeconds >= 90 ? 'STALLED' : quietSeconds >= 30 ? 'QUIET' : 'HEALTHY');
    health.dataset.health = stateName;
    health.textContent = session.health_message || (stateName === 'STALLED' ? `KEIN HEARTBEAT · ${quietSeconds}s` : stateName === 'QUIET' ? `MODELL ARBEITET · ${quietSeconds}s` : stateName === 'TERMINAL' ? session.status : 'LIVE · HEARTBEAT OK');
}

async function refreshSecurityMode() {
    try {
        const res = await fetch(`${state.API_BASE}/v1/security/mode`);
        if (!res.ok) return;
        const data = await res.json();
        renderSecurityMode(data.mode);
    } catch (e) {
        console.warn('Security mode query failed', e);
    }
}

function renderSecurityMode(mode) {
    const btn = element('vgt-code-mode-toggle');
    if (!btn) return;
    const isFull = mode === 'full_access';
    btn.textContent = isFull ? 'VOLLZUGRIFF' : 'INTERAKTIV';
    btn.dataset.mode = mode;
    btn.classList.toggle('full-access', isFull);
    btn.classList.toggle('interactive', !isFull);
    btn.title = isFull
        ? 'Vollzugriff aktiv: Aethel führt zulässige Aktionen autonom ohne Pop-Up aus. Klicken für Interaktiven Modus.'
        : 'Interaktiver Modus aktiv: Aethel fragt per Pop-Up nach Operator-Zustimmung. Klicken für Vollzugriff.';
}

async function toggleSecurityMode() {
    const btn = element('vgt-code-mode-toggle');
    const current = btn?.dataset.mode || 'full_access';
    const target = current === 'full_access' ? 'interactive' : 'full_access';
    try {
        const res = await fetch(`${state.API_BASE}/v1/security/mode`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ mode: target })
        });
        if (!res.ok) throw new Error(`Status ${res.status}`);
        const data = await res.json();
        renderSecurityMode(data.mode);
        showToast(target === 'full_access' ? 'Sicherheitsmodus: VOLLZUGRIFF aktiviert.' : 'Sicherheitsmodus: INTERAKTIV aktiviert.', 'success');
    } catch (e) {
        console.error('Failed to toggle security mode', e);
        showToast('Umschaltung des Sicherheitsmodus fehlgeschlagen', 'error');
    }
}


function initializeSessionConversation(session) {
	const messages = element('vgt-code-chat-messages');
	if (!messages) return;
	const task = document.createElement('div');
	task.className = 'vgt-code-message user';
	task.textContent = session.user_request || 'Coding task';
	const activity = document.createElement('section');
	activity.id = 'vgt-code-conversation-activity';
	activity.className = 'vgt-code-conversation-activity';
	messages.replaceChildren(task, activity);
}

function renderConversationActivity(session) {
	const activity = element('vgt-code-conversation-activity');
	if (!activity) return;
	const messages = element('vgt-code-chat-messages');
	const nearBottom = messages ? messages.scrollHeight - messages.scrollTop - messages.clientHeight < 72 : false;
	const preservedTop = messages?.scrollTop || 0;
	const events = session.events || [];
	const reverseEvents = [...events].reverse();
	const current = TERMINAL.has(session.status)
		? events.at(-1)
		: (reverseEvents.find(event => event.status === 'RUNNING') || reverseEvents.find(event => event.status === 'QUEUED' && event.type !== 'PLAN') || events.at(-1));
	const header = document.createElement('header');
	const pulse = document.createElement('span');
	pulse.className = `vgt-code-work-pulse ${TERMINAL.has(session.status) ? 'terminal' : ''}`;
	const heading = document.createElement('div');
	const title = document.createElement('strong');
	title.textContent = `VGT CODER · ${session.status}`;
	const models = document.createElement('small');
	models.textContent = `${session.worker_model} · tools: ${session.orchestrator_model} · ${formatDuration(session.duration_ms || 0)}`;
	heading.append(title, models);
	header.append(pulse, heading);

	const now = document.createElement('div');
	now.className = 'vgt-code-current-work';
	const phase = document.createElement('span');
	phase.textContent = current?.type || (TERMINAL.has(session.status) ? 'RESULT' : 'PREPARING');
	const currentText = document.createElement('div');
	const currentTitle = document.createElement('strong');
	currentTitle.textContent = operatorFacingTitle(current, session);
	const currentDetail = document.createElement('small');
	currentDetail.textContent = operatorFacingDetail(current);
	currentText.append(currentTitle, currentDetail);
	now.append(phase, currentText);

	const timeline = document.createElement('div');
	timeline.className = 'vgt-code-work-timeline';
	for (const event of events.slice(-80)) timeline.appendChild(renderConversationEvent(event));
	if (!events.length) timeline.appendChild(textBlock('', 'Session wird initialisiert…'));

	const result = document.createElement('div');
	result.className = 'vgt-code-conversation-result';
	if (session.agent_summary) {
		const resultTitle = document.createElement('strong');
		resultTitle.textContent = TERMINAL.has(session.status) ? 'ERGEBNIS' : 'LETZTES AGENT-UPDATE';
		const resultText = document.createElement('div');
		resultText.textContent = session.agent_summary;
		result.append(resultTitle, resultText);
	} else {
		result.classList.add('hidden');
	}
	activity.replaceChildren(header, now, timeline, result);
	window.requestAnimationFrame(() => {
		if (!messages) return;
		messages.scrollTop = nearBottom ? messages.scrollHeight : preservedTop;
	});
}

function renderConversationEvent(event) {
	const row = document.createElement('details');
	row.className = `vgt-code-work-event ${String(event.status || '').toLowerCase()}`;
	const summary = document.createElement('summary');
	const marker = document.createElement('span');
	marker.className = 'marker';
	marker.textContent = workEventGlyph(event.type);
	const body = document.createElement('div');
	const title = document.createElement('strong');
	title.textContent = humanizeWorkEvent(event);
	const detail = document.createElement('small');
	detail.textContent = event.path || event.command || event.summary || event.status || '';
	body.append(title, detail);
	const duration = document.createElement('time');
	duration.textContent = event.duration_ms ? `${event.duration_ms}ms` : event.status || '';
	summary.append(marker, body, duration);
	const evidence = document.createElement('pre');
	evidence.textContent = [event.summary, event.path && `File: ${event.path}`, event.command && `Command: ${event.command}`].filter(Boolean).join('\n');
	row.append(summary, evidence);
	return row;
}

function operatorFacingTitle(event, session) {
	if (!event) return TERMINAL.has(session.status) ? 'Coding session abgeschlossen' : 'Arbeitskontext wird vorbereitet';
	if (event.type === 'ANALYSIS') return 'Repository-Evidenz wird ausgewertet und der nächste prüfbare Schritt gewählt';
	if (event.path) return `${humanizeWorkEvent(event)} · ${event.path}`;
	if (event.command) return `${humanizeWorkEvent(event)} · ${event.command}`;
	return humanizeWorkEvent(event);
}

function operatorFacingDetail(event) {
	if (!event) return 'Projektgrenze, Modelle und Änderungsbaseline werden geladen.';
	if (event.type === 'ANALYSIS') return 'Sichtbar sind überprüfbare Entscheidungen und Tool-Aktionen – keine verborgenen internen Gedankengänge.';
	return event.summary || event.path || event.command || event.status || '';
}

function humanizeWorkEvent(event) {
	const names = {
		ANALYSIS: 'Analysiert nächsten Arbeitsschritt', DECISION: 'Entscheidung des Coding-Modells', READ_FILE: 'Inspiziert Datei',
		CREATE_FILE: 'Erstellt Datei', EDIT_FILE: 'Bearbeitet Datei', COMMAND: 'Führt Prüfung aus', APPROVAL: 'Wartet auf Freigabe',
		CHECKPOINT: 'Verifiziert Änderung', FINAL: 'Erstellt Abschlussbericht', ERROR: 'Arbeitsschritt fehlgeschlagen', PLAN: 'Plant Umsetzung', SEARCH: 'Untersucht Projekt',
	};
	return names[event?.type] || event?.title || event?.type || 'Agent-Aktivität';
}

function workEventGlyph(type) {
	return ({ ANALYSIS: '◌', DECISION: '◆', READ_FILE: '↗', CREATE_FILE: '+', EDIT_FILE: '∆', COMMAND: '›_', APPROVAL: '!', CHECKPOINT: '✓', FINAL: '■', ERROR: '×', PLAN: '◇' })[type] || '·';
}

function renderEvents(events) {
    const list = element('vgt-code-event-list');
    if (!list) return;
    const fragment = document.createDocumentFragment();
    for (const event of events.slice(-200)) {
        const detail = document.createElement('details');
        detail.className = `vgt-code-event ${String(event.status || '').toLowerCase()}`;
        const summary = document.createElement('summary');
        const type = document.createElement('span');
        type.className = 'vgt-code-event-seq';
        type.textContent = event.type || 'EVENT';
        const title = document.createElement('span');
        title.className = 'vgt-code-event-name';
        title.textContent = event.title || event.type || 'Event';
        const status = document.createElement('span');
        status.className = 'vgt-code-event-status';
        status.textContent = event.status || '';
        summary.append(type, title, status);
        const body = document.createElement('pre');
        body.textContent = [event.summary, event.path && `PATH: ${event.path}`, event.command && `COMMAND: ${event.command}`].filter(Boolean).join('\n');
        detail.append(summary, body);
        fragment.appendChild(detail);
    }
    if (!events.length) fragment.appendChild(textBlock('vgt-code-empty-event', 'Run accepted. Waiting for the first durable event.'));
    list.replaceChildren(fragment);
}

function scopedChanges(session) {
    if (!session) return [];
    if (codeState.scope === 'WORKTREE') return session.worktree_files || [];
    if (codeState.scope === 'BRANCH') return session.branch_files || [];
    return session.changed_files || [];
}

function renderChanges(session) {
    const list = element('vgt-code-changes');
    if (!list) return;
    const files = scopedChanges(session);
    const additions = files.reduce((sum, file) => sum + (file.additions || 0), 0);
    const deletions = files.reduce((sum, file) => sum + (file.deletions || 0), 0);
    element('vgt-code-changes-summary').textContent = `${files.length} files · +${additions} -${deletions}`;
    const fragment = document.createDocumentFragment();
    for (const file of files) {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'vgt-code-change';
        const status = document.createElement('b');
        status.textContent = file.status?.slice(0, 1) || 'M';
        const path = document.createElement('span');
        path.textContent = file.path;
        const stats = document.createElement('small');
        stats.textContent = file.binary ? 'BINARY' : `+${file.additions || 0} -${file.deletions || 0}`;
        button.append(status, path, stats);
        button.addEventListener('click', () => openDiff(file.path));
        fragment.appendChild(button);
    }
    if (!files.length) fragment.appendChild(textBlock('', 'No repository changes.'));
    list.replaceChildren(fragment);
}

function renderValidations(results) {
    const list = element('vgt-code-validation');
    if (!list) return;
    const fragment = document.createDocumentFragment();
    for (const result of results) {
        const row = document.createElement('details');
        row.className = `vgt-code-validation-row ${String(result.status).toLowerCase()}`;
        const summary = document.createElement('summary');
        summary.textContent = `${result.status} · ${result.name}`;
        const output = document.createElement('pre');
        output.textContent = [result.command, result.output].filter(Boolean).join('\n');
        row.append(summary, output);
        fragment.appendChild(row);
    }
    if (!results.length) fragment.appendChild(textBlock('', 'No validation results.'));
    list.replaceChildren(fragment);
}

function renderLiveBar(session) {
    const stats = session.stats || {};
    const active = !TERMINAL.has(session.status);
    element('vgt-code-livebar').classList.toggle('hidden', !active && !stats.files_changed);
    element('vgt-code-live-state').textContent = session.status;
    element('vgt-code-live-files').textContent = String(stats.files_changed || 0);
    element('vgt-code-live-added').textContent = `+${stats.lines_added || 0}`;
    element('vgt-code-live-deleted').textContent = `-${stats.lines_deleted || 0}`;
    element('vgt-code-live-tests').textContent = `${stats.tests_passed || 0} / ${stats.tests_failed || 0}`;
    element('vgt-code-live-elapsed').textContent = formatDuration(stats.duration_ms || session.duration_ms || 0);
}

async function openDiff(path) {
    if (!codeState.activeSessionID || !path) return;
    try {
        const url = `${state.API_BASE}/v1/coder/sessions/${encodeURIComponent(codeState.activeSessionID)}/diff?path=${encodeURIComponent(path)}&scope=${encodeURIComponent(codeState.scope)}`;
        codeState.activeDiff = await decodeResponse(await fetch(url, { headers: { Accept: 'application/json' } }), 'Diff');
        codeState.activeDiffPath = path;
        element('vgt-code-diff-title').textContent = `${codeState.activeDiff.status} · ${path}`;
        renderDiff();
        switchCodeTab('diff');
    } catch (error) { showToast(error.message, 'error'); }
}

function setDiffMode(mode) {
    codeState.diffMode = mode;
    element('vgt-code-diff-unified').classList.toggle('active', mode === 'unified');
    element('vgt-code-diff-split').classList.toggle('active', mode === 'split');
    renderDiff();
}

function renderDiff() {
    const container = element('vgt-code-diff-content');
    const diff = codeState.activeDiff;
    if (!container || !diff) return;
    if (diff.binary) return container.replaceChildren(textBlock('vgt-code-diff-empty', 'Binary file changed. Text diff is unavailable.'));
    const fragment = document.createDocumentFragment();
    for (const hunk of diff.hunks || []) {
        const section = document.createElement('section');
        const header = document.createElement('div');
        header.className = 'vgt-code-diff-hunk';
        header.textContent = hunk.header;
        section.appendChild(header);
        if (codeState.diffMode === 'split') section.appendChild(renderSplitLines(hunk.lines || []));
        else for (const line of hunk.lines || []) section.appendChild(renderUnifiedLine(line));
        fragment.appendChild(section);
    }
    if (!(diff.hunks || []).length) fragment.appendChild(textBlock('vgt-code-diff-empty', 'No textual changes in this scope.'));
    container.replaceChildren(fragment);
}

function renderUnifiedLine(line) {
    const row = document.createElement('div');
    row.className = `vgt-code-diff-line ${line.kind}`;
    const oldNumber = document.createElement('span'); oldNumber.textContent = line.old_number || '';
    const newNumber = document.createElement('span'); newNumber.textContent = line.new_number || '';
    const code = document.createElement('code'); code.textContent = `${line.kind === 'added' ? '+' : line.kind === 'deleted' ? '-' : ' '}${line.content}`;
    row.append(oldNumber, newNumber, code);
    return row;
}

function renderSplitLines(lines) {
    const grid = document.createElement('div');
    grid.className = 'vgt-code-diff-split-grid';
    let index = 0;
    while (index < lines.length) {
        const left = lines[index]?.kind === 'deleted' ? lines[index++] : null;
        const right = lines[index]?.kind === 'added' ? lines[index++] : (left ? null : lines[index++]);
        grid.append(renderSplitCell(left, 'deleted'), renderSplitCell(right, right?.kind || 'context'));
    }
    return grid;
}

function renderSplitCell(line, kind) {
    const cell = document.createElement('div');
    cell.className = `vgt-code-diff-cell ${kind}`;
    const number = document.createElement('span');
    number.textContent = line ? String(kind === 'deleted' ? line.old_number || '' : line.new_number || '') : '';
    const code = document.createElement('code');
    code.textContent = line?.content || '';
    cell.append(number, code);
    return cell;
}

function navigateDiff(direction) {
    const files = scopedChanges(codeState.currentSession);
    if (!files.length) return;
    const index = Math.max(0, files.findIndex(file => file.path === codeState.activeDiffPath));
    openDiff(files[(index + direction + files.length) % files.length].path);
}

function askCoderAboutDiff() {
    if (!codeState.activeDiffPath) return;
    element('vgt-code-objective').value = `Review the changes in ${codeState.activeDiffPath}. Check correctness, edge cases, security and tests, then implement any required fixes.`;
    switchCodeTab('chat');
    element('vgt-code-objective').focus();
}

function handleCoderShortcut(event) {
    const codeView = element('view-code');
    if (!codeView || codeView.classList.contains('hidden')) return;
    if ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key.toLowerCase() === 'd') {
        event.preventDefault();
        const first = scopedChanges(codeState.currentSession)[0];
        if (first) openDiff(first.path);
    } else if (event.key === 'Escape' && !element('vgt-code-diff-pane').classList.contains('hidden')) switchCodeTab('plan');
    else if (!['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName) && event.key.toLowerCase() === 'n') navigateDiff(1);
    else if (!['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName) && event.key.toLowerCase() === 'p') navigateDiff(-1);
}

function countFiles(entries) { return entries.reduce((count, entry) => count + (entry.kind === 'file' ? 1 : countFiles(entry.children || [])), 0); }

function renderTreeEntry(entry, depth) {
    const wrapper = document.createElement('div');
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'vgt-code-tree-item';
    button.dataset.kind = entry.kind;
    button.dataset.path = entry.path;
    button.style.paddingLeft = `${7 + Math.min(depth, 8) * 11}px`;
    const icon = document.createElement('span');
    icon.className = 'vgt-code-tree-icon';
    icon.textContent = entry.kind === 'directory' ? '▾' : fileGlyph(entry.name);
    const name = document.createElement('span');
    name.textContent = entry.name;
    button.append(icon, name);
    wrapper.appendChild(button);
    if (entry.kind === 'directory') {
        const children = document.createElement('div');
        children.className = 'vgt-code-tree-children';
        for (const child of entry.children || []) children.appendChild(renderTreeEntry(child, depth + 1));
        button.addEventListener('click', () => { children.hidden = !children.hidden; icon.textContent = children.hidden ? '▸' : '▾'; });
        wrapper.appendChild(children);
    } else button.addEventListener('click', () => openCodeFile(entry.path, button));
    return wrapper;
}

function fileGlyph(name) {
    const extension = name.includes('.') ? name.split('.').pop().toLowerCase() : '';
    if (['js', 'ts', 'jsx', 'tsx'].includes(extension)) return '◆';
    if (['go', 'rs', 'py', 'java', 'cs'].includes(extension)) return '◇';
    if (['json', 'yaml', 'yml', 'toml'].includes(extension)) return '≡';
    return '·';
}

async function openCodeFile(path, button) {
    if (!path || !codeState.project) return showToast('Open the matching project before reading this file.', 'error');
    if (isDirty() && !window.confirm('Ungespeicherte Änderungen verwerfen?')) return;
    setFileStatus('LOADING');
    try {
        const payload = await decodeResponse(await fetch(`${state.API_BASE}/v1/code/workspace/file?path=${encodeURIComponent(path)}`, { headers: { Accept: 'application/json' } }), 'File read');
        codeState.activePath = payload.path;
        codeState.originalContent = payload.content;
        const editor = element('vgt-code-content');
        editor.value = payload.content;
        editor.disabled = false;
        document.querySelectorAll('.vgt-code-tree-item.active').forEach(node => node.classList.remove('active'));
        button?.classList.add('active');
        element('vgt-code-active-file').textContent = payload.path;
        element('vgt-code-editor-title').textContent = payload.path;
        syncEditorMetrics();
        updateDirtyState();
        switchCodeTab('editor');
    } catch (error) { setFileStatus('READ FAILED'); showToast(error.message, 'error'); }
}

async function saveActiveFile() {
    const editor = element('vgt-code-content');
    if (!codeState.activePath || !editor || editor.disabled || !isDirty()) return;
    setFileStatus('SNAPSHOTTING');
    element('vgt-code-save').disabled = true;
    try {
        const payload = await decodeResponse(await fetch(`${state.API_BASE}/v1/code/workspace/file`, {
            method: 'PUT', headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
            body: JSON.stringify({ path: codeState.activePath, content: editor.value }),
        }), 'File save');
        codeState.originalContent = editor.value;
        updateDirtyState();
        showToast(`Saved with recovery snapshot ${payload.snapshot_id}`, 'success');
        if (codeState.activeSessionID) await syncSession(codeState.activeSessionID, false);
    } catch (error) { setFileStatus('SAVE FAILED'); showToast(error.message, 'error'); element('vgt-code-save').disabled = false; }
}

function isDirty() { const editor = element('vgt-code-content'); return Boolean(codeState.activePath && editor && editor.value !== codeState.originalContent); }
function updateDirtyState() { const dirty = isDirty(); element('vgt-code-save').disabled = !dirty; setFileStatus(dirty ? 'MODIFIED' : (codeState.activePath ? 'SAVED' : 'READ ONLY')); }
function setFileStatus(value) { if (element('vgt-code-file-state')) element('vgt-code-file-state').textContent = value; }
function syncEditorMetrics() {
    const editor = element('vgt-code-content');
    if (!editor) return;
    const count = Math.max(1, editor.value.split('\n').length);
    element('vgt-code-lines').textContent = Array.from({ length: count }, (_, index) => index + 1).join('\n');
    const prefix = editor.value.slice(0, editor.selectionStart);
    const line = prefix.split('\n').length;
    element('vgt-code-position').textContent = `LN ${line} · COL ${prefix.length - prefix.lastIndexOf('\n')}`;
}

function appendChatMessage(role, content) {
    const message = document.createElement('div');
    message.className = `vgt-code-message ${role}`;
    message.textContent = content;
    element('vgt-code-chat-messages')?.appendChild(message);
    if (element('vgt-code-chat-messages')) element('vgt-code-chat-messages').scrollTop = element('vgt-code-chat-messages').scrollHeight;
}
function renderReport(text, failed) { const report = element('vgt-code-run-report'); if (report) { report.textContent = text; report.classList.toggle('error', failed); } }
function setRunState(value) { if (element('vgt-code-run-state')) element('vgt-code-run-state').textContent = value; }
function textBlock(className, text) { const node = document.createElement('p'); node.className = className; node.textContent = text; return node; }
function formatDuration(ms) { const seconds = Math.max(0, Math.floor(ms / 1000)); return `${String(Math.floor(seconds / 60)).padStart(2, '0')}:${String(seconds % 60).padStart(2, '0')}`; }
function showToast(message, type) { if (typeof window.showAethelToast === 'function') window.showAethelToast(message, type); else console[type === 'error' ? 'error' : 'info'](message); }
