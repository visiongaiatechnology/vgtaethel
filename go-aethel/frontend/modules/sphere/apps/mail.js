// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/apps/mail.js
// Purpose: Sphere Mail Desk, Encrypted Communication & Mail to Task/Trip/Doc Workflows

import { state } from '../../state.js';
import { contextBus } from '../context_bus.js';

/**
 * Render Sphere Mail inside a window body
 * @param {HTMLElement} body
 */
export async function renderMailApp(body) {
    if (!body) return;
    body.replaceChildren();

    const container = document.createElement('div');
    container.className = 'sphere-mail-container font-mono';
    container.style.cssText = 'display:flex; flex-direction:column; height:100%; gap:8px; color:#fff;';

    const header = document.createElement('div');
    header.style.cssText = 'display:flex; justify-content:space-between; align-items:center; border-bottom:1px solid rgba(255,123,0,0.2); padding-bottom:6px;';
    header.innerHTML = `
        <span style="font-size:11px; color:var(--vgt-orange); font-weight:bold;">✉ SPHERE MAIL // ENCRYPTED INBOX</span>
        <button id="sphere-mail-sync" class="cyber-button" style="font-size:9px; padding:3px 8px; background:rgba(255,123,0,0.15); border:1px solid var(--vgt-orange); color:var(--vgt-orange); cursor:pointer; width:auto;">SYNCHRONISIEREN</button>
    `;

    const list = document.createElement('div');
    list.className = 'sphere-mail-list';
    list.style.cssText = 'flex:1; overflow-y:auto; display:flex; flex-direction:column; gap:8px; padding-right:4px;';

    container.append(header, list);
    body.appendChild(container);

    const loadMessages = async () => {
        list.replaceChildren();
        try {
            const res = await fetch(`${state.API_BASE}/v1/mail/messages?folder=INBOX`);
            if (!res.ok) throw new Error(`HTTP ${res.status}`);
            const data = await res.json();
            const messages = data.messages || [];

            if (messages.length === 0) {
                const empty = document.createElement('div');
                empty.style.cssText = 'text-align:center; padding:30px 10px; color:var(--vgt-text-dim); font-size:11px;';
                empty.textContent = 'Keine neuen verschlüsselten E-Mails im Postfach.';
                list.appendChild(empty);
                return;
            }

            messages.forEach(m => {
                const card = document.createElement('article');
                card.className = 'glass-card';
                card.style.cssText = 'background:rgba(0,0,0,0.35); border:1px solid rgba(255,255,255,0.06); border-radius:6px; padding:10px; display:flex; flex-direction:column; gap:4px;';

                const top = document.createElement('div');
                top.style.cssText = 'display:flex; justify-content:space-between; font-size:10px; color:var(--vgt-text-dim);';
                top.innerHTML = `<span>Von: <b style="color:#fff;">${m.from || 'Unbekannt'}</b></span><span>${m.date || 'Heute'}</span>`;

                const subject = document.createElement('strong');
                subject.style.cssText = 'font-size:11px; color:var(--vgt-cyan);';
                subject.textContent = m.subject || 'Ohne Betreff';

                const snippet = document.createElement('p');
                snippet.style.cssText = 'margin:0; font-size:10px; color:rgba(255,255,255,0.7); line-height:1.4;';
                snippet.textContent = (m.body_preview || m.snippet || '').slice(0, 140) + '...';

                const actions = document.createElement('div');
                actions.style.cssText = 'display:flex; gap:6px; justify-content:flex-end; margin-top:4px;';

                const btnSendToTask = document.createElement('button');
                btnSendToTask.type = 'button';
                btnSendToTask.className = 'cyber-button';
                btnSendToTask.style.cssText = 'font-size:8px; padding:2px 6px; background:rgba(0,240,255,0.1); border:1px solid var(--vgt-cyan); color:var(--vgt-cyan); cursor:pointer; width:auto;';
                btnSendToTask.textContent = '➔ IN TASK UMWANDELN';
                btnSendToTask.addEventListener('click', () => {
                    contextBus.openSendToDialog({
                        sourceApp: 'mail',
                        targetApp: 'task',
                        title: `Aufgabe aus Mail: ${m.subject}`,
                        content: m.body_preview || m.snippet || ''
                    });
                });

                actions.appendChild(btnSendToTask);
                card.append(top, subject, snippet, actions);
                list.appendChild(card);
            });
        } catch (err) {
            const errEl = document.createElement('div');
            errEl.style.cssText = 'color:var(--vgt-text-dim); font-size:11px; padding:10px;';
            errEl.textContent = `E-Mail Postfach nicht konfiguriert oder synchronisiert.`;
            list.appendChild(errEl);
        }
    };

    header.querySelector('#sphere-mail-sync').addEventListener('click', () => void loadMessages());
    void loadMessages();
}
