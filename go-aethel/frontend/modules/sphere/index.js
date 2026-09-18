// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/index.js
// Purpose: Master Sphere 2.0 Orchestrator (Desktop, Window Manager, Apps Dispatcher, Command Palette & Sidecar)

import { setupDesktopEnvironment } from './desktop.js';
import { setupCommandPalette, openCommandPalette } from './command_palette.js';
import { setupAethelSidecar } from './sidecar.js';
import { createSphereWindow, focusWindow } from './window_manager.js';
import { getAppDefinition } from './app_registry.js';
import { contextBus } from './context_bus.js';

// Modular Apps
import { renderWriterApp } from './apps/writer.js';
import { renderBrowserApp } from './apps/browser.js';
import { renderTravelApp } from './apps/travel.js';
import { renderPlannerApp } from './apps/planner.js';
import { renderResearchApp } from './apps/research.js';
import { renderFilesApp } from './apps/files.js';
import { renderDashboardApp } from './apps/dashboard.js';
import { renderMailApp } from './apps/mail.js';
import { renderWeatherApp } from './apps/weather.js';
import { renderMarketApp } from './apps/markets.js';
import { renderRunsApp } from './apps/runs.js';
import { renderLiveApp } from './apps/live.js';
import { renderTerminalApp } from './apps/terminal.js';
import { renderConsoleApp } from './apps/console.js';
import { renderNotesApp } from './apps/notes.js';

const appRenderers = {
    writer: renderWriterApp,
    browser: renderBrowserApp,
    travel: renderTravelApp,
    planner: renderPlannerApp,
    research: renderResearchApp,
    files: renderFilesApp,
    dashboard: renderDashboardApp,
    mail: renderMailApp,
    weather: renderWeatherApp,
    market: renderMarketApp,
    runs: renderRunsApp,
    live: renderLiveApp,
    terminal: renderTerminalApp,
    aethel: renderConsoleApp,
    notes: renderNotesApp
};

/**
 * Open or restore a Sphere application window
 * @param {string} appID
 */
export function openSphereApp(appID) {
    let win = document.getElementById(`sphere-window-${appID}`);
    if (!win) {
        win = createSphereWindow(appID);
    }
    if (!win) return;

    win.classList.remove('hidden', 'minimized');
    focusWindow(win);

    const body = win.querySelector('.sphere-app-body');
    const renderer = appRenderers[appID];
    if (renderer && body && !win.dataset.rendered) {
        win.dataset.rendered = 'true';
        void renderer(body);
    }

    contextBus.updateContext({ focusedApp: appID, focusedWindowId: win.id });
}

/**
 * Master Setup of Sphere 2.0 Workspace
 */
export function setupSphereWorkspace() {
    setupDesktopEnvironment();
    setupCommandPalette();
    setupAethelSidecar();

    // Wire Dock / Taskbar launchers
    document.querySelectorAll('.sphere-dock-item').forEach(btn => {
        const appID = btn.dataset.app;
        if (appID && btn.dataset.vgtWired !== 'true') {
            btn.dataset.vgtWired = 'true';
            btn.addEventListener('click', () => openSphereApp(appID));
        }
    });

    // Wire Command Bar
    const cmdBar = document.getElementById('sphere-command-form');
    if (cmdBar && cmdBar.dataset.vgtWired !== 'true') {
        cmdBar.dataset.vgtWired = 'true';
        cmdBar.addEventListener('submit', (e) => {
            e.preventDefault();
            const input = document.getElementById('sphere-command-input');
            if (input && input.value.trim()) {
                const q = input.value.trim();
                input.value = '';
                // Trigger command palette or direct execution
                openCommandPalette();
            }
        });
    }

    // Default open apps if fresh
    const hasOpened = localStorage.getItem('aethel_sphere_initialized');
    if (!hasOpened) {
        localStorage.setItem('aethel_sphere_initialized', 'true');
        openSphereApp('dashboard');
        openSphereApp('writer');
    }
}

// Global hook
window.openSphereApp = openSphereApp;
window.setupSphereWorkspace = setupSphereWorkspace;
