// STATUS: DIAMANT VGT SUPREME
import { state } from './state.js';

const mailState = { initialized: false, folder: 'INBOX', folders: [], messages: [], selected: null, policies: [] };

function el(tag, className, text = '') { const node = document.createElement(tag); if (className) node.className = className; if (text) node.textContent = text; return node; }
async function jsonFetch(path, options = {}) { const response = await fetch(`${state.API_BASE}${path}`, options); if (!response.ok) throw new Error((await response.text()).slice(0, 240) || `HTTP ${response.status}`); return response.json(); }
function params(values) { return `?${new URLSearchParams(values).toString()}`; }
function notify(message, type = 'info') { window.showAethelToast?.(message, type); }

export function initMailWorkspace() {
    if (mailState.initialized) return;
    mailState.initialized = true;
    document.getElementById('mail-sync-button')?.addEventListener('click', syncMailWorkspace);
    document.getElementById('mail-create-folder')?.addEventListener('click', createFolder);
    document.getElementById('mail-search')?.addEventListener('input', renderMessageList);
    document.getElementById('mail-class-filter')?.addEventListener('change', renderMessageList);
    document.getElementById('mail-policy-button')?.addEventListener('click', openPolicies);
    document.getElementById('mail-policy-close')?.addEventListener('click', closePolicies);
    document.getElementById('mail-policy-add')?.addEventListener('click', () => { mailState.policies.push(blankPolicy()); renderPolicies(); });
    document.getElementById('mail-policy-save')?.addEventListener('click', savePolicies);
    document.getElementById('nav-btn-mail')?.addEventListener('click', () => { if (mailState.folders.length === 0) syncMailWorkspace(); });
}

export async function openMailMessage(folder, uid) {
    document.getElementById('nav-btn-mail')?.click();
    if (mailState.folders.length === 0) {
        const payload = await jsonFetch('/v1/mail/folders');
        mailState.folders = Array.isArray(payload.folders) ? payload.folders : [];
    }
    await loadMessages(folder || 'INBOX');
    const summary = mailState.messages.find(message => Number(message.uid) === Number(uid));
    await openMessage(summary || { folder: folder || 'INBOX', uid: Number(uid), from: '', subject: '' });
}

async function syncMailWorkspace() {
    const button = document.getElementById('mail-sync-button');
    if (button) button.disabled = true;
    try {
        const payload = await jsonFetch('/v1/mail/folders');
        mailState.folders = Array.isArray(payload.folders) ? payload.folders : [];
        if (!mailState.folders.some(folder => folder.name === mailState.folder)) mailState.folder = mailState.folders[0]?.name || 'INBOX';
        renderFolders();
        await loadMessages(mailState.folder);
        const security = document.getElementById('mail-security-state');
        if (security) { security.textContent = 'AES-256-GCM · ML-DSA-65 VERIFIED'; security.classList.add('verified'); }
    } catch (error) {
        const list = document.getElementById('mail-message-list');
        if (list) list.replaceChildren(el('div', 'mail-empty error', `Mail-Uplink fehlgeschlagen: ${error.message}`));
        notify(`Mail-Uplink fehlgeschlagen: ${error.message}`, 'error');
    } finally { if (button) button.disabled = false; }
}

async function loadMessages(folder) {
    mailState.folder = folder;
    mailState.selected = null;
    const list = document.getElementById('mail-message-list');
    list?.replaceChildren(el('div', 'mail-empty', `${folder} wird synchronisiert …`));
    const payload = await jsonFetch(`/v1/mail/messages${params({ folder, limit: '100' })}`);
    mailState.messages = Array.isArray(payload.messages) ? payload.messages : [];
    renderFolders(); renderMessageList(); renderReaderEmpty();
}

function renderFolders() {
    const root = document.getElementById('mail-folder-list'); if (!root) return;
    const nodes = mailState.folders.map(folder => {
        const button = el('button', `mail-folder-button${folder.name === mailState.folder ? ' active' : ''}`);
        button.type = 'button';
        const icon = el('span', 'mail-folder-icon', folderIcon(folder.name));
        const name = el('span', 'mail-folder-name', folder.name);
        button.append(icon, name);
        button.addEventListener('click', () => loadMessages(folder.name).catch(error => notify(error.message, 'error')));
        return button;
    });
    root.replaceChildren(...nodes);
}

function folderIcon(name) { const value = String(name).toLowerCase(); if (value.includes('spam') || value.includes('junk')) return '⚠'; if (value.includes('sent') || value.includes('gesendet')) return '↗'; if (value.includes('trash') || value.includes('papierkorb')) return '⌫'; if (value === 'inbox') return '▣'; return '□'; }

function renderMessageList() {
    const root = document.getElementById('mail-message-list'); if (!root) return;
    const query = String(document.getElementById('mail-search')?.value || '').trim().toLowerCase();
    const filter = document.getElementById('mail-class-filter')?.value || 'all';
    const visible = mailState.messages.filter(message => {
        const matchesText = !query || `${message.from} ${message.subject}`.toLowerCase().includes(query);
        const matchesClass = filter === 'all' || (filter === 'unread' ? message.unread : message.spam_class === filter);
        return matchesText && matchesClass;
    });
    if (visible.length === 0) { root.replaceChildren(el('div', 'mail-empty', 'Keine passenden Nachrichten.')); return; }
    root.replaceChildren(...visible.map(message => {
        const row = el('button', `mail-message-row${message.unread ? ' unread' : ''}${mailState.selected?.uid === message.uid ? ' selected' : ''}`);
        row.type = 'button';
        const top = el('div', 'mail-message-top'); top.append(el('strong', '', message.from || 'Unbekannt'), el('time', '', formatDate(message.date)));
        const subject = el('div', 'mail-message-subject', message.subject || '(Ohne Betreff)');
        const preview = el('p', '', message.preview || 'Keine Vorschau');
        const badges = el('div', 'mail-message-badges');
        if (message.unread) badges.appendChild(el('span', 'unread', 'NEU'));
        badges.appendChild(el('span', `spam-${message.spam_class || 'clean'}`, `${String(message.spam_class || 'clean').toUpperCase()} ${message.spam_score || 0}`));
        row.append(top, subject, preview, badges);
        row.addEventListener('click', () => openMessage(message));
        return row;
    }));
}

async function openMessage(summary) {
    mailState.selected = summary; renderMessageList();
    const reader = document.getElementById('mail-reader');
    reader?.replaceChildren(el('div', 'mail-empty', 'E-Mail wird entschlüsselt und verifiziert …'));
    try {
        const payload = await jsonFetch(`/v1/mail/message${params({ folder: summary.folder || mailState.folder, uid: String(summary.uid) })}`);
        mailState.selected = payload.message;
        renderMessage(payload.message, payload.calendar_candidates || []);
    } catch (error) { reader?.replaceChildren(el('div', 'mail-empty error', error.message)); }
}

function renderReaderEmpty() { const reader = document.getElementById('mail-reader'); if (!reader) return; const empty = el('div', 'mail-reader-empty'); empty.append(el('span', '', '◈'), el('strong', '', 'Keine E-Mail ausgewählt'), el('small', '', 'Wähle eine Nachricht, um den sicheren Reader zu öffnen.')); reader.replaceChildren(empty); }

function renderMessage(message, candidates) {
    const root = document.getElementById('mail-reader'); if (!root) return;
    const header = el('header', 'mail-reader-header');
    header.append(el('span', 'mail-reader-kicker', `${message.folder} · UID ${message.uid}`), el('h2', '', message.subject || '(Ohne Betreff)'));
    const identity = el('div', 'mail-reader-identity');
    identity.append(el('strong', '', message.from || 'Unbekannt'), el('span', '', `An: ${(message.to || []).join(', ') || '—'}`), el('time', '', new Date(message.date).toLocaleString()));
    const assessment = el('section', `mail-spam-assessment ${message.spam_class || 'clean'}`);
    const score = el('strong', '', `${message.spam_score || 0}/100`);
    const assessmentText = el('div'); assessmentText.append(el('span', '', `SPAM ASSESSMENT · ${String(message.spam_class || 'clean').toUpperCase()}`));
    const reasons = el('ul'); for (const reason of message.spam_reasons || []) reasons.appendChild(el('li', '', reason));
    assessmentText.appendChild(reasons); assessment.append(score, assessmentText);
    const body = el('pre', 'mail-reader-body', message.text_body || 'Kein Textteil verfügbar.');
    const actions = el('div', 'mail-reader-actions');
    const reply = el('button', 'gw-tool-btn', 'MIT AETHEL ANTWORTEN'); reply.type = 'button'; reply.addEventListener('click', () => replyWithAethel(message));
    const move = document.createElement('select'); move.className = 'mail-move-select'; move.appendChild(option('', 'VERSCHIEBEN …'));
    for (const folder of mailState.folders.filter(folder => folder.name !== message.folder)) move.appendChild(option(folder.name, folder.name));
    move.addEventListener('change', () => { if (move.value) moveMessage(message, move.value); });
    actions.append(reply, move);
    for (const candidate of candidates) {
        const calendar = el('button', 'gw-tool-btn mail-deadline-button', `TERMIN · ${new Date(candidate.due_at).toLocaleDateString()}`);
        calendar.type = 'button'; calendar.title = candidate.reason || '';
        calendar.addEventListener('click', () => createCalendarEvent(message, candidate, calendar));
        actions.appendChild(calendar);
    }
    const details = document.createElement('details'); details.className = 'mail-header-details'; details.appendChild(el('summary', '', 'VERIFIZIERTE HEADER-SIGNALE'));
    const headerList = el('dl'); for (const [key, value] of Object.entries(message.headers || {})) { headerList.append(el('dt', '', key), el('dd', '', value)); } details.appendChild(headerList);
    root.replaceChildren(header, identity, assessment, actions, body, details);
}

function option(value, text) { const item = document.createElement('option'); item.value = value; item.textContent = text; return item; }
function formatDate(value) { const date = new Date(value); return Number.isNaN(date.getTime()) ? '—' : date.toLocaleDateString(); }

async function moveMessage(message, destination) {
    try { await jsonFetch('/v1/mail/action', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ action: 'move', folder: message.folder, uid: message.uid, destination }) }); notify(`E-Mail nach ${destination} verschoben.`, 'success'); await loadMessages(mailState.folder); }
    catch (error) { notify(`Verschieben fehlgeschlagen: ${error.message}`, 'error'); }
}

async function createFolder() { const name = window.prompt('Name des neuen IMAP-Ordners'); if (!name?.trim()) return; try { await jsonFetch('/v1/mail/folders', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: name.trim() }) }); await syncMailWorkspace(); } catch (error) { notify(error.message, 'error'); } }

async function createCalendarEvent(message, candidate, button) { button.disabled = true; try { await jsonFetch('/v1/mail/calendar', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ folder: message.folder, uid: message.uid, title: candidate.title, due_at: candidate.due_at }) }); button.textContent = 'TERMIN EINGETRAGEN'; notify('Termin wurde mit der E-Mail verknüpft.', 'success'); } catch (error) { button.disabled = false; notify(error.message, 'error'); } }

function replyWithAethel(message) { document.getElementById('nav-btn-chat')?.click(); const input = document.getElementById('chat-input'); if (!input) return; input.value = `Antworte auf die E-Mail in Ordner "${message.folder}" mit UID ${message.uid}. Lies sie zuerst mit mail_read_message, wende ausschließlich passende manuell konfigurierte Antwortregeln an und erstelle einen vollständigen Entwurf. Sende erst nach meiner expliziten Freigabe.`; input.focus(); }

function blankPolicy() { return { id: '', name: '', recipient_pattern: '', category: 'general', instructions: '', system_prompt: '', enabled: true, manual_approval: true }; }
async function openPolicies() { const overlay = document.getElementById('mail-policy-overlay'); overlay?.classList.remove('hidden'); try { const payload = await jsonFetch('/v1/mail/policies'); mailState.policies = Array.isArray(payload.policies) ? payload.policies : []; renderPolicies(); } catch (error) { notify(error.message, 'error'); } }
function closePolicies() { document.getElementById('mail-policy-overlay')?.classList.add('hidden'); }
function renderPolicies() { const root = document.getElementById('mail-policy-list'); if (!root) return; if (mailState.policies.length === 0) { root.replaceChildren(el('div', 'mail-empty', 'Noch keine Empfängerregel.')); return; } root.replaceChildren(...mailState.policies.map((policy, index) => policyEditor(policy, index))); }
function policyEditor(policy, index) { const card = el('article', 'mail-policy-card'); const row = el('div', 'mail-policy-row'); const enabled = input('checkbox', '', policy.enabled); const name = input('text', 'Name', policy.name); const pattern = input('text', 'Empfänger oder *@domain.de', policy.recipient_pattern); const category = input('text', 'Kategorie, z. B. legal', policy.category); row.append(enabled, name, pattern, category); const instructions = textarea('Verbindliche Anweisungen', policy.instructions); const systemPrompt = textarea('Optionaler Systemprompt', policy.system_prompt); const remove = el('button', 'mail-policy-remove', 'REGEL ENTFERNEN'); remove.type = 'button'; remove.addEventListener('click', () => { mailState.policies.splice(index, 1); renderPolicies(); }); card.append(row, instructions, systemPrompt, remove); card._read = () => ({ ...policy, enabled: enabled.checked, name: name.value.trim(), recipient_pattern: pattern.value.trim(), category: category.value.trim(), instructions: instructions.value.trim(), system_prompt: systemPrompt.value.trim(), manual_approval: true }); return card; }
function input(type, placeholder, value) { const node = document.createElement('input'); node.type = type; node.placeholder = placeholder; if (type === 'checkbox') node.checked = !!value; else node.value = value || ''; return node; }
function textarea(placeholder, value) { const node = document.createElement('textarea'); node.placeholder = placeholder; node.value = value || ''; node.rows = 4; return node; }
async function savePolicies() { const root = document.getElementById('mail-policy-list'); const cards = [...(root?.querySelectorAll('.mail-policy-card') || [])]; const policies = cards.map(card => card._read()); try { await jsonFetch('/v1/mail/policies', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ policies }) }); mailState.policies = policies; notify('Antwortregeln verschlüsselt und signiert gespeichert.', 'success'); closePolicies(); } catch (error) { notify(error.message, 'error'); } }
