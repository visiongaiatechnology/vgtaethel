import { state } from './state.js';

export async function fetchMemoriesList(query = "") {
    const container = document.getElementById("mem-list-container");
    if (!container) return;

    try {
        let memories = [];

        if (query) {
            // POST request to memory search endpoint
            const res = await fetch(`${state.API_BASE}/v1/memory/search`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ query: query })
            });
            memories = await res.json();
        } else {
            const res = await fetch(`${state.API_BASE}/v1/memory`);
            memories = await res.json();
        }

        if (!memories || memories.length === 0) {
            container.innerHTML = `<div style="color: var(--vgt-text-dim); text-align: center; margin-top: 30px;">Keine Erinnerungen im Gedächtnis gefunden.</div>`;
            return;
        }

        container.innerHTML = memories.map(m => {
            const dateStr = m.timestamp ? new Date(m.timestamp).toLocaleDateString() : "";
            let displayContent = m.content;
            if (m.category === "secret_reference" || displayContent.toLowerCase().includes("token") || displayContent.toLowerCase().includes("password") || displayContent.toLowerCase().includes("key")) {
                displayContent = "🔑 [SENSITIVER INHALT - VERBORGEN]";
            }
            return `
                <div class="glass-card" style="padding: 12px 18px; display: flex; justify-content: space-between; align-items: center; background: rgba(255,255,255,0.01); margin-bottom: 6px;">
                    <div style="text-align: left; max-width: 80%;">
                        <span class="log-op-badge" style="background: rgba(0, 240, 255, 0.1); color: var(--vgt-cyan); padding: 2px 5px; border-radius: 4px; font-size: 8px; font-weight: bold; text-transform: uppercase;">${m.category}</span>
                        <div style="font-size: 11px; margin-top: 6px; color: #fff;">${displayContent}</div>
                        ${dateStr ? `<div style="font-size: 8px; color: var(--vgt-text-dim); margin-top: 4px;">Gespeichert am: ${dateStr}</div>` : ""}
                    </div>
                    <button class="cyber-button font-mono" onclick="deleteMemoryItem('${m.id}')" style="width: auto; padding: 4px 10px; font-size: 8px; background: rgba(255, 0, 79, 0.1); border: 1px solid var(--vgt-red); color: var(--vgt-red);">LÖSCHEN</button>
                </div>
            `;
        }).join("");
    } catch(e) {
        console.error("Failed to load memories list", e);
    }
}

export async function addMemoryItem() {
    const elContent = document.getElementById("mem-add-content");
    const elCategory = document.getElementById("mem-add-category");
    if (!elContent) return;

    const content = elContent.value.trim();
    const category = elCategory ? elCategory.value : "general";

    if (!content) {
        alert("Erinnerung darf nicht leer sein.");
        return;
    }

    try {
        const res = await fetch(`${state.API_BASE}/v1/memory`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ content: content, category: category })
        });
        const data = await res.json();
        if (data.status === "success") {
            elContent.value = "";
            fetchMemoriesList();
            updateMemoryCount();
        } else {
            alert(`Fehler beim Speichern: ${data.message}`);
        }
    } catch(e) {
        console.error("Failed to save memory item", e);
    }
}

export async function deleteMemoryItem(id) {
    try {
        const res = await fetch(`${state.API_BASE}/v1/memory?id=${id}`, {
            method: 'DELETE'
        });
        const data = await res.json();
        if (data.status === "success") {
            fetchMemoriesList();
            updateMemoryCount();
        } else {
            alert(`Löschen fehlgeschlagen: ${data.message}`);
        }
    } catch(e) {
        console.error("Failed to delete memory item", e);
    }
}
window.deleteMemoryItem = deleteMemoryItem;

export async function updateMemoryCount() {
    const elMemoryCount = document.getElementById("memory-count");
    if (!elMemoryCount) return;

    try {
        const res = await fetch(`${state.API_BASE}/v1/memory`);
        const data = await res.json();
        if (Array.isArray(data)) {
            elMemoryCount.textContent = `${data.length} Einträge`;
        } else {
            elMemoryCount.textContent = `0 Einträge`;
        }
    } catch (e) {
        elMemoryCount.textContent = `Offline`;
    }
}

export function setupMemoryUIEvents() {
    const btnSearch = document.getElementById("mem-btn-search");
    const btnAdd = document.getElementById("mem-btn-add");
    const inputSearch = document.getElementById("mem-search-query");

    if (btnSearch) {
        btnSearch.addEventListener("click", () => {
            const query = inputSearch ? inputSearch.value.trim() : "";
            fetchMemoriesList(query);
        });
    }
    if (btnAdd) {
        btnAdd.addEventListener("click", addMemoryItem);
    }
}
