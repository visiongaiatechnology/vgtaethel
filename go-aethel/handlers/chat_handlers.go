package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go-aethel/agent"
	"go-aethel/personal"
	"go-aethel/provider"
	"go-aethel/skills"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type ChatRequest struct {
	ModelID             string            `json:"model_id"`
	Messages            []json.RawMessage `json:"messages"`
	Temperature         float64           `json:"temperature"`
	UseTools            bool              `json:"use_tools"`
	SystemPrompt        string            `json:"system_prompt"`
	ReasoningEffort     string            `json:"reasoning_effort,omitempty"`
	ReasoningVisibility string            `json:"reasoning_visibility,omitempty"`
	LiveOperatorActive  bool              `json:"live_operator_active"`
	SphereActive        bool              `json:"sphere_active"`
	ToolAllowlist       *[]string         `json:"tool_allowlist,omitempty"`
}

func applyReasoningOptions(payload map[string]interface{}, modelID string, request ChatRequest) {
	effort := strings.ToLower(strings.TrimSpace(request.ReasoningEffort))
	visibility := strings.ToLower(strings.TrimSpace(request.ReasoningVisibility))
	isGPTOSS := strings.Contains(modelID, "gpt-oss")
	isQwen3 := strings.Contains(modelID, "qwen3")
	isOpenAI56 := strings.Contains(modelID, "gpt-5.6")
	isGemini3 := strings.Contains(modelID, "gemini-3")

	if isOpenAI56 {
		if effort != "none" && effort != "low" && effort != "medium" && effort != "high" && effort != "xhigh" && effort != "max" {
			effort = "medium"
		}
		payload["reasoning_effort"] = effort
		return
	}

	if isGemini3 {
		if effort != "minimal" && effort != "low" && effort != "medium" && effort != "high" {
			effort = "medium"
		}
		payload["reasoning_effort"] = effort
		return
	}

	if isGPTOSS {
		if visibility == "hidden" {
			payload["reasoning_format"] = "hidden"
		} else if request.UseTools {
			payload["reasoning_format"] = "parsed"
		} else {
			payload["include_reasoning"] = true
		}
		if effort != "low" && effort != "medium" && effort != "high" {
			if request.UseTools {
				effort = "medium"
			} else {
				effort = "high"
			}
		}
		payload["reasoning_effort"] = effort
		return
	}

	if isQwen3 {
		if visibility == "hidden" {
			payload["reasoning_format"] = "hidden"
		} else {
			payload["reasoning_format"] = "parsed"
		}
		if strings.Contains(modelID, "qwen3.6") {
			if effort != "none" && effort != "default" {
				effort = "default"
			}
			payload["reasoning_effort"] = effort
		}
	}
}

func applyClaudeReasoningOptions(payload map[string]interface{}, modelID string, request ChatRequest) bool {
	id := strings.ToLower(modelID)
	effort := strings.ToLower(strings.TrimSpace(request.ReasoningEffort))
	allowed := func(options ...string) bool {
		for _, option := range options {
			if effort == option {
				return true
			}
		}
		return false
	}
	switch {
	case strings.Contains(id, "claude-fable-5"):
		if !allowed("low", "medium", "high", "xhigh", "max") {
			effort = "high"
		}
	case strings.Contains(id, "claude-opus-4-8"):
		if !allowed("low", "medium", "high", "xhigh", "max") {
			effort = "high"
		}
		payload["thinking"] = map[string]interface{}{"type": "adaptive"}
	case strings.Contains(id, "claude-sonnet-4-6"):
		if !allowed("low", "medium", "high", "max") {
			effort = "medium"
		}
		payload["thinking"] = map[string]interface{}{"type": "adaptive"}
	default:
		return false
	}
	payload["output_config"] = map[string]interface{}{"effort": effort}
	switch effort {
	case "xhigh", "max":
		payload["max_tokens"] = 64000
	case "high":
		payload["max_tokens"] = 32000
	default:
		payload["max_tokens"] = 16000
	}
	return true
}

func applyDeepSeekReasoningOptions(payload map[string]interface{}, request ChatRequest) {
	effort := strings.ToLower(strings.TrimSpace(request.ReasoningEffort))
	if effort == "none" {
		payload["thinking"] = map[string]interface{}{"type": "disabled"}
		delete(payload, "reasoning_effort")
		return
	}
	if effort != "max" {
		effort = "high"
	}
	payload["thinking"] = map[string]interface{}{"type": "enabled"}
	payload["reasoning_effort"] = effort
}

func toolDefinitionsForRequest(liveOperator, sphereActive bool, requestedAllowlists ...[]string) []map[string]interface{} {
	sphereAllowed := map[string]bool{
		"fs_read_file": true, "fs_list_dir": true, "fs_write_file": true, "fs_replace_file_content": true,
		"code_cartography": true, "nexus_save": true, "nexus_recall": true, "personal_memory_save": true,
		"personal_memory_recall": true, "web_browser": true, "weather_lookup": true, "market_lookup": true, "sphere_write_document": true, "media_control": true,
		"intelligence_status": true, "global_watch_nexus_context": true, "global_watch_focus": true,
		"global_watch_observe": true, "global_watch_toggle_layer": true, "global_watch_schedule_briefing": true,
		"navigate_ui": true, "global_watch_focus_region": true, "global_watch_time_window": true, "global_watch_open_report": true,
		"mail_list_messages": true, "mail_send_message": true,
		"intelligence_collect_source": true, "intelligence_propose_observation": true, "intelligence_create_case": true,
		"osint_add_custom_feed": true, "osint_set_briefing_prompt": true,
	}
	toolsRaw := state.skills.ToToolDefinitions()
	tools := make([]map[string]interface{}, 0, len(toolsRaw))
	allowlist := map[string]bool{}
	filterRequested := len(requestedAllowlists) > 0
	if len(requestedAllowlists) > 0 {
		for _, name := range requestedAllowlists[0] {
			if name = strings.TrimSpace(name); name != "" {
				allowlist[name] = true
			}
		}
	}
	for _, raw := range toolsRaw {
		tool, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		function, ok := tool["function"].(map[string]interface{})
		name, _ := function["name"].(string)
		if filterRequested && !allowlist[name] {
			continue
		}
		if sphereActive && !sphereAllowed[name] {
			continue
		}
		if !liveOperator && (name == "gui_control" || name == "gui_window_control" || name == "vision_context") {
			continue
		}
		tools = append(tools, tool)
	}
	return tools
}

func requestToolDefinitions(request ChatRequest) []map[string]interface{} {
	if request.ToolAllowlist == nil {
		return toolDefinitionsForRequest(request.LiveOperatorActive, request.SphereActive)
	}
	return toolDefinitionsForRequest(request.LiveOperatorActive, request.SphereActive, *request.ToolAllowlist)
}

type ClaudeDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	Thinking    string `json:"thinking,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

type ClaudeContentBlock struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

type ClaudeChunk struct {
	Type         string              `json:"type"`
	Index        int                 `json:"index"`
	Delta        *ClaudeDelta        `json:"delta,omitempty"`
	ContentBlock *ClaudeContentBlock `json:"content_block,omitempty"`
	Usage        *struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
	Message *struct {
		Usage *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage,omitempty"`
	} `json:"message,omitempty"`
}

type ClaudeToolCallState struct {
	Index     int
	ID        string
	Name      string
	Arguments strings.Builder
}

func isVisionModel(modelID string) bool {
	id := strings.ToLower(modelID)
	return strings.Contains(id, "scout") ||
		strings.Contains(id, "llama-4") ||
		strings.Contains(id, "qwen3.6") ||
		strings.Contains(id, "deepseek") ||
		strings.Contains(id, "vision") ||
		strings.Contains(id, "gpt-5") ||
		strings.Contains(id, "gemini-3") ||
		strings.Contains(id, "claude-sonnet") ||
		strings.Contains(id, "claude-opus") ||
		strings.Contains(id, "claude-fable")
}

func messageTextForVisionTrigger(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var parts []string
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, " ")
	default:
		return ""
	}
}

func shouldAttachViewportScreenshot(req ChatRequest, modelID string, messages []map[string]interface{}) bool {
	if !isVisionModel(modelID) {
		return false
	}
	if req.LiveOperatorActive {
		return true
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i]["role"] != "user" {
			continue
		}
		text := strings.ToLower(messageTextForVisionTrigger(messages[i]["content"]))
		triggers := []string{
			"was siehst", "bildschirm", "screen", "screenshot", "sichtbar", "anzeige",
			"auf dem pc", "auf meinem pc", "auf youtube", "youtube", "video", "browser",
			"schau mal", "guck mal", "kannst du sehen",
		}
		for _, trigger := range triggers {
			if strings.Contains(text, trigger) {
				return true
			}
		}
		return false
	}
	return false
}

func sanitizeMessagesForAPI(messages []map[string]interface{}) []map[string]interface{} {
	activeToolResponses := make(map[string]bool)
	for _, msg := range messages {
		if msg["role"] == "tool" {
			if id, ok := msg["tool_call_id"].(string); ok {
				activeToolResponses[id] = true
			}
		}
	}

	var cleanMessages []map[string]interface{}
	for _, msg := range messages {
		if msg["role"] == "assistant" {
			if tcs, ok := msg["tool_calls"]; ok {
				if tcSlice, ok := tcs.([]interface{}); ok {
					var validTCs []interface{}
					for _, tc := range tcSlice {
						if tcMap, ok := tc.(map[string]interface{}); ok {
							if id, ok := tcMap["id"].(string); ok {
								if activeToolResponses[id] {
									validTCs = append(validTCs, tc)
								}
							}
						}
					}
					if len(validTCs) > 0 {
						msg["tool_calls"] = validTCs
					} else {
						delete(msg, "tool_calls")
					}
				}
			}

			content, _ := msg["content"].(string)
			_, hasToolCalls := msg["tool_calls"]
			if strings.TrimSpace(content) == "" && !hasToolCalls {
				msg["content"] = "Aktion ausgeführt."
			}
		}

		if msg["role"] == "tool" {
			if id, ok := msg["tool_call_id"].(string); ok {
				hasMatchingCall := false
				for _, otherMsg := range messages {
					if otherMsg["role"] == "assistant" {
						if otherTCs, ok := otherMsg["tool_calls"].([]interface{}); ok {
							for _, tc := range otherTCs {
								if tcMap, ok := tc.(map[string]interface{}); ok {
									if otherId, ok := tcMap["id"].(string); ok && otherId == id {
										hasMatchingCall = true
										break
									}
								}
							}
						}
					}
				}
				if !hasMatchingCall {
					continue
				}
			}
		}

		cleanMessages = append(cleanMessages, msg)
	}

	return cleanMessages
}

func stripToolsFromMessages(messages []map[string]interface{}) []map[string]interface{} {
	var clean []map[string]interface{}
	for _, msg := range messages {
		if msg["role"] == "tool" {
			toolResult, _ := msg["content"].(string)
			copyMsg := map[string]interface{}{
				"role":    "user",
				"content": fmt.Sprintf("[Ergebnis von Aktion: %s]", toolResult),
			}
			clean = append(clean, copyMsg)
			continue
		}

		copyMsg := make(map[string]interface{})
		for k, v := range msg {
			if k != "tool_calls" {
				copyMsg[k] = v
			}
		}

		if msg["role"] == "assistant" {
			content, _ := msg["content"].(string)
			if tcs, ok := msg["tool_calls"]; ok {
				if tcSlice, ok := tcs.([]interface{}); ok {
					var tcTexts []string
					for _, tc := range tcSlice {
						if tcMap, ok := tc.(map[string]interface{}); ok {
							funcMap, _ := tcMap["function"].(map[string]interface{})
							name, _ := funcMap["name"].(string)
							args, _ := funcMap["arguments"].(string)
							tcTexts = append(tcTexts, fmt.Sprintf("[Aktion ausführen: %s mit Parametern %s]", name, args))
						}
					}
					if len(tcTexts) > 0 {
						if content != "" {
							content += "\n"
						}
						content += strings.Join(tcTexts, "\n")
					}
				}
			}
			copyMsg["content"] = content
		}

		content, _ := copyMsg["content"].(string)
		if strings.TrimSpace(content) == "" {
			copyMsg["content"] = "Aktion ausgeführt."
		}
		clean = append(clean, copyMsg)
	}
	return clean
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := state.getAPIKey()
	dsKey := state.getDeepSeekKey()

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: Invalid JSON body.\n\n")
		if ok {
			flusher.Flush()
		}
		return
	}
	if state.providers == nil {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: Provider registry unavailable.\n\n")
		if ok {
			flusher.Flush()
		}
		return
	}
	if selected, changed := state.providers.SelectAvailable(req.ModelID, state, req.UseTools, req.LiveOperatorActive); changed {
		req.ModelID = selected.ID
		fmt.Fprint(w, "data: [SYSTEM WARNING]: Der angeforderte Provider ist nicht verfügbar; Aethel wurde sicher zu einem konfigurierten Core geroutet.\n\n")
		if ok {
			flusher.Flush()
		}
	}
	_, toolsAllowed, registryErr := state.providers.ValidateChat(req.ModelID, req.SystemPrompt, req.Messages, req.UseTools, req.LiveOperatorActive)
	if registryErr != nil {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: %s.\n\n", registryErr.Error())
		if ok {
			flusher.Flush()
		}
		return
	}
	if req.UseTools && !toolsAllowed {
		fmt.Fprintf(w, "data: [SYSTEM WARNING]: Selected model has no verified tool capability; continuing in chat-only mode.\n\n")
		if ok {
			flusher.Flush()
		}
		req.UseTools = false
	}

	currentSessionChangesMu.Lock()
	currentSessionChanges = []FileChange{}
	currentSessionChangesMu.Unlock()

	isOllama := strings.HasPrefix(req.ModelID, "ollama/")
	isDeepSeek := strings.HasPrefix(req.ModelID, "deepseek/")
	isGemini := strings.HasPrefix(req.ModelID, "gemini/")
	isClaude := strings.HasPrefix(req.ModelID, "claude/")
	isOpenAINative := strings.HasPrefix(req.ModelID, "openai-native/")
	useClaudeAdapter := isClaude
	loggedUsage := false

	geminiKey := state.getGeminiKey()
	claudeKey := state.getClaudeKey()
	openaiNativeKey := state.getOpenAIKey()

	if isDeepSeek && dsKey == "" {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: DeepSeek API Key nicht konfiguriert. Bitte in den Settings eingeben.\n\n")
		if ok {
			flusher.Flush()
		}
		return
	}
	if isGemini && geminiKey == "" {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: Google Gemini API Key nicht konfiguriert. Bitte in den Settings eingeben.\n\n")
		if ok {
			flusher.Flush()
		}
		return
	}
	if isClaude && claudeKey == "" {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: Anthropic Claude API Key nicht konfiguriert. Bitte in den Settings eingeben.\n\n")
		if ok {
			flusher.Flush()
		}
		return
	}
	if isOpenAINative && openaiNativeKey == "" {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: OpenAI API Key nicht konfiguriert. Bitte in den Settings eingeben.\n\n")
		if ok {
			flusher.Flush()
		}
		return
	}
	if !isDeepSeek && !isOllama && !isGemini && !isClaude && !isOpenAINative && apiKey == "" {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: Groq API Key nicht konfiguriert.\n\n")
		if ok {
			flusher.Flush()
		}
		return
	}

	actualModelID := req.ModelID
	if isDeepSeek {
		actualModelID = strings.TrimPrefix(req.ModelID, "deepseek/")
	} else if isOllama {
		actualModelID = strings.TrimPrefix(req.ModelID, "ollama/")
		if actualModelID == "local-fallback" {
			fmt.Fprintf(w, "data: [SYSTEM ERROR]: Kein lokales Ollama gefunden. Bitte vergewissere dich, dass Ollama gestartet ist und Modelle installiert sind.\n\n")
			if ok {
				flusher.Flush()
			}
			return
		}
	} else if isGemini {
		actualModelID = strings.TrimPrefix(req.ModelID, "gemini/")
	} else if isClaude {
		actualModelID = strings.TrimPrefix(req.ModelID, "claude/")
		switch actualModelID {
		case "claude-fable-5":
			actualModelID = "claude-fable-5"
		case "claude-opus-4-8":
			actualModelID = "claude-opus-4-8"
		case "claude-sonnet-4-6":
			actualModelID = "claude-sonnet-4-6"
		case "claude-haiku-4-5":
			actualModelID = "claude-haiku-4-5"
		}
	} else if isOpenAINative {
		actualModelID = strings.TrimPrefix(req.ModelID, "openai-native/")
	} else {
		switch actualModelID {
		case "meta-llama/llama-4-scout-17b-16e-instruct":
			// llama-4-scout-17b-16e-instruct is deprecated and being turned off soon;
			// route it to openai/gpt-oss-120b as recommended by Groq.
			actualModelID = "openai/gpt-oss-120b"
		}
	}

	rawMessages := []map[string]interface{}{}
	systemPromptText := req.SystemPrompt
	if systemPromptText == "" {
		systemPromptText = "You are VGT Aethel."
	}
	if !strings.Contains(systemPromptText, "OPERATING SYSTEM CONTEXT") {
		systemPromptText += agent.GetOSContextPrompt()
	}
	systemPromptText += agent.GetRuntimeContextPrompt()
	if req.SphereActive {
		systemPromptText += `

SPHERE WORKSPACE DIRECTIVES:
- The operator has opened the AETHEL SPHERE WORKSPACE.
- Within this workspace, you have a dedicated visual interface consisting of:
  1. AETHEL WRITER (Document Canvas): For every request to create or replace visible Writer content, invoke sphere_write_document with complete content and format=html|markdown|plain. This is the only canonical Writer mutation contract. Never substitute a checklist, generic fs_write_file, JSON printed as chat text, or an unverified completion claim. The operator sees the verified document update in real time.
  2. AETHEL BROWSER: A dedicated headless web browser. Use the provider-native 'web_browser' tool call for an explicit URL or search only; never print a JSON wrapper, and never navigate to about:blank unless the operator explicitly requests it.
  3. AETHEL CONSOLE shares the exact same conversation and persistent-run context as the chat terminal. WORKSPACE INDEX, RUN DESK, LIVE FLOW, WEATHER PULSE and MEDIA CONSOLE expose shared artifacts, active runs, verified execution progress, weather and explicit media controls.
  4. For current weather, invoke weather_lookup with the requested city. For BTC, ETH, SOL or gold-price requests, invoke market_lookup; GOLD is transparently a PAXG token proxy, never claim it is official XAU fixing. WEATHER PULSE and MARKET PULSE present the same results. For code and agent work, LIVE FLOW shows the execution plan, tool events and verified evidence; keep the operator updated with concise observable progress, not private chain-of-thought.
- You can write whole books and documents inside the constrained workspace. Your voice mode is continuously active, and you should react and communicate interactively.`
	}
	if personalContext := personal.BuildPersonalContext(state.personal); personalContext != "" {
		systemPromptText += "\n\n" + tag("personal_context", personalContext)
	}
	rawMessages = append(rawMessages, map[string]interface{}{
		"role":    "system",
		"content": systemPromptText,
	})
	for _, m := range req.Messages {
		var mapped map[string]interface{}
		if err := json.Unmarshal(m, &mapped); err == nil {
			delete(mapped, "model_id")
			delete(mapped, "run_id")
			delete(mapped, "agent_run_id")
			delete(mapped, "session_id")
			delete(mapped, "timestamp")
			if !isDeepSeek {
				delete(mapped, "reasoning_content")
			}
			rawMessages = append(rawMessages, mapped)
		}
	}

	rawMessages = sanitizeMessagesForAPI(rawMessages)

	if shouldAttachViewportScreenshot(req, actualModelID, rawMessages) {
		lastUserIdx := -1
		for i := len(rawMessages) - 1; i >= 0; i-- {
			if rawMessages[i]["role"] == "user" {
				lastUserIdx = i
				break
			}
		}

		if lastUserIdx != -1 {
			imgBytes, err := skills.GetLatestScreenshot()
			if err == nil && len(imgBytes) > 0 {
				base64Img := base64.StdEncoding.EncodeToString(imgBytes)
				dataURI := "data:image/jpeg;base64," + base64Img

				origText := ""
				if contentStr, ok := rawMessages[lastUserIdx]["content"].(string); ok {
					origText = contentStr
				}

				rawMessages[lastUserIdx]["content"] = []map[string]interface{}{
					{
						"type": "text",
						"text": origText,
					},
					{
						"type": "image_url",
						"image_url": map[string]interface{}{
							"url": dataURI,
						},
					},
				}
				log.Printf("[LiveOperator] Attached desktop screenshot to last user prompt (%d base64 bytes)", len(base64Img))
			} else {
				log.Printf("[LiveOperator] Screenshot capture failed or returned empty: %v", err)
			}
		}
	}

	var compactSummary string
	if isDeepSeek {
		compactSummary, rawMessages = agent.CompactMessagesPreservingSystem(rawMessages)
	}

	var messages []map[string]interface{}

	if isDeepSeek {
		messages = append(messages, rawMessages[0])
		if compactSummary != "" {
			messages = append(messages, map[string]interface{}{
				"role":    "user",
				"content": tag("session_summary", compactSummary),
			})
		}
		messages = append(messages, rawMessages[1:]...)
	} else {
		messages = rawMessages
	}

	if isOllama && req.SystemPrompt != "" {
		firstUserIndex := -1
		for i, m := range messages {
			if m["role"] == "user" {
				firstUserIndex = i
				break
			}
		}
		if firstUserIndex != -1 {
			if origContent, ok := messages[firstUserIndex]["content"].(string); ok {
				if !strings.HasPrefix(origContent, "SYSTEM PROTOCOL:") {
					messages[firstUserIndex]["content"] = fmt.Sprintf("SYSTEM PROTOCOL:\n%s\n\nOPERATOR INPUT:\n%s", req.SystemPrompt, origContent)
				}
			}
		}
	}

	payload := map[string]interface{}{
		"model":          actualModelID,
		"messages":       messages,
		"stream":         true,
		"stream_options": map[string]interface{}{"include_usage": true},
	}

	isGPTOSS := strings.Contains(actualModelID, "gpt-oss")
	applyReasoningOptions(payload, actualModelID, req)

	if isDeepSeek {
		applyDeepSeekReasoningOptions(payload, req)
	} else if !isGPTOSS {
		payload["temperature"] = req.Temperature
	}

	if req.UseTools {
		tools := requestToolDefinitions(req)
		if len(tools) > 0 {
			payload["tools"] = agent.SortedToolDefinitions(tools)
			payload["tool_choice"] = "auto"
		}
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: Marshalling failed.\n\n")
		return
	}

	if useClaudeAdapter {
		var claudeSystem string
		var claudeMessages []map[string]interface{}
		for _, msg := range messages {
			role, _ := msg["role"].(string)
			if role == "system" {
				if c, ok := msg["content"].(string); ok {
					claudeSystem = c
				}
				continue
			}
			if role == "tool" {
				claudeMessages = append(claudeMessages, map[string]interface{}{
					"role": "user",
					"content": []map[string]interface{}{
						{
							"type":        "tool_result",
							"tool_use_id": msg["tool_call_id"],
							"content":     msg["content"],
						},
					},
				})
				continue
			}
			claudeMsg := map[string]interface{}{
				"role":    role,
				"content": msg["content"],
			}
			claudeMessages = append(claudeMessages, claudeMsg)
		}

		var claudeTools []map[string]interface{}
		if req.UseTools {
			for _, m := range requestToolDefinitions(req) {
				if fn, ok := m["function"].(map[string]interface{}); ok {
					params := fn["parameters"]
					claudeTools = append(claudeTools, map[string]interface{}{
						"name":         fn["name"],
						"description":  fn["description"],
						"input_schema": params,
					})
				}
			}
		}

		claudePayload := map[string]interface{}{
			"model":      actualModelID,
			"max_tokens": 16000,
			"messages":   claudeMessages,
			"stream":     true,
		}
		if claudeSystem != "" {
			claudePayload["system"] = claudeSystem
		}
		if len(claudeTools) > 0 {
			claudePayload["tools"] = claudeTools
		}
		claudeReasoning := applyClaudeReasoningOptions(claudePayload, actualModelID, req)
		if req.Temperature > 0 && !claudeReasoning {
			claudePayload["temperature"] = req.Temperature
		}
		claudeBytes, err := json.Marshal(claudePayload)
		if err != nil {
			fmt.Fprintf(w, "data: [SYSTEM ERROR]: Claude payload marshalling failed.\n\n")
			return
		}
		jsonBytes = claudeBytes
	}

	targetURL := provider.GroqURL
	targetKey := apiKey
	if isDeepSeek {
		targetURL = provider.DeepSeekURL
		targetKey = dsKey
	} else if isOllama {
		targetURL = "http://localhost:11434/v1/chat/completions"
		targetKey = "ollama"
	} else if isGemini {
		targetURL = provider.GeminiURL
		targetKey = geminiKey
	} else if isClaude {
		targetURL = provider.ClaudeURL
		targetKey = claudeKey
	} else if isOpenAINative {
		targetURL = provider.OpenAIURL
		targetKey = openaiNativeKey
	}

	httpClient := &http.Client{
		Timeout: 960 * time.Second,
	}

	var resp *http.Response
	var lastErr error
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		apiReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, bytes.NewBuffer(jsonBytes))
		if err != nil {
			fmt.Fprintf(w, "data: [SYSTEM ERROR]: Request creation failed: %v\n\n", err)
			return
		}
		apiReq.Header.Set("Authorization", "Bearer "+targetKey)
		apiReq.Header.Set("Content-Type", "application/json")
		if useClaudeAdapter {
			apiReq.Header.Del("Authorization")
			apiReq.Header.Set("x-api-key", targetKey)
			apiReq.Header.Set("anthropic-version", "2023-06-01")
		}

		resp, err = httpClient.Do(apiReq)
		if err == nil {
			lastErr = nil
			break
		}

		lastErr = err
		log.Printf("[RETRY] Connection failed (attempt %d/%d): %v", attempt, maxRetries, err)

		if attempt < maxRetries {
			fmt.Fprintf(w, "data: [SYSTEM WARNING]: Connection dropped. Retrying (Attempt %d/%d)... \n\n", attempt+1, maxRetries)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			time.Sleep(time.Duration(attempt) * 800 * time.Millisecond)
		}
	}

	if resp == nil {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: Connection failed after %d attempts: %v\n\n", maxRetries, lastErr)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr := string(bodyBytes)

		bodyLower := strings.ToLower(bodyStr)
		isToolError := strings.Contains(bodyLower, "does not support tools") ||
			strings.Contains(bodyLower, "tool_choice") ||
			strings.Contains(bodyLower, "does not support function") ||
			strings.Contains(bodyLower, "tools are not supported") ||
			strings.Contains(bodyLower, "tool calling") ||
			strings.Contains(bodyLower, "unsupported parameter") ||
			strings.Contains(bodyLower, "unknown parameter") ||
			strings.Contains(bodyLower, "unsupported field") ||
			strings.Contains(bodyLower, "invalid parameter") ||
			strings.Contains(bodyLower, "not support") ||
			resp.StatusCode == http.StatusBadRequest

		if isToolError && req.UseTools {
			log.Print("[TOOL FALLBACK] Selected core does not support tools. Retrying without tools.")
			fmt.Fprint(w, "data: [SYSTEM WARNING]: Der ausgewählte Core unterstützt keine Tools. Die Anfrage läuft ohne Tools weiter.\n\n")
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

			delete(payload, "tools")
			delete(payload, "tool_choice")
			payload["messages"] = stripToolsFromMessages(messages)

			retryBytes, err := json.Marshal(payload)
			if err == nil {
				retryReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, bytes.NewBuffer(retryBytes))
				if err == nil {
					retryReq.Header.Set("Authorization", "Bearer "+targetKey)
					retryReq.Header.Set("Content-Type", "application/json")

					retryResp, err := httpClient.Do(retryReq)
					if err == nil && retryResp.StatusCode == http.StatusOK {
						resp = retryResp
						defer resp.Body.Close()
						goto processStream
					}
					if err == nil {
						bodyBytes, _ = io.ReadAll(retryResp.Body)
						bodyStr = string(bodyBytes)
						resp = retryResp
						defer resp.Body.Close()
					}
				}
			}
		}

		provider := "GROQ"
		if isDeepSeek {
			provider = "DEEPSEEK"
		} else if isOllama {
			provider = "OLLAMA"
		} else if isGemini {
			provider = "GEMINI"
		} else if isClaude {
			provider = "CLAUDE"
		} else if isOpenAINative {
			provider = "OPENAI"
		}
		safeDetail := strings.ReplaceAll(bodyStr, targetKey, "[REDACTED]")
		safeDetail = strings.Join(strings.Fields(safeDetail), " ")
		if len([]rune(safeDetail)) > 700 {
			safeDetail = string([]rune(safeDetail)[:700]) + "…"
		}
		log.Printf("[%s API ERROR %d] %s", provider, resp.StatusCode, safeDetail)
		fmt.Fprintf(w, "data: [%s API ERROR %d]: %s\n\n", provider, resp.StatusCode, safeDetail)
		if ok {
			flusher.Flush()
		}
		return
	}

processStream:

	var activeClaudeToolCalls []*ClaudeToolCallState

	buffer := make([]byte, 4096)
	var streamBuf strings.Builder

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			streamBuf.WriteString(string(buffer[:n]))
			lines := strings.Split(streamBuf.String(), "\n")

			if len(lines) > 0 {
				streamBuf.Reset()
				streamBuf.WriteString(lines[len(lines)-1])
				lines = lines[:len(lines)-1]
			}

			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if line == "data: [DONE]" {
					break
				}

				if strings.HasPrefix(line, "data: ") {
					jsonStr := strings.TrimPrefix(line, "data: ")

					if useClaudeAdapter {
						var chunk ClaudeChunk
						if err := json.Unmarshal([]byte(jsonStr), &chunk); err == nil {
							switch chunk.Type {
							case "content_block_start":
								if chunk.ContentBlock != nil && chunk.ContentBlock.Type == "tool_use" {
									activeClaudeToolCalls = append(activeClaudeToolCalls, &ClaudeToolCallState{
										Index: chunk.Index,
										ID:    chunk.ContentBlock.ID,
										Name:  chunk.ContentBlock.Name,
									})
								}
							case "content_block_delta":
								if chunk.Delta != nil {
									if chunk.Delta.Type == "text_delta" {
										content := chunk.Delta.Text
										content = strings.ReplaceAll(content, "\n", "[VGT_NL]")
										content = strings.ReplaceAll(content, "\r", "")
										fmt.Fprintf(w, "data: %s\n\n", content)
										if ok {
											flusher.Flush()
										}
									} else if chunk.Delta.Type == "thinking_delta" {
										thinking := strings.ReplaceAll(chunk.Delta.Thinking, "\n", "[VGT_NL]")
										thinking = strings.ReplaceAll(thinking, "\r", "")
										fmt.Fprintf(w, "data: [[THINKING]]:%s\n\n", thinking)
										if ok {
											flusher.Flush()
										}
									} else if chunk.Delta.Type == "input_json_delta" {
										var targetCall *ClaudeToolCallState
										for _, tc := range activeClaudeToolCalls {
											if tc.Index == chunk.Index {
												targetCall = tc
												break
											}
										}
										if targetCall != nil {
											targetCall.Arguments.WriteString(chunk.Delta.PartialJSON)
											deltaSlice := []map[string]interface{}{
												{
													"index": chunk.Index,
													"id":    targetCall.ID,
													"type":  "function",
													"function": map[string]interface{}{
														"name":      targetCall.Name,
														"arguments": chunk.Delta.PartialJSON,
													},
												},
											}
											deltaJSON, _ := json.Marshal(deltaSlice)
											fmt.Fprintf(w, "data: [[TOOL_DELTA]]:%s\n\n", string(deltaJSON))
											if ok {
												flusher.Flush()
											}
										}
									}
								}
							case "message_delta":
								if chunk.Usage != nil {
									if !loggedUsage {
										loggedUsage = true
										LogAPICall(req.ModelID, chunk.Usage.InputTokens, chunk.Usage.OutputTokens, 0)
									}
									usageJSON, _ := json.Marshal(map[string]interface{}{
										"prompt_tokens":     chunk.Usage.InputTokens,
										"completion_tokens": chunk.Usage.OutputTokens,
										"total_tokens":      chunk.Usage.InputTokens + chunk.Usage.OutputTokens,
									})
									fmt.Fprintf(w, "data: [[USAGE]]:%s\n\n", string(usageJSON))
									if ok {
										flusher.Flush()
									}
								}
							case "message_stop":
								if len(activeClaudeToolCalls) > 0 {
									fmt.Fprintf(w, "data: [[TOOL_COMMIT]]\n\n")
									if ok {
										flusher.Flush()
									}
								}
							}
						}
					} else {
						type DeltaCall struct {
							Index    int    `json:"index"`
							ID       string `json:"id,omitempty"`
							Type     string `json:"type,omitempty"`
							Function struct {
								Name      string `json:"name,omitempty"`
								Arguments string `json:"arguments,omitempty"`
							} `json:"function"`
						}
						type StreamChunk struct {
							Choices []struct {
								Delta struct {
									Content          string      `json:"content,omitempty"`
									Reasoning        string      `json:"reasoning,omitempty"`
									ReasoningContent string      `json:"reasoning_content,omitempty"`
									ToolCalls        []DeltaCall `json:"tool_calls,omitempty"`
								} `json:"delta"`
								FinishReason string `json:"finish_reason,omitempty"`
							} `json:"choices"`
							Usage *struct {
								PromptTokens          int `json:"prompt_tokens"`
								CompletionTokens      int `json:"completion_tokens"`
								TotalTokens           int `json:"total_tokens"`
								PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens"`
								PromptCacheMissTokens int `json:"prompt_cache_miss_tokens"`
								PromptTokensDetails   *struct {
									CachedTokens int `json:"cached_tokens"`
								} `json:"prompt_tokens_details,omitempty"`
							} `json:"usage,omitempty"`
						}

						var chunk StreamChunk
						if err := json.Unmarshal([]byte(jsonStr), &chunk); err == nil {
							if chunk.Usage != nil {
								if !loggedUsage {
									loggedUsage = true
									LogAPICall(req.ModelID, chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens, chunk.Usage.PromptCacheHitTokens)
								}
								if chunk.Usage.PromptTokensDetails != nil && chunk.Usage.PromptCacheHitTokens == 0 {
									chunk.Usage.PromptCacheHitTokens = chunk.Usage.PromptTokensDetails.CachedTokens
								}
								usageJSON, _ := json.Marshal(chunk.Usage)
								fmt.Fprintf(w, "data: [[USAGE]]:%s\n\n", string(usageJSON))
								if ok {
									flusher.Flush()
								}
								if (isDeepSeek || isGPTOSS) && chunk.Usage.PromptTokens > 0 {
									go agent.LogCacheMetrics(agent.DeepSeekUsage{
										PromptTokens:          chunk.Usage.PromptTokens,
										CompletionTokens:      chunk.Usage.CompletionTokens,
										TotalTokens:           chunk.Usage.TotalTokens,
										PromptCacheHitTokens:  chunk.Usage.PromptCacheHitTokens,
										PromptCacheMissTokens: chunk.Usage.PromptCacheMissTokens,
									})
								}
							}

							if len(chunk.Choices) > 0 {
								choice := chunk.Choices[0]

								reasoningChunk := choice.Delta.ReasoningContent
								if reasoningChunk == "" {
									reasoningChunk = choice.Delta.Reasoning
								}
								if reasoningChunk != "" {
									reasoningChunk = strings.ReplaceAll(reasoningChunk, "\n", "[VGT_NL]")
									reasoningChunk = strings.ReplaceAll(reasoningChunk, "\r", "")
									fmt.Fprintf(w, "data: [[THINKING]]:%s\n\n", reasoningChunk)
									if ok {
										flusher.Flush()
									}
								}

								if choice.Delta.Content != "" {
									content := choice.Delta.Content
									content = strings.ReplaceAll(content, "\n", "[VGT_NL]")
									content = strings.ReplaceAll(content, "\r", "")
									fmt.Fprintf(w, "data: %s\n\n", content)
									if ok {
										flusher.Flush()
									}
								}

								if len(choice.Delta.ToolCalls) > 0 {
									toolCallJSON, _ := json.Marshal(choice.Delta.ToolCalls)
									fmt.Fprintf(w, "data: [[TOOL_DELTA]]:%s\n\n", string(toolCallJSON))
									if ok {
										flusher.Flush()
									}
								}

								if choice.FinishReason == "tool_calls" {
									fmt.Fprintf(w, "data: [[TOOL_COMMIT]]\n\n")
									if ok {
										flusher.Flush()
									}
								}
							}
						}
					}
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintf(w, "data: [SYSTEM ERROR]: Stream break: %v\n\n", err)
			break
		}
	}

	currentSessionChangesMu.Lock()
	if len(currentSessionChanges) > 0 {
		changesJSON, _ := json.Marshal(currentSessionChanges)
		fmt.Fprintf(w, "data: [[FILE_CHANGES]]:%s\n\n", string(changesJSON))
		if ok {
			flusher.Flush()
		}
	}
	currentSessionChangesMu.Unlock()
}

func tag(name, value string) string {
	return fmt.Sprintf("<%s>\n%s\n</%s>", name, value, name)
}
