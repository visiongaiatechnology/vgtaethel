// STATUS: DIAMANT VGT SUPREME
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
    market: { title: 'MARKET PULSE', accent: 'orange', left: 580, top: 80, width: 580, height: 450 },
    tasks: { title: 'AUTOMATIONEN & TASKS', accent: 'cyan', left: 380, top: 80, width: 620, height: 480 },
    live: { title: 'AETHEL LIVE FLOW', accent: 'purple', left: 470, top: 160, width: 470, height: 410 },
    notes: { title: 'SCRATCH NOTES', accent: 'orange', left: 220, top: 100, width: 380, height: 320 },
    editor: { title: 'AETHEL WRITER', accent: 'cyan', left: 48, top: 36, width: 520, height: 400 },
    browser: { title: 'AETHEL BROWSER', accent: 'purple', left: 420, top: 56, width: 480, height: 360 },
    terminal: { title: 'TERMINAL', accent: 'orange', left: 300, top: 120, width: 520, height: 350 },
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

function populateMarketApp(body) {
    body.replaceChildren(makeAppHeading('LIVE MARKET ENGINE // MULTI-ASSET FEED'));
    
    const toolbar = document.createElement('div');
    toolbar.className = 'sphere-market-toolbar';
    toolbar.style.display = 'flex';
    toolbar.style.flexDirection = 'column';
    toolbar.style.gap = '8px';
    toolbar.style.marginBottom = '12px';

    const topRow = document.createElement('div');
    topRow.style.display = 'flex';
    topRow.style.justifyContent = 'space-between';
    topRow.style.alignItems = 'center';
    topRow.style.gap = '8px';

    const tabsContainer = document.createElement('div');
    tabsContainer.className = 'sphere-market-tabs';
    tabsContainer.style.display = 'flex';
    tabsContainer.style.gap = '4px';

    const categories = [
        { id: 'all', label: 'ALLE' },
        { id: 'commodity', label: '🛢️ ROHSTOFFE' },
        { id: 'crypto', label: '₿ KRYPTO' },
        { id: 'index', label: '📈 INDIZES & AKTIEN' },
        { id: 'forex', label: '💱 DEVISEN' }
    ];

    let activeCategory = 'all';
    let searchQuery = '';

    categories.forEach(cat => {
        const tabBtn = document.createElement('button');
        tabBtn.type = 'button';
        tabBtn.className = `sphere-app-action ${cat.id === activeCategory ? 'active' : ''}`;
        tabBtn.style.fontSize = '9px';
        tabBtn.style.padding = '4px 8px';
        tabBtn.textContent = cat.label;
        tabBtn.addEventListener('click', () => {
            tabsContainer.querySelectorAll('button').forEach(b => b.classList.remove('active'));
            tabBtn.classList.add('active');
            activeCategory = cat.id;
            filterAndRenderQuotes();
        });
        tabsContainer.appendChild(tabBtn);
    });

    const refreshBtn = document.createElement('button');
    refreshBtn.type = 'button';
    refreshBtn.className = 'sphere-app-action';
    refreshBtn.textContent = '🔄 AKTUALISIEREN';
    topRow.append(tabsContainer, refreshBtn);

    const searchInput = document.createElement('input');
    searchInput.type = 'text';
    searchInput.placeholder = 'Symbol oder Name suchen (z.B. Öl, Brent, NVDA, Gold)...';
    searchInput.className = 'font-mono';
    searchInput.style.background = 'rgba(0,0,0,0.5)';
    searchInput.style.border = '1px solid rgba(0,240,255,0.2)';
    searchInput.style.color = '#fff';
    searchInput.style.padding = '6px 10px';
    searchInput.style.fontSize = '11px';
    searchInput.style.borderRadius = '6px';
    searchInput.style.outline = 'none';

    searchInput.addEventListener('input', (e) => {
        searchQuery = e.target.value.toLowerCase().trim();
        filterAndRenderQuotes();
    });

    toolbar.append(topRow, searchInput);

    const grid = document.createElement('div');
    grid.className = 'sphere-market-grid';
    grid.style.display = 'grid';
    grid.style.gridTemplateColumns = 'repeat(auto-fill, minmax(170px, 1fr))';
    grid.style.gap = '10px';
    grid.style.overflowY = 'auto';
    grid.style.maxHeight = '280px';

    const note = document.createElement('p');
    note.className = 'sphere-market-note';
    note.style.fontSize = '9px';
    note.style.color = 'var(--vgt-text-dim)';
    note.style.marginTop = '8px';
    note.textContent = 'Live-Kurse via Yahoo Finance & CoinGecko. 30s Multi-Source Cache aktiv.';

    let rawQuotes = [];

    function filterAndRenderQuotes() {
        grid.replaceChildren();
        let filtered = rawQuotes;

        if (activeCategory !== 'all') {
            filtered = filtered.filter(q => q.category === activeCategory);
        }

        if (searchQuery) {
            filtered = filtered.filter(q => 
                (q.symbol && q.symbol.toLowerCase().includes(searchQuery)) ||
                (q.name && q.name.toLowerCase().includes(searchQuery)) ||
                (q.instrument_note && q.instrument_note.toLowerCase().includes(searchQuery))
            );
        }

        if (filtered.length === 0) {
            const empty = document.createElement('span');
            empty.className = 'settings-empty-state';
            empty.textContent = 'Keine passenden Marktwerte gefunden.';
            grid.appendChild(empty);
            return;
        }

        for (const quote of filtered) {
            const card = document.createElement('article');
            const isUp = Number(quote.change_24h_percent) >= 0;
            card.className = `sphere-market-card ${isUp ? 'is-up' : 'is-down'}`;
            card.style.background = isUp ? 'rgba(57, 255, 20, 0.04)' : 'rgba(255, 0, 79, 0.04)';
            card.style.border = `1px solid ${isUp ? 'rgba(57, 255, 20, 0.25)' : 'rgba(255, 0, 79, 0.25)'}`;
            card.style.borderRadius = '8px';
            card.style.padding = '10px 12px';
            card.style.display = 'flex';
            card.style.flexDirection = 'column';
            card.style.gap = '4px';

            const identity = document.createElement('div');
            identity.style.display = 'flex';
            identity.style.justifyContent = 'space-between';
            identity.style.alignItems = 'baseline';

            const symbol = document.createElement('strong');
            symbol.style.color = '#fff';
            symbol.style.fontSize = '12px';
            symbol.textContent = quote.symbol;

            const catBadge = document.createElement('small');
            catBadge.style.fontSize = '8px';
            catBadge.style.fontFamily = 'var(--font-mono)';
            catBadge.style.color = 'var(--vgt-cyan)';
            catBadge.textContent = (quote.category || 'market').toUpperCase();

            identity.append(symbol, catBadge);

            const name = document.createElement('small');
            name.style.fontSize = '10px';
            name.style.color = 'var(--vgt-text-dim)';
            name.style.whiteSpace = 'nowrap';
            name.style.overflow = 'hidden';
            name.style.textOverflow = 'ellipsis';
            name.textContent = quote.name;

            const price = document.createElement('b');
            price.style.fontSize = '14px';
            price.style.color = '#fff';
            
            const currSymbol = quote.currency === 'EUR' ? '€' : (quote.currency === 'JPY' ? '¥' : '$');
            const pVal = Number(quote.price);
            const formattedPrice = pVal < 5 ? pVal.toFixed(4) : pVal.toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
            price.textContent = `${currSymbol}${formattedPrice}`;

            const change = document.createElement('span');
            const delta = Number(quote.change_24h_percent || 0);
            change.style.fontSize = '10px';
            change.style.fontWeight = 'bold';
            change.style.fontFamily = 'var(--font-mono)';
            change.style.color = isUp ? 'var(--vgt-green)' : 'var(--vgt-red)';
            change.textContent = `${delta >= 0 ? '+' : ''}${delta.toFixed(2)}% / 24H`;

            card.append(identity, name, price, change);

            if (quote.instrument_note) {
                const noteEl = document.createElement('small');
                noteEl.style.fontSize = '8px';
                noteEl.style.color = 'rgba(255,255,255,0.4)';
                noteEl.textContent = quote.instrument_note;
                card.append(noteEl);
            }

            grid.appendChild(card);
        }
    }

    const fetchQuotes = async () => {
        refreshBtn.disabled = true;
        grid.replaceChildren();
        const loading = document.createElement('span');
        loading.className = 'settings-empty-state';
        loading.textContent = 'Marktdaten für Öl, Rohstoffe, Crypto & Aktien werden abgerufen…';
        grid.appendChild(loading);

        try {
            const response = await fetch(`${state.API_BASE}/v1/markets`);
            if (!response.ok) throw new Error((await response.text()).slice(0, 120));
            const payload = await response.json();
            rawQuotes = payload.quotes || [];
            filterAndRenderQuotes();
        } catch (error) {
            grid.replaceChildren();
            const failure = document.createElement('span');
            failure.className = 'settings-empty-state';
            failure.textContent = `Marktdaten nicht verfügbar: ${error.message}`;
            grid.appendChild(failure);
        } finally {
            refreshBtn.disabled = false;
        }
    };

    refreshBtn.addEventListener('click', () => { void fetchQuotes(); });
    body.append(toolbar, grid, note);
    void fetchQuotes();
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
    if (appID === 'editor' || appID === 'browser' || appID === 'terminal') {
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
    if (appID === 'market') populateMarketApp(body);
    if (appID === 'tasks') void populateTasksApp(body);
    if (appID === 'live') populateLiveApp(body);
    if (appID === 'notes') populateNotesApp(body);
}

function populateNotesApp(body) {
    if (!body || body.dataset.notesReady === 'true') return;
    body.dataset.notesReady = 'true';
    body.innerHTML = '';
    const ta = document.createElement('textarea');
    ta.className = 'sphere-notes-area';
    ta.placeholder = 'Scratch notes — lokal im Browser (nicht Nexus).';
    ta.value = localStorage.getItem('aethel_sphere_notes') || '';
    ta.style.cssText = 'width:100%;height:100%;min-height:200px;resize:none;background:rgba(0,0,0,0.35);border:1px solid rgba(255,255,255,0.08);color:#fff;padding:12px;font:12px/1.5 var(--font-mono);border-radius:8px;';
    ta.addEventListener('input', () => localStorage.setItem('aethel_sphere_notes', ta.value));
    body.appendChild(ta);
}

/** Pure helpers for export (unit-tested via goja/static). */
export function htmlToMarkdownRough(html) {
    const tmp = document.createElement('div');
    tmp.innerHTML = sanitizeRichText(html || '');
    const walk = (node) => {
        if (node.nodeType === Node.TEXT_NODE) return node.textContent || '';
        if (node.nodeType !== Node.ELEMENT_NODE) return '';
        const tag = node.tagName.toLowerCase();
        const inner = Array.from(node.childNodes).map(walk).join('');
        if (tag === 'h1') return `# ${inner.trim()}\n\n`;
        if (tag === 'h2') return `## ${inner.trim()}\n\n`;
        if (tag === 'h3') return `### ${inner.trim()}\n\n`;
        if (tag === 'p' || tag === 'div') return `${inner.trim()}\n\n`;
        if (tag === 'br') return '\n';
        if (tag === 'li') return `- ${inner.trim()}\n`;
        if (tag === 'strong' || tag === 'b') return `**${inner}**`;
        if (tag === 'em' || tag === 'i') return `*${inner}*`;
        return inner;
    };
    return Array.from(tmp.childNodes).map(walk).join('').trim() + '\n';
}

export function buildSphereDocumentExport(html, format) {
    const clean = sanitizeRichText(html || '');
    if (format === 'md' || format === 'markdown') {
        return { filename: 'aethel-sphere-document.md', mime: 'text/markdown;charset=utf-8', body: htmlToMarkdownRough(clean) };
    }
    const wrapped = `<!DOCTYPE html><html><head><meta charset="utf-8"><title>Aethel Document</title></head><body>${clean}</body></html>`;
    return { filename: 'aethel-sphere-document.html', mime: 'text/html;charset=utf-8', body: wrapped };
}

function sphereWriterFilename(title, extension) {
    const base = String(title || 'aethel-sphere-document')
        .normalize('NFKC')
        .replace(/[^\p{L}\p{N}._-]+/gu, '-')
        .replace(/^-+|-+$/g, '')
        .slice(0, 80) || 'aethel-sphere-document';
    return `${base}.${extension}`;
}

function downloadBlob(filename, text, mime) {
    const blob = new Blob([text], { type: mime || 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 2000);
}

export function exportSphereDocument(format) {
    const editor = document.getElementById('sphere-editor-area');
    if (!editor) return null;
    const pack = buildSphereDocumentExport(editor.innerHTML, format);
    const title = document.getElementById('sphere-writer-title')?.value || 'aethel-sphere-document';
    pack.filename = sphereWriterFilename(title, format === 'md' ? 'md' : 'html');
    downloadBlob(pack.filename, pack.body, pack.mime);
    return pack;
}

function updateSphereWriterMetrics() {
    const editor = document.getElementById('sphere-editor-area');
    if (!editor) return;
    const text = (editor.textContent || '').trim();
    const words = text ? text.split(/\s+/u).filter(Boolean).length : 0;
    const characters = Array.from(text).length;
    const minutes = words ? Math.max(1, Math.ceil(words / 220)) : 0;
    const set = (id, value) => { const node = document.getElementById(id); if (node) node.textContent = value; };
    set('sphere-writer-word-count', `${words.toLocaleString()} WÖRTER`);
    set('sphere-writer-char-count', `${characters.toLocaleString()} ZEICHEN`);
    set('sphere-writer-read-time', `${minutes} MIN LESEZEIT`);
}

function setupSphereWriterDocumentControls(editorArea, triggerSave) {
    const title = document.getElementById('sphere-writer-title');
    const open = document.getElementById('editor-btn-open');
    const file = document.getElementById('sphere-writer-file');
    const create = document.getElementById('editor-btn-new');
    if (title) {
        title.value = localStorage.getItem('aethel_sphere_document_title') || title.value;
        title.addEventListener('input', () => localStorage.setItem('aethel_sphere_document_title', title.value.trim().slice(0, 120)));
    }
    open?.addEventListener('click', () => file?.click());
    file?.addEventListener('change', async () => {
        const selected = file.files?.[0];
        if (!selected) return;
        if (selected.size > 2 * 1024 * 1024) {
            window.dispatchEvent(new CustomEvent('aethel:toast', { detail: { message: 'Dokument überschreitet 2 MB.', type: 'error' } }));
            file.value = '';
            return;
        }
        const text = await selected.text();
        if (selected.type === 'text/html' || selected.name.toLowerCase().endsWith('.html')) {
            editorArea.innerHTML = sanitizeRichText(text);
        } else {
            const pre = document.createElement('pre');
            pre.textContent = text;
            editorArea.replaceChildren(pre);
        }
        if (title) title.value = selected.name.replace(/\.[^.]+$/, '').slice(0, 120);
        updateSphereWriterMetrics();
        triggerSave();
        file.value = '';
    });
    create?.addEventListener('click', () => {
        if (create.dataset.armed !== 'true') {
            create.dataset.armed = 'true';
            create.textContent = 'NOCHMAL KLICKEN';
            window.setTimeout(() => { create.dataset.armed = 'false'; create.textContent = 'NEU'; }, 3000);
            return;
        }
        create.dataset.armed = 'false';
        create.textContent = 'NEU';
        const heading = document.createElement('h1');
        heading.textContent = 'Neues Dokument';
        const paragraph = document.createElement('p');
        paragraph.textContent = 'Beginne mit dem Schreiben oder weise Aethel an, den Writer zu verwenden.';
        editorArea.replaceChildren(heading, paragraph);
        if (title) title.value = 'Neues Dokument';
        updateSphereWriterMetrics();
        triggerSave();
        editorArea.focus();
    });
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
    window.addEventListener('aethel:sphere-market', () => {
        const sphereNav = document.getElementById('nav-btn-sphere');
        sphereNav?.click();
        window.setTimeout(() => openSphereApp('market'), 80);
    });
    window.addEventListener('aethel:sphere-document-updated', async () => {
        const editor = document.getElementById('sphere-editor-area');
        if (!editor) return;
        try {
            const response = await fetch(`${state.API_BASE}/v1/sphere/document`, { cache: 'no-store' });
            if (!response.ok) return;
            editor.innerHTML = sanitizeRichText(await response.text());
            updateSphereWriterMetrics();
            openSphereApp('editor');
        } catch (error) {
            console.warn('Writer live synchronization failed', error);
        }
    });
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

    document.getElementById('editor-btn-export-html')?.addEventListener('click', (e) => {
        e.preventDefault();
        exportSphereDocument('html');
    });
    document.getElementById('editor-btn-export-md')?.addEventListener('click', (e) => {
        e.preventDefault();
        exportSphereDocument('md');
    });
    
    // XSS Sanitizer on paste & input to ensure security
    const editorArea = document.getElementById("sphere-editor-area");
    if (editorArea) {
        let saveTimeout = null;
        
        const saveDocument = () => {
            const html = sanitizeRichText(editorArea.innerHTML);
            const saveState = document.getElementById('sphere-writer-save-state');
            if (saveState) saveState.textContent = 'SPEICHERT …';
            fetch(`${state.API_BASE}/v1/sphere/document`, {
                method: "POST",
                headers: { "Content-Type": "text/html" },
                body: html
            }).then(response => {
                if (!response.ok) throw new Error(`HTTP ${response.status}`);
                if (saveState) saveState.textContent = 'GESPEICHERT';
            }).catch(e => {
                if (saveState) saveState.textContent = 'SPEICHERFEHLER';
                console.warn("Failed to save sphere document", e);
            });
        };

        const triggerDebouncedSave = () => {
            if (saveTimeout) clearTimeout(saveTimeout);
            saveTimeout = setTimeout(saveDocument, 800);
        };

        editorArea.addEventListener("input", () => {
            const saveState = document.getElementById('sphere-writer-save-state');
            if (saveState) saveState.textContent = 'UNGESPEICHERT';
            updateSphereWriterMetrics();
            triggerDebouncedSave();
        });
        setupSphereWriterDocumentControls(editorArea, saveDocument);

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
                updateSphereWriterMetrics();
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
    setupWinControls("sphere-window-terminal", "win-terminal-min", "win-terminal-max", "win-terminal-close");

    const terminalWin = document.getElementById("sphere-window-terminal");
    const terminalHeader = document.getElementById("sphere-window-terminal-header");
    if (terminalWin && terminalHeader) {
        makeDraggable(terminalWin, terminalHeader);
        makeResizable(terminalWin);
        terminalWin.addEventListener("mousedown", () => focusWindow(terminalWin));
        terminalWin.addEventListener("touchstart", () => focusWindow(terminalWin), { passive: true });
    }

    const termOutput = document.getElementById("sphere-terminal-output");
    const termInput = document.getElementById("sphere-terminal-input");
    const termRunBtn = document.getElementById("sphere-terminal-run-btn");
    const termBtnTasks = document.getElementById("sphere-term-btn-tasks");
    const termBtnRuns = document.getElementById("sphere-term-btn-runs");
    const termStatus = document.getElementById("sphere-term-status");

    function printToTerminal(text, type = "info") {
        if (!termOutput) return;
        const line = document.createElement("div");
        line.style.marginBottom = "4px";
        if (type === "error") {
            line.style.color = "var(--vgt-red)";
        } else if (type === "success") {
            line.style.color = "var(--vgt-green)";
        } else if (type === "command") {
            line.style.color = "var(--vgt-cyan)";
        } else {
            line.style.color = "#39ff14";
        }
        if (type === "command") {
            line.textContent = `$ ${text}`;
        } else {
            line.textContent = text;
        }
        termOutput.appendChild(line);
        termOutput.scrollTop = termOutput.scrollHeight;
    }

    async function loadTasks() {
        printToTerminal("Abfrage laufender System-Tasks...", "info");
        try {
            const res = await fetch(`${state.API_BASE}/v1/tasks/`);
            if (!res.ok) throw new Error(await res.text());
            const list = await res.json();
            termOutput.textContent = "";
            printToTerminal("=== RUNNING SYSTEM TASKS ===", "success");
            if (list.length === 0) {
                printToTerminal("Keine aktiven Hintergrund-Tasks vorhanden.", "info");
            } else {
                list.forEach(t => {
                    printToTerminal(`[${t.id || 'TASK'}] ${t.text} (${t.status || 'running'})`, "info");
                });
            }
        } catch (err) {
            printToTerminal(`Fehler beim Laden der Tasks: ${err.message}`, "error");
        }
    }

    async function loadRuns() {
        printToTerminal("Abfrage persistenter Agenten-Runs...", "info");
        try {
            const res = await fetch(`${state.API_BASE}/v1/runs`);
            if (!res.ok) throw new Error(await res.text());
            const data = await res.json();
            const runs = data.runs || [];
            termOutput.textContent = "";
            printToTerminal("=== ACTIVE AGENT RUNS ===", "success");
            if (runs.length === 0) {
                printToTerminal("Keine aktiven Agenten-Runs verzeichnet.", "info");
            } else {
                runs.forEach(r => {
                    printToTerminal(`[RUN_${r.id.slice(0,6)}] Ziel: ${r.objective || 'N/A'} - Status: ${r.status}`, "info");
                });
            }
        } catch (err) {
            printToTerminal(`Fehler beim Laden der Agenten-Runs: ${err.message}`, "error");
        }
    }

    async function executeShellCommand() {
        if (!termInput) return;
        const cmd = termInput.value.trim();
        if (!cmd) return;
        termInput.value = "";
        printToTerminal(cmd, "command");
        
        if (cmd.toLowerCase() === "clear" || cmd.toLowerCase() === "cls") {
            termOutput.textContent = "";
            return;
        }
        
        termStatus.textContent = "EXECUTING...";
        try {
            const res = await fetch(`${state.API_BASE}/v1/tasks/`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ text: cmd, objective: cmd })
            });
            if (!res.ok) throw new Error(await res.text());
            const task = await res.json();
            printToTerminal(`Task registriert: ${task.id}`, "success");
            printToTerminal(`Befehl gestartet. Logs im Hintergrund aktiv.`, "info");
        } catch (err) {
            printToTerminal(`Ausführungsfehler: ${err.message}`, "error");
        } finally {
            termStatus.textContent = "READY // LOCAL HOST UPLINK";
        }
    }

    if (termBtnTasks) termBtnTasks.addEventListener("click", loadTasks);
    if (termBtnRuns) termBtnRuns.addEventListener("click", loadRuns);
    if (termRunBtn) termRunBtn.addEventListener("click", executeShellCommand);
    if (termInput) {
        termInput.addEventListener("keydown", (e) => {
            if (e.key === "Enter") {
                executeShellCommand();
            }
        });
    }

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

export async function populateTasksApp(body) {
    if (!body) return;
    body.replaceChildren();

    const container = document.createElement('div');
    container.style.cssText = 'display:flex; flex-direction:column; height:100%; gap:12px; font-family:var(--font-mono); color:#fff;';

    const headerRow = document.createElement('div');
    headerRow.style.cssText = 'display:flex; justify-content:space-between; align-items:center; border-bottom:1px solid rgba(0,240,255,0.15); padding-bottom:10px;';

    const titleBox = document.createElement('div');
    const title = document.createElement('h4');
    title.style.cssText = 'margin:0; font-size:13px; color:var(--vgt-cyan); letter-spacing:0.05em; font-weight:bold;';
    title.textContent = 'AUTOMATIONEN & TERMIN-TASKS';

    const sub = document.createElement('small');
    sub.style.cssText = 'font-size:10px; color:var(--vgt-text-dim);';
    sub.textContent = 'Tägliche / Stündliche Aufgaben mit Vorab-Freigaben & Lagebericht-Popups';
    titleBox.append(title, sub);

    const btnNew = document.createElement('button');
    btnNew.className = 'cyber-button';
    btnNew.style.cssText = 'font-size:10px; padding:6px 12px; background:rgba(0,240,255,0.15); border:1px solid var(--vgt-cyan); color:var(--vgt-cyan); cursor:pointer; width:auto;';
    btnNew.textContent = '+ NEUEN TASK ANLEGEN';
    btnNew.addEventListener('click', () => openTaskCreateModal());

    headerRow.append(titleBox, btnNew);

    const taskListDiv = document.createElement('div');
    taskListDiv.style.cssText = 'flex:1; overflow-y:auto; display:flex; flex-direction:column; gap:10px; padding-right:4px;';

    async function loadTasks() {
        taskListDiv.replaceChildren();
        const loading = document.createElement('div');
        loading.style.cssText = 'font-size:11px; color:var(--vgt-text-dim); padding:10px;';
        loading.textContent = 'Lade geplante Tasks...';
        taskListDiv.appendChild(loading);

        try {
            const res = await fetch(`${state.API_BASE}/v1/kernel/tasks`);
            if (!res.ok) throw new Error(`HTTP ${res.status}`);
            const tasks = await res.json();
            taskListDiv.replaceChildren();

            if (!Array.isArray(tasks) || tasks.length === 0) {
                const empty = document.createElement('div');
                empty.style.cssText = 'font-size:11px; color:var(--vgt-text-dim); text-align:center; padding:30px 10px; border:1px dashed rgba(255,255,255,0.1); border-radius:8px;';
                const message = document.createElement('span');
                message.textContent = 'Keine aktiven Automationen vorhanden.';
                const create = document.createElement('button');
                create.type = 'button';
                create.className = 'vgt-inline-0d188e2f';
                create.textContent = '+ Ersten Task anlegen.';
                create.addEventListener('click', () => window.openTaskCreateModal?.());
                empty.append(message, document.createElement('br'), document.createElement('br'), create);
                taskListDiv.appendChild(empty);
                return;
            }

            for (const task of tasks) {
                const card = document.createElement('article');
                card.style.cssText = 'background:rgba(8,12,24,0.7); border:1px solid rgba(0,240,255,0.2); border-radius:8px; padding:12px; display:flex; flex-direction:column; gap:8px;';

                const cardTop = document.createElement('div');
                cardTop.style.cssText = 'display:flex; justify-content:space-between; align-items:center;';

                const taskTitle = document.createElement('strong');
                taskTitle.style.cssText = 'font-size:12px; color:#fff;';
                taskTitle.textContent = task.text || task.objective;

                const badges = document.createElement('div');
                badges.style.cssText = 'display:flex; gap:6px; align-items:center;';

                const schedBadge = document.createElement('span');
                schedBadge.style.cssText = 'font-size:9px; background:rgba(0,240,255,0.1); border:1px solid var(--vgt-cyan); color:var(--vgt-cyan); padding:2px 6px; border-radius:4px; text-transform:uppercase;';
                schedBadge.textContent = task.schedule_type === 'daily' ? `TÄGLICH ${task.scheduled_time || ''}` : task.schedule_type;

                const statusBadge = document.createElement('span');
                const isRunning = task.status === 'running';
                const isBlocked = task.status === 'blocked';
                statusBadge.style.cssText = `font-size:9px; padding:2px 6px; border-radius:4px; font-weight:bold; ${isRunning ? 'background:rgba(255,123,0,0.2); color:var(--vgt-orange); border:1px solid var(--vgt-orange);' : isBlocked ? 'background:rgba(255,0,79,0.2); color:var(--vgt-red); border:1px solid var(--vgt-red);' : 'background:rgba(57,255,20,0.15); color:var(--vgt-green); border:1px solid var(--vgt-green);'}`;
                statusBadge.textContent = (task.status || 'pending').toUpperCase();

                badges.append(schedBadge, statusBadge);
                cardTop.append(taskTitle, badges);

                const objText = document.createElement('p');
                objText.style.cssText = 'font-size:10px; color:rgba(255,255,255,0.7); margin:0; line-height:1.4;';
                objText.textContent = task.objective;

                const capRow = document.createElement('div');
                capRow.style.cssText = 'display:flex; gap:4px; flex-wrap:wrap; font-size:8px; color:var(--vgt-text-dim);';
                const caps = task.required_capabilities || [];
                if (caps.length > 0) {
                    const capLabel = document.createElement('span');
                    capLabel.textContent = 'Begrenzte Capabilities:';
                    capRow.appendChild(capLabel);
                    for (const c of caps) {
                        const tag = document.createElement('span');
                        tag.style.cssText = 'background:rgba(255,255,255,0.05); border:1px solid rgba(255,255,255,0.1); padding:1px 4px; border-radius:3px; color:var(--vgt-cyan);';
                        tag.textContent = `🛡️ ${c}`;
                        capRow.appendChild(tag);
                    }
                }

                const timingRow = document.createElement('div');
                timingRow.style.cssText = 'display:flex; justify-content:space-between; font-size:9px; color:var(--vgt-text-dim); border-top:1px solid rgba(255,255,255,0.05); padding-top:6px;';
                const nextStr = task.next_run_at ? new Date(task.next_run_at).toLocaleString('de-DE') : 'Sofort / Manuell';
                const lastStr = task.last_run_at && task.last_run_at !== 'never' ? new Date(task.last_run_at).toLocaleString('de-DE') : 'Niemals';
                const next = document.createElement('span');
                next.append('Nächster Lauf: ');
                const nextValue = document.createElement('b');
                nextValue.className = 'vgt-inline-52da9198';
                nextValue.textContent = nextStr;
                next.appendChild(nextValue);
                const last = document.createElement('span');
                last.textContent = `Zuletzt: ${lastStr}`;
                timingRow.append(next, last);

                const actRow = document.createElement('div');
                actRow.style.cssText = 'display:flex; gap:6px; justify-content:flex-end; margin-top:4px;';

                const btnRunNow = document.createElement('button');
                btnRunNow.className = 'cyber-button';
                btnRunNow.style.cssText = 'font-size:9px; padding:3px 8px; background:rgba(0,240,255,0.1); border:1px solid var(--vgt-cyan); color:var(--vgt-cyan); cursor:pointer; width:auto;';
                btnRunNow.textContent = '▶ JETZT AUSFÜHREN';
                btnRunNow.addEventListener('click', async () => {
                    btnRunNow.disabled = true;
                    btnRunNow.textContent = '⏳ STARTET...';
                    try {
                        await fetch(`${state.API_BASE}/v1/kernel/tasks/${task.id}/run`, { method: 'POST' });
                        setTimeout(loadTasks, 1500);
                    } catch(e) { console.error('Task run failed', e); }
                });

                const btnReport = document.createElement('button');
                btnReport.className = 'cyber-button';
                btnReport.style.cssText = 'font-size:9px; padding:3px 8px; background:rgba(255,123,0,0.1); border:1px solid var(--vgt-orange); color:var(--vgt-orange); cursor:pointer; width:auto;';
                btnReport.textContent = '📖 BERICHT ÖFFNEN';
                btnReport.addEventListener('click', () => showTaskResultPopup(task));

                const btnDelete = document.createElement('button');
                btnDelete.className = 'cyber-button';
                btnDelete.style.cssText = 'font-size:9px; padding:3px 8px; background:rgba(255,0,79,0.1); border:1px solid var(--vgt-red); color:var(--vgt-red); cursor:pointer; width:auto;';
                btnDelete.textContent = '🗑 LÖSCHEN';
                btnDelete.addEventListener('click', async () => {
                    if (confirm(`Task "${task.text}" wirklich löschen?`)) {
                        await fetch(`${state.API_BASE}/v1/kernel/tasks/${task.id}`, { method: 'DELETE' });
                        loadTasks();
                    }
                });

                actRow.append(btnRunNow, btnReport, btnDelete);
                card.append(cardTop, objText, capRow, timingRow, actRow);
                taskListDiv.appendChild(card);
            }
        } catch(e) {
            const failure = document.createElement('div');
            failure.className = 'vgt-inline-ac122a37';
            failure.textContent = `Fehler beim Laden der Tasks: ${String(e.message || 'unknown error')}`;
            taskListDiv.replaceChildren(failure);
        }
    }

    container.append(headerRow, taskListDiv);
    body.appendChild(container);
    void loadTasks();
}

export function openTaskCreateModal() {
    let dialog = document.getElementById('task-create-dialog');
    if (dialog) dialog.remove();

    dialog = document.createElement('div');
    dialog.id = 'task-create-dialog';
    dialog.style.cssText = 'position:fixed; inset:0; z-index:9999; background:rgba(4,6,14,0.85); backdrop-filter:blur(10px); display:flex; justify-content:center; align-items:center; font-family:var(--font-mono);';

    const modal = document.createElement('div');
    modal.style.cssText = 'width:90%; max-width:520px; background:linear-gradient(145deg, rgba(8,14,28,0.98), rgba(4,6,14,0.99)); border:1px solid var(--vgt-cyan); border-radius:12px; padding:20px; box-shadow:0 0 40px rgba(0,240,255,0.2); display:flex; flex-direction:column; gap:12px; color:#fff;';

    modal.innerHTML = `
        <div class="vgt-inline-6b4e7c6b">
            <strong class="vgt-inline-4ca260fc">⏰ NEUEN AUTOMATIONSTASK ANLEGEN</strong>
            <button id="task-modal-close" class="vgt-inline-9efd323e">✕</button>
        </div>
        <div class="vgt-inline-e5661c30">
            <div>
                <label class="vgt-inline-516019d4">TASK-TITEL / BEZEICHNUNG</label>
                <input id="task-input-title" type="text" placeholder="z.B. Täglicher Markt- & Lagebericht" class="vgt-inline-722eff9d" />
            </div>
            <div>
                <label class="vgt-inline-516019d4">ZIEL / KI-ANWEISUNG (OBJECTIVE)</label>
                <textarea id="task-input-objective" rows="3" placeholder="z.B. Prüfe stündlich die aktuellen Ölpreise und erstelle bei Veränderungen ein Lagebriefing in Sphere." class="vgt-inline-4950302e"></textarea>
            </div>
            <div class="vgt-inline-6eecb81c">
                <div>
                    <label class="vgt-inline-516019d4">RHYTHMUS / INTERVALL</label>
                    <select id="task-input-schedule" class="vgt-inline-647fcb4b">
                        <option value="daily">Täglich (Uhrzeit festlegen)</option>
                        <option value="hourly">Stündlich</option>
                        <option value="weekly">Wöchentlich</option>
                        <option value="interval">Intervall (Sekunden)</option>
                        <option value="cron">Cron (5 Felder, eingeschränkt)</option>
                        <option value="once">Einmalig (Sofort)</option>
                    </select>
                </div>
                <div>
                    <label class="vgt-inline-516019d4">UHRZEIT (BEI TÄGLICH)</label>
                    <input id="task-input-time" type="time" value="09:00" class="vgt-inline-09b6035a" />
                </div>
            </div>
            <div class="vgt-inline-6eecb81c">
                <div>
                    <label class="vgt-inline-516019d4">INTERVALLSEKUNDEN (60–604800)</label>
                    <input id="task-input-interval" type="number" min="60" max="604800" value="3600" class="vgt-inline-722eff9d" />
                </div>
                <div>
                    <label class="vgt-inline-516019d4">CRON: MINUTE STUNDE TAG MONAT WOCHENTAG</label>
                    <input id="task-input-cron" type="text" value="0 9 * * 1" class="vgt-inline-722eff9d" />
                </div>
            </div>
            <div class="vgt-inline-cf9a5227">
                <label class="vgt-inline-6762f599">🛡️ AUTONOME LESE- UND COLLECTOR-CAPABILITIES</label>
                <div class="vgt-inline-8a6a69c8">
                    <label class="vgt-inline-35f83c40"><input type="checkbox" class="task-cap-chk vgt-inline-dcf5a24b" value="intelligence.read" checked /> Intelligence lesen</label>
                    <label class="vgt-inline-35f83c40"><input type="checkbox" class="task-cap-chk vgt-inline-dcf5a24b" value="market.read" /> Märkte lesen</label>
                    <label class="vgt-inline-35f83c40"><input type="checkbox" class="task-cap-chk vgt-inline-dcf5a24b" value="weather.read" /> Wetter lesen</label>
                    <label class="vgt-inline-35f83c40"><input type="checkbox" class="task-cap-chk vgt-inline-dcf5a24b" value="browser.open_url" /> Öffentliche Web-Recherche</label>
                    <label class="vgt-inline-35f83c40"><input type="checkbox" class="task-cap-chk vgt-inline-dcf5a24b" value="intelligence.manage_sources" /> Policy-gesteuerte Connectoren</label>
                </div>
            </div>
            <div class="vgt-inline-7fa61ab6">
                <input type="checkbox" id="task-input-popup" checked class="vgt-inline-dcf5a24b" />
                <label for="task-input-popup" class="vgt-inline-0a54f2d3">Lagebericht-Popup nach Fertigstellung auf dem Desktop anzeigen</label>
            </div>
        </div>
        <div class="vgt-inline-ef13aea9">
            <button id="task-modal-cancel" class="cyber-button vgt-inline-8d72fd87">ABBRECHEN</button>
            <button id="task-modal-save" class="cyber-button vgt-inline-6ae288ca">BEGRENZTEN TASK ANLEGEN</button>
        </div>
    `;

    dialog.appendChild(modal);
    document.body.appendChild(dialog);

    dialog.querySelector('#task-modal-close').addEventListener('click', () => dialog.remove());
    dialog.querySelector('#task-modal-cancel').addEventListener('click', () => dialog.remove());

    dialog.querySelector('#task-modal-save').addEventListener('click', async () => {
        const title = dialog.querySelector('#task-input-title').value.trim();
        const objective = dialog.querySelector('#task-input-objective').value.trim();
        const schedule = dialog.querySelector('#task-input-schedule').value;
        const timeVal = dialog.querySelector('#task-input-time').value;
        const intervalSeconds = Number.parseInt(dialog.querySelector('#task-input-interval').value, 10);
        const cronExpression = dialog.querySelector('#task-input-cron').value.trim();
        const popup = dialog.querySelector('#task-input-popup').checked;

        if (!title || !objective) {
            alert('Bitte Titel und Anweisung eingeben.');
            return;
        }

        const selectedCaps = Array.from(dialog.querySelectorAll('.task-cap-chk:checked')).map(c => c.value);

        const payload = {
            text: title,
            objective: objective,
            schedule_type: schedule,
            scheduled_time: timeVal,
            required_capabilities: selectedCaps,
            interval_seconds: intervalSeconds,
            cron_expression: cronExpression,
            notify_popup: popup
        };

        try {
            const res = await fetch(`${state.API_BASE}/v1/kernel/tasks`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            if (res.ok) {
                dialog.remove();
                if (window.openSphereApp) window.openSphereApp('tasks');
            } else {
                alert('Fehler beim Erstellen des Tasks.');
            }
        } catch(e) {
            alert('Fehler: ' + e.message);
        }
    });
}
window.openTaskCreateModal = openTaskCreateModal;

export function showTaskResultPopup(task) {
    let popup = document.getElementById('task-result-popup-dialog');
    if (popup) popup.remove();

    popup = document.createElement('div');
    popup.id = 'task-result-popup-dialog';
    popup.style.cssText = 'position:fixed; inset:0; z-index:10000; background:rgba(4,6,14,0.85); backdrop-filter:blur(10px); display:flex; justify-content:center; align-items:center; font-family:var(--font-mono);';

    const contentDiv = document.createElement('div');
    contentDiv.style.cssText = 'width:90%; max-width:680px; max-height:85vh; background:linear-gradient(145deg, rgba(8,14,28,0.98), rgba(4,6,14,0.99)); border:1px solid var(--vgt-orange); border-radius:12px; padding:20px; box-shadow:0 0 50px rgba(255,123,0,0.25); display:flex; flex-direction:column; gap:12px; color:#fff;';

    const header = document.createElement('div');
    header.style.cssText = 'display:flex; justify-content:space-between; align-items:center; border-bottom:1px solid rgba(255,123,0,0.3); padding-bottom:10px;';

    const title = document.createElement('strong');
    title.style.cssText = 'font-size:13px; color:var(--vgt-orange); letter-spacing:0.05em; display:flex; align-items:center; gap:8px;';
    title.textContent = `AUTOMATISCHER LAGEBERICHT // ${String(task.text || task.objective || '')}`;

    const closeBtn = document.createElement('button');
    closeBtn.style.cssText = 'background:none; border:none; color:var(--vgt-text-dim); font-size:16px; cursor:pointer;';
    closeBtn.textContent = '✕';
    closeBtn.addEventListener('click', () => popup.remove());

    header.append(title, closeBtn);

    const body = document.createElement('div');
    body.style.cssText = 'flex:1; overflow-y:auto; padding:12px; background:rgba(0,0,0,0.4); border:1px solid rgba(255,255,255,0.06); border-radius:8px; font-size:12px; line-height:1.6; color:rgba(255,255,255,0.9); white-space:pre-wrap;';
    body.innerHTML = formatMarkdown(task.last_report || task.LastReport || 'Kein Lagebericht vorhanden.');

    const footer = document.createElement('div');
    footer.style.cssText = 'display:flex; justify-content:space-between; align-items:center; border-top:1px solid rgba(255,255,255,0.1); padding-top:10px; font-size:10px; color:var(--vgt-text-dim);';
    const executed = document.createElement('span');
    executed.textContent = `Ausgeführt: ${task.last_run_at ? new Date(task.last_run_at).toLocaleString('de-DE') : 'Kürzlich'}`;
    footer.appendChild(executed);

    const btnClose = document.createElement('button');
    btnClose.className = 'cyber-button';
    btnClose.style.cssText = 'font-size:10px; padding:6px 14px; background:rgba(255,123,0,0.2); border:1px solid var(--vgt-orange); color:var(--vgt-orange); cursor:pointer; width:auto;';
    btnClose.textContent = 'SCHLIESSEN';
    btnClose.addEventListener('click', () => popup.remove());

    footer.appendChild(btnClose);
    contentDiv.append(header, body, footer);
    popup.appendChild(contentDiv);
    document.body.appendChild(popup);
}
window.showTaskResultPopup = showTaskResultPopup;
