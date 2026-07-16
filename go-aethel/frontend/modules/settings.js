// VGT AETHEL // SETTINGS MODULE
// Handles: API Key management (Groq, DeepSeek, OpenAI, Gemini, Claude), setup wizard re-trigger, mounted dirs

import { state } from './state.js';
import * as api from './api.js';

export async function loadSettingsStatus() {
    try {
        const data = await api.getSettings();
        renderSettingsStatus(data);
        await loadCustomPersonasSettings();
        await loadProviderHealth();
		await loadMailConfig();
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
            label.classList.add('is-configured');
        } else {
            indicator.className = 'pulse-dot red';
            label.textContent = row.offText;
            label.classList.remove('is-configured');
        }
    }

    const osLabel = document.getElementById('settings-os-label');
    if (osLabel && data.os) osLabel.textContent = data.os.toUpperCase() + ' (AUTOMATISCH ERKANNT)';

    const dirsContainer = document.getElementById('settings-mounted-dirs');
    if (dirsContainer) {
        const dirs = data.mounted_dirs || [];
        dirsContainer.replaceChildren();
        if (dirs.length === 0) {
            const empty = document.createElement('span');
            empty.className = 'settings-empty-state';
            empty.textContent = 'Keine Verzeichnisse eingebunden.';
            dirsContainer.appendChild(empty);
        } else {
            dirs.forEach(directory => {
                const row = document.createElement('div');
                row.className = 'settings-directory-row';
                const path = document.createElement('span');
                path.textContent = String(directory);
                row.appendChild(path);
                dirsContainer.appendChild(row);
            });
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
    consolidateProviderSettings();
	installMailSettingsCard();
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

function installMailSettingsCard() {
	if (document.getElementById('settings-mail-card')) return;
	const layout = document.querySelector('.settings-layout .settings-col:last-child');
	if (!layout) return;
	const card = document.createElement('section');
	card.id = 'settings-mail-card';
	card.className = 'settings-card glass-card';
	const title = document.createElement('h4');
	title.className = 'ds-section-title';
	title.textContent = 'Sicheres E-Mail-Konto · IMAP / SMTP';
	const hint = document.createElement('p');
	hint.className = 'settings-hint';
	hint.textContent = 'IMAP liest den Posteingang, SMTP sendet. Passwort und Konfiguration werden lokal verschlüsselt; SMTP-Versand durch Aethel benötigt immer eine Einmalfreigabe.';
	const status = document.createElement('div');
	status.id = 'settings-mail-status';
	status.className = 'settings-feedback';
	status.textContent = 'NICHT KONFIGURIERT';
	const fields = document.createElement('div');
	fields.className = 'settings-provider-key-grid';
	const addField = (id, labelText, type = 'text', placeholder = '') => {
		const wrap = document.createElement('label');
		wrap.className = 'settings-field ds-field';
		const label = document.createElement('span');
		label.className = 'ds-label';
		label.textContent = labelText;
		const input = document.createElement('input');
		input.id = id;
		input.type = type;
		input.className = 'settings-input';
		input.placeholder = placeholder;
		input.autocomplete = 'off';
		wrap.append(label, input);
		fields.append(wrap);
		return input;
	};
	addField('settings-mail-address', 'E-Mail-Adresse', 'email', 'name@example.com');
	addField('settings-mail-name', 'Anzeigename', 'text', 'Aethel Operator');
	addField('settings-mail-user', 'Benutzername', 'text', 'name@example.com');
	addField('settings-mail-password', 'Passwort / App-Passwort', 'password', 'leer = unverändert');
	addField('settings-mail-imap-host', 'IMAP Host', 'text', 'imap.example.com');
	addField('settings-mail-imap-port', 'IMAP TLS Port', 'number', '993');
	addField('settings-mail-smtp-host', 'SMTP Host', 'text', 'smtp.example.com');
	addField('settings-mail-smtp-port', 'SMTP Port', 'number', '587');
	const securityWrap = document.createElement('label');
	securityWrap.className = 'settings-field ds-field';
	const securityLabel = document.createElement('span');
	securityLabel.className = 'ds-label';
	securityLabel.textContent = 'SMTP Transport';
	const securitySelect = document.createElement('select');
	securitySelect.id = 'settings-mail-security';
	securitySelect.className = 'settings-input';
	[['starttls', 'STARTTLS'], ['tls', 'TLS (Port 465)']].forEach(([value, text]) => {
		const option = document.createElement('option');
		option.value = value;
		option.textContent = text;
		securitySelect.append(option);
	});
	securityWrap.append(securityLabel, securitySelect);
	fields.append(securityWrap);
	const actions = document.createElement('div');
	actions.className = 'persona-registry-actions';
	const save = document.createElement('button');
	save.type = 'button';
	save.className = 'cyber-button font-mono';
	save.textContent = 'KONTO SPEICHERN';
	save.addEventListener('click', () => saveMailConfig());
	const test = document.createElement('button');
	test.type = 'button';
	test.className = 'cyber-button font-mono';
	test.textContent = 'IMAP TESTEN';
	test.addEventListener('click', () => testMailConfig());
	const remove = document.createElement('button');
	remove.type = 'button';
	remove.className = 'cyber-button font-mono ds-btn-danger';
	remove.textContent = 'KONTO ENTFERNEN';
	remove.addEventListener('click', () => removeMailConfig());
	actions.append(save, test, remove);
	card.append(title, hint, status, fields, actions);
	layout.append(card);
}

async function loadMailConfig() {
	const status = document.getElementById('settings-mail-status');
	if (!status) return;
	try {
		const response = await fetch(`${state.API_BASE}/v1/mail/config`);
		if (!response.ok) throw new Error(await response.text());
		const payload = await response.json();
		if (!payload.configured || !payload.config) {
			status.textContent = 'NICHT KONFIGURIERT';
			status.className = 'settings-feedback';
			return;
		}
		const config = payload.config;
		const values = {
			'settings-mail-address': config.email,
			'settings-mail-name': config.display_name,
			'settings-mail-user': config.username,
			'settings-mail-imap-host': config.imap_host,
			'settings-mail-imap-port': config.imap_port,
			'settings-mail-smtp-host': config.smtp_host,
			'settings-mail-smtp-port': config.smtp_port,
			'settings-mail-security': config.smtp_security,
		};
		Object.entries(values).forEach(([id, value]) => { const element = document.getElementById(id); if (element) element.value = value ?? ''; });
		const password = document.getElementById('settings-mail-password');
		if (password) password.value = '';
		status.textContent = 'KONFIGURIERT · TLS GESCHÜTZT';
		status.className = 'settings-feedback success';
	} catch (error) {
		status.textContent = `STATUSFEHLER: ${error.message}`;
		status.className = 'settings-feedback error';
	}
}

function mailConfigPayload() {
	const value = id => document.getElementById(id)?.value.trim() || '';
	const password = document.getElementById('settings-mail-password')?.value || '';
	return {
		config: {
			enabled: true,
			email: value('settings-mail-address'),
			display_name: value('settings-mail-name'),
			username: value('settings-mail-user'),
			imap_host: value('settings-mail-imap-host'),
			imap_port: Number(value('settings-mail-imap-port') || 993),
			smtp_host: value('settings-mail-smtp-host'),
			smtp_port: Number(value('settings-mail-smtp-port') || 587),
			smtp_security: value('settings-mail-security') || 'starttls',
		},
		password,
	};
}

async function saveMailConfig() {
	const status = document.getElementById('settings-mail-status');
	try {
		const response = await fetch(`${state.API_BASE}/v1/mail/config`, { method: 'PUT', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(mailConfigPayload()) });
		if (!response.ok) throw new Error((await response.text()).trim());
		if (status) { status.textContent = 'KONTO VERSCHLÜSSELT GESPEICHERT'; status.className = 'settings-feedback success'; }
		await loadMailConfig();
	} catch (error) {
		if (status) { status.textContent = `SPEICHERN FEHLGESCHLAGEN: ${error.message}`; status.className = 'settings-feedback error'; }
	}
}

async function testMailConfig() {
	const status = document.getElementById('settings-mail-status');
	if (status) { status.textContent = 'IMAP TLS TEST LÄUFT…'; status.className = 'settings-feedback'; }
	try {
		const response = await fetch(`${state.API_BASE}/v1/mail/test`, { method: 'POST' });
		if (!response.ok) throw new Error((await response.text()).trim());
		if (status) { status.textContent = 'IMAP VERBINDUNG GESUND'; status.className = 'settings-feedback success'; }
	} catch (error) {
		if (status) { status.textContent = `IMAP TEST FEHLGESCHLAGEN: ${error.message}`; status.className = 'settings-feedback error'; }
	}
}

async function removeMailConfig() {
	if (!confirm('E-Mail-Konto und lokal gespeichertes Passwort wirklich entfernen?')) return;
	const response = await fetch(`${state.API_BASE}/v1/mail/config`, { method: 'DELETE' });
	if (!response.ok) return;
	await loadMailConfig();
}

function consolidateProviderSettings() {
    if (document.getElementById('settings-provider-operations')) return;
    const credentials = document.getElementById('settings-credentials-card');
    const health = document.getElementById('settings-health-card');
    const keys = document.getElementById('settings-keys-card');
    const hostColumn = credentials?.parentElement;
    if (!credentials || !health || !keys || !hostColumn) return;

    const card = document.createElement('section');
    card.id = 'settings-provider-operations';
    card.className = 'settings-card settings-provider-operations glass-card';
    card.setAttribute('aria-labelledby', 'settings-provider-operations-title');
    const header = document.createElement('header');
    header.className = 'settings-provider-operations-header';
    const heading = document.createElement('div');
    const kicker = document.createElement('span');
    kicker.className = 'vgt-eyebrow';
    kicker.textContent = 'MODEL FABRIC // LIVE ROUTING';
    const title = document.createElement('h3');
    title.id = 'settings-provider-operations-title';
    title.textContent = 'Provider-Gesundheit, Zugänge & Fallback';
    const intro = document.createElement('p');
    intro.className = 'settings-hint';
    intro.textContent = 'Einsatzbereit bedeutet: Zugang konfiguriert, Live-Probe erfolgreich und Modellfähigkeiten kompatibel. Aethel fällt ausschließlich auf registrierte, erreichbare Alternativen zurück.';
    heading.append(kicker, title, intro);
    const refresh = document.createElement('button');
    refresh.type = 'button';
    refresh.className = 'gw-tool-btn';
    refresh.textContent = 'LIVE-STATUS PRÜFEN';
    refresh.addEventListener('click', () => loadProviderHealth());
    header.append(heading, refresh);

    const matrix = document.createElement('div');
    matrix.className = 'settings-provider-matrix';
    const makePane = (paneTitle, child) => {
        const pane = document.createElement('div');
        pane.className = 'settings-provider-pane';
        const label = document.createElement('h4');
        label.textContent = paneTitle;
        pane.append(label);
        if (child) pane.append(child);
        return pane;
    };
    matrix.append(
        makePane('Zugangsstatus', credentials.querySelector('.settings-status-list')),
        makePane('Erreichbarkeit & Routing', document.getElementById('settings-provider-health-list'))
    );

    const keyGrid = document.createElement('div');
    keyGrid.className = 'settings-provider-key-grid';
    keys.querySelectorAll('.settings-field').forEach(field => keyGrid.append(field));
    const keyHint = keys.querySelector('.settings-hint');
    const feedback = document.getElementById('settings-save-feedback');
    const save = document.getElementById('settings-btn-save');
    card.append(header, matrix);
    if (keyHint) card.append(keyHint);
    card.append(keyGrid);
    if (feedback) card.append(feedback);
    if (save) card.append(save);
    hostColumn.insertBefore(card, credentials);
    credentials.remove();
    health.remove();
    keys.remove();
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
    listContainer.replaceChildren();

    if (personas.length === 0) {
        const empty = document.createElement('div');
        empty.className = 'persona-registry-empty';
        empty.textContent = 'Keine benutzerdefinierten Personas angelegt.';
        listContainer.appendChild(empty);
        return;
    }

    for (const persona of personas) {
        const card = document.createElement('article');
        card.className = 'persona-registry-item';
        const identity = document.createElement('div');
        identity.className = 'persona-registry-identity';
        const name = document.createElement('strong');
        name.textContent = String(persona.name || 'Unbenannte Persona').toUpperCase();
        const prompt = document.createElement('span');
        prompt.textContent = String(persona.system_prompt || 'Kein System-Prompt hinterlegt.');
        prompt.title = prompt.textContent;
        identity.append(name, prompt);

        const actions = document.createElement('div');
        actions.className = 'persona-registry-actions';
        const edit = document.createElement('button');
        edit.type = 'button';
        edit.className = 'cyber-button font-mono persona-registry-edit';
        edit.textContent = 'BEARBEITEN';
        edit.addEventListener('click', () => editCustomPersona(persona.id));
        const remove = document.createElement('button');
        remove.type = 'button';
        remove.className = 'cyber-button font-mono persona-registry-delete';
        remove.textContent = 'LÖSCHEN';
        remove.addEventListener('click', () => deleteCustomPersona(persona.id));
        actions.append(edit, remove);
        card.append(identity, actions);
        listContainer.appendChild(card);
    }
}

async function saveCustomPersona() {
    const idInput = document.getElementById('settings-persona-id');
    const nameInput = document.getElementById('settings-persona-name');
    const promptInput = document.getElementById('settings-persona-prompt');
    if (!nameInput || !promptInput) return;

    const name = nameInput.value.trim();
    const prompt = promptInput.value.trim();

    if (!name || !prompt) {
        showPersonaFeedback('Bitte Name und System-Prompt ausfüllen.', 'error');
        (name ? promptInput : nameInput).focus();
        return;
    }
    if (name.length > 120 || prompt.length > 12000) {
        showPersonaFeedback('Persona überschreitet die zulässige Länge.', 'error');
        return;
    }

    const payload = {
        id: idInput ? idInput.value : '',
        name,
        system_prompt: prompt
    };

    const saveButton = document.getElementById('settings-persona-btn-save');
    if (saveButton) saveButton.disabled = true;
    try {
        const res = await api.savePersona(payload);
        if (res.status === 'success') {
            clearPersonaForm();
            await loadCustomPersonasSettings();
            showPersonaFeedback('Persona gespeichert und systemweit synchronisiert.', 'success');
        } else {
            throw new Error(res.message || 'Server hat die Persona nicht bestätigt.');
        }
    } catch(e) {
        console.error('Failed to save persona', e);
        showPersonaFeedback(`Serverfehler beim Speichern: ${e.message}`, 'error');
    } finally {
        if (saveButton) saveButton.disabled = false;
    }
}

function showPersonaFeedback(message, type = 'info') {
    const feedback = document.getElementById('settings-persona-feedback');
    if (feedback) {
        feedback.textContent = message;
        feedback.className = `persona-registry-feedback persona-feedback-${type}`;
        feedback.hidden = false;
    }
    window.showAethelToast?.(message, type);
}

function clearPersonaForm() {
    const idInput = document.getElementById('settings-persona-id');
    const nameInput = document.getElementById('settings-persona-name');
    const promptInput = document.getElementById('settings-persona-prompt');
    if (idInput) idInput.value = '';
    if (nameInput) nameInput.value = '';
    if (promptInput) promptInput.value = '';
    const feedback = document.getElementById('settings-persona-feedback');
    if (feedback) {
        feedback.textContent = '';
        feedback.hidden = true;
    }
}

function editCustomPersona(id) {
    if (!window.customPersonasList) return;
    const p = window.customPersonasList.find(x => x.id === id);
    if (!p) return;

    const idInput = document.getElementById('settings-persona-id');
    const nameInput = document.getElementById('settings-persona-name');
    const promptInput = document.getElementById('settings-persona-prompt');
    if (idInput) idInput.value = p.id;
    if (nameInput) nameInput.value = p.name;
    if (promptInput) promptInput.value = p.system_prompt;
    nameInput?.focus();
    showPersonaFeedback(`Persona „${p.name}“ wird bearbeitet.`, 'info');
}

async function deleteCustomPersona(id) {
    const persona = window.customPersonasList?.find(item => item.id === id);
    if (!confirm(`Persona „${persona?.name || id}“ wirklich löschen?`)) return;
    try {
        const res = await api.deletePersona(id);
        if (res.status === 'success') {
            clearPersonaForm();
            await loadCustomPersonasSettings();
            showPersonaFeedback('Persona gelöscht und Registries aktualisiert.', 'success');
        } else {
            throw new Error(res.message || 'Server hat das Löschen nicht bestätigt.');
        }
    } catch(e) {
        console.error('Failed to delete persona', e);
        showPersonaFeedback(`Persona konnte nicht gelöscht werden: ${e.message}`, 'error');
    }
}

export async function loadProviderHealth() {
    const container = document.getElementById('settings-provider-health-list');
    if (!container) return;
    try {
        const payload = await api.getProviderHealth();
        const providers = Array.isArray(payload.providers) ? payload.providers : [];
        container.replaceChildren();
        if (providers.length === 0) {
            const empty = document.createElement('div');
            empty.className = 'settings-empty-state';
            empty.textContent = 'Keine Provider-Daten verfügbar.';
            container.appendChild(empty);
            return;
        }
        providers.forEach((p, index) => {
            const row = document.createElement('div');
            row.className = `settings-provider-health-row ${p.reachable ? 'is-ready' : p.configured ? 'is-error' : 'is-offline'}`;

            const name = document.createElement('strong');
            name.textContent = p.provider.toUpperCase();
            const priority = document.createElement('small');
            priority.textContent = p.reachable ? `ROUTE ${String(index + 1).padStart(2, '0')}` : 'NICHT ROUTBAR';

            const status = document.createElement('div');
            status.className = 'settings-provider-health-state';

            const dot = document.createElement('span');
            dot.className = p.reachable ? 'pulse-dot green' : p.configured ? 'pulse-dot red' : 'pulse-dot';
            
            const detail = document.createElement('span');
            detail.textContent = p.configured ? (p.reachable ? `AKTIV (${p.detail})` : `ERRORED (${p.detail})`) : 'INAKTIV';

            const identity = document.createElement('div');
            identity.className = 'settings-provider-health-identity';
            identity.append(name, priority);
            status.append(dot, detail);
            row.append(identity, status);
            container.appendChild(row);
        });
    } catch(e) {
        console.error('Failed to load provider health', e);
    }
}
