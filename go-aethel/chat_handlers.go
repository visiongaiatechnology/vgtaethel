package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type ChatRequest struct {
	ModelID            string            `json:"model_id"`
	Messages           []json.RawMessage `json:"messages"`
	Temperature        float64           `json:"temperature"`
	UseTools           bool              `json:"use_tools"`
	SystemPrompt       string            `json:"system_prompt"`
	LiveOperatorActive bool              `json:"live_operator_active"`
}

func toolDefinitionsForRequest(liveOperator bool) []map[string]interface{} {
	toolsRaw := state.skills.ToToolDefinitions()
	tools := make([]map[string]interface{}, 0, len(toolsRaw))
	for _, raw := range toolsRaw {
		tool, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		function, ok := tool["function"].(map[string]interface{})
		name, _ := function["name"].(string)
		if !liveOperator && (name == "gui_control" || name == "gui_window_control" || name == "vision_context") {
			continue
		}
		tools = append(tools, tool)
	}
	return tools
}

type ClaudeDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
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
			continue
		}
		copyMsg := make(map[string]interface{})
		for k, v := range msg {
			if k != "tool_calls" {
				copyMsg[k] = v
			}
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
	requestedModelID := req.ModelID
	if selected, changed := state.providers.SelectAvailable(req.ModelID, state, req.UseTools, req.LiveOperatorActive); changed {
		req.ModelID = selected.ID
		fmt.Fprintf(w, "data: [SYSTEM WARNING]: Provider for '%s' is unavailable; securely routed to '%s'.\n\n", requestedModelID, selected.ID)
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
		case "openai/gpt-oss-120b":
			actualModelID = "llama-3.3-70b-versatile"
		case "openai/gpt-oss-20b":
			actualModelID = "llama-3.1-8b-instant"
		case "qwen/qwen3.6-27b":
			actualModelID = "llama-3.3-70b-versatile"
		case "meta-llama/llama-4-scout-17b-16e-instruct":
			actualModelID = "llama-3.1-8b-instant"
		}
	}

	rawMessages := []map[string]interface{}{}
	systemPromptText := req.SystemPrompt
	if systemPromptText == "" {
		systemPromptText = "You are VGT Aethel."
	}
	if !strings.Contains(systemPromptText, "OPERATING SYSTEM CONTEXT") {
		systemPromptText += getOSContextPrompt()
	}
	systemPromptText += getRuntimeContextPrompt()
	if personalContext := BuildPersonalContext(state.personal); personalContext != "" {
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
			imgBytes, err := GetLatestScreenshot()
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
		compactSummary, rawMessages = CompactMessagesPreservingSystem(rawMessages)
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
	isQwen3 := strings.Contains(actualModelID, "qwen3")

	if isGPTOSS {
		payload["include_reasoning"] = true
		if req.UseTools {
			payload["reasoning_effort"] = "medium"
		} else {
			payload["reasoning_effort"] = "high"
		}
	} else if isQwen3 {
		payload["reasoning_format"] = "parsed"
		if strings.Contains(actualModelID, "qwen3.6") {
			payload["reasoning_effort"] = "default"
		}
	}

	if isDeepSeek {
		payload["thinking"] = map[string]interface{}{"type": "enabled"}
		payload["reasoning_effort"] = "max"
	} else if !isGPTOSS {
		payload["temperature"] = req.Temperature
	}

	if req.UseTools {
		tools := toolDefinitionsForRequest(req.LiveOperatorActive)
		payload["tools"] = SortedToolDefinitions(tools)
		payload["tool_choice"] = "auto"
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
			for _, m := range toolDefinitionsForRequest(req.LiveOperatorActive) {
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
		if req.Temperature > 0 {
			claudePayload["temperature"] = req.Temperature
		}
		claudeBytes, err := json.Marshal(claudePayload)
		if err != nil {
			fmt.Fprintf(w, "data: [SYSTEM ERROR]: Claude payload marshalling failed.\n\n")
			return
		}
		jsonBytes = claudeBytes
	}

	targetURL := groqURL
	targetKey := apiKey
	if isDeepSeek {
		targetURL = deepseekURL
		targetKey = dsKey
	} else if isOllama {
		targetURL = "http://localhost:11434/v1/chat/completions"
		targetKey = "ollama"
	} else if isGemini {
		targetURL = geminiURL
		targetKey = geminiKey
	} else if isClaude {
		targetURL = claudeURL
		targetKey = claudeKey
	} else if isOpenAINative {
		targetURL = openaiURL
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
			log.Printf("[TOOL FALLBACK] Model %s does not support tools. Retrying without tools.", actualModelID)
			fmt.Fprintf(w, "data: [SYSTEM WARNING]: Core-Modell '%s' unterstützt keine Tools. Führe Anfrage ohne Tools aus... \n\n", actualModelID)
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
									go LogCacheMetrics(DeepSeekUsage{
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
