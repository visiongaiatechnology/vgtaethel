// STATUS: DIAMANT VGT SUPREME
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-aethel/agent"
	"go-aethel/coder"
	"go-aethel/security"
)

var coderSessions *coder.Manager

func InitCoderManager(manager *coder.Manager) { coderSessions = manager }

type coderStartRequest struct {
	Request           string  `json:"request"`
	WorkerModel       string  `json:"worker_model"`
	OrchestratorModel string  `json:"orchestrator_model"`
	ReasoningEffort   string  `json:"reasoning_effort,omitempty"`
	SystemPrompt      string  `json:"system_prompt,omitempty"`
	MaxTurns          int     `json:"max_turns,omitempty"`
	CostBudgetUSD     float64 `json:"cost_budget_usd,omitempty"`
}

func HandleCoderSessions(w http.ResponseWriter, r *http.Request) {
	setCodeJSONHeaders(w)
	if coderSessions == nil || state == nil || state.runs == nil {
		codeError(w, http.StatusServiceUnavailable, "coder_unavailable", "VGT Coder service is unavailable.")
		return
	}
	switch r.Method {
	case http.MethodGet:
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		writeCodeJSON(w, http.StatusOK, map[string]interface{}{"sessions": coderSessions.List(limit)})
	case http.MethodPost:
		startCoderSession(w, r)
	default:
		codeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET and POST are supported.")
	}
}

func HandleCoderSessionPath(w http.ResponseWriter, r *http.Request) {
	setCodeJSONHeaders(w)
	if coderSessions == nil || state == nil || state.runs == nil {
		codeError(w, http.StatusServiceUnavailable, "coder_unavailable", "VGT Coder service is unavailable.")
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/coder/sessions/"), "/"), "/")
	if len(parts) == 0 || security.ValidateResourceID(parts[0]) != nil {
		codeError(w, http.StatusBadRequest, "invalid_session", "Invalid coder session identifier.")
		return
	}
	sessionID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	switch {
	case r.Method == http.MethodGet && action == "":
		session, err := syncCoderSession(sessionID)
		if err != nil {
			codeError(w, http.StatusNotFound, "session_not_found", "Coder session not found.")
			return
		}
		writeCodeJSON(w, http.StatusOK, coder.PublicSession(session))
	case r.Method == http.MethodGet && action == "diff":
		session, ok := coderSessions.Get(sessionID)
		activeRoot, rootErr := CurrentCodeWorkspaceRoot()
		if !ok || rootErr != nil || !sameCanonicalPath(activeRoot, session.ProjectRoot) {
			codeError(w, http.StatusForbidden, "project_not_authorized", "Open the session project before reviewing its files.")
			return
		}
		diff, err := coderSessions.Diff(sessionID, r.URL.Query().Get("path"), normalizedCoderScope(r.URL.Query().Get("scope")))
		if err != nil {
			codeError(w, http.StatusBadRequest, "diff_unavailable", "Requested diff is unavailable.")
			return
		}
		writeCodeJSON(w, http.StatusOK, diff)
	case r.Method == http.MethodPost && action == "cancel":
		session, ok := coderSessions.Get(sessionID)
		if !ok || session.RunID == "" {
			codeError(w, http.StatusNotFound, "session_not_found", "Coder session not found.")
			return
		}
		if _, err := state.runs.Cancel(session.RunID); err != nil {
			codeError(w, http.StatusConflict, "cancel_rejected", "Coder session could not be cancelled.")
			return
		}
		updated, _ := syncCoderSession(sessionID)
		writeCodeJSON(w, http.StatusOK, coder.PublicSession(updated))
	default:
		codeError(w, http.StatusMethodNotAllowed, "unsupported_action", "Coder session action is unsupported.")
	}
}

func startCoderSession(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request coderStartRequest
	if err := decoder.Decode(&request); err != nil {
		codeError(w, http.StatusBadRequest, "invalid_request", "Invalid coder session request.")
		return
	}
	projectRoot, err := CurrentCodeWorkspaceRoot()
	if err != nil {
		codeError(w, http.StatusConflict, "no_project", "Select a coding project first.")
		return
	}
	worker := strings.TrimSpace(request.WorkerModel)
	if selected, changed := state.providers.SelectAvailable(worker, state, false, false); changed {
		worker = selected.ID
	}
	orchestrator := strings.TrimSpace(request.OrchestratorModel)
	if orchestrator == "" {
		orchestrator = worker
	}
	if selected, changed := state.providers.SelectAvailable(orchestrator, state, true, false); changed {
		orchestrator = selected.ID
	}
	objective := strings.TrimSpace(request.Request) + "\n\nAUTHORIZED PROJECT ROOT: " + projectRoot
	message, err := json.Marshal(map[string]string{"role": "user", "content": objective})
	if err != nil {
		codeError(w, http.StatusInternalServerError, "session_failed", "Coder session could not be initialized.")
		return
	}
	messages := []json.RawMessage{message}
	spec, _, err := state.providers.ValidateChat(worker, request.SystemPrompt, messages, false, false)
	if err != nil {
		codeError(w, http.StatusBadRequest, "worker_model_invalid", "Selected worker model is unavailable.")
		return
	}
	if _, supportsTools, validationErr := state.providers.ValidateChat(orchestrator, request.SystemPrompt, messages, true, false); validationErr != nil || !supportsTools {
		codeError(w, http.StatusBadRequest, "orchestrator_model_invalid", "Selected orchestrator model is not verified for tool execution.")
		return
	}
	session, err := coderSessions.Create(projectRoot, strings.TrimSpace(request.Request), worker, orchestrator)
	if err != nil {
		codeError(w, http.StatusBadRequest, "session_failed", "Coder session baseline could not be created.")
		return
	}
	budget := request.CostBudgetUSD
	if budget <= 0 {
		budget = spec.DefaultRunBudget
	}
	maxTurns := request.MaxTurns
	if maxTurns < 4 {
		maxTurns = 16
	}
	run, err := state.runs.Create(agent.CreateRunRequest{Objective: objective, ProfileID: "developer", ModelID: worker, OrchestratorModelID: orchestrator, ReasoningEffort: request.ReasoningEffort, Mode: "vgt_code", SystemPrompt: request.SystemPrompt, AgentMessages: messages, MaxAgentTurns: maxTurns, CostBudgetUSD: budget, Steps: []agent.RunStep{{Kind: agent.RunStepPlan, Title: "Repository analysieren und Implementierungsplan erstellen"}}})
	if err != nil {
		coderSessions.MarkFailed(session.SessionID, err.Error())
		codeError(w, http.StatusBadRequest, "run_failed", "Coder run could not be created.")
		return
	}
	if _, err = coderSessions.BindRun(session.SessionID, run.ID); err != nil {
		coderSessions.MarkFailed(session.SessionID, err.Error())
		codeError(w, http.StatusInternalServerError, "session_failed", "Coder run could not be bound to its session.")
		return
	}
	run, err = state.runs.Start(run.ID)
	if err != nil {
		coderSessions.MarkFailed(session.SessionID, err.Error())
		codeError(w, http.StatusConflict, "run_failed", "Coder run could not start.")
		return
	}
	updated, _ := coderSessions.Sync(session.SessionID, run)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(coder.PublicSession(updated))
	go agent.DriveChatAgentRun(run.ID, false)
}

func syncCoderSession(sessionID string) (coder.CoderSession, error) {
	session, ok := coderSessions.Get(sessionID)
	if !ok {
		return coder.CoderSession{}, errors.New("coder session not found")
	}
	if session.RunID == "" {
		return session, nil
	}
	if (session.Status == coder.StatusCompleted || session.Status == coder.StatusFailed || session.Status == coder.StatusCancelled) && !session.LastTrackedAt.IsZero() {
		return session, nil
	}
	run, ok := state.runs.Get(session.RunID)
	if !ok {
		return session, nil
	}
	if run.Status == agent.RunRunning && time.Since(run.UpdatedAt) >= 8*time.Minute {
		if stopped, cancelErr := state.runs.Cancel(run.ID); cancelErr == nil {
			run = stopped
		}
	}
	return coderSessions.Sync(sessionID, run)
}

func normalizedCoderScope(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "WORKTREE", "BRANCH":
		return strings.ToUpper(strings.TrimSpace(value))
	default:
		return "AGENT"
	}
}

func sameCanonicalPath(left, right string) bool {
	leftPath, leftErr := security.CanonicalDir(left)
	rightPath, rightErr := security.CanonicalDir(right)
	return leftErr == nil && rightErr == nil && strings.EqualFold(leftPath, rightPath)
}
