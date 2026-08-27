// STATUS: DIAMANT VGT SUPREME
import { state } from './state.js';
import { speak } from './voice.js';

let seenAlertIDs = new Set();
let unconfirmedAlertTimers = new Map();
let isMonitoring = false;

function playEmergencyAlarmSound(isExistential = false) {
    try {
        const AudioCtx = window.AudioContext || window.webkitAudioContext;
        if (!AudioCtx) return;
        const ctx = new AudioCtx();
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();

        osc.type = isExistential ? 'sawtooth' : 'sine';
        osc.frequency.setValueAtTime(isExistential ? 880 : 587.33, ctx.currentTime);
        osc.frequency.exponentialRampToValueAtTime(isExistential ? 440 : 880, ctx.currentTime + 0.6);

        gain.gain.setValueAtTime(0.3, ctx.currentTime);
        gain.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 0.6);

        osc.connect(gain);
        gain.connect(ctx.destination);

        osc.start();
        osc.stop(ctx.currentTime + 0.6);
    } catch (e) {
        console.warn('Audio alarm chime unavailable:', e);
    }
}

export function showEmergencyOverlay(alert) {
    let overlay = document.getElementById("emergency-alert-modal");
    if (!overlay) {
        overlay = document.createElement("div");
        overlay.id = "emergency-alert-modal";
        overlay.className = "modal-overlay";
        overlay.style.cssText = "z-index: 30000; position: fixed; inset: 0; background: rgba(18, 2, 8, 0.96); display: flex; justify-content: center; align-items: center; backdrop-filter: blur(12px);";
        document.body.appendChild(overlay);
    }

    const isExistential = alert.is_existential || alert.severity === 'EXISTENTIAL';
    const riskScore = alert.risk_score || 85;

    overlay.innerHTML = `
        <div class="glass-card" style="max-width: 580px; width: 92%; padding: 28px; display: flex; flex-direction: column; gap: 18px; border: 2px solid ${isExistential ? 'var(--vgt-red)' : 'var(--vgt-orange)'}; background: rgba(12, 3, 8, 0.98); box-shadow: 0 0 60px ${isExistential ? 'rgba(255,0,79,0.5)' : 'rgba(255,123,0,0.35)'}; border-radius: 12px; font-family: var(--font-mono);">
            <div style="display: flex; align-items: center; gap: 14px; color: ${isExistential ? 'var(--vgt-red)' : 'var(--vgt-orange)'};">
                <svg viewBox="0 0 24 24" width="34" height="34" stroke="currentColor" stroke-width="2.5" fill="none" class="vgt-inline-ddc9f789"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0zM12 9v4M12 17h.01"/></svg>
                <div>
                    <span style="font-size: 10px; letter-spacing: 0.15em; color: ${isExistential ? 'var(--vgt-red)' : 'var(--vgt-orange)'}; font-weight: bold;">
                        ${isExistential ? '🚨 EXISTENTIELLER NOTFALL-ALARM (RE-PERSISTENT)' : '⚠️ LOKALE SENTINEL-WARNUNG'}
                    </span>
                    <h3 class="vgt-inline-00238cdc">
                        ${escapeHtml(alert.title || 'UNBEKANNTE GEFAHR')}
                    </h3>
                </div>
            </div>

            <div style="background: rgba(255, 0, 79, 0.08); border: 1px solid ${isExistential ? 'rgba(255,0,79,0.35)' : 'rgba(255,123,0,0.35)'}; border-radius: 6px; padding: 16px; font-size: 12px; line-height: 1.6; color: #fff;">
                <div class="vgt-inline-168dba89">QUELLE: ${escapeHtml(alert.source || 'GLOBAL WATCH SENTINEL')} · STANDORT: ${escapeHtml(alert.city || 'LOKAL')}, ${escapeHtml(alert.country || '')}</div>
                <div>${escapeHtml(alert.description || '')}</div>
            </div>

            <div class="vgt-inline-7d0ff269">
                <span>RISK SCORE: <strong style="color: ${isExistential ? 'var(--vgt-red)' : 'var(--vgt-orange)'};">${riskScore}/100</strong> (Wiederholung aktiv)</span>
                <span>ZEITPUNKT: ${new Date(alert.timestamp || Date.now()).toLocaleTimeString()}</span>
            </div>

            <div class="vgt-inline-336a5798">
                <button id="emergency-btn-dismiss" class="cyber-button vgt-inline-5d6d1c06">
                    BESTÄTIGEN & SCHLIESSEN
                </button>
                <button id="emergency-btn-discuss" class="cyber-button vgt-inline-a4bf3bec">
                    MIT AETHEL BESPRECHEN ›
                </button>
            </div>
        </div>
    `;

    overlay.classList.remove("hidden");
    overlay.style.display = "flex";

    playEmergencyAlarmSound(isExistential);

    if (alert.voice_alert_text) {
        speak(alert.voice_alert_text);
    }

    const confirmAlert = () => {
        overlay.classList.add("hidden");
        overlay.style.display = "none";
        // Stop persistent re-notification loop for this alert
        if (unconfirmedAlertTimers.has(alert.id)) {
            clearInterval(unconfirmedAlertTimers.get(alert.id));
            unconfirmedAlertTimers.delete(alert.id);
        }
    };

    const btnDismiss = document.getElementById("emergency-btn-dismiss");
    if (btnDismiss) btnDismiss.onclick = confirmAlert;

    const btnDiscuss = document.getElementById("emergency-btn-discuss");
    if (btnDiscuss) {
        btnDiscuss.onclick = () => {
            confirmAlert();
            if (window.switchMode) window.switchMode("chat");
        };
    }

    // Schedule Risk-Based Persistent Re-Notification Loop if unconfirmed
    if (riskScore >= 75 && !unconfirmedAlertTimers.has(alert.id)) {
        const repeatIntervalMs = riskScore >= 85 ? 45000 : 90000; // 45s for high crisis, 90s for elevated
        const timer = setInterval(() => {
            if (unconfirmedAlertTimers.has(alert.id)) {
                // Re-sound alarm & re-open overlay if not confirmed yet
                showEmergencyOverlay(alert);
            }
        }, repeatIntervalMs);
        unconfirmedAlertTimers.set(alert.id, timer);
    }
}

function escapeHtml(str) {
    return String(str || '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

/** Push Personal Core home location into the sentinel so local filters stay current. */
export async function syncSentinelLocation(city, country) {
    try {
        await fetch(`${state.API_BASE}/v1/sentinel/alerts`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                city: String(city || '').trim(),
                country: String(country || '').trim()
            })
        });
    } catch (e) {
        // Non-fatal: backend also seeds location from personal profile on boot/save.
    }
}

export function startSentinelMonitor() {
    if (isMonitoring) return;
    isMonitoring = true;

    setInterval(async () => {
        try {
            const res = await fetch(`${state.API_BASE}/v1/sentinel/alerts`);
            if (!res.ok) return;
            const data = await res.json();
            const alerts = data.alerts || [];

            for (const alert of alerts) {
                if (!seenAlertIDs.has(alert.id)) {
                    seenAlertIDs.add(alert.id);
                    // Full emergency modal only for existential + city-level local matches.
                    // Country-only / global catalogue items must never force a Notfall-Overlay.
                    if (alert.is_existential === true && alert.local_match === true) {
                        showEmergencyOverlay(alert);
                    }
                }
            }
        } catch (e) {
            // Sentinel monitor quiet catch
        }
    }, 15000);
}
