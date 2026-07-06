// VGT AETHEL // SETTINGS MODULE
// Handles: API Key management (Groq, DeepSeek, OpenAI), setup wizard re-trigger, mounted dirs

import { state } from './state.js';
import * as api from './api.js';

export async function loadSettingsStatus() {
    try {
        const data = await api.getSettings();
        renderSettingsStatus(data);
    } catch (e) {
        console.error('Failed to load settings status', e);
    }
}

function renderSettingsStatus(data) {
    // Groq Key Status
    const groqIndicator = document.getElementById('settings-groq-status');
    const groqLabel = document.getElementById('settings-groq-label');
    if (groqIndicator && groqLabel) {
        if (data.groq_configured) {
            groqIndicator.className = 'pulse-dot green';
            groqLabel.textContent = 'GROQ KEY AKTIV (' + (data.groq_key_preview || '****') + ')';
            groqLabel.style.color = 'var(--vgt-cyan)';
        } else {
            groqIndicator.className = 'pulse-dot red';
            groqLabel.textContent = 'KEIN GROQ KEY GESETZT';
            groqLabel.style.color = 'var(--vgt-red)';
        }
    }

    // DeepSeek Key Status
    const dsIndicator = document.getElementById('settings-deepseek-status');
    const dsLabel = document.getElementById('settings-deepseek-label');
    if (dsIndicator && dsLabel) {
        if (data.deepseek_configured) {
            dsIndicator.className = 'pulse-dot green';
            dsLabel.textContent = 'DEEPSEEK KEY AKTIV (' + (data.deepseek_key_preview || '****') + ')';
            dsLabel.style.color = '#00c8ff';
        } else {
            dsIndicator.className = 'pulse-dot red';
            dsLabel.textContent = 'KEIN DEEPSEEK KEY GESETZT';
            dsLabel.style.color = 'var(--vgt-red)';
        }
    }

    // OpenAI Key Status
    const openaiIndicator = document.getElementById('settings-openai-status');
    const openaiLabel = document.getElementById('settings-openai-label');
    if (openaiIndicator && openaiLabel) {
        if (data.openai_configured) {
            openaiIndicator.className = 'pulse-dot green';
            openaiLabel.textContent = 'OPENAI KEY AKTIV (' + (data.openai_key_preview || '****') + ')';
            openaiLabel.style.color = 'var(--vgt-purple)';
        } else {
            openaiIndicator.className = 'pulse-dot';
            openaiLabel.textContent = 'OPENAI KEY NICHT GESETZT (Optional — für Premium Voice)';
            openaiLabel.style.color = 'var(--vgt-text-dim)';
        }
    }

    // Mounted Dirs
    const dirsContainer = document.getElementById('settings-mounted-dirs');
    if (dirsContainer) {
        const dirs = data.mounted_dirs || [];
        if (dirs.length === 0) {
            dirsContainer.innerHTML = '<span style="color: var(--vgt-text-dim); font-size: 10px;">Keine Verzeichnisse eingebunden.</span>';
        } else {
            dirsContainer.innerHTML = dirs.map(d =>
                `<div style="display:flex; justify-content:space-between; align-items:center; padding: 6px 8px; background: rgba(0,0,0,0.3); border: 1px solid rgba(255,255,255,0.05); border-radius:4px; margin-bottom:4px;">
                    <span style="font-size:10px; color: var(--vgt-cyan); font-family: var(--font-mono);">${d}</span>
                </div>`
            ).join('');
        }
    }

    // Clear input fields
    const groqInput = document.getElementById('settings-groq-input');
    const dsInput = document.getElementById('settings-deepseek-input');
    const openaiInput = document.getElementById('settings-openai-input');
    if (groqInput) groqInput.value = '';
    if (dsInput) dsInput.value = '';
    if (openaiInput) openaiInput.value = '';

    // Clear save feedback
    const feedback = document.getElementById('settings-save-feedback');
    if (feedback) {
        feedback.textContent = '';
        feedback.className = 'hidden';
    }
}

export function setupSettingsUIEvents() {
    // Save Keys Button
    const btnSave = document.getElementById('settings-btn-save');
    if (btnSave) {
        btnSave.addEventListener('click', saveSettingsKeys);
    }

    // Force Setup Wizard Button
    const btnForceSetup = document.getElementById('settings-btn-force-setup');
    if (btnForceSetup) {
        btnForceSetup.addEventListener('click', forceSetupWizard);
    }

    // Reset Config Button
    const btnReset = document.getElementById('settings-btn-reset');
    if (btnReset) {
        btnReset.addEventListener('click', resetConfig);
    }
}

async function saveSettingsKeys() {
    const groqInput = document.getElementById('settings-groq-input');
    const dsInput = document.getElementById('settings-deepseek-input');
    const openaiInput = document.getElementById('settings-openai-input');
    const btnSave = document.getElementById('settings-btn-save');

    const groqKey = groqInput ? groqInput.value.trim() : '';
    const dsKey = dsInput ? dsInput.value.trim() : '';
    const openaiKey = openaiInput ? openaiInput.value.trim() : '';

    // Validate: at least one AI key must be present
    if (!groqKey && !dsKey) {
        showSettingsFeedback('Mindestens ein Key erforderlich: Groq (gsk_...) oder DeepSeek (sk-...).', 'error');
        return;
    }
    if (groqKey && !groqKey.startsWith('gsk_')) {
        showSettingsFeedback('Ungültiges Groq Key Format. Muss mit "gsk_" beginnen.', 'error');
        return;
    }
    if (dsKey && !dsKey.startsWith('sk-')) {
        showSettingsFeedback('Ungültiges DeepSeek Key Format. Muss mit "sk-" beginnen.', 'error');
        return;
    }
    if (openaiKey && !openaiKey.startsWith('sk-')) {
        showSettingsFeedback('Ungültiges OpenAI Key Format. Muss mit "sk-" beginnen.', 'error');
        return;
    }

    if (btnSave) {
        btnSave.disabled = true;
        btnSave.textContent = 'SPEICHERT...';
    }

    try {
        const data = await api.submitSetup(groqKey, openaiKey, dsKey);
        if (data.status === 'success') {
            showSettingsFeedback('✅ Keys erfolgreich gespeichert! System bereit.', 'success');
            await loadSettingsStatus();
        } else {
            showSettingsFeedback('❌ Fehler: ' + (data.message || 'Unbekannter Fehler'), 'error');
        }
    } catch (e) {
        showSettingsFeedback('❌ Verbindung zum Core fehlgeschlagen.', 'error');
    } finally {
        if (btnSave) {
            btnSave.disabled = false;
            btnSave.textContent = 'KEYS SPEICHERN';
        }
    }
}

function forceSetupWizard() {
    const wizard = document.getElementById('setup-wizard');
    const appContainer = document.getElementById('app-container');
    if (wizard && appContainer) {
        // Clear old values
        const apiKeyInput = document.getElementById('api-key');
        const dsKeyInput = document.getElementById('deepseek-api-key');
        const openaiInput = document.getElementById('openai-api-key');
        if (apiKeyInput) apiKeyInput.value = '';
        if (dsKeyInput) dsKeyInput.value = '';
        if (openaiInput) openaiInput.value = '';

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
    setTimeout(() => {
        feedback.textContent = '';
        feedback.classList.add('hidden');
    }, 5000);
}
