import { state } from './state.js';
import * as api from './api.js';
import { updateChecklistUI } from './chat.js';
import { speak } from './voice.js';

let activeTeamRunId = null;
let customAgentCounter = 0;
let lastTraceKey = '';
const roleIDs = ['orchestrator', 'builder', 'reviewer', 'repairer', 'documentator'];

const DEFAULT_ROLE_PROMPTS = {
    orchestrator: 'Du zerlegst Ziele in verifizierbare Schritte.',
    builder: 'Du führst nur freigegebene, überprüfbare Arbeitsschritte aus.',
    reviewer: 'Du prüfst Ergebnisse gegen das Ziel.',
    repairer: 'Du analysierst Fehler und lieferst sichere Korrekturen.',
    documentator: 'Du erzeugst präzise Abschlussberichte.'
};

window.addCustomAgentUI = addCustomAgentUI;
window.removeCustomAgentUI = removeCustomAgentUI;

function persistBuilderConfiguration() {
    for (const role of roleIDs) {
        const model = document.getElementById(`agent-model-${role}`);
        const prompt = document.getElementById(`agent-prompt-${role}`);
        const persona = document.getElementById(`agent-persona-${role}`);
        if (model) localStorage.setItem(`aethel_agent_role_agent-model-${role}`, model.value);
        if (prompt) localStorage.setItem(`aethel_agent_text_agent-prompt-${role}`, prompt.value);
        if (persona) localStorage.setItem(`aethel_agent_persona_agent-persona-${role}`, persona.value);
    }
    for (const id of ['agent-team-task-input', 'agent-toggle-reviewer', 'agent-toggle-repairer', 'agent-toggle-documentator']) {
        const element = document.getElementById(id);
        if (element) localStorage.setItem(`aethel_agent_${element.type === 'checkbox' ? 'check' : 'text'}_${id}`, element.type === 'checkbox' ? String(element.checked) : element.value);
    }
}

function appendMessage(role, content, sender = '') {
    const output = document.getElementById('agent-chat-output');
    if (!output) return;
    if (output.textContent.includes('[STANDBY]')) output.replaceChildren();
    const bubble = document.createElement('div');
    bubble.className = `message ${role}`;
    const header = document.createElement('div');
    header.className = 'msg-header';
    header.textContent = sender || (role === 'user' ? 'OPERATOR // TERMINAL' : 'SYSTEM // RUNNER');
    const body = document.createElement('div');
    body.className = 'msg-body';
    body.textContent = content;
    bubble.append(header, body);
    output.appendChild(bubble);
    output.scrollTop = output.scrollHeight;
}

function restoreInputs() {
    for (const role of roleIDs) {
        const model = document.getElementById(`agent-model-${role}`);
        const prompt = document.getElementById(`agent-prompt-${role}`);
        const persona = document.getElementById(`agent-persona-${role}`);
        if (model) model.value = localStorage.getItem(`aethel_agent_role_agent-model-${role}`) || model.value;
        if (prompt) prompt.value = localStorage.getItem(`aethel_agent_text_agent-prompt-${role}`) || prompt.value || DEFAULT_ROLE_PROMPTS[role];
        if (persona) persona.value = localStorage.getItem(`aethel_agent_persona_agent-persona-${role}`) || persona.value;
    }
    for (const id of ['agent-team-task-input', 'agent-toggle-reviewer', 'agent-toggle-repairer', 'agent-toggle-documentator']) {
        const element = document.getElementById(id);
        const value = localStorage.getItem(`aethel_agent_${element?.type === 'checkbox' ? 'check' : 'text'}_${id}`);
        if (element && value !== null) {
            if (element.type === 'checkbox') element.checked = value === 'true';
            else element.value = value;
        }
    }
}

async function populateModels() {
    const payload = await api.getModels();
    for (const role of roleIDs) {
        const select = document.getElementById(`agent-model-${role}`);
        if (!select) continue;
        const remembered = localStorage.getItem(`aethel_agent_role_agent-model-${role}`) || select.value;
        select.replaceChildren();
        for (const model of payload.models || []) {
            const option = document.createElement('option');
            option.value = model.id;
            option.textContent = `${model.name} (${model.provider})`;
            select.appendChild(option);
        }
        if ([...select.options].some(option => option.value === remembered)) select.value = remembered;
    }
}

function addCustomAgentUI(name = '', model = '', prompt = '', active = true, persona = 'default') {
    const container = document.getElementById('custom-agents-container');
    if (!container) return;
    const id = `custom-agent-${++customAgentCounter}`;
    const card = document.createElement('article');
    card.id = id;
    card.className = 'custom-agent-card';
    const title = document.createElement('input'); title.className = 'custom-agent-name'; title.value = name; title.placeholder = 'Agentname';
    const modelInput = document.createElement('input'); modelInput.className = 'custom-agent-model'; modelInput.value = model; modelInput.placeholder = 'Modell-ID';
    const promptInput = document.createElement('textarea'); promptInput.className = 'custom-agent-prompt'; promptInput.value = prompt; promptInput.placeholder = 'Systemanweisung';
    const enabled = document.createElement('input'); enabled.type = 'checkbox'; enabled.className = 'custom-agent-active'; enabled.checked = active;
    const personaInput = document.createElement('input'); personaInput.className = 'custom-agent-persona'; personaInput.value = persona;
    const remove = document.createElement('button'); remove.type = 'button'; remove.textContent = 'ENTFERNEN'; remove.addEventListener('click', () => removeCustomAgentUI(id));
    card.append(title, modelInput, promptInput, enabled, personaInput, remove);
    container.appendChild(card);
}

function buildTeamSystemPrompt() {
    const sections = [];
    for (const role of roleIDs) {
        const toggle = document.getElementById(`agent-toggle-${role}`);
        if (toggle && !toggle.checked) continue;
        const prompt = document.getElementById(`agent-prompt-${role}`)?.value.trim();
        if (!prompt) continue;
        const persona = document.getElementById(`agent-persona-${role}`)?.value || 'default';
        const model = document.getElementById(`agent-model-${role}`)?.value || 'selected-runtime-model';
        sections.push(`[TEAM ROLE: ${role.toUpperCase()} | persona=${persona} | preferred_model=${model}]\n${prompt}`);
    }
    document.querySelectorAll('#custom-agents-container .custom-agent-card').forEach((card, index) => {
        if (!card.querySelector('.custom-agent-active')?.checked) return;
        const name = card.querySelector('.custom-agent-name')?.value.trim() || `custom-${index + 1}`;
        const prompt = card.querySelector('.custom-agent-prompt')?.value.trim();
        if (!prompt) return;
        const persona = card.querySelector('.custom-agent-persona')?.value.trim() || 'default';
        const model = card.querySelector('.custom-agent-model')?.value.trim() || 'selected-runtime-model';
        sections.push(`[CUSTOM TEAM ROLE: ${name} | persona=${persona} | preferred_model=${model}]\n${prompt}`);
    });
    sections.push('TEAM EXECUTION CONTRACT:\nPlane in verifizierbaren Schritten. Führe ausschließlich zulässige Tools aus. Reviewer validieren Ergebnisse, Repairer beheben konkrete Fehler und der Documentator erzeugt den Abschlussbericht. Freigaben, Pausen und Wiederaufnahme laufen über den persistenten Run.');
    return sections.join('\n\n');
}

function removeCustomAgentUI(id) {
    document.getElementById(id)?.remove();
}

function setControls(active) {
    document.getElementById('agent-btn-launch-team')?.classList.toggle('hidden', active);
    document.getElementById('agent-btn-abort-run')?.classList.toggle('hidden', !active);
    const task = document.getElementById('agent-team-task-input');
    if (task) task.disabled = active;
}

function trackerItemsForRun(run) {
    const items = (run.steps || []).map(step => {
        const state = step.status === 'verified' ? 'done' : step.status === 'running' || step.status === 'waiting_approval' ? 'in_progress' : 'todo';
        const tool = step.tool_name ? ` · ${step.tool_name}` : '';
        return { text: `${step.title || step.kind || 'Run-Schritt'}${tool}`, status: state };
    });
    if (run.status === 'running' && !items.some(item => item.status === 'in_progress')) {
        items.push({ text: 'CORTEX verarbeitet den nächsten verifizierten Schritt', status: 'in_progress' });
    }
    if (run.status === 'waiting_approval') {
        items.push({ text: 'Operator-Freigabe ausstehend – Sicherheitsdialog ist geöffnet', status: 'in_progress' });
    }
    return items;
}

function appendRunTrace(run) {
    const trace = Array.isArray(run.trace) && run.trace.length ? run.trace[run.trace.length - 1] : null;
    if (!trace) return;
    const key = `${run.id}:${trace.timestamp}:${trace.event}:${trace.step_id || ''}`;
    if (key === lastTraceKey) return;
    lastTraceKey = key;
    appendMessage('system_log', `[${String(run.status || 'running').toUpperCase()}] ${trace.detail || trace.event}`);
}

async function monitorTeamRun(runID) {
    while (activeTeamRunId === runID) {
        try {
            const run = await api.getRun(runID);
            updateChecklistUI(trackerItemsForRun(run));
            appendRunTrace(run);
            if (run.status === 'completed') {
                appendMessage('assistant', run.final_report || 'Team-Run erfolgreich abgeschlossen.', 'TEAM // ABSCHLUSS');
                activeTeamRunId = null;
                if (state.activeRunId === runID) state.activeRunId = null;
                setControls(false);
                return;
            }
            if (run.status === 'failed' || run.status === 'cancelled') {
                appendMessage('system_log', `[${String(run.status).toUpperCase()}] ${run.failure_reason || 'Run wurde beendet.'}`);
                activeTeamRunId = null;
                if (state.activeRunId === runID) state.activeRunId = null;
                setControls(false);
                return;
            }
        } catch (error) {
            appendMessage('system_log', `Run-Status konnte nicht geladen werden: ${error.message}`);
            activeTeamRunId = null;
            if (state.activeRunId === runID) state.activeRunId = null;
            setControls(false);
            return;
        }
        await new Promise(resolve => window.setTimeout(resolve, 700));
    }
}

async function startTeamRun() {
    persistBuilderConfiguration();
    const objective = document.getElementById('agent-team-task-input')?.value.trim();
    if (!objective) {
        window.alert('Bitte ein konkretes Ziel eingeben.');
        return;
    }
    const modelID = document.getElementById('agent-model-builder')?.value || state.currentModel;
    const prompt = buildTeamSystemPrompt();
    try {
        const response = await fetch(`${state.API_BASE}/v1/chat/runs`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ objective, profile_id: 'developer', mode: 'agent_team', model_id: modelID, system_prompt: prompt, messages: [{ role: 'user', content: objective }], cost_budget_usd: 5, max_agent_turns: 24 }) });
        if (!response.ok) throw new Error(await response.text());
        const run = await response.json();
        activeTeamRunId = run.id;
        state.activeRunId = run.id;
        lastTraceKey = '';
        setControls(true);
        updateChecklistUI([{ text: 'Agentenziel und AusfÃ¼hrungsrahmen werden validiert', status: 'in_progress' }]);
        appendMessage('assistant', `Persistenter Agenten-Run ${run.id} gestartet. Der Ablauf wird hier live angezeigt; Freigaben erscheinen als globales Sicherheitsdialogfenster.`);
        speak('Persistenter Agenten-Run gestartet.');
        void monitorTeamRun(run.id);
    } catch (error) {
        appendMessage('system_log', `Run konnte nicht gestartet werden: ${error.message}`);
    }
}

export function abortTeamRun() {
    if (!activeTeamRunId) return;
    fetch(`${state.API_BASE}/v1/runs/${encodeURIComponent(activeTeamRunId)}/cancel`, { method: 'POST' }).finally(() => {
        appendMessage('system_log', 'Aktiver Run wurde zum Abbruch angefordert.');
        activeTeamRunId = null;
        if (state.activeRunId) state.activeRunId = null;
        setControls(false);
    });
}

export function sendOperatorFeedback() {
    const input = document.getElementById('agent-user-input');
    const value = input?.value.trim();
    if (!value) return;
    input.value = '';
    appendMessage('user', value);
    appendMessage('system_log', 'Operator-Hinweis wurde lokal erfasst; der persistente Runner verarbeitet nur explizite Run-Schritte.');
}

export async function setupAgentBuilder() {
    try { await populateModels(); } catch (error) { console.error('Agent Builder models unavailable', error); }
    restoreInputs();
    if (window.refreshPersonasDropdowns) window.refreshPersonasDropdowns();
    document.getElementById('agent-btn-launch-team')?.addEventListener('click', startTeamRun);
    document.getElementById('agent-btn-abort-run')?.addEventListener('click', abortTeamRun);
    document.getElementById('agent-btn-send')?.addEventListener('click', sendOperatorFeedback);
    document.getElementById('agent-btn-add-custom')?.addEventListener('click', event => { event.preventDefault(); addCustomAgentUI(); });
    document.getElementById('agent-user-input')?.addEventListener('keydown', event => { if (event.key === 'Enter' && !event.shiftKey) { event.preventDefault(); sendOperatorFeedback(); } });
	try {
		const payload = await api.getRuns();
		const resumable = (payload.runs || payload || [])
			.filter(run => run.mode === 'agent_team' && ['running', 'waiting_approval', 'paused'].includes(run.status))
			.sort((left, right) => String(right.updated_at || '').localeCompare(String(left.updated_at || '')))[0];
		if (resumable) {
			activeTeamRunId = resumable.id;
			state.activeRunId = resumable.id;
			setControls(true);
			updateChecklistUI(trackerItemsForRun(resumable));
			appendMessage('assistant', `Agenten-Team-Run ${resumable.id} wurde wieder aufgenommen.`, 'SYSTEM // RECOVERY');
			void monitorTeamRun(resumable.id);
		}
	} catch (error) {
		console.warn('Agent Team recovery unavailable', error);
	}
}
