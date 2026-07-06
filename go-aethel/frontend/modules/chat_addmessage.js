export function formatMarkdown(text) {
    if (!text) return "";
    if (typeof window.marked !== 'undefined') {
        try {
            return window.marked.parse(text);
        } catch(e) {
            console.error("marked.parse error", e);
        }
    }
    return text.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/\n/g,"<br>");
}

export function addMessageToScreen(role, content, reasoning_content = null) {
    const elChatOutput = document.getElementById("chat-output");
    if (!elChatOutput) return 0;

    const messageDiv = document.createElement("div");
    messageDiv.className = `message ${role}`;
    
    let headerText = "SYSTEM // OVERWATCH";
    if (role === "user") headerText = "OPERATOR // TERMINAL";
    if (role === "assistant") headerText = "AETHEL // CORTEX";
    
    let bodyHtml = "";
    if (role === "assistant" && reasoning_content) {
        bodyHtml += `
            <details class="thinking-details" style="margin-bottom: 12px; background: rgba(0,200,255,0.03); border: 1px solid rgba(0,200,255,0.15); border-radius: 4px; padding: 8px;">
                <summary style="font-size: 10px; color: #00c8ff; cursor: pointer; font-family: var(--font-mono); outline: none; user-select: none;">
                    🧠 THOUGHT PROCESS (Click to expand)
                </summary>
                <div class="thinking-content" style="font-size: 11px; color: rgba(255,255,255,0.6); font-family: var(--font-mono); margin-top: 6px; white-space: pre-wrap; line-height: 1.4;">${reasoning_content}</div>
            </details>
        `;
    }
    
    const renderedContent = formatMarkdown(content);
    const isLong = content && content.length > 600;
    
    if (isLong) {
        bodyHtml += `<div class="msg-collapsible">`;
        bodyHtml += `<div class="msg-collapsed">${renderedContent.slice(0, 300)}...</div>`;
        bodyHtml += `<div class="msg-full" style="display:none;">${renderedContent}</div>`;
        bodyHtml += `<button class="msg-toggle-btn" style="font-size:9px;color:var(--vgt-cyan);background:rgba(0,240,255,0.05);border:1px solid rgba(0,240,255,0.2);padding:4px 12px;border-radius:4px;cursor:pointer;margin-top:8px;font-family:var(--font-mono);">▶ ENTFALTEN (${content.length} Zeichen)</button>`;
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
