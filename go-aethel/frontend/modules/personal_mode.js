import { state } from './state.js';
import * as api from './api.js';

function val(id) {
    return document.getElementById(id)?.value.trim() || "";
}

function setVal(id, value) {
    const el = document.getElementById(id);
    if (el) el.value = value || "";
}

function setChecked(id, value) {
	const el = document.getElementById(id);
	if (el) el.checked = !!value;
}

function setLevel(id, value) {
    const input = document.getElementById(id);
    const output = document.getElementById(`${id}-value`);
    const level = Math.max(0, Math.min(100, Number(value) || 0));
    if (input) input.value = String(level);
    if (output) output.textContent = `${level}%`;
}

export async function loadPersonalMode() {
    const statusEl = document.getElementById("personal-status");
    try {
        const status = await api.getPersonalStatus();
        const cfg = status.config || {};
        const profile = status.profile || {};
        await populatePersonalModelSelects(cfg.primary_model || state.currentModel, cfg.fallback_model || "");
        state.wakeWord = cfg.wake_word || state.wakeWord || "aethel";
        localStorage.setItem("aethel_wake_word", state.wakeWord);

        setChecked("personal-enabled", cfg.enabled);
        setChecked("personal-learning", cfg.learning_enabled);
        setLevel("personal-humor", cfg.humor_level ?? 35);
        setLevel("personal-honesty", cfg.honesty_level ?? 90);
        setLevel("personal-initiative", cfg.initiative_level ?? 60);
        setVal("personal-wake-word", cfg.wake_word || "aethel");
        setVal("personal-display-name", profile.display_name || "");
        setVal("personal-tone", profile.preferred_tone || "");
        setVal("personal-style", profile.assistant_style || "");
        setVal("personal-interests", (profile.interests || []).join(", "));
        setVal("personal-goals", (profile.goals || []).join(", "));
        setVal("personal-notes", profile.notes || "");

        if (statusEl) {
            statusEl.textContent = cfg.enabled
                ? `Aktiv. Profil: ${profile.display_name || "unbenannt"} | Memories: ${status.memory_count || 0}`
                : "Inaktiv. Setup ausfüllen und aktivieren.";
        }
        await renderPersonalMemories();
        await updateColleagueDashboard(profile, status);
    } catch(e) {
        console.error("Failed to load personal mode", e);
        if (statusEl) statusEl.textContent = "Persönlicher Modus nicht verfügbar.";
    }
}

async function populatePersonalModelSelects(primaryModel, fallbackModel) {
    const primary = document.getElementById("personal-primary-model");
    const fallback = document.getElementById("personal-fallback-model");
    if (!primary || !fallback) return;

    const data = await api.getModels();
    const models = data.models || [];
    const fill = (select, selected, includeNone) => {
        select.innerHTML = "";
        if (includeNone) {
            const none = document.createElement("option");
            none.value = "";
            none.textContent = "Kein Fallback";
            select.appendChild(none);
        }
        const groups = {};
        models.forEach(model => {
            const provider = model.provider || "Andere";
            if (!groups[provider]) groups[provider] = [];
            groups[provider].push(model);
        });
        Object.keys(groups).sort().forEach(provider => {
            const optGroup = document.createElement("optgroup");
            optGroup.label = provider;
            groups[provider].forEach(model => {
                const opt = document.createElement("option");
                opt.value = model.id;
                opt.textContent = model.name;
                if (model.id === selected) opt.selected = true;
                optGroup.appendChild(opt);
            });
            select.appendChild(optGroup);
        });
        if (selected && !models.some(model => model.id === selected)) {
            const opt = document.createElement("option");
            opt.value = selected;
            opt.textContent = `${selected} (nicht in Registry)`;
            opt.selected = true;
            select.appendChild(opt);
        }
    };

    fill(primary, primaryModel || models[0]?.id || "", false);
    fill(fallback, fallbackModel || "", true);
}

export function setupPersonalModeUIEvents() {
    const guidedBtn = document.getElementById("personal-btn-guided-setup");
    if (guidedBtn) guidedBtn.addEventListener("click", runGuidedSetupConversation);
    const setupBtn = document.getElementById("personal-btn-setup");
    if (setupBtn) setupBtn.addEventListener("click", savePersonalSetup);
    const configBtn = document.getElementById("personal-btn-config");
    if (configBtn) configBtn.addEventListener("click", savePersonalConfig);
    for (const id of ["personal-humor", "personal-honesty", "personal-initiative"]) {
        document.getElementById(id)?.addEventListener("input", event => setLevel(id, event.currentTarget.value));
    }
}

async function runGuidedSetupConversation() {
    const statusEl = document.getElementById("personal-status");
    if (statusEl) statusEl.textContent = "Aethel bereitet Setup-Fragen vor...";
    const config = collectPersonalConfig();
    const questions = await loadSetupQuestions(config);
    /*
        ["personal-display-name", "Wie soll Aethel dich nennen?"],
        ["personal-tone", "Wie soll Aethel mit dir sprechen? Beispiel: locker, direkt, ruhig, motivierend."],
        ["personal-style", "Was macht einen guten persönlichen Assistenten für dich aus?"],
        ["personal-interests", "Welche Interessen, Themen oder Medien soll Aethel über dich kennen?"],
        ["personal-goals", "Welche Ziele oder Projekte soll Aethel langfristig im Blick behalten?"],
        ["personal-notes", "Gibt es Grenzen, No-Gos oder wichtige persönliche Hinweise?"],
        ["personal-wake-word", "Welches Wake-Word möchtest du nutzen?"]
    ];
    */
    for (const item of questions) {
        const target = Array.isArray(item) ? item[0] : item.target;
        const question = Array.isArray(item) ? item[1] : item.question;
        const current = val(target);
        const answer = window.prompt(question, current);
        if (answer === null) return;
        setVal(target, answer);
    }
    setChecked("personal-enabled", true);
    setChecked("personal-learning", true);
    await savePersonalSetup();
}

async function loadSetupQuestions(config) {
    try {
        const res = await api.getPersonalSetupQuestions(config);
        if (res.status === "success" && Array.isArray(res.questions) && res.questions.length > 0) {
            return res.questions.filter(item => item && item.target && item.question);
        }
    } catch(e) {
        console.warn("Falling back to static personal setup questions", e);
    }
    return [
        { target: "personal-display-name", question: "Wie soll Aethel dich nennen?" },
        { target: "personal-tone", question: "Wie soll Aethel mit dir sprechen? Beispiel: locker, direkt, ruhig, motivierend." },
        { target: "personal-style", question: "Was macht einen guten persoenlichen Assistenten fuer dich aus?" },
        { target: "personal-interests", question: "Welche Interessen, Themen oder Medien soll Aethel ueber dich kennen?" },
        { target: "personal-goals", question: "Welche Ziele oder Projekte soll Aethel langfristig im Blick behalten?" },
        { target: "personal-notes", question: "Gibt es Grenzen, No-Gos oder wichtige persoenliche Hinweise?" },
        { target: "personal-wake-word", question: "Welches Wake-Word moechtest du nutzen?" }
    ];
}

async function savePersonalSetup() {
    const config = collectPersonalConfig();
    const payload = {
        answers: {
            display_name: val("personal-display-name"),
            preferred_tone: val("personal-tone"),
            assistant_style: val("personal-style"),
            interests: val("personal-interests"),
            goals: val("personal-goals"),
            notes: val("personal-notes")
        },
        config
    };
    const res = await api.runPersonalSetup(payload);
    if (res.status === "success") {
        state.wakeWord = config.wake_word || "aethel";
        localStorage.setItem("aethel_wake_word", state.wakeWord);
        await loadPersonalMode();
    }
}

function collectPersonalConfig() {
    return {
        enabled: !!document.getElementById("personal-enabled")?.checked,
        learning_enabled: !!document.getElementById("personal-learning")?.checked,
        humor_level: Number(document.getElementById("personal-humor")?.value || 35),
        honesty_level: Number(document.getElementById("personal-honesty")?.value || 90),
        initiative_level: Number(document.getElementById("personal-initiative")?.value || 60),
        wake_word: val("personal-wake-word") || "aethel",
        primary_model: val("personal-primary-model"),
        fallback_model: val("personal-fallback-model")
    };
}

async function savePersonalConfig() {
    const cfg = collectPersonalConfig();
    const res = await api.savePersonalConfig(cfg);
    if (res.status === "success") {
        state.wakeWord = cfg.wake_word || "aethel";
        localStorage.setItem("aethel_wake_word", state.wakeWord);
        await loadPersonalMode();
    }
}

async function renderPersonalMemories() {
    const container = document.getElementById("personal-memory-list");
    if (!container) return;
    const memories = await api.getPersonalMemories();
    if (!memories || memories.length === 0) {
        container.innerHTML = `<div style="color: var(--vgt-text-dim); text-align: center; padding: 24px;">Noch keine persönlichen Erinnerungen.</div>`;
        return;
    }
    container.replaceChildren(...memories.slice().reverse().map(createMemoryNode));
}

function createMemoryNode(mem) {
    const row = document.createElement("div");
    row.style.cssText = "border: 1px solid rgba(0,240,255,0.12); background: rgba(0,0,0,0.25); border-radius: 4px; padding: 10px; display: flex; justify-content: space-between; gap: 10px;";

    const body = document.createElement("div");
    body.style.cssText = "display: flex; flex-direction: column; gap: 4px;";

    const meta = document.createElement("span");
    meta.style.cssText = "color: var(--vgt-cyan); font-size: 8px; text-transform: uppercase;";
    meta.textContent = `${mem.type || "memory"} | ${(Number(mem.confidence || 0) * 100).toFixed(0)}%`;

    const content = document.createElement("span");
    content.style.cssText = "color: #fff; font-size: 10px;";
    content.textContent = mem.content || "";

    const source = document.createElement("span");
    source.style.cssText = "color: var(--vgt-text-dim); font-size: 8px;";
    source.textContent = `${mem.source || ""} | ${mem.created_at || ""}`;

    body.append(meta, content, source);

    const actions = document.createElement("div");
    actions.style.cssText = "display: flex; align-items: flex-start; gap: 6px;";

    const edit = document.createElement("button");
    edit.className = "cyber-button font-mono";
    edit.style.cssText = "width: auto; padding: 4px 8px; font-size: 8px;";
    edit.textContent = "EDIT";
    edit.addEventListener("click", () => editPersonalMemory(mem.id));

    const del = document.createElement("button");
    del.className = "cyber-button font-mono";
    del.style.cssText = "width: auto; padding: 4px 8px; font-size: 8px; border-color: var(--vgt-red); color: var(--vgt-red);";
    del.textContent = "DEL";
    del.addEventListener("click", () => deletePersonalMemory(mem.id));

    actions.append(edit, del);
    row.append(body, actions);
    return row;
}

export async function editPersonalMemory(id) {
    const memories = await api.getPersonalMemories();
    const mem = memories.find(item => item.id === id);
    if (!mem) return;
    const type = window.prompt("Typ der Erinnerung", mem.type || "preference");
    if (type === null) return;
    const content = window.prompt("Inhalt der Erinnerung", mem.content || "");
    if (content === null) return;
    const updated = {
        ...mem,
        type: type.trim(),
        content: content.trim()
    };
    if (!updated.type || !updated.content) return;
    await api.updatePersonalMemory(updated);
    await renderPersonalMemories();
}

export async function deletePersonalMemory(id) {
    await api.deletePersonalMemory(id);
    await renderPersonalMemories();
}

window.editPersonalMemory = editPersonalMemory;
window.deletePersonalMemory = deletePersonalMemory;

async function updateColleagueDashboard(profile, status) {
    const elOverview = document.getElementById("colleague-daily-overview");
    const elProjects = document.getElementById("colleague-active-projects");
    const elRuns = document.getElementById("colleague-prioritized-runs");
    const elSuggestions = document.getElementById("colleague-suggestions");
    if (!elOverview) return;

    // 1. Daily Overview / Greeting (Harden: Pure DOM construction, no innerHTML with user variable)
    const operatorName = profile.display_name || "Operator";
    const now = new Date();
    const deTime = now.toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit' });
    const deDate = now.toLocaleDateString('de-DE', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' });
    
    let timeGreeting = "Guten Tag";
    const hour = now.getHours();
    if (hour < 5) timeGreeting = "Gute Nacht";
    else if (hour < 11) timeGreeting = "Guten Morgen";
    else if (hour < 18) timeGreeting = "Guten Tag";
    else timeGreeting = "Guten Abend";

    elOverview.replaceChildren();
    const str1 = document.createElement("strong");
    str1.textContent = `${timeGreeting}, ${operatorName}!`;
    const br1 = document.createElement("br");
    const spanDate = document.createElement("span");
    spanDate.textContent = `Es ist ${deTime} Uhr am ${deDate}.`;
    const br2 = document.createElement("br");
    const spanText = document.createElement("span");
    spanText.textContent = "Aethel steht für autonome Impulse bereit. Systemzustand: ";
    const strStable = document.createElement("strong");
    strStable.textContent = "STABIL";
    const dot = document.createTextNode(".");
    
    elOverview.append(str1, br1, spanDate, br2, spanText, strStable, dot);

    // 2. Active Projects (Mounted Dirs) (Harden: Pure DOM, no innerHTML)
    try {
        const settings = await api.getSettings();
        const dirs = settings.mounted_dirs || [];
        elProjects.replaceChildren();
        if (dirs.length === 0) {
            const span = document.createElement("span");
            span.style.color = "var(--vgt-text-dim)";
            span.textContent = "Keine aktiven Projektordner freigegeben.";
            elProjects.appendChild(span);
        } else {
            for (const d of dirs) {
                const item = document.createElement("div");
                item.style.cssText = "display:flex;align-items:center;gap:6px;font-size:9px;color:var(--vgt-cyan);word-break:break-all;";
                const chevron = document.createElement("span");
                chevron.style.color = "var(--vgt-purple)";
                chevron.textContent = "›";
                const text = document.createTextNode(` ${d}`);
                item.append(chevron, text);
                elProjects.appendChild(item);
            }
        }
    } catch(e) {
        console.error("Failed to load active projects", e);
    }

    // 3. Prioritized Runs (paused or running)
    try {
        const runsPayload = await api.getRuns();
        const activeRuns = (runsPayload.runs || []).filter(run => ["running", "paused", "waiting_approval"].includes(run.status));
        elRuns.replaceChildren();
        if (activeRuns.length === 0) {
            const span = document.createElement("span");
            span.style.color = "var(--vgt-text-dim)";
            span.textContent = "Keine ausstehenden oder aktiven Runs.";
            elRuns.appendChild(span);
        } else {
            for (const run of activeRuns) {
                const item = document.createElement("div");
                item.style.cssText = "display:flex;justify-content:space-between;align-items:center;gap:4px;font-size:9px;";
                const title = document.createElement("span");
                title.textContent = run.objective.length > 25 ? run.objective.slice(0, 22) + "..." : run.objective;
                const statusBadge = document.createElement("strong");
                statusBadge.textContent = run.status.toUpperCase();
                statusBadge.style.color = run.status === "running" ? "var(--vgt-cyan)" : "var(--vgt-orange)";
                item.append(title, statusBadge);
                elRuns.appendChild(item);
            }
        }
    } catch(e) {
        console.error("Failed to load runs", e);
    }

    // 4. Proactive suggestions
    const suggestions = [
        { label: "🔍 Log-Audit analysieren", objective: "Analysiere security_audit.json und erstelle einen kurzen Bericht über blockierte Aktionen und Risiken" },
        { label: "🛠️ Systemdiagnose exportieren", objective: "Rufe die System-Diagnostics auf und überprüfe den Gesundheitszustand der Provider" },
        { label: "🧹 Workspace aufräumen", objective: "Analysiere den Workspace und finde temporäre oder ungenutzte Backup-Dateien zum Aufräumen" }
    ];

    elSuggestions.replaceChildren();
    for (const sug of suggestions) {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "cyber-button font-mono";
        btn.textContent = sug.label;
        btn.style.cssText = "width:100%;text-align:left;padding:6px 10px;font-size:9px;color:#fff;border-color:rgba(0,240,255,0.15);background:rgba(0,0,0,0.25);";
        btn.addEventListener("click", async () => {
            btn.disabled = true;
            try {
                const run = await api.createRun({
                    objective: sug.objective,
                    profile_id: "developer",
                    model_id: state.currentModel,
                    cost_budget_usd: 2,
                    steps: [
                        { kind: "plan", title: "Vorschlag auswerten & Planen" },
                        { kind: "report", title: "Verifizierten Bericht ausgeben" }
                    ]
                });
                await api.runAction(run.id, "start");
                window.alert(`Run '${sug.label}' erfolgreich im Run Center gestartet!`);
            } catch(e) {
                window.alert(`Vorschlag-Start fehlgeschlagen: ${e.message}`);
            } finally {
                btn.disabled = false;
            }
        });
        elSuggestions.appendChild(btn);
    }
}
