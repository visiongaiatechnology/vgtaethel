import { state } from './state.js';
import * as api from './api.js';

export async function fetchSecretsList() {
    const container = document.getElementById("secrets-list-container");
    if (!container) return;

    try {
        const secrets = await api.getSecrets();

        if (!secrets || secrets.length === 0) {
            container.innerHTML = `<div style="color: var(--vgt-text-dim); text-align: center; margin-top: 30px;">Keine sensitiven Einträge vorhanden.</div>`;
            return;
        }

        container.innerHTML = secrets.map(s => {
            const dateStr = s.created_at ? new Date(s.created_at).toLocaleDateString() : "unbekannt";
            return `
                <div class="glass-card" style="padding: 10px 15px; display: flex; justify-content: space-between; align-items: center; background: rgba(157, 78, 221, 0.02); border-color: rgba(157, 78, 221, 0.15); margin-bottom: 6px;">
                    <div style="text-align: left; max-width: 80%;">
                        <div style="font-size: 11px; font-weight: bold; color: var(--vgt-purple);">${s.id}</div>
                        <div style="font-size: 8px; color: var(--vgt-text-dim); margin-top: 4px;">Service: ${s.service} | Typ: ${s.type} | Erstellt: ${dateStr}</div>
                        <div style="font-size: 8px; color: var(--vgt-green); margin-top: 2px;">🔒 ENCRYPTED IN LOCAL AES VAULT</div>
                    </div>
                    <button class="cyber-button font-mono" onclick="deleteSecretItem('${s.id}')" style="width: auto; padding: 4px 10px; font-size: 8px; background: rgba(255, 0, 79, 0.1); border: 1px solid var(--vgt-red); color: var(--vgt-red);">LÖSCHEN</button>
                </div>
            `;
        }).join("");
    } catch(e) {
        console.error("Failed to load secrets list", e);
    }
}

export async function addSecretItem() {
    const elId = document.getElementById("secr-add-id");
    const elService = document.getElementById("secr-add-service");
    const elType = document.getElementById("secr-add-type");
    const elToken = document.getElementById("secr-add-token");
    const elApproval = document.getElementById("secr-add-approval");

    if (!elId || !elToken) return;

    const id = elId.value.trim();
    const service = elService ? elService.value.trim() : "";
    const type = elType ? elType.value : "api_token";
    const token = elToken.value.trim();
    const requiresApproval = elApproval ? elApproval.checked : true;

    if (!id || !token) {
        alert("Schlüssel ID und Klartext Secret Wert sind erforderlich.");
        return;
    }

    try {
        const res = await api.addSecret({
            id,
            service,
            type,
            token,
            requires_approval: requiresApproval
        });

        if (res.status === "success") {
            elId.value = "";
            if (elService) elService.value = "";
            elToken.value = "";
            fetchSecretsList();
        } else {
            alert(`Fehler beim Speichern: ${res.message || "Unbekannter Fehler"}`);
        }
    } catch(e) {
        console.error("Failed to save secret item", e);
    }
}

export async function deleteSecretItem(id) {
    try {
        const res = await api.deleteSecret(id);
        if (res.status === "success") {
            fetchSecretsList();
        } else {
            alert(`Löschen des Secrets fehlgeschlagen: ${res.message}`);
        }
    } catch(e) {
        console.error("Failed to delete secret item", e);
    }
}
window.deleteSecretItem = deleteSecretItem;

export function setupSecretsUIEvents() {
    const btnAdd = document.getElementById("secr-btn-add");
    if (btnAdd) {
        btnAdd.addEventListener("click", addSecretItem);
    }
}
