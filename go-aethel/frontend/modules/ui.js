import { state } from './state.js';
import * as api from './api.js';
import { refreshVoiceHealthHUD, speak } from './voice.js';
import { fetchActiveLeasesList, fetchSecurityAuditTrail, fetchKernelLogs } from './security.js';
import { fetchMemoriesList, updateMemoryCount } from './memory.js';
import { fetchTasksQueue, fetchKernelTasks } from './tasks.js';
import { loadChatHistory } from './chat.js';
import { currentLanguage } from './i18n.js';

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
        // Re-sync personal greeting when returning to Neural Core (no visit bump).
        import('./personal_mode.js').then(m => m.refreshNeuralCoreHome({ countVisit: false })).catch(() => {});
    } else if (mode === "settings") {
        import('./settings.js').then(m => m.loadSettingsStatus());
    } else if (mode === "personas") {
        import('./settings.js').then(m => m.loadCustomPersonasSettings());
    } else if (mode === "archive") {
        import('./chat.js').then(m => m.loadSessionsList());
    } else if (mode === "globalWatch") {
        import('./osint_watch.js').then(m => {
            m.refreshOSINTFeed("all");
            if (m.loadAndRenderRegionalRisks) m.loadAndRenderRegionalRisks();
            if (m.forceGlobeResize) {
                setTimeout(() => m.forceGlobeResize(), 50);
            }
        });
    } else if (mode === "case") {
        import('./case_workspace.js').then(m => m.refreshCaseWorkspace && m.refreshCaseWorkspace());
    }
}
window.switchMode = switchMode;

const CORE_READINESS_RETRY_DELAYS_MS = [250, 400, 650, 900, 1200, 1600, 2000, 2500];
const MODEL_REGISTRY_RETRY_DELAYS_MS = [250, 400, 650, 900, 1200, 1600, 2000, 2500];

function waitForBootstrap(delayMs) {
    return new Promise(resolve => window.setTimeout(resolve, delayMs));
}

export async function checkSystemStatus(retryCount = 0) {
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

            // Parallel registry/history loads — one slow endpoint must not leave the UI empty.
            await Promise.all([
                loadModels().catch(err => console.error("Model registry load failed", err)),
                loadVoices().catch(err => console.error("Voice registry load failed", err)),
                Promise.resolve(updateMemoryCount()).catch(() => {}),
            ]);
            await loadChatHistory().catch(err => console.error("Chat history bootstrap failed", err));
        }
    } catch (e) {
        console.error("System health check failed", e);
        if (retryCount < CORE_READINESS_RETRY_DELAYS_MS.length) {
            const delay = CORE_READINESS_RETRY_DELAYS_MS[retryCount];
            console.info(`Core noch nicht bereit; Readiness-Pruefung ${retryCount + 1}/${CORE_READINESS_RETRY_DELAYS_MS.length} in ${delay}ms.`);
            await waitForBootstrap(delay);
            return checkSystemStatus(retryCount + 1);
        }
        showSetupError("Verbindung zum Core fehlgeschlagen.");
        // Keep the direct binding as the terminal fallback after the bounded
        // readiness handshake. A transient 503 must never become persistent UI state.
        await loadModels().catch(() => {});
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

function applyModelsToDropdowns(models) {
    const elModelDropdown = document.getElementById("model-dropdown");
    const orchestratorDropdown = document.getElementById("orchestrator-model-dropdown");
    if (!elModelDropdown) return false;

	const clean = (Array.isArray(models) ? models : []).filter(model => model && typeof model.id === "string" && model.id.trim());
	state.modelRegistry = clean;
    if (clean.length === 0) {
		for (const dropdown of [elModelDropdown, orchestratorDropdown].filter(Boolean)) {
			dropdown.replaceChildren();
			const option = document.createElement("option");
			option.textContent = "KEIN KONFIGURIERTER PROVIDER / LOKALES MODELL";
			option.disabled = true;
			option.selected = true;
			dropdown.appendChild(option);
		}
		state.currentModel = '';
		state.orchestratorModel = '';
		configureReasoningDropdown();
		updateContextLimitLabel();
        return false;
    }

    const rememberedModel = localStorage.getItem("aethel_selected_model") || state.currentModel;
    const modelIds = clean.map(model => model.id);
    state.currentModel = modelIds.includes(rememberedModel) ? rememberedModel : clean[0].id;
    elModelDropdown.replaceChildren();

    const providerOrder = ["DeepSeek", "Groq", "OpenAI", "Gemini", "Claude", "Ollama", "Andere"];
    const providerRank = provider => {
        const index = providerOrder.indexOf(String(provider || "Andere"));
        return index < 0 ? providerOrder.length : index;
    };
    const orderedModels = [...clean].sort((left, right) => {
        const providerDifference = providerRank(left.provider) - providerRank(right.provider);
        if (providerDifference !== 0) return providerDifference;
        return String(left.name || left.id).localeCompare(String(right.name || right.id));
    });
	appendModelOptions(elModelDropdown, orderedModels, state.currentModel);
    elModelDropdown.value = state.currentModel;
    state.currentModel = elModelDropdown.value || clean[0].id;

    if (orchestratorDropdown) {
        const toolModels = orderedModels.filter(model => model.supports_tools !== false);
        const candidates = toolModels.length > 0 ? toolModels : orderedModels;
        const rememberedOrchestrator = localStorage.getItem('aethel_orchestrator_model') || state.orchestratorModel || state.currentModel;
        orchestratorDropdown.replaceChildren();
		appendModelOptions(orchestratorDropdown, candidates, rememberedOrchestrator);
        state.orchestratorModel = candidates.some(model => model.id === rememberedOrchestrator) ? rememberedOrchestrator : candidates[0].id;
        orchestratorDropdown.value = state.orchestratorModel;
        if (orchestratorDropdown.dataset.modelListenerBound !== 'true') {
            orchestratorDropdown.dataset.modelListenerBound = 'true';
			orchestratorDropdown.addEventListener('change', event => {
                state.orchestratorModel = event.target.value;
                localStorage.setItem('aethel_orchestrator_model', state.orchestratorModel);
			});
		}
	}
	configureReasoningDropdown();
	updateContextLimitLabel();

    updateLiveOperatorVisibility();
    if (elModelDropdown.dataset.modelListenerBound !== "true") {
        elModelDropdown.dataset.modelListenerBound = "true";
		elModelDropdown.addEventListener("change", event => {
            state.currentModel = event.target.value;
            localStorage.setItem("aethel_selected_model", state.currentModel);
            updateLiveOperatorVisibility();
			if (state.currentSessionId) localStorage.setItem('model_' + state.currentSessionId, state.currentModel);
			configureReasoningDropdown();
			updateContextLimitLabel();
		});
    }
    return true;
}

function applyModelRegistryPendingState() {
    const modelDropdown = document.getElementById("model-dropdown");
    const orchestratorDropdown = document.getElementById("orchestrator-model-dropdown");
    for (const dropdown of [modelDropdown, orchestratorDropdown].filter(Boolean)) {
        dropdown.replaceChildren();
        const option = document.createElement("option");
        option.textContent = "PROVIDER-REGISTRY WIRD SYNCHRONISIERT...";
        option.disabled = true;
        option.selected = true;
        dropdown.appendChild(option);
    }
}

function appendModelOptions(select, models, selectedID) {
	let activeProvider = '';
	for (const model of models) {
		const provider = String(model.provider || 'Andere').toUpperCase();
		if (provider !== activeProvider) {
			activeProvider = provider;
			const separator = document.createElement('option');
			separator.disabled = true;
			separator.className = 'model-provider-separator';
			separator.textContent = `──── ${provider} ────`;
			select.appendChild(separator);
		}
		const option = document.createElement('option');
		option.value = model.id;
		option.textContent = `  ${model.name || model.id}${model.discovered ? ' · LIVE' : ''}`;
		option.selected = model.id === selectedID;
		select.appendChild(option);
	}
}

function configureReasoningDropdown() {
	const dropdown = document.getElementById('reasoning-effort-dropdown');
	if (!dropdown) return;
	const model = state.modelRegistry.find(item => item.id === state.currentModel);
	const options = Array.isArray(model?.reasoning_options) ? model.reasoning_options : [];
	dropdown.replaceChildren();
	const automatic = document.createElement('option');
	automatic.value = 'auto';
	automatic.textContent = `AUTO${model?.default_reasoning ? ` · ${String(model.default_reasoning).toUpperCase()}` : ''}`;
	dropdown.appendChild(automatic);
	for (const effort of options) {
		const option = document.createElement('option');
		option.value = effort;
		option.textContent = String(effort).toUpperCase();
		dropdown.appendChild(option);
	}
	const storageKey = model?.id ? `aethel_reasoning_effort:${model.id}` : '';
	const remembered = (storageKey ? localStorage.getItem(storageKey) : '') || 'auto';
	dropdown.value = remembered === 'auto' || options.includes(remembered) ? remembered : 'auto';
	state.reasoningEffort = dropdown.value;
	dropdown.disabled = options.length === 0;
	if (dropdown.dataset.bound !== 'true') {
		dropdown.dataset.bound = 'true';
		dropdown.addEventListener('change', () => {
			state.reasoningEffort = dropdown.value;
			const activeModel = state.modelRegistry.find(item => item.id === state.currentModel);
			if (activeModel) localStorage.setItem(`aethel_reasoning_effort:${activeModel.id}`, state.reasoningEffort);
		});
	}
}

function updateContextLimitLabel() {
	const element = document.getElementById('context-utilization');
	const model = state.modelRegistry.find(item => item.id === state.currentModel);
	if (!element) return;
	if (!model) {
		element.textContent = '0 / —';
		return;
	}
	const limit = Number(model.context_tokens || 0);
	element.textContent = `0 / ${formatTokenLimit(limit)}`;
}

function formatTokenLimit(tokens) {
	if (!Number.isFinite(tokens) || tokens <= 0) return '—';
	if (tokens >= 1000000) return `${(tokens / 1000000).toFixed(tokens % 1000000 === 0 ? 0 : 2)}M`;
	return `${Math.round(tokens / 1024)}k`;
}

export async function loadModels(retryCount = 0) {
    try {
        const data = await api.getModels();
        const models = Array.isArray(data?.models) ? data.models.filter(model => model && typeof model.id === "string" && model.id.trim()) : [];
        if (models.length === 0) {
            if (retryCount < MODEL_REGISTRY_RETRY_DELAYS_MS.length) {
                applyModelRegistryPendingState();
                const delay = MODEL_REGISTRY_RETRY_DELAYS_MS[retryCount];
                console.info(`Provider-Registry noch leer; Synchronisierung ${retryCount + 1}/${MODEL_REGISTRY_RETRY_DELAYS_MS.length} in ${delay}ms.`);
                await waitForBootstrap(delay);
                return loadModels(retryCount + 1);
            }
            applyModelsToDropdowns([]);
            return;
        }
        applyModelsToDropdowns(models);
    } catch (e) {
        console.error("Failed to load models", e);
        if (retryCount < MODEL_REGISTRY_RETRY_DELAYS_MS.length) {
            applyModelRegistryPendingState();
            const delay = MODEL_REGISTRY_RETRY_DELAYS_MS[retryCount];
            console.info(`Provider-Registry nicht erreichbar; Wiederholung ${retryCount + 1}/${MODEL_REGISTRY_RETRY_DELAYS_MS.length} in ${delay}ms.`);
            await waitForBootstrap(delay);
            return loadModels(retryCount + 1);
        }
        applyModelsToDropdowns([]);
    }
}

export async function loadVoices(retryCount = 0) {
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
        if (retryCount < 10) {
            console.log(`Retrying to load voices in 500ms (Attempt ${retryCount + 1}/10)...`);
            setTimeout(() => loadVoices(retryCount + 1), 500);
            return;
        }
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

export function refreshPersonasDropdowns() {
    const sidebarDropdown = document.getElementById("persona-dropdown");
    if (!sidebarDropdown) return;

    const personas = window.customPersonasList || [];

    replacePersonaOptions(sidebarDropdown, 'Aethel Standard-Verhalten', personas);

    const activePersonaId = localStorage.getItem("aethel_active_persona") || "default";
    sidebarDropdown.value = activePersonaId;

    document.querySelectorAll(".agent-role-persona-select").forEach(select => {
        const role = select.id.replace("agent-persona-", "");
        replacePersonaOptions(select, `Standard-${role.toUpperCase()}`, personas);

        const savedVal = localStorage.getItem(`aethel_agent_persona_${select.id}`) || "default";
        select.value = savedVal;
    });
}

function replacePersonaOptions(select, defaultLabel, personas) {
    const fragment = document.createDocumentFragment();
    const fallback = document.createElement('option');
    fallback.value = 'default';
    fallback.textContent = defaultLabel;
    fragment.appendChild(fallback);
    for (const persona of personas) {
        const option = document.createElement('option');
        option.value = String(persona.id || '').slice(0, 128);
        option.textContent = String(persona.name || 'Persona').slice(0, 120).toUpperCase();
        fragment.appendChild(option);
    }
    select.replaceChildren(fragment);
}

export function getCombinedSystemPrompt() {
    const activePersonaId = localStorage.getItem('aethel_active_persona') || 'default';
    const persona = activePersonaId === 'default'
        ? null
        : (window.customPersonasList || []).find(candidate => candidate.id === activePersonaId);
    const languageNames = { de: 'Deutsch', en: 'English', ru: 'Russian', es: 'Español' };
    const responseLanguage = languageNames[currentLanguage()] || languageNames.de;
    let prompt = `SYSTEM IDENTITY: VGT AETHEL\nMODE: SOVEREIGN, EVIDENCE-DRIVEN\nOPERATOR LANGUAGE CONTRACT: Antworte auf ${responseLanguage}. Wechsle die Sprache nur auf ausdrücklichen Wunsch. Erhalte Code, Eigennamen, Quellen und technische Bezeichner unverändert.`;
    if (persona) {
        const personaName = String(persona.name || 'CUSTOM').slice(0, 80).toUpperCase();
        const personaPrompt = String(persona.system_prompt || '').slice(0, 4000);
        prompt += `\nCUSTOM PERSONA: ${personaName}\n${personaPrompt}`;
    }
    return prompt;
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
