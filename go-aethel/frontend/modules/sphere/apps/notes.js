// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/apps/notes.js
// Purpose: Local Scratch Notes with Send-To Actions in Sphere Desktop

import { contextBus } from '../context_bus.js';

export function renderNotesApp(body) {
    if (!body) return;
    body.replaceChildren();

    const container = document.createElement('div');
    container.className = 'font-mono';
    container.style.cssText = 'display:flex; flex-direction:column; gap:8px; height:100%; color:#fff;';

    const header = document.createElement('div');
    header.style.cssText = 'display:flex; justify-content:space-between; align-items:center;';

    const kicker = document.createElement('div');
    kicker.style.cssText = 'font-size:10px; color:var(--vgt-orange); letter-spacing:0.05em; font-weight:bold;';
    kicker.textContent = 'SCHNELLNOTIZEN // SCRATCHPAD';

    const btnSendTo = document.createElement('button');
    btnSendTo.type = 'button';
    btnSendTo.className = 'cyber-button';
    btnSendTo.style.cssText = 'font-size:9px; padding:2px 8px; background:rgba(0,240,255,0.1); border:1px solid var(--vgt-cyan); color:var(--vgt-cyan); cursor:pointer; width:auto;';
    btnSendTo.textContent = '➔ SEND TO...';

    header.append(kicker, btnSendTo);

    const textarea = document.createElement('textarea');
    textarea.className = 'sphere-notes-textarea';
    textarea.placeholder = 'Gedanken, Snippets, Notizen…';
    textarea.value = localStorage.getItem('aethel_sphere_notes') || '';
    textarea.style.cssText = 'flex:1; background:rgba(0,0,0,0.4); border:1px solid rgba(255,255,255,0.08); border-radius:6px; padding:12px; color:rgba(255,255,255,0.9); font-size:12px; font-family:var(--font-mono); resize:none; outline:none; line-height:1.5;';

    textarea.addEventListener('input', () => {
        localStorage.setItem('aethel_sphere_notes', textarea.value);
    });

    btnSendTo.addEventListener('click', () => {
        if (textarea.value.trim()) {
            contextBus.openSendToDialog({
                sourceApp: 'notes',
                title: 'Scratch Note',
                content: textarea.value
            });
        }
    });

    container.append(header, textarea);
    body.appendChild(container);
}
