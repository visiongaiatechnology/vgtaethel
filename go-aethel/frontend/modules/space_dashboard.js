// STATUS: DIAMANT VGT SUPREME
// Space Dashboard — Solarcommander mission layout + full NOAA telemetry (Aethel style)
import { state } from './state.js';
import { formatMarkdown } from './chat.js';

let activeSdoChannel = '193';
let activeAuroraHemi = 'north';
let activeChart = 'xray';
let lastWeatherPayload = null;
let lastSyncAt = 0;
let lastImageLoadedAt = 0;
let ageTicker = null;
let utcTicker = null;
let syncCooldownUntil = 0;
const sdoFetchGeneration = Object.create(null);
const STALE_AFTER_MS = 20 * 60 * 1000;
const SYNC_COOLDOWN_MS = 30_000;
const INITIAL_UPLINK_RETRIES_MS = Object.freeze([250, 750, 1_500]);
let chartResizeObserver = null;
let chartRedrawFrame = 0;

const SDO_CHANNELS = Object.freeze([
    { id: '193', label: '193 Å', title: 'Koronale Löcher', color: '#c99534' },
    { id: '171', label: '171 Å', title: 'Ruhige Korona', color: '#e8c347' },
    { id: '304', label: '304 Å', title: 'Chromosphäre', color: '#f26e3f' },
    { id: '131', label: '131 Å', title: 'Flares', color: '#3fa8f2' },
    { id: 'HMI', label: 'HMI', title: 'Magnetogramm', color: '#d66b22' }
]);

function sdoProxyURL(channel, forceFresh) {
    const ch = encodeURIComponent(channel);
    if (forceFresh || sdoFetchGeneration[channel] == null) {
        sdoFetchGeneration[channel] = Date.now();
    }
    let url = `${state.API_BASE}/v1/space/sdo_image?channel=${ch}&g=${sdoFetchGeneration[channel]}`;
    if (forceFresh) url += '&refresh=1';
    return url;
}

function formatAge(ms) {
    if (!ms || ms <= 0) return 'noch nicht geladen';
    const age = Date.now() - ms;
    if (age < 45_000) return 'gerade eben';
    const mins = Math.floor(age / 60_000);
    if (mins < 60) return `vor ${mins} Min.`;
    const hours = Math.floor(mins / 60);
    const rem = mins % 60;
    return rem ? `vor ${hours} Std. ${rem} Min.` : `vor ${hours} Std.`;
}

function isStale(ms) {
    return !ms || Date.now() - ms >= STALE_AFTER_MS;
}

function updateAgeUI() {
    const ageEl = document.getElementById('space-data-age');
    const staleEl = document.getElementById('space-stale-banner');
    const badge = document.getElementById('space-sync-badge');
    const imgAgeEl = document.getElementById('space-image-age');
    const lastSyncEl = document.getElementById('space-last-sync-time');
    const weatherMs = lastSyncAt;
    const imageMs = lastImageLoadedAt || lastSyncAt;

    if (ageEl) ageEl.textContent = `Telemetrie: ${formatAge(weatherMs)} · SDO: ${formatAge(imageMs)}`;
    if (imgAgeEl) imgAgeEl.textContent = `Bildstand: ${formatAge(imageMs)} · nur bei SYNC`;
    if (lastSyncEl && weatherMs) {
        lastSyncEl.textContent = new Date(weatherMs).toLocaleTimeString();
    }
    if (badge) {
        const stale = isStale(weatherMs) || isStale(imageMs);
        badge.classList.toggle('online', !stale);
        badge.classList.toggle('stale', stale);
        badge.textContent = stale
            ? 'DATEN ÄLTER ALS 20 MIN — SYNC EMPFOHLEN'
            : 'MANUELLER SYNC · KEIN AUTO-POLL';
    }
    if (staleEl) {
        const stale = isStale(weatherMs) || isStale(imageMs);
        staleEl.hidden = !stale;
        if (stale) {
            staleEl.textContent = 'Letzter Uplink älter als 20 Minuten. NOAA/NASA werden erst bei „SYNC TELEMETRIE“ erneut angefragt.';
        }
    }
}

function startAgeTicker() {
    if (!ageTicker) ageTicker = window.setInterval(updateAgeUI, 30_000);
    if (!utcTicker) {
        utcTicker = window.setInterval(() => {
            const el = document.getElementById('space-utc-clock');
            if (el) el.textContent = new Date().toISOString().slice(11, 19);
        }, 1000);
    }
    updateAgeUI();
}

export async function fetchSpaceWeatherData(opts = {}) {
    const forceImages = !!opts.forceImages;
    const container = document.getElementById('space-dashboard-content');
    const btn = document.getElementById('btn-refresh-space');
    if (forceImages && Date.now() < syncCooldownUntil) {
        return;
    }
    try {
        if (btn) {
            btn.disabled = true;
            btn.textContent = 'SYNC…';
        }
        if (forceImages) {
            for (const ch of SDO_CHANNELS) sdoFetchGeneration[ch.id] = Date.now();
            sdoFetchGeneration.aurora_n = Date.now();
            sdoFetchGeneration.aurora_s = Date.now();
        }
        const q = forceImages ? '?refresh=1' : '';
        const data = await fetchSpaceSnapshotWithRetry(q);
        lastWeatherPayload = data;
        lastSyncAt = Date.now();
        renderSpaceDashboardUI(data, { forceImages });
        startAgeTicker();
        if (forceImages) {
            syncCooldownUntil = Date.now() + SYNC_COOLDOWN_MS;
            startCooldownUI();
        }
    } catch (e) {
        console.warn('Space weather fetch failed', e);
        if (container && !container.querySelector('.space-mission-grid')) {
            container.replaceChildren();
            const err = document.createElement('div');
            err.className = 'space-error-banner';
            err.textContent = 'Weltraumwetter-Uplink fehlgeschlagen.';
            const retry = document.createElement('button');
            retry.type = 'button';
            retry.className = 'space-btn';
            retry.textContent = 'SYNC ERNEUT';
            retry.addEventListener('click', () => fetchSpaceWeatherData({ forceImages: true }));
            container.append(err, retry);
        }
    } finally {
        if (btn && Date.now() >= syncCooldownUntil) {
            btn.disabled = false;
            btn.textContent = 'SYNC TELEMETRIE';
        }
    }
}

async function fetchSpaceSnapshotWithRetry(query) {
    let lastError = null;
    for (let attempt = 0; attempt <= INITIAL_UPLINK_RETRIES_MS.length; attempt++) {
        try {
            const res = await fetch(`${state.API_BASE}/v1/space/weather${query}`, { cache: 'no-store' });
            if (!res.ok) throw new Error(`HTTP ${res.status}`);
            const data = await res.json();
            if (!data || typeof data !== 'object') throw new Error('Ungültiger Telemetrie-Snapshot');
            return data;
        } catch (error) {
            lastError = error;
            const delay = INITIAL_UPLINK_RETRIES_MS[attempt];
            if (delay == null) break;
            await new Promise(resolve => window.setTimeout(resolve, delay));
        }
    }
    throw lastError || new Error('Weltraumwetter-Uplink nicht verfügbar');
}

function startCooldownUI() {
    const btn = document.getElementById('btn-refresh-space');
    const cd = document.getElementById('space-cooldown');
    if (!btn) return;
    const tick = () => {
        const left = Math.ceil((syncCooldownUntil - Date.now()) / 1000);
        if (left <= 0) {
            btn.disabled = false;
            btn.textContent = 'SYNC TELEMETRIE';
            if (cd) cd.hidden = true;
            return;
        }
        btn.disabled = true;
        btn.textContent = `COOLDOWN ${left}s`;
        if (cd) {
            cd.hidden = false;
            cd.textContent = `Sync-Cooldown ${left}s (Schutz vor Blacklisting)`;
        }
        window.setTimeout(tick, 1000);
    };
    tick();
}

function el(tag, className, text) {
    const n = document.createElement(tag);
    if (className) n.className = className;
    if (text != null) n.textContent = text;
    return n;
}

function barFill(pct, className) {
    const track = el('div', 'space-bar-track');
    const fill = el('div', 'space-bar-fill' + (className ? ` ${className}` : ''));
    fill.style.width = `${Math.max(0, Math.min(100, pct))}%`;
    track.appendChild(fill);
    return track;
}

function splitBar(value, negScale, posScale) {
    // Bipolar bar: left = negative, right = positive
    const wrap = el('div', 'space-split-bar');
    const negSide = el('div', 'space-split-neg');
    const posSide = el('div', 'space-split-pos');
    const negFill = el('div', 'space-split-fill is-neg');
    const posFill = el('div', 'space-split-fill is-pos');
    if (value < 0) {
        negFill.style.width = `${Math.min(100, (Math.abs(value) / negScale) * 100)}%`;
        posFill.style.width = '0%';
    } else {
        negFill.style.width = '0%';
        posFill.style.width = `${Math.min(100, (value / posScale) * 100)}%`;
    }
    negSide.appendChild(negFill);
    posSide.appendChild(posFill);
    wrap.append(negSide, posSide);
    return wrap;
}

function scaleSegments(level, max, tone) {
    const wrap = el('div', 'space-scale-bar');
    const n = Math.max(0, Math.min(max, Number(level) || 0));
    for (let i = 1; i <= max; i += 1) {
        const seg = el('div', 'space-scale-seg' + (i <= n ? ` is-on tone-${tone}` : ''));
        wrap.appendChild(seg);
    }
    return wrap;
}

function panel(title, accent, bodyNodes, extraClass = '') {
    const card = el('section', `space-panel accent-${accent} ${extraClass}`.trim());
    const head = el('header', 'space-panel-head', title);
    const body = el('div', 'space-panel-body');
    for (const n of bodyNodes) body.appendChild(n);
    card.append(head, body);
    return card;
}

function bindSdoImage(img, loader, errorEl, channel, forceFresh = false) {
    const url = sdoProxyURL(channel, forceFresh);
    if (img.dataset.sdoUrl === url && img.classList.contains('is-ready')) {
        loader.hidden = true;
        errorEl.hidden = true;
        return;
    }
    loader.hidden = false;
    errorEl.hidden = true;
    img.classList.remove('is-ready');
    img.dataset.sdoUrl = url;
    img.src = url;
    img.onload = () => {
        loader.hidden = true;
        errorEl.hidden = true;
        img.classList.add('is-ready');
        lastImageLoadedAt = Date.now();
        updateAgeUI();
    };
    img.onerror = () => {
        loader.hidden = true;
        errorEl.hidden = false;
        img.classList.remove('is-ready');
    };
}

function buildImager(data, forceImages) {
    const card = el('section', 'space-imager space-panel accent-sun');
    const frame = el('div', 'space-imager-frame');

    const badges = el('div', 'space-imager-badges');
    badges.append(el('span', 'space-chip sun', 'SDO AIA'), el('span', 'space-chip', (SDO_CHANNELS.find(c => c.id === activeSdoChannel) || SDO_CHANNELS[0]).label));
    badges.lastChild.id = 'space-wavelength-label';

    const sunspot = el('div', 'space-imager-sunspot');
    sunspot.append(el('div', 'space-metric-label', 'Sonnenflecken (SSN)'), el('div', 'space-imager-ssn', String(data.sunspot_count ?? '—')));

    const img = el('img');
    img.id = 'space-sun-image';
    img.alt = 'NASA SDO';
    img.decoding = 'async';

    const loader = el('div', 'space-img-loader');
    loader.id = 'space-img-loader';
    loader.appendChild(el('div', 'space-spinner'));

    const errorEl = el('div', 'space-img-error');
    errorEl.id = 'space-img-error';
    errorEl.hidden = true;
    errorEl.append(el('strong', '', 'BILD NICHT VERFÜGBAR'), el('span', '', 'SDO-Proxy konnte das NASA-Bild nicht laden.'));
    const errRetry = el('button', 'space-btn compact', 'ERNEUT LADEN');
    errRetry.type = 'button';
    errRetry.addEventListener('click', () => {
        sdoFetchGeneration[activeSdoChannel] = Date.now();
        bindSdoImage(img, loader, errorEl, activeSdoChannel, true);
    });
    errorEl.appendChild(errRetry);

    frame.append(badges, sunspot, img, loader, errorEl, el('div', 'space-scanline'));

    const controls = el('div', 'space-channel-controls');
    controls.setAttribute('role', 'toolbar');
    for (const ch of SDO_CHANNELS) {
        const btn = el('button', 'space-channel-btn' + (ch.id === activeSdoChannel ? ' is-active' : ''));
        btn.type = 'button';
        btn.style.setProperty('--ch-color', ch.color);
        btn.title = `${ch.label} · ${ch.title}`;
        btn.dataset.channel = ch.id;
        if (ch.id === 'HMI') {
            btn.textContent = 'M';
            btn.classList.add('is-hmi');
        }
        btn.addEventListener('click', () => {
            activeSdoChannel = ch.id;
            controls.querySelectorAll('.space-channel-btn').forEach(b => {
                const on = b.dataset.channel === activeSdoChannel;
                b.classList.toggle('is-active', on);
            });
            const lab = document.getElementById('space-wavelength-label');
            if (lab) lab.textContent = ch.label;
            bindSdoImage(img, loader, errorEl, activeSdoChannel, false);
        });
        controls.appendChild(btn);
    }

    const caption = el('div', 'space-imager-caption', 'Bildstand: —');
    caption.id = 'space-image-age';
    card.append(frame, controls, caption);
    bindSdoImage(img, loader, errorEl, activeSdoChannel, forceImages);
    return card;
}

function buildAuroraMap(data, forceImages) {
    const card = el('section', 'space-aurora-map space-panel accent-green');
    const head = el('header', 'space-panel-head row');
    head.appendChild(el('span', '', 'POLARLICHTOVAL · OVATION'));
    const toggles = el('div', 'space-aurora-toggle');
    const btnN = el('button', 'space-mini-btn' + (activeAuroraHemi === 'north' ? ' is-active' : ''), 'NORD');
    const btnS = el('button', 'space-mini-btn' + (activeAuroraHemi === 'south' ? ' is-active' : ''), 'SÜD');
    btnN.type = btnS.type = 'button';
    toggles.append(btnN, btnS);
    head.appendChild(toggles);

    const frame = el('div', 'space-aurora-frame');
    const img = el('img');
    img.id = 'space-aurora-img';
    img.alt = 'Aurora OVATION map';
    const loader = el('div', 'space-img-loader');
    loader.appendChild(el('div', 'space-spinner'));
    frame.append(img, loader);

    const auroraPower = () => {
        const value = activeAuroraHemi === 'south' ? data.aurora_south_power_gw : data.aurora_north_power_gw;
        const fallback = data.aurora_hemispheric_power_gw;
        const numeric = Number(value ?? fallback);
        return Number.isFinite(numeric) ? `${numeric.toFixed(1)} GW` : '—';
    };
    const foot = el('div', 'space-aurora-foot');
    const left = el('div');
    left.append(el('div', 'space-metric-label', 'Hemispheric Power'), el('div', 'space-imager-ssn', auroraPower()));
    const powerEl = left.lastChild;
    const right = el('div');
    right.append(el('div', 'space-metric-label', 'Sichtbarkeit'), el('div', 'space-aurora-vis', String(data.aurora_activity_level || '—')));
    const lat = el('div', 'space-panel-note', `Min-Breite: ${data.aurora_min_lat || '—'} · Konfidenz ${data.aurora_confidence ?? 0}%`);
    foot.append(left, right);

    const loadAurora = (force) => {
        const ch = activeAuroraHemi === 'north' ? 'aurora_n' : 'aurora_s';
        const url = sdoProxyURL(ch, force);
        loader.hidden = false;
        img.classList.remove('is-ready');
        img.src = url;
        img.onload = () => {
            loader.hidden = true;
            img.classList.add('is-ready');
        };
        img.onerror = () => { loader.hidden = true; };
        powerEl.textContent = auroraPower();
    };

    btnN.addEventListener('click', () => {
        activeAuroraHemi = 'north';
        btnN.classList.add('is-active');
        btnS.classList.remove('is-active');
        loadAurora(false);
    });
    btnS.addEventListener('click', () => {
        activeAuroraHemi = 'south';
        btnS.classList.add('is-active');
        btnN.classList.remove('is-active');
        loadAurora(false);
    });

    card.append(head, frame, foot, lat);
    loadAurora(forceImages);
    return card;
}

function buildScales(data) {
    const r = Number(data.r_scale) || 0;
    const s = Number(data.s_scale) || 0;
    const g = Number(data.g_scale) || 0;
    const nodes = [
        { name: 'Radio Blackouts (R)', level: r, tone: 'r', label: `R${r} / R5` },
        { name: 'Solar Radiation (S)', level: s, tone: 's', label: `S${s} / S5` },
        { name: 'Geomagnetic Storms (G)', level: g, tone: 'g', label: `G${g} / G5` }
    ].map(row => {
        const block = el('div', 'space-scale-row');
        const top = el('div', 'space-scale-top');
        top.append(el('span', '', row.name), el('strong', `tone-${row.tone}`, row.label));
        block.append(top, scaleSegments(row.level, 5, row.tone));
        return block;
    });
    return panel('NOAA SPACE WEATHER SCALES · R / S / G', 'orange', nodes);
}

function buildWindMag(data) {
    const bt = Number(data.bt_total_nt) || 0;
    const bz = Number(data.bz_vector_nt) || 0;
    const wind = Number(data.solar_wind_speed_km_s) || 0;
    const dens = Number(data.solar_wind_density_p_cm3) || 0;

    const row = (label, value, unit, bar) => {
        const wrap = el('div', 'space-wm-row');
        const top = el('div', 'space-wm-top');
        top.append(el('span', 'space-metric-label', label), el('strong', '', `${value}${unit ? ' ' + unit : ''}`));
        wrap.append(top, bar);
        return wrap;
    };

    return panel('INTERPLANETARE DATEN · IMF / WIND', 'purple', [
        row('Bt (Total)', bt.toFixed(1), 'nT', barFill((bt / 40) * 100, 'fill-cyan')),
        row('Bz (Süd/Nord)', bz.toFixed(1), 'nT', splitBar(bz, 25, 25)),
        row('Wind Geschw.', Math.round(wind).toString(), 'km/s', barFill((wind / 1000) * 100, 'fill-blue')),
        row('Dichte', dens.toFixed(1), 'p/cm³', barFill((dens / 50) * 100, 'fill-orange'))
    ]);
}

function telemetryAvailable(data, key) {
    const source = data?.telemetry_sources?.[key];
    return source?.available === true;
}

function buildKpCard(data) {
    const kp = Number(data.kp_index);
    const g = Number(data.g_scale) || 0;
    const available = telemetryAvailable(data, 'kp');
    const body = el('div', 'space-kp-body');
    const hero = el('div', 'space-kp-hero-row');
    hero.append(
        el('div', 'space-kp-hero', available && Number.isFinite(kp) ? kp.toFixed(1) : '—'),
        el('div', 'space-kp-meta', available ? `/ 9.0\nG${g}` : 'NOAA\nWARTET')
    );
    const status = el('div', 'space-status-pill', available ? String(data.kp_status || '—') : 'UPLINK NICHT VERFÜGBAR');
    const bar = barFill(available && Number.isFinite(kp) ? (kp / 9) * 100 : 0, g >= 5 ? 'fill-red' : g >= 1 ? 'fill-orange' : 'fill-green');
    body.append(hero, status, bar);
    return panel('Kp INDEX', 'cyan', [body], 'space-border-sun');
}

function buildDstCard(data) {
    const available = telemetryAvailable(data, 'dst');
    const dst = Number(data.dst_index_nt);
    const body = el('div');
    const row = el('div', 'space-dst-row');
    row.append(el('div', 'space-dst-value', available && Number.isFinite(dst) ? dst.toFixed(0) : '—'), el('span', 'space-metric-label', available ? 'nT · 0 = normal' : 'Kyoto-Uplink ausstehend'));
    body.append(
        el('div', 'space-status-pill', available ? String(data.dst_status || '—') : 'UPLINK NICHT VERFÜGBAR'),
        row,
        splitBar(available && Number.isFinite(dst) ? dst : 0, 300, 50)
    );
    return panel('Dst INDEX (Kyoto)', 'purple', [body], 'space-border-purple');
}

function buildFlareHistory(data) {
    const body = el('div', 'space-flare-grid');
    const maxBox = el('div');
    maxBox.append(el('div', 'space-metric-label', 'Maximal 72h'), el('div', 'space-flare-big', String(data.flare_max_72h || data.solar_xray_flux_class || '—')));
    const lastBox = el('div', 'space-flare-last');
    lastBox.append(
        el('div', 'space-metric-label', 'Letzter Peak'),
        el('div', 'space-flare-big', String(data.flare_last_class || '—')),
        el('div', 'space-metric-label', String(data.flare_last_time || '—'))
    );
    body.append(maxBox, lastBox);

    const now = el('div', 'space-panel-note', `Jetzt: ${data.solar_xray_flux_class || '—'} · Proton ≥10 MeV: ${Number(data.proton_flux_10mev || 0).toFixed(1)} pfu`);
    const list = el('div', 'space-flare-list');
    const flares = Array.isArray(data.recent_flares) && data.recent_flares.length
        ? data.recent_flares
        : ['Keine C+-Peaks im Fenster'];
    for (const f of flares) list.appendChild(el('div', 'space-flare-row', String(f)));

    return panel('FLARE AKTIVITÄT (GOES)', 'orange', [body, now, list], 'space-border-red');
}

function buildForecast(data) {
    const grid = el('div', 'space-forecast-grid');
    const mk = (label, pct, tone) => {
        const box = el('div', 'space-forecast-box');
        box.append(
            el('div', 'space-metric-label', label),
            el('div', `space-forecast-pct tone-${tone}`, `${Number(pct) || 0}%`),
            barFill(Number(pct) || 0, `fill-${tone === 'r' ? 'orange' : tone === 'g' ? 'red' : 'purple'}`)
        );
        return box;
    };
    grid.append(
        mk('M-Class Flare', data.prob_m_class, 'r'),
        mk('X-Class Flare', data.prob_x_class, 'g'),
        mk('Proton Event', data.prob_proton, 's')
    );
    const geo = el('div', 'space-geo-forecast');
    geo.append(
        el('span', '', 'Geomagnetische Prognose (24h)'),
        el('strong', 'space-geo-pill', String(data.geo_forecast || 'STABLE'))
    );
    return panel('VORHERSAGE & WAHRSCHEINLICHKEIT', 'green', [grid, geo]);
}

function buildCharts(data) {
    const card = el('section', 'space-panel accent-cyan space-chart-panel');
    const head = el('header', 'space-panel-head row');
    head.appendChild(el('span', '', 'TELEMETRIE GRAPHEN'));
    const tabs = el('div', 'space-chart-tabs');
    const kinds = [
        { id: 'xray', label: 'X-Ray' },
        { id: 'mag', label: 'Mag Bz' },
        { id: 'proton', label: 'Proton' },
        { id: 'wind', label: 'Wind' },
        { id: 'kp', label: 'Kp' },
        { id: 'dst', label: 'Dst' }
    ];
    for (const k of kinds) {
        const b = el('button', 'space-mini-btn' + (activeChart === k.id ? ' is-active' : ''), k.label);
        b.type = 'button';
        b.addEventListener('click', () => {
            activeChart = k.id;
            tabs.querySelectorAll('.space-mini-btn').forEach(x => x.classList.remove('is-active'));
            b.classList.add('is-active');
            drawActiveChart(data);
        });
        tabs.appendChild(b);
    }
    head.appendChild(tabs);

    const canvas = document.createElement('canvas');
    canvas.id = 'space-telemetry-chart';
    canvas.className = 'space-chart-canvas';
    const wrap = el('div', 'space-chart-wrap');
    const tooltip = el('div', 'space-chart-tooltip');
    tooltip.hidden = true;
    tooltip.id = 'space-chart-tooltip';
    wrap.append(canvas, tooltip);
    bindChartInteractions(canvas, tooltip, data);
    card.append(head, wrap);

    // Canvas dimensions follow the rendered panel and device pixel ratio.
    queueMicrotask(() => installChartResizeObserver(data));
    return card;
}

function seriesForChart(data, kind) {
    if (kind === 'xray') return data.series_xray || [];
    if (kind === 'mag') return data.series_mag_bz || [];
    if (kind === 'proton') return data.series_proton || [];
    if (kind === 'wind') return data.series_wind || [];
    if (kind === 'kp') return data.series_kp || [];
    if (kind === 'dst') return data.series_dst || [];
    return [];
}

function drawActiveChartLegacy(data) {
    const canvas = document.getElementById('space-telemetry-chart');
    if (!canvas) return;
    const series = seriesForChart(data, activeChart);
    const ctx = canvas.getContext('2d');
    const w = canvas.width;
    const h = canvas.height;
    ctx.clearRect(0, 0, w, h);
    ctx.fillStyle = 'rgba(0,0,0,0.35)';
    ctx.fillRect(0, 0, w, h);

    if (!series.length) {
        ctx.fillStyle = 'rgba(160,180,200,0.7)';
        ctx.font = '12px monospace';
        ctx.fillText('Keine Seriendaten — SYNC für Live-NOAA', 16, h / 2);
        return;
    }

    const vals = series.map(p => Number(p.v)).filter(Number.isFinite);
    let min = Math.min(...vals);
    let max = Math.max(...vals);
    if (min === max) {
        min -= 1;
        max += 1;
    }
    const pad = 16;
    const plotW = w - pad * 2;
    const plotH = h - pad * 2;

    // grid
    ctx.strokeStyle = 'rgba(255,255,255,0.06)';
    ctx.lineWidth = 1;
    for (let i = 0; i <= 4; i++) {
        const y = pad + (plotH * i) / 4;
        ctx.beginPath();
        ctx.moveTo(pad, y);
        ctx.lineTo(w - pad, y);
        ctx.stroke();
    }

    const color = activeChart === 'xray' ? '#ff7b00'
        : activeChart === 'mag' ? '#a78bfa'
            : activeChart === 'proton' ? '#34d399'
                : '#00f0ff';

    ctx.strokeStyle = color;
    ctx.lineWidth = 2;
    ctx.beginPath();
    series.forEach((p, i) => {
        const v = Number(p.v);
        if (!Number.isFinite(v)) return;
        const x = pad + (i / Math.max(1, series.length - 1)) * plotW;
        const y = pad + (1 - (v - min) / (max - min)) * plotH;
        if (i === 0) ctx.moveTo(x, y);
        else ctx.lineTo(x, y);
    });
    ctx.stroke();

    // fill under
    ctx.lineTo(pad + plotW, pad + plotH);
    ctx.lineTo(pad, pad + plotH);
    ctx.closePath();
    ctx.fillStyle = color + '22';
    ctx.fill();

    ctx.fillStyle = 'rgba(200,220,240,0.8)';
    ctx.font = '10px monospace';
    ctx.fillText(`${activeChart.toUpperCase()} · n=${series.length}`, pad, 12);
    ctx.fillText(`min ${min.toFixed(2)}  max ${max.toFixed(2)}`, pad, h - 4);
}

function chartDescriptor(kind) {
    const descriptors = {
        xray: { label: 'X-RAY FLUX', unit: 'log₁₀ W/m²', color: '#ff7b00', decimals: 2 },
        mag: { label: 'IMF Bz', unit: 'nT', color: '#a78bfa', decimals: 1 },
        proton: { label: 'PROTON ≥10 MeV', unit: 'pfu', color: '#34d399', decimals: 2 },
        wind: { label: 'SOLAR WIND', unit: 'km/s', color: '#00f0ff', decimals: 0 },
        kp: { label: 'PLANETARY Kp', unit: 'Index / 9', color: '#f8cf4d', decimals: 2 },
        dst: { label: 'DST', unit: 'nT', color: '#f472b6', decimals: 0 }
    };
    return descriptors[kind] || descriptors.xray;
}

function chartGeometry(canvas) {
    const rect = canvas.getBoundingClientRect();
    const width = Math.max(320, Math.round(rect.width || 640));
    const height = Math.max(220, Math.round(rect.height || 260));
    const ratio = Math.min(2, Math.max(1, window.devicePixelRatio || 1));
    if (canvas.width !== width * ratio || canvas.height !== height * ratio) {
        canvas.width = width * ratio;
        canvas.height = height * ratio;
    }
    const ctx = canvas.getContext('2d');
    ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
    return { ctx, w: width, h: height };
}

function drawActiveChart(data, hoveredIndex = -1) {
    const canvas = document.getElementById('space-telemetry-chart');
    if (!canvas) return;
    const series = seriesForChart(data, activeChart).filter(point => Number.isFinite(Number(point?.v)));
    const { ctx, w, h } = chartGeometry(canvas);
    const descriptor = chartDescriptor(activeChart);
    ctx.clearRect(0, 0, w, h);
    ctx.fillStyle = 'rgba(0,0,0,0.35)';
    ctx.fillRect(0, 0, w, h);

    if (!series.length) {
        ctx.fillStyle = 'rgba(160,180,200,0.7)';
        ctx.font = '12px monospace';
        ctx.fillText('Für diese Quelle liegen noch keine Live-Seriendaten vor.', 16, h / 2);
        return;
    }

    const vals = series.map(point => Number(point.v));
    let min = Math.min(...vals);
    let max = Math.max(...vals);
    if (min === max) {
        min -= 1;
        max += 1;
    }
    const pad = { left: 48, right: 18, top: 28, bottom: 30 };
    const plotW = w - pad.left - pad.right;
    const plotH = h - pad.top - pad.bottom;
    ctx.strokeStyle = 'rgba(255,255,255,0.06)';
    ctx.lineWidth = 1;
    ctx.font = '10px monospace';
    for (let i = 0; i <= 4; i++) {
        const y = pad.top + (plotH * i) / 4;
        ctx.beginPath();
        ctx.moveTo(pad.left, y);
        ctx.lineTo(w - pad.right, y);
        ctx.stroke();
        ctx.fillStyle = 'rgba(200,220,240,0.68)';
        ctx.fillText((max - ((max - min) * i) / 4).toFixed(descriptor.decimals), 6, y + 3);
    }

    ctx.strokeStyle = descriptor.color;
    ctx.lineWidth = 2;
    ctx.beginPath();
    series.forEach((point, index) => {
        const value = Number(point.v);
        const x = pad.left + (index / Math.max(1, series.length - 1)) * plotW;
        const y = pad.top + (1 - (value - min) / (max - min)) * plotH;
        if (index === 0) ctx.moveTo(x, y);
        else ctx.lineTo(x, y);
    });
    ctx.stroke();
    ctx.lineTo(pad.left + plotW, pad.top + plotH);
    ctx.lineTo(pad.left, pad.top + plotH);
    ctx.closePath();
    ctx.fillStyle = descriptor.color + '22';
    ctx.fill();

    ctx.fillStyle = 'rgba(200,220,240,0.8)';
    ctx.fillText(`${descriptor.label} · n=${series.length} · ${descriptor.unit}`, pad.left, 15);
    const start = formatChartTime(series[0]?.t);
    const end = formatChartTime(series[series.length - 1]?.t);
    ctx.fillText(start, pad.left, h - 8);
    ctx.fillText(end, w - pad.right - ctx.measureText(end).width, h - 8);

    if (hoveredIndex >= 0 && hoveredIndex < series.length) {
        const point = series[hoveredIndex];
        const value = Number(point.v);
        const x = pad.left + (hoveredIndex / Math.max(1, series.length - 1)) * plotW;
        const y = pad.top + (1 - (value - min) / (max - min)) * plotH;
        ctx.save();
        ctx.strokeStyle = 'rgba(255,255,255,0.38)';
        ctx.setLineDash([3, 3]);
        ctx.beginPath();
        ctx.moveTo(x, pad.top);
        ctx.lineTo(x, pad.top + plotH);
        ctx.stroke();
        ctx.setLineDash([]);
        ctx.fillStyle = descriptor.color;
        ctx.beginPath();
        ctx.arc(x, y, 4.5, 0, Math.PI * 2);
        ctx.fill();
        ctx.restore();
    }
    canvas._aethelChart = { series, descriptor, geometry: { ...pad, plotW, plotH } };
}

function formatChartTime(rawTime) {
    const parsed = new Date(rawTime);
    if (Number.isNaN(parsed.getTime())) return String(rawTime || '—').slice(0, 18);
    return parsed.toLocaleString([], { hour: '2-digit', minute: '2-digit', day: '2-digit', month: '2-digit' });
}

function bindChartInteractions(canvas, tooltip, data) {
    canvas.addEventListener('mousemove', event => {
        const meta = canvas._aethelChart;
        if (!meta?.series?.length) return;
        const rect = canvas.getBoundingClientRect();
        const localX = event.clientX - rect.left;
        const index = Math.max(0, Math.min(meta.series.length - 1, Math.round(((localX - meta.geometry.left) / meta.geometry.plotW) * Math.max(1, meta.series.length - 1))));
        const point = meta.series[index];
        drawActiveChart(data, index);
        tooltip.hidden = false;
        tooltip.textContent = `${meta.descriptor.label}: ${Number(point.v).toFixed(meta.descriptor.decimals)} ${meta.descriptor.unit} · ${formatChartTime(point.t)}`;
        tooltip.style.left = `${Math.max(8, Math.min(rect.width - 270, localX + 12))}px`;
        tooltip.style.top = `${Math.max(8, event.clientY - rect.top + 12)}px`;
    });
    canvas.addEventListener('mouseleave', () => {
        tooltip.hidden = true;
        drawActiveChart(data);
    });
}

function installChartResizeObserver(data) {
    const canvas = document.getElementById('space-telemetry-chart');
    if (!canvas) return;
    if (chartResizeObserver) chartResizeObserver.disconnect();
    chartResizeObserver = new ResizeObserver(() => {
        cancelAnimationFrame(chartRedrawFrame);
        chartRedrawFrame = requestAnimationFrame(() => drawActiveChart(data));
    });
    chartResizeObserver.observe(canvas.parentElement);
    drawActiveChart(data);
}

function buildAnalysisConsole() {
    const card = el('section', 'space-panel space-analysis accent-cyan');
    const head = el('header', 'space-panel-head row');
    const titles = el('div');
    titles.append(el('div', 'space-eyebrow', 'AETHEL WELTRAUM-INTELLIGENZ'), el('h3', 'space-analysis-title', 'Kognitive Astrophysik-Analyse'));
    const btn = el('button', 'space-btn primary', 'ASTROPHYSIK-ANALYSE STARTEN ›');
    btn.type = 'button';
    btn.id = 'btn-generate-space-analysis';
    btn.addEventListener('click', generateSpaceAnalysis);
    head.append(titles, btn);
    const result = el('div', 'space-analysis-result', 'Öffnet ein Analyse-Popup: Datenpakete nacheinander → Zwischenberichte → Finaler Risikobericht (ohne Chat-Wechsel).');
    result.id = 'space-analysis-result';
    card.append(head, result);
    return card;
}

/* ── Multi-step analysis modal (wide tabs + orbit core + simulated thinking) ── */

let analysisAbort = null;
let analysisRunning = false;
let analysisActiveTab = 'live';
let thinkingTimer = null;
let progressTimer = null;
let currentPackageProgress = 0;
let pipelineOrder = [];
const packageReports = new Map(); // id → { title, risk, body, source, progress, status }

const PACKAGE_VISUALS = Object.freeze({
    geomag: { emoji: '🌍', hue: '#34d399', label: 'Geomagnetik', short: 'GEOMAGNETIK' },
    flares: { emoji: '☀️', hue: '#ff7b00', label: 'Flares / X-Ray', short: 'FLARES LADEN' },
    protons: { emoji: '⚛️', hue: '#a78bfa', label: 'Protonen', short: 'PROTONEN LADEN' },
    wind_imf: { emoji: '💨', hue: '#00f0ff', label: 'Sonnenwind / IMF', short: 'WIND / IMF' },
    aurora: { emoji: '🌌', hue: '#53f6b7', label: 'Aurora', short: 'AURORA SCOPE' },
    synthesis: { emoji: '🧠', hue: '#e8a07a', label: 'Finale Synthese', short: 'SYNTHESE' },
    idle: { emoji: '🛰️', hue: '#00f0ff', label: 'Pipeline', short: 'PIPELINE START' }
});

const THINKING_TEMPLATES = Object.freeze({
    geomag: ['Kp & G-Skala korrelieren…', 'Dst-Schwellen prüfen…', 'GPS / Netz / HF bewerten…'],
    flares: ['GOES X-Ray lesen…', 'R-Skala ableiten…', 'Flare-Peaks scannen…'],
    protons: ['Protonen ≥10 MeV…', 'S-Skala gewichten…', 'Satelliten-Risiko…'],
    wind_imf: ['Wind Speed/Density…', 'Bt / Bz koppeln…', 'IMF-Coupling…'],
    aurora: ['Hemispheric Power…', 'Min-Breite ableiten…', 'Mitteleuropa-Chance…'],
    synthesis: ['Domänen mergen…', 'Risikoklassen bauen…', 'Finalbericht…'],
    idle: ['Snapshot binden…', 'Pipeline vorbereiten…']
});

const DEFAULT_PIPELINE = Object.freeze([
    { id: 'geomag', title: 'Geomagnetik' },
    { id: 'flares', title: 'Flares / X-Ray' },
    { id: 'protons', title: 'Protonen' },
    { id: 'wind_imf', title: 'Sonnenwind / IMF' },
    { id: 'aurora', title: 'Aurora' },
    { id: 'synthesis', title: 'Finale Synthese' }
]);

function ensureAnalysisModal() {
    let overlay = document.getElementById('space-analysis-modal');
    if (overlay) return overlay;

    overlay = el('div', 'space-analysis-modal');
    overlay.id = 'space-analysis-modal';
    overlay.setAttribute('role', 'dialog');
    overlay.setAttribute('aria-modal', 'true');
    overlay.setAttribute('aria-labelledby', 'space-analysis-modal-title');
    overlay.hidden = true;

    const panel = el('section', 'space-analysis-panel space-analysis-panel-wide');

    // Header
    const header = el('header', 'space-analysis-modal-header');
    const titles = el('div');
    titles.append(
        el('div', 'space-eyebrow', 'MULTI-STEP PIPELINE · AETHEL CORE'),
        el('h2', 'space-analysis-modal-title', 'Weltraumwetter · KI-Lageanalyse')
    );
    titles.querySelector('h2').id = 'space-analysis-modal-title';
    const closeBtn = el('button', 'space-btn compact', 'SCHLIESSEN');
    closeBtn.type = 'button';
    closeBtn.id = 'space-analysis-close';
    closeBtn.addEventListener('click', closeAnalysisModal);
    header.append(titles, closeBtn);

    // Risk chips
    const riskRow = el('div', 'space-analysis-risk-row');
    riskRow.id = 'space-analysis-risk-row';

    // Tabs
    const tabs = el('div', 'space-analysis-tabs');
    tabs.setAttribute('role', 'tablist');
    const tabDefs = [
        { id: 'live', label: '● LIVE CORE' },
        { id: 'packages', label: 'PAKETE' },
        { id: 'report', label: 'FINALBERICHT' }
    ];
    for (const t of tabDefs) {
        const btn = el('button', 'space-analysis-tab' + (t.id === 'live' ? ' is-active' : ''), t.label);
        btn.type = 'button';
        btn.dataset.tab = t.id;
        btn.setAttribute('role', 'tab');
        btn.setAttribute('aria-selected', t.id === 'live' ? 'true' : 'false');
        btn.addEventListener('click', () => switchAnalysisTab(t.id));
        tabs.appendChild(btn);
    }

    // Tab panels
    const panels = el('div', 'space-analysis-tab-panels');

    // LIVE tab: left progress stack + center HUD ring (reference: loading mission UI)
    const livePanel = el('div', 'space-analysis-tab-panel is-active');
    livePanel.id = 'space-analysis-tab-live';
    livePanel.dataset.tabPanel = 'live';

    const liveGrid = el('div', 'space-analysis-live-grid space-analysis-live-hud');

    // LEFT: step list with progress bars
    const stepsCol = el('div', 'space-analysis-steps-col');
    stepsCol.append(el('div', 'space-metric-label', 'DATENPAKETE'));
    const stepRail = el('ol', 'space-analysis-step-rail space-analysis-step-stack');
    stepRail.id = 'space-analysis-steps';
    stepsCol.appendChild(stepRail);

    // CENTER: big orbit + title + %
    const orbitCol = el('div', 'space-analysis-orbit-col');
    const orbit = el('div', 'space-analysis-orbit');
    orbit.id = 'space-analysis-orbit';
    orbit.innerHTML = `
        <div class="space-orbit-glow"></div>
        <div class="space-orbit-ring space-orbit-ring-a"></div>
        <div class="space-orbit-ring space-orbit-ring-b"></div>
        <div class="space-orbit-ring space-orbit-ring-c"></div>
        <svg class="space-orbit-arc" viewBox="0 0 120 120" aria-hidden="true">
            <circle class="space-orbit-arc-track" cx="60" cy="60" r="52" />
            <circle class="space-orbit-arc-value" id="space-orbit-arc-value" cx="60" cy="60" r="52" />
        </svg>
        <div class="space-orbit-core">
            <span class="space-orbit-emoji" id="space-analysis-orbit-emoji">🛰️</span>
            <div class="space-orbit-label" id="space-analysis-orbit-label">PIPELINE START</div>
            <div class="space-orbit-pct" id="space-analysis-orbit-pct">0% Fortschritt</div>
        </div>
    `;
    const orbitMeta = el('div', 'space-orbit-meta');
    orbitMeta.id = 'space-analysis-orbit-meta';
    orbitMeta.textContent = 'Simulierter Fortschritt · KI im Hintergrund';
    const thinkLog = el('div', 'space-analysis-think-log space-analysis-think-compact');
    thinkLog.id = 'space-analysis-think-log';
    thinkLog.setAttribute('aria-live', 'polite');
    orbitCol.append(orbit, orbitMeta, thinkLog);

    liveGrid.append(stepsCol, orbitCol);
    livePanel.appendChild(liveGrid);

    // PACKAGES tab
    const pkgPanel = el('div', 'space-analysis-tab-panel');
    pkgPanel.id = 'space-analysis-tab-packages';
    pkgPanel.dataset.tabPanel = 'packages';
    pkgPanel.hidden = true;
    const pkgList = el('div', 'space-analysis-package-cards');
    pkgList.id = 'space-analysis-package-cards';
    pkgPanel.append(
        el('div', 'space-metric-label', 'ZWISCHENBERICHTE PRO DATENPAKET'),
        pkgList
    );

    // REPORT tab
    const reportPanel = el('div', 'space-analysis-tab-panel');
    reportPanel.id = 'space-analysis-tab-report';
    reportPanel.dataset.tabPanel = 'report';
    reportPanel.hidden = true;
    const finalWrap = el('div', 'space-analysis-final');
    finalWrap.id = 'space-analysis-final';
    finalWrap.append(el('div', 'space-metric-label', 'FINALER LAGEBERICHT'));
    const finalBody = el('article', 'space-analysis-final-body markdown-body');
    finalBody.id = 'space-analysis-final-body';
    finalBody.textContent = 'Noch kein Finalbericht — Pipeline läuft oder wartet.';
    finalWrap.appendChild(finalBody);
    reportPanel.appendChild(finalWrap);

    // Hidden intermediate host (kept for API compatibility / last package text)
    const intermediateBody = el('div');
    intermediateBody.id = 'space-analysis-intermediate-body';
    intermediateBody.hidden = true;

    panels.append(livePanel, pkgPanel, reportPanel, intermediateBody);

    const status = el('div', 'space-analysis-status');
    status.id = 'space-analysis-status';
    status.textContent = 'Bereit.';

    const footer = el('footer', 'space-analysis-modal-footer');
    const rerun = el('button', 'space-btn primary', 'ANALYSE NEU STARTEN');
    rerun.type = 'button';
    rerun.id = 'space-analysis-rerun';
    rerun.addEventListener('click', () => generateSpaceAnalysis({ force: true }));
    footer.append(status, rerun);

    panel.append(header, riskRow, tabs, panels, footer);
    overlay.appendChild(panel);
    overlay.addEventListener('click', (ev) => {
        if (ev.target === overlay) closeAnalysisModal();
    });
    document.body.appendChild(overlay);
    return overlay;
}

function switchAnalysisTab(tabId) {
    analysisActiveTab = tabId;
    document.querySelectorAll('.space-analysis-tab').forEach(btn => {
        const on = btn.dataset.tab === tabId;
        btn.classList.toggle('is-active', on);
        btn.setAttribute('aria-selected', on ? 'true' : 'false');
    });
    document.querySelectorAll('.space-analysis-tab-panel').forEach(panel => {
        const on = panel.dataset.tabPanel === tabId;
        panel.hidden = !on;
        panel.classList.toggle('is-active', on);
    });
}

function closeAnalysisModal() {
    if (analysisAbort) {
        try { analysisAbort.abort(); } catch (_) { /* ignore */ }
        analysisAbort = null;
    }
    stopThinkingSimulation();
    setOrbitVisual('idle', false);
    analysisRunning = false;
    const overlay = document.getElementById('space-analysis-modal');
    if (overlay) {
        overlay.hidden = true;
        overlay.classList.remove('is-open');
    }
}

function openAnalysisModal() {
    const overlay = ensureAnalysisModal();
    overlay.hidden = false;
    overlay.classList.add('is-open');
    switchAnalysisTab('live');
    return overlay;
}

function renderRiskChips(risks) {
    const row = document.getElementById('space-analysis-risk-row');
    if (!row || !risks) return;
    row.replaceChildren();
    const chips = [
        { label: 'GESAMT', value: `${risks.overall || '—'} · ${risks.overall_score ?? '—'}`, tone: riskTone(risks.overall) },
        { label: 'R', value: risks.radio_r || '—', tone: 'r' },
        { label: 'S', value: risks.radiation_s || '—', tone: 's' },
        { label: 'G', value: risks.geomag_g || '—', tone: 'g' },
        { label: 'IMF', value: risks.wind_imf || '—', tone: 'imf' },
        { label: 'AURORA', value: risks.aurora || '—', tone: 'aurora' }
    ];
    for (const c of chips) {
        const chip = el('div', `space-risk-chip tone-${c.tone}`);
        chip.append(el('span', 'space-risk-chip-label', c.label), el('strong', '', c.value));
        row.appendChild(chip);
    }
}

function riskTone(overall) {
    const o = String(overall || '').toUpperCase();
    if (o === 'EXTREME' || o === 'SEVERE') return 'g';
    if (o === 'HIGH') return 'r';
    if (o === 'ELEVATED') return 'imf';
    return 'ok';
}

function setOrbitVisual(packageId, spinning, pct) {
    const vis = PACKAGE_VISUALS[packageId] || PACKAGE_VISUALS.idle;
    const orbit = document.getElementById('space-analysis-orbit');
    const emoji = document.getElementById('space-analysis-orbit-emoji');
    const label = document.getElementById('space-analysis-orbit-label');
    const pctEl = document.getElementById('space-analysis-orbit-pct');
    if (emoji) emoji.textContent = vis.emoji;
    if (label) label.textContent = vis.short || vis.label;
    if (orbit) {
        orbit.style.setProperty('--orbit-hue', vis.hue);
        orbit.classList.toggle('is-spinning', !!spinning);
        orbit.classList.toggle('is-idle', !spinning);
    }
    if (typeof pct === 'number') {
        currentPackageProgress = Math.max(0, Math.min(100, pct));
        if (pctEl) pctEl.textContent = `${Math.round(currentPackageProgress)}% Fortschritt`;
        updateOrbitArc(currentPackageProgress);
    }
}

function updateOrbitArc(pct) {
    const arc = document.getElementById('space-orbit-arc-value');
    if (!arc) return;
    const r = 52;
    const c = 2 * Math.PI * r;
    const p = Math.max(0, Math.min(100, pct)) / 100;
    arc.style.strokeDasharray = `${c}`;
    arc.style.strokeDashoffset = `${c * (1 - p)}`;
}

function stopThinkingSimulation() {
    if (thinkingTimer) {
        clearInterval(thinkingTimer);
        thinkingTimer = null;
    }
    if (progressTimer) {
        clearInterval(progressTimer);
        progressTimer = null;
    }
}

function clearThinkingLog() {
    const log = document.getElementById('space-analysis-think-log');
    if (log) log.replaceChildren();
}

function appendThinkLine(text, kind = 'sim') {
    const log = document.getElementById('space-analysis-think-log');
    if (!log) return;
    const line = el('div', `space-think-line is-${kind}`);
    line.append(el('span', 'space-think-text', text));
    log.appendChild(line);
    log.scrollTop = log.scrollHeight;
    while (log.children.length > 6) log.removeChild(log.firstChild);
}

/**
 * Simulated thinking + smooth progress bar while waiting for SSE package results.
 * Real thinking is not required — UI always looks alive.
 */
function startThinkingSimulation(packageId) {
    stopThinkingSimulation();
    const templates = THINKING_TEMPLATES[packageId] || THINKING_TEMPLATES.idle;
    let i = 0;
    currentPackageProgress = packageId === 'idle' ? 4 : 8;
    setOrbitVisual(packageId, true, currentPackageProgress);
    setStepProgress(packageId, currentPackageProgress);
    appendThinkLine(templates[0] || 'Analysiere…', 'sim');

    thinkingTimer = setInterval(() => {
        i += 1;
        appendThinkLine(templates[i % templates.length], 'sim');
    }, 1400);

    // Smooth fake progress toward ~92% until real step_done arrives
    progressTimer = setInterval(() => {
        if (currentPackageProgress >= 92) return;
        const boost = currentPackageProgress < 40 ? 3.2 : currentPackageProgress < 70 ? 1.6 : 0.7;
        currentPackageProgress = Math.min(92, currentPackageProgress + boost);
        setOrbitVisual(packageId, true, currentPackageProgress);
        setStepProgress(packageId, currentPackageProgress);
    }, 280);
}

function setStepProgress(id, pct) {
    const li = document.querySelector(`.space-analysis-step[data-step-id="${CSS.escape(id)}"]`);
    if (!li) return;
    const fill = li.querySelector('.space-step-progress-fill');
    const label = li.querySelector('.space-step-progress-label');
    if (fill) fill.style.width = `${Math.max(0, Math.min(100, pct))}%`;
    if (label) label.textContent = `${Math.round(pct)}%`;
    const prev = packageReports.get(id) || { id };
    packageReports.set(id, { ...prev, progress: pct });
}

function overallPipelinePercent() {
    const order = pipelineOrder.length ? pipelineOrder : DEFAULT_PIPELINE.map(p => p.id);
    if (!order.length) return 0;
    let sum = 0;
    for (const id of order) {
        const st = packageReports.get(id)?.status;
        if (st === 'done') sum += 100;
        else if (st === 'running') sum += currentPackageProgress;
        else sum += 0;
    }
    return Math.round(sum / order.length);
}

function seedStepList(packages) {
    const list = document.getElementById('space-analysis-steps');
    if (!list) return;
    list.replaceChildren();
    const all = (packages && packages.length)
        ? packages.map(p => ({ id: p.id, title: PACKAGE_VISUALS[p.id]?.label || p.title || p.id, risk_label: p.risk_label }))
        : DEFAULT_PIPELINE.map(p => ({ ...p }));
    if (!all.some(p => p.id === 'synthesis')) {
        all.push({ id: 'synthesis', title: 'Finale Synthese', risk_label: '…' });
    }
    pipelineOrder = all.map(p => p.id);
    for (const pkg of all) {
        const vis = PACKAGE_VISUALS[pkg.id] || PACKAGE_VISUALS.idle;
        const prev = packageReports.get(pkg.id);
        const status = prev?.status || 'pending';
        const li = el('li', `space-analysis-step is-${status}`);
        li.dataset.stepId = pkg.id;
        li.style.setProperty('--step-hue', vis.hue);

        const head = el('div', 'space-analysis-step-head');
        const icon = el('span', 'space-analysis-step-icon');
        icon.textContent = vis.emoji;
        icon.style.setProperty('--step-hue', vis.hue);
        head.append(
            icon,
            el('span', 'space-analysis-step-title', vis.label || pkg.title || pkg.id)
        );

        const bar = el('div', 'space-step-progress');
        const fill = el('div', 'space-step-progress-fill');
        fill.className = 'space-step-progress-fill';
        const p = status === 'done' ? 100 : (prev?.progress || 0);
        fill.style.width = `${p}%`;
        bar.appendChild(fill);
        const foot = el('div', 'space-step-progress-foot');
        foot.append(
            el('span', 'space-step-progress-label', status === 'done' ? '100%' : `${Math.round(p)}%`),
            el('span', 'space-analysis-step-risk', pkg.risk_label || prev?.risk || '')
        );
        li.append(head, bar, foot);
        list.appendChild(li);

        if (!packageReports.has(pkg.id)) {
            packageReports.set(pkg.id, {
                id: pkg.id,
                title: vis.label || pkg.title,
                risk: pkg.risk_label || '',
                body: '',
                source: '',
                progress: p,
                status
            });
        }
    }
    renderPackageCards();
}

function setStepState(id, stateName, bodyText) {
    const li = document.querySelector(`.space-analysis-step[data-step-id="${CSS.escape(id)}"]`);
    if (li) {
        li.classList.remove('is-pending', 'is-running', 'is-done', 'is-error');
        li.classList.add(`is-${stateName}`);
    }
    const prev = packageReports.get(id) || { id, title: PACKAGE_VISUALS[id]?.label || id };
    const progress = stateName === 'done' ? 100 : stateName === 'running' ? Math.max(prev.progress || 0, currentPackageProgress) : (prev.progress || 0);
    packageReports.set(id, {
        ...prev,
        status: stateName,
        progress,
        body: bodyText != null ? bodyText : prev.body
    });
    setStepProgress(id, progress);
    if (stateName === 'done') {
        setStepProgress(id, 100);
        if (li) {
            const fill = li.querySelector('.space-step-progress-fill');
            if (fill) fill.style.width = '100%';
        }
    }
    renderPackageCards();
}

function renderPackageCards() {
    const host = document.getElementById('space-analysis-package-cards');
    if (!host) return;
    host.replaceChildren();
    if (packageReports.size === 0) {
        host.appendChild(el('div', 'space-panel-note', 'Noch keine abgeschlossenen Pakete.'));
        return;
    }
    for (const [id, rep] of packageReports) {
        if (id === 'synthesis' && !rep.body) continue;
        const vis = PACKAGE_VISUALS[id] || PACKAGE_VISUALS.idle;
        const card = el('article', 'space-package-card');
        const head = el('header', 'space-package-card-head');
        head.append(
            el('span', 'space-package-card-emoji', vis.emoji),
            el('div', 'space-package-card-title', rep.title || id),
            el('span', 'space-package-card-risk', rep.risk || '')
        );
        const body = el('div', 'space-package-card-body markdown-body');
        try {
            body.innerHTML = formatMarkdown(String(rep.body || '').slice(0, 4000));
        } catch (_) {
            body.textContent = String(rep.body || '');
        }
        const src = el('div', 'space-package-card-src', rep.source ? `Quelle: ${rep.source}` : '');
        card.append(head, body, src);
        host.appendChild(card);
    }
}

function setAnalysisStatus(text, isError) {
    const elStatus = document.getElementById('space-analysis-status');
    if (!elStatus) return;
    elStatus.textContent = text;
    elStatus.classList.toggle('is-error', !!isError);
}

function setIntermediateBody(markdown) {
    const body = document.getElementById('space-analysis-intermediate-body');
    if (body) {
        try {
            body.innerHTML = formatMarkdown(markdown || '');
        } catch (_) {
            body.textContent = markdown || '';
        }
    }
}

function setFinalReport(markdown, risks) {
    const body = document.getElementById('space-analysis-final-body');
    if (risks) renderRiskChips(risks);
    if (body) {
        try {
            body.innerHTML = formatMarkdown(markdown || '');
        } catch (_) {
            body.textContent = markdown || '';
        }
    }
    const hint = document.getElementById('space-analysis-result');
    if (hint) hint.textContent = 'Analyse abgeschlossen — Finalbericht im Tab „FINALBERICHT“.';
    switchAnalysisTab('report');
    setOrbitVisual('synthesis', false, 100);
    const meta = document.getElementById('space-analysis-orbit-meta');
    if (meta) meta.textContent = `Pipeline 100% · ${overallPipelinePercent()}% Domänen`;
}

/**
 * Primary analysis entry: opens modal on Space view, runs multi-package pipeline via /v1/space/analysis.
 * Does NOT navigate to Chat.
 */
export async function generateSpaceAnalysis(opts = {}) {
    if (analysisRunning && !opts.force) return;
    if (!lastWeatherPayload) {
        openAnalysisModal();
        setAnalysisStatus('Kein Weltraumwetter-Snapshot. Bitte zuerst SYNC TELEMETRIE.', true);
        seedStepList([]);
        return;
    }

    openAnalysisModal();
    analysisRunning = true;
    packageReports.clear();
    clearThinkingLog();
    currentPackageProgress = 0;
    if (analysisAbort) {
        try { analysisAbort.abort(); } catch (_) { /* ignore */ }
    }
    analysisAbort = new AbortController();

    setAnalysisStatus('Starte Multi-Step-Analyse…');
    seedStepList(DEFAULT_PIPELINE.map(p => ({ ...p })));
    // Jump straight into first domain visual (no endless idle loop)
    setOrbitVisual('geomag', true, 5);
    startThinkingSimulation('geomag');
    setStepState('geomag', 'running');
    renderPackageCards();
    switchAnalysisTab('live');

    const startBtn = document.getElementById('btn-generate-space-analysis');
    if (startBtn) {
        startBtn.disabled = true;
        startBtn.textContent = 'ANALYSE LÄUFT…';
    }

    try {
        const res = await fetch(`${state.API_BASE}/v1/space/analysis`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            signal: analysisAbort.signal,
            body: JSON.stringify({
                model_id: state.currentModel,
                snapshot: lastWeatherPayload,
                language: localStorage.getItem('aethel_ui_language') || 'de'
            })
        });
        if (!res.ok) {
            throw new Error((await res.text()).slice(0, 240) || `HTTP ${res.status}`);
        }
        if (!res.body) throw new Error('Kein Analyse-Stream empfangen.');

        const reader = res.body.getReader();
        const decoder = new TextDecoder('utf-8');
        let buffer = '';

        while (true) {
            const { value, done } = await reader.read();
            if (done) break;
            buffer += decoder.decode(value, { stream: true });
            const frames = buffer.split('\n\n');
            buffer = frames.pop() || '';
            for (const frame of frames) {
                for (const line of frame.split('\n')) {
                    if (!line.startsWith('data:')) continue;
                    const payload = line.slice(5).trimStart();
                    if (payload === '[DONE]') continue;
                    // Real thinking deltas from model if ever present
                    if (payload.startsWith('[[THINKING]]:')) {
                        appendThinkLine(payload.slice(13).trim(), 'real');
                        continue;
                    }
                    let evt;
                    try {
                        evt = JSON.parse(payload);
                    } catch (_) {
                        continue;
                    }
                    handleAnalysisEvent(evt);
                }
            }
        }
        stopThinkingSimulation();
        setOrbitVisual('synthesis', false, 100);
        setAnalysisStatus(`Pipeline abgeschlossen · Gesamt ${overallPipelinePercent()}%`);
        appendThinkLine('Alle Pakete verarbeitet · Finalbericht verfügbar.', 'ok');
    } catch (err) {
        stopThinkingSimulation();
        setOrbitVisual('idle', false, currentPackageProgress);
        if (err?.name === 'AbortError') {
            setAnalysisStatus('Analyse abgebrochen.', true);
        } else {
            setAnalysisStatus(`Analyse fehlgeschlagen: ${err?.message || err}`, true);
            appendThinkLine(`Fehler: ${err?.message || err}`, 'err');
        }
    } finally {
        analysisRunning = false;
        analysisAbort = null;
        if (startBtn) {
            startBtn.disabled = false;
            startBtn.textContent = 'ASTROPHYSIK-ANALYSE STARTEN ›';
        }
    }
}

function handleAnalysisEvent(evt) {
    const type = evt?.type;
    const data = evt?.data;
    if (type === 'pipeline') {
        // Preserve running/done progress when re-seeding labels/risks from server
        const saved = new Map(packageReports);
        if (Array.isArray(data?.packages)) {
            seedStepList(data.packages);
            for (const [id, rep] of saved) {
                if (rep.status === 'done' || rep.status === 'running') {
                    packageReports.set(id, { ...packageReports.get(id), ...rep });
                    setStepState(id, rep.status, rep.body);
                }
            }
        }
        if (data?.risk_classes) renderRiskChips(data.risk_classes);
        setAnalysisStatus('Domänen-Queue aktiv — Fortschritt simuliert bis KI antwortet…');
        appendThinkLine('Queue geladen · warte auf Domänen-Engines…', 'ok');
        // Keep current package animation if already running
        if (!progressTimer) startThinkingSimulation('geomag');
        return;
    }
    if (type === 'step_start') {
        const id = data?.id || 'geomag';
        // Mark previous running packages done only if server advanced
        setStepState(id, 'running');
        startThinkingSimulation(id);
        setAnalysisStatus(`Analysiere: ${PACKAGE_VISUALS[id]?.label || data?.title || id}`);
        const meta = document.getElementById('space-analysis-orbit-meta');
        if (meta) meta.textContent = data?.focus || PACKAGE_VISUALS[id]?.label || id;
        const prev = packageReports.get(id) || {};
        packageReports.set(id, {
            ...prev,
            id,
            title: PACKAGE_VISUALS[id]?.label || data?.title || id,
            risk: data?.risk_label || prev.risk || '',
            status: 'running'
        });
        if (data?.risk_label) {
            const riskEl = document.querySelector(`.space-analysis-step[data-step-id="${CSS.escape(id)}"] .space-analysis-step-risk`);
            if (riskEl) riskEl.textContent = data.risk_label;
        }
        return;
    }
    if (type === 'step_done') {
        const id = data?.package_id || data?.id;
        stopThinkingSimulation();
        currentPackageProgress = 100;
        setOrbitVisual(id, false, 100);
        setStepState(id, 'done', data?.body);
        if (data?.risk_label) {
            const riskEl = document.querySelector(`.space-analysis-step[data-step-id="${CSS.escape(id)}"] .space-analysis-step-risk`);
            if (riskEl) riskEl.textContent = data.risk_label;
        }
        packageReports.set(id, {
            ...(packageReports.get(id) || {}),
            id,
            title: PACKAGE_VISUALS[id]?.label || data?.title || id,
            risk: data?.risk_label || '',
            body: data?.body || '',
            source: data?.source || '',
            progress: 100,
            status: 'done'
        });
        renderPackageCards();
        appendThinkLine(`✓ ${PACKAGE_VISUALS[id]?.label || id} fertig`, 'ok');
        setAnalysisStatus(`Paket fertig: ${PACKAGE_VISUALS[id]?.label || id} · Gesamt ${overallPipelinePercent()}%`);
        if (data?.body) setIntermediateBody(data.body);
        return;
    }
    if (type === 'final') {
        stopThinkingSimulation();
        setOrbitVisual('synthesis', false, 100);
        setStepState('synthesis', 'done', data?.report);
        setFinalReport(data?.report || '', data?.risk_classes);
        setAnalysisStatus('Finalbericht bereit.');
        appendThinkLine('Synthese fertig · Tab FINALBERICHT', 'ok');
    }
}

function renderSpaceDashboardUI(data, opts = {}) {
    const forceImages = !!opts.forceImages;
    const container = document.getElementById('space-dashboard-content');
    if (!container) return;
    container.replaceChildren();
    container.classList.add('space-dashboard-root');

    // Header (Solarcommander-style mission bar)
    const header = el('div', 'space-mission-header');
    const left = el('div');
    const live = el('div', 'space-live-line');
    live.append(el('span', 'space-live-dot'), el('span', '', 'VGT SOLAR COMMAND // MISSION CONTROL'));
    left.append(
        live,
        el('h3', '', 'SOLAR COMMAND · ECHTZEIT TELEMETRIE'),
        el('div', 'space-data-age', 'Telemetrie: — · SDO: —')
    );
    left.querySelector('.space-data-age').id = 'space-data-age';

    const right = el('div', 'space-mission-actions');
    const utcBox = el('div', 'space-utc-box');
    utcBox.append(el('div', 'space-metric-label', 'SYSTEMZEIT UTC'), el('div', 'space-utc-clock', '—'));
    utcBox.querySelector('.space-utc-clock').id = 'space-utc-clock';
    const syncMeta = el('div', 'space-sync-meta');
    syncMeta.append(el('div', 'space-metric-label', 'LETZTES UPDATE'), el('div', '', '—'));
    syncMeta.lastChild.id = 'space-last-sync-time';

    const badge = el('span', 'space-chip online', 'MANUELLER SYNC · KEIN AUTO-POLL');
    badge.id = 'space-sync-badge';
    const refresh = el('button', 'space-btn', 'SYNC TELEMETRIE');
    refresh.type = 'button';
    refresh.id = 'btn-refresh-space';
    refresh.addEventListener('click', () => fetchSpaceWeatherData({ forceImages: true }));
    right.append(utcBox, syncMeta, badge, refresh);
    header.append(left, right);

    const staleBanner = el('div', 'space-stale-banner');
    staleBanner.id = 'space-stale-banner';
    staleBanner.hidden = true;
    const cooldown = el('div', 'space-cooldown');
    cooldown.id = 'space-cooldown';
    cooldown.hidden = true;

    // Solarcommander 12-col: left visuals | center scales+charts+forecast | right data cards
    const grid = el('div', 'space-mission-grid space-mission-grid-sc');
    const colLeft = el('div', 'space-col space-col-left');
    colLeft.append(buildImager(data, forceImages), buildAuroraMap(data, forceImages));

    const colCenter = el('div', 'space-col space-col-center');
    colCenter.append(buildScales(data), buildCharts(data), buildForecast(data));

    const colRight = el('div', 'space-col space-col-right');
    colRight.append(buildKpCard(data), buildDstCard(data), buildWindMag(data), buildFlareHistory(data));

    grid.append(colLeft, colCenter, colRight);
    container.append(header, staleBanner, cooldown, grid, buildAnalysisConsole());
    updateAgeUI();
    const utc = document.getElementById('space-utc-clock');
    if (utc) utc.textContent = new Date().toISOString().slice(11, 19);
}

export function initSpaceDashboard() {
    fetchSpaceWeatherData({ forceImages: true });
    startAgeTicker();
}
