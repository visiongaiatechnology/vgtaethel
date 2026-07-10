package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RunStatus is the durable lifecycle of an agent run. A run may only move
// forward, except that an operator may pause/resume it before completion.
type RunStatus string

const (
	RunQueued          RunStatus = "queued"
	RunRunning         RunStatus = "running"
	RunWaitingApproval RunStatus = "waiting_approval"
	RunPaused          RunStatus = "paused"
	RunCancelled       RunStatus = "cancelled"
	RunCompleted       RunStatus = "completed"
	RunFailed          RunStatus = "failed"
)

type RunStepStatus string

const (
	StepPending         RunStepStatus = "pending"
	StepRunning         RunStepStatus = "running"
	StepWaitingApproval RunStepStatus = "waiting_approval"
	StepVerified        RunStepStatus = "verified"
	StepFailed          RunStepStatus = "failed"
	StepSkipped         RunStepStatus = "skipped"
)

type RunStepKind string

const (
	RunStepPlan         RunStepKind = "plan"
	RunStepTool         RunStepKind = "tool"
	RunStepVerification RunStepKind = "verification"
	RunStepReport       RunStepKind = "report"
)

// AgentProfile is an explicit least-privilege profile. Profiles are data, not
// prompt text, so the execution layer can enforce their capability boundary.
type AgentProfile struct {
	ID                  string       `json:"id"`
	Name                string       `json:"name"`
	Description         string       `json:"description"`
	AllowedCapabilities []Capability `json:"allowed_capabilities"`
	MaxSteps            int          `json:"max_steps"`
	MaxToolCalls        int          `json:"max_tool_calls"`
}

type RunStep struct {
	ID               string          `json:"id"`
	Kind             RunStepKind     `json:"kind"`
	Title            string          `json:"title"`
	Status           RunStepStatus   `json:"status"`
	ToolName         string          `json:"tool_name,omitempty"`
	ToolCallID       string          `json:"tool_call_id,omitempty"`
	ToolArgs         json.RawMessage `json:"tool_args,omitempty"`
	ExpectedContains string          `json:"expected_contains,omitempty"`
	ApprovalGranted  bool            `json:"approval_granted,omitempty"`
	Result           string          `json:"result,omitempty"`
	Error            string          `json:"error,omitempty"`
	EvidenceBefore   string          `json:"evidence_before,omitempty"`
	EvidenceAfter    string          `json:"evidence_after,omitempty"`
	EvidenceChanged  bool            `json:"evidence_changed,omitempty"`
	StartedAt        time.Time       `json:"started_at,omitempty"`
	FinishedAt       time.Time       `json:"finished_at,omitempty"`
}

type RunTrace struct {
	Timestamp time.Time `json:"timestamp"`
	Event     string    `json:"event"`
	StepID    string    `json:"step_id,omitempty"`
	Detail    string    `json:"detail"`
}

type AgentRun struct {
	ID             string            `json:"id"`
	Objective      string            `json:"objective"`
	ProfileID      string            `json:"profile_id"`
	ModelID        string            `json:"model_id,omitempty"`
	Mode           string            `json:"mode,omitempty"`
	LiveOperator   bool              `json:"live_operator_active,omitempty"`
	SystemPrompt   string            `json:"system_prompt,omitempty"`
	AgentMessages  []json.RawMessage `json:"agent_messages,omitempty"`
	AgentTurn      int               `json:"agent_turn"`
	MaxAgentTurns  int               `json:"max_agent_turns"`
	FinalReport    string            `json:"final_report,omitempty"`
	Status         RunStatus         `json:"status"`
	Steps          []RunStep         `json:"steps"`
	Trace          []RunTrace        `json:"trace"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	CompletedAt    *time.Time        `json:"completed_at,omitempty"`
	FailureReason  string            `json:"failure_reason,omitempty"`
	ToolCalls      int               `json:"tool_calls"`
	EstimatedCost  float64           `json:"estimated_cost_usd"`
	CostBudgetUSD  float64           `json:"cost_budget_usd"`
	SpentUSD       float64           `json:"spent_usd"`
	ApprovalStepID string            `json:"approval_step_id,omitempty"`
}

type CreateRunRequest struct {
	Objective     string            `json:"objective"`
	ProfileID     string            `json:"profile_id"`
	ModelID       string            `json:"model_id,omitempty"`
	Mode          string            `json:"mode,omitempty"`
	LiveOperator  bool              `json:"live_operator_active,omitempty"`
	SystemPrompt  string            `json:"system_prompt,omitempty"`
	AgentMessages []json.RawMessage `json:"agent_messages,omitempty"`
	MaxAgentTurns int               `json:"max_agent_turns,omitempty"`
	CostBudgetUSD float64           `json:"cost_budget_usd,omitempty"`
	Steps         []RunStep         `json:"steps"`
}

// RunEngine is a process-local scheduler backed by an atomic on-disk journal.
// It intentionally keeps no pointer into a slice after releasing its mutex.
type RunEngine struct {
	mu        sync.RWMutex
	filePath  string
	runs      map[string]AgentRun
	profiles  map[string]AgentProfile
	advancing map[string]bool
}

func NewRunEngine(filePath string) *RunEngine {
	e := &RunEngine{
		filePath:  filePath,
		runs:      make(map[string]AgentRun),
		profiles:  defaultAgentProfiles(),
		advancing: make(map[string]bool),
	}
	_ = e.load()
	return e
}

func defaultAgentProfiles() map[string]AgentProfile {
	profiles := []AgentProfile{
		{ID: "researcher", Name: "Researcher", Description: "Recherche und lokale Analyse ohne Schreib- oder Steuerrechte.", AllowedCapabilities: []Capability{CapFsRead, CapMemoryRead, CapBrowserOpen, CapBrowserRead}, MaxSteps: 16, MaxToolCalls: 12},
		{ID: "developer", Name: "Developer", Description: "Entwicklungsarbeit in explizit freigegebenen Workspaces.", AllowedCapabilities: []Capability{CapFsRead, CapFsWrite, CapFsMount, CapSysExec, CapMemoryRead, CapMemoryWrite, CapTaskCreate, CapTaskSchedule, CapGuiMoveMouse, CapGuiClick, CapGuiType, CapGuiPressKey, CapScreenRead}, MaxSteps: 32, MaxToolCalls: 24},
		{ID: "browser_operator", Name: "Browser Operator", Description: "Sichtbare Browser- und Mediensteuerung mit Operator-Freigaben.", AllowedCapabilities: []Capability{CapBrowserOpen, CapBrowserCtl, CapBrowserRead, CapScreenRead, CapGuiMoveMouse, CapGuiClick, CapGuiType, CapGuiPressKey}, MaxSteps: 24, MaxToolCalls: 18},
		{ID: "personal_assistant", Name: "Personal Assistant", Description: "Lokaler persönlicher Assistent ohne Systemschreibrechte.", AllowedCapabilities: []Capability{CapMemoryRead, CapMemoryWrite, CapFsRead, CapMediaControl, CapTaskCreate}, MaxSteps: 16, MaxToolCalls: 10},
	}
	result := make(map[string]AgentProfile, len(profiles))
	for _, profile := range profiles {
		result[profile.ID] = profile
	}
	return result
}

func (e *RunEngine) load() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	data, _, err := readSealedFile(e.filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var list []AgentRun
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	for _, run := range list {
		if run.ID == "" {
			continue
		}
		// A process cannot safely resume an in-flight syscall after restart.
		// Preserve its trace and make operator intent explicit.
		if run.Status == RunRunning {
			run.Status = RunPaused
			run.Trace = appendRunTrace(run.Trace, "recovery_paused", "Run was paused after application restart.", "")
		}
		e.runs[run.ID] = run
	}
	return e.saveLocked()
}

func (e *RunEngine) saveLocked() error {
	list := make([]AgentRun, 0, len(e.runs))
	for _, run := range e.runs {
		list = append(list, cloneRun(run))
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.After(list[j].CreatedAt) })
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(e.filePath), 0700); err != nil {
		return err
	}
	return writeSealedFile(e.filePath, data)
}

func newRunID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "run_" + hex.EncodeToString(raw[:]), nil
}

func cloneRun(run AgentRun) AgentRun {
	copyRun := run
	copyRun.Steps = append([]RunStep(nil), run.Steps...)
	copyRun.Trace = append([]RunTrace(nil), run.Trace...)
	for i := range copyRun.Steps {
		copyRun.Steps[i].ToolArgs = append(json.RawMessage(nil), run.Steps[i].ToolArgs...)
	}
	copyRun.AgentMessages = make([]json.RawMessage, len(run.AgentMessages))
	for i := range run.AgentMessages {
		copyRun.AgentMessages[i] = append(json.RawMessage(nil), run.AgentMessages[i]...)
	}
	return copyRun
}

func appendRunTrace(trace []RunTrace, event, detail, stepID string) []RunTrace {
	trace = append(trace, RunTrace{Timestamp: time.Now().UTC(), Event: event, Detail: clampRunDetail(detail), StepID: stepID})
	if len(trace) > 300 {
		return trace[len(trace)-300:]
	}
	return trace
}

func clampRunDetail(detail string) string {
	detail = strings.TrimSpace(detail)
	if len([]rune(detail)) > 6000 {
		return string([]rune(detail)[:6000]) + "…"
	}
	return detail
}

func (e *RunEngine) Profiles() []AgentProfile {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]AgentProfile, 0, len(e.profiles))
	for _, profile := range e.profiles {
		profile.AllowedCapabilities = append([]Capability(nil), profile.AllowedCapabilities...)
		result = append(result, profile)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (e *RunEngine) Create(input CreateRunRequest) (AgentRun, error) {
	objective := strings.TrimSpace(input.Objective)
	if len([]rune(objective)) < 3 || len([]rune(objective)) > 8000 {
		return AgentRun{}, errors.New("objective must contain between 3 and 8000 characters")
	}
	profileID := strings.TrimSpace(input.ProfileID)
	if profileID == "" {
		profileID = "developer"
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	profile, ok := e.profiles[profileID]
	if !ok {
		return AgentRun{}, errors.New("unknown agent profile")
	}
	if len(input.Steps) == 0 {
		input.Steps = []RunStep{
			{Kind: RunStepPlan, Title: "Arbeitsplan bestätigen"},
			{Kind: RunStepReport, Title: "Ergebnis berichten"},
		}
	}
	if len(input.Steps) > profile.MaxSteps {
		return AgentRun{}, fmt.Errorf("profile allows at most %d steps", profile.MaxSteps)
	}
	if input.CostBudgetUSD <= 0 {
		input.CostBudgetUSD = 2
	}
	if input.CostBudgetUSD > 1000 {
		return AgentRun{}, errors.New("run cost budget exceeds operator safety limit")
	}
	for index := range input.Steps {
		step := &input.Steps[index]
		if step.Kind != RunStepPlan && step.Kind != RunStepTool && step.Kind != RunStepVerification && step.Kind != RunStepReport {
			return AgentRun{}, errors.New("invalid run step kind")
		}
		if strings.TrimSpace(step.Title) == "" || len([]rune(step.Title)) > 300 {
			return AgentRun{}, errors.New("every run step needs a title up to 300 characters")
		}
		if step.Kind == RunStepTool {
			if strings.TrimSpace(step.ToolName) == "" || len(step.ToolArgs) == 0 || !json.Valid(step.ToolArgs) {
				return AgentRun{}, errors.New("tool steps need a valid tool name and JSON arguments")
			}
		}
		step.ID = fmt.Sprintf("step_%02d", index+1)
		step.Status = StepPending
	}
	id, err := newRunID()
	if err != nil {
		return AgentRun{}, err
	}
	now := time.Now().UTC()
	maxTurns := input.MaxAgentTurns
	if maxTurns <= 0 {
		maxTurns = 24
	}
	if maxTurns > 128 {
		return AgentRun{}, errors.New("agent turn limit exceeds safety boundary")
	}
	runMode := strings.TrimSpace(input.Mode)
	if runMode == "" {
		runMode = "chat_agent"
	}
	run := AgentRun{ID: id, Objective: objective, ProfileID: profileID, ModelID: strings.TrimSpace(input.ModelID), Mode: runMode, LiveOperator: input.LiveOperator, SystemPrompt: input.SystemPrompt, AgentMessages: append([]json.RawMessage(nil), input.AgentMessages...), MaxAgentTurns: maxTurns, Status: RunQueued, Steps: input.Steps, CreatedAt: now, UpdatedAt: now, CostBudgetUSD: input.CostBudgetUSD}
	run.Trace = appendRunTrace(run.Trace, "created", "Persistent agent run created.", "")
	e.runs[run.ID] = run
	if err := e.saveLocked(); err != nil {
		delete(e.runs, run.ID)
		return AgentRun{}, err
	}
	return cloneRun(run), nil
}

func (e *RunEngine) List() []AgentRun {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]AgentRun, 0, len(e.runs))
	for _, run := range e.runs {
		result = append(result, cloneRun(run))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.After(result[j].UpdatedAt) })
	return result
}

func (e *RunEngine) RecordCost(id string, costUSD float64) (AgentRun, error) {
	if costUSD < 0 {
		return AgentRun{}, errors.New("negative run cost is invalid")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	run, ok := e.runs[id]
	if !ok {
		return AgentRun{}, errors.New("run not found")
	}
	if run.Status == RunCompleted || run.Status == RunCancelled || run.Status == RunFailed {
		return AgentRun{}, errors.New("terminal run cannot accept cost")
	}
	run.SpentUSD += costUSD
	run.EstimatedCost = run.SpentUSD
	if run.SpentUSD > run.CostBudgetUSD {
		run.Status = RunPaused
		run.Trace = appendRunTrace(run.Trace, "budget_paused", "Run exceeded its operator-defined cost budget.", "")
	}
	run.UpdatedAt = time.Now().UTC()
	e.runs[id] = run
	if err := e.saveLocked(); err != nil {
		return AgentRun{}, err
	}
	return cloneRun(run), nil
}

func (e *RunEngine) AppendAgentToolSteps(id string, calls []AgentToolCall) (AgentRun, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	run, ok := e.runs[id]
	if !ok {
		return AgentRun{}, errors.New("run not found")
	}
	if run.Mode != "chat_agent" || run.Status != RunRunning {
		return AgentRun{}, errors.New("chat agent run is not active")
	}
	profile := e.profiles[run.ProfileID]
	if len(run.Steps)+len(calls) > profile.MaxSteps {
		return AgentRun{}, errors.New("agent profile step limit reached")
	}
	for _, call := range calls {
		if call.ID == "" || call.Name == "" || len(call.Arguments) == 0 || !json.Valid(call.Arguments) {
			return AgentRun{}, errors.New("invalid agent tool call")
		}
		step := RunStep{ID: fmt.Sprintf("step_%02d", len(run.Steps)+1), Kind: RunStepTool, Title: "Tool: " + call.Name, Status: StepPending, ToolName: call.Name, ToolCallID: call.ID, ToolArgs: append(json.RawMessage(nil), call.Arguments...)}
		run.Steps = append(run.Steps, step)
		run.Trace = appendRunTrace(run.Trace, "tool_planned", call.Name, step.ID)
	}
	run.UpdatedAt = time.Now().UTC()
	e.runs[id] = run
	if err := e.saveLocked(); err != nil {
		return AgentRun{}, err
	}
	return cloneRun(run), nil
}

func (e *RunEngine) UpdateAgentProgress(id string, messages []json.RawMessage, turn int, assistantText string) (AgentRun, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	run, ok := e.runs[id]
	if !ok {
		return AgentRun{}, errors.New("run not found")
	}
	if run.Mode != "chat_agent" {
		return AgentRun{}, errors.New("run is not a chat agent")
	}
	run.AgentMessages = make([]json.RawMessage, len(messages))
	for index := range messages {
		run.AgentMessages[index] = append(json.RawMessage(nil), messages[index]...)
	}
	run.AgentTurn = turn
	if strings.TrimSpace(assistantText) != "" {
		run.FinalReport = clampRunDetail(assistantText)
	}
	run.UpdatedAt = time.Now().UTC()
	run.Trace = appendRunTrace(run.Trace, "model_turn", fmt.Sprintf("Agent model turn %d completed.", turn), "")
	e.runs[id] = run
	if err := e.saveLocked(); err != nil {
		return AgentRun{}, err
	}
	return cloneRun(run), nil
}

func (e *RunEngine) CompleteAgent(id, report string, messages []json.RawMessage) (AgentRun, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	run, ok := e.runs[id]
	if !ok {
		return AgentRun{}, errors.New("run not found")
	}
	if run.Status != RunRunning {
		return AgentRun{}, errors.New("run is not active")
	}
	now := time.Now().UTC()
	run.Status = RunCompleted
	run.CompletedAt = &now
	run.UpdatedAt = now
	run.FinalReport = clampRunDetail(report)
	run.AgentMessages = make([]json.RawMessage, len(messages))
	for index := range messages {
		run.AgentMessages[index] = append(json.RawMessage(nil), messages[index]...)
	}
	run.Steps = append(run.Steps, RunStep{ID: fmt.Sprintf("step_%02d", len(run.Steps)+1), Kind: RunStepReport, Title: "Verifizierter Abschlussbericht", Status: StepVerified, Result: run.FinalReport, StartedAt: now, FinishedAt: now})
	run.Trace = appendRunTrace(run.Trace, "completed", "Agent delivered final report.", "")
	e.runs[id] = run
	if err := e.saveLocked(); err != nil {
		return AgentRun{}, err
	}
	return cloneRun(run), nil
}

func (e *RunEngine) FailAgent(id, reason string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	run, ok := e.runs[id]
	if !ok || run.Status == RunCompleted || run.Status == RunCancelled {
		return
	}
	now := time.Now().UTC()
	run.Status = RunFailed
	run.FailureReason = clampRunDetail(reason)
	run.CompletedAt = &now
	run.UpdatedAt = now
	run.Trace = appendRunTrace(run.Trace, "failed", run.FailureReason, "")
	e.runs[id] = run
	_ = e.saveLocked()
}

// PauseOnTurnLimit suspends a run that exhausted its agent turn budget without
// failing it permanently. The operator can review the trace and resume.
func (e *RunEngine) PauseOnTurnLimit(id string, reachedTurn int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	run, ok := e.runs[id]
	if !ok || run.Status != RunRunning {
		return
	}
	run.Status = RunPaused
	run.UpdatedAt = time.Now().UTC()
	run.Trace = appendRunTrace(run.Trace, "turn_limit_paused",
		fmt.Sprintf("Agent reached the %d-turn budget. Resume to continue or cancel to discard.", reachedTurn), "")
	e.runs[id] = run
	_ = e.saveLocked()
}

func (e *RunEngine) Get(id string) (AgentRun, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	run, ok := e.runs[id]
	return cloneRun(run), ok
}

func (e *RunEngine) PendingApproval(id string) (RunStep, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	run, ok := e.runs[id]
	if !ok || run.Status != RunWaitingApproval || run.ApprovalStepID == "" {
		return RunStep{}, false
	}
	for _, step := range run.Steps {
		if step.ID == run.ApprovalStepID && step.Status == StepWaitingApproval {
			return step, true
		}
	}
	return RunStep{}, false
}

func (e *RunEngine) Approve(id, token string, approvals *ApprovalManager, policy *PolicyEngine) (AgentRun, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	run, ok := e.runs[id]
	if !ok {
		return AgentRun{}, errors.New("run not found")
	}
	if run.Status != RunWaitingApproval || run.ApprovalStepID == "" {
		return AgentRun{}, errors.New("run is not waiting for approval")
	}
	stepIndex := -1
	for index := range run.Steps {
		if run.Steps[index].ID == run.ApprovalStepID && run.Steps[index].Status == StepWaitingApproval {
			stepIndex = index
			break
		}
	}
	if stepIndex < 0 {
		return AgentRun{}, errors.New("approval step not found")
	}
	step := &run.Steps[stepIndex]
	allowed, decision, report := policy.Evaluate(step.ToolName, string(step.ToolArgs), false)
	if allowed || decision != "needs_approval" {
		return AgentRun{}, errors.New("approval target policy state changed")
	}
	if err := approvals.Consume(token, step.ToolName, string(step.ToolArgs), report.Capability, id); err != nil {
		return AgentRun{}, err
	}
	step.Status = StepPending
	step.ApprovalGranted = true
	step.StartedAt = time.Time{}
	step.FinishedAt = time.Time{}
	run.Status = RunRunning
	run.ApprovalStepID = ""
	run.UpdatedAt = time.Now().UTC()
	run.Trace = appendRunTrace(run.Trace, "approval_consumed", "Operator approved exact pending action once.", step.ID)
	e.runs[id] = run
	if err := e.saveLocked(); err != nil {
		return AgentRun{}, err
	}
	return cloneRun(run), nil
}

func (e *RunEngine) transition(id string, allowed map[RunStatus]bool, next RunStatus, event, detail string) (AgentRun, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	run, ok := e.runs[id]
	if !ok {
		return AgentRun{}, errors.New("run not found")
	}
	if !allowed[run.Status] {
		return AgentRun{}, fmt.Errorf("run cannot transition from %s to %s", run.Status, next)
	}
	run.Status = next
	run.UpdatedAt = time.Now().UTC()
	run.Trace = appendRunTrace(run.Trace, event, detail, "")
	if next == RunCompleted || next == RunCancelled || next == RunFailed {
		completed := run.UpdatedAt
		run.CompletedAt = &completed
	}
	e.runs[id] = run
	if err := e.saveLocked(); err != nil {
		return AgentRun{}, err
	}
	return cloneRun(run), nil
}

func (e *RunEngine) Start(id string) (AgentRun, error) {
	return e.transition(id, map[RunStatus]bool{RunQueued: true, RunPaused: true}, RunRunning, "started", "Operator started or resumed run.")
}

func (e *RunEngine) Pause(id string) (AgentRun, error) {
	return e.transition(id, map[RunStatus]bool{RunQueued: true, RunRunning: true, RunWaitingApproval: true}, RunPaused, "paused", "Operator paused run.")
}

func (e *RunEngine) Cancel(id string) (AgentRun, error) {
	return e.transition(id, map[RunStatus]bool{RunQueued: true, RunRunning: true, RunWaitingApproval: true, RunPaused: true}, RunCancelled, "cancelled", "Operator cancelled run.")
}

// Advance executes at most one durable step. The caller can safely invoke it
// repeatedly; pause, cancellation and approval always take effect between steps.
func (e *RunEngine) Advance(id string, policy *PolicyEngine, registry *SkillRegistry) (AgentRun, error) {
	e.mu.Lock()
	run, ok := e.runs[id]
	if !ok {
		e.mu.Unlock()
		return AgentRun{}, errors.New("run not found")
	}
	if run.Status != RunRunning {
		e.mu.Unlock()
		return cloneRun(run), fmt.Errorf("run is not active: %s", run.Status)
	}
	if e.advancing[id] {
		e.mu.Unlock()
		return cloneRun(run), errors.New("run step is already advancing")
	}
	profile := e.profiles[run.ProfileID]
	nextIndex := -1
	for i := range run.Steps {
		if run.Steps[i].Status == StepPending || run.Steps[i].Status == StepWaitingApproval {
			nextIndex = i
			break
		}
	}
	if nextIndex == -1 {
		now := time.Now().UTC()
		run.Status = RunCompleted
		run.CompletedAt = &now
		run.UpdatedAt = now
		run.Trace = appendRunTrace(run.Trace, "completed", "All steps verified.", "")
		e.runs[id] = run
		err := e.saveLocked()
		e.mu.Unlock()
		return cloneRun(run), err
	}
	step := run.Steps[nextIndex]
	if step.Status == StepWaitingApproval {
		run.Status = RunWaitingApproval
		run.ApprovalStepID = step.ID
		run.UpdatedAt = time.Now().UTC()
		run.Trace = appendRunTrace(run.Trace, "waiting_approval", "Tool step requires a signed operator approval.", step.ID)
		e.runs[id] = run
		err := e.saveLocked()
		e.mu.Unlock()
		return cloneRun(run), err
	}
	step.Status = StepRunning
	step.StartedAt = time.Now().UTC()
	run.Steps[nextIndex] = step
	run.UpdatedAt = step.StartedAt
	run.Trace = appendRunTrace(run.Trace, "step_started", step.Title, step.ID)
	e.runs[id] = run
	e.advancing[id] = true
	if err := e.saveLocked(); err != nil {
		delete(e.advancing, id)
		e.mu.Unlock()
		return AgentRun{}, err
	}
	e.mu.Unlock()

	// Execute outside the run lock. The final update re-reads the run so a
	// concurrent pause/cancel cannot be overwritten by a stale snapshot.
	evidenceBefore := desktopEvidenceForStep(step)
	result, waitingApproval, executionErr := executeRunStep(step, profile, policy, registry)
	evidenceAfter := ""
	if !waitingApproval {
		evidenceAfter = desktopEvidenceForStep(step)
	} else {
		evidenceBefore = ""
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.advancing, id)
	run, ok = e.runs[id]
	if !ok {
		return AgentRun{}, errors.New("run removed during execution")
	}
	current := &run.Steps[nextIndex]
	if run.Status == RunCancelled || run.Status == RunPaused {
		current.Status = StepPending
		current.StartedAt = time.Time{}
		run.Trace = appendRunTrace(run.Trace, "step_deferred", "Step result discarded because operator changed run state.", current.ID)
		run.UpdatedAt = time.Now().UTC()
		e.runs[id] = run
		err := e.saveLocked()
		return cloneRun(run), err
	}
	current.FinishedAt = time.Now().UTC()
	current.EvidenceBefore = evidenceBefore
	current.EvidenceAfter = evidenceAfter
	current.EvidenceChanged = evidenceBefore != "" && evidenceAfter != "" && evidenceBefore != evidenceAfter
	if waitingApproval {
		current.Status = StepWaitingApproval
		current.Result = ""
		run.Status = RunWaitingApproval
		run.ApprovalStepID = current.ID
		run.Trace = appendRunTrace(run.Trace, "approval_requested", result, current.ID)
	} else if executionErr != nil && isRecoverableReadScopeError(current, executionErr) {
		// A read outside the jail is not an execution failure: the agent needs
		// the real, bounded result in its context so it can request a read mount
		// and continue. Writes and every other failure remain fail-closed.
		current.Status = StepVerified
		current.Error = clampRunDetail(executionErr.Error())
		current.Result = clampRunDetail("TOOL_ERROR: " + current.Error + "\nRECOVERY: Request fs_mount_folder with access 'read' for this exact existing folder, then retry the original source inspection tool. Do not attempt a write operation outside the VGT workspace.")
		run.ToolCalls++
		run.Trace = appendRunTrace(run.Trace, "tool_recoverable_scope_error", current.Error, current.ID)
	} else if executionErr != nil {
		current.Status = StepFailed
		current.Error = clampRunDetail(executionErr.Error())
		run.Status = RunFailed
		run.FailureReason = current.Error
		completed := current.FinishedAt
		run.CompletedAt = &completed
		run.Trace = appendRunTrace(run.Trace, "step_failed", current.Error, current.ID)
	} else {
		current.Status = StepVerified
		current.Result = clampRunDetail(result)
		if current.Kind == RunStepTool {
			run.ToolCalls++
		}
		run.Trace = appendRunTrace(run.Trace, "step_verified", current.Title, current.ID)
	}
	run.UpdatedAt = time.Now().UTC()
	e.runs[id] = run
	if err := e.saveLocked(); err != nil {
		return AgentRun{}, err
	}
	return cloneRun(run), nil
}

// isRecoverableReadScopeError deliberately recognizes only source inspection
// tools at the jail boundary. It never turns a write, command, parser, or
// permission failure into an agent retry loop.
func isRecoverableReadScopeError(step *RunStep, err error) bool {
	if step == nil || err == nil || (step.ToolName != "fs_list_dir" && step.ToolName != "fs_read_file" && step.ToolName != "code_cartography") {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "path escaped the configured Windows workspace jail") ||
		strings.Contains(message, "Path is not inside the workspace or any mounted directory")
}

func executeRunStep(step RunStep, profile AgentProfile, policy *PolicyEngine, registry *SkillRegistry) (result string, waitingApproval bool, err error) {
	switch step.Kind {
	case RunStepPlan:
		return "Plan is persisted and ready for controlled execution.", false, nil
	case RunStepReport:
		return "Run report assembled from verified trace.", false, nil
	case RunStepVerification:
		return "Verification checkpoint completed.", false, nil
	case RunStepTool:
		allowed, decision, report := policy.Evaluate(step.ToolName, string(step.ToolArgs), false)
		if !profileAllows(profile, report.Capability) {
			return "", false, fmt.Errorf("profile %s is not allowed to use capability %s", profile.ID, report.Capability)
		}
		if !allowed && !(step.ApprovalGranted && decision == "needs_approval") {
			if decision == "needs_approval" {
				return "Operator approval required for " + string(report.Capability) + ".", true, nil
			}
			return "", false, errors.New("policy blocked tool execution")
		}
		skill, exists := registry.Get(step.ToolName)
		if !exists {
			return "", false, errors.New("tool is no longer registered")
		}
		toolResult, toolErr := skill.Execute(step.ToolArgs)
		return toolResult, false, toolErr
	default:
		return "", false, errors.New("unsupported step kind")
	}
}

func profileAllows(profile AgentProfile, capability Capability) bool {
	// A zero capability means the scanner has no specific mapping for this tool.
	// The PolicyEngine already gates such tools via the RiskLevel path; there is
	// no additional profile-level capability boundary to enforce.
	if capability == "" {
		return true
	}
	for _, allowed := range profile.AllowedCapabilities {
		if allowed == capability {
			return true
		}
	}
	return false
}

func desktopEvidenceForStep(step RunStep) string {
	if step.Kind != RunStepTool || (step.ToolName != "gui_control" && step.ToolName != "gui_window_control") {
		return ""
	}
	if err := CapturePrimaryDisplay(); err != nil {
		return "unavailable"
	}
	data, err := GetLatestScreenshot()
	if err != nil || len(data) == 0 {
		return "unavailable"
	}
	digest := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", digest)
}
