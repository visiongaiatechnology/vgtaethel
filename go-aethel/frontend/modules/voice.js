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
            if (elSpeechIndicator) elSpeechIndicator.textContent = "Freisprech-Modus aktiv... Warte...";
            updateVoiceSphereUI("CORE STANDBY", "voice-sphere");
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
    
    if (state.isVoiceCallActive) {
        updateVoiceSphereUI("CORE LISTENING...", "voice-sphere listening");
    } else {
        updateVoiceSphereUI("CORE STANDBY", "voice-sphere");
    }
    
    setTimeout(() => {
        state.speechCooldownActive = false;
        
        if (state.pendingToolRequest && state.recognition && !state.isVoiceCallActive) {
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
            executeApprovedTool(req.msgIndex, req.name, req.args, false);
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
            executeApprovedTool(req.msgIndex, req.name, req.args, false);
            return true;
        }
    }
    
    return false;
}

export function speak(text) {
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

    if (state.currentVoice.startsWith("browser:") || (!state.hasOpenAI && !state.currentVoice.includes("Lokal"))) {
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
            voice = browserVoices.find(v => v.lang.startsWith("de"));
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

    const audioUrl = `${state.API_BASE}/v1/audio/speech?text=${encodeURIComponent(cleanedText)}&voice=${encodeURIComponent(state.currentVoice)}&t=${Date.now()}`;
    state.activeAudio = new Audio(audioUrl);
    
    state.activeAudio.onended = () => {
        state.activeAudio = null;
        handleAethelSpeechCompleted();
    };

    state.activeAudio.onerror = (e) => {
        console.error("Audio playback error from Core TTS", e);
        state.activeAudio = null;
        
        if (state.synth) {
            const utterance = new SpeechSynthesisUtterance(cleanedText);
            utterance.lang = 'de-DE';
            utterance.onend = () => {
                handleAethelSpeechCompleted();
            };
            utterance.onerror = () => {
                handleAethelSpeechCompleted();
            };
            state.synth.speak(utterance);
        } else {
            handleAethelSpeechCompleted();
        }
    };

    state.activeAudio.play().catch(err => {
        console.error("Playback failed, trying browser TTS fallback", err);
        if (state.synth) {
            const utterance = new SpeechSynthesisUtterance(cleanedText);
            utterance.lang = 'de-DE';
            utterance.onend = () => {
                handleAethelSpeechCompleted();
            };
            state.synth.speak(utterance);
        } else {
            handleAethelSpeechCompleted();
        }
    });
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
        
        const elUserInput = document.getElementById("user-input");
        if (elUserInput) {
            elUserInput.value = result[0].transcript;
            if (elSpeechIndicator) elSpeechIndicator.textContent = `Erkannt: "${result[0].transcript}"`;
            const { sendMessage } = await import('./chat.js');
            sendMessage();
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
const SPEAKING_THRESHOLD = 0.025;
const SILENCE_DELAY = 1200;

export async function startWhisperVad() {
    if (audioContext) return;
    
    try {
        mediaStream = await navigator.mediaDevices.getUserMedia({ audio: true });
        
        vadRecorder = new MediaRecorder(mediaStream, { mimeType: 'audio/webm' });
        vadChunks = [];
        
        vadRecorder.ondataavailable = (e) => {
            if (e.data && e.data.size > 0) {
                vadChunks.push(e.data);
            }
        };
        
        const elSpeechIndicator = document.getElementById("speech-indicator");

        vadRecorder.onstop = async () => {
            const blob = new Blob(vadChunks, { type: 'audio/webm' });
            vadChunks = [];
            
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
                
                if (res.status === 401) {
                    if (elSpeechIndicator) elSpeechIndicator.textContent = "Whisper Fehler: Groq Key nicht gesetzt.";
                    if (state.isVoiceCallActive) {
                        updateVoiceSphereUI("CORE LISTENING...", "voice-sphere listening");
                    }
                    return;
                }
                
                const data = await res.json();
                const transcript = (data.text || "").trim();
                
                if (transcript) {
                    handleWhisperTranscript(transcript);
                } else {
                    if (state.isVoiceCallActive) {
                        updateVoiceSphereUI("CORE LISTENING...", "voice-sphere listening");
                    }
                }
            } catch(e) {
                console.error("Whisper transcription failed", e);
                if (elSpeechIndicator) elSpeechIndicator.textContent = "Whisper Verbindung fehlgeschlagen.";
                if (state.isVoiceCallActive) {
                    updateVoiceSphereUI("CORE LISTENING...", "voice-sphere listening");
                }
            }
        };
        
        const AudioContextClass = window.AudioContext || window.webkitAudioContext;
        audioContext = new AudioContextClass();
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
                }
                requestAnimationFrame(checkVolume);
                return;
            }
            
            if (rms > SPEAKING_THRESHOLD) {
                lastSpeechTime = Date.now();
                if (!isRecordingVad) {
                    console.log("Speech detected, starting recording...");
                    isRecordingVad = true;
                    vadChunks = [];
                    try { vadRecorder.start(250); } catch(e) {}
                    updateVoiceSphereUI("CORE SPEAKING...", "voice-sphere listening active-speech");
                }
                
                clearTimeout(silenceTimer);
                silenceTimer = null;
            } else {
                if (isRecordingVad && !silenceTimer) {
                    silenceTimer = setTimeout(() => {
                        console.log("Silence duration met, stopping recorder...");
                        if (isRecordingVad) {
                            try { vadRecorder.stop(); } catch(e) {}
                            isRecordingVad = false;
                        }
                        silenceTimer = null;
                    }, SILENCE_DELAY);
                }
            }
            
            requestAnimationFrame(checkVolume);
        }
        
        checkVolume();
        
    } catch(e) {
        console.error("Failed to start Whisper VAD stream", e);
        if (elSpeechIndicator) elSpeechIndicator.textContent = "Mikrofon Zugriff für VAD verweigert.";
    }
}

export function stopWhisperVad() {
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

export async function handleWhisperTranscript(transcript) {
    if (!transcript) return;
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
    
    const elUserInput = document.getElementById("user-input");
    const elSpeechIndicator = document.getElementById("speech-indicator");
    if (elUserInput) {
        elUserInput.value = transcript;
        if (elSpeechIndicator) elSpeechIndicator.textContent = `Erkannt: "${transcript}"`;
        const { sendMessage } = await import('./chat.js');
        sendMessage();
    }
}
