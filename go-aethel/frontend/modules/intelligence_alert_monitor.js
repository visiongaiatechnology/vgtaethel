import { state } from './state.js';

let stream = null;
const deliveredAlerts = new Set();

function createAlertToast(alert) {
    const host = document.body;
    if (!host || !alert?.event_id || deliveredAlerts.has(alert.event_id)) return;
    deliveredAlerts.add(alert.event_id);
    if (deliveredAlerts.size > 128) deliveredAlerts.clear();

    const toast = document.createElement('button');
    toast.type = 'button';
    toast.className = 'aethel-intelligence-alert';
    toast.style.cssText = [
        'position:fixed', 'right:24px', 'bottom:24px', 'z-index:10050', 'max-width:360px',
        'padding:14px 16px', 'text-align:left', 'background:rgba(20,8,18,.97)',
        'border:1px solid #ff4d79', 'border-radius:8px', 'box-shadow:0 0 28px rgba(255,77,121,.28)',
        'color:#f4f8ff', 'font-family:var(--font-mono,monospace)', 'cursor:pointer'
    ].join(';');
    const label = document.createElement('div');
    label.textContent = `GLOBAL WATCH ALERT · ${String(alert.severity || 'medium').toUpperCase()} · ${Number(alert.confidence || 0)}%`;
    label.style.cssText = 'color:#ff7a9e;font-size:10px;letter-spacing:.08em;margin-bottom:7px;font-weight:700;';
    const title = document.createElement('div');
    title.textContent = String(alert.title || 'New intelligence threshold alert');
    title.style.cssText = 'font-size:12px;line-height:1.45;';
    const hint = document.createElement('div');
    hint.textContent = 'Click to inspect in Global Watch. Observation remains proposed until reviewed.';
    hint.style.cssText = 'margin-top:8px;color:#aebbd0;font-size:9px;line-height:1.4;';
    toast.append(label, title, hint);
    toast.addEventListener('click', () => {
        document.getElementById('nav-btn-global-watch')?.click();
        toast.remove();
    });
    host.appendChild(toast);
    window.setTimeout(() => toast.remove(), 18000);
}

export function startGlobalIntelligenceAlertMonitor() {
    if (stream || typeof EventSource === 'undefined') return;
    stream = new EventSource(`${state.API_BASE}/v1/intelligence/stream`);
    stream.addEventListener('intelligence', event => {
        try {
            const payload = JSON.parse(event.data);
            if (payload?.type === 'global_watch.alert') createAlertToast(payload.alert);
        } catch (_) {
            // Malformed local stream frames are ignored; EventSource reconnects.
        }
    });
    stream.onerror = () => {
        // EventSource retries the local core stream automatically.
    };
}
