// STATUS: DIAMANT VGT SUPREME
import { state } from './state.js';
import * as api from './api.js';

let started = false;

function element(tag, className, text) {
    const node = document.createElement(tag);
    if (className) node.className = className;
    if (text) node.textContent = text;
    return node;
}

function buildDialog(displayName, location) {
    const overlay = element('div', 'startup-briefing-overlay');
    overlay.setAttribute('role', 'dialog');
    overlay.setAttribute('aria-modal', 'true');
    overlay.setAttribute('aria-labelledby', 'startup-briefing-title');
    const panel = element('section', 'startup-briefing-panel');
    const eyebrow = element('span', 'startup-briefing-eyebrow', 'AETHEL // PERSONAL INTELLIGENCE');
    const title = element('h2', '', `Willkommen zurück, ${displayName}`);
    title.id = 'startup-briefing-title';
    const context = element('p', 'startup-briefing-context', location ? `Lagefokus: ${location}` : 'Globaler Lagefokus');
    const status = element('div', 'startup-briefing-status', 'Bitte warten — aktuelle Quellen werden korreliert …');
    const report = element('article', 'startup-briefing-report markdown-body');
    report.hidden = true;
    const actions = element('div', 'startup-briefing-actions');
    const speak = element('button', 'gw-tool-btn', 'VORLESEN');
    speak.type = 'button';
    speak.disabled = true;
    const discuss = element('button', 'gw-tool-btn', 'MIT AETHEL BESPRECHEN');
    discuss.type = 'button';
    discuss.disabled = true;
    const close = element('button', 'gw-tool-btn', 'SCHLIESSEN');
    close.type = 'button';
    actions.append(speak, discuss, close);
    panel.append(eyebrow, title, context, status, report, actions);
    overlay.appendChild(panel);
    document.body.appendChild(overlay);
    close.addEventListener('click', () => overlay.remove());
    return { overlay, status, report, speak, discuss };
}

async function readBriefingStream(response, target) {
    if (!response.body) throw new Error('Der Briefing-Stream ist nicht verfügbar.');
    const reader = response.body.getReader();
    const decoder = new TextDecoder('utf-8');
    let buffer = '';
    let report = '';
    const formatFn = (typeof window.formatMarkdown === 'function') ? window.formatMarkdown : (t => t);

    for (;;) {
        const { value, done } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const frames = buffer.split('\n\n');
        buffer = frames.pop() || '';
        for (const frame of frames) {
            for (const line of frame.split('\n')) {
                if (!line.startsWith('data:')) continue;
                const payload = line.slice(5).trimStart();
                if (payload === '[DONE]') continue;
                if (payload.startsWith('[[THINKING]]:') || payload.startsWith('[[TOOL_DELTA]]:') || payload.startsWith('[[USAGE]]:') || payload === '[[TOOL_COMMIT]]') continue;
                if (payload.startsWith('[SYSTEM ERROR]') || payload.includes(' API ERROR ')) throw new Error(payload.slice(0, 240));
                try {
                    const parsed = JSON.parse(payload);
                    report += parsed.content || parsed.choices?.[0]?.delta?.content || '';
                } catch (_) {
                    report += payload.replaceAll('[VGT_NL]', '\n');
                }
            }
        }
        target.innerHTML = formatFn(report);
        target.scrollTop = target.scrollHeight;
    }
    return report.trim();
}

export async function runConfiguredStartupBriefing() {
    if (started) return;
    started = true;
    let status;
    try {
        status = await api.getPersonalStatus();
    } catch (_) {
        return;
    }
    const config = status?.config || {};
    const profile = status?.profile || {};
    if (!config.enabled || !config.startup_briefing || !profile.display_name) return;

    const location = [profile.location_city, profile.location_country].filter(Boolean).join(', ');
    const dialog = buildDialog(profile.display_name, location);
    const controller = new AbortController();
    const deadline = window.setTimeout(() => controller.abort(), 120000);
    try {
        const language = localStorage.getItem('aethel_ui_language') || 'de';
        const response = await fetch(`${state.API_BASE}/v1/osint/briefing`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ model_id: state.currentModel, language }),
            signal: controller.signal
        });
        if (!response.ok) throw new Error((await response.text()).slice(0, 240) || `HTTP ${response.status}`);
        dialog.report.hidden = false;
        const report = await readBriefingStream(response, dialog.report);
        if (!report) throw new Error('Das Modell hat keinen Lagebericht geliefert.');
        dialog.status.textContent = 'Lagebericht abgeschlossen';
        dialog.speak.disabled = false;
        dialog.discuss.disabled = false;
        dialog.speak.addEventListener('click', async () => {
            const { speak } = await import('./voice.js');
            await speak(report);
        });
        dialog.discuss.addEventListener('click', () => {
            document.getElementById('nav-btn-chat')?.click();
            const input = document.getElementById('chat-input');
            if (input) {
                input.value = `[PERSÖNLICHES START-LAGEBRIEFING]\nBesprich diesen Bericht mit mir. Trenne Fakten, Bewertung und Unsicherheit.\n\n${report.slice(0, 14000)}`;
                input.focus();
            }
            dialog.overlay.remove();
        });
        if (config.startup_read_aloud) dialog.speak.click();
    } catch (error) {
        const message = error?.name === 'AbortError' ? 'Zeitlimit nach 120 Sekunden erreicht.' : (error instanceof Error ? error.message : 'Unbekannter Fehler');
        dialog.status.textContent = `Lagebericht fehlgeschlagen: ${message}`;
        dialog.status.classList.add('is-error');
    } finally {
        window.clearTimeout(deadline);
    }
}
