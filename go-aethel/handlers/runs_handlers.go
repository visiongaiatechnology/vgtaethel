package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"go-aethel/agent"
	"go-aethel/security"
	"go-aethel/system"
)

func handleRuns(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if state == nil || state.runs == nil {
		http.Error(w, "Run engine unavailable", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"runs":     state.runs.List(),
			"profiles": state.runs.Profiles(),
		})
	case http.MethodPost:
		var req agent.CreateRunRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "Invalid run request", http.StatusBadRequest)
			return
		}
		if req.ModelID != "" {
			spec, ok := state.providers.Resolve(req.ModelID)
			if !ok {
				http.Error(w, "Run model is not registered", http.StatusBadRequest)
				return
			}
			if req.CostBudgetUSD <= 0 {
				req.CostBudgetUSD = spec.DefaultRunBudget
			}
		}
		run, err := state.runs.Create(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(run)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleRunsPath(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if state == nil || state.runs == nil {
		http.Error(w, "Run engine unavailable", http.StatusServiceUnavailable)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/v1/runs/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" || len(parts) > 2 || !security.SafeResourceIDPattern.MatchString(parts[0]) {
		http.Error(w, "Invalid run path", http.StatusBadRequest)
		return
	}
	id := parts[0]
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		run, ok := state.runs.Get(id)
		if !ok {
			http.Error(w, "Run not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(run)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	action := parts[1]
	if action == "approval" {
		var request struct {
			ApprovalToken string `json:"approval_token"`
		}
		if decodeErr := json.NewDecoder(r.Body).Decode(&request); decodeErr != nil && decodeErr != io.EOF {
			http.Error(w, "Invalid approval payload", http.StatusBadRequest)
			return
		}
		step, pending := state.runs.PendingApproval(id)
		if !pending {
			http.Error(w, "Run is not waiting for approval", http.StatusConflict)
			return
		}
		_, decision, report := state.policy.Evaluate(step.ToolName, string(step.ToolArgs), false)
		if decision != "needs_approval" {
			http.Error(w, "Approval target is no longer eligible", http.StatusConflict)
			return
		}
		if request.ApprovalToken == "" {
			grant, token, issueErr := state.approvals.Issue(step.ToolName, string(step.ToolArgs), report.Capability, id)
			if issueErr != nil {
				http.Error(w, "Approval service unavailable", http.StatusServiceUnavailable)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"approval_id": grant.ID, "approval_token": token, "expires_at": grant.ExpiresAt, "tool_name": step.ToolName, "tool_args": step.ToolArgs, "capability": report.Capability, "risk_level": report.RiskLevel})
			return
		}
		run, approveErr := state.runs.Approve(id, request.ApprovalToken, state.approvals, state.policy)
		if approveErr != nil {
			http.Error(w, "Approval invalid or expired", http.StatusForbidden)
			return
		}
		json.NewEncoder(w).Encode(run)
		if run.Mode == "chat_agent" || run.Mode == "agent_team" {
			go agent.DriveChatAgentRun(id, run.LiveOperator)
		}
		return
	}
	var (
		run       agent.AgentRun
		err       error
		autoDrive bool
	)
	switch action {
	case "start", "resume":
		run, err = state.runs.Start(id)
		autoDrive = true
	case "pause":
		run, err = state.runs.Pause(id)
	case "cancel":
		run, err = state.runs.Cancel(id)
	case "advance":
		run, err = state.runs.Advance(id, state.policy, state.skills)
	case "cost":
		var req struct {
			CostUSD float64 `json:"cost_usd"`
		}
		if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
			http.Error(w, "Invalid cost payload", http.StatusBadRequest)
			return
		}
		run, err = state.runs.RecordCost(id, req.CostUSD)
	default:
		http.Error(w, "Unknown run action", http.StatusNotFound)
		return
	}
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Run not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	json.NewEncoder(w).Encode(run)
	if autoDrive {
		if run.Mode == "chat_agent" || run.Mode == "agent_team" {
			go agent.DriveChatAgentRun(id, run.LiveOperator)
		}
	}
}

func handleArtifacts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	artifacts, err := system.DefaultFileSnapshots.ListArtifacts()
	if err != nil {
		http.Error(w, "Artifact store unavailable", http.StatusServiceUnavailable)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"artifacts": artifacts})
}
