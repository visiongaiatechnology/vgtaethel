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

let personalHydrated = false;
let personalHydrateInFlight = null;

/**
 * Loads saved Personal Core config/profile into the form, memories and Neural Core greeting.
 * Safe to call at boot (prefetch) and again when opening the view.
 */
export async function loadPersonalMode() {
    if (personalHydrateInFlight) return personalHydrateInFlight;
    personalHydrateInFlight = (async () => {
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
            setChecked("personal-startup-briefing", cfg.startup_briefing);
            setChecked("personal-startup-read-aloud", cfg.startup_read_aloud);
            setLevel("personal-humor", cfg.humor_level ?? 35);
            setLevel("personal-honesty", cfg.honesty_level ?? 90);
            setLevel("personal-initiative", cfg.initiative_level ?? 60);
            setVal("personal-wake-word", cfg.wake_word || "aethel");
            setVal("personal-display-name", profile.display_name || "");
            setVal("personal-location-city", profile.location_city || "");
            setVal("personal-location-country", profile.location_country || "");
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
            // Sync Neural Core greeting from saved profile (do not re-count visits).
            await refreshNeuralCoreHome({ countVisit: false, profile });
            personalHydrated = true;
        } catch (e) {
            console.error("Failed to load personal mode", e);
            if (statusEl) statusEl.textContent = "Persönlicher Modus nicht verfügbar.";
            // Retry Neural Core name paint even if form hydration failed partially.
            await refreshNeuralCoreHome({ countVisit: false }).catch(() => {});
        } finally {
            personalHydrateInFlight = null;
        }
    })();
    return personalHydrateInFlight;
}

/** Boot-time prefetch so Personal Core / Neural Core are not empty until the user opens the view. */
export async function hydratePersonalModeAtBoot() {
    // Always load saved form/profile first (deduped if already in flight).
    await loadPersonalMode();
    // Count this app open once for the welcome-back greeting.
    await refreshNeuralCoreHome({ countVisit: true });
}

async function populatePersonalModelSelects(primaryModel, fallbackModel) {
    const primary = document.getElementById("personal-primary-model");
    const fallback = document.getElementById("personal-fallback-model");
    if (!primary || !fallback) return;

    let models = [];
    try {
        const data = await api.getModels();
        models = Array.isArray(data?.models) ? data.models : [];
    } catch (error) {
        console.warn("Personal model selects: registry unavailable, using selected ids only", error);
    }
    const fill = (select, selected, includeNone) => {
        select.replaceChildren();
        if (includeNone) {
            const none = document.createElement("option");
            none.value = "";
            none.textContent = "Kein Fallback";
            select.appendChild(none);
        }
        // Flat options only (WebView2-safe) — no optgroup.
        for (const model of models) {
            if (!model?.id) continue;
            const opt = document.createElement("option");
            opt.value = model.id;
            opt.textContent = model.name ? `${model.name} (${model.provider || "Andere"})` : model.id;
            if (model.id === selected) opt.selected = true;
            select.appendChild(opt);
        }
        if (selected && !models.some(model => model.id === selected)) {
            const opt = document.createElement("option");
            opt.value = selected;
            opt.textContent = `${selected} (gespeichert)`;
            opt.selected = true;
            select.appendChild(opt);
        }
    };

    fill(primary, primaryModel || models[0]?.id || "", false);
    fill(fallback, fallbackModel || "", true);
}

export function setupPersonalModeUIEvents() {
    if (setupPersonalModeUIEvents._bound) return;
    setupPersonalModeUIEvents._bound = true;

    const guidedBtn = document.getElementById("personal-btn-guided-setup");
    if (guidedBtn) guidedBtn.addEventListener("click", runGuidedSetupConversation);
    const setupBtn = document.getElementById("personal-btn-setup");
    if (setupBtn) setupBtn.addEventListener("click", savePersonalSetup);
    const configBtn = document.getElementById("personal-btn-config");
    if (configBtn) configBtn.addEventListener("click", savePersonalConfig);
    for (const id of ["personal-humor", "personal-honesty", "personal-initiative"]) {
        document.getElementById(id)?.addEventListener("input", event => setLevel(id, event.currentTarget.value));
    }
    document.getElementById("neural-core-open-personal")?.addEventListener("click", () => {
        document.getElementById("nav-btn-personal")?.click();
    });
    document.getElementById("neural-core-open-gw")?.addEventListener("click", () => {
        document.getElementById("nav-btn-global-watch")?.click();
    });
    document.getElementById("neural-core-start-setup")?.addEventListener("click", () => {
        document.getElementById("nav-btn-personal")?.click();
        setTimeout(() => document.getElementById("personal-btn-guided-setup")?.click(), 200);
    });
    // Do not fetch personal data here — boot hydrate / switchMode own the load timing
    // so we never paint before the core is READY or leave fields empty after a race.
}

/**
 * Neural Core greeting: first visit → setup prompt; later visits → "Willkommen zurück {name}".
 * Tracks visit count in localStorage; name comes from Personal profile DisplayName.
 * @param {{ countVisit?: boolean, profile?: { display_name?: string }, retries?: number }} [options]
 */
export async function refreshNeuralCoreHome(options = {}) {
    const countVisit = options.countVisit !== false;
    const retries = Number.isFinite(options.retries) ? options.retries : 3;
    const greeting = document.getElementById("neural-core-greeting");
    const sub = document.getElementById("neural-core-subline");
    const firstRun = document.getElementById("neural-core-first-run");
    if (!greeting) return;

    let visits = Number(localStorage.getItem("aethel_neural_core_visits") || "0");
    if (!Number.isFinite(visits) || visits < 0) visits = 0;
    if (countVisit) {
        visits += 1;
        localStorage.setItem("aethel_neural_core_visits", String(visits));
    }

    let displayName = String(options.profile?.display_name || "").trim();
    let setupNeeded = !displayName;
    if (!displayName) {
        for (let attempt = 0; attempt < retries; attempt++) {
            try {
                const status = await api.getPersonalStatus();
                displayName = String(status?.profile?.display_name || "").trim();
                setupNeeded = !!status?.setup_needed || !displayName;
                break;
            } catch (_) {
                if (attempt < retries - 1) {
                    await new Promise(resolve => setTimeout(resolve, 300 * (attempt + 1)));
                } else {
                    setupNeeded = true;
                }
            }
        }
    }

    if (displayName && visits >= 2) {
        greeting.textContent = `Willkommen zurück, ${displayName}`;
        if (sub) sub.textContent = "Personal Core aktiv · Kommandozentrale bereit · Zero CDN";
        firstRun?.classList.add("hidden");
    } else if (displayName) {
        greeting.textContent = `Willkommen, ${displayName}`;
        if (sub) sub.textContent = "Schön, dich kennenzulernen · Neural Core online";
        firstRun?.classList.add("hidden");
    } else {
        greeting.textContent = "Willkommen, Operator";
        if (sub) sub.textContent = "Richte deinen Personal Core ein — danach begrüßt dich Aethel persönlich.";
        if (visits <= 2) firstRun?.classList.remove("hidden");
        else firstRun?.classList.add("hidden");
    }
    if (setupNeeded && visits === 1) {
        firstRun?.classList.remove("hidden");
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
            location_city: val("personal-location-city"),
            location_country: val("personal-location-country"),
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
        startup_briefing: !!document.getElementById("personal-startup-briefing")?.checked,
        startup_read_aloud: !!document.getElementById("personal-startup-read-aloud")?.checked,
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
        const empty = document.createElement('div');
        empty.className = 'personal-empty-state';
        empty.textContent = 'Noch keine persönlichen Erinnerungen.';
        container.replaceChildren(empty);
        return;
    }
    container.replaceChildren(...memories.slice().reverse().map(createMemoryNode));
}

function createMemoryNode(mem) {
    const row = document.createElement("div");
    row.className = 'personal-memory-card';

    const body = document.createElement("div");
    body.className = 'personal-memory-copy';

    const meta = document.createElement("span");
    meta.className = 'personal-memory-meta';
    meta.textContent = `${mem.type || "memory"} | ${(Number(mem.confidence || 0) * 100).toFixed(0)}%`;

    const content = document.createElement("span");
    content.className = 'personal-memory-content';
    content.textContent = mem.content || "";

    const source = document.createElement("span");
    source.className = 'personal-memory-source';
    source.textContent = `${mem.source || ""} | ${mem.created_at || ""}`;

    body.append(meta, content, source);

    const actions = document.createElement("div");
    actions.className = 'personal-memory-actions';

    const edit = document.createElement("button");
    edit.className = "cyber-button font-mono personal-memory-button";
    edit.textContent = "EDIT";
    edit.addEventListener("click", () => editPersonalMemory(mem.id));

    const del = document.createElement("button");
    del.className = "cyber-button font-mono personal-memory-button danger";
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
            span.className = 'colleague-empty-state';
            span.textContent = "Keine aktiven Projektordner freigegeben.";
            elProjects.appendChild(span);
        } else {
            for (const d of dirs) {
                const item = document.createElement("div");
                item.className = 'colleague-project-row';
                const chevron = document.createElement("span");
                chevron.className = 'colleague-chevron';
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
            span.className = 'colleague-empty-state';
            span.textContent = "Keine ausstehenden oder aktiven Runs.";
            elRuns.appendChild(span);
        } else {
            for (const run of activeRuns) {
                const item = document.createElement("div");
                item.className = 'colleague-run-row';
                const title = document.createElement("span");
                title.textContent = run.objective.length > 25 ? run.objective.slice(0, 22) + "..." : run.objective;
                const statusBadge = document.createElement("strong");
                statusBadge.textContent = run.status.toUpperCase();
                statusBadge.className = run.status === 'running' ? 'running' : 'attention';
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
        btn.className = "cyber-button font-mono colleague-suggestion-button";
        btn.textContent = sug.label;
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
                window.showAethelToast?.(`Run '${sug.label}' im Run Center gestartet.`, 'success');
            } catch(e) {
                window.showAethelToast?.(`Vorschlag-Start fehlgeschlagen: ${e.message}`, 'error');
            } finally {
                btn.disabled = false;
            }
        });
        elSuggestions.appendChild(btn);
    }
}
