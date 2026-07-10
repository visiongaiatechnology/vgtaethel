package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

type AgentToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type agentInferenceResult struct {
	Text         string
	Thinking     string
	ToolCalls    []AgentToolCall
	InputTokens  int
	OutputTokens int
	CachedTokens int
	Err          error
}

type ChatAgentStartRequest struct {
	Objective          string            `json:"objective"`
	ProfileID          string            `json:"profile_id"`
	ModelID            string            `json:"model_id"`
	SystemPrompt       string            `json:"system_prompt"`
	Messages           []json.RawMessage `json:"messages"`
	CostBudgetUSD      float64           `json:"cost_budget_usd"`
	MaxAgentTurns      int               `json:"max_agent_turns"`
	LiveOperatorActive bool              `json:"live_operator_active"`
}

var chatAgentWorkers = struct {
	sync.Mutex
	active map[string]bool
}{active: make(map[string]bool)}

func handleChatAgentRuns(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request ChatAgentStartRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid chat agent request", http.StatusBadRequest)
		return
	}
	if request.ProfileID == "" {
		request.ProfileID = "developer"
	}
	spec, toolsAllowed, err := state.providers.ValidateChat(request.ModelID, request.SystemPrompt, request.Messages, true, request.LiveOperatorActive)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !toolsAllowed {
		http.Error(w, "selected model is not verified for tool-capable agent runs", http.StatusBadRequest)
		return
	}
	if request.CostBudgetUSD <= 0 {
		request.CostBudgetUSD = spec.DefaultRunBudget
	}
	run, err := state.runs.Create(CreateRunRequest{
		Objective: request.Objective, ProfileID: request.ProfileID, ModelID: request.ModelID,
		Mode: "chat_agent", LiveOperator: request.LiveOperatorActive, SystemPrompt: request.SystemPrompt, AgentMessages: request.Messages,
		MaxAgentTurns: request.MaxAgentTurns, CostBudgetUSD: request.CostBudgetUSD,
		Steps: []RunStep{{Kind: RunStepPlan, Title: "Agentenziel und Ausführungsrahmen validieren"}},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	run, err = state.runs.Start(run.ID)
	if err != nil {
		http.Error(w, "Run could not start", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(run)
	go driveChatAgentRun(run.ID, request.LiveOperatorActive)
}

func driveChatAgentRun(id string, liveOperator bool) {
	chatAgentWorkers.Lock()
	if chatAgentWorkers.active[id] {
		chatAgentWorkers.Unlock()
		return
	}
	chatAgentWorkers.active[id] = true
	chatAgentWorkers.Unlock()
	defer func() {
		chatAgentWorkers.Lock()
		delete(chatAgentWorkers.active, id)
		chatAgentWorkers.Unlock()
	}()

	for {
		run, ok := state.runs.Get(id)
		if !ok || run.Status != RunRunning {
			return
		}
		if run.AgentTurn >= run.MaxAgentTurns {
			// Pause instead of hard-failing so the operator can review the
			// trace and resume the run with a higher turn budget if needed.
			state.runs.PauseOnTurnLimit(id, run.AgentTurn)
			return
		}

		progressed, stopped, err := executePendingChatSteps(run)
		if err != nil {
			state.runs.FailAgent(id, err.Error())
			return
		}
		if stopped {
			return
		}
		if progressed {
			continue
		}

		run, _ = state.runs.Get(id)
		result := invokeAgentModel(run, run.LiveOperator || liveOperator)
		if result.Err != nil {
			state.runs.FailAgent(id, result.Err.Error())
			return
		}
		cost := CalculateInferenceCost(run.ModelID, result.InputTokens, result.OutputTokens, result.CachedTokens)
		if cost > 0 {
			updated, costErr := state.runs.RecordCost(id, cost)
			if costErr != nil {
				state.runs.FailAgent(id, costErr.Error())
				return
			}
			if updated.Status == RunPaused {
				return
			}
		}

		messages := append([]json.RawMessage(nil), run.AgentMessages...)
		assistantMessage, err := marshalAssistantAgentMessage(result)
		if err != nil {
			state.runs.FailAgent(id, "Assistant response could not be persisted.")
			return
		}
		messages = append(messages, assistantMessage)
		if _, err := state.runs.UpdateAgentProgress(id, messages, run.AgentTurn+1, result.Text); err != nil {
			state.runs.FailAgent(id, err.Error())
			return
		}
		if len(result.ToolCalls) == 0 {
			if strings.TrimSpace(result.Text) == "" {
				state.runs.FailAgent(id, "Provider returned an empty final response.")
				return
			}
			_, err := state.runs.CompleteAgent(id, result.Text, messages)
			if err != nil {
				state.runs.FailAgent(id, err.Error())
			}
			return
		}
		if _, err := state.runs.AppendAgentToolSteps(id, result.ToolCalls); err != nil {
			state.runs.FailAgent(id, err.Error())
			return
		}
	}
}

func executePendingChatSteps(run AgentRun) (progressed, stopped bool, err error) {
	for _, step := range run.Steps {
		if step.Status != StepPending && step.Status != StepWaitingApproval {
			continue
		}
		if step.Status == StepWaitingApproval {
			return false, true, nil
		}
		updated, advanceErr := state.runs.Advance(run.ID, state.policy, state.skills)
		if advanceErr != nil {
			return false, false, advanceErr
		}
		if updated.Status == RunWaitingApproval || updated.Status == RunPaused {
			return true, true, nil
		}
		if updated.Status == RunFailed {
			return true, true, errors.New(updated.FailureReason)
		}
		if step.Kind == RunStepTool {
			completed := findRunStep(updated, step.ID)
			if completed == nil || completed.Status != StepVerified {
				return true, false, errors.New("tool step did not produce a verified result")
			}
			messages := append([]json.RawMessage(nil), updated.AgentMessages...)
			if !hasToolResult(messages, completed.ToolCallID) {
				toolMessage, marshalErr := json.Marshal(map[string]interface{}{"role": "tool", "tool_call_id": completed.ToolCallID, "name": completed.ToolName, "content": completed.Result})
				if marshalErr != nil {
					return true, false, marshalErr
				}
				messages = append(messages, toolMessage)
				if _, updateErr := state.runs.UpdateAgentProgress(run.ID, messages, updated.AgentTurn, ""); updateErr != nil {
					return true, false, updateErr
				}
			}
		}
		return true, false, nil
	}
	return false, false, nil
}

func findRunStep(run AgentRun, id string) *RunStep {
	for index := range run.Steps {
		if run.Steps[index].ID == id {
			return &run.Steps[index]
		}
	}
	return nil
}

func hasToolResult(messages []json.RawMessage, callID string) bool {
	for _, raw := range messages {
		var message map[string]interface{}
		if json.Unmarshal(raw, &message) == nil && message["role"] == "tool" && message["tool_call_id"] == callID {
			return true
		}
	}
	return false
}

func marshalAssistantAgentMessage(result agentInferenceResult) (json.RawMessage, error) {
	message := map[string]interface{}{"role": "assistant", "content": result.Text}
	if result.Thinking != "" {
		message["reasoning_content"] = result.Thinking
	}
	if len(result.ToolCalls) > 0 {
		calls := make([]map[string]interface{}, 0, len(result.ToolCalls))
		for _, call := range result.ToolCalls {
			calls = append(calls, map[string]interface{}{"id": call.ID, "type": "function", "function": map[string]interface{}{"name": call.Name, "arguments": string(call.Arguments)}})
		}
		message["tool_calls"] = calls
	}
	return json.Marshal(message)
}

func invokeAgentModel(run AgentRun, liveOperator bool) agentInferenceResult {
	payload, err := json.Marshal(ChatRequest{ModelID: run.ModelID, Messages: run.AgentMessages, Temperature: .15, UseTools: shouldEnableAgentTools(run.Objective), SystemPrompt: run.SystemPrompt, LiveOperatorActive: liveOperator})
	if err != nil {
		return agentInferenceResult{Err: err}
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat", bytes.NewReader(payload))
	recorder := httptest.NewRecorder()
	handleChat(recorder, req)
	result := parseAgentSSE(recorder.Body.String())
	promoteCodeCartographyFallback(&result, run.Objective)
	return result
}

var (
	fallbackCartographyPath   = regexp.MustCompile(`"path"\s*:\s*"([^"]+)"`)
	fallbackCartographyOutput = regexp.MustCompile(`"output_path"\s*:\s*"([^"]+)"`)
	fallbackCartographyMax    = regexp.MustCompile(`"max_files"\s*:\s*([0-9]+)`)
)

// Some otherwise tool-capable providers occasionally serialize a dedicated
// cartography call as a bare JSON object in assistant text. This narrow bridge
// accepts only that exact, explicitly requested operation; it never promotes
// arbitrary model text into a tool call.
func promoteCodeCartographyFallback(result *agentInferenceResult, objective string) {
	if result == nil || len(result.ToolCalls) != 0 || !strings.Contains(strings.ToLower(objective), "kartografie") {
		return
	}
	raw := strings.TrimSpace(result.Text)
	for _, candidate := range bareJSONObjectCandidates(raw) {
		pathMatch := fallbackCartographyPath.FindStringSubmatch(candidate)
		outputMatch := fallbackCartographyOutput.FindStringSubmatch(candidate)
		if len(pathMatch) != 2 || len(outputMatch) != 2 || strings.TrimSpace(pathMatch[1]) == "" {
			continue
		}
		output := strings.ReplaceAll(outputMatch[1], `\\`, `\`)
		if filepath.Ext(output) != ".md" {
			continue
		}
		args := CodeCartographyArgs{Path: strings.ReplaceAll(pathMatch[1], `\\`, `\`), OutputPath: output}
		if maxMatch := fallbackCartographyMax.FindStringSubmatch(candidate); len(maxMatch) == 2 {
			var maxFiles int
			if _, err := fmt.Sscanf(maxMatch[1], "%d", &maxFiles); err == nil && maxFiles >= 1 && maxFiles <= cartographyMaxFiles {
				args.MaxFiles = maxFiles
			}
		}
		encoded, err := json.Marshal(args)
		if err != nil {
			return
		}
		result.Err = nil
		result.ToolCalls = []AgentToolCall{{ID: "fallback_cartography_1", Name: "code_cartography", Arguments: encoded}}
		result.Text = ""
		return
	}

}

// bareJSONObjectCandidates extracts independent JSON-shaped objects without
// requiring their string values to be valid JSON escapes. This is necessary
// for Windows paths that a provider printed with single backslashes.
func bareJSONObjectCandidates(raw string) []string {
	objects := make([]string, 0, 3)
	depth, start := 0, -1
	inString, escaped := false, false
	for index := 0; index < len(raw); index++ {
		char := raw[index]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '"' {
				inString = false
			}
			continue
		}
		switch char {
		case '"':
			inString = true
		case '{':
			if depth == 0 {
				start = index
			}
			depth++
		case '}':
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 && start >= 0 {
				objects = append(objects, raw[start:index+1])
				start = -1
			}
		}
	}
	return objects
}

// shouldEnableAgentTools is deliberately conservative. A persistent run is
// also used for ordinary conversation, where exposing control tools invites a
// model to act instead of simply answering. Tool schemas are supplied only for
// an explicit operational request; all actual execution remains policy-gated.
func shouldEnableAgentTools(objective string) bool {
	text := strings.ToLower(strings.TrimSpace(objective))
	if text == "" {
		return false
	}
	operationalTerms := []string{
		"datei", "ordner", "verzeichnis", "workspace", "pfad", "öffne", "schau", "prüf", "analys", "liste", "suche", "recherch",
		"erstelle", "schreib", "ändere", "bearbeit", "sortier", "verschieb", "lösche", "starte", "führe", "klick", "tippe",
		"browser", "youtube", "musik", "video", "screenshot", "bildschirm", "computer", "terminal", "code", "kartografie", "architektur", "programm", "build", "test", "install",
	}
	for _, term := range operationalTerms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func parseAgentSSE(stream string) agentInferenceResult {
	type buffer struct {
		ID, Name, Args string
		Index          int
	}
	buffers := map[int]*buffer{}
	result := agentInferenceResult{}
	scanner := bufio.NewScanner(strings.NewReader(stream))
	scanner.Buffer(make([]byte, 4096), 8*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimPrefix(line, "data:")
		if strings.HasPrefix(data, " ") {
			data = data[1:]
		}
		switch {
		case strings.HasPrefix(data, "[[THINKING]]:"):
			result.Thinking += strings.ReplaceAll(strings.TrimPrefix(data, "[[THINKING]]:"), "[VGT_NL]", "\n")
		case strings.HasPrefix(data, "[[USAGE]]:"):
			var usage struct {
				PromptTokens         int `json:"prompt_tokens"`
				CompletionTokens     int `json:"completion_tokens"`
				PromptCacheHitTokens int `json:"prompt_cache_hit_tokens"`
			}
			if json.Unmarshal([]byte(strings.TrimPrefix(data, "[[USAGE]]:")), &usage) == nil {
				result.InputTokens = usage.PromptTokens
				result.OutputTokens = usage.CompletionTokens
				result.CachedTokens = usage.PromptCacheHitTokens
			}
		case strings.HasPrefix(data, "[[TOOL_DELTA]]:"):
			var deltas []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			}
			if json.Unmarshal([]byte(strings.TrimPrefix(data, "[[TOOL_DELTA]]:")), &deltas) == nil {
				for _, delta := range deltas {
					current := buffers[delta.Index]
					if current == nil {
						current = &buffer{Index: delta.Index}
						buffers[delta.Index] = current
					}
					if delta.ID != "" {
						current.ID = delta.ID
					}
					if delta.Function.Name != "" {
						current.Name = delta.Function.Name
					}
					current.Args += delta.Function.Arguments
				}
			}
		case strings.HasPrefix(data, "[SYSTEM ERROR]") || strings.Contains(data, " API ERROR "):
			result.Err = errors.New(data)
		case strings.HasPrefix(data, "[[") || strings.HasPrefix(data, "[SYSTEM WARNING]"):
			continue
		default:
			result.Text += strings.ReplaceAll(data, "[VGT_NL]", "\n")
		}
	}
	if err := scanner.Err(); err != nil {
		result.Err = err
	}
	indices := make([]int, 0, len(buffers))
	for index := range buffers {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	for _, index := range indices {
		call := buffers[index]
		arguments := json.RawMessage(call.Args)
		if call.ID == "" || call.Name == "" || !json.Valid(arguments) {
			result.Err = fmt.Errorf("provider emitted an invalid tool call")
			return result
		}
		result.ToolCalls = append(result.ToolCalls, AgentToolCall{ID: call.ID, Name: call.Name, Arguments: arguments})
	}
	return result
}
