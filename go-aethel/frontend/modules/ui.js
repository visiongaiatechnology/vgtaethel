import { state } from './state.js';
import * as api from './api.js';
import { refreshVoiceHealthHUD, speak } from './voice.js';
import { fetchActiveLeasesList, fetchSecurityAuditTrail, fetchKernelLogs } from './security.js';
import { fetchMemoriesList, updateMemoryCount } from './memory.js';
import { fetchTasksQueue, fetchKernelTasks } from './tasks.js';
import { loadChatHistory } from './chat.js';

export function switchMode(mode) {
    if (!state.views[mode]) return;
    
    Object.keys(state.views).forEach(key => {
        if (state.views[key]) state.views[key].classList.add("hidden");
        if (state.navButtons[key]) state.navButtons[key].classList.remove("active");
    });
    
    state.views[mode].classList.remove("hidden");
    if (state.navButtons[mode]) state.navButtons[mode].classList.add("active");
    
    const label = document.getElementById("current-mode-label");
    if (label) {
        label.textContent = mode.toUpperCase() + " MODE";
    }

    const prevSphereActive = state.isSphereActive;
    state.isSphereActive = (mode === "sphere");
    
    if (prevSphereActive && !state.isSphereActive) {
        import('./voice.js').then(m => {
            try { m.endWakeSession(); } catch(e) {}
        });
    }

    if (mode === "security") {
        fetchActiveLeasesList();
        fetchSecurityAuditTrail();
    } else if (mode === "memory") {
        fetchMemoriesList();
        import('./secrets.js').then(m => m.fetchSecretsList());
    } else if (mode === "personal") {
        import('./personal_mode.js').then(m => m.loadPersonalMode());
    } else if (mode === "sphere") {
        // Auto-activate voice call if not muted and not already active
        if (!state.isVoiceMuted && !state.isVoiceCallActive) {
            const btnVoiceLink = document.getElementById("btn-voice-link");
            if (btnVoiceLink) btnVoiceLink.click();
        }
        // Force active wake session inside the sphere!
        import('./voice.js').then(m => {
            if (state.isVoiceCallActive && !state.isWakeSessionActive) {
                try { m.activateWakeSession(); } catch(e) {}
            }
        });
    } else if (mode === "tasks") {
        fetchTasksQueue();
    } else if (mode === "core") {
        fetchKernelLogs();
        refreshVoiceHealthHUD();
    } else if (mode === "settings") {
        import('./settings.js').then(m => m.loadSettingsStatus());
    } else if (mode === "personas") {
        import('./settings.js').then(m => m.loadCustomPersonasSettings());
    } else if (mode === "archive") {
        import('./chat.js').then(m => m.loadSessionsList());
    }
}
window.switchMode = switchMode;

export async function checkSystemStatus() {
    const elWizard = document.getElementById("setup-wizard");
    const elAppContainer = document.getElementById("app-container");
    const elSphereStatus = document.getElementById("sphere-hud-status");

    try {
        const data = await api.checkSystemStatus();
        
        if (data.status === "SETUP_REQUIRED") {
            elWizard.classList.remove("hidden");
            elAppContainer.classList.add("hidden");
            
            // Check if Ollama is running and show/hide local AI button
            const elBtnUseLocal = document.getElementById("btn-use-local");
            if (elBtnUseLocal) {
                if (data.ollama_ready === "true") {
                    elBtnUseLocal.classList.remove("hidden");
                } else {
                    elBtnUseLocal.classList.add("hidden");
                }
            }
        } else {
            elWizard.classList.add("hidden");
            elAppContainer.classList.remove("hidden");
            state.hasOpenAI = (data.openai_ready === "true");
            if (elSphereStatus) {
                elSphereStatus.textContent = state.hasOpenAI ? "AETHEL CORE // ONYX ACTIVE" : "AETHEL CORE // BROWSER VOICE ACTIVE";
            }
            
            // Auto OS detection integration
            state.os = data.os || "windows";
            updateSystemProtocolWithOS();

            await loadModels();
            await loadVoices();
            updateMemoryCount();
            await loadChatHistory();
        }
    } catch (e) {
        console.error("System health check failed", e);
        showSetupError("Verbindung zum Core fehlgeschlagen.");
    }
}

export function updateSystemProtocolWithOS() {
    const detectedOS = state.os || "windows";
    const osUpper = detectedOS.toUpperCase();
    
    let osContext = `\n\nOPERATING SYSTEM CONTEXT:\n`;
    osContext += `- Der Host-Computer läuft unter dem Betriebssystem: ${osUpper}\n`;
    
    if (detectedOS === "windows") {
        osContext += `- Nutze Windows-spezifische Befehle (z.B. PowerShell-Cmdlets, cmd.exe, dir, Set-Location).\n`;
        osContext += `- Wenn du PowerShell-Befehle oder Skripte über sys_exec_cmd ausführen willst, verwende 'powershell' als Befehl und das Skript/die Befehlskette als Argument (z.B. args: ["-NoProfile", "-NonInteractive", "-Command", "..."]).\n`;
        osContext += `- Pfade nutzen Backslashes (\\).\n`;
    } else if (detectedOS === "darwin") {
        osContext += `- Der Host-Computer ist ein macOS (Darwin) System.\n`;
        osContext += `- Nutze Unix/macOS-spezifische Befehle (bash/sh, ls, cd, open).\n`;
        osContext += `- Pfade nutzen Vorwärts-Slashes (/).\n`;
    } else {
        osContext += `- Der Host-Computer ist ein Linux-System.\n`;
        osContext += `- Nutze Linux-Befehle (bash/sh, ls, cd, xdotool).\n`;
        osContext += `- Pfade nutzen Vorwärts-Slashes (/).\n`;
    }
    
    if (!state.originalProtocol) {
        state.originalProtocol = state.VGT_SYSTEM_PROTOCOL;
    }
    state.VGT_SYSTEM_PROTOCOL = state.originalProtocol + osContext;
    console.log(`[OS DETECTED] System Protocol updated for OS: ${osUpper}`);
}

export async function loadModels() {
    const elModelDropdown = document.getElementById("model-dropdown");
    try {
        const data = await api.getModels();
        const models = Array.isArray(data?.models) ? data.models.filter(model => model && typeof model.id === "string" && model.id.trim()) : [];
        if (!elModelDropdown) return;
        if (models.length === 0) {
            elModelDropdown.replaceChildren();
            const option = document.createElement("option");
            option.textContent = "KEINE MODELLE IN DER REGISTRY";
            option.disabled = true;
            option.selected = true;
            elModelDropdown.appendChild(option);
            return;
        }

        const rememberedModel = localStorage.getItem("aethel_selected_model") || state.currentModel;
        const modelIds = models.map(model => model.id);
        state.currentModel = modelIds.includes(rememberedModel) ? rememberedModel : models[0].id;
        elModelDropdown.replaceChildren();

        const groups = new Map();
        for (const model of models) {
            const provider = String(model.provider || "Andere");
            if (!groups.has(provider)) groups.set(provider, []);
            groups.get(provider).push(model);
        }
        const providerOrder = ["DeepSeek", "Groq", "OpenAI", "Gemini", "Claude", "Ollama", "Andere"];
        const providers = [...providerOrder.filter(provider => groups.has(provider)), ...[...groups.keys()].filter(provider => !providerOrder.includes(provider)).sort((a, b) => a.localeCompare(b))];
        for (const provider of providers) {
            const groupEl = document.createElement("optgroup");
            groupEl.label = provider.toUpperCase();
            for (const model of groups.get(provider)) {
                const option = document.createElement("option");
                option.value = model.id;
                option.textContent = model.name || model.id;
                option.selected = model.id === state.currentModel;
                groupEl.appendChild(option);
            }
            elModelDropdown.appendChild(groupEl);
        }
        elModelDropdown.value = state.currentModel;
        state.currentModel = elModelDropdown.value || models[0].id;
        updateLiveOperatorVisibility();
        if (elModelDropdown.dataset.modelListenerBound !== "true") {
            elModelDropdown.dataset.modelListenerBound = "true";
            elModelDropdown.addEventListener("change", event => {
                state.currentModel = event.target.value;
                localStorage.setItem("aethel_selected_model", state.currentModel);
                updateLiveOperatorVisibility();
                if (state.currentSessionId) localStorage.setItem('model_' + state.currentSessionId, state.currentModel);
            });
        }
    } catch (e) {
        console.error("Failed to load models", e);
        if (elModelDropdown) {
            elModelDropdown.replaceChildren();
            const option = document.createElement("option");
            option.textContent = "CORE NICHT ERREICHBAR";
            option.disabled = true;
            option.selected = true;
            elModelDropdown.appendChild(option);
        }
    }
}

export async function loadVoices() {
    const elVoiceDropdown = document.getElementById("voice-dropdown");
    try {
        const data = await api.getVoices();
        
        if (state.synth) {
            const browserVoices = state.synth.getVoices().filter(v => v.lang.startsWith("de"));
            browserVoices.forEach(bv => {
                const browserId = "browser:" + bv.name;
                if (!data.some(d => d.id === browserId)) {
                    data.push({
                        id: browserId,
                        name: `${bv.name.replace("Microsoft ", "").replace("Google ", "")} (Browser)`,
                        type: "browser",
                        gender: bv.name.toLowerCase().includes("female") || bv.name.toLowerCase().includes("hedda") || bv.name.toLowerCase().includes("katja") ? "weiblich" : "männlich"
                    });
                }
            });
        }
        
        if (elVoiceDropdown) {
            elVoiceDropdown.innerHTML = "";
            
            // Prevent selected voice from resetting on startup if Edge online voices are still loading
            const hasVoice = data.some(v => v.id === state.currentVoice);
            if (!hasVoice && state.currentVoice && state.currentVoice !== "onyx") {
                const opt = document.createElement("option");
                opt.value = state.currentVoice;
                let displayName = state.currentVoice.replace("browser:", "").replace("Microsoft ", "").replace("Google ", "") + " (Lade...)";
                opt.textContent = displayName;
                opt.selected = true;
                elVoiceDropdown.appendChild(opt);
            }
            
            data.forEach(v => {
                const opt = document.createElement("option");
                opt.value = v.id;
                opt.textContent = v.name;
                if (v.id === state.currentVoice) {
                    opt.selected = true;
                }
                elVoiceDropdown.appendChild(opt);
            });
            
            // Remove old change listener and bind new one
            const fresh = elVoiceDropdown.cloneNode(true);
            elVoiceDropdown.parentNode.replaceChild(fresh, elVoiceDropdown);
            fresh.addEventListener("change", (e) => {
                state.currentVoice = e.target.value;
                localStorage.setItem("aethel_voice", state.currentVoice);
            });
        }
    } catch (e) {
        console.error("Failed to load voices", e);
        if (elVoiceDropdown) {
            elVoiceDropdown.replaceChildren();
            const option = document.createElement("option");
            option.textContent = "VOICE CORE NICHT ERREICHBAR";
            option.disabled = true;
            option.selected = true;
            elVoiceDropdown.appendChild(option);
        }
    }
}

export async function handleSetupSubmit() {
    const elApiKey = document.getElementById("api-key");
    const elDeepSeekKey = document.getElementById("deepseek-api-key");
    const elOpenAiApiKey = document.getElementById("openai-api-key");
    const elGeminiApiKey = document.getElementById("gemini-api-key");
    const elClaudeApiKey = document.getElementById("claude-api-key");
    const elBtnInitiate = document.getElementById("btn-initiate");

    const key = elApiKey ? elApiKey.value.trim() : "";
    const deepseekKey = elDeepSeekKey ? elDeepSeekKey.value.trim() : "";
    const openaiKey = elOpenAiApiKey ? elOpenAiApiKey.value.trim() : "";
    const geminiKey = elGeminiApiKey ? elGeminiApiKey.value.trim() : "";
    const claudeKey = elClaudeApiKey ? elClaudeApiKey.value.trim() : "";

    // At least one AI key required
    const hasGroq = key.startsWith('gsk_');
    const hasDeepSeek = deepseekKey.startsWith('sk-');
    const hasOpenAI = openaiKey.startsWith('sk-');
    const hasGemini = geminiKey.startsWith('AIza');
    const hasClaude = claudeKey.startsWith('sk-ant-');
    if (!hasGroq && !hasDeepSeek && !hasOpenAI && !hasGemini && !hasClaude) {
        showSetupError("Mindestens einen gültigen Key eingeben (Groq, DeepSeek, OpenAI, Gemini oder Claude).");
        return;
    }

    elBtnInitiate.disabled = true;
    elBtnInitiate.querySelector("span").textContent = "BOOTING CORE...";

    try {
        const data = await api.submitSetup(key, openaiKey, deepseekKey, geminiKey, claudeKey);

        if (data.status === "success") {
            document.getElementById("setup-error").classList.add("hidden");
            setTimeout(async () => {
                await checkSystemStatus();
                elBtnInitiate.disabled = false;
                elBtnInitiate.querySelector("span").textContent = "SYSTEM INITIALISIEREN";
            }, 1200);
        } else {
            showSetupError(data.message);
            elBtnInitiate.disabled = false;
            elBtnInitiate.querySelector("span").textContent = "SYSTEM INITIALISIEREN";
        }
    } catch (e) {
        showSetupError("Keine Verbindung zum Core-Server.");
        elBtnInitiate.disabled = false;
        elBtnInitiate.querySelector("span").textContent = "SYSTEM INITIALISIEREN";
    }
}

export function showSetupError(msg) {
    const elSetupError = document.getElementById("setup-error");
    const elSetupErrorText = document.getElementById("setup-error-text");
    elSetupErrorText.textContent = msg;
    elSetupError.classList.remove("hidden");
}

export function isVisionModel(modelId) {
    if (!modelId) return false;
    const id = modelId.toLowerCase();
    return id.includes("scout") || 
           id.includes("llama-4") || 
           id.includes("qwen3.6") || 
           id.includes("deepseek") || 
           id.includes("vision") ||
           id.includes("gpt-5") ||
           id.includes("gemini-3") ||
           id.includes("claude-sonnet") ||
           id.includes("claude-opus") ||
           id.includes("claude-fable");
}

export function updateLiveOperatorVisibility() {
    const btn = document.getElementById("nav-btn-control");
    if (!btn) return;

    const isVision = isVisionModel(state.currentModel);
    if (isVision) {
        btn.style.display = "flex";
    } else {
        btn.style.display = "none";
        // If we are currently in the Live Operator view and switch to a non-vision model,
        // force jump back to the safe chat panel.
        const controlView = document.getElementById("view-control");
        if (controlView && !controlView.classList.contains("hidden")) {
            switchMode("chat");
        }
    }
}

export async function handleLocalOnlySetup() {
    const elBtnUseLocal = document.getElementById("btn-use-local");
    const label = elBtnUseLocal ? elBtnUseLocal.querySelector("span") : null;
    const oldText = label ? label.textContent : "";

    if (elBtnUseLocal && label) {
        elBtnUseLocal.disabled = true;
        label.textContent = "BOOTING LOKAL...";
    }
    
    try {
        const data = await api.submitSetup("local", "", ""); // Submit setup with key: "local"
        if (data.status === "success") {
            const errBox = document.getElementById("setup-error");
            if (errBox) errBox.classList.add("hidden");
            setTimeout(async () => {
                await checkSystemStatus();
                if (elBtnUseLocal && label) {
                    elBtnUseLocal.disabled = false;
                    label.textContent = oldText;
                }
            }, 1200);
        } else {
            showSetupError(data.message);
            if (elBtnUseLocal && label) {
                elBtnUseLocal.disabled = false;
                label.textContent = oldText;
            }
        }
    } catch(e) {
        console.error("Local setup request failed", e);
        showSetupError("Keine Verbindung zum Core-Server.");
        if (elBtnUseLocal && label) {
            elBtnUseLocal.disabled = false;
            label.textContent = oldText;
        }
    }
}

export async function refreshAPICosts() {
    const elCostsToday = document.getElementById("hud-costs-today");
    const elCostsMonth = document.getElementById("hud-costs-month");
    if (!elCostsToday || !elCostsMonth) return;
    try {
        const data = await api.getAPICosts();
        if (data) {
            elCostsToday.textContent = `$${(data.today || 0).toFixed(4)}`;
            elCostsMonth.textContent = `$${(data.month || 0).toFixed(4)}`;
        }
    } catch (e) {
        console.error("Failed to refresh API costs", e);
    }
}

export function getCombinedSystemPrompt() {
    const activePersonaId = localStorage.getItem("aethel_active_persona") || "default";
    let prompt = state.VGT_SYSTEM_PROTOCOL.trim();
    if (activePersonaId !== "default" && window.customPersonasList) {
        const p = window.customPersonasList.find(x => x.id === activePersonaId);
        if (p) {
            prompt = `SYSTEM IDENTITY: VGT AETHEL [CUSTOM PERSONA: ${p.name.toUpperCase()}]\nSTATUS: ONLINE\nMODE: SOVEREIGN\n\nCUSTOM PERSONA DIRECTIVES:\n${p.system_prompt}\n\n=========================================\n\n` + prompt;
        }
    }
    return prompt;
}

export function refreshPersonasDropdowns() {
    const sidebarDropdown = document.getElementById("persona-dropdown");
    if (!sidebarDropdown) return;

    const personas = window.customPersonasList || [];

    let sidebarHtml = `<option value="default">Aethel Standard-Verhalten</option>`;
    personas.forEach(p => {
        sidebarHtml += `<option value="${p.id}">${p.name.toUpperCase()}</option>`;
    });
    sidebarDropdown.innerHTML = sidebarHtml;

    const activePersonaId = localStorage.getItem("aethel_active_persona") || "default";
    sidebarDropdown.value = activePersonaId;

    document.querySelectorAll(".agent-role-persona-select").forEach(select => {
        const role = select.id.replace("agent-persona-", "");
        let selectHtml = `<option value="default">Standard-${role.toUpperCase()}</option>`;
        personas.forEach(p => {
            selectHtml += `<option value="${p.id}">${p.name.toUpperCase()}</option>`;
        });
        select.innerHTML = selectHtml;

        const savedVal = localStorage.getItem(`aethel_agent_persona_${select.id}`) || "default";
        select.value = savedVal;
    });
}

window.refreshPersonasDropdowns = refreshPersonasDropdowns;

// Bind selection listener immediately
setTimeout(() => {
    const el = document.getElementById("persona-dropdown");
    if (el) {
        el.addEventListener("change", (e) => {
            localStorage.setItem("aethel_active_persona", e.target.value);
        });
    }
}, 100);
