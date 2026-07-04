import { state } from './state.js';
import * as api from './api.js';

export async function fetchTasksQueue() {
    const container = document.getElementById("tasks-list-container");
    if (!container) return;

    try {
        const tasks = await api.getTasks();

        if (!tasks || tasks.length === 0) {
            container.innerHTML = `<div style="color: var(--vgt-text-dim); text-align: center; margin-top: 30px;">Keine geplanten Tasks vorhanden.</div>`;
            return;
        }

        container.innerHTML = tasks.map(t => {
            let badgeClass = "status-completed";
            if (t.status === "running") badgeClass = "status-running";
            else if (t.status === "blocked") badgeClass = "status-blocked";
            else if (t.status === "failed") badgeClass = "status-rejected";
            else if (t.status === "pending") badgeClass = "status-pending";

            const nextRun = t.next_run_at ? new Date(t.next_run_at).toLocaleTimeString() : "none";
            const auditRefsCount = t.audit_refs ? t.audit_refs.length : 0;

            return `
                <div class="glass-card" style="padding: 12px 18px; display: flex; flex-direction: column; gap: 8px; background: rgba(255,255,255,0.01); border-color: rgba(0, 240, 255, 0.15); margin-bottom: 6px;">
                    <div style="display: flex; justify-content: space-between; align-items: center;">
                        <div style="text-align: left;">
                            <div style="font-size: 11px; font-weight: bold; color: #fff; text-decoration: ${t.done ? "line-through" : "none"};">${t.text}</div>
                            <div style="font-size: 8px; color: var(--vgt-text-dim); margin-top: 3px;">ID: ${t.id} | Schedule: ${t.schedule_type} | Next: ${nextRun}</div>
                        </div>
                        <span class="tool-status-badge ${badgeClass}" style="font-size: 8px; padding: 4px 8px;">${t.status.toUpperCase()}</span>
                    </div>
                    <div style="font-size: 9px; color: var(--vgt-text-dark); background: rgba(0,0,0,0.3); padding: 6px; border-radius: 4px; border-left: 2px solid var(--vgt-border);">
                        <strong>Objective:</strong> ${t.objective}
                    </div>
                    ${t.last_report ? `
                        <div style="font-size: 9px; color: var(--vgt-cyan);">
                            <strong>Last Run Report:</strong> ${t.last_report}
                        </div>
                    ` : ""}
                    <div style="display: flex; justify-content: space-between; align-items: center; margin-top: 4px;">
                        <div style="font-size: 8px; color: var(--vgt-text-dim);">Audit Refs: ${auditRefsCount} entries</div>
                        <div style="display: flex; gap: 6px;">
                            ${t.status === "running" || t.status === "blocked" ? "" : `<button class="cyber-button font-mono" onclick="triggerTaskRun('${t.id}')" style="width: auto; padding: 3px 8px; font-size: 8px; background: rgba(0, 240, 255, 0.08); border: 1px solid var(--vgt-cyan); color: var(--vgt-cyan);">RUN NOW</button>`}
                            ${t.status === "running" ? `<button class="cyber-button font-mono" onclick="triggerTaskPause('${t.id}')" style="width: auto; padding: 3px 8px; font-size: 8px; background: rgba(255, 123, 0, 0.08); border: 1px solid var(--vgt-orange); color: var(--vgt-orange);">PAUSE</button>` : ""}
                            <button class="cyber-button font-mono" onclick="triggerTaskDelete('${t.id}')" style="width: auto; padding: 3px 8px; font-size: 8px; background: rgba(255, 0, 79, 0.08); border: 1px solid var(--vgt-red); color: var(--vgt-red);">DELETE</button>
                        </div>
                    </div>
                </div>
            `;
        }).join("");
    } catch(e) {
        console.error("Failed to load tasks", e);
    }
}

export async function addTaskItem() {
    const elInput = document.getElementById("task-add-text");
    const elObjective = document.getElementById("task-add-objective");
    const elSchedule = document.getElementById("task-add-schedule");
    const elInterval = document.getElementById("task-add-interval");
    const elCapabilities = document.getElementById("task-add-capabilities");
    const elRisk = document.getElementById("task-add-risk");

    if (!elInput) return;

    const text = elInput.value.trim();
    if (!text) {
        alert("Task-Titel darf nicht leer sein.");
        return;
    }

    const objective = elObjective ? elObjective.value.trim() : text;
    const scheduleType = elSchedule ? elSchedule.value : "once";
    const intervalSeconds = elInterval ? parseInt(elInterval.value) || 60 : 60;
    
    let requiredCapabilities = [];
    if (elCapabilities && elCapabilities.value.trim()) {
        requiredCapabilities = elCapabilities.value.split(",").map(c => c.trim()).filter(Boolean);
    }
    const riskLevel = elRisk ? elRisk.value : "Moderate";

    try {
        const res = await api.addTask({
            text: text,
            objective: objective,
            schedule_type: scheduleType,
            interval_seconds: intervalSeconds,
            required_capabilities: requiredCapabilities,
            risk_level: riskLevel
        });

        if (res.status === "success") {
            elInput.value = "";
            if (elObjective) elObjective.value = "";
            if (elCapabilities) elCapabilities.value = "";
            fetchTasksQueue();
            fetchKernelTasks();
        } else {
            alert(`Fehler beim Hinzufügen des Tasks: ${res.message}`);
        }
    } catch(e) {
        console.error("Failed to create task", e);
    }
}

export async function triggerTaskRun(id) {
    try {
        const res = await api.runTaskManual(id);
        if (res.status === "success") {
            fetchTasksQueue();
        } else {
            alert("Fehler beim Starten des Tasks.");
        }
    } catch(e) {
        console.error(e);
    }
}
window.triggerTaskRun = triggerTaskRun;

export async function triggerTaskPause(id) {
    try {
        const res = await api.pauseTaskManual(id);
        if (res.status === "success") {
            fetchTasksQueue();
        } else {
            alert("Fehler beim Pausieren des Tasks.");
        }
    } catch(e) {
        console.error(e);
    }
}
window.triggerTaskPause = triggerTaskPause;

export async function triggerTaskDelete(id) {
    try {
        const res = await api.deleteTask(id);
        if (res.status === "success") {
            fetchTasksQueue();
            fetchKernelTasks();
        } else {
            alert("Fehler beim Löschen des Tasks.");
        }
    } catch(e) {
        console.error(e);
    }
}
window.triggerTaskDelete = triggerTaskDelete;

export async function fetchKernelTasks() {
    const elTaskChecklistContainer = document.getElementById("task-checklist-container");
    if (!elTaskChecklistContainer) return;
    try {
        const tasks = await api.getTasks();
        
        if (!tasks || tasks.length === 0) {
            elTaskChecklistContainer.innerHTML = `<div style="color: var(--vgt-text-dim); text-align: center; padding: 5px 0;">Keine aktiven Tasks.</div>`;
            return;
        }
        
        elTaskChecklistContainer.innerHTML = tasks.map(t => {
            const doneClass = t.done ? "done" : "";
            return `
                <div class="task-item ${doneClass}">
                    <div class="task-checkbox font-mono"></div>
                    <span style="font-size: 9px;">${t.text}</span>
                </div>
            `;
        }).join("");
    } catch(e) {
        console.error("Failed to fetch tasks", e);
    }
}

export function setupTasksUIEvents() {
    const btnAdd = document.getElementById("task-btn-add");
    const selectSchedule = document.getElementById("task-add-schedule");
    const inputInterval = document.getElementById("task-add-interval");

    if (btnAdd) {
        btnAdd.addEventListener("click", addTaskItem);
    }

    if (selectSchedule && inputInterval) {
        selectSchedule.addEventListener("change", (e) => {
            if (e.target.value === "interval") {
                inputInterval.disabled = false;
                inputInterval.style.opacity = "1";
            } else {
                inputInterval.disabled = true;
                inputInterval.style.opacity = "0.5";
            }
        });
    }
}
