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
    if (typeof window.marked !== "undefined") {
        try {
            // GFM explizit einschalten für Tabellen-Unterstützung
            return window.marked.parse(text, { gfm: true });
        } catch(e) {
            console.error("marked error", e);
        }
    }
    // Fallback: Einfache Markdown-Tabellen manuell rendern
    return simpleTableFallback(text);
}

/**
 * Fallback für Markdown-Tabellen, falls marked nicht lädt
 */
function simpleTableFallback(text) {
    // Erstmal HTML-Entities escapen
    let result = text.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;");
    
    // Einfache Markdown-Tabelle erkennen (mindestens 2 Zeilen mit |)
    const lines = result.split("\n");
    let inTable = false;
    let tableLines = [];
    let output = [];
    
    for (let i = 0; i < lines.length; i++) {
        const line = lines[i];
        const trimmed = line.trim();
        
        // Tabellen-Zeile erkennen: enthält | und ist keine reine Trennzeile
        const isTableRow = trimmed.includes("|") && !/^[\s\|:\-]+$/.test(trimmed);
        const isSeparator = trimmed.includes("|") && /^[\s\|:\-]+$/.test(trimmed);
        
        if (isTableRow || isSeparator) {
            if (!inTable) {
                inTable = true;
                tableLines = [];
            }
            // Trennzeile überspringen
            if (!isSeparator) {
                tableLines.push(trimmed);
            }
        } else {
            if (inTable) {
                // Tabelle beenden und rendern
                output.push(renderSimpleTable(tableLines));
                inTable = false;
                tableLines = [];
            }
            output.push(line);
        }
    }
    
    // Offene Tabelle am Ende schließen
    if (inTable) {
        output.push(renderSimpleTable(tableLines));
    }
    
    return output.join("\n").replace(/\n/g, "<br>");
}

function renderSimpleTable(rows) {
    if (rows.length < 2) return rows.join("\n");
    
    let html = '<table class="cyber-table">';
    
    // Erste Zeile = Header
    const headerCells = splitRow(rows[0]);
    html += '<thead><tr>';
    for (const cell of headerCells) {
        html += `<th>${cell.trim()}</th>`;
    }
    html += '</tr></thead>';
    
    // Rest = Body
    html += '<tbody>';
    for (let i = 1; i < rows.length; i++) {
        const cells = splitRow(rows[i]);
        html += '<tr>';
        for (const cell of cells) {
            html += `<td>${cell.trim()}</td>`;
        }
        html += '</tr>';
    }
    html += '</tbody></table>';
    
    return html;
}

function splitRow(row) {
    // Entferne führende/schließende Pipe und splitte
    const cleaned = row.replace(/^\||\|$/g, '').trim();
    const cells = [];
    let current = '';
    let inEscape = false;
    
    for (const ch of cleaned) {
        if (ch === '\\' && !inEscape) {
            inEscape = true;
            continue;
        }
        if (ch === '|' && !inEscape) {
            cells.push(current.trim());
            current = '';
            continue;
        }
        if (inEscape) {
            current += '\\';
            inEscape = false;
        }
        current += ch;
    }
    cells.push(current.trim());
    return cells;
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
        bodyHtml += `<details class="thinking-details" style="margin-bottom:12px;background:rgba(0,200,255,0.03);border:1px solid rgba(0,200,255,0.15);border-radius:4px;padding:8px;"><summary style="font-size:10px;color:#00c8ff;cursor:pointer;font-family:var(--font-mono);outline:none;user-select:none;">\u{1F9E0} THOUGHT PROCESS</summary><div class="thinking-content" style="font-size:11px;color:rgba(255,255,255,0.6);font-family:var(--font-mono);margin-top:6px;white-space:pre-wrap;line-height:1.4;">${reasoning_content}</div></details>`;
    }
    const rendered = formatMarkdown(content);
    if (content && content.length > 600) {
        bodyHtml += `<div class="msg-collapsible collapsed"><div class="msg-content">${rendered}</div><div class="msg-fade-overlay"></div><button class="msg-toggle-btn" style="font-size:9px;color:var(--vgt-cyan);background:rgba(0,240,255,0.05);border:1px solid rgba(0,240,255,0.2);padding:4px 12px;border-radius:4px;cursor:pointer;margin-top:8px;font-family:var(--font-mono);">\u25B6 ENTFALTEN (${content.length})</button></div>`;
    } else {
        bodyHtml += rendered;
    }
    messageDiv.innerHTML = `<div class="msg-header">${headerText}</div><div class="msg-body">${bodyHtml}</div>`;
    elChatOutput.appendChild(messageDiv);
    const toggle = messageDiv.querySelector('.msg-toggle-btn');
    if (toggle) {
        toggle.addEventListener('click', function() {
            const container = messageDiv.querySelector('.msg-collapsible');
            if (container) {
                const isCollapsed = container.classList.contains('collapsed');
                if (isCollapsed) {
                    container.classList.remove('collapsed');
                    this.textContent = '\u25BC EINGEKLAPPT';
                } else {
                    container.classList.add('collapsed');
                    this.textContent = `\u25B6 ENTFALTEN (${content.length})`;
                }
            }
        });
    }
    while (elChatOutput.children.length > 10) {
        if (elChatOutput.firstChild) elChatOutput.removeChild(elChatOutput.firstChild);
    }
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

export function updateMessageWithThinking(index, thinking, content, startTime, isFinished = false) {
    const elChatOutput = document.getElementById("chat-output");
    if (!elChatOutput) return;

    const msgElement = elChatOutput.children[index];
    if (msgElement) {
        const body = msgElement.querySelector(".msg-body");
        if (body) {
            let html = "";
            if (thinking || !isFinished) {
                let elapsedText = "";
                if (startTime) {
                    const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
                    elapsedText = ` - ${elapsed}s`;
                }
                const openAttr = isFinished ? "" : "open";
                const dots = isFinished ? "" : `
                    <span class="thinking-loader-dots" style="margin-left: 6px; display: inline-flex; gap: 2px;">
                        <span style="color: var(--vgt-cyan);">●</span>
                        <span style="color: var(--vgt-cyan);">●</span>
                        <span style="color: var(--vgt-cyan);">●</span>
                    </span>
                `;
                const contentText = thinking ? thinking : "Aethel analysiert Aufgabe...";
                html += `
                    <details class="thinking-details" ${openAttr} style="margin-bottom: 12px; background: rgba(0,200,255,0.03); border: 1px solid rgba(0,200,255,0.15); border-radius: 4px; padding: 8px;">
                        <summary style="font-size: 10px; color: #00c8ff; cursor: pointer; font-family: var(--font-mono); outline: none; user-select: none; display: flex; align-items: center;">
                            🧠 GEDANKENGANG${elapsedText}${dots}
                        </summary>
                        <div class="thinking-content" style="font-size: 11px; color: rgba(255,255,255,0.6); font-family: var(--font-mono); margin-top: 6px; white-space: pre-wrap; line-height: 1.4;">${contentText}</div>
                    </details>
                `;
            }
            let renderedContent = formatMarkdown(content);
            if (!isFinished && content) {
                const cursorHTML = `<span class="chat-cursor" style="display: inline-block; width: 6px; height: 14px; background: var(--vgt-cyan, #00f0ff); margin-left: 4px; animation: chat-cursor-blink 0.8s step-start infinite; vertical-align: middle;"></span><style>@keyframes chat-cursor-blink { 50% { opacity: 0; } }</style>`;
                const lastTagRegex = /(<\/p>|<\/li>|<\/code>|<\/td>|<\/div>|<\/pre>|<\/h[1-6]>)(<\/[a-z]+>|\s)*$/i;
                if (lastTagRegex.test(renderedContent)) {
                    renderedContent = renderedContent.replace(lastTagRegex, (match, p1, p2) => cursorHTML + p1 + (p2 || ""));
                } else {
                    renderedContent += cursorHTML;
                }
            }
            html += renderedContent;
            body.innerHTML = html;
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
            <button class="btn-approve" onclick="approveTool(${msgIndex}, '${toolName}', ${JSON.stringify(args).replace(/"/g, '&quot;')}, true)">AUSFÜHREN</button>
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

        if (data.status === "success") {
            if (data.checklist) {
                updateChecklistUI(data.checklist);
            }
            if (data.file_changes) {
                if (!window.currentFileChanges) window.currentFileChanges = [];
                data.file_changes.forEach(c => {
                    const existing = window.currentFileChanges.find(x => x.path === c.path);
                    if (existing) {
                        existing.added += c.added;
                        existing.removed += c.removed;
                    } else {
                        window.currentFileChanges.push(c);
                    }
                });
            }
        }

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
    sendMessage(true);
}

export async function sendMessage(isContinuation = false) {
    const input = document.getElementById("user-input");
    if (!input) return;

    if (!isContinuation) {
        window.currentFileChanges = [];
    }

    let prompt = "";
    if (!isContinuation) {
        prompt = input.value.trim();
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
    } else {
        stopSpeaking();
    }

    const elBtnMic = document.getElementById("btn-mic");
    const elSpeechIndicator = document.getElementById("speech-indicator");
    if (elBtnMic) elBtnMic.className = "mic-button processing";
    if (elSpeechIndicator) elSpeechIndicator.textContent = "Kortex verarbeitet Sequenz...";

    const msgIndex = addMessageToScreen("assistant", "");
    state.currentAssistantMsgIndex = msgIndex;
    let fullResponseText = "";
    let thinkingText = "";
    let toolBuffer = { id: "", name: "", args: "" };
    let pendingAutoExecute = null;
    let hasCommittedToolCall = false;
    const inferenceStartTime = Date.now();
    let thinkingTimer = null;
    // Start periodic timer to update elapsed thinking time and keep UI alive/responsive
    thinkingTimer = setInterval(() => {
        updateMessageWithThinking(msgIndex, thinkingText, fullResponseText, inferenceStartTime);
    }, 100);

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
                            if (call.function?.name) toolBuffer.name = call.function.name;
                            if (call.function?.arguments) toolBuffer.args += call.function.arguments;
                        } catch (e) {
                            console.error("Tool Delta parsing failed", e);
                        }
                    }
                    else if (data.trim() === "[[TOOL_COMMIT]]") {
                        const toolCallId = toolBuffer.id || "call_" + Math.random().toString(36).substring(2, 9);
                        state.pendingToolCallId = toolCallId;
                        state.pendingToolCallName = toolBuffer.name;
                        
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
                        hasCommittedToolCall = true;
                        
                        let parsedArgs = {};
                        try { parsedArgs = JSON.parse(toolBuffer.args || "{}"); } catch(e) {}
                        
                        const isSafeTool = (toolBuffer.name === "nexus_save" || toolBuffer.name === "nexus_recall");
                        
                        if (state.isFullAutonomy || isSafeTool) {
                            showToolRequest(msgIndex, toolBuffer.name, parsedArgs);
                            pendingAutoExecute = { msgIndex, name: toolBuffer.name, args: parsedArgs };
                        } else {
                            showToolRequest(msgIndex, toolBuffer.name, parsedArgs);
                            
                            let speakName = toolBuffer.name;
                            if (speakName === "sys_exec_cmd") speakName = "Systembefehl";
                            if (speakName === "web_browser") speakName = "Webbrowser";
                            if (speakName === "fs_write_file") speakName = "Datei schreiben";
                            
                            state.pendingToolRequest = { msgIndex, name: toolBuffer.name, args: parsedArgs };
                            speak(`Freigabe für ${speakName} erforderlich. Bitte freigeben oder ablehnen sagen.`);
                        }
                        
                        toolBuffer = { id: "", name: "", args: "" };
                    }
                    else if (data.startsWith("[[THINKING]]:")) {
                        const chunk = data.slice(13);
                        thinkingText += chunk.replaceAll("[VGT_NL]", "\n");
                        updateMessageWithThinking(msgIndex, thinkingText, fullResponseText, inferenceStartTime);
                    }
                    else if (data.startsWith("[[USAGE]]:")) {
                        const jsonStr = data.slice(10).trim();
                        try {
                            const usage = JSON.parse(jsonStr);
                            let maxContext = 131072;
                            if (state.currentModel && state.currentModel.toLowerCase().includes("deepseek")) {
                                maxContext = 1000000;
                            }
                            const pct = ((usage.total_tokens / maxContext) * 100).toFixed(2);
                            const elContext = document.getElementById("context-utilization");
                            if (elContext) {
                                const contextLabel = maxContext >= 1000000 ? "1M" : `${Math.round(maxContext / 1024)}k`;
                                let txt = `${usage.total_tokens.toLocaleString()} / ${contextLabel} (${pct}%)`;
                                if (usage.prompt_cache_hit_tokens > 0) {
                                    txt += ` [⚡ Cache Hit: ${usage.prompt_cache_hit_tokens.toLocaleString()}]`;
                                }
                                elContext.textContent = txt;
                            }
                        } catch (e) {
                            console.error("Usage parsing failed", e);
                        }
                    }
                    else if (data.startsWith("[[FILE_CHANGES]]:")) {
                        const jsonStr = data.slice(17).trim();
                        try {
                            const changes = JSON.parse(jsonStr);
                            if (!window.currentFileChanges) window.currentFileChanges = [];
                            changes.forEach(c => {
                                const existing = window.currentFileChanges.find(x => x.path === c.path);
                                if (existing) {
                                    existing.added += c.added;
                                    existing.removed += c.removed;
                                } else {
                                    window.currentFileChanges.push(c);
                                }
                            });
                        } catch (e) {
                            console.error("File changes parsing failed", e);
                        }
                    }
                    else if (data.startsWith("[SYSTEM ERROR]") || data.startsWith("[GROQ API ERROR")) {
                        fullResponseText += `\n${data}`;
                        updateMessageWithThinking(msgIndex, thinkingText, fullResponseText, inferenceStartTime);
                    }
                    else {
                        fullResponseText += data.replaceAll("[VGT_NL]", "\n");
                        updateMessageWithThinking(msgIndex, thinkingText, fullResponseText, inferenceStartTime);
                    }
                }
            }
        }

        updateMessageWithThinking(msgIndex, thinkingText, fullResponseText, inferenceStartTime, true);

        if (fullResponseText.trim() || thinkingText.trim()) {
            if (hasCommittedToolCall) {
                const lastMsg = state.messageHistory[state.messageHistory.length - 1];
                if (lastMsg && lastMsg.role === "assistant") {
                    lastMsg.content = fullResponseText || null;
                    lastMsg.reasoning_content = thinkingText || null;
                }
            } else {
                state.messageHistory.push({
                    role: "assistant",
                    content: fullResponseText || "",
                    reasoning_content: thinkingText || null
                });
            }
            saveChatHistory();
            speak(fullResponseText);
            
            if (window.currentFileChanges && window.currentFileChanges.length > 0) {
                appendFileChangesToMessage(msgIndex, window.currentFileChanges);
            }
            
            if (fullResponseText.includes("[[CONTINUE]]")) {
                setTimeout(() => {
                    triggerNextInference();
                }, 2000);
            }
        }
        
        if (pendingAutoExecute) {
            executeApprovedTool(pendingAutoExecute.msgIndex, pendingAutoExecute.name, pendingAutoExecute.args, true);
        }
        
    } catch (e) {
        console.error("Inference fetch error", e);
        updateMessageBody(msgIndex, `[CONNECTION TIMEOUT]: ${e.message}`);
    } finally {
        if (thinkingTimer) {
            clearInterval(thinkingTimer);
        }
        resetMicButton();
        updateMemoryCount();
    }
}

export async function loadSessionsList() {
    const container = document.getElementById("archive-grid");
    if (!container) return;
    try {
        const res = await fetch(`${state.API_BASE}/v1/chat/sessions`);
        const sessions = await res.json();
        
        if (!sessions || sessions.length === 0) {
            container.innerHTML = `
                <div style="grid-column: 1 / -1; color: var(--vgt-text-dim); font-size: 11px; text-align: center; padding: 40px; font-family: var(--font-mono);">
                    Keine archivierten Chat-Impulse vorhanden.
                </div>`;
            return;
        }
        
        container.innerHTML = sessions.map(s => {
            const dateStr = s.timestamp || "Unbekannt";
            const activeClass = (s.id === state.currentSessionId) ? "border-color: var(--vgt-cyan); background: rgba(0,240,255,0.05);" : "";
            
            return `
                <div class="glass-card" style="padding: 16px; display: flex; flex-direction: column; gap: 12px; font-family: var(--font-mono); ${activeClass}">
                    <div style="display: flex; justify-content: space-between; align-items: flex-start; gap: 10px;">
                        <span style="font-size: 11px; color: #fff; font-weight: bold; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 180px;" title="${s.title}">${s.title}</span>
                        <span class="session-delete-btn" style="cursor: pointer; font-size: 16px; color: var(--vgt-text-dim);" onclick="event.stopPropagation(); window.deleteSession('${s.id}')" title="Löschen">&times;</span>
                    </div>
                    <div style="font-size: 9px; color: var(--vgt-text-dim); line-height: 1.4;">
                        <div>DATUM: ${dateStr}</div>
                        <div>ID: ${s.id}</div>
                    </div>
                    <div style="display: flex; gap: 8px; margin-top: 5px;">
                        <button class="cyber-button" style="padding: 8px; font-size: 9px; flex: 1; height: auto;" onclick="window.loadSessionAndSwitch('${s.id}')">
                            LADEN
                        </button>
                    </div>
                </div>
            `;
        }).join("");
    } catch(e) {
        console.error("Failed to load sessions list", e);
    }
}

window.loadSessionAndSwitch = async function(id) {
    await loadSession(id);
    import('./ui.js').then(m => m.switchMode("chat"));
};

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
        updateChecklistUI([]);
        fetch(`${state.API_BASE}/v1/chat/checklist`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify([])
        }).catch(e => {});
        
        state.messageHistory.forEach((msg) => {
            if (msg.role === "user" || msg.role === "assistant") {
                addMessageToScreen(msg.role, msg.content, msg.reasoning_content);
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

    // Reset checklist UI and backend checklist
    updateChecklistUI([]);
    fetch(`${state.API_BASE}/v1/chat/checklist`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify([])
    }).catch(e => console.error("Failed to reset backend checklist", e));
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
    // Max 120 Nachrichten in der History (Context Window)
    if (state.messageHistory.length > 120) {
        state.messageHistory = state.messageHistory.slice(-120);
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

// Global helpers for checklist close and diff viewer rendering
let isChecklistWired = false;
function wireChecklistClose() {
    if (isChecklistWired) return;
    const btnCloseChecklist = document.getElementById("btn-close-checklist");
    if (btnCloseChecklist) {
        btnCloseChecklist.addEventListener("click", () => {
            const container = document.getElementById("current-task-checklist-items");
            if (container) {
                const isHidden = container.style.display === "none";
                container.style.display = isHidden ? "flex" : "none";
                btnCloseChecklist.textContent = isHidden ? "[ MINIMIEREN ]" : "[ MAXIMIEREN ]";
            }
        });
        isChecklistWired = true;
    }
}

export function updateChecklistUI(checklist) {
    wireChecklistClose();
    const panel = document.getElementById("current-task-checklist-panel");
    const container = document.getElementById("current-task-checklist-items");
    if (!panel || !container) return;

    if (!checklist || checklist.length === 0) {
        panel.classList.add("hidden");
        container.innerHTML = "";
        return;
    }

    panel.classList.remove("hidden");
    container.innerHTML = "";

    checklist.forEach((item, index) => {
        const div = document.createElement("div");
        div.style.display = "flex";
        div.style.alignItems = "center";
        div.style.gap = "8px";
        div.style.padding = "2px 0";

        let statusIcon = "⬜";
        let textStyle = "color: var(--vgt-text-dim);";
        
        if (item.status === "in_progress") {
            statusIcon = "⚡";
            textStyle = "color: var(--vgt-cyan); font-weight: bold;";
        } else if (item.status === "done") {
            statusIcon = "✅";
            textStyle = "color: var(--vgt-green); text-decoration: line-through; opacity: 0.7;";
        }

        div.innerHTML = `
            <span>${statusIcon}</span>
            <span style="${textStyle}">${item.text}</span>
        `;
        container.appendChild(div);
    });
}

export function appendFileChangesToMessage(msgIndex, changes) {
    const elChatOutput = document.getElementById("chat-output");
    if (!elChatOutput) return;

    const msgElement = elChatOutput.children[msgIndex];
    if (!msgElement) return;

    const body = msgElement.querySelector(".msg-body");
    if (!body) return;

    const oldBox = body.querySelector(".file-changes-box");
    if (oldBox) oldBox.remove();

    let totalAdded = 0;
    let totalRemoved = 0;
    changes.forEach(c => {
        totalAdded += c.added;
        totalRemoved += c.removed;
    });

    let html = `
        <div class="file-changes-box" style="margin-top: 16px; border: 1px solid rgba(0,240,255,0.25); background: rgba(0,240,255,0.03); border-radius: 6px; font-family: var(--font-mono); overflow: hidden;">
            <details style="outline: none;" open>
                <summary style="padding: 10px 14px; font-size: 10px; color: var(--vgt-cyan); cursor: pointer; user-select: none; display: flex; justify-content: space-between; align-items: center; outline: none;">
                    <span>📁 ${changes.length} Datei${changes.length > 1 ? 'en' : ''} geändert (+${totalAdded} -${totalRemoved} Zeilen)</span>
                    <span style="font-size: 8px; color: var(--vgt-text-dim);">[ DETAILS ]</span>
                </summary>
                <div style="padding: 10px 14px; border-top: 1px solid rgba(0,240,255,0.1); display: flex; flex-direction: column; gap: 8px; max-height: 180px; overflow-y: auto;">
    `;

    changes.forEach(c => {
        html += `
            <div style="display: flex; justify-content: space-between; align-items: center; font-size: 10px;">
                <span style="color: #fff;">📄 ${c.file}</span>
                <span style="color: var(--vgt-text-dim); font-size: 9px; margin-left: 10px; flex-grow: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 200px; text-align: left;">${c.path}</span>
                <span style="font-size: 9px; font-weight: bold; margin-left: 10px;">
                    <span style="color: var(--vgt-green);">+${c.added}</span> 
                    <span style="color: var(--vgt-red); margin-left: 4px;">-${c.removed}</span>
                </span>
            </div>
        `;
    });

    html += `
                </div>
            </details>
        </div>
    `;

    body.insertAdjacentHTML('beforeend', html);
    scrollToBottom();
}
