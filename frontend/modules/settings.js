// VGT AETHEL // SETTINGS MODULE
// Handles: API Key management (Groq, DeepSeek, OpenAI, Gemini, Claude), setup wizard re-trigger, mounted dirs

import { state } from './state.js';
import * as api from './api.js';

function escapeHtml(value) {
    return String(value ?? "")
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}

function jsArg(value) {
    return escapeHtml(JSON.stringify(String(value ?? "")));
}

export async function loadSettingsStatus() {
    try {
        const data = await api.getSettings();
        renderSettingsStatus(data);
        await loadCustomPersonasSettings();
        await loadProviderHealth();
    } catch (e) {
        console.error('Failed to load settings status', e);
    }
}

function renderSettingsStatus(data) {
    const rows = [
        { indicator: 'settings-groq-status',     label: 'settings-groq-label',     configured: data.groq_configured,     preview: data.groq_key_preview,     onText: 'GROQ KEY AKTIV',     offText: 'KEIN GROQ KEY GESETZT',                color: 'var(--vgt-cyan)' },
        { indicator: 'settings-deepseek-status', label: 'settings-deepseek-label', configured: data.deepseek_configured, preview: data.deepseek_key_preview, onText: 'DEEPSEEK KEY AKTIV', offText: 'KEIN DEEPSEEK KEY GESETZT',            color: '#00c8ff' },
        { indicator: 'settings-openai-status',   label: 'settings-openai-label',   configured: data.openai_configured,   preview: data.openai_key_preview,   onText: 'OPENAI KEY AKTIV',   offText: 'OPENAI KEY NICHT GESETZT (Optional)',  color: 'var(--vgt-purple)' },
        { indicator: 'settings-gemini-status',   label: 'settings-gemini-label',   configured: data.gemini_configured,   preview: data.gemini_key_preview,   onText: 'GEMINI KEY AKTIV',   offText: 'GEMINI KEY NICHT GESETZT (Optional)',  color: '#4285f4' },
        { indicator: 'settings-claude-status',   label: 'settings-claude-label',   configured: data.claude_configured,   preview: data.claude_key_preview,   onText: 'CLAUDE KEY AKTIV',   offText: 'CLAUDE KEY NICHT GESETZT (Optional)',  color: '#d97757' },
    ];

    for (const row of rows) {
        const indicator = document.getElementById(row.indicator);
        const label     = document.getElementById(row.label);
        if (!indicator || !label) continue;
        if (row.configured) {
            indicator.className = 'pulse-dot green';
            label.textContent = `${row.onText} (${row.preview || '****'})`;
            label.style.color = row.color;
        } else {
            indicator.className = 'pulse-dot red';
            label.textContent = row.offText;
            label.style.color = 'var(--vgt-text-dim)';
        }
    }

    const osLabel = document.getElementById('settings-os-label');
    if (osLabel && data.os) osLabel.textContent = data.os.toUpperCase() + ' (AUTOMATISCH ERKANNT)';

    const dirsContainer = document.getElementById('settings-mounted-dirs');
    if (dirsContainer) {
        const dirs = data.mounted_dirs || [];
        if (dirs.length === 0) {
            dirsContainer.innerHTML = '<span style="color: var(--vgt-text-dim); font-size: 10px;">Keine Verzeichnisse eingebunden.</span>';
        } else {
            dirsContainer.innerHTML = dirs.map(d =>
                `<div style="display:flex; justify-content:space-between; align-items:center; padding: 6px 8px; background: rgba(0,0,0,0.3); border: 1px solid rgba(255,255,255,0.05); border-radius:4px; margin-bottom:4px;">
                    <span style="font-size:10px; color: var(--vgt-cyan); font-family: var(--font-mono);">${escapeHtml(d)}</span>
                </div>`
            ).join('');
        }
    }

    ['settings-groq-input','settings-deepseek-input','settings-openai-input','settings-gemini-input','settings-claude-input'].forEach(id => {
        const el = document.getElementById(id);
        if (el) el.value = '';
    });

    const feedback = document.getElementById('settings-save-feedback');
    if (feedback) { feedback.textContent = ''; feedback.className = 'hidden'; }
}

export function setupSettingsUIEvents() {
    const btnSave = document.getElementById('settings-btn-save');
    if (btnSave) btnSave.addEventListener('click', saveSettingsKeys);

    const btnForceSetup = document.getElementById('settings-btn-force-setup');
    if (btnForceSetup) btnForceSetup.addEventListener('click', forceSetupWizard);

    const btnReset = document.getElementById('settings-btn-reset');
    if (btnReset) btnReset.addEventListener('click', resetConfig);

    // Custom Persona Event Listeners
    const btnSavePersona = document.getElementById('settings-persona-btn-save');
    if (btnSavePersona) btnSavePersona.addEventListener('click', saveCustomPersona);

    const btnClearPersona = document.getElementById('settings-persona-btn-clear');
    if (btnClearPersona) btnClearPersona.addEventListener('click', clearPersonaForm);
}

async function saveSettingsKeys() {
    const groqKey   = (document.getElementById('settings-groq-input')    ?.value.trim()) || '';
    const dsKey     = (document.getElementById('settings-deepseek-input') ?.value.trim()) || '';
    const openaiKey = (document.getElementById('settings-openai-input')   ?.value.trim()) || '';
    const geminiKey = (document.getElementById('settings-gemini-input')   ?.value.trim()) || '';
    const claudeKey = (document.getElementById('settings-claude-input')   ?.value.trim()) || '';
    const btnSave   = document.getElementById('settings-btn-save');

    if (!groqKey && !dsKey && !openaiKey && !geminiKey && !claudeKey) {
        showSettingsFeedback('Mindestens ein Key erforderlich (Groq, DeepSeek, OpenAI, Gemini oder Claude).', 'error');
        return;
    }
    if (groqKey   && !groqKey.startsWith('gsk_'))     { showSettingsFeedback('Ungültiges Groq Key Format. Muss mit "gsk_" beginnen.', 'error');       return; }
    if (dsKey     && !dsKey.startsWith('sk-'))         { showSettingsFeedback('Ungültiges DeepSeek Key Format. Muss mit "sk-" beginnen.', 'error');     return; }
    if (openaiKey && !openaiKey.startsWith('sk-'))     { showSettingsFeedback('Ungültiges OpenAI Key Format. Muss mit "sk-" beginnen.', 'error');       return; }
    if (geminiKey && !geminiKey.startsWith('AIza'))    { showSettingsFeedback('Ungültiges Gemini Key Format. Muss mit "AIza" beginnen.', 'error');      return; }
    if (claudeKey && !claudeKey.startsWith('sk-ant-')) { showSettingsFeedback('Ungültiges Claude Key Format. Muss mit "sk-ant-" beginnen.', 'error'); return; }

    if (btnSave) { btnSave.disabled = true; btnSave.textContent = 'SPEICHERT...'; }

    try {
        const data = await api.submitSetup(groqKey, openaiKey, dsKey, geminiKey, claudeKey);
        if (data.status === 'success') {
            showSettingsFeedback('✅ Keys gespeichert & AES-256 verschlüsselt!', 'success');
            await loadSettingsStatus();
        } else {
            showSettingsFeedback('❌ Fehler: ' + (data.message || 'Unbekannter Fehler'), 'error');
        }
    } catch (e) {
        showSettingsFeedback('❌ Verbindung zum Core fehlgeschlagen.', 'error');
    } finally {
        if (btnSave) { btnSave.disabled = false; btnSave.textContent = 'KEYS SPEICHERN'; }
    }
}

function forceSetupWizard() {
    const wizard = document.getElementById('setup-wizard');
    const appContainer = document.getElementById('app-container');
    if (wizard && appContainer) {
        ['api-key','deepseek-api-key','openai-api-key','gemini-api-key','claude-api-key'].forEach(id => {
            const el = document.getElementById(id);
            if (el) el.value = '';
        });
        wizard.classList.remove('hidden');
        appContainer.classList.add('hidden');
    }
}

async function resetConfig() {
    if (!confirm('Wirklich alle Keys löschen und neu konfigurieren? Die App wird in den Setup-Modus versetzt.')) return;
    try {
        await api.resetSettings();
        forceSetupWizard();
    } catch (e) {
        showSettingsFeedback('❌ Reset fehlgeschlagen: ' + e.message, 'error');
    }
}

function showSettingsFeedback(msg, type) {
    const feedback = document.getElementById('settings-save-feedback');
    if (!feedback) return;
    feedback.textContent = msg;
    feedback.className = type === 'error' ? 'settings-feedback error' : 'settings-feedback success';
    feedback.classList.remove('hidden');
    setTimeout(() => { feedback.textContent = ''; feedback.classList.add('hidden'); }, 5000);
}

export async function loadCustomPersonasSettings() {
    try {
        const personas = await api.getPersonas();
        window.customPersonasList = personas; // Cache globally on window
        renderCustomPersonasList(personas);
        if (window.refreshPersonasDropdowns) {
            window.refreshPersonasDropdowns();
        }
    } catch (e) {
        console.error('Failed to load custom personas', e);
    }
}

function renderCustomPersonasList(personas) {
    const listContainer = document.getElementById('settings-personas-list');
    if (!listContainer) return;
    
    if (personas.length === 0) {
        listContainer.innerHTML = '<span style="color: var(--vgt-text-dim); font-size: 10px;">Keine benutzerdefinierten Personas angelegt.</span>';
        return;
    }
    
    listContainer.innerHTML = personas.map(p => {
        const idArg = jsArg(p.id);
        return `
        <div style="display:flex; justify-content:space-between; align-items:center; padding: 6px 8px; background: rgba(0,0,0,0.3); border: 1px solid rgba(255,255,255,0.05); border-radius:4px; margin-bottom:4px;">
            <div style="display:flex; flex-direction:column; gap: 2px;">
                <span style="font-size:10px; color: var(--vgt-cyan); font-weight:bold;">${escapeHtml(p.name).toUpperCase()}</span>
                <span style="font-size:8px; color: var(--vgt-text-dim); text-overflow:ellipsis; overflow:hidden; white-space:nowrap; max-width: 240px;">${escapeHtml(p.system_prompt)}</span>
            </div>
            <div style="display:flex; gap: 8px;">
                <button class="cyber-button font-mono" style="padding: 2px 6px; font-size: 8px; background: rgba(0,240,255,0.05); border-color: var(--vgt-cyan); color: var(--vgt-cyan);" onclick="window.editCustomPersona(${idArg})">EDIT</button>
                <button class="cyber-button font-mono" style="padding: 2px 6px; font-size: 8px; background: rgba(255,0,79,0.05); border-color: var(--vgt-red); color: var(--vgt-red);" onclick="window.deleteCustomPersona(${idArg})">DEL</button>
            </div>
        </div>
    `;
    }).join('');
}

async function saveCustomPersona() {
    const idInput = document.getElementById('settings-persona-id');
    const nameInput = document.getElementById('settings-persona-name');
    const promptInput = document.getElementById('settings-persona-prompt');
    if (!nameInput || !promptInput) return;

    const name = nameInput.value.trim();
    const prompt = promptInput.value.trim();

    if (!name || !prompt) {
        alert('Bitte Name und System-Prompt ausfüllen.');
        return;
    }

    const payload = {
        id: idInput ? idInput.value : '',
        name,
        system_prompt: prompt
    };

    try {
        const res = await api.savePersona(payload);
        if (res.status === 'success') {
            clearPersonaForm();
            await loadCustomPersonasSettings();
        } else {
            alert('Fehler beim Speichern der Persona: ' + (res.message || JSON.stringify(res)));
        }
    } catch(e) {
        console.error('Failed to save persona', e);
        alert('Fehler beim Speichern: ' + e.message);
    }
}

function clearPersonaForm() {
    const idInput = document.getElementById('settings-persona-id');
    const nameInput = document.getElementById('settings-persona-name');
    const promptInput = document.getElementById('settings-persona-prompt');
    if (idInput) idInput.value = '';
    if (nameInput) nameInput.value = '';
    if (promptInput) promptInput.value = '';
}

window.editCustomPersona = function(id) {
    if (!window.customPersonasList) return;
    const p = window.customPersonasList.find(x => x.id === id);
    if (!p) return;

    const idInput = document.getElementById('settings-persona-id');
    const nameInput = document.getElementById('settings-persona-name');
    const promptInput = document.getElementById('settings-persona-prompt');
    if (idInput) idInput.value = p.id;
    if (nameInput) nameInput.value = p.name;
    if (promptInput) promptInput.value = p.system_prompt;
};

window.deleteCustomPersona = async function(id) {
    if (!confirm('Diese Persona wirklich löschen?')) return;
    try {
        const res = await api.deletePersona(id);
        if (res.status === 'success') {
            clearPersonaForm();
            await loadCustomPersonasSettings();
        }
    } catch(e) {
        console.error('Failed to delete persona', e);
    }
};

export async function loadProviderHealth() {
    const container = document.getElementById('settings-provider-health-list');
    if (!container) return;
    try {
        const payload = await api.getProviderHealth();
        const providers = Array.isArray(payload.providers) ? payload.providers : [];
        container.replaceChildren();
        if (providers.length === 0) {
            container.innerHTML = '<div style="color:var(--vgt-text-dim);text-align:center;font-size:10px;">Keine Provider-Daten verfügbar.</div>';
            return;
        }
        for (const p of providers) {
            const row = document.createElement('div');
            row.style.cssText = 'display:flex;justify-content:space-between;align-items:center;padding:8px;background:rgba(0,0,0,0.3);border:1px solid rgba(255,255,255,0.05);border-radius:4px;font-size:10px;';

            const name = document.createElement('strong');
            name.textContent = p.provider.toUpperCase();
            name.style.color = p.configured ? 'var(--vgt-cyan)' : 'var(--vgt-text-dim)';

            const status = document.createElement('div');
            status.style.cssText = 'display:flex;align-items:center;gap:6px;';

            const dot = document.createElement('span');
            dot.className = p.reachable ? 'pulse-dot green' : p.configured ? 'pulse-dot red' : 'pulse-dot';
            
            const detail = document.createElement('span');
            detail.style.cssText = 'font-family:var(--font-mono);font-size:8px;color:rgba(255,255,255,0.5);';
            detail.textContent = p.configured ? (p.reachable ? `AKTIV (${p.detail})` : `ERRORED (${p.detail})`) : 'INAKTIV';

            status.append(dot, detail);
            row.append(name, status);
            container.appendChild(row);
        }
    } catch(e) {
        console.error('Failed to load provider health', e);
    }
}
