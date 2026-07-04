import { state } from './state.js';
import { speak } from './voice.js';

export async function refreshSecurityHUD() {
    try {
        const res = await fetch(`${state.API_BASE}/v1/security/status`);
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
    }
}

export async function fetchActiveLeasesList() {
    const container = document.getElementById("sec-leases-container");
    if (!container) return;

    try {
        const res = await fetch(`${state.API_BASE}/v1/security/leases`);
        const leases = await res.json();

        if (!leases || leases.length === 0) {
            container.innerHTML = `<div style="color: var(--vgt-text-dim); text-align: center; margin-top: 30px;">Keine aktiven Leases vorhanden.</div>`;
            return;
        }

        container.innerHTML = leases.map(l => {
            const expDate = new Date(l.expires_at).toLocaleTimeString();
            return `
                <div class="glass-card" style="padding: 10px 15px; display: flex; justify-content: space-between; align-items: center; background: rgba(255,255,255,0.01); border-color: rgba(157, 78, 221, 0.2); margin-bottom: 6px;">
                    <div style="text-align: left;">
                        <div style="font-weight: bold; color: var(--vgt-purple);">${l.capability}</div>
                        <div style="font-size: 8px; color: var(--vgt-text-dim); margin-top: 2px;">ID: ${l.lease_id} | Bis: ${expDate}</div>
                    </div>
                    <button class="cyber-button font-mono" onclick="revokePermissionLease('${l.lease_id}')" style="width: auto; padding: 4px 10px; font-size: 8px; background: rgba(255, 0, 79, 0.1); border: 1px solid var(--vgt-red); color: var(--vgt-red);">WIDERRUFEN</button>
                </div>
            `;
        }).join("");
    } catch(e) {
        console.error("Failed to load active leases list", e);
    }
}

export async function revokePermissionLease(leaseID) {
    try {
        const res = await fetch(`${state.API_BASE}/v1/security/leases?id=${leaseID}`, {
            method: 'DELETE'
        });
        const data = await res.json();
        if (data.status === "success") {
            refreshSecurityHUD();
            fetchActiveLeasesList();
        } else {
            alert(`Revocation failed: ${data.message}`);
        }
    } catch(e) {
        console.error("Failed to revoke lease", e);
    }
}
window.revokePermissionLease = revokePermissionLease;

export async function fetchSecurityAuditTrail() {
    const tbody = document.getElementById("sec-audit-tbody");
    if (!tbody) return;

    try {
        const res = await fetch(`${state.API_BASE}/v1/security/audit`);
        const logs = await res.json();

        if (!logs || logs.length === 0) {
            tbody.innerHTML = `<tr><td colspan="5" style="padding: 20px; text-align: center; color: var(--vgt-text-dim);">Keine Logbucheinträge vorhanden.</td></tr>`;
            return;
        }

        tbody.innerHTML = logs.reverse().slice(0, 30).map(entry => {
            let riskClass = "safe";
            if (entry.risk === "Critical" || entry.risk === "Forbidden") riskClass = "red";
            else if (entry.risk === "High") riskClass = "orange";
            else if (entry.risk === "Moderate") riskClass = "yellow";

            let decisionClass = "allowed";
            if (entry.decision === "blocked") decisionClass = "blocked";
            else if (entry.decision === "requested_approval") decisionClass = "pending";

            const dateStr = new Date(entry.timestamp).toLocaleDateString();
            const timeStr = new Date(entry.timestamp).toLocaleTimeString();

            const blockHash = entry.hash ? entry.hash.substring(0, 8) : "none";
            const prevHash = entry.prev_hash ? entry.prev_hash.substring(0, 8) : "none";

            return `
                <tr style="border-bottom: 1px solid rgba(255,255,255,0.03);">
                    <td style="padding: 6px 4px; color: var(--vgt-text-dim);">${dateStr} ${timeStr}</td>
                    <td style="padding: 6px 4px; font-weight: bold;">
                        <span class="log-op-badge" style="background: rgba(157, 78, 221, 0.1); color: var(--vgt-purple); border: 1px solid rgba(157, 78, 221, 0.2); padding: 2px 4px; border-radius: 3px; font-size: 8px;">${entry.operation}</span>
                        <div style="font-size: 8px; color: var(--vgt-text-dim); margin-top: 2px;">Cap: ${entry.target}</div>
                    </td>
                    <td style="padding: 6px 4px;" class="${riskClass}">${entry.risk}</td>
                    <td style="padding: 6px 4px;" class="log-status ${decisionClass}">${entry.decision.toUpperCase()}</td>
                    <td style="padding: 6px 4px; font-family: var(--font-mono); color: var(--vgt-cyan); font-size: 8px;">
                        <span>Chain block: ${blockHash}</span>
                        <div style="color: var(--vgt-text-dark);">Prev: ${prevHash}</div>
                    </td>
                </tr>
            `;
        }).join("");
    } catch(e) {
        console.error("Failed to fetch audit log trail", e);
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

export function openPermissionGate(toolName, capability, riskLevel, riskScore, threats, args, msgIndex) {
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
        threatList.innerHTML = threats.map(t => `<li>${t}</li>`).join("");
    } else {
        threatWarning.classList.add("hidden");
        threatList.innerHTML = "";
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
        executeApprovedTool(msgIndex, toolName, args, true);
    });

    active15m.addEventListener("click", async () => {
        modal.classList.add("hidden");
        const ok = await addPermissionLease(capability, 15);
        if (ok) {
            const { executeApprovedTool } = await import('./chat.js');
            executeApprovedTool(msgIndex, toolName, args, false);
        } else {
            alert("Fehler beim Erstellen des 15m-Leases.");
        }
    });

    active1h.addEventListener("click", async () => {
        modal.classList.add("hidden");
        const ok = await addPermissionLease(capability, 60);
        if (ok) {
            const { executeApprovedTool } = await import('./chat.js');
            executeApprovedTool(msgIndex, toolName, args, false);
        } else {
            alert("Fehler beim Erstellen des 1h-Leases.");
        }
    });

    activeReject.addEventListener("click", async () => {
        modal.classList.add("hidden");
        const { rejectTool } = await import('./chat.js');
        rejectTool(msgIndex);
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
        const logs = await res.json();
        
        if (!logs || logs.length === 0) {
            elKernelLogsTbody.innerHTML = `<tr><td colspan="4" style="padding: 20px; text-align: center; color: var(--vgt-text-dim);">Keine Logbucheinträge.</td></tr>`;
            return;
        }
        
        elKernelLogsTbody.innerHTML = logs.map(log => {
            const opClass = log.op.replace(" ", "_");
            return `
                <tr style="border-bottom: 1px solid rgba(255,255,255,0.03);">
                    <td style="padding: 6px 4px; color: var(--vgt-text-dim);">${log.timestamp}</td>
                    <td style="padding: 6px 4px;"><span class="log-op-badge ${opClass}">${log.op}</span></td>
                    <td style="padding: 6px 4px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 170px;" title="${log.target}">${log.target}</td>
                    <td style="padding: 6px 4px; text-align: right;" class="log-status ${log.status}">${log.status}</td>
                </tr>
            `;
        }).join("");
    } catch (e) {
        console.error("Failed to fetch kernel logs", e);
    }
}
