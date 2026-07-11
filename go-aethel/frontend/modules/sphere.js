import { state } from './state.js';
import { sendMessage } from './chat.js';

// Strict HTML Sanitizer to prevent XSS (Cross-Site Scripting)
export function sanitizeRichText(html) {
    if (!html) return "";
    
    // Parse the HTML string into a DOM structure in memory (isolated from live DOM)
    const parser = new DOMParser();
    const doc = parser.parseFromString(html, "text/html");
    
    const allowedTags = new Set([
        "h1", "h2", "h3", "p", "b", "i", "u", "ul", "ol", "li", "br", "div", "span", "strong", "em"
    ]);
    
    const cleanNode = (node) => {
        // Text node - completely safe
        if (node.nodeType === Node.TEXT_NODE) {
            return document.createTextNode(node.textContent);
        }
        
        // Non-element node - skip/discard
        if (node.nodeType !== Node.ELEMENT_NODE) {
            return null;
        }
        
        const tagName = node.tagName.toLowerCase();
        
        // Tag is not in whitelist - convert to simple text representation (escaped text content)
        if (!allowedTags.has(tagName)) {
            return document.createTextNode(node.textContent);
        }
        
        // Recreate the element to strip all standard and malicious attributes
        const newEl = document.createElement(tagName);
        
        // Clean attributes (only allow safe styles for text alignment)
        if (node.hasAttribute("style")) {
            const styleValue = node.getAttribute("style").toLowerCase().trim();
            // Match safe text alignments only
            const allowedStyles = [];
            if (styleValue.includes("text-align: center") || styleValue.includes("text-align:center")) {
                allowedStyles.push("text-align: center;");
            } else if (styleValue.includes("text-align: right") || styleValue.includes("text-align:right")) {
                allowedStyles.push("text-align: right;");
            } else if (styleValue.includes("text-align: left") || styleValue.includes("text-align:left")) {
                allowedStyles.push("text-align: left;");
            } else if (styleValue.includes("text-align: justify") || styleValue.includes("text-align:justify")) {
                allowedStyles.push("text-align: justify;");
            }
            if (allowedStyles.length > 0) {
                newEl.setAttribute("style", allowedStyles.join(" "));
            }
        }
        
        // Recursively clean and append children
        node.childNodes.forEach(child => {
            const cleanChild = cleanNode(child);
            if (cleanChild) {
                newEl.appendChild(cleanChild);
            }
        });
        
        return newEl;
    };
    
    // Process body nodes
    const fragment = document.createDocumentFragment();
    doc.body.childNodes.forEach(child => {
        const cleanChild = cleanNode(child);
        if (cleanChild) {
            fragment.appendChild(cleanChild);
        }
    });
    
    // Return sanitized HTML string
    const container = document.createElement("div");
    container.appendChild(fragment);
    return container.innerHTML;
}

// Draggable window implementation
export function makeDraggable(windowEl, handleEl) {
    let pos1 = 0, pos2 = 0, pos3 = 0, pos4 = 0;
    
    handleEl.addEventListener("mousedown", dragMouseDown);
    handleEl.addEventListener("touchstart", dragTouchStart, { passive: true });
    
    function dragMouseDown(e) {
        if (e.target.closest('button, input, textarea, select')) return;
        e.preventDefault();
        focusWindow(windowEl);
        
        // Get mouse cursor position at startup
        pos3 = e.clientX;
        pos4 = e.clientY;
        
        document.addEventListener("mouseup", closeDragElement);
        document.addEventListener("mousemove", elementDrag);
    }
    
    function dragTouchStart(e) {
        if (e.target.closest('button, input, textarea, select')) return;
        focusWindow(windowEl);
        if (e.touches.length === 1) {
            pos3 = e.touches[0].clientX;
            pos4 = e.touches[0].clientY;
            
            document.addEventListener("touchend", closeDragElement);
            document.addEventListener("touchmove", elementTouchDrag, { passive: false });
        }
    }
    
    function elementDrag(e) {
        e.preventDefault();
        
        // Calculate new cursor position
        pos1 = pos3 - e.clientX;
        pos2 = pos4 - e.clientY;
        pos3 = e.clientX;
        pos4 = e.clientY;
        
        // Set element's new position
        applyNewPosition(windowEl.offsetTop - pos2, windowEl.offsetLeft - pos1);
    }
    
    function elementTouchDrag(e) {
        if (e.touches.length === 1) {
            // Touch move may trigger scroll, prevent it
            e.preventDefault();
            
            pos1 = pos3 - e.touches[0].clientX;
            pos2 = pos4 - e.touches[0].clientY;
            pos3 = e.touches[0].clientX;
            pos4 = e.touches[0].clientY;
            
            applyNewPosition(windowEl.offsetTop - pos2, windowEl.offsetLeft - pos1);
        }
    }
    
    function applyNewPosition(top, left) {
        const desktop = document.querySelector(".sphere-desktop");
        if (!desktop) return;
        
        const rect = desktop.getBoundingClientRect();
        const winRect = windowEl.getBoundingClientRect();
        
        // Bound checks to keep window within workspace view
        const minTop = 0;
        const maxTop = rect.height - 40; // Allow header to remain visible
        const minLeft = -winRect.width + 100;
        const maxLeft = rect.width - 100;
        
        const finalTop = Math.max(minTop, Math.min(top, maxTop));
        const finalLeft = Math.max(minLeft, Math.min(left, maxLeft));
        
        windowEl.style.top = finalTop + "px";
        windowEl.style.left = finalLeft + "px";
    }
    
    function closeDragElement() {
        // Stop moving when mouse/touch is released
        document.removeEventListener("mouseup", closeDragElement);
        document.removeEventListener("mousemove", elementDrag);
        document.removeEventListener("touchend", closeDragElement);
        document.removeEventListener("touchmove", elementTouchDrag);
    }
}

// WebView2 does not reliably expose native CSS resize controls on complex
// flex windows. These explicit edge handles keep resizing deterministic for
// both static and dynamically created Sphere applications.
export function makeResizable(windowEl) {
    if (!windowEl || windowEl.dataset.resizableBound === 'true') return;
    windowEl.dataset.resizableBound = 'true';
    const edges = ['n', 'ne', 'e', 'se', 's', 'sw', 'w', 'nw'];
    for (const edge of edges) {
        const handle = document.createElement('div');
        handle.className = `sphere-resize-handle sphere-resize-${edge}`;
        handle.dataset.edge = edge;
        handle.setAttribute('aria-hidden', 'true');
        handle.addEventListener('mousedown', event => startResize(event, edge));
        windowEl.append(handle);
    }

    function startResize(event, edge) {
        if (windowEl.classList.contains('maximized') || windowEl.classList.contains('minimized')) return;
        event.preventDefault();
        event.stopPropagation();
        focusWindow(windowEl);
        const desktop = document.querySelector('.sphere-desktop');
        if (!desktop) return;
        const startRect = windowEl.getBoundingClientRect();
        const desktopRect = desktop.getBoundingClientRect();
        const startX = event.clientX;
        const startY = event.clientY;
        const startLeft = windowEl.offsetLeft;
        const startTop = windowEl.offsetTop;
        const minWidth = 280;
        const minHeight = 190;
        const onMove = moveEvent => {
            moveEvent.preventDefault();
            const deltaX = moveEvent.clientX - startX;
            const deltaY = moveEvent.clientY - startY;
            let left = startLeft;
            let top = startTop;
            let width = startRect.width;
            let height = startRect.height;
            if (edge.includes('e')) width = startRect.width + deltaX;
            if (edge.includes('s')) height = startRect.height + deltaY;
            if (edge.includes('w')) { width = startRect.width - deltaX; left = startLeft + deltaX; }
            if (edge.includes('n')) { height = startRect.height - deltaY; top = startTop + deltaY; }
            if (width < minWidth && edge.includes('w')) left = startLeft + startRect.width - minWidth;
            if (height < minHeight && edge.includes('n')) top = startTop + startRect.height - minHeight;
            width = Math.max(minWidth, Math.min(width, desktopRect.width - Math.max(0, left) - 12));
            height = Math.max(minHeight, Math.min(height, desktopRect.height - Math.max(0, top) - 12));
            left = Math.max(0, Math.min(left, desktopRect.width - minWidth));
            top = Math.max(0, Math.min(top, desktopRect.height - minHeight));
            windowEl.style.left = `${left}px`;
            windowEl.style.top = `${top}px`;
            windowEl.style.width = `${width}px`;
            windowEl.style.height = `${height}px`;
        };
        const onUp = () => {
            document.removeEventListener('mousemove', onMove);
            document.removeEventListener('mouseup', onUp);
        };
        document.addEventListener('mousemove', onMove);
        document.addEventListener('mouseup', onUp);
    }
}

// Bring active window to front
export function focusWindow(windowEl) {
    const allWindows = document.querySelectorAll(".sphere-window");
    allWindows.forEach(win => {
        win.classList.remove("active-focus");
        win.style.zIndex = 10;
    });
    
    windowEl.classList.add("active-focus");
    windowEl.style.zIndex = 100;
    state.activeSphereWindow = windowEl.id;
}

const sphereAppDefinitions = {
    aethel: { title: 'AETHEL CONSOLE', accent: 'cyan', left: 180, top: 74, width: 520, height: 390 },
    workspace: { title: 'WORKSPACE INDEX', accent: 'cyan', left: 96, top: 112, width: 420, height: 360 },
    runs: { title: 'RUN DESK', accent: 'purple', left: 540, top: 138, width: 410, height: 330 },
    media: { title: 'MEDIA CONSOLE', accent: 'orange', left: 320, top: 190, width: 360, height: 250 },
    weather: { title: 'WEATHER PULSE', accent: 'cyan', left: 650, top: 74, width: 350, height: 310 },
    live: { title: 'AETHEL LIVE FLOW', accent: 'purple', left: 470, top: 160, width: 470, height: 410 }
};
let sphereConsoleObserver = null;
let sphereLivePollTimer = null;

function createSphereWindow(appID) {
    const definition = sphereAppDefinitions[appID];
    const layer = document.getElementById('sphere-app-layer');
    if (!definition || !layer) return null;
    const windowEl = document.createElement('section');
    windowEl.id = `sphere-window-${appID}`;
    windowEl.className = `sphere-window sphere-app-window sphere-accent-${definition.accent}`;
    windowEl.style.left = `${definition.left}px`;
    windowEl.style.top = `${definition.top}px`;
    windowEl.style.width = `${definition.width}px`;
    windowEl.style.height = `${definition.height}px`;

    const header = document.createElement('header');
    header.className = 'sphere-window-header';
    const title = document.createElement('strong');
    title.className = 'sphere-window-title';
    title.textContent = definition.title;
    const controls = document.createElement('div');
    controls.className = 'window-controls';
    const min = document.createElement('button'); min.type = 'button'; min.className = 'win-btn sphere-win-min'; min.title = 'Minimieren';
    const max = document.createElement('button'); max.type = 'button'; max.className = 'win-btn sphere-win-max'; max.title = 'Maximieren';
    const close = document.createElement('button'); close.type = 'button'; close.className = 'win-btn sphere-win-close'; close.title = 'Schließen';
    controls.append(min, max, close); header.append(title, controls);
    const body = document.createElement('div');
    body.className = 'sphere-app-body';
    windowEl.append(header, body); layer.appendChild(windowEl);
    makeDraggable(windowEl, header);
    makeResizable(windowEl);
    windowEl.addEventListener('mousedown', () => focusWindow(windowEl));
    for (const button of [min, max, close]) button.addEventListener('mousedown', event => event.stopPropagation());
    min.addEventListener('click', event => { event.stopPropagation(); windowEl.classList.add('minimized'); });
    max.addEventListener('click', event => { event.stopPropagation(); windowEl.classList.remove('minimized'); windowEl.classList.toggle('maximized'); focusWindow(windowEl); });
    close.addEventListener('click', event => { event.stopPropagation(); windowEl.classList.add('hidden'); });
    return windowEl;
}

function populateAethelApp(body) {
    body.replaceChildren(makeAppHeading('GEMEINSAMER AGENTENKONTEXT · LIVE'));
    const transcript = document.createElement('div'); transcript.className = 'sphere-console-transcript';
    const form = document.createElement('div'); form.className = 'sphere-console-form';
    const input = document.createElement('textarea'); input.rows = 2; input.placeholder = 'Aethel im gemeinsamen Workspace ansprechen…';
    const send = document.createElement('button'); send.type = 'button'; send.className = 'sphere-app-action'; send.textContent = 'SENDEN';
    const sync = () => {
        const chat = document.getElementById('chat-output');
        if (!chat) return;
        const messages = [...chat.querySelectorAll('.message, .system-message')].slice(-8);
        transcript.replaceChildren();
        for (const message of messages) {
            const row = document.createElement('div');
            row.className = message.classList.contains('user') ? 'sphere-console-user' : 'sphere-console-aethel';
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
        input.disabled = true; send.disabled = true;
        sharedInput.value = text;
        input.value = '';
        sync();
        try { await sendMessage(); } finally { input.disabled = false; send.disabled = false; sync(); }
    };
    send.addEventListener('click', () => { void submit(); });
    input.addEventListener('keydown', event => { if (event.key === 'Enter' && !event.shiftKey) { event.preventDefault(); void submit(); } });
    sphereConsoleObserver?.disconnect();
    const chat = document.getElementById('chat-output');
    if (chat) {
        sphereConsoleObserver = new MutationObserver(sync);
        sphereConsoleObserver.observe(chat, { childList: true, subtree: true, characterData: true });
    }
    form.append(input, send); body.append(transcript, form); sync();
}

function makeAppHeading(text) {
    const heading = document.createElement('div');
    heading.className = 'sphere-app-kicker';
    heading.textContent = text;
    return heading;
}

async function populateWorkspaceApp(body) {
    body.replaceChildren(makeAppHeading('VGT_WORKSPACE · METADATEN')); 
    const list = document.createElement('div'); list.className = 'sphere-app-list';
    const refresh = document.createElement('button'); refresh.type = 'button'; refresh.className = 'sphere-app-action'; refresh.textContent = 'INDEX AKTUALISIEREN';
    const render = async () => {
        list.replaceChildren();
        try {
            const response = await fetch(`${state.API_BASE}/v1/sphere/workspace`);
            if (!response.ok) throw new Error(`Workspace ${response.status}`);
            const payload = await response.json();
            const entries = payload.entries || [];
            if (entries.length === 0) {
                const empty = document.createElement('span'); empty.textContent = 'Noch keine sichtbaren Workspace-Artefakte.'; list.append(empty);
                return;
            }
            for (const entry of entries) {
                const row = document.createElement('div'); row.className = 'sphere-workspace-row';
                const path = document.createElement('span'); path.textContent = `${entry.kind === 'folder' ? '▸' : '·'} ${entry.path}`;
                const meta = document.createElement('small'); meta.textContent = entry.kind === 'folder' ? 'ORDNER' : `${Number(entry.size || 0).toLocaleString('de-DE')} B`;
                row.append(path, meta); list.append(row);
            }
        } catch (error) {
            const failure = document.createElement('span'); failure.textContent = `Workspace nicht verfügbar: ${error.message}`; list.append(failure);
        }
    };
    refresh.addEventListener('click', () => { void render(); });
    body.append(refresh, list); await render();
}

async function populateRunsApp(body) {
    body.replaceChildren(makeAppHeading('PERSISTENTE AGENT RUNS'));
    const list = document.createElement('div'); list.className = 'sphere-app-list'; body.append(list);
    try {
        const response = await fetch(`${state.API_BASE}/v1/runs`);
        if (!response.ok) throw new Error(`Run Center ${response.status}`);
        const payload = await response.json();
        const runs = (payload.runs || []).slice(0, 8);
        if (runs.length === 0) {
            const empty = document.createElement('span'); empty.textContent = 'Keine aktiven oder gespeicherten Runs.'; list.append(empty); return;
        }
        for (const run of runs) {
            const row = document.createElement('div'); row.className = 'sphere-run-row';
            const objective = document.createElement('span'); objective.textContent = run.objective || 'Ohne Ziel';
            const status = document.createElement('strong'); status.textContent = String(run.status || 'unknown').toUpperCase(); status.className = `sphere-run-${run.status || 'unknown'}`;
            row.append(objective, status); list.append(row);
        }
    } catch (error) {
        const failure = document.createElement('span'); failure.textContent = `Run Desk nicht verfügbar: ${error.message}`; list.append(failure);
    }
}

function populateMediaApp(body) {
    body.replaceChildren(makeAppHeading('SICHTBARE MEDIENSTEUERUNG'));
    const stateLine = document.createElement('div'); stateLine.className = 'sphere-media-status'; stateLine.textContent = 'Bereit für lokale Medienbefehle.';
    const controls = document.createElement('div'); controls.className = 'sphere-media-controls';
    for (const [label, action] of [['PLAY / PAUSE', 'play_pause'], ['LAUTER', 'volume_up'], ['LEISER', 'volume_down'], ['NÄCHSTER TITEL', 'next']]) {
        const button = document.createElement('button'); button.type = 'button'; button.className = 'sphere-app-action'; button.textContent = label;
        button.addEventListener('click', async () => {
            button.disabled = true; stateLine.textContent = `${label} wird an den Core übergeben…`;
            try {
                const response = await fetch(`${state.API_BASE}/v1/tools/execute`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: 'media_control', args: { action } }) });
                const result = await response.json();
                stateLine.textContent = result.status === 'success' ? 'Befehl ausgeführt.' : (result.error || 'Freigabe oder Ausführung ausstehend.');
            } catch (error) { stateLine.textContent = `Media Core nicht erreichbar: ${error.message}`; }
            finally { button.disabled = false; }
        });
        controls.append(button);
    }
    body.append(stateLine, controls);
}

function populateWeatherApp(body) {
    body.replaceChildren(makeAppHeading('LIVE WEATHER // CITY LOOKUP'));
    const form = document.createElement('form'); form.className = 'sphere-widget-form';
    const city = document.createElement('input'); city.type = 'text'; city.maxLength = 80; city.value = localStorage.getItem('aethel_sphere_weather_city') || 'Köln'; city.placeholder = 'Stadt eingeben'; city.setAttribute('aria-label', 'Stadt für Wetterabfrage');
    const refresh = document.createElement('button'); refresh.type = 'submit'; refresh.className = 'sphere-app-action'; refresh.textContent = 'ABRUFEN';
    const display = document.createElement('div'); display.className = 'sphere-weather-display';
    const render = async () => {
        const query = city.value.trim();
        if (!query) return;
        refresh.disabled = true; display.textContent = 'Wetterdaten werden abgerufen...';
        try {
            const response = await fetch(`${state.API_BASE}/v1/weather?city=${encodeURIComponent(query)}`);
            if (!response.ok) throw new Error((await response.text()).slice(0, 160));
            const weather = await response.json();
            localStorage.setItem('aethel_sphere_weather_city', query);
            display.replaceChildren();
            const temp = document.createElement('strong'); temp.textContent = `${Number(weather.temperature_c).toFixed(1)} °C`;
            const summary = document.createElement('span'); summary.textContent = `${weather.summary || 'Unbekannt'} · Wind ${Number(weather.wind_speed_kmh).toFixed(1)} km/h`;
            const place = document.createElement('small'); place.textContent = `${weather.city || query}${weather.country ? `, ${weather.country}` : ''} · ${weather.observed_at || ''}`;
            display.append(temp, summary, place);
        } catch (error) { display.textContent = `Wetter nicht verfügbar: ${error.message}`; }
        finally { refresh.disabled = false; }
    };
    form.addEventListener('submit', event => { event.preventDefault(); void render(); });
    form.append(city, refresh); body.append(form, display); void render();
}

function populateLiveApp(body) {
    body.replaceChildren(makeAppHeading('LIVE RUN // PLAN, TOOLS & EVIDENCE'));
    const status = document.createElement('div'); status.className = 'sphere-live-status';
    const timeline = document.createElement('div'); timeline.className = 'sphere-live-timeline';
    const render = async () => {
        try {
            const response = await fetch(`${state.API_BASE}/v1/runs`);
            if (!response.ok) throw new Error(`Run Center ${response.status}`);
            const payload = await response.json();
            const runs = payload.runs || [];
            const run = runs.find(item => ['running', 'waiting_approval', 'paused'].includes(item.status)) || runs[0];
            timeline.replaceChildren();
            if (!run) { status.textContent = 'Kein aktiver Run. Starte eine Coding- oder Agentenaufgabe; der Ablauf erscheint hier live.'; return; }
            status.textContent = `${String(run.status || 'unknown').toUpperCase()} · ${run.objective || 'Ohne Ziel'} · Turn ${run.agent_turn || 0}/${run.max_agent_turns || 0}`;
            const entries = [...(run.trace || []).slice(-10).map(trace => ({ kind: 'trace', label: trace.event, detail: trace.detail, at: trace.timestamp })), ...(run.steps || []).slice(-8).map(step => ({ kind: 'step', label: `${step.kind || 'step'} · ${step.status || 'pending'}`, detail: step.title || step.result || step.error || '', at: step.finished_at || step.started_at }))];
            entries.sort((a, b) => String(a.at || '').localeCompare(String(b.at || '')));
            for (const entry of entries.slice(-14)) {
                const row = document.createElement('article'); row.className = `sphere-live-entry sphere-live-${entry.kind}`;
                const label = document.createElement('strong'); label.textContent = entry.label || 'UPDATE';
                const detail = document.createElement('span'); detail.textContent = entry.detail || 'Status aktualisiert.';
                row.append(label, detail); timeline.append(row);
            }
        } catch (error) { status.textContent = `Live Flow nicht verfügbar: ${error.message}`; }
    };
    body.append(status, timeline); void render();
    if (sphereLivePollTimer) clearInterval(sphereLivePollTimer);
    sphereLivePollTimer = setInterval(() => {
        const windowEl = body.closest('.sphere-window');
        if (!windowEl || windowEl.classList.contains('hidden')) return;
        void render();
    }, 2200);
}

function openSphereApp(appID) {
    if (appID === 'editor' || appID === 'browser') {
        const windowEl = document.getElementById(`sphere-window-${appID}`);
        if (windowEl) { windowEl.classList.remove('hidden', 'minimized'); focusWindow(windowEl); }
        return;
    }
    const windowEl = document.getElementById(`sphere-window-${appID}`) || createSphereWindow(appID);
    if (!windowEl) return;
    const desktop = document.querySelector('.sphere-desktop');
    if (desktop && !windowEl.classList.contains('maximized')) {
        const bounds = desktop.getBoundingClientRect();
        const width = Math.min(windowEl.offsetWidth || 420, Math.max(300, bounds.width - 32));
        const height = Math.min(windowEl.offsetHeight || 320, Math.max(220, bounds.height - 38));
        const left = Math.max(16, Math.min(parseInt(windowEl.style.left, 10) || 16, bounds.width - width - 16));
        const top = Math.max(16, Math.min(parseInt(windowEl.style.top, 10) || 16, bounds.height - height - 16));
        windowEl.style.width = `${width}px`;
        windowEl.style.height = `${height}px`;
        windowEl.style.left = `${left}px`;
        windowEl.style.top = `${top}px`;
    }
    windowEl.classList.remove('hidden', 'minimized'); focusWindow(windowEl);
    const body = windowEl.querySelector('.sphere-app-body');
    if (appID === 'workspace') void populateWorkspaceApp(body);
    if (appID === 'runs') void populateRunsApp(body);
    if (appID === 'media') populateMediaApp(body);
    if (appID === 'aethel') populateAethelApp(body);
    if (appID === 'weather') populateWeatherApp(body);
    if (appID === 'live') populateLiveApp(body);
}

function setupSphereShell() {
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
    document.getElementById('sphere-btn-fullscreen')?.addEventListener('click', async () => {
        try {
            if (document.fullscreenElement) await document.exitFullscreen();
            else await view?.requestFullscreen();
        } catch (error) { console.warn('Sphere fullscreen unavailable', error); }
    });
    document.addEventListener('fullscreenchange', () => {
        const button = document.getElementById('sphere-btn-fullscreen');
        if (button) button.textContent = document.fullscreenElement ? '⛶ VOLLBILD BEENDEN' : '⛶ VOLLBILD';
    });
    document.querySelectorAll('[data-sphere-app]').forEach(button => button.addEventListener('click', () => {
        document.querySelectorAll('[data-sphere-app]').forEach(item => item.classList.remove('active'));
        button.classList.add('active'); openSphereApp(button.dataset.sphereApp);
    }));
    setupSphereCommandBar();
    setupSphereHelp();
}

function setupSphereCommandBar() {
    const form = document.getElementById('sphere-command-form');
    const input = document.getElementById('sphere-command-input');
    const send = document.getElementById('sphere-command-send');
    if (!form || !input || !send || form.dataset.bound === 'true') return;
    form.dataset.bound = 'true';

    const resize = () => {
        input.style.height = 'auto';
        input.style.height = `${Math.min(Math.max(input.scrollHeight, 32), 116)}px`;
    };
    const submit = async () => {
        const text = input.value.trim();
        const sharedInput = document.getElementById('user-input');
        if (!text || !sharedInput) return;
        input.disabled = true;
        send.disabled = true;
        sharedInput.value = text;
        input.value = '';
        resize();
        try {
            await sendMessage();
        } finally {
            input.disabled = false;
            send.disabled = false;
            input.focus();
        }
    };
    input.addEventListener('input', resize);
    input.addEventListener('keydown', event => {
        if (event.key === 'Enter' && !event.shiftKey) {
            event.preventDefault();
            void submit();
        }
    });
    form.addEventListener('submit', event => {
        event.preventDefault();
        void submit();
    });
    resize();
}

function setupSphereHelp() {
    const dialog = document.getElementById('sphere-help-dialog');
    const opener = document.getElementById('sphere-btn-help');
    if (!dialog || !opener || dialog.dataset.bound === 'true') return;
    dialog.dataset.bound = 'true';
    const close = () => {
        dialog.classList.add('hidden');
        opener.focus();
    };
    opener.addEventListener('click', () => {
        dialog.classList.remove('hidden');
        dialog.querySelector('#sphere-help-close')?.focus();
    });
    dialog.querySelectorAll('[data-sphere-help-close], #sphere-help-close').forEach(button => button.addEventListener('click', close));
    document.addEventListener('keydown', event => {
        if (event.key === 'Escape' && !dialog.classList.contains('hidden')) close();
    });
}

// Core WYSIWYG Editor Actions
export function executeEditorAction(command, value = null) {
    const editor = document.getElementById("sphere-editor-area");
    if (!editor) return;
    
    editor.focus();
    
    if (command === "formatBlock" && value) {
        document.execCommand("formatBlock", false, value);
    } else {
        document.execCommand(command, false, value);
    }
    
    // Trigger input event to update state/history
    const event = new Event('input', { bubbles: true });
    editor.dispatchEvent(event);
}

// Setup events inside the Sphere workspace
export function setupSphereWorkspace() {
    setupSphereShell();
    // Make windows draggable
    const editorWin = document.getElementById("sphere-window-editor");
    const editorHeader = document.getElementById("sphere-window-editor-header");
    if (editorWin && editorHeader) {
        makeDraggable(editorWin, editorHeader);
        makeResizable(editorWin);
        editorWin.addEventListener("mousedown", () => focusWindow(editorWin));
        editorWin.addEventListener("touchstart", () => focusWindow(editorWin), { passive: true });
    }
    
    const browserWin = document.getElementById("sphere-window-browser");
    const browserHeader = document.getElementById("sphere-window-browser-header");
    if (browserWin && browserHeader) {
        makeDraggable(browserWin, browserHeader);
        makeResizable(browserWin);
        browserWin.addEventListener("mousedown", () => focusWindow(browserWin));
        browserWin.addEventListener("touchstart", () => focusWindow(browserWin), { passive: true });
    }
    
    // Wire Rich-Text editor toolbar actions
    const toolbarActions = [
        { id: "editor-btn-bold", command: "bold" },
        { id: "editor-btn-italic", command: "italic" },
        { id: "editor-btn-underline", command: "underline" },
        { id: "editor-btn-ul", command: "insertUnorderedList" },
        { id: "editor-btn-ol", command: "insertOrderedList" },
        { id: "editor-btn-left", command: "justifyLeft" },
        { id: "editor-btn-center", command: "justifyCenter" },
        { id: "editor-btn-right", command: "justifyRight" },
        { id: "editor-btn-h1", command: "formatBlock", value: "h1" },
        { id: "editor-btn-h2", command: "formatBlock", value: "h2" },
        { id: "editor-btn-h3", command: "formatBlock", value: "h3" },
        { id: "editor-btn-p", command: "formatBlock", value: "p" }
    ];
    
    toolbarActions.forEach(action => {
        const btn = document.getElementById(action.id);
        if (btn) {
            btn.addEventListener("click", (e) => {
                e.preventDefault();
                e.stopPropagation();
                executeEditorAction(action.command, action.value);
            });
        }
    });
    
    // XSS Sanitizer on paste & input to ensure security
    const editorArea = document.getElementById("sphere-editor-area");
    if (editorArea) {
        let saveTimeout = null;
        
        const saveDocument = () => {
            const html = sanitizeRichText(editorArea.innerHTML);
            fetch(`${state.API_BASE}/v1/sphere/document`, {
                method: "POST",
                headers: { "Content-Type": "text/html" },
                body: html
            }).catch(e => console.warn("Failed to save sphere document", e));
        };

        const triggerDebouncedSave = () => {
            if (saveTimeout) clearTimeout(saveTimeout);
            saveTimeout = setTimeout(saveDocument, 800);
        };

        editorArea.addEventListener("input", triggerDebouncedSave);

        editorArea.addEventListener("paste", (e) => {
            e.preventDefault();
            const text = (e.originalEvent || e).clipboardData.getData('text/plain');
            const cleanText = text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
            document.execCommand("insertHTML", false, cleanText);
            triggerDebouncedSave();
        });
        
        // Listen to changes in case they are set via API (innerHTML or insertHTML)
        editorArea.addEventListener("blur", () => {
            const rawContent = editorArea.innerHTML;
            const cleanContent = sanitizeRichText(rawContent);
            if (rawContent !== cleanContent) {
                // Only replace if content actually changed to avoid cursor jumps
                const selection = window.getSelection();
                const range = selection.rangeCount > 0 ? selection.getRangeAt(0) : null;
                const offset = range ? range.startOffset : 0;
                
                editorArea.innerHTML = cleanContent;
                
                // Try to restore cursor
                if (range && editorArea.firstChild) {
                    try {
                        const newRange = document.createRange();
                        newRange.setStart(editorArea.firstChild, Math.min(offset, editorArea.firstChild.length || 0));
                        newRange.collapse(true);
                        selection.removeAllRanges();
                        selection.addRange(newRange);
                    } catch(err) {
                        console.warn("Failed to restore editor cursor offset", err);
                    }
                }
            }
            saveDocument();
        });

        // Load the initial document once on setup
        fetch(`${state.API_BASE}/v1/sphere/document`)
            .then(r => r.text())
            .then(html => {
                editorArea.innerHTML = sanitizeRichText(html);
            })
            .catch(e => console.warn("Failed to load initial sphere document", e));
    }
    
    // Window control buttons
    const setupWinControls = (winId, btnMinId, btnMaxId, btnCloseId) => {
        const win = document.getElementById(winId);
        const btnMin = document.getElementById(btnMinId);
        const btnMax = document.getElementById(btnMaxId);
        const btnClose = document.getElementById(btnCloseId);
        
        if (btnMin && win) {
            btnMin.addEventListener("click", () => win.classList.add("minimized"));
        }
        if (btnMax && win) {
            btnMax.addEventListener("click", () => {
                win.classList.remove("minimized");
                win.classList.toggle("maximized");
            });
        }
        if (btnClose && win) {
            btnClose.addEventListener("click", () => win.classList.add("hidden"));
        }
    };
    
    setupWinControls("sphere-window-editor", "win-editor-min", "win-editor-max", "win-editor-close");
    setupWinControls("sphere-window-browser", "win-browser-min", "win-browser-max", "win-browser-close");

    // Periodically poll for document updates written by Aethel (only when view-sphere is visible and user is not editing)
    setInterval(() => {
        const sphereView = document.getElementById("view-sphere");
        if (!sphereView || sphereView.classList.contains("hidden")) return;
        
        const editorAreaEl = document.getElementById("sphere-editor-area");
        if (!editorAreaEl || document.activeElement === editorAreaEl) return;
        
        fetch(`${state.API_BASE}/v1/sphere/document`)
            .then(r => r.text())
            .then(html => {
                const cleanHtml = sanitizeRichText(html);
                if (editorAreaEl.innerHTML !== cleanHtml) {
                    editorAreaEl.innerHTML = cleanHtml;
                }
            })
            .catch(e => console.warn("Failed to sync sphere document", e));
    }, 1500);
}
