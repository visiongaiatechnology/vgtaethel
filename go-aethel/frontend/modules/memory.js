import { state } from './state.js';

function renderMemoryState(container, message, error = false) {
    const stateElement = document.createElement('div');
    stateElement.className = `memory-empty-state${error ? ' error' : ''}`;
    stateElement.textContent = message;
    container.replaceChildren(stateElement);
}

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
            if (!res.ok) throw new Error(`memory search unavailable (${res.status})`);
            memories = await res.json();
        } else {
            const res = await fetch(`${state.API_BASE}/v1/memory`);
            if (!res.ok) throw new Error(`memory unavailable (${res.status})`);
            memories = await res.json();
        }

        if (!memories || memories.length === 0) {
            renderMemoryState(container, query ? 'Keine passenden Erinnerungen gefunden.' : 'Noch keine Erinnerungen gespeichert.');
            return;
        }

        const cards = memories.map(memory => {
            const card = document.createElement('article');
            card.className = 'memory-entry-card';
            const copy = document.createElement('div');
            const category = document.createElement('span');
            category.className = 'memory-category-badge';
            category.textContent = String(memory.category || 'general');
            const content = document.createElement('p');
            let displayContent = String(memory.content || '');
            const lowerContent = displayContent.toLowerCase();
            if (memory.category === 'secret_reference' || lowerContent.includes('token') || lowerContent.includes('password') || lowerContent.includes('key')) {
                displayContent = '🔑 [SENSITIVER INHALT – VERBORGEN]';
            }
            content.textContent = displayContent;
            copy.append(category, content);
            if (memory.timestamp) {
                const timestamp = new Date(memory.timestamp);
                const date = document.createElement('time');
                date.dateTime = Number.isNaN(timestamp.getTime()) ? '' : timestamp.toISOString();
                date.textContent = `Gespeichert: ${Number.isNaN(timestamp.getTime()) ? '—' : timestamp.toLocaleDateString()}`;
                copy.appendChild(date);
            }
            const remove = document.createElement('button');
            remove.type = 'button';
            remove.className = 'memory-delete-button';
            remove.textContent = 'Löschen';
            remove.addEventListener('click', () => { void deleteMemoryItem(String(memory.id || '')); });
            card.append(copy, remove);
            return card;
        });
        container.replaceChildren(...cards);
    } catch(e) {
        console.error("Failed to load memories list", e);
        renderMemoryState(container, 'Gedächtnis-Core nicht erreichbar.', true);
    }
}

export async function addMemoryItem() {
    const elContent = document.getElementById("mem-add-content");
    const elCategory = document.getElementById("mem-add-category");
    if (!elContent) return;

    const content = elContent.value.trim();
    const category = elCategory ? elCategory.value : "general";

    if (!content) {
        window.showAethelToast?.('Erinnerung darf nicht leer sein.', 'error');
        return;
    }

    try {
        const res = await fetch(`${state.API_BASE}/v1/memory`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ content: content, category: category, source: "operator", operator_approved: true })
        });
        if (!res.ok) throw new Error(`memory save rejected (${res.status})`);
        const data = await res.json();
        if (data.status === "success") {
            elContent.value = "";
            fetchMemoriesList();
            updateMemoryCount();
        } else {
            window.showAethelToast?.(`Fehler beim Speichern: ${data.message || 'Unbekannter Fehler'}`, 'error');
        }
    } catch(e) {
        console.error("Failed to save memory item", e);
        window.showAethelToast?.('Erinnerung konnte nicht gespeichert werden.', 'error');
    }
}

export async function deleteMemoryItem(id) {
    try {
        const res = await fetch(`${state.API_BASE}/v1/memory?id=${encodeURIComponent(id)}`, {
            method: 'DELETE'
        });
        if (!res.ok) throw new Error(`memory delete rejected (${res.status})`);
        const data = await res.json();
        if (data.status === "success") {
            fetchMemoriesList();
            updateMemoryCount();
        } else {
            window.showAethelToast?.(`Löschen fehlgeschlagen: ${data.message || 'Unbekannter Fehler'}`, 'error');
        }
    } catch(e) {
        console.error("Failed to delete memory item", e);
        window.showAethelToast?.('Erinnerung konnte nicht gelöscht werden.', 'error');
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
    inputSearch?.addEventListener('keydown', event => {
        if (event.key !== 'Enter') return;
        event.preventDefault();
        void fetchMemoriesList(inputSearch.value.trim());
    });
    if (btnAdd) {
        btnAdd.addEventListener("click", addMemoryItem);
    }
}
