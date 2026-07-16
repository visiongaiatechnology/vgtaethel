import { state } from './state.js';
import * as api from './api.js';

function renderVaultState(container, message, error = false) {
    const stateElement = document.createElement('div');
    stateElement.className = `memory-empty-state${error ? ' error' : ''}`;
    stateElement.textContent = message;
    container.replaceChildren(stateElement);
}

export async function fetchSecretsList() {
    const container = document.getElementById("secrets-list-container");
    if (!container) return;

    try {
        const secrets = await api.getSecrets();

        if (!secrets || secrets.length === 0) {
            renderVaultState(container, 'Keine sensitiven Einträge vorhanden.');
            return;
        }

        const cards = secrets.map(secret => {
            const card = document.createElement('article');
            card.className = 'vault-entry-card';
            const copy = document.createElement('div');
            const title = document.createElement('strong');
            title.textContent = String(secret.id || 'unnamed-secret');
            const createdAt = new Date(secret.created_at);
            const meta = document.createElement('span');
            meta.textContent = `Service: ${String(secret.service || '—')} · Typ: ${String(secret.type || '—')} · Erstellt: ${Number.isNaN(createdAt.getTime()) ? '—' : createdAt.toLocaleDateString()}`;
            const encrypted = document.createElement('small');
            encrypted.textContent = '🔒 ENCRYPTED IN LOCAL VAULT';
            copy.append(title, meta, encrypted);
            const remove = document.createElement('button');
            remove.type = 'button';
            remove.className = 'memory-delete-button';
            remove.textContent = 'Löschen';
            remove.addEventListener('click', () => { void deleteSecretItem(String(secret.id || '')); });
            card.append(copy, remove);
            return card;
        });
        container.replaceChildren(...cards);
    } catch(e) {
        console.error("Failed to load secrets list", e);
        renderVaultState(container, 'Secret Vault nicht erreichbar.', true);
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
        window.showAethelToast?.('Schlüssel-ID und Secret-Wert sind erforderlich.', 'error');
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
            window.showAethelToast?.(`Fehler beim Speichern: ${res.message || "Unbekannter Fehler"}`, 'error');
        }
    } catch(e) {
        console.error("Failed to save secret item", e);
        window.showAethelToast?.('Secret konnte nicht gespeichert werden.', 'error');
    }
}

export async function deleteSecretItem(id) {
    try {
        const res = await api.deleteSecret(id);
        if (res.status === "success") {
            fetchSecretsList();
        } else {
            window.showAethelToast?.(`Löschen des Secrets fehlgeschlagen: ${res.message || 'Unbekannter Fehler'}`, 'error');
        }
    } catch(e) {
        console.error("Failed to delete secret item", e);
        window.showAethelToast?.('Secret konnte nicht gelöscht werden.', 'error');
    }
}
window.deleteSecretItem = deleteSecretItem;

export function setupSecretsUIEvents() {
    const btnAdd = document.getElementById("secr-btn-add");
    if (btnAdd) {
        btnAdd.addEventListener("click", addSecretItem);
    }
}
