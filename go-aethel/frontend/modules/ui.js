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
    
    if (mode === "security") {
        fetchActiveLeasesList();
        fetchSecurityAuditTrail();
    } else if (mode === "memory") {
        fetchMemoriesList();
        import('./secrets.js').then(m => m.fetchSecretsList());
    } else if (mode === "tasks") {
        fetchTasksQueue();
    } else if (mode === "core") {
        fetchKernelLogs();
        refreshVoiceHealthHUD();
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
        } else {
            elWizard.classList.add("hidden");
            elAppContainer.classList.remove("hidden");
            state.hasOpenAI = (data.openai_ready === "true");
            if (elSphereStatus) {
                elSphereStatus.textContent = state.hasOpenAI ? "AETHEL CORE // ONYX ACTIVE" : "AETHEL CORE // BROWSER VOICE ACTIVE";
            }
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

export async function loadModels() {
    const elModelDropdown = document.getElementById("model-dropdown");
    try {
        const data = await api.getModels();
        if (elModelDropdown) {
            elModelDropdown.innerHTML = "";
            data.models.forEach(model => {
                const opt = document.createElement("option");
                opt.value = model.id;
                opt.textContent = `${model.name} (${model.provider.toUpperCase()})`;
                if (model.id === state.currentModel) {
                    opt.selected = true;
                }
                elModelDropdown.appendChild(opt);
            });
            
            elModelDropdown.addEventListener("change", (e) => {
                state.currentModel = e.target.value;
            });
        }
    } catch (e) {
        console.error("Failed to load models", e);
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
            data.forEach(v => {
                const opt = document.createElement("option");
                opt.value = v.id;
                opt.textContent = v.name;
                if (v.id === state.currentVoice) {
                    opt.selected = true;
                }
                elVoiceDropdown.appendChild(opt);
            });
            
            elVoiceDropdown.addEventListener("change", (e) => {
                state.currentVoice = e.target.value;
                localStorage.setItem("aethel_voice", state.currentVoice);
            });
        }
    } catch (e) {
        console.error("Failed to load voices", e);
    }
}

export async function handleSetupSubmit() {
    const elApiKey = document.getElementById("api-key");
    const elOpenAiApiKey = document.getElementById("openai-api-key");
    const elBtnInitiate = document.getElementById("btn-initiate");

    const key = elApiKey.value.trim();
    const openaiKey = elOpenAiApiKey ? elOpenAiApiKey.value.trim() : "";
    if (!key) {
        showSetupError("Uplink-Key darf nicht leer sein.");
        return;
    }

    elBtnInitiate.disabled = true;
    elBtnInitiate.querySelector("span").textContent = "BOOTING CORE...";

    try {
        const data = await api.submitSetup(key, openaiKey);

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
