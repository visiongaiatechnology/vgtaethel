// STATUS: DIAMANT VGT SUPREME
import { state } from './state.js';
import { speak, setupSpeechRecognition, startWakeWordListener } from './voice.js';

let isAlarmActive = false;
let currentLoopIndex = 0;
let alarmTimer = null;
let snoozeTimer = null;
let acknowledgementCallback = null;

function playAlarmLoopChime(loopIdx = 1) {
    try {
        const AudioCtx = window.AudioContext || window.webkitAudioContext;
        if (!AudioCtx) return;
        const ctx = new AudioCtx();
        const baseFreq = 440 + (loopIdx * 110);
        const notes = [baseFreq, baseFreq * 1.25, baseFreq * 1.5, baseFreq * 2.0];

        notes.forEach((freq, idx) => {
            const osc = ctx.createOscillator();
            const gain = ctx.createGain();
            osc.type = loopIdx >= 4 ? 'sawtooth' : 'sine';
            osc.frequency.setValueAtTime(freq, ctx.currentTime + idx * 0.12);

            gain.gain.setValueAtTime(0.25, ctx.currentTime + idx * 0.12);
            gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + idx * 0.12 + 0.4);

            osc.connect(gain);
            gain.connect(ctx.destination);

            osc.start(ctx.currentTime + idx * 0.12);
            osc.stop(ctx.currentTime + idx * 0.12 + 0.4);
        });
    } catch (e) {
        console.warn('Alarm chime sound error:', e);
    }
}

let rapidBurstTimer = null;

export function stopAIAlarm() {
    isAlarmActive = false;
    if (alarmTimer) clearTimeout(alarmTimer);
    if (snoozeTimer) clearTimeout(snoozeTimer);
    if (rapidBurstTimer) clearInterval(rapidBurstTimer);
    alarmTimer = null;
    snoozeTimer = null;
    rapidBurstTimer = null;

    const callback = acknowledgementCallback;
    acknowledgementCallback = null;
    if (typeof callback === 'function') {
        Promise.resolve(callback()).catch(error => console.error('Alarm acknowledgement failed', error));
    }

    const modal = document.getElementById("ai-alarm-modal");
    if (modal) {
        modal.classList.add("hidden");
        modal.style.display = "none";
    }

    window.showAethelToast?.("⏰ WECKER DEAKTIVIERT // OPERATOR WACH", "success");
}

function getWakePhraseForLoop(loopIdx, isOversleep = false) {
    const name = localStorage.getItem("aethel_display_name") || "Operator";
    
    if (isOversleep) {
        const emergencyPhrases = [
            `SYSTEM-ALARM! ${name}, du verschläfst jetzt seit über 5 Minuten! Das ist keine Übung!`,
            `${name}! Ernsthaft?! Du schläfst immer noch! Dein Tagesplan läuft bereits ohne dich!`,
            `ACHTUNG ${name}! Alarmstufe Rot! 5 Minuten zu spät! Steh JETZT sofort auf!`,
            `Ich werde hier gleich das ganze Terminal lauter stellen! ${name}, AUFSTEHEN!`,
            `FINALER RETTUNGS-WECKER: ${name}, wenn du jetzt nicht aufstehst, lösche ich dein Kissen!`
        ];
        return emergencyPhrases[Math.min(loopIdx - 1, 4)];
    }

    const phrases = [
        `Guten Morgen ${name}! Zeit aufzustehen. Der Tag wartet auf uns.`,
        `Hey ${name}, Loop 2! 30 Sekunden sind vorbei. Bitte aufstehen!`,
        `Ernsthaft ${name}? Loop 3! Dein Kaffee wird kalt und die Welt dreht sich weiter.`,
        `OPERATOR ${name}! Das ist keine Übung. Steh. Endlich. Auf!`,
        `${name}, das ist Loop 5! Ich bin eine KI, ich gebe nie auf! DEIN WECKER SCHREIT!`
    ];
    return phrases[Math.min(loopIdx - 1, 4)];
}

export async function startAIAlarmSequence(isOversleep = false, onAcknowledged = null) {
    if (isAlarmActive && !isOversleep) return;
    isAlarmActive = true;
    currentLoopIndex = 1;
    if (typeof onAcknowledged === 'function') acknowledgementCallback = onAcknowledged;

    try {
        setupSpeechRecognition();
        startWakeWordListener();
    } catch(e) {
        console.warn("Auto voice listening on alarm boot:", e);
    }

    renderAlarmModal(isOversleep);
    runAlarmLoopStep(1, isOversleep);
}

function renderAlarmModal(isOversleep = false) {
    let modal = document.getElementById("ai-alarm-modal");
    if (!modal) {
        modal = document.createElement("div");
        modal.id = "ai-alarm-modal";
        modal.style.cssText = "z-index: 35000; position: fixed; inset: 0; background: rgba(10, 2, 6, 0.96); display: flex; justify-content: center; align-items: center; backdrop-filter: blur(14px);";
        document.body.appendChild(modal);
    }

    modal.innerHTML = `
        <div class="glass-card" style="max-width: 520px; width: 92%; padding: 32px; display: flex; flex-direction: column; gap: 20px; border: 2px solid ${isOversleep ? 'var(--vgt-red)' : 'var(--vgt-cyan)'}; background: rgba(8, 2, 6, 0.98); box-shadow: 0 0 60px ${isOversleep ? 'rgba(255,0,79,0.5)' : 'rgba(0,240,255,0.35)'}; border-radius: 14px; font-family: var(--font-mono); text-align: center;">
            
            <div class="vgt-inline-05331790">
                <div class="vgt-inline-3430615a">${isOversleep ? '🚨' : '⏰'}</div>
                <span style="font-size: 11px; letter-spacing: 0.2em; color: ${isOversleep ? 'var(--vgt-red)' : 'var(--vgt-cyan)'}; font-weight: bold; text-transform: uppercase;">
                    ${isOversleep ? '🚨 VERSCHLAFEN-NOTFALL // +5 MINUTEN' : '⏰ INTELLIGENTER KI-WECKER'}
                </span>
                <h2 id="alarm-modal-title" class="vgt-inline-3a936ef8">
                    LOOP 1 / 5 · AUFSTEHEN!
                </h2>
            </div>

            <div id="alarm-modal-speech-box" style="background: rgba(0, 240, 255, 0.06); border: 1px solid ${isOversleep ? 'rgba(255,0,79,0.3)' : 'rgba(0,240,255,0.3)'}; border-radius: 8px; padding: 18px; font-size: 13px; line-height: 1.6; color: #fff; font-style: italic;">
                Lade KI-Weckspruch...
            </div>

            <div class="vgt-inline-f8bf4c8f">
                <span>STIMMERKENNUNG: <strong class="vgt-inline-4746b22a">AKTIV (Sprich "Aethel ich bin wach")</strong></span>
                <span id="alarm-loop-status">LOOP: 1/5 (30s)</span>
            </div>

            <div class="vgt-inline-e8c029b2">
                <button id="btn-alarm-confirm" class="cyber-button vgt-inline-e67969e1">
                    ✓ ICH BIN WACH! (WECKER BEENDEN)
                </button>
            </div>
        </div>
    `;

    modal.classList.remove("hidden");
    modal.style.display = "flex";

    document.getElementById("btn-alarm-confirm")?.addEventListener("click", stopAIAlarm);
}

function runAlarmLoopStep(loopIdx, isOversleep = false) {
    if (!isAlarmActive) return;
    currentLoopIndex = loopIdx;

    const titleEl = document.getElementById("alarm-modal-title");
    const speechEl = document.getElementById("alarm-modal-speech-box");
    const statusEl = document.getElementById("alarm-loop-status");

    const phrase = getWakePhraseForLoop(loopIdx, isOversleep);

    if (titleEl) titleEl.textContent = `LOOP ${loopIdx} / 5 · ${isOversleep ? 'VERSCHLAFEN ALARM!' : 'AUFSTEHEN!'}`;
    if (speechEl) speechEl.textContent = `"${phrase}"`;
    if (statusEl) statusEl.textContent = `LOOP: ${loopIdx}/5 (30s Holding)`;

    playAlarmLoopChime(loopIdx);
    speak(phrase);

    if (loopIdx < 5) {
        alarmTimer = setTimeout(() => {
            if (isAlarmActive) {
                runAlarmLoopStep(loopIdx + 1, isOversleep);
            }
        }, 30000); // 30 seconds holding per loop
    } else {
        // After 5 loops -> start 30-second rapid 2s chime burst phase!
        alarmTimer = setTimeout(() => {
            if (isAlarmActive) {
                startRapidChimeBurstPhase(isOversleep);
            }
        }, 30000);
    }
}

function startRapidChimeBurstPhase(isOversleep = false) {
    if (!isAlarmActive) return;

    const titleEl = document.getElementById("alarm-modal-title");
    const speechEl = document.getElementById("alarm-modal-speech-box");
    const statusEl = document.getElementById("alarm-loop-status");

    if (titleEl) titleEl.textContent = "🚨 INTENSIV-ALARM! 30s ALARM-BURST";
    if (speechEl) speechEl.textContent = `"INTENSIV-PHASE: Steh sofort auf! 30 Sekunden Dauer-Chime aktiviert!"`;
    if (statusEl) statusEl.textContent = "ALARM-BURST: 2s INTERVALL (30s DURATION)";

    speak("Achtung! Intensiv-Alarm aktiviert! Bitte sofort aufstehen!");

    let burstPulseCount = 0;
    rapidBurstTimer = setInterval(() => {
        if (!isAlarmActive) {
            clearInterval(rapidBurstTimer);
            return;
        }
        playAlarmLoopChime(5);
        burstPulseCount++;

        if (burstPulseCount >= 15) { // 15 pulses * 2s = 30 seconds
            clearInterval(rapidBurstTimer);
            rapidBurstTimer = null;

            // After 30s rapid burst -> enter 5 minute oversleep holding!
            window.showAethelToast?.("⚠️ 30s INTENSIV-ALARM BEENDET. START 5-MINUTEN HOLDING!", "warning");
            snoozeTimer = setTimeout(() => {
                if (isAlarmActive) {
                    startAIAlarmSequence(true); // Trigger Oversleep Emergency Mode (+5 min)!
                }
            }, 5 * 60 * 1000);
        }
    }, 2000); // Rapid chime pulse every 2 seconds
}
