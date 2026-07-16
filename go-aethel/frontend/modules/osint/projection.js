import * as GlobeProjection from '../global_watch_projection.js';

let sharedGlobeProjection = null;

export function globeRadius(scale, cw, ch) {
  return GlobeProjection.globeRadius(scale, cw, ch);
}

export function getGlobeProjection(rotYRad, rotX, scale, cw, ch) {
  if (
    sharedGlobeProjection
    && sharedGlobeProjection.rotationY === rotYRad
    && sharedGlobeProjection.rotationX === rotX
    && sharedGlobeProjection.scale === scale
    && sharedGlobeProjection.width === cw
    && sharedGlobeProjection.height === ch
  ) {
    return sharedGlobeProjection;
  }
  sharedGlobeProjection = GlobeProjection.createProjection(rotYRad, rotX, scale, cw, ch);
  return sharedGlobeProjection;
}

export function projectLatLon(lat, lon, rotYRad, rotX, scale, cw, ch) {
  const point = GlobeProjection.project(
    getGlobeProjection(rotYRad, rotX, scale, cw, ch),
    lat,
    lon,
  );
  return { x: point.x, y: point.y, visible: point.visible, z2: point.depth };
}

export function unprojectScreenToLatLon(px, py, rotYRad, rotX, scale, cw, ch) {
  const point = GlobeProjection.unproject(
    getGlobeProjection(rotYRad, rotX, scale, cw, ch),
    px,
    py,
  );
  if (!point) return null;
  return { lat: point.lat, lon: point.lon, z2: point.depth, u: point.u, v: point.v };
}

export function projectUnprojectRoundTrip(lat, lon, rotY, rotX, scale, cw, ch) {
  const p = projectLatLon(lat, lon, rotY, rotX, scale, cw, ch);
  if (!p.visible) return { ok: false, reason: 'backface' };
  const back = unprojectScreenToLatLon(p.x - 0.5, p.y - 0.5, rotY, rotX, scale, cw, ch);
  if (!back) return { ok: false, reason: 'outside' };
  return {
    ok: true,
    latErr: Math.abs(back.lat - lat),
    lonErr: Math.min(Math.abs(back.lon - lon), 360 - Math.abs(back.lon - lon)),
    backLat: back.lat,
    backLon: back.lon,
  };
}

export function applyDomainFilter(events, domain) {
  if (!domain || domain === 'all') return events || [];
  return (events || []).filter(ev => ev.domain === domain);
}

export function hitTestPin(mx, my, pins, scale) {
  const s = scale || 1;
  for (let i = 0; i < (pins || []).length; i++) {
    const p = pins[i];
    const dx = p.x - mx;
    const dy = p.y - my;
    const hitRadius = 18 * s;
    if (dx * dx + dy * dy < hitRadius * hitRadius) {
      return p.idx;
    }
  }
  return -1;
}

export function applyDragDelta(rotY, dx) {
  return rotY + dx * 0.0055;
}

export function applyWheelZoom(scale, deltaY) {
  return Math.max(0.55, Math.min(2.6, scale - deltaY * 0.0014));
}

export function resetGlobeView() {
  const f = computeGlobeFocusRotation(10.5, 50);
  return { rotY: f.rotY, rotX: f.rotX, scale: 1.15, selectedIndex: -1 };
}

export function computeGlobeFocusRotation(lon, lat) {
  return GlobeProjection.focusRotation(lon, lat);
}

export const REGION_FOCUS_TABLE = {
  global: { lon: 10, lat: 20, scale: 1.05, label: 'Global' },
  europe: { lon: 10, lat: 50, scale: 1.55, label: 'Europe' },
  germany: { lon: 10.5, lat: 51.2, scale: 1.9, label: 'Germany' },
  mena: { lon: 35, lat: 28, scale: 1.4, label: 'MENA' },
  asia: { lon: 105, lat: 35, scale: 1.25, label: 'Asia' },
  americas: { lon: -95, lat: 30, scale: 1.2, label: 'Americas' },
  oceania: { lon: 145, lat: -25, scale: 1.35, label: 'Oceania' },
  africa: { lon: 20, lat: 5, scale: 1.25, label: 'Africa' },
};

export const REGION_GEO_BOUNDS = Object.freeze({
  germany: { minLat: 47.2, maxLat: 55.2, minLon: 5.5, maxLon: 15.8, terms: ['deutschland', 'germany', 'berlin', 'bundestag'] },
  europe: { minLat: 34, maxLat: 72, minLon: -25, maxLon: 45, terms: ['europa', 'europe', 'eu', 'european'] },
  mena: { minLat: 12, maxLat: 42, minLon: -18, maxLon: 65, terms: ['mena', 'middle east', 'nahost'] },
  asia: { minLat: -10, maxLat: 78, minLon: 45, maxLon: 180, terms: ['asia', 'asien'] },
  americas: { minLat: -60, maxLat: 82, minLon: -170, maxLon: -30, terms: ['america', 'amerika', 'usa'] },
  oceania: { minLat: -50, maxLat: 5, minLon: 105, maxLon: 180, terms: ['oceania', 'ozeanien', 'australia'] },
  africa: { minLat: -36, maxLat: 38, minLon: -20, maxLon: 55, terms: ['africa', 'afrika'] },
});

export function eventMatchesRegion(event, region) {
  const key = String(region || 'global').toLowerCase();
  if (key === 'global') return true;
  const bounds = REGION_GEO_BOUNDS[key];
  if (!bounds) return true;
  const lat = Number(event?.lat ?? event?.latitude);
  const lon = Number(event?.lon ?? event?.longitude);
  if (Number.isFinite(lat) && Number.isFinite(lon) && lat >= bounds.minLat && lat <= bounds.maxLat && lon >= bounds.minLon && lon <= bounds.maxLon) return true;
  const text = `${event?.title || ''} ${event?.summary || ''} ${event?.source || ''}`.toLowerCase();
  return bounds.terms.some(term => text.includes(term));
}

export function filterEventsByRegion(events, region) {
  return (events || []).filter(event => eventMatchesRegion(event, region));
}

export function isGeoEvent(ev) {
  return !!(ev && ev.has_geo === true);
}

export function safeExternalURL(value) {
    try {
        const url = new URL(String(value || ""));
        if (url.protocol !== "https:" || url.username || url.password) return null;
        return url;
    } catch (_) {
        return null;
    }
}

export function appendTextElement(parent, tagName, text, className = "") {
    const element = document.createElement(tagName);
    if (className) element.className = className;
    element.textContent = String(text ?? "");
    parent.appendChild(element);
    return element;
}

export function clampGwPanelPosition(left, top, panelW, panelH, viewW, viewH) {
    const maxL = Math.max(0, (viewW || 0) - Math.min(panelW || 0, viewW || 0) * 0.15);
    const maxT = Math.max(0, (viewH || 0) - 40);
    const minL = -Math.max(0, (panelW || 0) - 80);
    return {
        left: Math.max(minL, Math.min(left, maxL)),
        top: Math.max(0, Math.min(top, maxT)),
    };
}

/** Pure event timestamp helper (same logic as feed UI). */
export function eventTimestampMs(ev) {
    if (!ev) return 0;
    const raw = ev.timestamp || ev.observed_at || ev.time || ev.created_at;
    if (raw == null || raw === '') return 0;
    if (typeof raw === 'number') return raw < 1e12 ? raw * 1000 : raw;
    const t = Date.parse(raw);
    return Number.isFinite(t) ? t : 0;
}

/** Pure filter used by feed cards + globe pins (same window). */
export function filterEventsByTimeWindow(events, hours) {
    const list = events || [];
    const h = Number(hours);
    if (!h || h <= 0) return list.slice();
    const cutoff = Date.now() - h * 3600 * 1000;
    return list.filter(ev => {
        const ms = eventTimestampMs(ev);
        if (!ms) return false;
        return ms >= cutoff && ms <= Date.now() + 10 * 60 * 1000;
    });
}

/**
 * Build visible globe pins from events.
 * Hours window: prefers window.__gwTimeWindowHours / #gw-time-window (same as getGwTimeWindowHours).
 */
export function buildGlobePins(events, rotY, rotX, scale, cw, ch) {
  const pins = [];
  const el = (typeof document !== 'undefined') ? document.getElementById('gw-time-window') : null;
  const hoursRaw = el
    ? Number(el.value)
    : (typeof window !== 'undefined' && window.__gwTimeWindowHours != null
        ? Number(window.__gwTimeWindowHours)
        : 24);
  const hours = Number.isFinite(hoursRaw) ? hoursRaw : 24;
  const timed = hours > 0
    ? filterEventsByTimeWindow(events, hours)
    : (events || []);
  const region = (typeof window !== 'undefined' && window.__gwRegionFilter) ? window.__gwRegionFilter : 'global';
  const source = filterEventsByRegion(timed, region);
  source.forEach((ev) => {
    const idx = (events || []).indexOf(ev);
    if (!isGeoEvent(ev)) return;
    if (hours > 0) {
      const ms = eventTimestampMs(ev);
      if (ms && ms < Date.now() - hours * 3600 * 1000) return;
    }
    const lat = ev.lat != null ? ev.lat : 0;
    const lon = ev.lon != null ? ev.lon : 0;
    const p = projectLatLon(lat, lon, rotY, rotX, scale, cw, ch);
    if (p.visible) {
      pins.push({ idx, x: p.x, y: p.y, domain: ev.domain, title: ev.title, has_geo: true });
    }
  });
  return pins;
}
