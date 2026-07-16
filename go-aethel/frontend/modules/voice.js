import { state } from './state.js';
import * as api from './api.js';
import { rejectTool, executeApprovedTool } from './chat.js';

export function updateVoiceSphereUI(statusText, classState) {
    const mainSphere = document.getElementById("voice-sphere");
    const mainStatus = document.getElementById("sphere-hud-status");
    const miniSphere = document.getElementById("control-voice-sphere");
    const miniStatus = document.getElementById("control-voice-status");

    let miniBg = "var(--vgt-cyan)";
    let miniShadow = "0 0 8px var(--vgt-cyan)";

    if (classState === "voice-sphere listening") {
        miniBg = "var(--vgt-cyan)";
        miniShadow = "0 0 12px var(--vgt-cyan)";
    } else if (classState === "voice-sphere listening active-speech") {
        miniBg = "var(--vgt-green)";
        miniShadow = "0 0 15px var(--vgt-green)";
    } else if (classState === "voice-sphere speaking") {
        miniBg = "var(--vgt-purple)";
        miniShadow = "0 0 15px var(--vgt-purple)";
    } else if (classState === "voice-sphere processing") {
        miniBg = "var(--vgt-orange)";
        miniShadow = "0 0 12px var(--vgt-orange)";
    } else {
        miniBg = "rgba(0, 240, 255, 0.4)";
        miniShadow = "none";
    }

    if (mainSphere) mainSphere.className = classState;
    if (mainStatus) mainStatus.textContent = statusText;
    if (miniSphere) {
        miniSphere.style.background = miniBg;
        miniSphere.style.boxShadow = miniShadow;
    }
    if (miniStatus) miniStatus.textContent = statusText;

    // Support for the Sphere Workspace background canvas
    const workspaceSphere = document.getElementById("workspace-sphere");
    if (workspaceSphere) {
        let workState = "idle";
        if (classState.includes("listening")) workState = "listening";
        else if (classState.includes("speaking")) workState = "speaking";
        else if (classState.includes("processing")) workState = "processing";
        workspaceSphere.className = "sphere-canvas " + workState;
    }
}

export function stopSpeaking() {
    if (state.activeAudio) {
        try { state.activeAudio.pause(); } catch(e) {}
        state.activeAudio = null;
    }
    if (state.synth) {
        try { state.synth.cancel(); } catch(e) {}
    }
    state.isAethelSpeaking = false;
    state.currentlySpeakingText = "";
}

export function resetMicButton() {
    const elBtnMic = document.getElementById("btn-mic");
    const elSpeechIndicator = document.getElementById("speech-indicator");
    if (!elBtnMic) return;

    state.isListening = false;
    const isSpeaking = (state.activeAudio && !state.activeAudio.paused) || (state.synth && state.synth.speaking);

    if (state.isVoiceCallActive) {
        if (isSpeaking) {
            elBtnMic.className = "mic-button processing";
            if (elSpeechIndicator) elSpeechIndicator.textContent = "Freisprech-Modus aktiv... Ausgabe aktiv...";
        } else {
            elBtnMic.className = "mic-button listening";
            if (state.isWakeSessionActive) {
                if (elSpeechIndicator) elSpeechIndicator.textContent = "Whisper aktiv... Warte...";
                updateVoiceSphereUI("CORE LISTENING...", "voice-sphere listening");
            } else {
                if (elSpeechIndicator) elSpeechIndicator.textContent = `Wake-Word aktiv: "${state.wakeWord || "Aethel"}"`;
                updateVoiceSphereUI("WAKE STANDBY", "voice-sphere listening");
            }
        }
    } else {
        elBtnMic.className = "mic-button";
        if (elSpeechIndicator) elSpeechIndicator.textContent = "Push-to-Talk inaktiv";
    }
}

export function handleAethelSpeechCompleted() {
    state.isAethelSpeaking = false;
    state.currentlySpeakingText = "";
    
    state.speechCooldownActive = true;
    
    if (state.isSphereActive) {
        updateVoiceSphereUI("CORE LISTENING...", "voice-sphere listening");
    } else if (state.isVoiceCallActive && state.isWakeSessionActive) {
        updateVoiceSphereUI("CORE LISTENING...", "voice-sphere listening");
    } else if (state.isVoiceCallActive) {
        updateVoiceSphereUI("WAKE STANDBY", "voice-sphere listening");
    } else {
        updateVoiceSphereUI("CORE STANDBY", "voice-sphere");
    }
    
    setTimeout(() => {
        state.speechCooldownActive = false;
        
        if (state.isSphereActive && state.isVoiceCallActive && !state.isWakeSessionActive) {
            try { activateWakeSession(); } catch(e) {}
        } else if (localRecFallbackActive && state.isVoiceCallActive) {
            try { state.recognition.start(); } catch(e) {}
        } else if (state.pendingToolRequest && state.recognition && !state.isVoiceCallActive) {
            try {
                state.recognition.start();
                const elBtnMic = document.getElementById("btn-mic");
                const elSpeechIndicator = document.getElementById("speech-indicator");
                if (elBtnMic) elBtnMic.className = "mic-button listening";
                if (elSpeechIndicator) elSpeechIndicator.textContent = "System hört zu (Warte auf Freigabe)...";
            } catch(e) {
                console.error("Auto-start recognition failed", e);
            }
        }
    }, 1000);
}

export function handlePendingToolResponse(text) {
    if (!state.pendingToolRequest) return false;
    
    const textClean = text.toLowerCase().trim().replace(/[.,\/#!$%\^&\*;:{}=\-_`~()!?]/g, "");
    
    const rejectRoots = ["nein", "nee", "ne", "no", "n", "nicht", "ablehn", "block", "brech", "weiger", "stop", "stopp", "halt"];
    if (rejectRoots.some(root => textClean.includes(root))) {
        const req = state.pendingToolRequest;
        state.pendingToolRequest = null;
        rejectTool(req.msgIndex);
        return true;
    }
    
    const isHighRisk = state.pendingToolRequest.riskLevel === "High" || state.pendingToolRequest.riskLevel === "Critical" || state.pendingToolRequest.riskLevel === "Forbidden";
    
    if (isHighRisk) {
        const secureConfirmRoots = ["bestätig", "bestaetig", "freigeb", "confirm"];
        if (secureConfirmRoots.some(root => textClean.includes(root))) {
            const req = state.pendingToolRequest;
            state.pendingToolRequest = null;
            executeApprovedTool(req.msgIndex, req.name, req.args, req.approvalToken || "");
            return true;
        }
        
        const simpleApproveRoots = ["ja", "jo", "jep", "ok", "yes", "y", "mach", "go"];
        if (simpleApproveRoots.some(root => textClean.includes(root))) {
            speak("Sicherheitsfreigabe verweigert. Sagen Sie: 'Aethel, bestätige Ausführung'.");
            return true;
        }
    } else {
        const approveRoots = ["ja", "jo", "jep", "ok", "yes", "y", "geb", "gib", "frei", "laub", "führ", "fuehr", "stät", "staet", "genehm", "mach", "go", "feuer", "freigeb", "freigab", "erlaub", "bestätig", "bestaetig"];
        if (approveRoots.some(root => textClean.includes(root))) {
            const req = state.pendingToolRequest;
            state.pendingToolRequest = null;
            executeApprovedTool(req.msgIndex, req.name, req.args, req.approvalToken || "");
            return true;
        }
    }
    
    return false;
}

export async function speak(text) {
    if (state.isVoiceMuted) return;

    stopSpeaking();

    let cleanedText = text.replace(/[*_`#\-]/g, ' ')
                          .replace(/\[\[TOOL_DELTA\]\].*/g, '')
                          .replace(/\[TOOL OUTPUT.*/g, '')
                          .replace(/Aktion: navigate.*/g, '')
                          .trim();

    if (!cleanedText) return;

    state.isAethelSpeaking = true;
    state.currentlySpeakingText = cleanedText;

    if (state.recognition) {
        try { state.recognition.stop(); } catch(e) {}
    }

    const isBrowserVoice = state.currentVoice.startsWith("browser:");
    const isPremiumVoice = ["onyx", "nova", "alloy", "echo", "fable", "shimmer"].includes(state.currentVoice);

    if (isBrowserVoice || (isPremiumVoice && !state.hasOpenAI)) {
        if (!state.synth) {
            handleAethelSpeechCompleted();
            return;
        }

        const utterance = new SpeechSynthesisUtterance(cleanedText);
        utterance.lang = 'de-DE';

        const browserVoices = state.synth.getVoices();
        let targetVoiceName = state.currentVoice.replace("browser:", "");
        let voice = browserVoices.find(v => v.name === targetVoiceName);

        if (!voice) {
            // Prioritize high-quality online neural voices over robotic offline ones
            voice = browserVoices.find(v => v.lang.startsWith("de") && (v.name.toLowerCase().includes("online") || v.name.toLowerCase().includes("natural")));
            if (!voice) {
                voice = browserVoices.find(v => v.lang.startsWith("de"));
            }
        }

        if (voice) {
            utterance.voice = voice;
        }

        utterance.onend = () => {
            handleAethelSpeechCompleted();
        };
        utterance.onerror = (err) => {
            console.error("Browser speech synthesis error:", err);
            handleAethelSpeechCompleted();
        };

        state.synth.speak(utterance);
        return;
    }

    if (state.isVoiceCallActive) {
        updateVoiceSphereUI("CORE TRANSMITTING...", "voice-sphere speaking");
    }

    // Use fetch() to probe the TTS endpoint before creating an Audio element.
    // Audio.onerror does NOT reliably fire on HTTP 4xx/5xx responses — only on
    // network failures. By fetching as a blob first, we get the status code and
    // can fall back to browser TTS cleanly if Sherpa is not available.
    try {
        const resp = await fetch(`${state.API_BASE}/v1/audio/speech`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                text: cleanedText,
                voice: state.currentVoice,
                format: 'wav'
            })
        });
        if (!resp.ok) {
            // Backend TTS unavailable (e.g. no Sherpa model installed → HTTP 500)
            console.warn(`[Voice] TTS backend returned ${resp.status} — falling back to browser TTS`);
            throw new Error(`TTS backend HTTP ${resp.status}`);
        }
        const blob = await resp.blob();
        const blobUrl = URL.createObjectURL(blob);
        state.activeAudio = new Audio(blobUrl);

        state.activeAudio.onended = () => {
            URL.revokeObjectURL(blobUrl);
            state.activeAudio = null;
            handleAethelSpeechCompleted();
        };

        state.activeAudio.onerror = (e) => {
            console.error("Audio playback error", e);
            URL.revokeObjectURL(blobUrl);
            state.activeAudio = null;
            handleAethelSpeechCompleted();
        };

        state.activeAudio.play().catch(err => {
            console.error("Playback .play() rejected:", err);
            URL.revokeObjectURL(blobUrl);
            state.activeAudio = null;
            handleAethelSpeechCompleted();
        });

    } catch (err) {
        // Sherpa unavailable or network error → browser TTS fallback
        console.warn("[Voice] Falling back to browser TTS:", err.message);
        state.activeAudio = null;
        if (state.synth) {
            const utterance = new SpeechSynthesisUtterance(cleanedText);
            utterance.lang = 'de-DE';
            const bVoices = state.synth.getVoices();
            const deVoice = bVoices.find(v => v.lang.startsWith("de") && (v.name.toLowerCase().includes("online") || v.name.toLowerCase().includes("natural")))
                         || bVoices.find(v => v.lang.startsWith("de"));
            if (deVoice) utterance.voice = deVoice;
            utterance.onend = () => handleAethelSpeechCompleted();
            utterance.onerror = () => handleAethelSpeechCompleted();
            state.synth.speak(utterance);
        } else {
            handleAethelSpeechCompleted();
        }
    }
}


export function setupSpeechRecognition() {
    const SpeechRec = window.SpeechRecognition || window.webkitSpeechRecognition;
    const elSpeechIndicator = document.getElementById("speech-indicator");
    const elBtnMic = document.getElementById("btn-mic");

    if (!SpeechRec) {
        if (elSpeechIndicator) elSpeechIndicator.textContent = "Sprachsteuerung im Browser nicht unterstützt.";
        if (elBtnMic) elBtnMic.disabled = true;
        return;
    }

    state.recognition = new SpeechRec();
    state.recognition.continuous = false;
    state.recognition.lang = 'de-DE';
    state.recognition.interimResults = true;
    state.recognition.maxAlternatives = 1;

    state.recognition.onstart = () => {
        state.isListening = true;
        if (elBtnMic) elBtnMic.className = "mic-button listening";

        if (state.isVoiceCallActive) {
            if (elSpeechIndicator) elSpeechIndicator.textContent = "Freisprech-Modus aktiv... Bitte sprechen.";
            updateVoiceSphereUI("CORE LISTENING...", "voice-sphere listening");
        } else {
            if (elSpeechIndicator) elSpeechIndicator.textContent = "System hört zu... Sprechen Sie jetzt.";
        }
    };

    state.recognition.onresult = async (event) => {
        const result = event.results[event.results.length - 1];
        const isFinal = result.isFinal;
        const transcript = result[0].transcript.toLowerCase().trim();
        
        if (!isFinal) {
            if (state.isAethelSpeaking && transcript.length > 1) {
                const cleanSpeakingText = state.currentlySpeakingText.toLowerCase();
                if (!cleanSpeakingText.includes(transcript)) {
                    console.log("Barge-in detected via interim result:", transcript);
                    stopSpeaking();
                    updateVoiceSphereUI("CORE LISTENING...", "voice-sphere listening");
                    
                    if (elSpeechIndicator) elSpeechIndicator.textContent = `Interrupted: "${transcript}"`;
                    try { state.recognition.stop(); } catch(e) {}
                    return;
                }
            }
            return;
        }
        
        if (state.speechCooldownActive) {
            console.log("Ignored result due to speech cooldown (echo prevention)");
            return;
        }
        
        if (state.currentlySpeakingText) {
            const cleanTranscript = transcript.replace(/[^a-zA-Z0-9 ]/g, "").toLowerCase();
            const cleanAethelText = state.currentlySpeakingText.replace(/[^a-zA-Z0-9 ]/g, "").toLowerCase();
            
            if (cleanAethelText.includes(cleanTranscript) || cleanTranscript.includes(cleanAethelText)) {
                console.log("Discarded final result (echo match):", transcript);
                return;
            }
        }
        
        if (state.pendingToolRequest) {
            if (elSpeechIndicator) elSpeechIndicator.textContent = `Erkannt: "${transcript}" (Sprachfreigabe)`;
            if (handlePendingToolResponse(transcript)) {
                return;
            }
        }
        
        const isAgentActive = state.views.agent && !state.views.agent.classList.contains("hidden");
        const inputId = isAgentActive ? "agent-user-input" : "user-input";
        const elUserInput = document.getElementById(inputId);
        if (elUserInput) {
            elUserInput.value = result[0].transcript;
            if (elSpeechIndicator) elSpeechIndicator.textContent = `Erkannt: "${result[0].transcript}"`;
            if (isAgentActive) {
                const { sendOperatorFeedback } = await import('./agent_builder.js');
                sendOperatorFeedback();
            } else {
                const { sendMessage } = await import('./chat.js');
                sendMessage();
            }
        }
    };

    state.recognition.onerror = (event) => {
        console.error("Speech recognition error", event.error);
        if (event.error !== "no-speech") {
            if (elSpeechIndicator) elSpeechIndicator.textContent = `Sprachfehler: ${event.error}`;
        }
        resetMicButton();
    };

    state.recognition.onend = () => {
        resetMicButton();
    };
}

export async function refreshVoiceHealthHUD() {
    try {
        const status = await api.getVoiceHealth();

        const healthStatus = document.getElementById("voice-health-status");
        const healthOpenai = document.getElementById("voice-health-openai");
        const healthSapi5 = document.getElementById("voice-health-sapi5");
        const healthWhisper = document.getElementById("voice-health-whisper");

        if (healthStatus) {
            healthStatus.textContent = status.status;
            if (status.status.startsWith("ONLINE")) {
                healthStatus.className = "cyan-text";
            } else {
                healthStatus.className = "yellow-text";
            }
        }

        const setHealthBadge = (el, available) => {
            if (!el) return;
            if (available) {
                el.textContent = "ACTIVE";
                el.className = "green-text";
            } else {
                el.textContent = "UNAVAILABLE";
                el.className = "red-text";
            }
        };

        setHealthBadge(healthOpenai, status.openai_tts_available);
        setHealthBadge(healthSapi5, status.local_sapi5_available);
        setHealthBadge(healthWhisper, status.groq_whisper_available);

    } catch(e) {
        console.error("Failed to load voice subsystem health logs", e);
        const healthStatus = document.getElementById("voice-health-status");
        if (healthStatus) { healthStatus.textContent = "CORE UNREACHABLE"; healthStatus.className = "red-text"; }
    }
}

let mediaStream = null;
let vadRecorder = null;
let audioContext = null;
let analyserNode = null;
let isRecordingVad = false;
let vadChunks = [];
let silenceTimer = null;
let lastSpeechTime = 0;
let localRecFallbackActive = false;
const SPEAKING_THRESHOLD = 0.012;
const SILENCE_DELAY = 1200;
const WAKE_SESSION_MS = 45000;

function normalizeWakeText(value) {
    return String(value || "")
        .toLowerCase()
        .normalize("NFD")
        .replace(/[\u0300-\u036f]/g, "")
        .replace(/[^a-z0-9äöüß ]/gi, " ")
        .replace(/\s+/g, " ")
        .trim();
}

function configuredWakeWords() {
    const base = normalizeWakeText(state.wakeWord || "aethel");
    const words = new Set([base, "aethel", "athel", "ethel", "äthel"]);
    return [...words].filter(Boolean);
}

export function startWakeWordListener() {
    if (state.isSphereActive) {
        if (state.isVoiceCallActive && !state.isWakeSessionActive) {
            activateWakeSession();
        }
        return;
    }
    const SpeechRec = window.SpeechRecognition || window.webkitSpeechRecognition;
    const elSpeechIndicator = document.getElementById("speech-indicator");
    if (!SpeechRec || state.isWakeWordArmed) return;

    if (!state.wakeWordRecognizer) {
        state.wakeWordRecognizer = new SpeechRec();
        state.wakeWordRecognizer.continuous = false;
        state.wakeWordRecognizer.lang = 'de-DE';
        state.wakeWordRecognizer.interimResults = false;
        state.wakeWordRecognizer.maxAlternatives = 1;

        state.wakeWordRecognizer.onstart = () => {
            state.isWakeWordArmed = true;
            if (!state.isWakeSessionActive) {
                updateVoiceSphereUI("WAKE STANDBY", "voice-sphere listening");
                if (elSpeechIndicator) elSpeechIndicator.textContent = `Wake-Word aktiv: "${state.wakeWord || "Aethel"}"`;
            }
        };

        state.wakeWordRecognizer.onresult = (event) => {
            const result = event.results[event.results.length - 1];
            const transcript = normalizeWakeText(result?.[0]?.transcript || "");
            const matched = configuredWakeWords().some(word => transcript.includes(word));
            if (matched && state.isVoiceCallActive && !state.isWakeSessionActive) {
                activateWakeSession();
            }
        };

        state.wakeWordRecognizer.onerror = (event) => {
            if (event.error !== "no-speech" && event.error !== "aborted") {
                console.warn("Wake recognition error", event.error);
            }
        };

        state.wakeWordRecognizer.onend = () => {
            state.isWakeWordArmed = false;
            if (state.isVoiceCallActive && !state.isWakeSessionActive) {
                setTimeout(() => {
                    try { startWakeWordListener(); } catch(e) {}
                }, 500);
            }
        };
    }

    try {
        state.wakeWordRecognizer.start();
    } catch(e) {
        state.isWakeWordArmed = false;
    }
}

export function stopWakeWordListener() {
    state.isWakeWordArmed = false;
    if (state.wakeWordRecognizer) {
        try { state.wakeWordRecognizer.stop(); } catch(e) {}
    }
}

export function endWakeSession() {
    if (state.isSphereActive) {
        extendWakeSession();
        return;
    }
    state.isWakeSessionActive = false;
    if (state.wakeSessionTimer) {
        clearTimeout(state.wakeSessionTimer);
        state.wakeSessionTimer = null;
    }
    stopWhisperVad();
    if (state.isVoiceCallActive) {
        updateVoiceSphereUI("WAKE STANDBY", "voice-sphere listening");
        startWakeWordListener();
    } else {
        updateVoiceSphereUI("CORE STANDBY", "voice-sphere");
    }
}

export function extendWakeSession() {
    if (!state.isWakeSessionActive) return;
    if (state.wakeSessionTimer) clearTimeout(state.wakeSessionTimer);
    state.wakeSessionTimer = setTimeout(() => {
        if (!state.isAethelSpeaking && !state.pendingToolRequest) {
            endWakeSession();
        } else {
            extendWakeSession();
        }
    }, WAKE_SESSION_MS);
}

export function activateWakeSession() {
    if (!state.isVoiceCallActive || state.isWakeSessionActive) return;
    stopWakeWordListener();
    state.isWakeSessionActive = true;
    updateVoiceSphereUI("AETHEL AKTIV", "voice-sphere processing");
    const elSpeechIndicator = document.getElementById("speech-indicator");
    if (elSpeechIndicator) elSpeechIndicator.textContent = "Aethel aktiv. Whisper wird aktiviert...";
    if (!localRecFallbackActive) {
        if (state.isSphereActive) {
            speak("Sphere-Modus aktiv.");
        } else {
            speak("Ja, ich höre.");
        }
    }
    setTimeout(() => {
        if (state.isVoiceCallActive && state.isWakeSessionActive) {
            startWhisperVad();
            extendWakeSession();
        }
    }, 900);
}

export async function startWhisperVad() {
    if (audioContext) return;
    
    try {
        mediaStream = await navigator.mediaDevices.getUserMedia({ audio: true });
        
        const elSpeechIndicator = document.getElementById("speech-indicator");

        // Create a fresh MediaRecorder for each utterance — reusing the same
        // instance after stop() is unreliable in Chromium/WebView2.
        function createFreshRecorder() {
            const rec = new MediaRecorder(mediaStream, { mimeType: 'audio/webm' });
            let chunks = [];

            rec.ondataavailable = (e) => {
                if (e.data && e.data.size > 0) chunks.push(e.data);
            };

            rec.onstop = async () => {
                const blob = new Blob(chunks, { type: 'audio/webm' });
                chunks = [];

                if (blob.size < 1000) return;

                try {
                    if (elSpeechIndicator) {
                        elSpeechIndicator.textContent = "Analysiere Sprache mit Whisper...";
                    }
                    updateVoiceSphereUI("CORE PROCESSING...", "voice-sphere processing");

                    const formData = new FormData();
                    formData.append("audio", blob, "speech.webm");

                    const res = await fetch(`${state.API_BASE}/v1/audio/transcribe`, {
                        method: 'POST',
                        body: formData
                    });

                    if (res.status === 401 || res.status === 500) {
                        console.warn(`Whisper transcription endpoint returned status ${res.status}, falling back to local SpeechRecognition...`);
                        stopWhisperVad();
                        startLocalSpeechRecognitionFallback();
                        return;
                    }

                    const data = await res.json();
                    const transcript = (data.text || "").trim();

                    if (transcript) {
                        handleWhisperTranscript(transcript);
                    } else {
                        if (state.isVoiceCallActive) updateVoiceSphereUI("CORE LISTENING...", "voice-sphere listening");
                    }
                } catch(e) {
                    console.error("Whisper transcription failed, falling back to local SpeechRecognition...", e);
                    stopWhisperVad();
                    startLocalSpeechRecognitionFallback();
                }
            };

            return rec;
        }

        // Replace global vadRecorder with the factory approach
        vadRecorder = createFreshRecorder();
        
        const AudioContextClass = window.AudioContext || window.webkitAudioContext;
        audioContext = new AudioContextClass();
        if (audioContext.state === 'suspended') {
            audioContext.resume().catch(err => console.warn("Failed to resume AudioContext", err));
        }
        const source = audioContext.createMediaStreamSource(mediaStream);
        analyserNode = audioContext.createAnalyser();
        analyserNode.fftSize = 512;
        source.connect(analyserNode);
        
        const bufferLength = analyserNode.frequencyBinCount;
        const dataArray = new Float32Array(bufferLength);
        
        isRecordingVad = false;
        
        function checkVolume() {
            if (!audioContext || audioContext.state === 'closed') return;
            
            analyserNode.getFloatTimeDomainData(dataArray);
            
            let sumSquares = 0.0;
            for (let i = 0; i < dataArray.length; i++) {
                sumSquares += dataArray[i] * dataArray[i];
            }
            const rms = Math.sqrt(sumSquares / dataArray.length);
            
            if (state.isAethelSpeaking || state.speechCooldownActive) {
                if (isRecordingVad) {
                    try { vadRecorder.stop(); } catch(e) {}
                    isRecordingVad = false;
                    clearTimeout(silenceTimer);
                    silenceTimer = null;
                    // Prepare a fresh recorder for next utterance
                    vadRecorder = createFreshRecorder();
                }
                requestAnimationFrame(checkVolume);
                return;
            }
            
            if (rms > SPEAKING_THRESHOLD) {
                lastSpeechTime = Date.now();
                if (!isRecordingVad) {
                    console.log("Speech detected, starting fresh recording...");
                    isRecordingVad = true;
                    // Always use a known-inactive recorder
                    if (vadRecorder.state !== 'inactive') {
                        vadRecorder = createFreshRecorder();
                    }
                    try { vadRecorder.start(250); } catch(e) {
                        console.error("MediaRecorder start failed, retrying with fresh instance:", e);
                        vadRecorder = createFreshRecorder();
                        try { vadRecorder.start(250); } catch(e2) {
                            console.error("MediaRecorder double-start failed:", e2);
                            isRecordingVad = false;
                        }
                    }
                    updateVoiceSphereUI("CORE SPEAKING...", "voice-sphere listening active-speech");
                }
                
                clearTimeout(silenceTimer);
                silenceTimer = null;
            } else {
                if (isRecordingVad && !silenceTimer) {
                    silenceTimer = setTimeout(() => {
                        console.log("Silence duration met, stopping recorder...");
                        if (isRecordingVad) {
                            const oldRecorder = vadRecorder;
                            // Prepare fresh recorder BEFORE stopping so next utterance is ready
                            vadRecorder = createFreshRecorder();
                            isRecordingVad = false;
                            try { oldRecorder.stop(); } catch(e) {}
                        }
                        silenceTimer = null;
                    }, SILENCE_DELAY);
                }
            }
            
            requestAnimationFrame(checkVolume);
        }
        
        checkVolume();
        
    } catch(e) {
        console.error("Failed to start Whisper VAD stream, falling back to local SpeechRecognition", e);
        const elSpeechIndicator = document.getElementById("speech-indicator");
        if (elSpeechIndicator) elSpeechIndicator.textContent = "Mikrofon Zugriff für VAD verweigert.";
        startLocalSpeechRecognitionFallback();
    }
}


export function stopWhisperVad() {
    localRecFallbackActive = false;
    if (audioContext) {
        try { audioContext.close(); } catch(e) {}
        audioContext = null;
    }
    if (mediaStream) {
        mediaStream.getTracks().forEach(t => t.stop());
        mediaStream = null;
    }
    if (vadRecorder && vadRecorder.state !== 'inactive') {
        try { vadRecorder.stop(); } catch(e) {}
    }
    vadRecorder = null;
    isRecordingVad = false;
    clearTimeout(silenceTimer);
    silenceTimer = null;
}

export function startLocalSpeechRecognitionFallback() {
    if (localRecFallbackActive) return;
    
    const SpeechRec = window.SpeechRecognition || window.webkitSpeechRecognition;
    if (!SpeechRec) {
        console.error("Local browser SpeechRecognition API not supported on this client.");
        return;
    }
    
    localRecFallbackActive = true;
    console.log("[Voice] Starting browser-native SpeechRecognition fallback...");
    
    if (!state.recognition) {
        state.recognition = new SpeechRec();
        state.recognition.continuous = false;
        state.recognition.lang = 'de-DE';
        state.recognition.interimResults = false;
        
        state.recognition.onstart = () => {
            updateVoiceSphereUI("CORE LISTENING...", "voice-sphere listening");
        };
        
        state.recognition.onresult = async (event) => {
            const transcript = event.results[event.results.length - 1][0].transcript;
            if (transcript.trim()) {
                console.log("[Voice fallback] Transcribed:", transcript);
                handleWhisperTranscript(transcript);
            }
        };
        
        state.recognition.onerror = (e) => {
            if (e.error !== "no-speech" && e.error !== "aborted") {
                console.error("Local SpeechRecognition fallback error:", e.error);
            }
        };
        
        state.recognition.onend = () => {
            if (state.isVoiceCallActive && localRecFallbackActive) {
                setTimeout(() => {
                    if (state.isVoiceCallActive && !state.isAethelSpeaking && localRecFallbackActive) {
                        try { state.recognition.start(); } catch(err) {}
                    }
                }, 300);
            }
        };
    }
    
    try {
        state.recognition.start();
    } catch(e) {
        console.warn("SpeechRecognition already active or failed to start:", e.message);
    }
}

export async function handleWhisperTranscript(transcript) {
    if (!transcript) return;
    extendWakeSession();
    const textLower = transcript.toLowerCase().trim();
    const textClean = textLower.replace(/[.,\/#!$%\^&\*;:{}=\-_`~()!?]/g, "");

    const hallucinations = [
        "vielen dank", "vielen dank fürs zuschauen", "vielen dank fur das zuschauen",
        "thank you", "thank you for watching", "untertitel", "untertitel von amara.org",
        "untertitel im auftrag der zdfredaktion", "untertitelung", "untertitelung des zdf",
        "untertitelung des zdf 2020", "untertitelung zdf 2020", "zdf untertitel",
        "zdf", "zdf 2020", "musik", "music", "lachen", "laughter", "gerausche",
        "rauschen", "husten", "applause", "applaus"
    ];

    if (hallucinations.includes(textClean) || textClean === "") {
        console.log("Discarded Whisper VAD hallucination:", transcript);
        if (state.isVoiceCallActive) {
            updateVoiceSphereUI("CORE LISTENING...", "voice-sphere listening");
        }
        return;
    }
    
    if (state.pendingToolRequest) {
        const elSpeechIndicator = document.getElementById("speech-indicator");
        if (elSpeechIndicator) elSpeechIndicator.textContent = `Erkannt: "${transcript}" (Sprachfreigabe)`;
        if (handlePendingToolResponse(transcript)) {
            return;
        }
    }
    
    const isAgentActive = state.views.agent && !state.views.agent.classList.contains("hidden");
    const inputId = isAgentActive ? "agent-user-input" : "user-input";
    const elUserInput = document.getElementById(inputId);
    const elSpeechIndicator = document.getElementById("speech-indicator");
    if (elUserInput) {
        elUserInput.value = transcript;
        if (elSpeechIndicator) elSpeechIndicator.textContent = `Erkannt: "${transcript}"`;
        if (isAgentActive) {
            const { sendOperatorFeedback } = await import('./agent_builder.js');
            sendOperatorFeedback();
        } else {
            const { sendMessage } = await import('./chat.js');
            sendMessage();
        }
    }
}
