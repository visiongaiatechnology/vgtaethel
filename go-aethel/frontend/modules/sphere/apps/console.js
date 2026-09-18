// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/apps/console.js
// Purpose: Direct Aethel Conversational Agent Window inside Sphere Desktop

import { sendMessage } from '../../chat.js';

let sphereConsoleObserver = null;

export function renderConsoleApp(body) {
    if (!body) return;
    body.replaceChildren();

    const container = document.createElement('div');
    container.className = 'font-mono';
    container.style.cssText = 'display:flex; flex-direction:column; gap:8px; height:100%; color:#fff;';

    const kicker = document.createElement('div');
    kicker.className = 'sphere-app-kicker';
    kicker.style.cssText = 'font-size:10px; color:var(--vgt-cyan); letter-spacing:0.05em; font-weight:bold;';
    kicker.textContent = 'GEMEINSAMER AGENTENKONTEXT · LIVE';

    const transcript = document.createElement('div');
    transcript.className = 'sphere-console-transcript';
    transcript.style.cssText = 'flex:1; overflow-y:auto; padding:10px; background:rgba(0,0,0,0.4); border:1px solid rgba(255,255,255,0.06); border-radius:6px; font-size:11px; display:flex; flex-direction:column; gap:6px;';

    const form = document.createElement('div');
    form.style.cssText = 'display:flex; gap:6px;';

    const input = document.createElement('textarea');
    input.rows = 2;
    input.placeholder = 'Aethel im gemeinsamen Workspace ansprechen…';
    input.style.cssText = 'flex:1; background:rgba(0,0,0,0.4); border:1px solid rgba(0,240,255,0.2); color:#fff; padding:6px 10px; border-radius:4px; font-family:var(--font-mono); font-size:11px; resize:none; outline:none;';

    const sendBtn = document.createElement('button');
    sendBtn.type = 'button';
    sendBtn.className = 'cyber-button';
    sendBtn.style.cssText = 'font-size:10px; padding:6px 12px; background:rgba(0,240,255,0.2); border:1px solid var(--vgt-cyan); color:var(--vgt-cyan); cursor:pointer; width:auto;';
    sendBtn.textContent = 'SENDEN';

    const sync = () => {
        const chat = document.getElementById('chat-output');
        if (!chat) return;
        const messages = [...chat.querySelectorAll('.message, .system-message')].slice(-8);
        transcript.replaceChildren();
        for (const message of messages) {
            const row = document.createElement('div');
            const isUser = message.classList.contains('user');
            row.style.cssText = `padding:6px 8px; border-radius:4px; ${isUser ? 'background:rgba(0,240,255,0.1); color:#fff;' : 'background:rgba(255,255,255,0.03); color:rgba(255,255,255,0.9);'}`;
            row.textContent = message.textContent.trim();
            transcript.append(row);
        }
        transcript.scrollTop = transcript.scrollHeight;
    };

    const submit = async () => {
        const text = input.value.trim();
        if (!text) return;
        const sharedInput = document.getElementById('user-input');
        if (!sharedInput) return;
        input.disabled = true;
        sendBtn.disabled = true;
        sharedInput.value = text;
        input.value = '';
        sync();
        try {
            await sendMessage();
        } finally {
            input.disabled = false;
            sendBtn.disabled = false;
            sync();
        }
    };

    sendBtn.addEventListener('click', () => { void submit(); });
    input.addEventListener('keydown', event => {
        if (event.key === 'Enter' && !event.shiftKey) {
            event.preventDefault();
            void submit();
        }
    });

    sphereConsoleObserver?.disconnect();
    const chat = document.getElementById('chat-output');
    if (chat) {
        sphereConsoleObserver = new MutationObserver(sync);
        sphereConsoleObserver.observe(chat, { childList: true, subtree: true, characterData: true });
    }

    form.append(input, sendBtn);
    container.append(kicker, transcript, form);
    body.appendChild(container);
    sync();
}
