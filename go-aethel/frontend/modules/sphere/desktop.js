// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/desktop.js
// Purpose: Virtual Desktop Environments, Desktop Switcher, Ambient Lighting, and Fullscreen

import { state } from '../state.js';
import { contextBus } from './context_bus.js';
import { getAppDefinition } from './app_registry.js';

export const DESKTOPS = [
    { id: 'PERSONAL', label: 'PERSONAL', icon: '👤', desc: 'Dashboard, Mails, Wetter & Termine' },
    { id: 'WORK', label: 'WORK', icon: '💼', desc: 'Writer, Pläne, Aufgaben & Vault' },
    { id: 'RESEARCH', label: 'RESEARCH', icon: '🔬', desc: 'Browser, Quellen & Research Desk' },
    { id: 'TRAVEL', label: 'TRAVEL', icon: '🗺', desc: 'Trip Planner & Global Watch Bridge' },
    { id: 'PROJECT', label: 'PROJECT', icon: '⚙', desc: 'Agent Runs, Terminal & Live Flow' },
    { id: 'INCIDENT', label: 'INCIDENT', icon: '🚨', desc: 'Sicherheit, Risiken & Audit' }
];

let activeVirtualDesktop = 'PERSONAL';

/**
 * Setup virtual desktop topbar switcher and ambient lighting
 */
export function setupDesktopEnvironment() {
    const topbar = document.querySelector('.sphere-topbar');
    if (!topbar || topbar.dataset.vgtDesktopReady === 'true') return;
    topbar.dataset.vgtDesktopReady = 'true';

    // Insert Virtual Desktop Switcher in the center of the topbar
    const switcher = document.createElement('nav');
    switcher.className = 'sphere-virtual-desktop-switcher font-mono';
    switcher.setAttribute('aria-label', 'Virtuelle Desktops');
    switcher.style.cssText = 'display:flex; gap:4px; background:rgba(0,0,0,0.4); padding:3px 6px; border-radius:6px; border:1px solid rgba(0,240,255,0.15);';

    DESKTOPS.forEach((d, idx) => {
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.className = `sphere-vdesktop-btn ${d.id === activeVirtualDesktop ? 'active' : ''}`;
        btn.dataset.desktopId = d.id;
        btn.title = `${d.label} (Ctrl+Alt+${idx + 1}) - ${d.desc}`;
        btn.style.cssText = 'background:none; border:none; color:var(--vgt-text-dim); font-size:10px; padding:3px 8px; border-radius:4px; cursor:pointer; font-weight:bold; transition:all 0.2s; display:flex; align-items:center; gap:4px;';
        
        const icon = document.createElement('span');
        icon.textContent = d.icon;
        const text = document.createElement('span');
        text.textContent = d.label;

        btn.append(icon, text);

        btn.addEventListener('click', () => {
            switchVirtualDesktop(d.id);
        });

        switcher.appendChild(btn);
    });

    // Replace or insert into topbar
    const brand = topbar.querySelector('.sphere-brand');
    if (brand) {
        brand.after(switcher);
    } else {
        topbar.prepend(switcher);
    }

    // Voice Listen / Mute Toggle Button in Topbar
    const actions = topbar.querySelector('.sphere-topbar-actions');
    if (actions && !document.getElementById('sphere-btn-voice-toggle')) {
        const btnVoiceToggle = document.createElement('button');
        btnVoiceToggle.id = 'sphere-btn-voice-toggle';
        btnVoiceToggle.type = 'button';
        btnVoiceToggle.className = 'sphere-shell-button font-mono';
        btnVoiceToggle.title = 'Aethel Sprachdienst aktivieren / stummschalten';
        btnVoiceToggle.style.cssText = 'font-size:10px; display:flex; align-items:center; gap:4px; cursor:pointer;';

        const updateVoiceBtn = () => {
            if (state.isVoiceCallActive) {
                btnVoiceToggle.style.borderColor = 'var(--vgt-green)';
                btnVoiceToggle.style.color = 'var(--vgt-green)';
                btnVoiceToggle.style.background = 'rgba(57,255,20,0.1)';
                btnVoiceToggle.innerHTML = `<span style="color:var(--vgt-green);">●</span> SPRACHE: AKTIV`;
            } else {
                btnVoiceToggle.style.borderColor = 'rgba(255,255,255,0.2)';
                btnVoiceToggle.style.color = 'var(--vgt-text-dim)';
                btnVoiceToggle.style.background = 'none';
                btnVoiceToggle.innerHTML = `<span style="color:var(--vgt-red);">○</span> SPRACHE: STUMM`;
            }
        };
        updateVoiceBtn();

        btnVoiceToggle.addEventListener('click', () => {
            if (window.toggleVoiceListening) {
                window.toggleVoiceListening();
                updateVoiceBtn();
            }
        });

        setInterval(updateVoiceBtn, 1000);
        actions.prepend(btnVoiceToggle);
    }

    // Ambient Lighting
    const view = document.getElementById('view-sphere');
    const ambient = document.getElementById('sphere-ambient-level');
    const ambientOutput = document.getElementById('sphere-ambient-output');
    const savedAmbient = Number(localStorage.getItem('aethel_sphere_ambient'));

    const applyAmbient = value => {
        const level = Math.max(0, Math.min(28, Number(value) || 0));
        view?.style.setProperty('--sphere-ambient-opacity', `${level / 100}`);
        if (ambientOutput) ambientOutput.textContent = `${level}%`;
        if (ambient) ambient.value = String(level);
        localStorage.setItem('aethel_sphere_ambient', String(level));
    };
    applyAmbient(Number.isFinite(savedAmbient) ? savedAmbient : 12);
    ambient?.addEventListener('input', event => applyAmbient(event.currentTarget.value));

    // Fullscreen
    document.getElementById('sphere-btn-fullscreen')?.addEventListener('click', async () => {
        try {
            if (document.fullscreenElement) await document.exitFullscreen();
            else await view?.requestFullscreen();
        } catch (err) {
            console.warn('Sphere fullscreen unavailable', err);
        }
    });

    // Keyboard shortcuts for desktop switching
    document.addEventListener('keydown', (e) => {
        if (e.ctrlKey && e.altKey && e.key >= '1' && e.key <= '6') {
            e.preventDefault();
            const index = parseInt(e.key, 10) - 1;
            if (DESKTOPS[index]) {
                switchVirtualDesktop(DESKTOPS[index].id);
            }
        }
    });

    // Restore saved active desktop
    const savedDesktop = localStorage.getItem('aethel_sphere_active_desktop');
    if (savedDesktop && DESKTOPS.some(d => d.id === savedDesktop)) {
        switchVirtualDesktop(savedDesktop);
    }
}

/**
 * Switch the active virtual desktop
 * @param {string} desktopID
 */
export function switchVirtualDesktop(desktopID) {
    activeVirtualDesktop = desktopID;
    localStorage.setItem('aethel_sphere_active_desktop', desktopID);

    // Update switcher buttons UI
    document.querySelectorAll('.sphere-vdesktop-btn').forEach(btn => {
        const isActive = btn.dataset.desktopId === desktopID;
        btn.classList.toggle('active', isActive);
        if (isActive) {
            btn.style.background = 'rgba(0,240,255,0.18)';
            btn.style.color = '#fff';
            btn.style.boxShadow = '0 0 10px rgba(0,240,255,0.3)';
        } else {
            btn.style.background = 'none';
            btn.style.color = 'var(--vgt-text-dim)';
            btn.style.boxShadow = 'none';
        }
    });

    // Filter open windows based on desktop affinity
    document.querySelectorAll('.sphere-window').forEach(win => {
        const appID = win.dataset.appId || win.id.replace('sphere-window-', '');
        const def = getAppDefinition(appID);
        
        // Show if app belongs to this desktop or was opened manually
        if (def && def.desktop) {
            if (def.desktop === desktopID) {
                win.classList.remove('hidden-vdesktop');
            } else if (win.classList.contains('hidden-vdesktop')) {
                // Keep hidden
            }
        }
    });

    contextBus.updateContext({ activeDesktop: desktopID });

    window.dispatchEvent(new CustomEvent('aethel:toast', {
        detail: { message: `Virtueller Desktop: ${desktopID}`, type: 'info' }
    }));
}
