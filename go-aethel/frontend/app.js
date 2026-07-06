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
import { setupSettingsUIEvents } from './modules/settings.js';

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
        settings: document.getElementById("view-settings"),
        archive: document.getElementById("view-archive"),
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
        settings: document.getElementById("nav-btn-settings"),
        archive: document.getElementById("nav-btn-archive"),
    };

    setupViewNavigation();
    setupSpeechRecognition();
    setupEventListeners();
    setupMemoryUIEvents();
    setupSecretsUIEvents();
    setupTasksUIEvents();
    setupControlUIEvents();
    setupSettingsUIEvents();

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

    // Fade out splash screen after 3 seconds
    const splash = document.getElementById("startup-splash-screen");
    const splashStatus = document.getElementById("splash-status-text");
    if (splash) {
        if (splashStatus) {
            setTimeout(() => splashStatus.textContent = "INITIALIZING CORE LEASES...", 800);
            setTimeout(() => splashStatus.textContent = "VERIFYING SECURITY SHIELD...", 1600);
            setTimeout(() => splashStatus.textContent = "SYSTEM BOOT READY", 2400);
        }
        setTimeout(() => {
            splash.style.opacity = "0";
            setTimeout(() => splash.remove(), 800);
        }, 3000);
    }
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
            const modal = document.getElementById("new-chat-mount-modal");
            const input = document.getElementById("new-chat-mount-input");
            if (modal) {
                if (input) input.value = "";
                modal.classList.remove("hidden");
            } else {
                startNewSession();
            }
        });
    }

    // Wire mount popup events
    const newChatBtnBrowse = document.getElementById("new-chat-btn-browse");
    const newChatBtnSkip = document.getElementById("new-chat-btn-skip");
    const newChatBtnConfirm = document.getElementById("new-chat-btn-confirm");
    const newChatMountModal = document.getElementById("new-chat-mount-modal");
    const newChatMountInput = document.getElementById("new-chat-mount-input");

    if (newChatBtnBrowse) {
        newChatBtnBrowse.addEventListener("click", async () => {
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.SelectDirectory) {
                const dir = await window.go.main.App.SelectDirectory();
                if (dir && newChatMountInput) {
                    newChatMountInput.value = dir;
                }
            }
        });
    }

    if (newChatBtnSkip) {
        newChatBtnSkip.addEventListener("click", () => {
            if (newChatMountModal) newChatMountModal.classList.add("hidden");
            startNewSession();
        });
    }

    if (newChatBtnConfirm) {
        newChatBtnConfirm.addEventListener("click", async () => {
            if (newChatMountModal) newChatMountModal.classList.add("hidden");
            const path = newChatMountInput ? newChatMountInput.value.trim() : "";
            startNewSession();
            if (path) {
                try {
                    const res = await fetch(`${state.API_BASE}/v1/tools/execute`, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                            name: "fs_mount_folder",
                            args: { path: path },
                            override_security: true
                        })
                    });
                    const data = await res.json();
                    if (data.status === "success") {
                        const elChatOutput = document.getElementById("chat-output");
                        if (elChatOutput) {
                            elChatOutput.insertAdjacentHTML('beforeend', `
                                <div class="system-message font-mono" style="color: var(--vgt-cyan); border-color: var(--vgt-cyan); background: rgba(0, 240, 255, 0.05); margin-top: 10px;">
                                    <p>[ORDNER FREIGEGEBEN]</p>
                                    <p>Verzeichnis '${path}' wurde erfolgreich gemountet und für Aethel freigegeben.</p>
                                </div>
                            `);
                        }
                    }
                } catch (e) {
                    console.error("Failed to mount folder on new session", e);
                }
            }
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

    // Wiring for FULL MACHINE Mode (Vollautonomie)
    const elFullMachineRow = document.getElementById("sidebar-full-machine-row");
    const elFullMachineStatus = document.getElementById("full-machine-status");
    const elFullMachineModal = document.getElementById("full-machine-modal");
    const btnFullMachineConfirm = document.getElementById("full-machine-btn-confirm");
    const btnFullMachineCancel = document.getElementById("full-machine-btn-cancel");

    if (elFullMachineRow) {
        elFullMachineRow.addEventListener("click", () => {
            if (!state.isFullAutonomy) {
                // Open warning modal
                if (elFullMachineModal) elFullMachineModal.classList.remove("hidden");
            } else {
                // Disable immediately
                state.isFullAutonomy = false;
                if (elFullMachineStatus) {
                    elFullMachineStatus.textContent = "DISABLED";
                    elFullMachineStatus.style.color = "var(--vgt-red)";
                }
                import('./modules/voice.js').then(m => m.speak("Vollautonomer Modus deaktiviert. Sicherheitsüberwachung aktiv."));
            }
        });
    }

    if (btnFullMachineConfirm) {
        btnFullMachineConfirm.addEventListener("click", () => {
            state.isFullAutonomy = true;
            if (elFullMachineStatus) {
                elFullMachineStatus.textContent = "ENABLED";
                elFullMachineStatus.style.color = "var(--vgt-green)";
            }
            if (elFullMachineModal) elFullMachineModal.classList.add("hidden");
            import('./modules/voice.js').then(m => m.speak("Vollautonomer Modus aktiv. Der Kernel unterliegt direkter KI-Initiative."));
        });
    }

    if (btnFullMachineCancel) {
        btnFullMachineCancel.addEventListener("click", () => {
            if (elFullMachineModal) elFullMachineModal.classList.add("hidden");
        });
    }

    // Startup Warning Modal wiring
    const btnWarningAccept = document.getElementById("startup-warning-btn-accept");
    const warningModal = document.getElementById("startup-warning-modal");
    if (btnWarningAccept && warningModal) {
        btnWarningAccept.addEventListener("click", () => {
            warningModal.classList.add("hidden");
            import('./modules/voice.js').then(m => m.speak("Core initialisiert. Willkommen bei Äthel."));
        });
    }

    // Wire donation modal
    const btnDonation = document.getElementById("btn-donation");
    const modalDonation = document.getElementById("donation-modal");
    const btnDonationClose = document.getElementById("donation-btn-close");

    if (btnDonation && modalDonation) {
        btnDonation.addEventListener("click", () => {
            modalDonation.classList.remove("hidden");
        });
    }

    if (btnDonationClose && modalDonation) {
        btnDonationClose.addEventListener("click", () => {
            modalDonation.classList.add("hidden");
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
