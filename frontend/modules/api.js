import { state } from './state.js';

async function jsonResponse(response, label) {
    if (response.ok) return response.json();
    const detail = (await response.text()).slice(0, 300);
    throw new Error(`${label} (${response.status}): ${detail || 'unavailable'}`);
}

export async function checkSystemStatus() {
    const res = await fetch(`${state.API_BASE}/health`);
    return jsonResponse(res, 'Core health');
}

export async function submitSetup(key, openaiKey, deepseekKey = '', geminiKey = '', claudeKey = '') {
    const res = await fetch(`${state.API_BASE}/v1/setup`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
            api_key: key || '',
            openai_api_key: openaiKey || '',
            deepseek_api_key: deepseekKey || '',
            gemini_api_key: geminiKey || '',
            claude_api_key: claudeKey || ''
        })
    });
    return jsonResponse(res, 'Setup');
}

export async function getModels() {
    const res = await fetch(`${state.API_BASE}/v1/models`);
    return jsonResponse(res, 'Model registry');
}

export async function getVoices() {
    const res = await fetch(`${state.API_BASE}/v1/audio/voices`);
    return jsonResponse(res, 'Voice registry');
}

export async function getVoiceHealth() {
    const res = await fetch(`${state.API_BASE}/v1/audio/health`);
    return jsonResponse(res, 'Voice health');
}

export async function runVoiceTest(voice) {
    const res = await fetch(`${state.API_BASE}/v1/audio/test`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ voice })
    });
    return res.json();
}

export async function getSecrets() {
    const res = await fetch(`${state.API_BASE}/v1/secrets`);
    return res.json();
}

export async function addSecret(item) {
    const res = await fetch(`${state.API_BASE}/v1/secrets`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(item)
    });
    return res.json();
}

export async function deleteSecret(id) {
    const res = await fetch(`${state.API_BASE}/v1/secrets?id=${encodeURIComponent(id)}`, {
        method: 'DELETE'
    });
    return res.json();
}

export async function getTasks() {
    const res = await fetch(`${state.API_BASE}/v1/kernel/tasks/`);
    return res.json();
}

export async function getRuns() {
    const res = await fetch(`${state.API_BASE}/v1/runs`);
    if (!res.ok) throw new Error(`Run Center unavailable (${res.status})`);
    return res.json();
}

export async function getRun(id) {
    const res = await fetch(`${state.API_BASE}/v1/runs/${encodeURIComponent(id)}`);
    if (!res.ok) throw new Error(`Agent run unavailable (${res.status})`);
    return res.json();
}

export async function getArtifacts() {
    const res = await fetch(`${state.API_BASE}/v1/artifacts`);
    if (!res.ok) throw new Error(`Artifact Center unavailable (${res.status})`);
    return res.json();
}

export async function getProviderHealth() {
    const res = await fetch(`${state.API_BASE}/v1/providers/health`);
    if (!res.ok) throw new Error(`Provider Health Service unavailable (${res.status})`);
    return res.json();
}

export async function createRun(run) {
    const res = await fetch(`${state.API_BASE}/v1/runs`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(run)
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

export async function runAction(id, action, payload = undefined) {
    const runID = encodeURIComponent(id);
    const res = await fetch(`${state.API_BASE}/v1/runs/${runID}/${encodeURIComponent(action)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: payload === undefined ? undefined : JSON.stringify(payload)
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

export async function addTask(task) {
    const res = await fetch(`${state.API_BASE}/v1/kernel/tasks/`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(task)
    });
    return res.json();
}

export async function deleteTask(id) {
    const taskID = encodeURIComponent(id);
    const res = await fetch(`${state.API_BASE}/v1/kernel/tasks/${taskID}`, {
        method: 'DELETE'
    });
    return res.json();
}

export async function runTaskManual(id) {
    const taskID = encodeURIComponent(id);
    const res = await fetch(`${state.API_BASE}/v1/kernel/tasks/${taskID}/run`, {
        method: 'POST'
    });
    return res.json();
}

export async function pauseTaskManual(id) {
    const taskID = encodeURIComponent(id);
    const res = await fetch(`${state.API_BASE}/v1/kernel/tasks/${taskID}/pause`, {
        method: 'POST'
    });
    return res.json();
}

export async function getSettings() {
    const res = await fetch(`${state.API_BASE}/v1/settings`);
    return res.json();
}

export async function resetSettings() {
    const res = await fetch(`${state.API_BASE}/v1/settings/reset`, {
        method: 'POST'
    });
    return res.json();
}

export async function getAPICosts() {
    const res = await fetch(`${state.API_BASE}/v1/settings/costs`);
    return res.json();
}

export async function getPersonas() {
    const res = await fetch(`${state.API_BASE}/v1/settings/personas`);
    return jsonResponse(res, 'Get personas');
}

export async function savePersona(persona) {
    const res = await fetch(`${state.API_BASE}/v1/settings/personas`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(persona)
    });
    return jsonResponse(res, 'Save persona');
}

export async function deletePersona(id) {
    const res = await fetch(`${state.API_BASE}/v1/settings/personas?id=${encodeURIComponent(id)}`, {
        method: 'DELETE'
    });
    return jsonResponse(res, 'Delete persona');
}

export async function getPersonalStatus() {
    const res = await fetch(`${state.API_BASE}/v1/personal/status`);
    return res.json();
}

export async function getPersonalConfig() {
    const res = await fetch(`${state.API_BASE}/v1/personal/config`);
    return res.json();
}

export async function savePersonalConfig(config) {
    const res = await fetch(`${state.API_BASE}/v1/personal/config`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config)
    });
    return res.json();
}

export async function getPersonalProfile() {
    const res = await fetch(`${state.API_BASE}/v1/personal/profile`);
    return res.json();
}

export async function savePersonalProfile(profile) {
    const res = await fetch(`${state.API_BASE}/v1/personal/profile`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(profile)
    });
    return res.json();
}

export async function getPersonalMemories() {
    const res = await fetch(`${state.API_BASE}/v1/personal/memories`);
    return res.json();
}

export async function savePersonalMemory(memory) {
    const res = await fetch(`${state.API_BASE}/v1/personal/memories`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...memory, operator_approved: true })
    });
    return res.json();
}

export async function updatePersonalMemory(memory) {
    const res = await fetch(`${state.API_BASE}/v1/personal/memories`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(memory)
    });
    return res.json();
}

export async function deletePersonalMemory(id) {
    const res = await fetch(`${state.API_BASE}/v1/personal/memories?id=${encodeURIComponent(id)}`, {
        method: 'DELETE'
    });
    return res.json();
}

export async function runPersonalSetup(payload) {
    const res = await fetch(`${state.API_BASE}/v1/personal/setup`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
    });
    return res.json();
}

export async function getPersonalSetupQuestions(config) {
    const res = await fetch(`${state.API_BASE}/v1/personal/setup/questions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ config })
    });
    return res.json();
}

export async function runPersonalLearning(text) {
    const res = await fetch(`${state.API_BASE}/v1/personal/learn`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text })
    });
    return res.json();
}
