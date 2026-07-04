import { state } from './state.js';

export async function checkSystemStatus() {
    const res = await fetch(`${state.API_BASE}/health`);
    return res.json();
}

export async function submitSetup(key, openaiKey) {
    const res = await fetch(`${state.API_BASE}/v1/setup`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
            api_key: key,
            openai_api_key: openaiKey
        })
    });
    return res.json();
}

export async function getModels() {
    const res = await fetch(`${state.API_BASE}/v1/models`);
    return res.json();
}

export async function getVoices() {
    const res = await fetch(`${state.API_BASE}/v1/audio/voices`);
    return res.json();
}

export async function getVoiceHealth() {
    const res = await fetch(`${state.API_BASE}/v1/audio/health`);
    return res.json();
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
    const res = await fetch(`${state.API_BASE}/v1/secrets?id=${id}`, {
        method: 'DELETE'
    });
    return res.json();
}

export async function getTasks() {
    const res = await fetch(`${state.API_BASE}/v1/kernel/tasks/`);
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
    const res = await fetch(`${state.API_BASE}/v1/kernel/tasks/${id}`, {
        method: 'DELETE'
    });
    return res.json();
}

export async function runTaskManual(id) {
    const res = await fetch(`${state.API_BASE}/v1/kernel/tasks/${id}/run`, {
        method: 'POST'
    });
    return res.json();
}

export async function pauseTaskManual(id) {
    const res = await fetch(`${state.API_BASE}/v1/kernel/tasks/${id}/pause`, {
        method: 'POST'
    });
    return res.json();
}
