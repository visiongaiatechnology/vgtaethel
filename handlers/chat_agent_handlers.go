package handlers

import (
	"encoding/json"
	"go-aethel/agent"
	"net/http"
)

func HandleChatAgentRuns(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request agent.ChatAgentStartRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid chat agent request", http.StatusBadRequest)
		return
	}
	request.ProfileID = agent.ResolveChatAgentProfile(request)
	if request.SphereActive {
		request.SystemPrompt += `

SPHERE WORKSPACE DIRECTIVES:
- The operator has opened the AETHEL SPHERE WORKSPACE.
- Within this workspace, you have a dedicated visual interface consisting of:
  1. AETHEL WRITER (Document Canvas): For every request to create or replace visible Writer content, invoke sphere_write_document with complete content and format=html|markdown|plain. This is the only canonical Writer mutation contract. Never substitute a checklist, generic fs_write_file, JSON printed as chat text, or an unverified completion claim. The operator sees the verified document update in real time.
  2. AETHEL BROWSER: A dedicated headless web browser. Use the provider-native 'web_browser' tool call for an explicit URL or search only; never print a JSON wrapper, and never navigate to about:blank unless the operator explicitly requests it.
  3. AETHEL CONSOLE shares the exact same conversation and persistent-run context as the chat terminal. WORKSPACE INDEX, RUN DESK, LIVE FLOW, WEATHER PULSE and MEDIA CONSOLE expose shared artifacts, active runs, verified execution progress, weather and explicit media controls.
  4. For current weather, invoke weather_lookup with the requested city. For BTC, ETH, SOL or gold-price requests, invoke market_lookup; GOLD is transparently a PAXG token proxy, never claim it is official XAU fixing. WEATHER PULSE and MARKET PULSE present the same results. For code and agent work, LIVE FLOW shows the execution plan, tool events and verified evidence; keep the operator updated with concise observable progress, not private chain-of-thought.
- You can write whole books and documents inside the constrained workspace. Your voice mode is continuously active, and you should react and communicate interactively.`
	}
	if selected, changed := state.providers.SelectAvailable(request.ModelID, state, false, request.LiveOperatorActive); changed {
		request.ModelID = selected.ID
	}
	if request.OrchestratorModelID == "" {
		request.OrchestratorModelID = request.ModelID
	}
	requiresOrchestrator := agent.RequiresChatOrchestrator(request)
	if requiresOrchestrator {
		if selected, changed := state.providers.SelectAvailable(request.OrchestratorModelID, state, true, request.LiveOperatorActive); changed {
			request.OrchestratorModelID = selected.ID
		}
	} else {
		request.OrchestratorModelID = request.ModelID
	}
	spec, _, err := state.providers.ValidateChat(request.ModelID, request.SystemPrompt, request.Messages, false, request.LiveOperatorActive)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if requiresOrchestrator {
		if _, orchestratorTools, orchestratorErr := state.providers.ValidateChat(request.OrchestratorModelID, request.SystemPrompt, request.Messages, true, request.LiveOperatorActive); orchestratorErr != nil || !orchestratorTools {
			http.Error(w, "orchestrator model is not verified for tool-capable runs", http.StatusBadRequest)
			return
		}
	}
	if request.CostBudgetUSD <= 0 {
		request.CostBudgetUSD = spec.DefaultRunBudget
	}
	runMode := "chat_agent"
	if request.Mode == "agent_team" {
		runMode = request.Mode
	}
	run, err := state.runs.Create(agent.CreateRunRequest{
		Objective: request.Objective, ProfileID: request.ProfileID, ModelID: request.ModelID, OrchestratorModelID: request.OrchestratorModelID,
		Mode: runMode, LiveOperator: request.LiveOperatorActive, SphereActive: request.SphereActive, SystemPrompt: request.SystemPrompt, AgentMessages: request.Messages,
		MaxAgentTurns: request.MaxAgentTurns, CostBudgetUSD: request.CostBudgetUSD, ReasoningEffort: request.ReasoningEffort,
		Steps: []agent.RunStep{{Kind: agent.RunStepPlan, Title: "Agentenziel und Ausführungsrahmen validieren"}},
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
	go agent.DriveChatAgentRun(run.ID, request.LiveOperatorActive)
}
