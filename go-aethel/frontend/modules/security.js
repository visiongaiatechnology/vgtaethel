import { state } from './state.js';
import { speak } from './voice.js';

function renderEmpty(container, message, className = 'security-empty-state') {
    const empty = document.createElement('div');
    empty.className = className;
    empty.textContent = message;
    container.replaceChildren(empty);
}

export async function refreshSecurityHUD() {
    try {
        const res = await fetch(`${state.API_BASE}/v1/security/status`);
        if (!res.ok) throw new Error(`security status unavailable (${res.status})`);
        const status = await res.json();
        
        const hudLeasesCount = document.getElementById("hud-leases-count");
        if (hudLeasesCount) hudLeasesCount.textContent = status.active_leases_count;

        const coreStatusLeases = document.getElementById("core-status-leases");
        if (coreStatusLeases) coreStatusLeases.textContent = `${status.active_leases_count} active leases`;

        const integrityLabel = document.getElementById("hud-integrity-label");
        const integrityDot = document.getElementById("hud-integrity-dot");
        const coreStatusIntegrity = document.getElementById("core-status-integrity");

        if (status.security_status === "TAMPERED") {
            if (integrityLabel) integrityLabel.textContent = "TAMPERED WARNING";
            if (integrityDot) integrityDot.className = "pulse-dot red-dot";
            if (coreStatusIntegrity) {
                coreStatusIntegrity.textContent = "TAMPER ALERT (CHAIN FAILURE)";
                coreStatusIntegrity.className = "red-text font-bold";
            }
        } else {
            if (integrityLabel) integrityLabel.textContent = "SECURE";
            if (integrityDot) integrityDot.className = "pulse-dot green";
            if (coreStatusIntegrity) {
                coreStatusIntegrity.textContent = "SECURE";
                coreStatusIntegrity.className = "green-text";
            }
        }
    } catch(e) {
        console.error("Failed to refresh security HUD", e);
        const integrityLabel = document.getElementById("hud-integrity-label");
        const integrityDot = document.getElementById("hud-integrity-dot");
        if (integrityLabel) integrityLabel.textContent = "CORE OFFLINE";
        if (integrityDot) integrityDot.className = "pulse-dot red-dot";
    }
}

export async function fetchActiveLeasesList() {
    const container = document.getElementById("sec-leases-container");
    if (!container) return;

    try {
        const res = await fetch(`${state.API_BASE}/v1/security/leases`);
        if (!res.ok) throw new Error(`lease service unavailable (${res.status})`);
        const leases = await res.json();

        if (!leases || leases.length === 0) {
            renderEmpty(container, 'Keine aktiven Berechtigungen vorhanden.');
            return;
        }

        const cards = leases.map(lease => {
            const card = document.createElement('article');
            card.className = 'security-lease-card';
            const copy = document.createElement('div');
            const capability = document.createElement('strong');
            capability.textContent = String(lease.capability || 'unknown');
            const meta = document.createElement('span');
            const expDate = new Date(lease.expires_at);
            meta.textContent = `ID: ${String(lease.lease_id || '—')} · Bis: ${Number.isNaN(expDate.getTime()) ? '—' : expDate.toLocaleTimeString()}`;
            copy.append(capability, meta);
            const revoke = document.createElement('button');
            revoke.type = 'button';
            revoke.className = 'security-revoke-button';
            revoke.textContent = 'Widerrufen';
            revoke.addEventListener('click', () => { void revokePermissionLease(String(lease.lease_id || '')); });
            card.append(copy, revoke);
            return card;
        });
        container.replaceChildren(...cards);
    } catch(e) {
        console.error("Failed to load active leases list", e);
        renderEmpty(container, 'Berechtigungsdienst nicht erreichbar.', 'security-empty-state error');
    }
}

export async function revokePermissionLease(leaseID) {
    try {
        const res = await fetch(`${state.API_BASE}/v1/security/leases?id=${encodeURIComponent(leaseID)}`, {
            method: 'DELETE'
        });
        const data = await res.json();
        if (data.status === "success") {
            refreshSecurityHUD();
            fetchActiveLeasesList();
        } else {
            window.showAethelToast?.(`Widerruf fehlgeschlagen: ${data.message || 'Unbekannter Fehler'}`, 'error');
        }
    } catch(e) {
        console.error("Failed to revoke lease", e);
        window.showAethelToast?.('Berechtigung konnte nicht widerrufen werden.', 'error');
    }
}
window.revokePermissionLease = revokePermissionLease;

export async function fetchSecurityAuditTrail() {
    const tbody = document.getElementById("sec-audit-tbody");
    if (!tbody) return;

    try {
        const res = await fetch(`${state.API_BASE}/v1/security/audit`);
        if (!res.ok) throw new Error(`audit service unavailable (${res.status})`);
        const logs = await res.json();

        if (!logs || logs.length === 0) {
            const row = document.createElement('tr');
            const cell = document.createElement('td');
            cell.colSpan = 5;
            cell.className = 'security-table-empty';
            cell.textContent = 'Keine Audit-Einträge vorhanden.';
            row.appendChild(cell);
            tbody.replaceChildren(row);
            return;
        }

        const rows = logs.reverse().slice(0, 30).map(entry => {
            let riskClass = "safe";
            if (entry.risk === "Critical" || entry.risk === "Forbidden") riskClass = "red";
            else if (entry.risk === "High") riskClass = "orange";
            else if (entry.risk === "Moderate") riskClass = "yellow";

            let decisionClass = "allowed";
            if (entry.decision === "blocked") decisionClass = "blocked";
            else if (entry.decision === "requested_approval") decisionClass = "pending";

            const timestamp = new Date(entry.timestamp);
            const row = document.createElement('tr');
            row.className = 'security-audit-row';
            const timeCell = document.createElement('td');
            timeCell.textContent = Number.isNaN(timestamp.getTime()) ? '—' : `${timestamp.toLocaleDateString()} ${timestamp.toLocaleTimeString()}`;
            const operationCell = document.createElement('td');
            const operation = document.createElement('span');
            operation.className = 'security-operation-badge';
            operation.textContent = String(entry.operation || 'unknown');
            const target = document.createElement('small');
            target.textContent = `Cap: ${String(entry.target || '—')}`;
            operationCell.append(operation, target);
            const riskCell = document.createElement('td');
            riskCell.className = `security-risk ${riskClass}`;
            riskCell.textContent = String(entry.risk || '—');
            const decisionCell = document.createElement('td');
            decisionCell.className = `log-status ${decisionClass}`;
            decisionCell.textContent = String(entry.decision || '').toUpperCase();
            const hashCell = document.createElement('td');
            hashCell.className = 'security-chain-cell';
            const currentHash = document.createElement('span');
            currentHash.textContent = `Block: ${entry.hash ? String(entry.hash).slice(0, 8) : 'none'}`;
            const previousHash = document.createElement('small');
            previousHash.textContent = `Prev: ${entry.prev_hash ? String(entry.prev_hash).slice(0, 8) : 'none'}`;
            hashCell.append(currentHash, previousHash);
            row.append(timeCell, operationCell, riskCell, decisionCell, hashCell);
            return row;
        });
        tbody.replaceChildren(...rows);
    } catch(e) {
        console.error("Failed to fetch audit log trail", e);
        const row = document.createElement('tr');
        const cell = document.createElement('td');
        cell.colSpan = 5;
        cell.className = 'security-table-empty error';
        cell.textContent = 'Audit-Trail nicht erreichbar.';
        row.appendChild(cell);
        tbody.replaceChildren(row);
    }
}

export async function addPermissionLease(capability, durationMinutes) {
    try {
        const res = await fetch(`${state.API_BASE}/v1/security/leases`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                capability: capability,
                duration_minutes: durationMinutes
            })
        });
        const data = await res.json();
        return data.status === "success";
    } catch(e) {
        console.error("Failed to create temporary lease", e);
        return false;
    }
}

export function openPermissionGate(toolName, capability, riskLevel, riskScore, threats, args, msgId, approvalToken = "") {
    const modal = document.getElementById("permission-gate-modal");
    if (!modal) return;

    document.getElementById("gate-tool-name").textContent = toolName;
    document.getElementById("gate-capability").textContent = capability || "unknown";
    document.getElementById("gate-risk-level").textContent = `${riskLevel || "MODERATE"} (Risk Score: ${riskScore || 0})`;
    document.getElementById("gate-arguments").textContent = JSON.stringify(args, null, 2);

    const threatWarning = document.getElementById("gate-threat-warning");
    const threatList = document.getElementById("gate-threat-list");
    if (threats && threats.length > 0) {
        threatWarning.classList.remove("hidden");
        threatList.replaceChildren(...threats.map((t) => {
            const li = document.createElement("li");
            li.textContent = t;
            return li;
        }));
    } else {
        threatWarning.classList.add("hidden");
        threatList.replaceChildren();
    }

    const btnOnce = document.getElementById("gate-btn-approve-once");
    const btn15m = document.getElementById("gate-btn-approve-15m");
    const btn1h = document.getElementById("gate-btn-approve-1h");
    const btnReject = document.getElementById("gate-btn-reject");

    const cloneAndReplace = (btn) => {
        const newBtn = btn.cloneNode(true);
        btn.parentNode.replaceChild(newBtn, btn);
        return newBtn;
    };

    const activeOnce = cloneAndReplace(btnOnce);
    const active15m = cloneAndReplace(btn15m);
    const active1h = cloneAndReplace(btn1h);
    const activeReject = cloneAndReplace(btnReject);

    activeOnce.addEventListener("click", async () => {
        modal.classList.add("hidden");
        const { executeApprovedTool } = await import('./chat.js');
        executeApprovedTool(msgId, toolName, args, approvalToken);
    });

    active15m.addEventListener("click", async () => {
        modal.classList.add("hidden");
        const ok = await addPermissionLease(capability, 15);
        if (ok) {
            const { executeApprovedTool } = await import('./chat.js');
            executeApprovedTool(msgId, toolName, args);
        } else {
            window.showAethelToast?.('15-Minuten-Berechtigung konnte nicht erstellt werden.', 'error');
        }
    });

    active1h.addEventListener("click", async () => {
        modal.classList.add("hidden");
        const ok = await addPermissionLease(capability, 60);
        if (ok) {
            const { executeApprovedTool } = await import('./chat.js');
            executeApprovedTool(msgId, toolName, args);
        } else {
            window.showAethelToast?.('1-Stunden-Berechtigung konnte nicht erstellt werden.', 'error');
        }
    });

    activeReject.addEventListener("click", async () => {
        modal.classList.add("hidden");
        const { rejectTool } = await import('./chat.js');
        rejectTool(msgId);
    });

    modal.classList.remove("hidden");
    
    let speakName = toolName;
    if (speakName === "sys_exec_cmd") speakName = "Systembefehl";
    if (speakName === "web_browser") speakName = "Webbrowser";
    if (speakName === "fs_write_file") speakName = "Datei schreiben";
    speak(`Freigabe für ${speakName} erforderlich. Bitte freigeben.`);
}

export async function fetchKernelLogs() {
    const elKernelLogsTbody = document.getElementById("logs-tbody");
    if (!elKernelLogsTbody) return;
    try {
        const res = await fetch(`${state.API_BASE}/v1/kernel/logs`);
        if (!res.ok) throw new Error(`kernel logs unavailable (${res.status})`);
        const logs = await res.json();
        
        if (!logs || logs.length === 0) {
            const row = document.createElement('tr');
            const cell = document.createElement('td');
            cell.colSpan = 4;
            cell.className = 'security-table-empty';
            cell.textContent = 'Keine Kernel-Einträge.';
            row.appendChild(cell);
            elKernelLogsTbody.replaceChildren(row);
            return;
        }
        
        const rows = logs.map(log => {
            const opClass = String(log.op || "").replace(/[^A-Za-z0-9_-]/g, "_");
            const statusClass = String(log.status || "").replace(/[^A-Za-z0-9_-]/g, "_");
            const row = document.createElement('tr');
            row.className = 'security-audit-row';
            const timestamp = document.createElement('td');
            timestamp.textContent = String(log.timestamp || '—');
            const operation = document.createElement('td');
            const badge = document.createElement('span');
            badge.className = `log-op-badge ${opClass}`;
            badge.textContent = String(log.op || '—');
            operation.appendChild(badge);
            const target = document.createElement('td');
            target.className = 'security-target-cell';
            target.textContent = String(log.target || '—');
            target.title = String(log.target || '');
            const status = document.createElement('td');
            status.className = `log-status ${statusClass}`;
            status.textContent = String(log.status || '—');
            row.append(timestamp, operation, target, status);
            return row;
        });
        elKernelLogsTbody.replaceChildren(...rows);
    } catch (e) {
        console.error("Failed to fetch kernel logs", e);
        const row = document.createElement('tr');
        const cell = document.createElement('td');
        cell.colSpan = 4;
        cell.className = 'security-table-empty error';
        cell.textContent = 'Kernel-Logs nicht erreichbar.';
        row.appendChild(cell);
        elKernelLogsTbody.replaceChildren(row);
    }
}
