export function formatMarkdown(text) {
    if (!text) return "";
    if (typeof window.marked !== 'undefined') {
        try {
            return sanitizeHtml(window.marked.parse(text));
        } catch(e) {
            console.error("marked.parse error", e);
        }
    }
    return text.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/\n/g,"<br>");
}

function escapeHtml(value) {
    return String(value)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

function sanitizeHtml(html) {
    const doc = new DOMParser().parseFromString(`<div>${html}</div>`, "text/html");
    const blockedTags = new Set(["script", "style", "iframe", "object", "embed", "link", "meta", "base", "form"]);
    doc.body.querySelectorAll("*").forEach((node) => {
        if (blockedTags.has(node.tagName.toLowerCase())) {
            node.remove();
            return;
        }
        [...node.attributes].forEach((attr) => {
            const name = attr.name.toLowerCase();
            const value = attr.value.trim().toLowerCase();
            if (name.startsWith("on") || name === "srcdoc" || value.startsWith("javascript:") || value.startsWith("data:text/html")) {
                node.removeAttribute(attr.name);
            }
        });
    });
    return doc.body.firstElementChild ? doc.body.firstElementChild.innerHTML : "";
}

export function addMessageToScreen(role, content, reasoning_content = null) {
    const elChatOutput = document.getElementById("chat-output");
    if (!elChatOutput) return 0;

    const messageDiv = document.createElement("div");
    messageDiv.className = `message ${role}`;
    
    let headerText = "System";
    if (role === "user") headerText = "Du";
    if (role === "assistant") headerText = "Aethel";
    
    let bodyHtml = "";
    if (role === "assistant" && reasoning_content) {
        bodyHtml += `
            <details class="thinking-details">
                <summary class="vgt-inline-f7b06c8b">
                    🧠 Denkprozess anzeigen
                </summary>
                <div class="thinking-content vgt-inline-d3a143ff">${escapeHtml(reasoning_content)}</div>
            </details>
        `;
    }
    
    const renderedContent = formatMarkdown(content);
    const isLong = content && content.length > 600;
    
    if (isLong) {
        bodyHtml += `<div class="msg-collapsible">`;
        bodyHtml += `<div class="msg-collapsed">${renderedContent.slice(0, 300)}...</div>`;
        bodyHtml += `<div class="msg-full vgt-inline-7830d708">${renderedContent}</div>`;
        bodyHtml += `<button class="msg-toggle-btn vgt-inline-6b439284">▶ ENTFALTEN (${content.length} Zeichen)</button>`;
        bodyHtml += `</div>`;
    } else {
        bodyHtml += renderedContent;
    }

    messageDiv.innerHTML = `
        <div class="msg-header">${headerText}</div>
        <div class="msg-body">${bodyHtml}</div>
    `;
    
    elChatOutput.appendChild(messageDiv);
    
    // --- CLICK-HANDLER für ENTFALTEN ---
    const toggleBtn = messageDiv.querySelector('.msg-toggle-btn');
    if (toggleBtn) {
        toggleBtn.addEventListener('click', function() {
            const collapsed = messageDiv.querySelector('.msg-collapsed');
            const full = messageDiv.querySelector('.msg-full');
            if (collapsed && full) {
                const isHidden = full.style.display === 'none';
                full.style.display = isHidden ? 'block' : 'none';
                collapsed.style.display = isHidden ? 'none' : 'block';
                this.textContent = isHidden ? '▼ EINKLAPPEN' : `▶ ENTFALTEN (${content.length} Zeichen)`;
            }
        });
    }
    
    // --- MAX 10 NACHRICHTEN IM DOM ---
    while (elChatOutput.children.length > 10) {
        if (elChatOutput.firstChild) {
            elChatOutput.removeChild(elChatOutput.firstChild);
        }
    }
    
    scrollToBottom();
    return elChatOutput.children.length - 1;
}
