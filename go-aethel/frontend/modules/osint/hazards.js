// ─── Hazard classification (pure — unit-tested via goja) ─────────────────────

export function magnitudeColorBand(mag) {
  const m = Number(mag);
  if (!Number.isFinite(m)) return 'unknown';
  if (m <= 2.5) return 'green';
  if (m <= 4.5) return 'orange';
  return 'red';
}

export function magnitudeBandColor(mag) {
  const band = magnitudeColorBand(mag);
  if (band === 'green') return { fill: 'rgba(52,211,153,0.55)', stroke: 'rgba(52,211,153,0.95)', hex: '#34d399', band };
  if (band === 'orange') return { fill: 'rgba(251,146,60,0.55)', stroke: 'rgba(251,146,60,0.95)', hex: '#fb923c', band };
  if (band === 'red') return { fill: 'rgba(244,63,94,0.6)', stroke: 'rgba(244,63,94,0.98)', hex: '#f43f5e', band };
  return { fill: 'rgba(148,163,184,0.45)', stroke: 'rgba(148,163,184,0.9)', hex: '#94a3b8', band: 'unknown' };
}

export function volcanoMarkerColor(isErupting) {
  if (isErupting) return { fill: 'rgba(244,63,94,0.75)', stroke: 'rgba(244,63,94,1)', hex: '#f43f5e', band: 'red' };
  return { fill: 'rgba(251,146,60,0.45)', stroke: 'rgba(251,146,60,0.9)', hex: '#fb923c', band: 'orange' };
}

export function parseMagnitudeFromEvent(ev) {
  if (!ev) return null;
  if (ev.magnitude != null && Number.isFinite(Number(ev.magnitude))) return Number(ev.magnitude);
  if (ev.mag != null && Number.isFinite(Number(ev.mag))) return Number(ev.mag);
  const blob = `${ev.title || ''} ${ev.summary || ''} ${ev.raw_text || ''} ${ev.RawText || ''}`;
  let m = blob.match(/\bM\s*([0-9]+(?:\.[0-9]+)?)\b/i);
  if (m) return parseFloat(m[1]);
  m = blob.match(/magnitude\s*([0-9]+(?:\.[0-9]+)?)/i);
  if (m) return parseFloat(m[1]);
  return null;
}

export function isEarthquakeEvent(ev) {
  if (!ev) return false;
  if (ev.hazard_type === 'earthquake' || ev.event_type === 'earthquake') return true;
  const src = String(ev.source || ev.source_id || ev.SourceID || '').toLowerCase();
  if (src.includes('usgs') || src.includes('earthquake')) return true;
  const id = String(ev.id || '').toLowerCase();
  if (id.startsWith('usgs') || id.includes('earthquake')) return true;
  const blob = `${ev.title || ''} ${ev.summary || ''}`.toLowerCase();
  if (blob.includes('[earthquake]') || blob.includes('earthquake') || /\bm\s*[0-9]+\.[0-9]+\s*-/.test(blob)) return true;
  return parseMagnitudeFromEvent(ev) != null && (blob.includes('quake') || src.includes('usgs'));
}

export function isVolcanoEvent(ev) {
  if (!ev) return false;
  if (ev.hazard_type === 'volcano' || ev.event_type === 'volcano') return true;
  const src = String(ev.source || ev.source_id || '').toLowerCase();
  if (src.includes('volcano') || src.includes('smithsonian')) return true;
  const blob = `${ev.title || ''} ${ev.summary || ''} ${ev.raw_text || ''}`.toLowerCase();
  if (blob.includes('[volcano') || blob.includes('volcano') || blob.includes('eruption') || blob.includes('volcanic')) return true;
  if (src.includes('eonet') && (blob.includes('volcano') || blob.includes('volcanic'))) return true;
  return false;
}

export function isEruptingVolcano(ev) {
  if (!isVolcanoEvent(ev)) return false;
  if (ev.erupting === true || ev.status === 'erupting') return true;
  const blob = `${ev.title || ''} ${ev.summary || ''} ${ev.raw_text || ''}`.toLowerCase();
  if (blob.includes('[volcano erupting]') || blob.includes('erupting') || blob.includes('eruption')) return true;
  if (blob.includes('volcano') || blob.includes('volcanic')) return true;
  return true;
}

export function formatArticleDateTime(raw) {
  if (raw == null || raw === '') {
    return { date: '—', time: '—', iso: '', display: '—' };
  }
  let d;
  if (raw instanceof Date) d = raw;
  else if (typeof raw === 'number') d = new Date(raw < 1e12 ? raw * 1000 : raw);
  else d = new Date(raw);
  if (Number.isNaN(d.getTime())) {
    return { date: String(raw), time: '—', iso: '', display: String(raw) };
  }
  const date = d.toLocaleDateString('de-DE', { year: 'numeric', month: '2-digit', day: '2-digit' });
  const time = d.toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  return { date, time, iso: d.toISOString(), display: `${date} ${time}` };
}

export function escapeHtml(unsafe) {
    if (!unsafe) return "";
    return unsafe
         .replace(/&/g, "&amp;")
         .replace(/</g, "&lt;")
         .replace(/>/g, "&gt;")
         .replace(/"/g, "&quot;")
         .replace(/'/g, "&#039;");
}

export function formatTime(isoString) {
    try {
        const d = new Date(isoString);
        return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) + " Uhr";
    } catch (_) {
        return "";
    }
}

export function freshness(isoString) {
    try {
        const ageMs = Date.now() - new Date(isoString).getTime();
        const mins = Math.floor(ageMs / 60000);
        if (mins < 1) return 'just now';
        if (mins < 60) return mins + 'm ago';
        return Math.floor(mins/60) + 'h ago';
    } catch (_) { return ''; }
}

/** Pure epistemic layer label for RAW / INFERENCE / VERIFIED UI. */
export function epistemicLayer(ev) {
    const s = String(ev && (ev.status || ev.layer) || '').toLowerCase();
    if (s === 'verified') return 'verified';
    if (s === 'corroborated') return 'inference';
    if (s === 'unverified' || s === 'proposed' || s === 'inference' || (ev && ev.provenance === 'unified-shared-model')) return 'inference';
    if (s === 'raw' || s === '') return 'raw';
    if (s === 'disputed' || s === 'rejected' || s === 'hypothesis') return 'inference';
    return 'raw';
}
