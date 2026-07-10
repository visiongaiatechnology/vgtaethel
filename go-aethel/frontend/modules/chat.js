import { state } from './state.js';
import { speak, resetMicButton, stopSpeaking } from './voice.js';
import { openPermissionGate, fetchKernelLogs, refreshSecurityHUD } from './security.js';
import { updateMemoryCount } from './memory.js';
import { requestApprovalForRun } from './run_approval_monitor.js';
import { runPersonalLearning } from './api.js';

export function scrollToBottom() {
    const el = document.getElementById("chat-output");
    if (el) el.scrollTop = el.scrollHeight;
}

function escapeHtml(value) {
    return String(value ?? "")
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}

function jsArg(value) {
    return escapeHtml(JSON.stringify(String(value ?? "")));
}

function sanitizeHtml(html) {
    const doc = new DOMParser().parseFromString(`<div>${html}</div>`, "text/html");
    const allowedTags = new Set(["a", "blockquote", "br", "code", "del", "div", "em", "h1", "h2", "h3", "h4", "h5", "h6", "hr", "li", "ol", "p", "pre", "span", "strong", "table", "tbody", "td", "th", "thead", "tr", "ul"]);
    const allowedAttributes = new Set(["class", "href", "title"]);
    doc.body.querySelectorAll("*").forEach((node) => {
        const tag = node.tagName.toLowerCase();
        if (!allowedTags.has(tag)) {
            node.replaceWith(doc.createTextNode(node.textContent || ""));
            return;
        }
        [...node.attributes].forEach((attr) => {
            const name = attr.name.toLowerCase();
            const value = attr.value.trim().toLowerCase();
            if (!allowedAttributes.has(name) || name.startsWith("on") || value.startsWith("javascript:") || value.startsWith("data:")) {
                node.removeAttribute(attr.name);
            }
        });
        if (tag === "a") {
            const href = node.getAttribute("href");
            if (href && !/^(https?:|mailto:)/i.test(href)) node.removeAttribute("href");
            if (node.hasAttribute("href")) {
                node.setAttribute("target", "_blank");
                node.setAttribute("rel", "noopener noreferrer");
            }
        }
    });
    return doc.body.firstElementChild ? doc.body.firstElementChild.innerHTML : "";
}

export function formatMarkdown(text) {
    if (!text) return "";
    if (typeof window.marked !== "undefined") {
        try {
            // GFM explizit einschalten für Tabellen-Unterstützung
            return sanitizeHtml(window.marked.parse(text, { gfm: true }));
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

function getFriendlyModelName(modelId) {
    if (!modelId) return "";
    const parts = modelId.split("/");
    let name = parts[parts.length - 1];
    name = name.replace("-instruct", "").replace("-chat", "");
    return name.toUpperCase();
}

export function addMessageToScreen(role, content, reasoning_content = null, model_id = null) {
    const elChatOutput = document.getElementById("chat-output");
    if (!elChatOutput) return "";

    // Intercept system/tool output and bundle it inside a collapsible activity log console
    if (role === "system") {
        let consoleEl = state.activeConsoleId ? document.getElementById(state.activeConsoleId) : null;
        if (!consoleEl) {
            const consoleId = `console-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
            state.activeConsoleId = consoleId;
            
            consoleEl = document.createElement("details");
            consoleEl.className = "working-console";
            consoleEl.id = consoleId;
            consoleEl.setAttribute("open", "");
            consoleEl.innerHTML = `
                <summary>🛠️ AETHEL ARBEITSPROZESS (AKTIVE AUSFÜHRUNG)</summary>
                <div class="console-log-body"></div>
            `;
            elChatOutput.appendChild(consoleEl);
        }
        
        const logBody = consoleEl.querySelector(".console-log-body");
        if (logBody && content) {
            const logLine = document.createElement("div");
            logLine.className = "console-log-line";
            
            // Highlight tool results vs general status warnings
            if (content.startsWith("[TOOL OUTPUT")) {
                logLine.style.color = "var(--vgt-green)";
            } else if (content.startsWith("[VGT SECURITY")) {
                logLine.style.color = "var(--vgt-orange)";
            } else if (content.startsWith("[SYSTEM ERROR") || content.startsWith("[GROQ API ERROR")) {
                logLine.style.color = "var(--vgt-red)";
            }
            
            logLine.textContent = content;
            logBody.appendChild(logLine);
            logBody.scrollTop = logBody.scrollHeight;
        }
        scrollToBottom();
        return state.activeConsoleId;
    }

    const messageDiv = document.createElement("div");
    messageDiv.className = `message ${role}`;

    const msgId = `msg-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
    messageDiv.id = msgId;

    let headerText = "SYSTEM // OVERWATCH";
    if (role === "user") headerText = "OPERATOR // TERMINAL";
    if (role === "assistant") {
        headerText = "AETHEL // CORTEX";
        if (model_id) {
            headerText += ` [${getFriendlyModelName(model_id)}]`;
        }
    }
    let bodyHtml = "";
    if (role === "assistant" && reasoning_content) {
        bodyHtml += `<details class="thinking-details" style="margin-bottom:12px;background:rgba(0,200,255,0.03);border:1px solid rgba(0,200,255,0.15);border-radius:4px;padding:8px;"><summary style="font-size:10px;color:#00c8ff;cursor:pointer;font-family:var(--font-mono);outline:none;user-select:none;">🧠 THOUGHT PROCESS</summary><div class="thinking-content" style="font-size:11px;color:rgba(255,255,255,0.6);font-family:var(--font-mono);margin-top:6px;white-space:pre-wrap;line-height:1.4;">${escapeHtml(reasoning_content)}</div></details>`;
    }
    const rendered = formatMarkdown(content);
    if (content && content.length > 600) {
        bodyHtml += `<div class="msg-collapsible collapsed"><div class="msg-content">${rendered}</div><div class="msg-fade-overlay"></div><button class="msg-toggle-btn" style="font-size:9px;color:var(--vgt-cyan);background:rgba(0,240,255,0.05);border:1px solid rgba(0,240,255,0.2);padding:4px 12px;border-radius:4px;cursor:pointer;margin-top:8px;font-family:var(--font-mono);">▶ ENTFALTEN (${content.length})</button></div>`;
    } else {
        bodyHtml += rendered;
    }
    messageDiv.innerHTML = `<div class="msg-header">${escapeHtml(headerText)}</div><div class="msg-body">${bodyHtml}</div>`;
    elChatOutput.appendChild(messageDiv);
    const toggle = messageDiv.querySelector('.msg-toggle-btn');
    if (toggle) {
        toggle.addEventListener('click', function() {
            const container = messageDiv.querySelector('.msg-collapsible');
            if (container) {
                const isCollapsed = container.classList.contains('collapsed');
                if (isCollapsed) {
                    container.classList.remove('collapsed');
                    this.textContent = '▼ EINGEKLAPPT';
                } else {
                    container.classList.add('collapsed');
                    this.textContent = `▶ ENTFALTEN (${content.length})`;
                }
            }
        });
    }
    while (elChatOutput.children.length > 10) {
        if (elChatOutput.firstChild) elChatOutput.removeChild(elChatOutput.firstChild);
    }
    scrollToBottom();
    return msgId;
}

export function updateMessageBody(msgId, content) {
    const msgElement = document.getElementById(msgId);
    if (msgElement) {
        const body = msgElement.querySelector(".msg-body");
        if (body) {
            body.innerHTML = formatMarkdown(content);
        }
    }
    scrollToBottom();
}

export function updateMessageWithThinking(msgId, thinking, content, startTime, isFinished = false) {
    const msgElement = document.getElementById(msgId);
    if (msgElement) {
        const body = msgElement.querySelector(".msg-body");
        if (body) {
            let html = "";
            if (!isFinished || thinking) {
                let elapsedText = "0.0s";
                if (startTime) {
                    elapsedText = ((Date.now() - startTime) / 1000).toFixed(1) + "s";
                }
                const openAttr = isFinished ? "" : "open";
                const loader = isFinished ? "" : `
                    <span class="thinking-spinner" style="display: inline-block; width: 8px; height: 8px; border: 1.5px solid rgba(0,240,255,0.2); border-top-color: var(--vgt-cyan); border-radius: 50%; animation: spin 1s linear infinite; margin-right: 6px; vertical-align: middle;"></span>
                    <style>@keyframes spin { to { transform: rotate(360deg); } }</style>
                `;
                const contentText = escapeHtml(thinking ? thinking : (isFinished ? "Analyse abgeschlossen." : "Kortex analysiert Aufgabe..."));
                html += `
                    <details class="thinking-details" ${openAttr} style="margin-bottom: 12px; background: rgba(0,200,255,0.02); border: 1px solid rgba(0,200,255,0.12); border-radius: 6px; padding: 10px; font-family: var(--font-mono);">
                        <summary style="font-size: 10px; color: var(--vgt-cyan); cursor: pointer; outline: none; user-select: none; display: flex; align-items: center; justify-content: space-between;">
                            <div style="display: flex; align-items: center;">
                                ${loader}
                                <span>🧠 CORTEX DENKPROZESS</span>
                            </div>
                            <span style="font-size: 9px; color: var(--vgt-text-dim);">${elapsedText}</span>
                        </summary>
                        <div class="thinking-content" style="font-size: 11px; color: rgba(255,255,255,0.55); margin-top: 8px; white-space: pre-wrap; line-height: 1.5; border-top: 1px solid rgba(0,240,255,0.05); padding-top: 6px;">${contentText}</div>
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

    if (isFinished && state.activeConsoleId) {
        const consoleEl = document.getElementById(state.activeConsoleId);
        if (consoleEl) {
            consoleEl.removeAttribute("open"); // Collapse intermediate activity window
        }
        state.activeConsoleId = null; // Clear out active log target reference
    }

    scrollToBottom();
}

export function showToolRequest(msgId, toolName, args) {
    const msgElement = document.getElementById(msgId);
    if (!msgElement) return;

    document.getElementById(`tool-box-${msgId}`)?.remove();

    const box = document.createElement("div");
    box.className = "tool-request-box";
    box.id = `tool-box-${msgId}`;

    const title = document.createElement("h5");
    title.textContent = `AUTORISIERUNG ERFORDERLICH: [${toolName}]`;
    const pre = document.createElement("pre");
    pre.className = "tool-args";
    pre.textContent = JSON.stringify(args, null, 2);
    const actions = document.createElement("div");
    actions.className = "tool-actions";
    const approve = document.createElement("button");
    approve.className = "btn-approve";
    approve.textContent = "AUSFÜHREN";
    approve.addEventListener("click", () => window.approveTool(msgId, toolName, args, true));
    const reject = document.createElement("button");
    reject.className = "btn-reject";
    reject.textContent = "BLOCKIEREN";
    reject.addEventListener("click", () => rejectTool(msgId));
    actions.replaceChildren(approve, reject);
    box.replaceChildren(title, pre, actions);

    msgElement.appendChild(box);
    scrollToBottom();
}

function appendToolResult(call, content) {
    state.messageHistory.push({
        role: "tool",
        tool_call_id: call.id,
        name: call.name,
        content
    });
}

function finishQueuedTool(msgId, content, cancelRemaining = false) {
    const active = state.pendingToolQueue.shift();
    if (!active) return;
    appendToolResult(active, content);

    if (cancelRemaining) {
        for (const skipped of state.pendingToolQueue) {
            appendToolResult(skipped, "VGT SECURITY INTERVENTION: Nicht ausgeführt, weil ein vorheriger Tool-Aufruf abgelehnt oder blockiert wurde.");
        }
        state.pendingToolQueue = [];
    }

    state.pendingToolRequest = null;
    state.pendingToolCallId = "";
    state.pendingToolCallName = "";
    saveChatHistory();

    if (state.pendingToolQueue.length > 0 && !cancelRemaining) {
        presentNextQueuedTool(msgId);
        return;
    }
    triggerNextInference();
}

function presentNextQueuedTool(msgId) {
    const next = state.pendingToolQueue[0];
    if (!next || state.agentPaused) return;

    state.pendingToolCallId = next.id;
    state.pendingToolCallName = next.name;
    showToolRequest(msgId, next.name, next.args);

    const safeAutoTool = next.name === "nexus_save" || next.name === "nexus_recall";
    if (state.isFullAutonomy || safeAutoTool) {
        setTimeout(() => executeApprovedTool(msgId, next.name, next.args), 0);
        return;
    }

    state.pendingToolRequest = { msgId, name: next.name, args: next.args };
    const label = next.name === "sys_exec_cmd" ? "Systembefehl" : next.name === "web_browser" ? "Webbrowser" : next.name;
    speak(`Freigabe für ${label} erforderlich. Bitte freigeben oder ablehnen sagen.`);
}

export function rejectTool(msgId) {
    const box = document.getElementById(`tool-box-${msgId}`);
    if (box) {
        box.innerHTML = `<span class="tool-status-badge status-rejected">SECURITY BLOCK: Ausführung vom Operator blockiert.</span>`;
    }
    
    addAgentLog(`Befehlsausführung vom Operator abgelehnt: ${state.pendingToolCallName || "tool"}`, "warning");

    finishQueuedTool(msgId, "VGT SECURITY INTERVENTION: Operator hat die Ausführung blockiert.", true);
    fetchKernelLogs();
}
window.rejectTool = rejectTool;

export async function executeApprovedTool(msgId, name, args, approvalToken = "") {
    if (state.agentPaused) {
        const pausedBox = document.getElementById(`tool-box-${msgId}`);
        if (pausedBox) pausedBox.textContent = "AUSFÜHRUNG PAUSIERT – Operator muss fortsetzen.";
        return;
    }
    const box = document.getElementById(`tool-box-${msgId}`);
    if (box) {
        box.innerHTML = `<div class="font-mono text-xs animate-pulse text-vgt-cyan">Sende Befehl an Kernel...</div>`;
    }

    if (name === "web_browser") {
        document.getElementById("browser-tab-title").textContent = "VERBINDE...";
        document.getElementById("browser-url-input").value = args.url || (args.search_query ? `Suche: ${args.search_query}` : "Lade...");
        document.getElementById("browser-placeholder").classList.remove("hidden");
        document.getElementById("browser-screenshot").classList.add("hidden");
    }

    addAgentLog(`Sende Befehl an Kernel: [${name}]`, "tool");

    try {
        const res = await fetch(`${state.API_BASE}/v1/tools/execute`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                name: name,
                args: args,
                approval_token: approvalToken || undefined
            })
        });
        
        const data = await res.json();
        fetchKernelLogs();
        refreshSecurityHUD();

        if (data.status === "success") {
            addAgentLog(`Befehl [${name}] erfolgreich beendet.`, "success");
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
        } else if (data.status !== "security_intervention" && data.status !== "security_blocked") {
            addAgentLog(`Befehl [${name}] fehlgeschlagen: ${data.error || "Unbekannter Fehler"}`, "error");
        }

        if (data.status === "security_intervention") {
			state.pendingToolRequest = {
				msgIndex: msgId,
				name,
				args,
				riskLevel: data.risk_level,
				approvalToken: data.approval_token || ""
			};
            addAgentLog(`Sicherheitsprüfung erfordert Operator-Zustimmung für: [${name}]`, "warning");
            if (box) {
                box.innerHTML = `<span class="tool-status-badge status-rejected" style="color: var(--vgt-orange)">SECURITY GATE: Bestätigung ausstehend...</span>`;
            }
            openPermissionGate(name, data.capability, data.risk_level, data.risk_score, data.threats, args, msgId, data.approval_token || "");
            return;
        }

        if (data.status === "security_blocked") {
            addAgentLog(`Befehl permanent blockiert (VGT FIREWALL): [${name}]`, "error");
            if (box) {
                box.innerHTML = `<span class="tool-status-badge status-rejected">SECURITY BLOCK: Aktion permanent blockiert (FORBIDDEN).</span>`;
            }
            speak("Systembefehl wurde permanent blockiert.");

            const blockedMsg = `[VGT SECURITY BLOCK]: ${data.message || "Aktion verboten."}`;
            addMessageToScreen("system", blockedMsg);

            finishQueuedTool(msgId, "VGT SECURITY INTERVENTION: Aktion permanent blockiert (FORBIDDEN).", true);
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
        
        finishQueuedTool(msgId, resultText);

    } catch (e) {
        console.error("Tool exec error", e);
        if (box) {
            box.innerHTML = `<span class="tool-status-badge status-failed">CONNECTION FAILED: ${escapeHtml(e.message)}</span>`;
        }
        
        if (name === "web_browser") {
            document.getElementById("browser-tab-title").textContent = "Browser Fehler";
            document.getElementById("browser-url-input").value = "about:error";
        }
        finishQueuedTool(msgId, `ERROR: Tool execution request failed: ${e.message}`);
        fetchKernelLogs();
    }
}
window.approveTool = function(msgId, name, args, approvalToken = "") {
    executeApprovedTool(msgId, name, args, approvalToken);
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
    if (state.agentPaused) return;
    if (state.agenticTurnCount >= state.maxAgenticTurns) {
        state.agenticTurnCount = 0;
        state.agenticStuckCount = 0;
        console.warn("[Agentic] Turn limit reached; stopping autonomous chain");
        return;
    }
    state.agenticTurnCount++;
    sendMessage(true);
}

const waitForRunPoll = (milliseconds) => new Promise(resolve => setTimeout(resolve, milliseconds));

async function handlePersistentRunApproval(run) {
    await requestApprovalForRun(run);
}

async function sendMessageViaPersistentRun() {
    const input = document.getElementById('user-input');
    const prompt = input?.value.trim() || '';
    if (!prompt) return;

    if (state.activeRunId) {
        await fetch(`${state.API_BASE}/v1/runs/${encodeURIComponent(state.activeRunId)}/cancel`, { method: 'POST' }).catch(() => {});
    }
    stopSpeaking();
    if (input) input.value = '';
    addMessageToScreen('user', prompt);
    state.messageHistory.push({ role: 'user', content: prompt });
    await saveChatHistory();

    const msgId = addMessageToScreen('assistant', '', null, state.currentModel);
    state.currentAssistantMsgIndex = msgId;
    const startedAt = Date.now();
    updateMessageWithThinking(msgId, 'Persistenter Agent Run wird initialisiert…', '', startedAt);
    const uiModule = await import('./ui.js');
    const systemPrompt = uiModule.getCombinedSystemPrompt();
    const liveOperator = !!(document.getElementById('view-control') && !document.getElementById('view-control').classList.contains('hidden'));
    const profile = liveOperator ? 'browser_operator' : 'developer';

    const createRes = await fetch(`${state.API_BASE}/v1/chat/runs`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            objective: prompt,
            profile_id: profile,
            model_id: state.currentModel,
            system_prompt: systemPrompt,
            messages: state.messageHistory,
            cost_budget_usd: 2,
            max_agent_turns: state.maxAgenticTurns,
            live_operator_active: liveOperator
        })
    });
    if (!createRes.ok) throw new Error(await createRes.text());
    let run = await createRes.json();
    state.activeRunId = run.id;

    for (let poll = 0; poll < 1200; poll++) {
        const response = await fetch(`${state.API_BASE}/v1/runs/${encodeURIComponent(run.id)}`);
        if (!response.ok) throw new Error(await response.text());
        run = await response.json();
        const lastTrace = Array.isArray(run.trace) && run.trace.length ? run.trace[run.trace.length - 1] : null;
        const progress = `${(run.steps || []).filter(step => step.status === 'verified').length}/${(run.steps || []).length}`;
        const thinking = `${String(run.status || '').toUpperCase()} · Schritt ${progress}${lastTrace ? `\n${lastTrace.detail}` : ''}`;
        updateMessageWithThinking(msgId, thinking, run.final_report || '', startedAt, false);

        if (run.status === 'waiting_approval') {
            await handlePersistentRunApproval(run);
        } else if (run.status === 'completed') {
            const report = String(run.final_report || 'Run abgeschlossen.').trim();
            updateMessageWithThinking(msgId, '', report, startedAt, true);
            state.messageHistory.push({ role: 'assistant', content: report, model_id: run.model_id || state.currentModel, run_id: run.id });
            await saveChatHistory();
            speak(report);
            state.activeRunId = null;
            resetMicButton();
            return;
        } else if (run.status === 'failed' || run.status === 'cancelled') {
            const errorText = run.failure_reason || `Run ${run.status}.`;
            updateMessageWithThinking(msgId, '', `[AGENT RUN ${String(run.status).toUpperCase()}]: ${errorText}`, startedAt, true);
            state.activeRunId = null;
            resetMicButton();
            return;
        } else if (run.status === 'paused') {
            updateMessageWithThinking(msgId, '', `[AGENT RUN PAUSED]: Kostenlimit oder Operator-Pause erreicht. Fortsetzung im Run Center möglich.`, startedAt, true);
            state.activeRunId = null;
            resetMicButton();
            return;
        }
        await waitForRunPoll(750);
    }
    throw new Error('Agent Run polling deadline reached. Der Run bleibt im Run Center erhalten.');
}

export async function sendMessage(isContinuation = false) {
	if (!isContinuation) {
		try {
			await sendMessageViaPersistentRun();
		} catch (error) {
			console.error('Persistent agent run failed', error);
			addMessageToScreen('system', `[AGENT RUN ERROR]: ${error.message}`);
			state.activeRunId = null;
			resetMicButton();
		}
		return;
	}
	// Legacy continuation calls are intentionally ignored. The Go runner owns
	// all subsequent model/tool turns and persists them independently of the UI.
	return;

    const input = document.getElementById("user-input");
    if (!input) return;

    if (!isContinuation) {
        state.agentPaused = false;
        if (state.pendingToolQueue.length > 0) {
            for (const queuedCall of state.pendingToolQueue) {
                appendToolResult(queuedCall, "VGT OPERATOR INTERRUPTION: Nicht ausgeführt, weil der Operator eine neue Anfrage gestartet hat.");
            }
            state.pendingToolQueue = [];
            state.pendingToolRequest = null;
            state.pendingToolCallId = "";
            state.pendingToolCallName = "";
            saveChatHistory();
        }
        window.currentFileChanges = [];
        state.activeConsoleId = null;
        state.agenticTurnCount = 0;
        state.agenticStuckCount = 0;
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

    const msgId = addMessageToScreen("assistant", "", null, state.currentModel);
    state.currentAssistantMsgIndex = msgId;
    let fullResponseText = "";
    let thinkingText = "";
    const toolBuffers = new Map();
    let hasCommittedToolCall = false;
    const inferenceStartTime = Date.now();
    let thinkingTimer = null;
    let inferenceController = null;

    // Track whether the PREVIOUS run finished with a tool response pending.
    // This allows us to detect when the model outputs reasoning text after
    // a tool result but then stops without requesting another tool.
    const hadToolResultContext = !!isContinuation &&
        state.messageHistory.length > 0 &&
        state.messageHistory[state.messageHistory.length - 1]?.role === "tool";

    // Start periodic timer to update elapsed thinking time and keep UI alive/responsive
    thinkingTimer = setInterval(() => {
        updateMessageWithThinking(msgId, thinkingText, fullResponseText, inferenceStartTime);
    }, 100);

    try {
        // Build message list for this inference call
        // When continuing after a tool result, inject an agentic push message
        // to force the model to continue its plan without asking for user input.
        let messagesForInference = [...state.messageHistory];
        let injectedContinuationMsg = false;
        if (isContinuation) {
            // Check if the last message in history is a tool result
            const lastMsg = messagesForInference[messagesForInference.length - 1];
            if (lastMsg && lastMsg.role === "tool") {
                // The protocol is sent as a user turn so every provider adapter
                // preserves the original system instruction unchanged.
                messagesForInference = [
                    ...messagesForInference,
                    {
                        role: "user",
                        content: "[AGENT CONTROL]: Tool-Ergebnis empfangen. Prüfe das Ergebnis, führe nur den nächsten notwendigen Schritt aus oder liefere eine Abschlussantwort. Wiederhole keine fehlgeschlagene Aktion blind."
                    }
                ];
                injectedContinuationMsg = true;
            }
        }

        const uiModule = await import('./ui.js');
        const systemPromptText = uiModule.getCombinedSystemPrompt();

        state.activeInferenceController?.abort();
        inferenceController = new AbortController();
        state.activeInferenceController = inferenceController;
        const res = await fetch(`${state.API_BASE}/v1/chat`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            signal: inferenceController.signal,
            body: JSON.stringify({
                model_id: state.currentModel,
                messages: messagesForInference,
                temperature: 0.15,
                use_tools: true,
                system_prompt: systemPromptText,
                live_operator_active: !!(document.getElementById("view-control") && !document.getElementById("view-control").classList.contains("hidden"))
            })
        });


        if (!res.body) {
            updateMessageBody(msgId, "[CORE ERROR]: Kein Stream empfangen.");
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
                            for (const call of Array.isArray(delta) ? delta : []) {
                                const key = String(call.index ?? call.id ?? toolBuffers.size);
                                const buffer = toolBuffers.get(key) || { index: Number(call.index ?? 0), id: "", name: "", args: "" };
                                if (call.id) buffer.id = call.id;
                                if (call.function?.name) buffer.name = call.function.name;
                                if (call.function?.arguments) buffer.args += call.function.arguments;
                                toolBuffers.set(key, buffer);
                            }
                        } catch (e) {
                            console.error("Tool Delta parsing failed", e);
                        }
                    }
                    else if (data.trim() === "[[TOOL_COMMIT]]") {
                        const calls = [...toolBuffers.values()]
                            .sort((a, b) => a.index - b.index)
                            .filter(call => call.id && call.name && /^[a-z][a-z0-9_]{1,63}$/.test(call.name));
                        if (calls.length === 0 || calls.length !== toolBuffers.size || calls.some(call => {
                            try { JSON.parse(call.args || "{}"); return false; } catch (_) { return true; }
                        })) {
                            fullResponseText += "\n[SYSTEM ERROR]: Ungültiger Tool-Aufruf vom Modell verworfen.";
                            updateMessageWithThinking(msgId, thinkingText, fullResponseText, inferenceStartTime);
                            toolBuffers.clear();
                            continue;
                        }

                        state.messageHistory.push({
                            role: "assistant",
                            content: fullResponseText || null,
                            tool_calls: calls.map(call => ({
                                id: call.id,
                                type: "function",
                                function: { name: call.name, arguments: call.args }
                            })),
                            model_id: state.currentModel
                        });
                        saveChatHistory();
                        hasCommittedToolCall = true;
                        state.pendingToolQueue = calls.map(call => ({ ...call, args: JSON.parse(call.args || "{}") }));
                        toolBuffers.clear();
                        presentNextQueuedTool(msgId);
                    }
                    else if (data.startsWith("[[THINKING]]:")) {
                        const chunk = data.slice(13);
                        thinkingText += chunk.replaceAll("[VGT_NL]", "\n");
                        updateMessageWithThinking(msgId, thinkingText, fullResponseText, inferenceStartTime);
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
                    else if (data.startsWith("[SYSTEM ERROR]") || /^\[[A-Z]+ API ERROR \d+\]/.test(data)) {
                        fullResponseText += `\n${data}`;
                        updateMessageWithThinking(msgId, thinkingText, fullResponseText, inferenceStartTime);
                    }
                    else {
                        fullResponseText += data.replaceAll("[VGT_NL]", "\n");
                        updateMessageWithThinking(msgId, thinkingText, fullResponseText, inferenceStartTime);
                    }
                }
            }
        }

        // Handle empty API responses to prevent UI hangs
        if (!fullResponseText.trim() && !thinkingText.trim()) {
            if (hadToolResultContext) {
                state.agenticStuckCount++;
                if (state.agenticStuckCount <= 2) {
                    console.warn(`[Agentic] Empty response recovery #${state.agenticStuckCount}`);
                    const el = document.getElementById(msgId);
                    if (el) el.remove();
                    setTimeout(() => triggerNextInference(), 1500);
                } else {
                    state.agenticStuckCount = 0;
                    updateMessageBody(msgId, "[CORE ERROR]: Der Agent hat nach dem Tool-Ergebnis wiederholt leer geantwortet. Die Sequenz wurde sicher beendet.");
                }
                return;
            }
            updateMessageBody(msgId, "[CORE ERROR]: Der Provider hat eine leere Antwort geliefert. Bitte Modell oder Verbindung prüfen.");
        }

        const displayText = fullResponseText.trim();
        updateMessageWithThinking(msgId, thinkingText, displayText, inferenceStartTime, true);

        if (fullResponseText.trim() || thinkingText.trim()) {
            if (hasCommittedToolCall) {
                const lastMsg = state.messageHistory[state.messageHistory.length - 1];
                if (lastMsg && lastMsg.role === "assistant") {
                    lastMsg.content = fullResponseText || null;
                    lastMsg.reasoning_content = thinkingText || null;
                    lastMsg.model_id = state.currentModel;
                }
            } else {
                state.messageHistory.push({
                    role: "assistant",
                    content: fullResponseText || "",
                    reasoning_content: thinkingText || null,
                    model_id: state.currentModel
                });
            }
            saveChatHistory();

            const visibleText = fullResponseText.trim();
            speak(visibleText);

            if (window.currentFileChanges && window.currentFileChanges.length > 0) {
                appendFileChangesToMessage(msgId, window.currentFileChanges);
            }
            
            // Legacy continuation loop is unreachable; the persistent Go runner owns the chain.
            const isAgenticRun = hasCommittedToolCall || hadToolResultContext;

            if (false && isAgenticRun) {
                setTimeout(() => {
                    state.agenticStuckCount = 0;
                    triggerNextInference();
                }, 2000);
            } else if (hadToolResultContext && !hasCommittedToolCall && !fullResponseText.trim()) {
                // The model received a tool result, output reasoning text,
                // but made no new tool call or text reply.
                // This is the "reasoning-then-stop" hang. Auto-recover.
                if (!state.agenticStuckCount) state.agenticStuckCount = 0;
                state.agenticStuckCount++;

                if (state.agenticStuckCount <= 4) {
                    console.warn(`[Agentic] Stuck recovery #${state.agenticStuckCount} — re-pushing continuation after reasoning block`);
                    setTimeout(() => triggerNextInference(), 1800);
                } else {
                    state.agenticStuckCount = 0;
                    console.warn("[Agentic] Max stuck recoveries reached — stopping chain");
                }
            } else {
                // Normal text reply or finished agentic run — reset stuck counter
                state.agenticStuckCount = 0;
                if (!hasCommittedToolCall) state.agenticTurnCount = 0;
            }
        }
        
    } catch (e) {
        if (e?.name === "AbortError") {
            updateMessageBody(msgId, "[OPERATOR]: Aktuelle Inferenz wurde abgebrochen.");
            return;
        }
        console.error("Inference fetch error", e);
        updateMessageBody(msgId, `[CONNECTION TIMEOUT]: ${e.message}`);
    } finally {
        if (thinkingTimer) {
            clearInterval(thinkingTimer);
        }
        if (state.activeInferenceController === inferenceController) state.activeInferenceController = null;
        resetMicButton();
        updateMemoryCount();
        import('./ui.js').then(m => m.refreshAPICosts());
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
            const safeTitle = escapeHtml(s.title || "");
            const safeID = escapeHtml(s.id || "");
            const safeDate = escapeHtml(dateStr);
            const idArg = jsArg(s.id || "");
            
            return `
                <div class="glass-card" style="padding: 16px; display: flex; flex-direction: column; gap: 12px; font-family: var(--font-mono); ${activeClass}">
                    <div style="display: flex; justify-content: space-between; align-items: flex-start; gap: 10px;">
                        <span style="font-size: 11px; color: #fff; font-weight: bold; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 180px;" title="${safeTitle}">${safeTitle}</span>
                        <span class="session-delete-btn" style="cursor: pointer; font-size: 16px; color: var(--vgt-text-dim);" onclick="event.stopPropagation(); window.deleteSession(${idArg})" title="Löschen">&times;</span>
                    </div>
                    <div style="font-size: 9px; color: var(--vgt-text-dim); line-height: 1.4;">
                        <div>DATUM: ${safeDate}</div>
                        <div>ID: ${safeID}</div>
                    </div>
                    <div style="display: flex; gap: 8px; margin-top: 5px;">
                        <button class="cyber-button" style="padding: 8px; font-size: 9px; flex: 1; height: auto;" onclick="window.loadSessionAndSwitch(${idArg})">
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
    
    // Restore saved model for this session if stored in localStorage
    const savedModel = localStorage.getItem('model_' + id);
    if (savedModel) {
        state.currentModel = savedModel;
        const elDropdown = document.getElementById("model-dropdown");
        if (elDropdown) {
            elDropdown.value = savedModel;
            // Trigger UI updates for live operator panel based on restored model
            elDropdown.dispatchEvent(new Event('change'));
        }
    }
    
    document.querySelectorAll(".session-item").forEach(el => el.classList.remove("active"));
    const elChatOutput = document.getElementById("chat-output");
    
    try {
        const res = await fetch(`${state.API_BASE}/v1/chat/sessions/load?id=${encodeURIComponent(id)}`);
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
                addMessageToScreen(msg.role, msg.content, msg.reasoning_content, msg.model_id);
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
    
    if (state.currentModel) {
        localStorage.setItem('model_' + state.currentSessionId, state.currentModel);
    }
    
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
        await fetch(`${state.API_BASE}/v1/chat/sessions/delete?id=${encodeURIComponent(id)}`, {
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

function triggerPersonalLearning(assistantText) {
    const recentUser = [...state.messageHistory].reverse().find(msg => msg.role === "user");
    const userText = recentUser?.content || "";
    const text = `${userText}\n${assistantText || ""}`.trim();
    if (!text) return;
    runPersonalLearning(text).catch(err => console.debug("Personal learning skipped", err));
}

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

export function addAgentLog(text, type = "info") {
    const container = document.getElementById("agent-logs-container");
    if (!container) return;
    
    if (container.innerHTML.includes("[STANDBY]")) {
        container.innerHTML = "";
    }
    
    const timeStr = new Date().toLocaleTimeString();
    const logDiv = document.createElement("div");
    logDiv.style.borderBottom = "1px solid rgba(255,255,255,0.02)";
    logDiv.style.padding = "4px 0";
    
    let prefix = "[INFO]";
    let colorStyle = "color: var(--vgt-text-dim);";
    if (type === "success") {
        prefix = "[OK]";
        colorStyle = "color: var(--vgt-green);";
    } else if (type === "error") {
        prefix = "[FAIL]";
        colorStyle = "color: var(--vgt-red); font-weight: bold;";
    } else if (type === "warning") {
        prefix = "[WARN]";
        colorStyle = "color: var(--vgt-orange);";
    } else if (type === "tool") {
        prefix = "[TOOL]";
        colorStyle = "color: var(--vgt-cyan); font-weight: bold;";
    }
    
    const timeEl = document.createElement("span");
    timeEl.style.cssText = "color: var(--vgt-text-dim); font-size: 8px;";
    timeEl.textContent = `[${timeStr}] `;
    const textEl = document.createElement("span");
    textEl.style.cssText = colorStyle;
    textEl.textContent = `${prefix} ${text}`;
    logDiv.replaceChildren(timeEl, textEl);
    container.appendChild(logDiv);
    container.scrollTop = container.scrollHeight;
}

export function updateChecklistUI(checklist) {
    wireChecklistClose();
    const panel = document.getElementById("current-task-checklist-panel");
    const container = document.getElementById("current-task-checklist-items");

    // Agent Tracker components
    const agentStepsContainer = document.getElementById("agent-steps-container");
    const agentCurrentFocus = document.getElementById("agent-current-focus");
    const agentStatusLabel = document.getElementById("agent-status-label");

    if (!panel || !container) return;

    if (!checklist || checklist.length === 0) {
        panel.classList.add("hidden");
        container.innerHTML = "";

        if (agentCurrentFocus) agentCurrentFocus.textContent = "Kein aktiver Agenten-Run gestartet.";
        if (agentStatusLabel) {
            agentStatusLabel.textContent = "Bereit für Aufgaben";
            agentStatusLabel.style.color = "var(--vgt-green)";
        }
        if (agentStepsContainer) {
            agentStepsContainer.innerHTML = `<div style="color: var(--vgt-text-dim); text-align: center; margin-top: 40px;">Warte auf einen Agent-Run, um den Plan anzuzeigen...</div>`;
        }
        return;
    }

    panel.classList.remove("hidden");
    container.innerHTML = "";
    if (agentStepsContainer) agentStepsContainer.innerHTML = "";

    let currentFocusText = "";
    let hasActiveStep = false;

    checklist.forEach((item, index) => {
        // Chat list render
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
            if (!currentFocusText) {
                currentFocusText = item.text;
                hasActiveStep = true;
            }
        } else if (item.status === "done") {
            statusIcon = "✅";
            textStyle = "color: var(--vgt-green); text-decoration: line-through; opacity: 0.7;";
        }

        const iconSpan = document.createElement("span");
        iconSpan.textContent = statusIcon;
        const textSpan = document.createElement("span");
        textSpan.style.cssText = textStyle;
        textSpan.textContent = item.text || "";
        div.replaceChildren(iconSpan, textSpan);
        container.appendChild(div);

        // Agent Tracker render
        if (agentStepsContainer) {
            const stepDiv = document.createElement("div");
            stepDiv.style.border = item.status === "in_progress" ? "1px solid var(--vgt-cyan)" : "1px solid rgba(255,255,255,0.03)";
            stepDiv.style.background = item.status === "in_progress" ? "rgba(0, 240, 255, 0.04)" : "rgba(0, 0, 0, 0.2)";
            stepDiv.style.padding = "10px 14px";
            stepDiv.style.borderRadius = "6px";
            stepDiv.style.display = "flex";
            stepDiv.style.alignItems = "center";
            stepDiv.style.gap = "12px";
            stepDiv.style.fontSize = "11px";

            let stepIcon = `<span style="color: var(--vgt-text-dim);">[ ]</span>`;
            if (item.status === "in_progress") {
                stepIcon = `<span class="animate-pulse" style="color: var(--vgt-cyan);">⚡</span>`;
            } else if (item.status === "done") {
                stepIcon = `<span style="color: var(--vgt-green);">✅</span>`;
            }

            const stepIconWrap = document.createElement("div");
            stepIconWrap.innerHTML = sanitizeHtml(stepIcon);
            const stepText = document.createElement("div");
            stepText.style.cssText = `flex-grow: 1; ${textStyle}`;
            stepText.textContent = item.text || "";
            stepDiv.replaceChildren(stepIconWrap, stepText);
            agentStepsContainer.appendChild(stepDiv);
        }
    });

    if (agentCurrentFocus) {
        if (hasActiveStep) {
            agentCurrentFocus.textContent = `Schritt in Arbeit: "${currentFocusText}"`;
        } else {
            const firstPending = checklist.find(x => x.status === "todo");
            if (firstPending) {
                agentCurrentFocus.textContent = `Nächster Schritt vorbereitet: "${firstPending.text}"`;
            } else {
                agentCurrentFocus.textContent = "Alle geplanten Schritte erfolgreich abgearbeitet.";
            }
        }
    }

    if (agentStatusLabel) {
        const isDone = checklist.every(x => x.status === "done");
        if (isDone) {
            agentStatusLabel.textContent = "PLAN ERFOLGREICH BEENDET";
            agentStatusLabel.style.color = "var(--vgt-green)";
        } else {
            agentStatusLabel.textContent = "AKTIVER AGENTEN-LAUF";
            agentStatusLabel.style.color = "var(--vgt-cyan)";
        }
    }
}


export function appendFileChangesToMessage(msgId, changes) {
    if (!changes || changes.length === 0) return;

    const msgElement = document.getElementById(msgId);
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
        const safeFile = escapeHtml(c.file || "");
        const safePath = escapeHtml(c.path || "");
        const added = Number.isFinite(Number(c.added)) ? Number(c.added) : 0;
        const removed = Number.isFinite(Number(c.removed)) ? Number(c.removed) : 0;
        html += `
            <div style="display: flex; justify-content: space-between; align-items: center; font-size: 10px;">
                <span style="color: #fff;">📄 ${safeFile}</span>
                <span style="color: var(--vgt-text-dim); font-size: 9px; margin-left: 10px; flex-grow: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 200px; text-align: left;">${safePath}</span>
                <span style="font-size: 9px; font-weight: bold; margin-left: 10px;">
                    <span style="color: var(--vgt-green);">+${added}</span> 
                    <span style="color: var(--vgt-red); margin-left: 4px;">-${removed}</span>
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
