// STATUS: DIAMANT VGT SUPREME
import * as api from './api.js';

const delivered = new Set();
let monitorStarted = false;
let activeNotice = null;
let pendingQueue = [];

function node(tag, className, text = '') {
    const value = document.createElement(tag);
    if (className) value.className = className;
    if (text) value.textContent = text;
    return value;
}

function priorityLabel(priority) {
    if (priority === 'critical') return 'SOFORT-AKTION';
    if (priority === 'high') return 'AUFMERKSAMKEIT';
    return 'PERSONAL OPERATIONS';
}

function formatNoticeBody(notice) {
    if (notice.metadata?.format !== 'weather_json') return notice.body || '';
    try {
        const weather = JSON.parse(notice.body);
        return `${weather.summary || 'Wetterlage'} · ${Number(weather.temperature_c).toFixed(1)} °C · Wind ${Number(weather.wind_speed_kmh).toFixed(1)} km/h\nStand: ${weather.observed_at || 'aktuell'}`;
    } catch (_) {
        return notice.body || '';
    }
}

async function operationAction(action, payload) {
    await api.personalOperationAction(action, payload);
}

function closeOverlay(overlay) {
    activeNotice = null;
    overlay.remove();
    window.setTimeout(showNextNotice, 180);
}

function openNotice(notice) {
    if (notice.kind === 'alarm') {
        activeNotice = notice;
        import('./alarm_engine.js').then(module => module.startAIAlarmSequence(false, async () => {
            await operationAction('acknowledge', { id: notice.id });
            activeNotice = null;
            window.setTimeout(showNextNotice, 180);
        })).catch(error => {
            activeNotice = null;
            console.error('Personal alarm delivery failed', error);
        });
        return;
    }

    activeNotice = notice;
    const overlay = node('div', `personal-operation-overlay priority-${notice.priority || 'normal'}`);
    overlay.setAttribute('role', 'dialog');
    overlay.setAttribute('aria-modal', 'true');
    overlay.setAttribute('aria-labelledby', 'personal-operation-title');
    const panel = node('section', 'personal-operation-panel');
    const header = node('header', 'personal-operation-header');
    const identity = node('div', 'personal-operation-identity');
    identity.append(
        node('span', 'personal-operation-eyebrow', priorityLabel(notice.priority)),
        node('h2', '', notice.title || 'Aethel Meldung')
    );
    identity.querySelector('h2').id = 'personal-operation-title';
    const kind = node('span', 'personal-operation-kind', String(notice.kind || 'notice').replaceAll('_', ' '));
    header.append(identity, kind);

    const body = node('div', 'personal-operation-body');
    const copy = node('p', '', formatNoticeBody(notice));
    body.appendChild(copy);

    const meta = node('div', 'personal-operation-meta');
    meta.append(
        node('span', '', `QUELLE · ${notice.source || 'AETHEL'}`),
        node('span', '', new Date(notice.created_at).toLocaleString())
    );

    const actions = node('footer', 'personal-operation-actions');
    const discuss = node('button', 'gw-tool-btn', 'MIT AETHEL BESPRECHEN');
    discuss.type = 'button';
    discuss.addEventListener('click', async () => {
        await operationAction('acknowledge', { id: notice.id });
        document.getElementById('nav-btn-chat')?.click();
        const input = document.getElementById('chat-input');
        if (input) {
            input.value = `[PERSONAL OPERATIONS · ${notice.kind}]\n${notice.title}\n${formatNoticeBody(notice)}\n\nBesprich das mit mir und leite sinnvolle nächste Schritte ab.`;
            input.focus();
        }
        closeOverlay(overlay);
    });
    if (notice.kind === 'task_update') {
        const runCenter = node('button', 'gw-tool-btn', 'RUN CENTER');
        runCenter.type = 'button';
        runCenter.addEventListener('click', () => document.getElementById('nav-btn-tasks')?.click());
        actions.appendChild(runCenter);
    }
    if (notice.kind === 'mail_deadline' && notice.metadata?.folder && notice.metadata?.uid) {
        const openMail = node('button', 'gw-tool-btn', 'E-MAIL ÖFFNEN');
        openMail.type = 'button';
        openMail.addEventListener('click', async () => {
            const workspace = await import('./mail_workspace.js');
            await workspace.openMailMessage(notice.metadata.folder, Number(notice.metadata.uid));
            closeOverlay(overlay);
        });
        actions.appendChild(openMail);
    }
    const read = node('button', 'gw-tool-btn', 'VORLESEN');
    read.type = 'button';
    read.addEventListener('click', async () => {
        const voice = await import('./voice.js');
        await voice.speak(`${notice.title}. ${formatNoticeBody(notice)}`);
    });
    const snooze = node('button', 'gw-tool-btn', '15 MIN SCHLUMMERN');
    snooze.type = 'button';
    snooze.addEventListener('click', async () => {
        await operationAction('snooze', { id: notice.id, minutes: 15 });
        delivered.delete(notice.id);
        closeOverlay(overlay);
    });
    const acknowledge = node('button', 'gw-tool-btn personal-operation-primary', 'ERLEDIGT');
    acknowledge.type = 'button';
    acknowledge.addEventListener('click', async () => {
        await operationAction('acknowledge', { id: notice.id });
        closeOverlay(overlay);
    });
    actions.append(read, snooze, discuss, acknowledge);
    panel.append(header, body, meta, actions);
    overlay.appendChild(panel);
    document.body.appendChild(overlay);
    if (notice.speak) read.click();
}

function showNextNotice() {
    if (activeNotice || pendingQueue.length === 0) return;
    const next = pendingQueue.shift();
    openNotice(next);
}

async function pollOperations() {
    try {
        const payload = await api.getPersonalOperations();
        const items = Array.isArray(payload?.items) ? payload.items : [];
        const fresh = items.filter(item => item?.id && !delivered.has(item.id));
        for (const item of fresh) {
            delivered.add(item.id);
            pendingQueue.push(item);
        }
        pendingQueue.sort((left, right) => {
            const score = { critical: 3, high: 2, normal: 1 };
            return (score[right.priority] || 0) - (score[left.priority] || 0);
        });
        showNextNotice();
    } catch (error) {
        console.warn('Personal Operations monitor unavailable', error);
    }
}

export function startPersonalOperationsMonitor() {
    if (monitorStarted) return;
    monitorStarted = true;
    pollOperations();
    window.setInterval(pollOperations, 4000);
}
