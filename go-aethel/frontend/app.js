// VGT AETHEL // MAIN ENTRYPOINT MODULE (ES6)

import { state } from './modules/state.js';
import { switchMode, checkSystemStatus, handleSetupSubmit } from './modules/ui.js';
import { sendMessage, startNewSession, loadSession, deleteSession } from './modules/chat.js';
import { setupSpeechRecognition, stopSpeaking, resetMicButton } from './modules/voice.js';
import { refreshSecurityHUD, fetchActiveLeasesList, fetchSecurityAuditTrail } from './modules/security.js';
import { setupMemoryUIEvents, updateMemoryCount } from './modules/memory.js';
import { setupSecretsUIEvents } from './modules/secrets.js';
import { setupControlUIEvents } from './modules/control.js';
import { setupTasksUIEvents, fetchKernelTasks } from './modules/tasks.js';

// Initialize Application
async function init() {
    // Map view panels
    state.views = {
        core: document.getElementById("view-core"),
        chat: document.getElementById("view-chat"),
        agent: document.getElementById("view-agent"),
        control: document.getElementById("view-control"),
        security: document.getElementById("view-security"),
        memory: document.getElementById("view-memory"),
        tasks: document.getElementById("view-tasks"),
    };

    // Map navigation buttons
    state.navButtons = {
        core: document.getElementById("nav-btn-core"),
        chat: document.getElementById("nav-btn-chat"),
        agent: document.getElementById("nav-btn-agent"),
        control: document.getElementById("nav-btn-control"),
        security: document.getElementById("nav-btn-security"),
        memory: document.getElementById("nav-btn-memory"),
        tasks: document.getElementById("nav-btn-tasks"),
    };

    setupViewNavigation();
    setupSpeechRecognition();
    setupEventListeners();
    setupMemoryUIEvents();
    setupSecretsUIEvents();
    setupTasksUIEvents();
    setupControlUIEvents();

    await checkSystemStatus();
    initWakeWordListener();

    // Default view is core HUD
    switchMode("core");

    // Periodic polling for status bar HUD and active viewport logs
    setInterval(() => {
        refreshSecurityHUD();
        const activeView = Object.keys(state.views).find(key => state.views[key] && !state.views[key].classList.contains("hidden"));
        
        if (activeView === "core") {
            import('./modules/security.js').then(m => m.fetchKernelLogs());
            import('./modules/voice.js').then(m => m.refreshVoiceHealthHUD());
        }
        if (activeView === "security") {
            fetchActiveLeasesList();
            fetchSecurityAuditTrail();
        }
        fetchKernelTasks();
    }, 3000);
}

// Setup Tab Switching Navigation
function setupViewNavigation() {
    Object.keys(state.navButtons).forEach(key => {
        if (state.navButtons[key]) {
            state.navButtons[key].addEventListener("click", () => {
                switchMode(key);
            });
        }
    });
}

// Setup UI Event Listeners
function setupEventListeners() {
    const elBtnInitiate = document.getElementById("btn-initiate");
    const elApiKey = document.getElementById("api-key");
    const elBtnSend = document.getElementById("btn-send");
    const elUserInput = document.getElementById("user-input");
    const elBtnNewChat = document.getElementById("btn-new-chat");
    const elBtnMic = document.getElementById("btn-mic");
    const elBtnToggleVoice = document.getElementById("btn-toggle-voice");
    const elBtnVoiceLink = document.getElementById("btn-voice-link");

    const iconVoiceOn = document.getElementById("icon-voice-on");
    const iconVoiceOff = document.getElementById("icon-voice-off");

    const elBtnTabBrowser = document.getElementById("btn-tab-browser");
    const elBtnTabCore = document.getElementById("btn-tab-core");
    const elBtnTabLogs = document.getElementById("btn-tab-logs");

    const elBrowserHudView = document.getElementById("browser-hud-view");
    const elCoreHudView = document.getElementById("core-hud-view");
    const elKernelHudView = document.getElementById("kernel-hud-view");
    const elSphereStatus = document.getElementById("sphere-hud-status");
    const elVoiceSphere = document.getElementById("voice-sphere");

    // API Setup Wizard submit
    if (elBtnInitiate) {
        elBtnInitiate.addEventListener("click", handleSetupSubmit);
    }
    if (elApiKey) {
        elApiKey.addEventListener("keypress", (e) => {
            if (e.key === "Enter") handleSetupSubmit();
        });
    }

    // Chat sending
    if (elBtnSend) {
        elBtnSend.addEventListener("click", sendMessage);
    }
    if (elUserInput) {
        elUserInput.addEventListener("keypress", (e) => {
            if (e.key === "Enter") sendMessage();
        });
    }

    if (elBtnNewChat) {
        elBtnNewChat.addEventListener("click", () => {
            startNewSession();
        });
    }

    // Microphone click
    if (elBtnMic) {
        elBtnMic.addEventListener("click", () => {
            if (!state.recognition) return;
            if (state.isListening) {
                state.recognition.stop();
            } else {
                if (state.synth) {
                    try { state.synth.cancel(); } catch(e) {}
                }
                state.recognition.start();
            }
        });
    }

    // Voice Call Mode Toggle
    if (elBtnVoiceLink) {
        elBtnVoiceLink.addEventListener("click", async () => {
            state.isVoiceCallActive = !state.isVoiceCallActive;
            stopSpeaking();
            
            const voiceModule = await import('./modules/voice.js');

            if (state.isVoiceCallActive) {
                elBtnVoiceLink.classList.add("active");
                
                if (state.wakeWordRecognizer) {
                    try { state.wakeWordRecognizer.stop(); } catch(e) {}
                }
                if (state.recognition) {
                    try { state.recognition.stop(); } catch(e) {}
                }
                
                // Auto switch tab to Core neural view
                if (elBtnTabBrowser) elBtnTabBrowser.classList.remove("active");
                if (elBtnTabCore) elBtnTabCore.classList.add("active");
                if (elBtnTabLogs) elBtnTabLogs.classList.remove("active");
                
                if (elBrowserHudView) elBrowserHudView.classList.add("hidden");
                if (elCoreHudView) elCoreHudView.classList.remove("hidden");
                if (elKernelHudView) elKernelHudView.classList.add("hidden");
                voiceModule.updateVoiceSphereUI("CORE INITIALIZING...", "voice-sphere processing");
                
                voiceModule.speak("Sprachverbindung hergestellt.");
                
                setTimeout(() => {
                    if (state.isVoiceCallActive) {
                        voiceModule.startWhisperVad();
                    }
                }, 1000);
            } else {
                elBtnVoiceLink.classList.remove("active");
                if (state.activeAudio) state.activeAudio.pause();
                if (state.synth) state.synth.cancel();
                
                voiceModule.stopWhisperVad();
                
                if (elBtnTabBrowser) elBtnTabBrowser.classList.add("active");
                if (elBtnTabCore) elBtnTabCore.classList.remove("active");
                if (elBtnTabLogs) elBtnTabLogs.classList.remove("active");
                
                if (elBrowserHudView) elBrowserHudView.classList.remove("hidden");
                if (elCoreHudView) elCoreHudView.classList.add("hidden");
                if (elKernelHudView) elKernelHudView.classList.add("hidden");
                voiceModule.updateVoiceSphereUI("CORE STANDBY", "voice-sphere");
                
                voiceModule.speak("Sprachverbindung getrennt.");
                if (state.recognition) {
                    try { state.recognition.stop(); } catch(e) {}
                }
                
                if (state.wakeWordRecognizer) {
                    try { state.wakeWordRecognizer.start(); } catch(e) {}
                }
            }
        });
    }

    // Manual Tab Button clicks
    if (elBtnTabBrowser && elBtnTabCore) {
        elBtnTabBrowser.addEventListener("click", () => {
            elBtnTabBrowser.classList.add("active");
            elBtnTabCore.classList.remove("active");
            if (elBtnTabLogs) elBtnTabLogs.classList.remove("active");
            
            elBrowserHudView.classList.remove("hidden");
            elCoreHudView.classList.add("hidden");
            if (elKernelHudView) elKernelHudView.classList.add("hidden");
        });

        elBtnTabCore.addEventListener("click", () => {
            elBtnTabCore.classList.add("active");
            elBtnTabBrowser.classList.remove("active");
            if (elBtnTabLogs) elBtnTabLogs.classList.remove("active");
            
            elCoreHudView.classList.remove("hidden");
            elBrowserHudView.classList.add("hidden");
            if (elKernelHudView) elKernelHudView.classList.add("hidden");
        });
        
        if (elBtnTabLogs) {
            elBtnTabLogs.addEventListener("click", () => {
                elBtnTabLogs.classList.add("active");
                elBtnTabBrowser.classList.remove("active");
                elBtnTabCore.classList.remove("active");
                
                if (elKernelHudView) elKernelHudView.classList.remove("hidden");
                if (elBrowserHudView) elBrowserHudView.classList.add("hidden");
                if (elCoreHudView) elCoreHudView.classList.add("hidden");
                
                import('./modules/security.js').then(m => m.fetchKernelLogs());
            });
        }
    }

    // Voice Mute Toggle
    if (elBtnToggleVoice) {
        elBtnToggleVoice.addEventListener("click", () => {
            state.isVoiceMuted = !state.isVoiceMuted;
            if (state.isVoiceMuted) {
                elBtnToggleVoice.classList.remove("active");
                if (iconVoiceOn) iconVoiceOn.classList.add("hidden");
                if (iconVoiceOff) iconVoiceOff.classList.remove("hidden");
                stopSpeaking();
            } else {
                elBtnToggleVoice.classList.add("active");
                if (iconVoiceOn) iconVoiceOn.classList.remove("hidden");
                if (iconVoiceOff) iconVoiceOff.classList.add("hidden");
            }
        });
    }

    // Connect Handoff Modal buttons
    const btnHandoffCopy = document.getElementById("handoff-btn-copy");
    const btnHandoffClose = document.getElementById("handoff-btn-close");
    const handoffModal = document.getElementById("handoff-modal");
    
    if (btnHandoffCopy) {
        btnHandoffCopy.addEventListener("click", () => {
            const body = document.getElementById("handoff-modal-content");
            if (body) {
                navigator.clipboard.writeText(body.textContent);
                btnHandoffCopy.textContent = "KOPIERT!";
                setTimeout(() => { btnHandoffCopy.textContent = "PROMPT KOPIEREN"; }, 1500);
            }
        });
    }
    if (btnHandoffClose && handoffModal) {
        btnHandoffClose.addEventListener("click", () => {
            handoffModal.classList.add("hidden");
        });
    }
}

function initWakeWordListener() {
    const SpeechRec = window.SpeechRecognition || window.webkitSpeechRecognition;
    if (!SpeechRec) return;
    
    state.wakeWordRecognizer = new SpeechRec();
    state.wakeWordRecognizer.continuous = false;
    state.wakeWordRecognizer.lang = 'de-DE';
    state.wakeWordRecognizer.interimResults = false;
    
    state.wakeWordRecognizer.onresult = (event) => {
        const text = event.results[0][0].transcript.toLowerCase();
        if (text.includes("aethel") || text.includes("äthel") || text.includes("ethel")) {
            console.log("Wake word detected!");
            const elBtnVoiceLink = document.getElementById("btn-voice-link");
            if (elBtnVoiceLink && !state.isVoiceCallActive) {
                elBtnVoiceLink.click();
            }
        }
    };
    
    state.wakeWordRecognizer.onend = () => {
        if (!state.isVoiceCallActive) {
            setTimeout(() => {
                try { state.wakeWordRecognizer.start(); } catch(e) {}
            }, 800);
        }
    };
    
    document.addEventListener("click", () => {
        if (!state.isVoiceCallActive) {
            try { state.wakeWordRecognizer.start(); } catch(e) {}
        }
    }, { once: true });
}

// Start client on load
window.addEventListener("DOMContentLoaded", () => {
    init();
    if (window.speechSynthesis) {
        window.speechSynthesis.onvoiceschanged = () => {
            import('./modules/ui.js').then(m => m.loadVoices());
        };
    }
});
