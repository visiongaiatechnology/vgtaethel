import { state } from '../state.js';
import {
  globeContainer, setGlobeContainer,
  activeFeedEvents,
  cameraData,
  satData,
  cachedRiskMarkers,
  globalWatchPreferences, updateGlobalWatchPreferences,
  globalWatchDefaults,
  REGION_GEO,
  localGlobeSelectedIndex, setLocalGlobeSelectedIndex,
  localGlobeScale, localGlobeCanvas,
  globeKnownEventIDs, setGlobeKnownEventIDs,
  globeTransientEventTimer, setGlobeTransientEventTimer,
  globeTransientEventID, setGlobeTransientEventID,
  globeIdleRotationBlockedUntil, setGlobeIdleRotationBlockedUntil,
  localGlobeRotY, localGlobeRotX,
  globalWatchAutoRefreshTimer, setGlobalWatchAutoRefreshTimer,
  globalWatchHazardTimer, setGlobalWatchHazardTimer
} from './state.js';
import {
  safeExternalURL,
  appendTextElement,
  clampGwPanelPosition,
  filterEventsByRegion,
  filterEventsByTimeWindow,
  eventMatchesRegion,
  isGeoEvent,
} from './projection.js';
import {
  visibleLayers
} from './layers.js';
import {
  initPureLocalGlobe,
  drawPureLocalGlobe,
  requestGlobeRender,
  focusGlobeOnLonLat,
  highlightEventInList,
  forceGlobeResize,
} from './globe_render.js';
import {
  refreshOSINTFeed,
  loadAndRenderRegionalRisks,
  openExplainDrawer,
  loadAndRenderPersonalImpact,
  syncPersonalAndRefreshImpact,
  loadAndRenderAlerts,
  subscribeGlobalWatchCommands,
  getGwTimeWindowHours,
  setGwTimeWindowHours,
  eventTimestampMs,
  activeGlobalWatchDomain,
  stableGlobeEventID,
  focusRegionByKey,
} from './feed_and_risks.js';
import {
  showSelectionDetails,
  showCameraDetails,
  promoteCurrentSelection,
  saveSelectionToNexus,
  aiBriefingForSelection,
  observeAtSelection,
  wireSelectedEventAIActions,
} from './selection_and_chat.js';
import {
  triggerAIBriefing,
  wireBriefingWorkspace,
  openGwReportReader,
  downloadTextFile
} from './briefing_and_reader.js';

// Public re-exports for app.js / ui.js entry surface
export { refreshOSINTFeed, loadAndRenderRegionalRisks, forceGlobeResize, focusRegionByKey };

export function showAethelToast(message, type = 'info') {
    const existing = document.getElementById('aethel-toast');
    if (existing) existing.remove();

    const toast = document.createElement('div');
    toast.id = 'aethel-toast';

    let borderColor = 'rgba(0,240,255,0.8)';
    let glowColor = 'rgba(0,240,255,0.2)';
    let title = 'INFO';
    if (type === 'success') {
        borderColor = 'rgba(57,255,20,0.8)';
        glowColor = 'rgba(57,255,20,0.2)';
        title = 'SUCCESS';
    } else if (type === 'error') {
        borderColor = 'rgba(255,0,79,0.8)';
        glowColor = 'rgba(255,0,79,0.2)';
        title = 'ALERT';
    }

    toast.style.cssText = `
        position: fixed;
        bottom: 24px;
        right: 24px;
        z-index: 10000;
        min-width: 250px;
        max-width: 350px;
        padding: 12px 16px;
        background: rgba(8,14,28,0.96);
        border: 1px solid ${borderColor};
        border-radius: 6px;
        box-shadow: 0 0 20px ${glowColor};
        color: #fff;
        font-family: var(--font-mono, monospace);
        font-size: 10px;
        opacity: 0;
        transform: translateY(20px);
        transition: opacity 0.3s ease, transform 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275);
    `;

    const head = document.createElement('div');
    head.style.cssText = `font-weight:bold; color:${borderColor.replace('0.8', '1')}; margin-bottom:4px; letter-spacing:0.08em;`;
    head.textContent = 'AETHEL • ' + title;
    const body = document.createElement('div');
    body.style.cssText = 'font-size:11px; line-height:1.4; color:#fff;';
    body.textContent = String(message ?? '');
    toast.replaceChildren(head, body);

    document.body.appendChild(toast);

    setTimeout(() => {
        toast.style.opacity = '1';
        toast.style.transform = 'translateY(0)';
    }, 50);

    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateY(20px)';
        setTimeout(() => toast.remove(), 300);
    }, 4500);
}

export function showAethelModal({ title, contentHtml, contentNode, onSave }) {
    const existing = document.getElementById('aethel-custom-modal');
    if (existing) existing.remove();

    const overlay = document.createElement('div');
    overlay.id = 'aethel-custom-modal';
    overlay.style.cssText = 'position:fixed; top:0; left:0; width:100vw; height:100vh; background:rgba(2,4,12,0.85); backdrop-filter:blur(8px); z-index:9999; display:flex; align-items:center; justify-content:center; opacity:0; transition:opacity 0.2s ease;';

    const card = document.createElement('div');
    card.className = 'glass-card';
    card.style.cssText = 'background:rgba(8,14,28,0.97); border:1px solid rgba(0,240,255,0.4); border-radius:10px; box-shadow:0 0 35px rgba(0,240,255,0.25); padding:24px; min-width:340px; max-width:500px; width:90%; transform:scale(0.95); transition:transform 0.2s cubic-bezier(0.175, 0.885, 0.32, 1.15); color:#fff; font-family:var(--font-mono, monospace); position:relative;';

    const closeBtn = document.createElement('button');
    closeBtn.id = 'aethel-modal-close';
    closeBtn.type = 'button';
    closeBtn.textContent = '×';
    closeBtn.style.cssText = 'position:absolute; top:12px; right:12px; background:none; border:none; color:rgba(255,255,255,0.6); cursor:pointer; font-size:16px; line-height:1; font-family:var(--font-mono);';

    const brand = document.createElement('div');
    brand.style.cssText = 'font-size:9px; color:#0ff; margin-bottom:6px; letter-spacing:0.1em; font-weight:bold;';
    brand.textContent = 'AETHEL CORE // SYSTEM INTERFACE';

    const titleEl = document.createElement('div');
    titleEl.style.cssText = 'font-size:13px; font-weight:700; color:#fff; margin-bottom:18px; border-bottom:1px solid rgba(0,240,255,0.2); padding-bottom:8px;';
    titleEl.textContent = String(title ?? '');

    const body = document.createElement('div');
    body.id = 'aethel-modal-body';
    body.style.cssText = 'font-size:11px; margin-bottom:20px; line-height:1.45;';
    if (contentNode) {
        body.appendChild(contentNode);
    } else if (contentHtml) {
        body.innerHTML = contentHtml;
    }

    const errEl = document.createElement('div');
    errEl.id = 'aethel-modal-error';
    errEl.style.cssText = 'color:#ff004f; font-size:10px; margin-bottom:12px; display:none; min-height:15px;';

    const actions = document.createElement('div');
    actions.style.cssText = 'display:flex; justify-content:flex-end; gap:10px;';
    const cancelBtn = document.createElement('button');
    cancelBtn.id = 'aethel-modal-cancel';
    cancelBtn.type = 'button';
    cancelBtn.className = 'cyber-button';
    cancelBtn.style.cssText = 'font-size:10px; padding:6px 12px; background:rgba(255,255,255,0.05); border-color:rgba(255,255,255,0.2); color:#fff; width:auto;';
    cancelBtn.textContent = 'CANCEL';
    const submitBtn = document.createElement('button');
    submitBtn.id = 'aethel-modal-submit';
    submitBtn.type = 'button';
    submitBtn.className = 'cyber-button';
    submitBtn.style.cssText = 'font-size:10px; padding:6px 12px; width:auto; border-color:#0ff; color:#0ff;';
    submitBtn.textContent = 'CONFIRM';
    actions.append(cancelBtn, submitBtn);

    card.append(closeBtn, brand, titleEl, body, errEl, actions);
    overlay.appendChild(card);
    document.body.appendChild(overlay);

    setTimeout(() => {
        overlay.style.opacity = '1';
        card.style.transform = 'scale(1)';
    }, 10);

    const close = () => {
        overlay.style.opacity = '0';
        card.style.transform = 'scale(0.95)';
        setTimeout(() => overlay.remove(), 200);
    };

    closeBtn.onclick = close;
    cancelBtn.onclick = close;

    overlay.onclick = (e) => {
        if (e.target === overlay) close();
    };

    submitBtn.onclick = async () => {
        errEl.style.display = 'none';
        try {
            submitBtn.disabled = true;
            submitBtn.textContent = 'PROCESSING...';
            if (typeof onSave === 'function') {
                await onSave(close);
            } else {
                close();
            }
        } catch (err) {
            errEl.textContent = (err && err.message) ? err.message : 'An error occurred';
            errEl.style.display = 'block';
            submitBtn.disabled = false;
            submitBtn.textContent = 'CONFIRM';
        }
    };
}

window.showAethelModal = showAethelModal;
window.showAethelToast = showAethelToast;

export function updateLayerCounts() {
    const setTxt = (id, v) => { const el = document.getElementById(id); if (el) el.textContent = String(v); };
    setTxt('gw-cnt-cameras', cameraData.length);
    setTxt('gw-cnt-sats', satData.length);
    setTxt('gw-cnt-news', (activeFeedEvents || []).length);
    setTxt('gw-cnt-risks', (typeof cachedRiskMarkers !== 'undefined' && cachedRiskMarkers) ? cachedRiskMarkers.length : '—');
    setTxt('gw-cnt-borders', visibleLayers.borders ? 'ON' : 'OFF');
}

export function wireRegionChips() {
    const chips = document.querySelectorAll('.gw-region-chip[data-focus]');
    chips.forEach(chip => {
        if (chip._gwBound) return;
        chip._gwBound = true;
        chip.addEventListener('click', () => {
            chips.forEach(c => c.classList.remove('active'));
            chip.classList.add('active');
            const key = chip.getAttribute('data-focus') || 'global';
            const f = focusRegionByKey(key);
            const status = document.getElementById('gw-refresh-info');
            if (status) status.textContent = `FOCUS: ${f.label || key.toUpperCase()}`;
        });
    });
}

export function wireGlobalSearch() {
    const g = document.getElementById('gw-global-search');
    const feed = document.getElementById('gw-feed-search');
    if (g && !g._gwBound) {
        g._gwBound = true;
        g.addEventListener('input', () => {
            if (feed) {
                feed.value = g.value;
                feed.dispatchEvent(new Event('input', { bubbles: true }));
            }
        });
    }
}

export function wireTimeline() {
    const range = document.getElementById('gw-timeline-range');
    if (!range || range._gwBound) return;
    range._gwBound = true;
    range.addEventListener('input', () => {
        window.__gwTimelineRecency = Number(range.value) / 100;
        const active = document.querySelector('.gw-domain-filter.active');
        const domain = active ? active.getAttribute('data-domain') : 'all';
        void refreshOSINTFeed(domain || 'all', false, showSelectionDetails, openGwReportReader);
    });
}

export function wireGwCollapsibles() {
    document.querySelectorAll('[data-gw-collapse-btn]').forEach(btn => {
        if (btn._gwBound) return;
        btn._gwBound = true;
        btn.addEventListener('click', (e) => {
            e.preventDefault();
            const key = btn.getAttribute('data-gw-collapse-btn');
            const body = document.querySelector(`[data-gw-collapse-body="${key}"]`);
            if (!body) return;
            const open = !body.classList.contains('collapsed');
            body.classList.toggle('collapsed', open);
            btn.setAttribute('aria-expanded', open ? 'false' : 'true');
            const chev = btn.querySelector('.gw-collapse-chevron');
            if (chev) chev.textContent = open ? '▸' : '▾';
        });
    });
}

export function presentNewGlobeEvent(events) {
    const currentIDs = new Set((events || []).map(stableGlobeEventID));
    if (globeKnownEventIDs === null) {
        setGlobeKnownEventIDs(currentIDs);
        return;
    }
    const candidates = filterEventsByTimeWindow(events, getGwTimeWindowHours())
        .filter(event => eventMatchesRegion(event, window.__gwRegionFilter || 'global'))
        .filter(event => !globeKnownEventIDs.has(stableGlobeEventID(event)) && isGeoEvent(event))
        .sort((a, b) => eventTimestampMs(b) - eventTimestampMs(a));
    setGlobeKnownEventIDs(currentIDs);
    const event = candidates[0];
    if (!event) return;
    const index = events.indexOf(event);
    const id = stableGlobeEventID(event);
    setGlobeTransientEventID(id);
    setLocalGlobeSelectedIndex(index);
    focusGlobeOnLonLat(event.lon ?? event.longitude, event.lat ?? event.latitude, { scale: Math.max(1.1, localGlobeScale) });
    setGlobeIdleRotationBlockedUntil(performance.now() + 10000);
    const width = localGlobeCanvas?.clientWidth || 640;
    const height = localGlobeCanvas?.clientHeight || 480;
    const pin = { x: width / 2, y: height / 2 };
    showSelectionDetails(event, pin);
    highlightEventInList(index, showSelectionDetails);
    if (globeTransientEventTimer) window.clearTimeout(globeTransientEventTimer);
    const timer = window.setTimeout(() => {
        if (globeTransientEventID !== id) return;
        document.getElementById('gw-selection-panel')?.classList.add('hidden');
        setGlobeTransientEventID('');
        setGlobeIdleRotationBlockedUntil(performance.now());
        requestGlobeRender();
    }, 10000);
    setGlobeTransientEventTimer(timer);
}

export function scheduleGlobalWatchAutoRefresh() {
    if (globalWatchAutoRefreshTimer) window.clearInterval(globalWatchAutoRefreshTimer);
    setGlobalWatchAutoRefreshTimer(null);
    const seconds = Number(globalWatchPreferences.autoRefreshSeconds) || 0;
    const pollLabel = document.getElementById('gw-feed-poll-label');
    if (pollLabel) pollLabel.textContent = seconds > 0 ? `AUTO-SYNC · ${seconds}s` : 'AUTO-SYNC · AUS';
    if (seconds <= 0) return;
    const timer = window.setInterval(() => {
        const view = document.getElementById('view-global-watch');
        if (!view || view.classList.contains('hidden') || document.hidden) return;
        void refreshOSINTFeed(activeGlobalWatchDomain(), false, showSelectionDetails, openGwReportReader);
    }, seconds * 1000);
    setGlobalWatchAutoRefreshTimer(timer);
}

export function scheduleGlobalWatchHazardAnimation() {
    if (globalWatchHazardTimer) window.clearInterval(globalWatchHazardTimer);
    setGlobalWatchHazardTimer(null);
    window.__gwHazardAnimInterval = null;
    if (!globalWatchPreferences.hazardAnimation) return;
    const interval = Math.max(125, Math.round(1000 / globalWatchPreferences.hazardFPS));
    const timer = window.setInterval(() => {
        const view = document.getElementById('view-global-watch');
        if (!view || view.classList.contains('hidden') || document.hidden) return;
        // visibleLayers proxy mirrors globeLayers[prop].visible (earthquakes/volcanoes)
        const needAnimation = Boolean(visibleLayers.earthquakes || visibleLayers.volcanoes);
        if (needAnimation) requestGlobeRender();
    }, interval);
    setGlobalWatchHazardTimer(timer);
    window.__gwHazardAnimInterval = timer;
}

/** Bind #gw-personal-sync / #gw-personal-refresh (settings personal-impact strip). */
export function wirePersonalImpactControls() {
    const syncBtn = document.getElementById('gw-personal-sync');
    if (syncBtn && !syncBtn._gwBound) {
        syncBtn._gwBound = true;
        syncBtn.addEventListener('click', () => { void syncPersonalAndRefreshImpact(); });
    }
    const refreshBtn = document.getElementById('gw-personal-refresh');
    if (refreshBtn && !refreshBtn._gwBound) {
        refreshBtn._gwBound = true;
        refreshBtn.addEventListener('click', () => { void loadAndRenderPersonalImpact(); });
    }
}

export function applyGlobalWatchPreferences(nextPreferences) {
    updateGlobalWatchPreferences(nextPreferences);
    scheduleGlobalWatchAutoRefresh();
    scheduleGlobalWatchHazardAnimation();
    requestGlobeRender();
}

export function wireGlobalWatchRuntimeSettings() {
    const controls = {
        renderQuality: document.getElementById('gw-render-quality'),
        autoRefreshSeconds: document.getElementById('gw-auto-refresh'),
        feedLimit: document.getElementById('gw-feed-limit'),
        clusterMode: document.getElementById('gw-cluster-mode'),
        idleRotation: document.getElementById('gw-idle-rotation'),
        hazardFPS: document.getElementById('gw-hazard-fps'),
        hazardAnimation: document.getElementById('gw-hazard-animation'),
    };
    if (!controls.renderQuality || controls.renderQuality.dataset.bound === 'true') return;

    const renderValues = () => {
        controls.renderQuality.value = globalWatchPreferences.renderQuality;
        controls.autoRefreshSeconds.value = String(globalWatchPreferences.autoRefreshSeconds);
        controls.feedLimit.value = String(globalWatchPreferences.feedLimit);
        controls.clusterMode.value = globalWatchPreferences.clusterMode;
        controls.idleRotation.checked = globalWatchPreferences.idleRotation;
        controls.hazardFPS.value = String(globalWatchPreferences.hazardFPS);
        controls.hazardAnimation.checked = globalWatchPreferences.hazardAnimation;
        controls.hazardFPS.disabled = !globalWatchPreferences.hazardAnimation;
    };
    const collectValues = () => ({
        renderQuality: controls.renderQuality.value,
        autoRefreshSeconds: Number(controls.autoRefreshSeconds.value),
        feedLimit: Number(controls.feedLimit.value),
        clusterMode: controls.clusterMode.value,
        idleRotation: controls.idleRotation.checked,
        hazardFPS: Number(controls.hazardFPS.value),
        hazardAnimation: controls.hazardAnimation.checked,
    });
    Object.values(controls).forEach(control => {
        control.dataset.bound = 'true';
        control.addEventListener('change', () => {
            applyGlobalWatchPreferences(collectValues());
            renderValues();
            if (control === controls.feedLimit) void refreshOSINTFeed(activeGlobalWatchDomain(), false, showSelectionDetails, openGwReportReader);
        });
    });
    const reset = document.getElementById('gw-runtime-reset');
    if (reset && reset.dataset.bound !== 'true') {
        reset.dataset.bound = 'true';
        reset.addEventListener('click', () => {
            applyGlobalWatchPreferences(globalWatchDefaults());
            renderValues();
            void refreshOSINTFeed(activeGlobalWatchDomain(), false, showSelectionDetails, openGwReportReader);
        });
    }
    renderValues();
}

export function injectOSINTControls() {
    const controls = document.getElementById("osint-controls");
    if (!controls || controls._gwBound) return;
    controls._gwBound = true;

    document.getElementById("btn-add-source").onclick = () => {
        const addSrcHtml = `
            <div style="display:flex; flex-direction:column; gap:12px;">
                <div>
                    <label style="display:block; font-size:9px; color:#0ff; margin-bottom:4px; font-weight:700;">SOURCE TYPE</label>
                    <select id="modal-src-type" style="width:100%; font-size:11px; background:#111; border:1px solid rgba(0,240,255,0.3); border-radius:4px; color:#fff; padding:8px; font-family:var(--font-mono);">
                        <option value="rss">RSS / ATOM NEWS</option>
						<option value="telegram">TELEGRAM PUBLIC CHANNEL</option>
                        <option value="earthquake-geojson">EARTHQUAKE GEOJSON (USGS SCHEMA)</option>
                        <option value="volcano-eonet">VOLCANO JSON (NASA EONET SCHEMA)</option>
                    </select>
                </div>
                <div>
                    <label style="display:block; font-size:9px; color:#0ff; margin-bottom:4px; font-weight:700;">SOURCE NAME</label>
                    <input id="modal-src-name" placeholder="Name (e.g. Spiegel Top News)" style="width:100%; font-size:11px; background:#111; border:1px solid rgba(0,240,255,0.3); border-radius:4px; color:#fff; padding:8px; font-family:var(--font-mono);">
                </div>
                <div>
                    <label style="display:block; font-size:9px; color:#0ff; margin-bottom:4px; font-weight:700;">SOURCE URL (HTTPS)</label>
                    <input id="modal-src-url" placeholder="RSS: https://domain/rss.xml · Telegram: https://t.me/s/channel" style="width:100%; font-size:11px; background:#111; border:1px solid rgba(0,240,255,0.3); border-radius:4px; color:#fff; padding:8px; font-family:var(--font-mono);">
                </div>
                <div>
                    <label style="display:block; font-size:9px; color:#0ff; margin-bottom:4px; font-weight:700;">CLASSIFICATION DOMAIN</label>
                    <select id="modal-src-domain" style="width:100%; font-size:11px; background:#111; border:1px solid rgba(0,240,255,0.3); border-radius:4px; color:#fff; padding:8px; font-family:var(--font-mono);">
                        <option value="general">general</option>
                        <option value="geo">geo</option>
                        <option value="cyber">cyber</option>
                        <option value="economic">economic</option>
                        <option value="humanitarian">humanitarian</option>
                    </select>
                </div>
            </div>
        `;

        showAethelModal({
            title: "ADD NEW INTEL SOURCE (RSS)",
            contentHtml: addSrcHtml,
            onSave: async (closeModal) => {
                const name = document.getElementById("modal-src-name").value.trim();
                const type = document.getElementById("modal-src-type").value;
                const url = document.getElementById("modal-src-url").value.trim();
				const domain = (type === 'rss' || type === 'telegram') ? document.getElementById("modal-src-domain").value : 'geo';

                if (!name || !url) throw new Error("Both Name and URL are required.");

                const res = await fetch(`${state.API_BASE}/v1/osint/collectors`, {
                    method: "POST",
                    headers: {"Content-Type": "application/json"},
                    body: JSON.stringify({ name, type, url, domain })
                });
                if (!res.ok) throw new Error(await res.text());

                showAethelToast("Eigene Quelle erfolgreich hinzugefügt! Refresh wird initiiert.", "success");
                closeModal();
                if (window.loadAndRenderCollectors) window.loadAndRenderCollectors();
                
                const active = document.querySelector(".gw-domain-filter.active");
                refreshOSINTFeed(active ? active.getAttribute("data-domain") : "all", true, showSelectionDetails, openGwReportReader);
            }
        });
    };

    document.getElementById("btn-edit-prompt").onclick = () => {
        const currentDefault = "Du bist VGT AETHEL, ein hochmodernes, datenschutzkonformes OSINT-Analysesystem. Analysiere das Lagebild objektiv, strukturiert und mit Fokus auf Quellen und Beweisbarkeit. Gib konkrete Handlungsempfehlungen und schlage Cases vor.";
        const existing = window.__osintCustomPrompt || currentDefault;
        
        const editPromptHtml = `
            <div style="display:flex; flex-direction:column; gap:8px;">
                <label style="display:block; font-size:9px; color:#9d4edd; font-weight:700;">SYSTEM INSTRUCTIONS FOR BRIEFING GENERATION</label>
                <textarea id="modal-prompt-text" style="width:100%; height:130px; font-size:11px; background:#111; border:1px solid rgba(157,78,221,0.4); border-radius:4px; color:#fff; padding:8px; font-family:var(--font-mono); resize:vertical; line-height:1.4;">${existing}</textarea>
                <span style="opacity:0.5; font-size:9px; line-height:1.3;">These instructions govern how Aethel synthesizes global observations into briefings. Stored locally.</span>
            </div>
        `;

        showAethelModal({
            title: "CUSTOMIZE OSINT BRIEFING PROMPT",
            contentHtml: editPromptHtml,
            onSave: async (closeModal) => {
                const custom = document.getElementById("modal-prompt-text").value.trim();
                window.__osintCustomPrompt = custom;
                try {
                    localStorage.setItem('aethel_osint_briefing_prompt', custom);
                } catch (_) {}
                showAethelToast("Custom OSINT Prompt persistent gespeichert!", "success");
                closeModal();
            }
        });
    };

    const bridgeBtn = document.createElement("button");
    bridgeBtn.className = "cyber-button";
    bridgeBtn.style.cssText = "font-size:8px; padding:3px 7px; margin-left:4px;";
    bridgeBtn.textContent = "HIGH-VALUE → INTELLIGENCE";
    bridgeBtn.onclick = async () => {
        const high = activeFeedEvents.filter(ev => (ev.confidence || 0) >= 0.7 || (ev.confidence || 0) >= 70);
        if (high.length === 0) {
            showAethelToast("Keine high-confidence Events im aktuellen Filter.", "error");
            return;
        }
        let count = 0;
        for (const ev of high.slice(0, 5)) {
            try {
                await fetch(`${state.API_BASE}/v1/intelligence/events`, {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({
                        title: ev.title,
                        summary: ev.summary,
                        source: ev.source,
                        source_url: ev.url || ev.source_url || '',
                        latitude: ev.lat || 0,
                        longitude: ev.lon || 0,
                        confidence: Math.round((ev.confidence || 0.75) * 100),
                        severity: 'medium'
                    })
                });
                count++;
            } catch (_) {}
        }
        showAethelToast(`${count} hochwertige Beobachtungen als Proposed Events übernommen.`, "success");
        const intelBtn = document.getElementById('nav-btn-case');
        if (intelBtn) intelBtn.click();
    };
    controls.appendChild(bridgeBtn);

    const listDiv = document.getElementById('collector-list');
    if (!listDiv) return;

    async function loadAndRenderCollectors() {
        listDiv.replaceChildren();
        appendTextElement(listDiv, 'div', 'Quellen werden geladen…', 'gw-source-empty');
        try {
            const res = await fetch(`${state.API_BASE}/v1/osint/collectors`);
            if (!res.ok) throw new Error('fetch failed');
            const configs = await res.json();
            listDiv.replaceChildren();
            if (!configs || !configs.length) {
                appendTextElement(listDiv, 'div', 'Keine RSS-Quellen konfiguriert.', 'gw-source-empty');
                return;
            }
            configs.forEach(cfg => {
                const name = cfg.name || cfg.Name || 'unnamed';
                const dom = cfg.domain || cfg.Domain || '';
                const sourceType = cfg.type || cfg.Type || 'rss';
                const row = document.createElement('article');
                row.className = 'gw-source-row';
                const label = document.createElement('span');
                label.className = 'gw-source-name';
                label.textContent = String(name);
                const domainLabel = document.createElement('small');
                domainLabel.textContent = `${String(sourceType).toUpperCase()} · ${String(dom || 'general').toUpperCase()}`;
                label.appendChild(domainLabel);
                row.appendChild(label);
                const del = document.createElement('button');
                del.type = 'button';
                del.textContent = 'ENTFERNEN';
                del.className = 'gw-source-delete';
                del.onclick = () => {
                    const confirmNode = document.createElement('div');
                    const p1 = document.createElement('span');
                    p1.textContent = 'Möchtest du die RSS-Quelle ';
                    const strong = document.createElement('strong');
                    strong.textContent = String(name);
                    const p2 = document.createElement('span');
                    p2.textContent = ' wirklich entfernen?';
                    confirmNode.append(p1, strong, p2);
                    showAethelModal({
                        title: 'CONFIRM SOURCE DELETION',
                        contentNode: confirmNode,
                        onSave: async (closeModal) => {
                            const delRes = await fetch(`${state.API_BASE}/v1/osint/collectors?name=${encodeURIComponent(name)}`, { method: 'DELETE' });
                            if (!delRes.ok) throw new Error(await delRes.text());
                            showAethelToast('Quelle "' + String(name) + '" wurde entfernt.', 'success');
                            closeModal();
                            loadAndRenderCollectors();
                            const active = document.querySelector('.gw-domain-filter.active');
                            refreshOSINTFeed(active ? active.getAttribute('data-domain') : 'all', false, showSelectionDetails, openGwReportReader);
                        }
                    });
                };
                row.appendChild(del);
                listDiv.appendChild(row);
            });
        } catch (e) {
            listDiv.replaceChildren();
            appendTextElement(listDiv, 'div', 'Quellen konnten nicht geladen werden.', 'gw-source-empty gw-source-error');
        }
    }
    loadAndRenderCollectors();
    window.loadAndRenderCollectors = loadAndRenderCollectors;
    const sourceRefresh = document.getElementById('gw-source-refresh');
    if (sourceRefresh && !sourceRefresh._gwBound) {
        sourceRefresh._gwBound = true;
        sourceRefresh.addEventListener('click', () => void loadAndRenderCollectors());
    }
}

export function hazardSourceEnabled(source) {
    return localStorage.getItem(`aethel_gw_source_${source}_enabled`) !== 'false';
}

export function wireHazardSourcePreferences() {
    const bindings = [
        ['gw-source-usgs-enabled', 'usgs'],
        ['gw-source-volcano-enabled', 'volcano'],
    ];
    bindings.forEach(([id, source]) => {
        const checkbox = document.getElementById(id);
        if (!checkbox) return;
        checkbox.checked = hazardSourceEnabled(source);
        if (checkbox._gwBound) return;
        checkbox._gwBound = true;
        checkbox.addEventListener('change', () => {
            localStorage.setItem(`aethel_gw_source_${source}_enabled`, String(checkbox.checked));
            showAethelToast(`${source.toUpperCase()}-Quelle ${checkbox.checked ? 'aktiviert' : 'deaktiviert'}.`, 'success');
        });
    });
}

export function makeGwPanelDraggable(panelEl, handleEl, storageKey) {
    if (!panelEl || !handleEl || handleEl.dataset.gwDragBound === 'true') return;
    handleEl.dataset.gwDragBound = 'true';
    let dragging = false;
    let ox = 0, oy = 0, startL = 0, startT = 0;

    const applyPos = (left, top) => {
        const host = panelEl.offsetParent || document.getElementById('view-global-watch') || document.body;
        const hw = host.clientWidth || window.innerWidth;
        const hh = host.clientHeight || window.innerHeight;
        const clamped = clampGwPanelPosition(left, top, panelEl.offsetWidth, panelEl.offsetHeight, hw, hh);
        panelEl.style.left = clamped.left + 'px';
        panelEl.style.top = clamped.top + 'px';
        panelEl.style.right = 'auto';
        panelEl.style.bottom = 'auto';
        if (storageKey) {
            try {
                sessionStorage.setItem(storageKey, JSON.stringify({ left: clamped.left, top: clamped.top }));
            } catch (_) {}
        }
    };

    if (storageKey) {
        try {
            const raw = sessionStorage.getItem(storageKey);
            if (raw) {
                const p = JSON.parse(raw);
                if (typeof p.left === 'number' && typeof p.top === 'number') {
                    applyPos(p.left, p.top);
                }
            }
        } catch (_) {}
    }

    handleEl.addEventListener('pointerdown', (e) => {
        if (e.target.closest('button, input, select, a, textarea')) return;
        dragging = true;
        ox = e.clientX;
        oy = e.clientY;
        const rect = panelEl.getBoundingClientRect();
        const host = panelEl.offsetParent || document.getElementById('view-global-watch') || document.body;
        const hostRect = host.getBoundingClientRect();
        startL = rect.left - hostRect.left;
        startT = rect.top - hostRect.top;
        panelEl.style.zIndex = String(80 + Math.floor(Date.now() % 1000));
        try { handleEl.setPointerCapture(e.pointerId); } catch (_) {}
        e.preventDefault();
    });
    handleEl.addEventListener('pointermove', (e) => {
        if (!dragging) return;
        applyPos(startL + (e.clientX - ox), startT + (e.clientY - oy));
    });
    const end = () => { dragging = false; };
    handleEl.addEventListener('pointerup', end);
    handleEl.addEventListener('pointercancel', end);
}

export function wireGwFloatPanels() {
    const risk = document.getElementById('gw-risk-hud');
    const riskHead = document.getElementById('gw-risk-hud-head') || risk?.querySelector('.gw-risk-hud-head');
    const riskToggle = document.getElementById('gw-risk-hud-collapse');
    const riskBody = document.getElementById('gw-risk-hud-body');
    if (risk && riskHead) {
        makeGwPanelDraggable(risk, riskHead, 'aethel_gw_risk_hud_pos');
        risk.classList.add('gw-float-panel');
    }
    if (riskToggle && riskBody && !riskToggle._gwBound) {
        riskToggle._gwBound = true;
        const apply = (collapsed) => {
            riskBody.classList.toggle('collapsed', collapsed);
            risk?.classList.toggle('gw-risk-hud-collapsed', collapsed);
            riskToggle.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
            riskToggle.textContent = collapsed ? '▸' : '▾';
            try { sessionStorage.setItem('aethel_gw_risk_collapsed', collapsed ? '1' : '0'); } catch (_) {}
        };
        apply(sessionStorage.getItem('aethel_gw_risk_collapsed') === '1');
        riskToggle.addEventListener('click', (e) => {
            e.stopPropagation();
            apply(!riskBody.classList.contains('collapsed'));
        });
    }

    const sel = document.getElementById('gw-selection-panel');
    const selHead = sel?.querySelector('.gw-selection-head');
    if (sel && selHead) makeGwPanelDraggable(sel, selHead, 'aethel_gw_selection_pos');
}

export function wireGwSettingsModal() {
    const modal = document.getElementById('gw-settings-modal');
    const openBtn = document.getElementById('gw-settings-open');
    if (!modal || !openBtn || openBtn._gwBound) return;
    openBtn._gwBound = true;
    const open = () => {
        modal.classList.remove('hidden');
        void loadAndRenderConnectors();
        void loadGwSettingsSources();
    };
    const close = () => modal.classList.add('hidden');
    openBtn.addEventListener('click', open);
    modal.querySelectorAll('[data-gw-settings-close]').forEach(el => el.addEventListener('click', close));
}

export async function loadGwSettingsSources() {
    const box = document.getElementById('gw-settings-sources-list');
    if (!box) return;
    try {
        const res = await fetch(`${state.API_BASE}/v1/intelligence/sources`);
        if (!res.ok) throw new Error('sources unavailable');
        const data = await res.json();
        const sources = data.sources || data || [];
        if (Array.isArray(sources) && sources.length) {
            box.textContent = sources.map((s, i) => {
                if (typeof s === 'string') return `${i + 1}. ${s}`;
                return `${i + 1}. ${s.name || s.id || s.source || JSON.stringify(s).slice(0, 80)}`;
            }).join('\n');
        } else {
            box.textContent = 'Keine Quellen-Metadaten — Feed-Quellen erscheinen in den Meldungskarten.';
        }
    } catch (_) {
        const names = [...new Set((activeFeedEvents || []).map(e => e.source).filter(Boolean))].slice(0, 40);
        box.textContent = names.length
            ? names.map((n, i) => `${i + 1}. ${n}`).join('\n')
            : 'Quellenliste leer (noch kein Feed geladen).';
    }
}

export function wireGwTimeWindow() {
    const el = document.getElementById('gw-time-window');
    if (!el || el._gwBound) return;
    el._gwBound = true;
    window.__gwTimeWindowHours = getGwTimeWindowHours();
    el.addEventListener('change', () => {
        setGwTimeWindowHours(el.value);
        const active = document.querySelector('.gw-domain-filter.active');
        const domain = active ? active.getAttribute('data-domain') : 'all';
        void refreshOSINTFeed(domain || 'all', false, showSelectionDetails, openGwReportReader);
    });
}

export function wireGwReportReaderEvents() {
    const openBtn = document.getElementById('gw-open-report-reader');
    const closeBtn = document.getElementById('gw-report-close');
    const exportBtn = document.getElementById('gw-report-export');
    const panel = document.getElementById('gw-report-reader');
    if (openBtn && !openBtn._gwBound) {
        openBtn._gwBound = true;
        openBtn.addEventListener('click', () => {
            const hours = getGwTimeWindowHours();
            const filtered = activeFeedEvents;
            const lines = [
                `# Global Watch Report`,
                ``,
                `Zeitfenster: ${hours > 0 ? hours + 'h' : 'ALLE'} · Meldungen: ${filtered.length}`,
                ``,
                ...filtered.slice(0, 40).map((ev, i) => {
                    const dt = formatArticleDateTime(ev.timestamp || ev.observed_at);
                    return `## ${i + 1}. ${ev.title || 'Ohne Titel'}\n- Quelle: ${ev.source || '—'}\n- Domain: ${ev.domain || '—'}\n- Datum: ${dt.date}\n- Uhrzeit: ${dt.time}\n\n${ev.summary || ''}\n`;
                })
            ];
            openGwReportReader('Lagebericht Meldungsfeed', lines.join('\n'));
        });
    }
    const closeAll = () => panel?.classList.add('hidden');
    if (closeBtn && !closeBtn._gwBound) {
        closeBtn._gwBound = true;
        closeBtn.addEventListener('click', closeAll);
    }
    panel?.querySelectorAll('[data-gw-reader-close]').forEach((el) => {
        if (el._gwBound) return;
        el._gwBound = true;
        el.addEventListener('click', closeAll);
    });
    if (exportBtn && !exportBtn._gwBound) {
        exportBtn._gwBound = true;
        exportBtn.addEventListener('click', () => {
            const md = panel?.dataset.exportMd || (document.getElementById('gw-report-reader-body')?.innerText || '');
            downloadTextFile('aethel-global-watch-report.md', md, 'text/markdown');
        });
    }
    const selReader = document.getElementById('gw-selection-open-reader');
    if (selReader && !selReader._gwBound) {
        selReader._gwBound = true;
        selReader.addEventListener('click', () => {
            const idx = localGlobeSelectedIndex;
            const ev = (idx >= 0 && activeFeedEvents[idx]) ? activeFeedEvents[idx] : null;
            if (ev) openGwReportReader(ev);
            else showAethelToast('Keine Meldung ausgewählt', 'info');
        });
    }
}

export function wireGwAiCommand() {
    const input = document.getElementById('gw-ai-command');
    const send = document.getElementById('gw-ai-command-send');
    if (!input || !send || send._gwBound) return;
    send._gwBound = true;
    const submit = async () => {
        const text = input.value.trim();
        if (!text) return;
        const lower = text.toLowerCase();
        if (/\b(deutschland|germany|berlin)\b/.test(lower)) {
            focusRegionByKey('germany');
        } else if (/\beuropa|europe\b/.test(lower)) {
            focusRegionByKey('europe');
        } else if (/\b(global|welt)\b/.test(lower)) {
            focusRegionByKey('global');
        }
        if (/\b24\s*h|letzte 24|last 24\b/.test(lower)) setGwTimeWindowHours(24);
        else if (/\b6\s*h|letzte 6\b/.test(lower)) setGwTimeWindowHours(6);
        else if (/\b72\s*h|3\s*tage\b/.test(lower)) setGwTimeWindowHours(72);
        else if (/\b7\s*d|7\s*tage|woche\b/.test(lower)) setGwTimeWindowHours(168);

        const shared = document.getElementById('user-input');
        const wrapped = `[GLOBAL WATCH OPERATOR]\nZeitfenster: ${getGwTimeWindowHours()}h\nAnweisung: ${text}\n` +
            `Nutze global_watch_nexus_context sowie für sichtbare Steuerung global_watch_focus_region und global_watch_time_window. Behaupte keine Änderung ohne bestätigtes Tool-Ergebnis.`;
        if (shared) {
            shared.value = wrapped;
            try {
                const { sendMessage } = await import('../chat.js');
                await sendMessage();
            } catch (err) {
                console.warn('GW AI command send failed', err);
                showAethelToast('Anweisung konnte nicht gesendet werden', 'error');
            }
        }
        input.value = '';
        const active = document.querySelector('.gw-domain-filter.active');
        void refreshOSINTFeed(active ? active.getAttribute('data-domain') : 'all', false, showSelectionDetails, openGwReportReader);
    };
    send.addEventListener('click', () => { void submit(); });
    input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') { e.preventDefault(); void submit(); }
    });
}

export function wireConnectorsPanel() {
    const refresh = document.getElementById('gw-connectors-refresh');
    if (refresh && !refresh._gwBound) {
        refresh._gwBound = true;
        refresh.addEventListener('click', () => { void loadAndRenderConnectors(); });
    }
    const fetchAll = document.getElementById('gw-connector-fetch-all');
    if (fetchAll && !fetchAll._gwBound) {
        fetchAll._gwBound = true;
        fetchAll.addEventListener('click', async () => {
            fetchAll.disabled = true;
            try {
                await fetchConnector('builtin-rss');
                await fetchConnector('builtin-usgs');
                await fetchConnector('builtin-eonet');
                await fetchConnector('builtin-volcano');
                await fetchConnector('builtin-cyber');
                await fetchConnector('builtin-humanitarian');
                await fetchConnector('builtin-economic');
                showAethelToast('Connector fetch complete → Shared model', 'success');
                const active = document.querySelector('.gw-domain-filter.active');
                void refreshOSINTFeed(active ? active.getAttribute('data-domain') : 'all', false, showSelectionDetails, openGwReportReader);
                void loadAndRenderConnectors();
            } catch (e) {
                showAethelToast('Connector fetch: ' + (e.message || 'error'), 'error');
            } finally {
                fetchAll.disabled = false;
            }
        });
    }
}

export async function fetchConnector(name) {
    const res = await fetch(`${state.API_BASE}/v1/intelligence/connectors/fetch`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name })
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

export async function loadAndRenderConnectors() {
    const list = document.getElementById('gw-settings-connectors-list') || document.getElementById('gw-connectors-list');
    if (!list) return;
    try {
        const res = await fetch(`${state.API_BASE}/v1/intelligence/connectors`);
        if (!res.ok) throw new Error('connectors unavailable');
        const data = await res.json();
        const connectors = data.connectors || [];
        list.replaceChildren();
        if (!connectors.length) {
            appendTextElement(list, 'div', 'No connectors registered', 'gw-risk-empty');
            return;
        }
        connectors.forEach(c => {
            const card = document.createElement('div');
            card.className = 'gw-connector-card';
            const top = document.createElement('div');
            top.className = 'gw-connector-card-top';
            const name = document.createElement('span');
            name.textContent = String(c.name || '?');
            const health = document.createElement('span');
            health.textContent = String(c.health || '—').slice(0, 18);
            top.append(name, health);
            const meta = document.createElement('div');
            meta.className = 'gw-connector-meta';
            meta.textContent = `v${c.version || '?'} · trust ${c.trust_tier} · ${c.polling || ''}`;
            const btn = document.createElement('button');
            btn.type = 'button';
            btn.className = 'gw-tool-btn';
            btn.textContent = 'FETCH → SHARED';
            btn.addEventListener('click', async () => {
                btn.disabled = true;
                try {
                    const out = await fetchConnector(c.name);
                    showAethelToast(`${c.name}: fetched ${out.fetched || 0} · ingested ${out.ingested || 0}`, 'success');
                    const active = document.querySelector('.gw-domain-filter.active');
                    void refreshOSINTFeed(active ? active.getAttribute('data-domain') : 'all', false, showSelectionDetails, openGwReportReader);
                } catch (e) {
                    showAethelToast(String(e.message || e).slice(0, 120), 'error');
                } finally {
                    btn.disabled = false;
                    void loadAndRenderConnectors();
                }
            });
            card.append(top, meta, btn);
            list.appendChild(card);
        });
    } catch (e) {
        list.replaceChildren();
        appendTextElement(list, 'div', 'Connectors unavailable', 'gw-risk-empty');
    }
}

// Aethel / UI bridge: navigate views and apply GW commands from skills / chat tools
window.AETHEL_NAVIGATE_VIEW = function(viewKey) {
    const map = {
        core: 'nav-btn-core',
        chat: 'nav-btn-chat',
        global_watch: 'nav-btn-global-watch',
        globe: 'nav-btn-global-watch',
        sphere: 'nav-btn-sphere',
        personal: 'nav-btn-personal',
        case: 'nav-btn-case',
        tasks: 'nav-btn-tasks',
        settings: 'nav-btn-settings',
		memory: 'nav-btn-memory',
		agents: 'nav-btn-agent',
		agent_tracker: 'nav-btn-agent',
    };
    const id = map[String(viewKey || '').toLowerCase()];
    const btn = id ? document.getElementById(id) : null;
    if (btn) { btn.click(); return true; }
    return false;
};

window.AETHEL_GW_COMMAND = function(cmd) {
    if (!cmd || typeof cmd !== 'object') return false;
    if (cmd.action === 'focus_region' && cmd.region) {
        focusRegionByKey(cmd.region);
        return true;
    }
    if (cmd.action === 'focus' && (cmd.lat != null || cmd.latitude != null)) {
        focusGlobeOnLonLat(cmd.lon ?? cmd.longitude, cmd.lat ?? cmd.latitude, { scale: cmd.zoom, snap: true });
        return true;
    }
    if (cmd.action === 'time_window') {
        setGwTimeWindowHours(cmd.hours);
        const active = document.querySelector('.gw-domain-filter.active');
        void refreshOSINTFeed(active ? active.getAttribute('data-domain') : 'all', false);
        return true;
    }
    if (cmd.action === 'open_report') {
        openGwReportReader(cmd.title || 'Report', cmd.body || cmd.markdown || '');
        return true;
    }
    if (cmd.action === 'navigate') {
        return !!window.AETHEL_NAVIGATE_VIEW(cmd.view || cmd.target);
    }
    return false;
};

// HTML onclick / external bridge (monolith parity)
window.promoteCurrentSelection = promoteCurrentSelection;
window.saveSelectionToNexus = saveSelectionToNexus;
window.aiBriefingForSelection = aiBriefingForSelection;
window.observeAtSelection = observeAtSelection;

export async function initGlobalWatch() {
    const container = document.getElementById("osint-globe");
    if (!container) return;
    setGlobeContainer(container);

    initPureLocalGlobe(showSelectionDetails, showCameraDetails);
    subscribeGlobalWatchCommands();
    wireSelectedEventAIActions();

    const filters = document.querySelectorAll(".gw-domain-filter");
    filters.forEach(btn => {
        btn.addEventListener("click", () => {
            filters.forEach(f => f.classList.remove("active"));
            btn.classList.add("active");
            const domain = btn.getAttribute("data-domain");
            refreshOSINTFeed(domain, false, showSelectionDetails, openGwReportReader);
        });
    });

    const searchEl = document.getElementById('gw-feed-search');
    if (searchEl && !searchEl._gwBound) {
        searchEl._gwBound = true;
        searchEl.addEventListener('input', () => {
            const active = document.querySelector('.gw-domain-filter.active');
            void refreshOSINTFeed(active ? active.getAttribute('data-domain') : 'all', false, showSelectionDetails, openGwReportReader);
        });
    }

    const layerSel = document.getElementById('gw-feed-layer-filter');
    if (layerSel && !layerSel._gwBound) {
        layerSel._gwBound = true;
        layerSel.addEventListener('change', () => {
            const active = document.querySelector('.gw-domain-filter.active');
            void refreshOSINTFeed(active ? active.getAttribute('data-domain') : 'all', false, showSelectionDetails, openGwReportReader);
        });
    }

    const layerToggles = document.querySelectorAll(".gw-layer-row[data-layer], .gw-layer-toggle[data-layer]");
    layerToggles.forEach(btn => {
        const layer = btn.getAttribute("data-layer");
        if (!layer || !(layer in visibleLayers)) return;
        btn.classList.toggle("active", !!visibleLayers[layer]);
        btn.addEventListener("click", () => {
            visibleLayers[layer] = !visibleLayers[layer];
            btn.classList.toggle("active", !!visibleLayers[layer]);
            updateLayerCounts();
            drawPureLocalGlobe();
        });
    });

    updateLayerCounts();
    wireRegionChips();
    wireGwCollapsibles();
    wireGwSettingsModal();
    wireGlobalWatchRuntimeSettings();
    wireGwTimeWindow();
    wireGwReportReaderEvents();
    wireGwFloatPanels();
    wireGwAiCommand();
    wireGlobalSearch();
    wireTimeline();
    wireConnectorsPanel();
    wireHazardSourcePreferences();
    void loadAndRenderConnectors();

    const addCamBtn = document.getElementById("btn-add-cam-static") || document.createElement("button");
    if (!addCamBtn.id) {
      addCamBtn.textContent = "+CAM";
      addCamBtn.style.cssText = "font-size:7px; padding:1px 3px; margin-left:4px;";
    }
    addCamBtn.onclick = () => {
        const modalHtml = `
            <div style="display:flex; flex-direction:column; gap:12px;">
                <div>
                    <label style="display:block; font-size:9px; color:#0ff; margin-bottom:4px; font-weight:700;">CAMERA NAME</label>
                    <input id="modal-cam-name" placeholder="Camera Name (e.g. London Traffic)" style="width:100%; font-size:11px; background:#111; border:1px solid rgba(0,240,255,0.3); border-radius:4px; color:#fff; padding:8px; font-family:var(--font-mono);">
                </div>
                <div style="display:grid; grid-template-columns:1fr 1fr; gap:10px;">
                    <div>
                        <label style="display:block; font-size:9px; color:#0ff; margin-bottom:4px; font-weight:700;">LATITUDE</label>
                        <input id="modal-cam-lat" placeholder="e.g. 51.5074" style="width:100%; font-size:11px; background:#111; border:1px solid rgba(0,240,255,0.3); border-radius:4px; color:#fff; padding:8px; font-family:var(--font-mono);">
                    </div>
                    <div>
                        <label style="display:block; font-size:9px; color:#0ff; margin-bottom:4px; font-weight:700;">LONGITUDE</label>
                        <input id="modal-cam-lon" placeholder="e.g. -0.1278" style="width:100%; font-size:11px; background:#111; border:1px solid rgba(0,240,255,0.3); border-radius:4px; color:#fff; padding:8px; font-family:var(--font-mono);">
                    </div>
                </div>
                <div>
                    <label style="display:block; font-size:9px; color:#0ff; margin-bottom:4px; font-weight:700;">STREAM URL (HTTPS, Optional)</label>
                    <input id="modal-cam-url" placeholder="https://domain.com/cam.jpg" style="width:100%; font-size:11px; background:#111; border:1px solid rgba(0,240,255,0.3); border-radius:4px; color:#fff; padding:8px; font-family:var(--font-mono);">
                </div>
            </div>
        `;
        showAethelModal({
            title: "ADD LOCAL CAMERA STATIONS",
            contentHtml: modalHtml,
            onSave: async (closeModal) => {
                const name = document.getElementById("modal-cam-name").value.trim();
                const latStr = document.getElementById("modal-cam-lat").value.trim();
                const lonStr = document.getElementById("modal-cam-lon").value.trim();
                const streamHint = document.getElementById("modal-cam-url").value.trim();
                
                const lat = parseFloat(latStr);
                const lon = parseFloat(lonStr);
                
                if (!name) throw new Error("Camera Name is required.");
                if (isNaN(lat) || isNaN(lon)) throw new Error("Valid coordinates are required.");
                if (lat < -90 || lat > 90 || lon < -180 || lon > 180) throw new Error("Coordinates out of range (-90 to 90 lat, -180 to 180 lon).");
                
                const camEntry = {lat, lon, name: name.slice(0, 120) + " (user)"};
                if (streamHint) {
                    const stream = safeExternalURL(streamHint);
                    if (!stream) throw new Error("Only valid secure HTTPS stream URLs are accepted.");
                    camEntry.stream = stream.toString();
                }
                
                cameraData.push(camEntry);
                closeModal();
                showAethelToast("Kamera erfolgreich hinzugefügt! Aktivieren Sie den Layer 'CAMERAS' zum Anzeigen.", "success");
                drawPureLocalGlobe();
            }
        });
    };
    const layerDiv = document.querySelector(".gw-layer-toggles");
    const staticBtn = document.getElementById("btn-add-cam-static");
    if (layerDiv && !staticBtn) layerDiv.appendChild(addCamBtn);

    const refreshBtn = document.getElementById("gw-refresh-btn");
    if (refreshBtn) {
        refreshBtn.addEventListener("click", () => {
            const activeFilter = document.querySelector(".gw-domain-filter.active");
            const domain = activeFilter ? activeFilter.getAttribute("data-domain") : "all";
            refreshOSINTFeed(domain, true, showSelectionDetails, openGwReportReader);
        });
    }

    const briefingBtn = document.getElementById("gw-briefing-btn");
    if (briefingBtn) {
        briefingBtn.addEventListener("click", triggerAIBriefing);
    }
    wireBriefingWorkspace();

    const closeBriefingBtn = document.getElementById("gw-briefing-close");
    if (closeBriefingBtn) {
        closeBriefingBtn.addEventListener("click", () => {
            document.getElementById("gw-briefing-overlay").classList.remove("visible");
        });
    }

    refreshOSINTFeed("all", false, showSelectionDetails, openGwReportReader);
    scheduleGlobalWatchAutoRefresh();
    scheduleGlobalWatchHazardAnimation();
    wirePersonalImpactControls();
    void loadAndRenderPersonalImpact();
    injectOSINTControls();

    try {
        const saved = localStorage.getItem('aethel_osint_briefing_prompt');
        if (saved) window.__osintCustomPrompt = saved;
    } catch (_) {}
}
