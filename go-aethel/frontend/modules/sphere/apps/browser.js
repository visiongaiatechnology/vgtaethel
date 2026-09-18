// STATUS: DIAMANT VGT SUPREME
// Module: frontend/modules/sphere/apps/browser.js
// Purpose: Sphere Browser 2.0 with Multi-Tab Management, Live Viewport, AI Control HUD & Send-To

import { state } from '../../state.js';
import { contextBus } from '../context_bus.js';

let activeTabs = [
    { id: 'tab-1', title: 'Google Suche', url: 'https://www.google.de', active: true }
];
let currentTabID = 'tab-1';

/**
 * Render Sphere Browser 2.0 inside a window body
 * @param {HTMLElement} body
 */
export async function renderBrowserApp(body) {
    if (!body) return;
    body.replaceChildren();

    const container = document.createElement('div');
    container.className = 'sphere-browser-container font-mono';
    container.style.cssText = 'display:flex; flex-direction:column; height:100%; gap:8px; color:#fff;';

    // 1. Tab Bar
    const tabbar = document.createElement('div');
    tabbar.className = 'sphere-browser-tabbar';
    tabbar.style.cssText = 'display:flex; align-items:center; gap:4px; background:rgba(0,0,0,0.4); padding:4px 6px; border-radius:6px 6px 0 0; overflow-x:auto;';

    const tabsContainer = document.createElement('div');
    tabsContainer.style.cssText = 'display:flex; gap:4px; flex:1; overflow-x:auto;';

    const newTabBtn = document.createElement('button');
    newTabBtn.type = 'button';
    newTabBtn.className = 'toolbar-btn';
    newTabBtn.style.cssText = 'background:none; border:1px solid rgba(255,255,255,0.15); color:var(--vgt-cyan); padding:2px 8px; border-radius:4px; cursor:pointer; font-weight:bold; font-size:12px;';
    newTabBtn.textContent = '+';
    newTabBtn.title = 'Neuen Tab öffnen';

    tabbar.append(tabsContainer, newTabBtn);

    // 2. Navigation & Address Bar
    const navBar = document.createElement('div');
    navBar.style.cssText = 'display:flex; align-items:center; gap:6px; background:rgba(0,0,0,0.3); padding:6px 8px; border-radius:6px; border:1px solid rgba(138,43,226,0.25);';

    const btnBack = document.createElement('button');
    btnBack.type = 'button';
    btnBack.className = 'toolbar-btn';
    btnBack.style.cssText = 'background:none; border:none; color:#fff; cursor:pointer; font-size:12px;';
    btnBack.textContent = '◀';
    btnBack.title = 'Zurück';

    const btnForward = document.createElement('button');
    btnForward.type = 'button';
    btnForward.className = 'toolbar-btn';
    btnForward.style.cssText = 'background:none; border:none; color:#fff; cursor:pointer; font-size:12px;';
    btnForward.textContent = '▶';
    btnForward.title = 'Vorwärts';

    const btnReload = document.createElement('button');
    btnReload.type = 'button';
    btnReload.className = 'toolbar-btn';
    btnReload.style.cssText = 'background:none; border:none; color:#fff; cursor:pointer; font-size:12px;';
    btnReload.textContent = '⟳';
    btnReload.title = 'Neu laden';

    const lockIcon = document.createElement('span');
    lockIcon.style.cssText = 'color:var(--vgt-green); font-size:11px;';
    lockIcon.textContent = '🔒';
    lockIcon.title = 'Sichere HTTPS-Verbindung';

    const urlInput = document.createElement('input');
    urlInput.id = 'sphere-browser-url-input';
    urlInput.type = 'text';
    urlInput.value = 'https://www.google.de';
    urlInput.style.cssText = 'flex:1; background:rgba(0,0,0,0.4); border:1px solid rgba(255,255,255,0.1); border-radius:4px; color:#fff; padding:4px 8px; font-size:11px; font-family:var(--font-mono); outline:none;';

    const btnSendTo = document.createElement('button');
    btnSendTo.type = 'button';
    btnSendTo.className = 'cyber-button';
    btnSendTo.style.cssText = 'font-size:9px; padding:3px 8px; background:rgba(138,43,226,0.15); border:1px solid var(--vgt-purple); color:var(--vgt-purple); cursor:pointer; width:auto;';
    btnSendTo.textContent = '➔ SEND TO...';
    btnSendTo.addEventListener('click', () => {
        contextBus.openSendToDialog({
            sourceApp: 'browser',
            title: urlInput.value,
            content: `Webseite: ${urlInput.value}`,
            metadata: { url: urlInput.value }
        });
    });

    navBar.append(btnBack, btnForward, btnReload, lockIcon, urlInput, btnSendTo);

    // 3. AI Control HUD Banner
    const aiHud = document.createElement('div');
    aiHud.id = 'sphere-browser-ai-hud';
    aiHud.style.cssText = 'display:none; align-items:center; justify-content:space-between; background:rgba(138,43,226,0.15); border:1px solid var(--vgt-purple); border-radius:6px; padding:6px 10px; font-size:10px;';
    aiHud.innerHTML = `<span style="color:var(--vgt-purple); display:flex; align-items:center; gap:6px;"><span class="sphere-pulse-dot" style="width:6px; height:6px; border-radius:50%; background:var(--vgt-purple); display:inline-block;"></span> <strong>AI CONTROL ACTIVE</strong> · Aethel interagiert mit der Webseite</span><button type="button" id="sphere-browser-pause-ai" class="toolbar-btn" style="background:none; border:1px solid var(--vgt-purple); color:#fff; font-size:9px; padding:2px 6px; border-radius:4px; cursor:pointer;">PAUSIEREN</button>`;

    // 4. Viewport Body
    const viewportArea = document.createElement('div');
    viewportArea.id = 'sphere-browser-viewport-area';
    viewportArea.style.cssText = 'flex:1; overflow:hidden; position:relative; background:rgba(0,0,0,0.5); border:1px solid rgba(255,255,255,0.08); border-radius:6px; display:flex; justify-content:center; align-items:center;';

    const placeholder = document.createElement('div');
    placeholder.id = 'sphere-browser-placeholder-view';
    placeholder.style.cssText = 'text-align:center; color:var(--vgt-text-dim); font-size:11px; line-height:1.5; padding:20px;';
    placeholder.innerHTML = `
        <p style="color:#fff; font-weight:bold; font-size:12px; margin-bottom:6px;">AETHEL SECURE BROWSER ENGINE</p>
        <span>Gib eine HTTPS-URL ein oder weise Aethel im Chat an:</span><br>
        <code style="color:var(--vgt-cyan); display:inline-block; margin-top:8px;">"Recherchiere die besten Hotels in Tokio auf Booking.com"</code>
    `;

    const screenshotImg = document.createElement('img');
    screenshotImg.id = 'sphere-browser-live-img';
    screenshotImg.className = 'hidden';
    screenshotImg.style.cssText = 'width:100%; height:100%; object-fit:contain; border-radius:4px;';

    viewportArea.append(placeholder, screenshotImg);

    container.append(tabbar, navBar, aiHud, viewportArea);
    body.appendChild(container);

    // Tab Management functions
    function renderTabs() {
        tabsContainer.replaceChildren();
        activeTabs.forEach(t => {
            const tabBtn = document.createElement('div');
            const isActive = t.id === currentTabID;
            tabBtn.className = `sphere-browser-tab ${isActive ? 'active' : ''}`;
            tabBtn.style.cssText = `display:flex; align-items:center; gap:6px; padding:4px 10px; border-radius:4px 4px 0 0; cursor:pointer; font-size:10px; max-width:180px; ${isActive ? 'background:rgba(138,43,226,0.25); border-top:2px solid var(--vgt-purple); color:#fff;' : 'background:rgba(255,255,255,0.03); color:var(--vgt-text-dim);'}`;

            const titleSpan = document.createElement('span');
            titleSpan.style.cssText = 'white-space:nowrap; overflow:hidden; text-overflow:ellipsis;';
            titleSpan.textContent = t.title || 'Tab';

            const closeX = document.createElement('button');
            closeX.type = 'button';
            closeX.style.cssText = 'background:none; border:none; color:inherit; font-size:10px; cursor:pointer; padding:0 2px;';
            closeX.textContent = '✕';
            closeX.addEventListener('click', (e) => {
                e.stopPropagation();
                if (activeTabs.length > 1) {
                    activeTabs = activeTabs.filter(item => item.id !== t.id);
                    if (currentTabID === t.id) currentTabID = activeTabs[0].id;
                    renderTabs();
                }
            });

            tabBtn.append(titleSpan, closeX);

            tabBtn.addEventListener('click', () => {
                currentTabID = t.id;
                urlInput.value = t.url;
                renderTabs();
            });

            tabsContainer.appendChild(tabBtn);
        });
    }

    newTabBtn.addEventListener('click', () => {
        const newID = `tab-${Date.now()}`;
        activeTabs.push({ id: newID, title: 'Neuer Tab', url: 'https://www.google.de', active: true });
        currentTabID = newID;
        urlInput.value = 'https://www.google.de';
        renderTabs();
    });

    const navigate = async () => {
        let url = urlInput.value.trim();
        if (!url) return;
        if (!url.startsWith('http://') && !url.startsWith('https://')) {
            url = 'https://' + url;
        }
        urlInput.value = url;

        // Update active tab title and URL
        const tab = activeTabs.find(t => t.id === currentTabID);
        if (tab) {
            tab.url = url;
            tab.title = url.replace(/^https?:\/\//, '').slice(0, 20);
            renderTabs();
        }

        try {
            const res = await fetch(`${state.API_BASE}/v1/tools/execute`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    name: 'web_browser',
                    args: { action: 'navigate', url: url }
                })
            });
            if (res.ok) {
                screenshotImg.src = `${state.API_BASE}/browser/screenshot.png?t=${Date.now()}`;
                screenshotImg.classList.remove('hidden');
                placeholder.classList.add('hidden');
            }
        } catch (err) {
            console.warn('Navigation request error', err);
        }
    };

    urlInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            void navigate();
        }
    });

    btnReload.addEventListener('click', () => void navigate());

    renderTabs();
}
