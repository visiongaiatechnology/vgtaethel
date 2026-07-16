// VGT AETHEL // MAIN ENTRYPOINT MODULE (ES6)

import { state } from './modules/state.js';
import { switchMode, checkSystemStatus, handleSetupSubmit, handleLocalOnlySetup, refreshAPICosts } from './modules/ui.js';
import { sendMessage, startNewSession, loadSession, deleteSession } from './modules/chat.js';
import { setupSpeechRecognition, stopSpeaking, resetMicButton, startWakeWordListener, stopWakeWordListener, endWakeSession } from './modules/voice.js';
import { refreshSecurityHUD, fetchActiveLeasesList, fetchSecurityAuditTrail } from './modules/security.js';
import { setupMemoryUIEvents, updateMemoryCount } from './modules/memory.js';
import { setupSecretsUIEvents } from './modules/secrets.js';
import { setupControlUIEvents } from './modules/control.js';
import { setupTasksUIEvents, fetchKernelTasks } from './modules/tasks.js';
import { setupSettingsUIEvents, loadCustomPersonasSettings } from './modules/settings.js';
import { startGlobalRunApprovalMonitor } from './modules/run_approval_monitor.js';
import { startGlobalIntelligenceAlertMonitor } from './modules/intelligence_alert_monitor.js';
import { setupUIModernization } from './modules/ui_modernization.js';
import { setupI18n } from './modules/i18n.js';
import { speakPersonalizedStartupGreeting } from './modules/startup_greeting.js';

// Initialize Application

async function init() {
    // Map view panels
    state.views = {
        core: document.getElementById("view-core"),
        chat: document.getElementById("view-chat"),
        personas: document.getElementById("view-personas"),
        agent: document.getElementById("view-agent"),
        control: document.getElementById("view-control"),
        security: document.getElementById("view-security"),
        memory: document.getElementById("view-memory"),
        personal: document.getElementById("view-personal"),
        sphere: document.getElementById("view-sphere"),
        tasks: document.getElementById("view-tasks"),
        settings: document.getElementById("view-settings"),
        archive: document.getElementById("view-archive"),
        globalWatch: document.getElementById("view-global-watch"),
        case: document.getElementById("view-case"),
    };

    // Map navigation buttons
    state.navButtons = {
        core: document.getElementById("nav-btn-core"),
        chat: document.getElementById("nav-btn-chat"),
        personas: document.getElementById("nav-btn-personas"),
        agent: document.getElementById("nav-btn-agent"),
        control: document.getElementById("nav-btn-control"),
        security: document.getElementById("nav-btn-security"),
        memory: document.getElementById("nav-btn-memory"),
        personal: document.getElementById("nav-btn-personal"),
        sphere: document.getElementById("nav-btn-sphere"),
        tasks: document.getElementById("nav-btn-tasks"),
        settings: document.getElementById("nav-btn-settings"),
        archive: document.getElementById("nav-btn-archive"),
        globalWatch: document.getElementById("nav-btn-global-watch"),
        case: document.getElementById("nav-btn-case"),
    };

    // Setup and trigger splash screen status updates
    const splash = document.getElementById("startup-splash-screen");
    const splashStatus = document.getElementById("splash-status-text");
    const splashStartedAt = Date.now();
    const minimumSplashDuration = 3200;
    
    let splashDismissed = false;
    const dismissSplash = () => {
        if (splashDismissed) return;
        splashDismissed = true;
        if (splashStatus) splashStatus.textContent = "SYSTEM BOOT READY // OPERATOR LINK ESTABLISHED";
        const remaining = Math.max(0, minimumSplashDuration - (Date.now() - splashStartedAt));
        setTimeout(() => {
            if (splash) {
                splash.classList.add("startup-splash-exit");
                splash.style.opacity = "0";
                setTimeout(() => splash.remove(), 800);
            }
        }, remaining + 450);
    };

    if (splash && splashStatus) {
        setTimeout(() => { if (!splashDismissed) splashStatus.textContent = "INITIALIZING POLICY LEASES..."; }, 650);
        setTimeout(() => { if (!splashDismissed) splashStatus.textContent = "VERIFYING SECURITY SHIELD..."; }, 1350);
        setTimeout(() => { if (!splashDismissed) splashStatus.textContent = "MOUNTING NEXUS MEMORY..."; }, 2100);
        setTimeout(() => { if (!splashDismissed) splashStatus.textContent = "SYNCHRONIZING OPERATOR INTERFACE..."; }, 2800);
    }

    try {
        setupI18n();
        setupViewNavigation();
        setupSpeechRecognition();
        setupEventListeners();
        setupMemoryUIEvents();
        setupSecretsUIEvents();
        setupTasksUIEvents();
        setupControlUIEvents();
        setupSettingsUIEvents();
        setupUIModernization();
        // Collapsible sidebar sections
        document.querySelectorAll(".nav-section-header").forEach(header => {
            const section = header.getAttribute("data-section");
            const content = document.getElementById(`nav-section-${section}`);
            
            // Restore state from localStorage
            const isCollapsed = localStorage.getItem(`aethel_sidebar_collapsed_${section}`) === "true";
            if (isCollapsed) {
                header.classList.add("collapsed");
                content?.classList.add("collapsed");
            }
            
            header.addEventListener("click", () => {
                header.classList.toggle("collapsed");
                content?.classList.toggle("collapsed");
                localStorage.setItem(`aethel_sidebar_collapsed_${section}`, header.classList.contains("collapsed"));
            });
        });

        // Initialize OSINT Global Watch View
        import('./modules/osint_watch.js').then(m => m.initGlobalWatch()).catch(e => console.error("Global Watch initialization failed", e));
        import('./modules/case_workspace.js').then(m => m.initCaseWorkspace()).catch(e => console.error("Case Workspace initialization failed", e));

        // Wire UI events early (no data fetch yet — hydration runs after core READY).
        import('./modules/personal_mode.js').then(m => m.setupPersonalModeUIEvents()).catch(error => console.error('Personal Mode disabled during boot', error));
        import('./modules/agent_builder.js').then(m => m.setupAgentBuilder()).catch(error => console.error('Agent Builder disabled during boot', error));
        import('./modules/sphere.js').then(m => m.setupSphereWorkspace()).catch(error => console.error('Sphere Workspace disabled during boot', error));

        await checkSystemStatus();
        // After core is READY: hydrate Personal Core + Neural Core greeting so
        // saved profile/config are not empty until the user opens the view.
        await import('./modules/personal_mode.js')
            .then(m => m.hydratePersonalModeAtBoot())
            .catch(error => console.error('Personal Core boot hydrate failed', error));
        await refreshAPICosts().catch(error => console.error('API costs unavailable at boot', error));
        await loadCustomPersonasSettings().catch(error => console.error('Personas unavailable at boot', error));
        window.setTimeout(() => {
            import('./modules/startup_briefing.js')
                .then(module => module.runConfiguredStartupBriefing())
                .catch(error => console.error('Startup briefing unavailable', error));
        }, 1800);

        // Update version badge dynamically from backend ProductVersion (1.0.0-beta.2 -> BETA V2).
        // Robust retries to ensure visibility even if bindings load late (addresses prior "not visible" reports).
        // BETA V2 · PRODUCTION CANDIDATE
        function updateVersionBadge(attempt = 0) {
            const maxAttempts = 6;
            const badge = document.querySelector('.release-badge');
            if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetVersion) {
                window.go.main.App.GetVersion().then(v => {
                    if (badge && v) {
                        if (v.includes('beta')) {
                            const betaPart = v.split('-beta.')[1] || '';
                            badge.textContent = betaPart ? 'BETA V' + betaPart : 'BETA V2';
                        } else {
                            badge.textContent = v;
                        }
                        // Also sync status-sub if present
                        const sub = document.querySelector('.status-sub');
                        if (sub && sub.textContent.includes('BETA')) {
                            const release = v.includes('beta') ? (v.split('-beta.')[1] ? 'BETA V' + v.split('-beta.')[1] : 'BETA V2') : v;
                            const mode = document.createElement('span');
                            mode.id = 'current-mode-label';
                            mode.className = 'cyan-text';
                            mode.textContent = 'NEURAL CORE';
                            sub.replaceChildren(document.createTextNode(release + ' :: '), mode);
                        }
                    }
                }).catch(() => {
                    if (attempt < maxAttempts) setTimeout(() => updateVersionBadge(attempt + 1), 250 * (attempt + 1));
                });
                return;
            }
            if (attempt < maxAttempts) {
                setTimeout(() => updateVersionBadge(attempt + 1), 250 * (attempt + 1));
            }
        }
        updateVersionBadge();
        // Additional late retries (more attempts for robustness on slow binding)
        setTimeout(() => updateVersionBadge(1), 800);
        setTimeout(() => updateVersionBadge(2), 1600);
        setTimeout(() => updateVersionBadge(3), 2500);

        // Default view is core HUD
        switchMode("core");
    } catch (e) {
        console.error("Boot execution failed", e);
    } finally {
        if (splashStatus) splashStatus.textContent = "SYSTEM BOOT READY // OPERATOR LINK ESTABLISHED";
    }

    // Freigaben sind ein globaler Sicherheitszustand, kein Run-Center-Detail.
    startGlobalRunApprovalMonitor();
    // High-confidence Global Watch observations must be visible from every view.
    startGlobalIntelligenceAlertMonitor();

    // Periodic polling for status bar HUD and active viewport logs
    setInterval(() => {
        refreshSecurityHUD();
        refreshAPICosts();
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
    const elBtnCodeCartography = document.getElementById("btn-code-cartography");
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
    const elBtnUseLocal = document.getElementById("btn-use-local");
    if (elBtnUseLocal) {
        elBtnUseLocal.addEventListener("click", handleLocalOnlySetup);
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
        // Submit on Enter without Shift, insert newline on Shift+Enter
        elUserInput.addEventListener("keydown", (e) => {
            if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                sendMessage();
                // Reset height back to default after sending
                elUserInput.style.height = "52px";
            }
        });

        // Auto-expand/shrink height based on input content length
        elUserInput.addEventListener("input", () => {
            elUserInput.style.height = "52px"; // reset
            const scrollHeight = elUserInput.scrollHeight;
            if (scrollHeight > 52) {
                // Limit maximum expanded height to 180px
                elUserInput.style.height = Math.min(scrollHeight, 180) + "px";
            }
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
                            args: { path: path }
                        })
                    });
                    const data = await res.json();
                    if (data.status === "success") {
                        const elChatOutput = document.getElementById("chat-output");
                        if (elChatOutput) {
                            const box = document.createElement("div");
                            box.className = "system-message font-mono";
                            box.style.cssText = "color: var(--vgt-cyan); border-color: var(--vgt-cyan); background: rgba(0, 240, 255, 0.05); margin-top: 10px;";
                            const title = document.createElement("p");
                            title.textContent = "[ORDNER FREIGEGEBEN]";
                            const detail = document.createElement("p");
                            detail.textContent = `Verzeichnis '${path}' wurde erfolgreich gemountet und fuer Aethel freigegeben.`;
                            box.replaceChildren(title, detail);
                            elChatOutput.appendChild(box);
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
                
                voiceModule.speak("Wake-Modus aktiv.");
                setTimeout(() => {
                    if (state.isVoiceCallActive && !state.isWakeSessionActive) {
                        voiceModule.startWakeWordListener();
                    }
                }, 800);
            } else {
                elBtnVoiceLink.classList.remove("active");
                if (state.activeAudio) state.activeAudio.pause();
                if (state.synth) state.synth.cancel();
                
                voiceModule.stopWakeWordListener();
                voiceModule.endWakeSession();
                
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
            }
        });
    }

    if (elBtnCodeCartography && elUserInput) {
        elBtnCodeCartography.addEventListener("click", () => {
            const prefix = "Erstelle eine Code Kartografie für den Ordner ";
            if (!elUserInput.value.trim()) elUserInput.value = prefix;
            elUserInput.focus();
            elUserInput.setSelectionRange(elUserInput.value.length, elUserInput.value.length);
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
        btnWarningAccept.addEventListener("click", async () => {
            warningModal.classList.add("hidden");
            await speakPersonalizedStartupGreeting();
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

    // Neural Core Evaluation wiring
    const ncWorldText = document.getElementById("nc-eval-world-text");
    const ncLocalText = document.getElementById("nc-eval-local-text");
    const ncTimestamp = document.getElementById("nc-eval-timestamp");
    const ncTriggerBtn = document.getElementById("nc-eval-trigger-btn");

    async function loadNeuralCoreEvaluation(forceUpdate = false) {
        if (!ncWorldText || !ncLocalText) return;
        try {
            if (forceUpdate) {
                ncWorldText.textContent = "GENERATING SITUATION MATRIX...";
                ncLocalText.textContent = "COMPUTING PERSONAL CORRELATIONS...";
                if (ncTriggerBtn) {
                    ncTriggerBtn.disabled = true;
                    ncTriggerBtn.textContent = "GENERATING...";
                }
            }
            const method = forceUpdate ? 'POST' : 'GET';
            const res = await fetch(`${state.API_BASE}/v1/intelligence/evaluation`, { method });
            if (!res.ok) throw new Error(await res.text());
            const data = await res.json();
            
            ncWorldText.textContent = data.world_state || "Keine globalen Daten verzeichnet.";
            ncLocalText.textContent = data.local_state || "Keine lokalen Auswirkungen berechnet.";
            
            const dateStr = data.last_updated && !data.last_updated.startsWith("0001-") 
                ? new Date(data.last_updated).toLocaleTimeString() 
                : "INITIALIZING";
            ncTimestamp.textContent = `LAST EVALUATED: ${dateStr}`;
            if (forceUpdate) {
                showAethelToast("Neural Core Lage-Analyse erfolgreich aktualisiert!", "success");
            }
        } catch (err) {
            console.error("Failed to load situation report:", err);
            if (forceUpdate) {
                showAethelToast("Lage-Analyse fehlgeschlagen: " + err.message, "error");
            }
        } finally {
            if (ncTriggerBtn) {
                ncTriggerBtn.disabled = false;
                ncTriggerBtn.textContent = "LAGE-ANALYSE AKTUALISIEREN (PUSH)";
            }
        }
    }

    if (ncTriggerBtn) {
        ncTriggerBtn.addEventListener("click", () => loadNeuralCoreEvaluation(true));
    }
    // Load initial
    loadNeuralCoreEvaluation(false);
}

function legacyWakeWordListenerDisabled() {
    return;
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
