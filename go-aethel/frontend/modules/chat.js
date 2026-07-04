import { state } from './state.js';
import { speak, resetMicButton, stopSpeaking } from './voice.js';
import { openPermissionGate, fetchKernelLogs, refreshSecurityHUD } from './security.js';
import { updateMemoryCount } from './memory.js';

export function scrollToBottom() {
    const el = document.getElementById("chat-output");
    if (el) el.scrollTop = el.scrollHeight;
}

export function formatMarkdown(text) {
    if (!text) return "";
    
    let escaped = text
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;");
        
    const codeBlocks = [];
    escaped = escaped.replace(/```(.*?)```/gs, (match, p1) => {
        const placeholder = `___CODEBLOCK_${codeBlocks.length}___`;
        codeBlocks.push(`<pre class='tool-args'>${p1}</pre>`);
        return placeholder;
    });
    
    const lines = escaped.split("\n");
    let inTable = false;
    let tableHtml = "";
    const outputLines = [];
    
    for (let i = 0; i < lines.length; i++) {
        const line = lines[i].trim();
        
        if (line.includes("|")) {
            let cells = line.split("|").map(c => c.trim());
            
            if (cells[0] === "" && line.startsWith("|")) {
                cells.shift();
            }
            if (cells[cells.length - 1] === "" && line.endsWith("|")) {
                cells.pop();
            }
            
            const isSeparator = cells.length > 0 && cells.every(c => c.match(/^:?-+:?$/));
            
            if (isSeparator) {
                if (!inTable && outputLines.length > 0) {
                    const prevLine = outputLines.pop();
                    if (prevLine.includes("|")) {
                        let prevCells = prevLine.split("|").map(c => c.trim());
                        if (prevCells[0] === "" && prevLine.trim().startsWith("|")) prevCells.shift();
                        if (prevCells[prevCells.length - 1] === "" && prevLine.trim().endsWith("|")) prevCells.pop();
                        
                        inTable = true;
                        tableHtml = "<table class='cyber-table'><thead><tr>";
                        prevCells.forEach(c => { tableHtml += `<th>${c}</th>`; });
                        tableHtml += "</tr></thead><tbody>";
                    } else {
                        outputLines.push(prevLine);
                    }
                }
                continue;
            }
            
            if (inTable) {
                tableHtml += "<tr>";
                cells.forEach(c => { tableHtml += `<td>${c}</td>`; });
                tableHtml += "</tr>";
            } else {
                outputLines.push(lines[i]);
            }
        } else {
            if (inTable) {
                inTable = false;
                tableHtml += "</tbody></table>";
                outputLines.push(tableHtml);
                tableHtml = "";
            }
            outputLines.push(lines[i]);
        }
    }
    if (inTable) {
        tableHtml += "</tbody></table>";
        outputLines.push(tableHtml);
    }
    
    let finalHtml = outputLines.join("\n");
    
    for (let i = 0; i < codeBlocks.length; i++) {
        finalHtml = finalHtml.replace(`___CODEBLOCK_${i}___`, codeBlocks[i]);
    }
    
    finalHtml = finalHtml
        .replace(/\n/g, "<br>")
        .replace(/\*\*(.*?)\*\*/g, "<strong>$1</strong>")
        .replace(/`(.*?)`/g, "<code class='font-mono text-xs px-1 bg-black/40 rounded border border-white/5'>$1</code>");
        
    return finalHtml;
}

export function addMessageToScreen(role, content) {
    const elChatOutput = document.getElementById("chat-output");
    if (!elChatOutput) return 0;

    const messageDiv = document.createElement("div");
    messageDiv.className = `message ${role}`;
    
    let headerText = "SYSTEM // OVERWATCH";
    if (role === "user") headerText = "OPERATOR // TERMINAL";
    if (role === "assistant") headerText = "AETHEL // CORTEX";
    
    messageDiv.innerHTML = `
        <div class="msg-header">${headerText}</div>
        <div class="msg-body">${formatMarkdown(content)}</div>
    `;
    
    elChatOutput.appendChild(messageDiv);
    scrollToBottom();
    
    return elChatOutput.children.length - 1;
}

export function updateMessageBody(index, content) {
    const elChatOutput = document.getElementById("chat-output");
    if (!elChatOutput) return;

    const msgElement = elChatOutput.children[index];
    if (msgElement) {
        const body = msgElement.querySelector(".msg-body");
        if (body) {
            body.innerHTML = formatMarkdown(content);
        }
    }
    scrollToBottom();
}

export function showToolRequest(msgIndex, toolName, args) {
    const elChatOutput = document.getElementById("chat-output");
    if (!elChatOutput) return;

    const msgElement = elChatOutput.children[msgIndex];
    if (!msgElement) return;

    const box = document.createElement("div");
    box.className = "tool-request-box";
    box.id = `tool-box-${msgIndex}`;

    box.innerHTML = `
        <h5>AUTORISIERUNG ERFORDERLICH: [${toolName}]</h5>
        <pre class="tool-args">${JSON.stringify(args, null, 2)}</pre>
        <div class="tool-actions">
            <button class="btn-approve" onclick="approveTool(${msgIndex}, '${toolName}', ${JSON.stringify(args).replace(/"/g, '&quot;')}, false)">AUSFÜHREN</button>
            <button class="btn-reject" onclick="rejectTool(${msgIndex})">BLOCKIEREN</button>
        </div>
    `;

    msgElement.appendChild(box);
    scrollToBottom();
}

export function rejectTool(msgIndex) {
    const box = document.getElementById(`tool-box-${msgIndex}`);
    if (box) {
        box.innerHTML = `<span class="tool-status-badge status-rejected">SECURITY BLOCK: Ausführung vom Operator blockiert.</span>`;
    }
    
    state.messageHistory.push({
        role: "tool",
        tool_call_id: state.pendingToolCallId || ("call_" + Math.random().toString(36).substring(2, 9)),
        name: state.pendingToolCallName || "tool",
        content: "VGT SECURITY INTERVENTION: Operator hat die Ausführung blockiert."
    });
    saveChatHistory();
    state.pendingToolRequest = null;
    fetchKernelLogs();
    triggerNextInference();
}
window.rejectTool = rejectTool;

export async function executeApprovedTool(msgIndex, name, args, overrideSecurity = false) {
    const box = document.getElementById(`tool-box-${msgIndex}`);
    if (box) {
        box.innerHTML = `<div class="font-mono text-xs animate-pulse text-vgt-cyan">Sende Befehl an Kernel...</div>`;
    }

    if (name === "web_browser") {
        document.getElementById("browser-tab-title").textContent = "VERBINDE...";
        document.getElementById("browser-url-input").value = args.url || (args.search_query ? `Suche: ${args.search_query}` : "Lade...");
        document.getElementById("browser-placeholder").classList.remove("hidden");
        document.getElementById("browser-screenshot").classList.add("hidden");
    }

    try {
        const res = await fetch(`${state.API_BASE}/v1/tools/execute`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                name: name,
                args: args,
                override_security: overrideSecurity
            })
        });
        
        const data = await res.json();
        fetchKernelLogs();
        refreshSecurityHUD();

        if (data.status === "security_intervention") {
            if (box) {
                box.innerHTML = `<span class="tool-status-badge status-rejected" style="color: var(--vgt-orange)">SECURITY GATE: Bestätigung ausstehend...</span>`;
            }
            openPermissionGate(name, data.capability, data.risk_level, data.risk_score, data.threats, args, msgIndex);
            return;
        }

        if (data.status === "security_blocked") {
            if (box) {
                box.innerHTML = `<span class="tool-status-badge status-rejected">SECURITY BLOCK: Aktion permanent blockiert (FORBIDDEN).</span>`;
            }
            speak("Systembefehl wurde permanent blockiert.");

            const blockedMsg = `[VGT SECURITY BLOCK]: ${data.message || "Aktion verboten."}`;
            addMessageToScreen("system", blockedMsg);

            state.messageHistory.push({
                role: "tool",
                tool_call_id: state.pendingToolCallId || ("call_" + Math.random().toString(36).substring(2, 9)),
                name: name,
                content: "VGT SECURITY INTERVENTION: Aktion permanent blockiert (FORBIDDEN)."
            });
            saveChatHistory();
            state.pendingToolRequest = null;
            triggerNextInference();
            return;
        }

        const resultText = data.status === "success" ? data.result : `ERROR: ${data.error}`;
        
        if (box) {
            box.innerHTML = `<span class="tool-status-badge status-completed">EXECUTED SUCCESSFULLY</span>`;
        }

        if (name === "web_browser") {
            if (data.status === "success") {
                let title = "Geladene Seite";
                let finalURL = args.url || "";
                
                const resultStr = data.result;
                const urlMatch = resultStr.match(/URL:\s*(.*)/);
                const titleMatch = resultStr.match(/Titel:\s*(.*)/);
                
                if (urlMatch) finalURL = urlMatch[1].split("\n")[0].trim();
                if (titleMatch) title = titleMatch[1].split("\n")[0].trim();
                
                document.getElementById("browser-tab-title").textContent = title;
                document.getElementById("browser-url-input").value = finalURL;
                
                document.getElementById("browser-screenshot").src = `${state.API_BASE}/browser/screenshot.png?t=${Date.now()}`;
                document.getElementById("browser-screenshot").classList.remove("hidden");
                document.getElementById("browser-placeholder").classList.add("hidden");
            } else {
                document.getElementById("browser-tab-title").textContent = "Browser Fehler";
                document.getElementById("browser-url-input").value = "about:error";
            }
        }

        // Open Handoff Modal display if external delegation completes
        if (name === "agent_handoff" && data.status === "success") {
            try {
                const promptRes = await fetch(`${state.API_BASE}/v1/tools/execute`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name: "fs_read_file", args: { path: "./vgt_workspace/handoff_payload.md" } })
                });
                const promptData = await promptRes.json();
                if (promptData.status === "success") {
                    openHandoffModal(args.agent_name, promptData.result);
                }
            } catch(e) {
                console.error("Handoff payload fetch failed", e);
            }
        }

        const responseMsg = `[TOOL OUTPUT (${name})]: ${resultText}`;
        addMessageToScreen("system", responseMsg);
        
        state.messageHistory.push({
            role: "tool",
            tool_call_id: state.pendingToolCallId || ("call_" + Math.random().toString(36).substring(2, 9)),
            name: name,
            content: resultText
        });
        saveChatHistory();
        state.pendingToolRequest = null;

        triggerNextInference();

    } catch (e) {
        console.error("Tool exec error", e);
        if (box) {
            box.innerHTML = `<span class="tool-status-badge status-failed">CONNECTION FAILED: ${e.message}</span>`;
        }
        
        if (name === "web_browser") {
            document.getElementById("browser-tab-title").textContent = "Browser Fehler";
            document.getElementById("browser-url-input").value = "about:error";
        }
        fetchKernelLogs();
    }
}
window.approveTool = function(msgIndex, name, args, overrideSecurity = false) {
    executeApprovedTool(msgIndex, name, args, overrideSecurity);
};

export function openHandoffModal(agentName, content) {
    const modal = document.getElementById("handoff-modal");
    const badge = document.getElementById("handoff-agent-badge");
    const body = document.getElementById("handoff-modal-content");
    
    if (badge) badge.textContent = agentName.toUpperCase();
    if (body) body.textContent = content;
    
    if (modal) modal.classList.remove("hidden");
}

export function triggerNextInference() {
    const input = document.getElementById("user-input");
    if (input) {
        input.value = "...";
        sendMessage();
    }
}

export async function sendMessage() {
    const input = document.getElementById("user-input");
    if (!input) return;

    const prompt = input.value.trim();
    if (!prompt) return;

    if (state.pendingToolRequest) {
        const { handlePendingToolResponse } = await import('./voice.js');
        if (handlePendingToolResponse(prompt)) {
            input.value = "";
            return;
        }
    }

    stopSpeaking();

    if (state.recognition && state.isVoiceCallActive) {
        try { state.recognition.stop(); } catch(e) {}
    }

    input.value = "";
    
    addMessageToScreen("user", prompt);
    state.messageHistory.push({ role: "user", content: prompt });
    saveChatHistory();

    const elBtnMic = document.getElementById("btn-mic");
    const elSpeechIndicator = document.getElementById("speech-indicator");
    if (elBtnMic) elBtnMic.className = "mic-button processing";
    if (elSpeechIndicator) elSpeechIndicator.textContent = "Kortex verarbeitet Sequenz...";

    const msgIndex = addMessageToScreen("assistant", "");
    state.currentAssistantMsgIndex = msgIndex;
    let fullResponseText = "";
    let toolBuffer = { id: "", name: "", args: "" };

    try {
        const res = await fetch(`${state.API_BASE}/v1/chat`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                model_id: state.currentModel,
                messages: state.messageHistory,
                temperature: 0.15,
                use_tools: true,
                system_prompt: state.VGT_SYSTEM_PROTOCOL.trim()
            })
        });

        if (!res.body) {
            updateMessageBody(msgIndex, "[CORE ERROR]: Kein Stream empfangen.");
            resetMicButton();
            return;
        }

        const reader = res.body.getReader();
        const decoder = new TextDecoder("utf-8");
        let streamBuffer = "";

        while (true) {
            const { value, done } = await reader.read();
            if (done) break;

            streamBuffer += decoder.decode(value, { stream: true });
            const lines = streamBuffer.split("\n");
            streamBuffer = lines.pop();

            for (const line of lines) {
                if (line.startsWith("data:")) {
                    let data = line.slice(5);
                    if (line.startsWith("data: ")) data = line.slice(6);
                    if (data.endsWith("\r")) data = data.slice(0, -1);

                    if (data.startsWith("[[TOOL_DELTA]]:")) {
                        const jsonStr = data.slice(15).trim();
                        try {
                            const delta = JSON.parse(jsonStr);
                            const call = delta[0];
                            if (call.id) toolBuffer.id = call.id;
                            if (call.function?.name) toolBuffer.name += call.function.name;
                            if (call.function?.arguments) toolBuffer.args += call.function.arguments;
                        } catch (e) {
                            console.error("Tool Delta parsing failed", e);
                        }
                    }
                    else if (data.trim() === "[[TOOL_COMMIT]]") {
                        const toolCallId = toolBuffer.id || "call_" + Math.random().toString(36).substring(2, 9);
                        state.pendingToolCallId = toolCallId;
                        state.pendingToolCallName = toolBuffer.name;
                        
                        let parsedArgs = {};
                        try { parsedArgs = JSON.parse(toolBuffer.args || "{}"); } catch(e) {}
                        
                        state.messageHistory.push({
                            role: "assistant",
                            content: fullResponseText || null,
                            tool_calls: [
                                {
                                    id: toolCallId,
                                    type: "function",
                                    function: {
                                        name: toolBuffer.name,
                                        arguments: toolBuffer.args
                                    }
                                }
                            ]
                        });
                        saveChatHistory();
                        
                        showToolRequest(msgIndex, toolBuffer.name, parsedArgs);
                        
                        let speakName = toolBuffer.name;
                        if (speakName === "sys_exec_cmd") speakName = "Systembefehl";
                        if (speakName === "web_browser") speakName = "Webbrowser";
                        if (speakName === "fs_write_file") speakName = "Datei schreiben";
                        
                        state.pendingToolRequest = { msgIndex, name: toolBuffer.name, args: parsedArgs };
                        speak(`Freigabe für ${speakName} erforderlich. Bitte freigeben oder ablehnen sagen.`);
                        
                        toolBuffer = { id: "", name: "", args: "" };
                    }
                    else if (data.startsWith("[[USAGE]]:")) {
                        const jsonStr = data.slice(10).trim();
                        try {
                            const usage = JSON.parse(jsonStr);
                            const maxContext = 131072;
                            const pct = ((usage.total_tokens / maxContext) * 100).toFixed(2);
                            const elContext = document.getElementById("context-utilization");
                            if (elContext) {
                                elContext.textContent = `${usage.total_tokens.toLocaleString()} / 128k (${pct}%)`;
                            }
                        } catch (e) {
                            console.error("Usage parsing failed", e);
                        }
                    }
                    else if (data.startsWith("[SYSTEM ERROR]") || data.startsWith("[GROQ API ERROR")) {
                        fullResponseText += `\n${data}`;
                        updateMessageBody(msgIndex, fullResponseText);
                    }
                    else {
                        fullResponseText += data;
                        updateMessageBody(msgIndex, fullResponseText);
                    }
                }
            }
        }

        if (fullResponseText.trim()) {
            state.messageHistory.push({ role: "assistant", content: fullResponseText });
            saveChatHistory();
            speak(fullResponseText);
            
            if (fullResponseText.includes("[[CONTINUE]]")) {
                setTimeout(() => {
                    triggerNextInference();
                }, 2000);
            }
        }
        
    } catch (e) {
        console.error("Inference fetch error", e);
        updateMessageBody(msgIndex, `[CONNECTION TIMEOUT]: ${e.message}`);
    } finally {
        resetMicButton();
        updateMemoryCount();
    }
}

export async function loadSessionsList() {
    const elSessionsListContainer = document.getElementById("sessions-list-container");
    if (!elSessionsListContainer) return;
    try {
        const res = await fetch(`${state.API_BASE}/v1/chat/sessions`);
        const sessions = await res.json();
        
        if (!sessions || sessions.length === 0) {
            elSessionsListContainer.innerHTML = `<div style="color: var(--vgt-text-dim); font-size: 9px; text-align: center; padding: 10px 0;">Keine gespeicherten Chats.</div>`;
            return;
        }
        
        elSessionsListContainer.innerHTML = sessions.map(s => {
            const activeClass = (s.id === state.currentSessionId) ? "active" : "";
            return `
                <div class="session-item ${activeClass}" onclick="event.stopPropagation(); window.loadSession('${s.id}')">
                    <span class="session-title" title="${s.title}">${s.title}</span>
                    <span class="session-delete-btn font-mono" onclick="event.stopPropagation(); window.deleteSession('${s.id}')">&times;</span>
                </div>
            `;
        }).join("");
    } catch(e) {
        console.error("Failed to load sessions list", e);
    }
}

export async function loadSession(id) {
    if (!id) return;
    state.currentSessionId = id;
    
    document.querySelectorAll(".session-item").forEach(el => el.classList.remove("active"));
    const elChatOutput = document.getElementById("chat-output");
    
    try {
        const res = await fetch(`${state.API_BASE}/v1/chat/sessions/load?id=${id}`);
        const historyData = await res.json();
        
        if (elChatOutput) elChatOutput.innerHTML = "";
        state.messageHistory = historyData || [];
        
        state.messageHistory.forEach((msg) => {
            if (msg.role === "user" || msg.role === "assistant") {
                addMessageToScreen(msg.role, msg.content);
            } else if (msg.role === "system") {
                if (msg.content.startsWith("[TOOL OUTPUT") || msg.content.includes("VGT PROTOCOL INITIALIZED")) {
                    addMessageToScreen("system", msg.content);
                }
            } else if (msg.role === "tool") {
                addMessageToScreen("system", `[TOOL OUTPUT (${msg.name})]: ${msg.content}`);
            }
        });
        
        scrollToBottom();
        loadSessionsList();
    } catch(e) {
        console.error("Failed to load session", e);
    }
}
window.loadSession = loadSession;

export function startNewSession() {
    state.currentSessionId = "session_" + Date.now();
    
    const elChatOutput = document.getElementById("chat-output");
    if (elChatOutput) {
        elChatOutput.innerHTML = `
            <div class="system-message font-mono">
                <p>[VGT PROTOCOL INITIALIZED]</p>
                <p>Souveräne Terminal-Schnittstelle betriebsbereit. Neuer Impuls gestartet.</p>
            </div>
        `;
    }
    state.messageHistory = [];
    saveChatHistory();
}

export async function deleteSession(id) {
    try {
        await fetch(`${state.API_BASE}/v1/chat/sessions/delete?id=${id}`, {
            method: 'DELETE'
        });
        
        if (state.currentSessionId === id) {
            startNewSession();
        } else {
            loadSessionsList();
        }
    } catch(e) {
        console.error("Failed to delete session", e);
    }
}
window.deleteSession = deleteSession;

export async function loadChatHistory() {
    try {
        const res = await fetch(`${state.API_BASE}/v1/chat/sessions`);
        const sessions = await res.json();
        
        if (sessions && sessions.length > 0) {
            await loadSession(sessions[0].id);
        } else {
            startNewSession();
        }
    } catch(e) {
        console.error("Failed to bootstrap chat sessions", e);
        startNewSession();
    }
}

export async function saveChatHistory() {
    if (!state.currentSessionId) {
        state.currentSessionId = "session_" + Date.now();
    }
    try {
        await fetch(`${state.API_BASE}/v1/chat/sessions/save`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                id: state.currentSessionId,
                messages: state.messageHistory
            })
        });
        loadSessionsList();
    } catch(e) {
        console.error("Failed to save session chat history", e);
    }
}
