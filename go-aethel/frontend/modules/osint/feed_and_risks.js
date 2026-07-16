import { state } from '../state.js';
import {
  activeFeedEvents, setActiveFeedEvents,
  globalWatchPreferences,
  cachedRiskMarkers, setCachedRiskMarkers,
  REGION_GEO,
  globalWatchCommandStream, setGlobalWatchCommandStream,
  localGlobeRotY, localGlobeRotX, localGlobeScale, localGlobeCanvas,
  localGlobeSelectedIndex, setLocalGlobeSelectedIndex
} from './state.js';
import {
  isEarthquakeEvent,
  parseMagnitudeFromEvent,
  isVolcanoEvent,
  isEruptingVolcano,
  formatArticleDateTime,
  freshness,
  epistemicLayer,
} from './hazards.js';
import {
  filterEventsByTimeWindow,
  filterEventsByRegion,
  eventMatchesRegion,
  safeExternalURL,
  appendTextElement,
  projectLatLon,
  eventTimestampMs as eventTimestampMsPure,
  REGION_FOCUS_TABLE,
} from './projection.js';
import { requestGlobeRender, focusGlobeOnLonLat, clearPins } from './globe_render.js';
import { showSelectionDetails } from './selection_and_chat.js';
import { openGwReportReader } from './briefing_and_reader.js';

export function getGwTimeWindowHours() {
    const el = document.getElementById('gw-time-window');
    const v = el ? Number(el.value) : (window.__gwTimeWindowHours != null ? Number(window.__gwTimeWindowHours) : 24);
    return Number.isFinite(v) ? v : 24;
}

export function setGwTimeWindowHours(hours) {
    const h = Number(hours);
    window.__gwTimeWindowHours = Number.isFinite(h) ? h : 24;
    const el = document.getElementById('gw-time-window');
    if (el) el.value = String(window.__gwTimeWindowHours);
}

export function eventTimestampMs(ev) {
    return eventTimestampMsPure(ev);
}

/** Focus globe + refilter feed for a named region (monolith parity). */
export function focusRegionByKey(key) {
  const regionKey = String(key || '').toLowerCase();
  const f = REGION_FOCUS_TABLE[regionKey] || REGION_FOCUS_TABLE.global;
  window.__gwRegionFilter = REGION_FOCUS_TABLE[regionKey] ? regionKey : 'global';
  focusGlobeOnLonLat(f.lon, f.lat, { scale: f.scale, snap: true });
  const active = document.querySelector('.gw-domain-filter.active');
  void refreshOSINTFeed(active ? active.getAttribute('data-domain') : 'all', false, showSelectionDetails, openGwReportReader);
  return f;
}

export function activeGlobalWatchDomain() {
    const active = document.querySelector('.gw-domain-filter.active');
    return active ? active.getAttribute('data-domain') || 'all' : 'all';
}

export function stableGlobeEventID(event) {
    return String(event?.id || event?.ID || `${event?.source || ''}|${event?.title || ''}|${eventTimestampMs(event)}`);
}

export async function refreshOSINTFeed(domain = "all", triggerBackendCollect = false, showSelectionDetailsFn = showSelectionDetails, openGwReportReaderFn = openGwReportReader) {
    const feedList = document.getElementById("gw-feed-list");
    const feedCount = document.getElementById("gw-feed-count");
    const refreshInfo = document.getElementById("gw-refresh-info");

    if (!feedList) return;

    try {
		const strictHours = getGwTimeWindowHours();
		let url = `${state.API_BASE}/v1/osint/feeds?domain=${encodeURIComponent(domain)}&hours=${encodeURIComponent(strictHours)}`;
        if (triggerBackendCollect) {
            refreshInfo.textContent = "COLLECTING FRESH INTELLIGENCE...";
        }

        const res = await fetch(url);
        const data = await res.json();

        const loaded = (data.events || []).map((ev) => {
            const e = { ...ev };
            if (isEarthquakeEvent(e)) {
                e.hazard_type = 'earthquake';
                const mag = parseMagnitudeFromEvent(e);
                if (mag != null) e.magnitude = mag;
            } else if (isVolcanoEvent(e)) {
                e.hazard_type = 'volcano';
                e.erupting = isEruptingVolcano(e);
            }
            return e;
        });
        setActiveFeedEvents(loaded);
        refreshInfo.textContent = `LAST SYNC: ${data.last_refresh || "JETZT"}`;

        let idata = null;
        try {
            const ires = await fetch(`${state.API_BASE}/v1/intelligence/events`);
            if (ires.ok) {
                idata = await ires.json();
                const iEvents = (idata.events || []).filter(e => {
                    if (String(e.source || '').toUpperCase() !== 'GLOBAL_WATCH') return false;
                    if (e.latitude == null || e.longitude == null) return false;
                    const sum = (e.summary || '') + '';
                    if (sum.includes('Layer ') || (e.title || '').includes('Layer toggle')) return false;
                    if (Math.abs(e.latitude) < 0.0001 && Math.abs(e.longitude) < 0.0001) return false;
                    return true;
                }).map(e => {
                    const mapped = {
                        id: e.id,
                        lat: e.latitude,
                        lon: e.longitude,
                        has_geo: true,
                        domain: e.domain || 'intel',
                        source: e.source || 'shared-intel',
                        title: e.title,
                        summary: e.summary,
                        timestamp: e.observed_at,
                        confidence: e.confidence,
                        severity: e.severity,
                        status: 'unverified',
                        provenance: 'unified-shared-model',
                        url: e.source_url || e.url || '',
                    };
                    if (isEarthquakeEvent(mapped)) {
                        mapped.hazard_type = 'earthquake';
                        const mag = parseMagnitudeFromEvent(mapped);
                        if (mag != null) mapped.magnitude = mag;
                    } else if (isVolcanoEvent(mapped)) {
                        mapped.hazard_type = 'volcano';
                        mapped.erupting = isEruptingVolcano(mapped);
                    }
                    return mapped;
                });
                const seen = new Set(activeFeedEvents.map(x => (x.id || x.title || '') + ''));
                for (const ie of iEvents) {
                    const key = (ie.id || ie.title || '') + '';
                    if (!seen.has(key)) {
                        activeFeedEvents.push(ie);
                        seen.add(key);
                    }
                }
            }
        } catch (_) {}

        if (activeFeedEvents.length > globalWatchPreferences.feedLimit) {
            setActiveFeedEvents(activeFeedEvents.slice(0, globalWatchPreferences.feedLimit));
        }
        const timeFilteredEvents = filterEventsByTimeWindow(activeFeedEvents, getGwTimeWindowHours());
        const regionFilteredEvents = filterEventsByRegion(timeFilteredEvents, window.__gwRegionFilter || 'global');
        if (feedCount) feedCount.textContent = `${regionFilteredEvents.length} / ${activeFeedEvents.length} sichtbar`;

        for (const ie of (idata && idata.events ? idata.events : [])) {
          if (ie.title && ie.title.includes("Focus request") && ie.latitude != null && ie.longitude != null) {
            focusGlobeOnLonLat(ie.longitude, ie.latitude);
            setLocalGlobeSelectedIndex(activeFeedEvents.length - 1);
            break;
          }
        }

        feedList.replaceChildren();
        clearPins();

        if (activeFeedEvents.length === 0) {
            const empty = appendTextElement(feedList, 'div', 'NO ACTIVE OBSERVATIONS FOUND', 'gw-risk-empty');
            empty.style.padding = '20px';
            requestGlobeRender();
            return;
        }

        const searchQ = (document.getElementById('gw-feed-search')?.value || '').trim().toLowerCase();
        const layerFilter = (document.getElementById('gw-feed-layer-filter')?.value || 'all').toLowerCase();
        const timeHours = getGwTimeWindowHours();
        window.__gwVisibleEvents = filterEventsByRegion(filterEventsByTimeWindow(activeFeedEvents, timeHours), window.__gwRegionFilter || 'global');

        let rendered = 0;
        activeFeedEvents.forEach((ev, idx) => {
            if (!eventMatchesRegion(ev, window.__gwRegionFilter || 'global')) return;
            if (timeHours > 0) {
                const ms = eventTimestampMs(ev);
                if (!ms || ms < Date.now() - timeHours * 3600 * 1000 || ms > Date.now() + 10 * 60 * 1000) return;
            }
            const layer = epistemicLayer(ev);
            if (layerFilter !== 'all' && layer !== layerFilter) return;
            if (searchQ) {
                const hay = `${ev.title || ''} ${ev.source || ''} ${ev.summary || ''}`.toLowerCase();
                if (!hay.includes(searchQ)) return;
            }

            const card = document.createElement('div');
            card.className = `gw-event-card layer-${layer}`;
            card.setAttribute('role', 'listitem');
            card.dataset.idx = String(idx);

            let dotClass = 'general';
            if (ev.domain === 'geo') dotClass = 'geo';
            if (ev.domain === 'cyber') dotClass = 'cyber';
            if (ev.domain === 'economic') dotClass = 'economic';
            if (ev.domain === 'humanitarian') dotClass = 'humanitarian';

            const layerLabel = layer === 'verified' ? 'VERIFIED' : layer === 'inference' ? 'INFERENCE' : 'RAW';
            const sourceURL = safeExternalURL(ev.url);

            const header = document.createElement('div');
            header.className = 'gw-event-meta-row';
            const dot = document.createElement('span');
            dot.className = `gw-domain-dot ${dotClass}`;
            appendTextElement(header, 'span', ev.source || 'Feed', 'gw-event-source');
            const dtCard = formatArticleDateTime(ev.timestamp || ev.observed_at);
            appendTextElement(header, 'span', `${dtCard.date} ${dtCard.time} ${freshness(ev.timestamp)}`, 'gw-event-time font-mono');
            appendTextElement(header, 'span', layerLabel, `gw-event-badge layer-${layer}`);
            if (isEarthquakeEvent(ev)) {
                const mag = parseMagnitudeFromEvent(ev);
                appendTextElement(header, 'span', mag != null ? `M${Number(mag).toFixed(1)}` : 'QUAKE', 'gw-event-badge layer-raw');
            } else if (isVolcanoEvent(ev)) {
                appendTextElement(header, 'span', 'VULKAN', 'gw-event-badge layer-inference');
            }
            header.prepend(dot);

            const title = appendTextElement(card, 'div', ev.title || 'Untitled observation', 'gw-event-title');
            const summary = document.createElement('div');
            summary.className = 'gw-event-summary';
            summary.style.display = 'none';
            const summaryText = document.createElement('div');
            summaryText.textContent = ev.summary || 'Keine Detailbeschreibung. Unverified unless layer=VERIFIED.';
            summary.appendChild(summaryText);
            if (ev.provenance) {
                const prov = document.createElement('div');
                prov.style.cssText = 'margin-top:4px; font-family:var(--font-mono); font-size:8px; color:rgba(0,240,255,0.55);';
                prov.textContent = 'provenance: ' + String(ev.provenance);
                summary.appendChild(prov);
            }
            if (sourceURL) {
                const link = document.createElement('a');
                link.href = sourceURL.toString();
                link.target = '_blank';
                link.rel = 'noopener noreferrer';
                link.className = 'cyan-text';
                link.textContent = 'Originalquelle aufrufen';
                link.style.cssText = 'text-decoration:underline; margin-top:4px; display:inline-block;';
                summary.appendChild(link);
            }
            const promote = document.createElement('button');
            promote.type = 'button';
            promote.className = 'promote-btn gw-promote-btn';
            promote.textContent = '→ CASE / PROMOTE';
            summary.appendChild(promote);
            card.replaceChildren(header, title, summary);
            rendered++;

            card.addEventListener("click", (e) => {
                if (e.target.classList.contains('promote-btn') || e.target.closest('a')) return;

                const allCards = feedList.querySelectorAll(".gw-event-card");
                allCards.forEach(c => c.classList.remove("highlighted"));
                card.classList.add("highlighted");

                setLocalGlobeSelectedIndex(idx);
                const evClicked = activeFeedEvents[idx];
                if (evClicked && (evClicked.lat != null || evClicked.lon != null)) {
                    focusGlobeOnLonLat(evClicked.lon, evClicked.lat);
                }
                const pinsNow = window.__globePins || [];
                const p = pinsNow.find(pp => pp.idx === idx) || (evClicked ? projectLatLon(evClicked.lat||0, evClicked.lon||0, localGlobeRotY, localGlobeRotX, localGlobeScale, (localGlobeCanvas&&localGlobeCanvas.width)||620, (localGlobeCanvas&&localGlobeCanvas.height)||420) : null);
                if (evClicked) {
                    if (showSelectionDetailsFn) showSelectionDetailsFn(evClicked, p);
                    if (openGwReportReaderFn) openGwReportReaderFn(evClicked);
                }
            });

            const promoteBtn = card.querySelector('.promote-btn');
            if (promoteBtn) {
                promoteBtn.addEventListener('click', async (e) => {
                    e.stopPropagation();
                    promoteBtn.disabled = true;
                    promoteBtn.textContent = 'PROMOTING...';

                    const pTitle = ev.title || 'OSINT Observation';
                    const pSummary = ev.summary || '';
                    const pSource = ev.source || 'OSINT Feed';

                    let pLat = ev.lat;
                    let pLon = ev.lon;
                    const hasRealGeo = (pLat != null && pLon != null && (Math.abs(pLat) > 0.001 || Math.abs(pLon) > 0.001));
                    if (!hasRealGeo) {
                        pLat = null;
                        pLon = null;
                    }

                    try {
                        const eventId = ev.id || ev.ID || '';
                        if (window.AETHEL_PROMOTE_TO_CASE) {
                            await window.AETHEL_PROMOTE_TO_CASE(pTitle, pSummary, pSource, pLat, pLon, eventId);
                        } else {
                            await fetch(`${state.API_BASE}/v1/intelligence/events`, {
                                method: 'POST',
                                headers: {'Content-Type':'application/json'},
                                body: JSON.stringify({
                                    title: pTitle,
                                    summary: pSummary,
                                    source: pSource,
                                    source_url: ev.url || '',
                                    latitude: pLat,
                                    longitude: pLon,
                                    confidence: Math.round((ev.confidence||0.75)*100),
                                    severity: 'medium'
                                })
                            });
                            if (window.showAethelToast) window.showAethelToast('Observation als Proposed Event in Intelligence aufgenommen. Öffne CASE WORKSPACE.', 'success');
                        }
                        const caseBtn = document.getElementById('nav-btn-case');
                        if (caseBtn) caseBtn.click();
                    } catch (err) {
                        console.error(err);
                        if (window.showAethelToast) window.showAethelToast('Promotion fehlgeschlagen: ' + err.message, 'error');
                    } finally {
                        promoteBtn.disabled = false;
                        promoteBtn.textContent = '→ CASE ERÖFFNEN / PROMOTE';
                    }
                });
            }

            feedList.appendChild(card);
        });

        if (feedCount) feedCount.textContent = `${rendered} / ${activeFeedEvents.length} sichtbar`;
        const metricObs = document.getElementById('gw-metric-obs');
        if (metricObs) metricObs.textContent = `OBS ${activeFeedEvents.length}`;

        if (rendered === 0) {
            appendTextElement(feedList, 'div', 'Keine Meldungen für Filter. RAW/INFERENCE/VERIFIED trennen.', 'gw-risk-empty');
        }

        void loadAndRenderRegionalRisks();
        requestGlobeRender();

    } catch (e) {
        console.error("OSINT feed update failed", e);
        feedList.replaceChildren();
        const error = appendTextElement(feedList, 'div', `Fehler beim Laden der OSINT-Daten: ${e instanceof Error ? e.message : 'unbekannter Fehler'}`);
        error.style.cssText = 'padding:15px; color:var(--vgt-red); font-size:10px;';
    }
}

const REGION_FOCUS_LON = {
    GERMANY: 10, FRANCE: 2, USA: -95, UKRAINE: 31, UK: -3, BERLIN: 13.4
};

export async function loadAndRenderRegionalRisks() {
    const listEl = document.getElementById("gw-risk-hud-list");
    if (!listEl) return;

    try {
        let res = await fetch(`${state.API_BASE}/v1/intelligence/risk`);
        if (!res.ok) {
            res = await fetch(`${state.API_BASE}/v1/intelligence/risks`);
        }
        if (!res.ok) throw new Error("Failed to fetch risk values");
        let payload = await res.json();
        let risks = Array.isArray(payload) ? payload : [];
        if (!risks.length && payload && payload.risks && typeof payload.risks === 'object' && !Array.isArray(payload.risks)) {
            risks = Object.entries(payload.risks).map(([id, rs]) => ({
                region_id: id,
                region_name: id,
                overall_risk: rs.overall_risk != null ? rs.overall_risk : rs.OverallRisk,
                primary_drivers: rs.primary_drivers || rs.PrimaryDrivers || [],
                trend: rs.trend || rs.Trend || 'stable'
            }));
        }

        listEl.replaceChildren();
        const metricRisk = document.getElementById('gw-metric-risk');
        if (metricRisk) metricRisk.textContent = `RISK ${risks.length}`;

        const markers = risks.map((r) => {
            const id = String(r.region_id || r.region_name || '').toUpperCase();
            const geo = REGION_GEO[id] || { lat: null, lon: null };
            return {
                region_id: id,
                overall_risk: Number(r.overall_risk) || 0,
                trend: r.trend || 'stable',
                lat: geo.lat,
                lon: geo.lon
            };
        }).filter((m) => m.lat != null);
        setCachedRiskMarkers(markers);

        if (!risks.length) {
            setCachedRiskMarkers([]);
            appendTextElement(listEl, 'div', 'NO REGIONS ACTIVE', 'gw-risk-empty');
            requestGlobeRender();
            return;
        }

        risks
            .slice()
            .sort((a, b) => (Number(b.overall_risk) || 0) - (Number(a.overall_risk) || 0))
            .forEach(r => {
                const overall = Number(r.overall_risk) || 0;
                const item = document.createElement('div');
                item.className = 'gw-risk-item';
                item.title = 'Click to focus map · double-click for explain score';

                item.addEventListener('click', () => {
                    const rid = String(r.region_id || r.region_name || '').toUpperCase();
                    const geo = REGION_GEO[rid] || { lat: 20, lon: REGION_FOCUS_LON[rid] != null ? REGION_FOCUS_LON[rid] : 0 };
                    focusGlobeOnLonLat(geo.lon, geo.lat);
                    if (window.showAethelToast) window.showAethelToast(`Fokus: ${r.region_name || rid}`, 'info');
                });
                item.addEventListener('dblclick', async () => {
                    const rid = r.region_id || r.region_name || 'GERMANY';
                    await openExplainDrawer(rid);
                });

                let color = 'var(--vgt-green)';
                if (overall > 60) color = 'var(--vgt-red)';
                else if (overall > 25) color = 'var(--vgt-orange)';

                const trendChar = r.trend === 'up' ? '▲' : r.trend === 'down' ? '▼' : '■';
                const top = document.createElement('div');
                top.className = 'gw-risk-item-top';
                const name = document.createElement('span');
                name.className = 'gw-risk-item-name';
                name.textContent = r.region_name || r.region_id || 'region';
                const score = document.createElement('span');
                score.style.color = color;
                score.textContent = `${overall.toFixed(1)}% ${trendChar}`;
                top.append(name, score);

                const bar = document.createElement('div');
                bar.className = 'gw-risk-bar';
                const fill = document.createElement('div');
                fill.className = 'gw-risk-bar-fill';
                fill.style.width = `${Math.max(0, Math.min(100, overall))}%`;
                fill.style.background = color;
                fill.style.boxShadow = `0 0 5px ${color}`;
                bar.appendChild(fill);

                item.append(top, bar);
                const drivers = r.primary_drivers || [];
                if (drivers.length) {
                    const d = document.createElement('div');
                    d.className = 'gw-risk-driver';
                    d.textContent = 'DRIVER: ' + String(drivers[0]);
                    item.appendChild(d);
                }
                listEl.appendChild(item);
            });
        requestGlobeRender();
        void loadAndRenderAlerts();
    } catch (err) {
        console.error('Failed to render risk list', err);
        listEl.replaceChildren();
        const errEl = appendTextElement(listEl, 'div', 'HUD LOAD ERROR', 'gw-risk-empty');
        errEl.style.color = 'var(--vgt-red)';
    }
}

export async function openExplainDrawer(regionId) {
    const drawer = document.getElementById('gw-explain-drawer');
    const body = document.getElementById('gw-explain-body');
    const closeBtn = document.getElementById('gw-explain-close');
    if (!drawer || !body) {
        if (window.showAethelToast) window.showAethelToast('Explain drawer unavailable', 'error');
        return;
    }
    body.textContent = 'Loading explain score for ' + regionId + '…';
    drawer.classList.remove('hidden');
    if (closeBtn && !closeBtn._gwBound) {
        closeBtn._gwBound = true;
        closeBtn.addEventListener('click', () => drawer.classList.add('hidden'));
    }
    try {
        const er = await fetch(`${state.API_BASE}/v1/intelligence/explain/${encodeURIComponent(regionId)}`);
        if (!er.ok) throw new Error('explain failed');
        const data = await er.json();
        body.textContent = data.explanation || JSON.stringify(data, null, 2);
    } catch (e) {
        body.textContent = 'Explain unavailable: ' + (e.message || 'error');
    }
}

export async function loadAndRenderPersonalImpact() {
    const body = document.getElementById('gw-personal-impact-body');
    if (!body) return;
    try {
        const res = await fetch(`${state.API_BASE}/v1/intelligence/personal/impact`);
        if (!res.ok) throw new Error('impact fetch failed');
        const data = await res.json();
        body.textContent = data.brief || data.disclaimer || 'No personal impact data.';
    } catch (e) {
        body.textContent = 'Personal Impact unavailable. Enable Personal Core + SYNC. ' + (e.message || '');
    }
}

export async function syncPersonalAndRefreshImpact() {
    const body = document.getElementById('gw-personal-impact-body');
    if (body) body.textContent = 'Syncing PersonalStore → SharedIntelStore…';
    try {
        const res = await fetch(`${state.API_BASE}/v1/intelligence/personal/sync`, { method: 'POST' });
        if (!res.ok) throw new Error(await res.text());
        if (window.showAethelToast) window.showAethelToast('Personal Context synced (opt-in)', 'success');
        await loadAndRenderPersonalImpact();
    } catch (e) {
        if (window.showAethelToast) window.showAethelToast('Personal sync failed: ' + (e.message || 'error'), 'error');
        if (body) body.textContent = 'Sync failed. Enable Personal Core profile first.';
    }
}

export async function loadAndRenderAlerts() {
    const list = document.getElementById('gw-alerts-list');
    const metric = document.getElementById('gw-metric-alert');
    if (!list) return;
    try {
        const res = await fetch(`${state.API_BASE}/v1/intelligence/alerts`);
        if (!res.ok) throw new Error('alerts fetch failed');
        const data = await res.json();
        const alerts = (data.alerts || []).filter(a => !a.acknowledged && !a.Acknowledged);
        if (metric) metric.textContent = 'ALERT ' + alerts.length;
        list.replaceChildren();
        if (!alerts.length) {
            appendTextElement(list, 'div', 'No active alerts', 'gw-risk-empty');
            return;
        }
        alerts.slice(0, 12).forEach((a) => {
            const row = document.createElement('div');
            row.className = 'gw-alert-row';
            const top = document.createElement('div');
            top.className = 'gw-alert-row-top';
            const left = document.createElement('span');
            left.textContent = String(a.region || a.Region || '?') + ' · ' + String(a.severity || a.Severity || 'med').toUpperCase();
            const ack = document.createElement('button');
            ack.type = 'button';
            ack.className = 'gw-alert-ack';
            ack.textContent = 'ACK';
            ack.addEventListener('click', async (ev) => {
                ev.stopPropagation();
                const id = a.id || a.ID;
                try {
                    await fetch(`${state.API_BASE}/v1/intelligence/alerts/${encodeURIComponent(id)}/ack`, { method: 'POST' });
                    void loadAndRenderAlerts();
                } catch (_) {
                    if (window.showAethelToast) window.showAethelToast('Ack failed', 'error');
                }
            });
            const evaluate = document.createElement('button');
            evaluate.type = 'button';
            evaluate.className = 'gw-alert-evaluate';
            evaluate.textContent = a.ai_assessment ? 'KI NEU BEWERTEN' : 'KI BEWERTEN';
            evaluate.addEventListener('click', async event => {
                event.stopPropagation();
                const id = a.id || a.ID;
                evaluate.disabled = true;
                evaluate.textContent = 'ANALYSE…';
                try {
                    const response = await fetch(`${state.API_BASE}/v1/intelligence/alerts/${encodeURIComponent(id)}/evaluate`, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ model_id: state.currentModel, language: localStorage.getItem('aethel_ui_language') || 'de' }),
                    });
                    if (!response.ok) throw new Error((await response.text()).slice(0, 300));
                    if (window.showAethelToast) window.showAethelToast('KI-Assessment gespeichert. Deterministische Baseline bleibt unverändert.', 'success');
                    await loadAndRenderAlerts();
                } catch (error) {
                    if (window.showAethelToast) window.showAethelToast(`KI-Bewertung fehlgeschlagen: ${error.message}`, 'error');
                    evaluate.disabled = false;
                    evaluate.textContent = 'KI BEWERTEN';
                }
            });
            const actionGroup = document.createElement('span');
            actionGroup.className = 'gw-alert-actions';
            actionGroup.append(evaluate, ack);
            top.append(left, actionGroup);
            const reason = document.createElement('div');
            reason.textContent = a.reason || a.Reason || '';
            reason.style.color = 'var(--vgt-text-dim)';
            row.append(top, reason);
            const assessment = a.ai_assessment || a.AIAssessment;
            if (assessment) {
                const aiPanel = document.createElement('section');
                aiPanel.className = `gw-alert-ai gw-alert-ai-${assessment.severity || 'medium'}`;
                const aiHead = document.createElement('div');
                aiHead.className = 'gw-alert-ai-head';
                const aiLabel = document.createElement('strong');
                aiLabel.textContent = `KI-ASSESSMENT · ${String(assessment.severity || '').toUpperCase()} · ${Number(assessment.confidence || 0)}%`;
                const aiStatus = document.createElement('span');
                aiStatus.textContent = String(assessment.status || 'unverified').toUpperCase();
                aiHead.append(aiLabel, aiStatus);
                const aiSummary = document.createElement('p');
                aiSummary.textContent = assessment.summary || '';
                const aiRationale = document.createElement('details');
                const rationaleTitle = document.createElement('summary');
                rationaleTitle.textContent = 'Begründung, Unsicherheiten und Maßnahmen';
                const rationaleBody = document.createElement('div');
                rationaleBody.className = 'gw-alert-ai-detail';
                appendTextElement(rationaleBody, 'p', assessment.rationale || '');
                const uncertainties = Array.isArray(assessment.uncertainties) ? assessment.uncertainties : [];
                const actions = Array.isArray(assessment.recommended_actions) ? assessment.recommended_actions : [];
                if (uncertainties.length) appendTextElement(rationaleBody, 'p', `Unsicherheiten: ${uncertainties.join(' · ')}`);
                if (actions.length) appendTextElement(rationaleBody, 'p', `Nächste Schritte: ${actions.join(' · ')}`);
                aiRationale.append(rationaleTitle, rationaleBody);
                aiPanel.append(aiHead, aiSummary, aiRationale);
                row.appendChild(aiPanel);
            }
            row.addEventListener('click', () => {
                const rid = String(a.region || a.Region || '').toUpperCase();
                const geo = REGION_GEO[rid];
                if (geo) focusGlobeOnLonLat(geo.lon, geo.lat);
                void openExplainDrawer(rid || 'GERMANY');
            });
            list.appendChild(row);
        });
    } catch (e) {
        list.replaceChildren();
        appendTextElement(list, 'div', 'Alerts unavailable', 'gw-risk-empty');
        if (metric) metric.textContent = 'ALERT —';
    }
}

export function subscribeGlobalWatchCommands() {
    if (globalWatchCommandStream || typeof EventSource === 'undefined') return;
    const stream = new EventSource(`${state.API_BASE}/v1/intelligence/stream`);
    setGlobalWatchCommandStream(stream);
    stream.addEventListener('intelligence', event => {
        try {
            const payload = JSON.parse(event.data);
            const command = payload.command;
            if (!command || payload.type !== 'global_watch.command') return;
            const status = document.getElementById('gw-refresh-info');
            if (typeof window.AETHEL_GW_COMMAND === 'function') {
                const bridged = window.AETHEL_GW_COMMAND({
                    action: command.action,
                    region: command.region,
                    lat: command.latitude,
                    lon: command.longitude,
                    zoom: command.zoom,
                    hours: command.hours,
                    view: command.view,
                    title: command.label,
                    body: command.body,
                });
                if (bridged) {
                    if (status) status.textContent = `AI CMD: ${command.action}`;
                    requestGlobeRender();
                    return;
                }
            }
            if (command.action === 'focus') {
                focusGlobeOnLonLat(command.longitude, command.latitude, { scale: command.zoom, snap: true });
                if (status) status.textContent = `AI FOCUS: ${command.label || `${Number(command.latitude).toFixed(2)}, ${Number(command.longitude).toFixed(2)}`}`;
            } else if (command.action === 'focus_region' && command.region) {
                focusRegionByKey(command.region);
                if (status) status.textContent = `AI REGION: ${String(command.region).toUpperCase()}`;
            } else if (command.action === 'time_window') {
                setGwTimeWindowHours(command.hours);
                const active = document.querySelector('.gw-domain-filter.active');
                void refreshOSINTFeed(active ? active.getAttribute('data-domain') : 'all', false);
                if (status) status.textContent = `AI TIME: ${command.hours > 0 ? command.hours + 'h' : 'ALL'}`;
            } else if (command.action === 'navigate' && command.view) {
                if (typeof window.AETHEL_NAVIGATE_VIEW === 'function') window.AETHEL_NAVIGATE_VIEW(command.view);
                if (status) status.textContent = `AI NAV: ${String(command.view).toUpperCase()}`;
            }
            requestGlobeRender();
        } catch (_) {}
    });
}
