// STATUS: DIAMANT VGT SUPREME
package coder

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"go-aethel/agent"
	"go-aethel/security"
)

type Manager struct {
	mu       sync.RWMutex
	filePath string
	sessions map[string]CoderSession
	tracker  RepositoryTracker
}

func NewManager(filePath string) *Manager {
	manager := &Manager{filePath: filePath, sessions: map[string]CoderSession{}, tracker: RepositoryTracker{}}
	_ = manager.load()
	return manager
}

func (manager *Manager) Create(projectRoot, request, workerModel, orchestratorModel string) (CoderSession, error) {
	request = strings.TrimSpace(request)
	if len([]rune(request)) < 3 || len([]rune(request)) > 8000 {
		return CoderSession{}, errors.New("coder request must contain between 3 and 8000 characters")
	}
	repositoryRoot, branch, mode := manager.tracker.InspectRoot(projectRoot)
	baselineChanges, baselineFiles, err := manager.tracker.CaptureBaseline(repositoryRoot, mode)
	if err != nil {
		return CoderSession{}, errors.New("coder baseline could not be captured")
	}
	id, err := newSessionID()
	if err != nil {
		return CoderSession{}, err
	}
	now := time.Now().UTC()
	session := CoderSession{
		SessionID: id, ProjectRoot: projectRoot, RepositoryRoot: repositoryRoot, ProjectName: filepathBase(projectRoot), Branch: branch,
		BaseBranch: detectBaseBranch(repositoryRoot), DiffMode: mode, WorkerModel: workerModel, OrchestratorModel: orchestratorModel,
		StartedAt: now, Status: StatusCreated, UserRequest: request, Events: []CoderEvent{}, ChangedFiles: []CoderChangedFile{}, WorktreeFiles: []CoderChangedFile{}, BranchFiles: []CoderChangedFile{},
		ValidationResults: []CoderValidationResult{}, Warnings: []string{}, BaselineChanges: baselineChanges, BaselineFiles: baselineFiles,
	}
	if mode != "GIT" {
		session.Warnings = append(session.Warnings, "DIFF MODE // FILESYSTEM FALLBACK")
	}
	session.Events = append(session.Events, CoderEvent{ID: id + "-event-1", SessionID: id, Timestamp: now, Type: "PLAN", Status: EventQueued, Title: "Coding session created", Summary: "Project boundary and baseline captured."})
	manager.mu.Lock()
	manager.sessions[id] = session
	err = manager.saveLocked()
	manager.mu.Unlock()
	return cloneSession(session), err
}

func (manager *Manager) BindRun(sessionID, runID string) (CoderSession, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	session, ok := manager.sessions[sessionID]
	if !ok {
		return CoderSession{}, errors.New("coder session not found")
	}
	session.RunID = runID
	session.Status = StatusPlanning
	manager.sessions[sessionID] = session
	if err := manager.saveLocked(); err != nil {
		return CoderSession{}, err
	}
	return cloneSession(session), nil
}

func (manager *Manager) Sync(sessionID string, run agent.AgentRun) (CoderSession, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	session, ok := manager.sessions[sessionID]
	if !ok {
		return CoderSession{}, errors.New("coder session not found")
	}
	session.Status = mapRunStatus(run)
	session.LastActivityAt = run.UpdatedAt.UTC()
	session.CanCancel = !terminalStatus(session.Status)
	session.Health, session.HealthMessage = healthForRun(session, run)
	session.AgentSummary = run.FinalReport
	if run.CompletedAt != nil {
		finished := run.CompletedAt.UTC()
		session.FinishedAt = &finished
	}
	session.DurationMS = durationMilliseconds(session.StartedAt, session.FinishedAt)
	session.Events = eventsFromRun(session.SessionID, run)
	session.ValidationResults = validationsFromRun(run)
	shouldTrack := false
	if !terminalStatus(session.Status) {
		shouldTrack = time.Since(session.LastTrackedAt) >= 300*time.Millisecond
	} else if session.LastTrackedAt.IsZero() || (session.FinishedAt != nil && session.LastTrackedAt.Before(*session.FinishedAt)) {
		shouldTrack = true
	}
	if shouldTrack {
		worktree, worktreeErr := manager.tracker.Changes(session, "WORKTREE")
		agentFiles, agentErr := manager.tracker.Changes(session, "AGENT")
		branchFiles, branchErr := manager.tracker.Changes(session, "BRANCH")
		if worktreeErr == nil {
			session.WorktreeFiles = worktree
		}
		if agentErr == nil {
			session.ChangedFiles = agentFiles
		}
		if branchErr == nil {
			session.BranchFiles = branchFiles
		}
		if worktreeErr != nil || agentErr != nil {
			session.Warnings = appendUnique(session.Warnings, "Repository refresh temporarily unavailable.")
		}
		session.LastTrackedAt = time.Now().UTC()
	}
	session.Stats = statsForSession(session, run)
	manager.sessions[sessionID] = session
	if err := manager.saveLocked(); err != nil {
		return CoderSession{}, err
	}
	return cloneSession(session), nil
}

func healthForRun(session CoderSession, run agent.AgentRun) (string, string) {
	if terminalStatus(session.Status) {
		return "TERMINAL", string(session.Status)
	}
	age := time.Since(run.UpdatedAt)
	if age >= 90*time.Second {
		return "STALLED", fmt.Sprintf("KEIN HEARTBEAT · %ds · WATCHDOG AKTIV", int(age.Seconds()))
	}
	if age >= 30*time.Second {
		return "QUIET", fmt.Sprintf("MODELL ARBEITET · %ds", int(age.Seconds()))
	}
	return "HEALTHY", "LIVE · HEARTBEAT OK"
}

func (manager *Manager) Get(sessionID string) (CoderSession, bool) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	session, ok := manager.sessions[sessionID]
	return cloneSession(session), ok
}

func (manager *Manager) List(limit int) []CoderSession {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	if limit < 1 || limit > 200 {
		limit = 50
	}
	result := make([]CoderSession, 0, len(manager.sessions))
	for _, session := range manager.sessions {
		result = append(result, PublicSession(session))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].StartedAt.After(result[j].StartedAt) })
	if len(result) > limit {
		result = result[:limit]
	}
	return result
}

// PublicSession removes internal repository baselines before transport to the UI.
func PublicSession(session CoderSession) CoderSession {
	result := cloneSession(session)
	result.BaselineChanges = nil
	result.BaselineFiles = nil
	return result
}

func (manager *Manager) Diff(sessionID, path, scope string) (CoderDiff, error) {
	session, ok := manager.Get(sessionID)
	if !ok {
		return CoderDiff{}, errors.New("coder session not found")
	}
	return manager.tracker.Diff(session, path, scope)
}

func (manager *Manager) MarkFailed(sessionID, reason string) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	session, ok := manager.sessions[sessionID]
	if !ok {
		return
	}
	now := time.Now().UTC()
	session.Status = StatusFailed
	session.FinishedAt = &now
	session.DurationMS = durationMilliseconds(session.StartedAt, session.FinishedAt)
	session.Warnings = appendUnique(session.Warnings, clamp(reason, 500))
	manager.sessions[sessionID] = session
	_ = manager.saveLocked()
}

func (manager *Manager) load() error {
	data, _, err := security.ReadSealedFile(manager.filePath)
	if errors.Is(err, osErrNotExist()) {
		return nil
	}
	if err != nil {
		return err
	}
	var sessions []CoderSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		return err
	}
	for _, session := range sessions {
		if session.SessionID != "" {
			manager.sessions[session.SessionID] = session
		}
	}
	return nil
}

func (manager *Manager) saveLocked() error {
	sessions := make([]CoderSession, 0, len(manager.sessions))
	for _, session := range manager.sessions {
		sessions = append(sessions, session)
	}
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].StartedAt.After(sessions[j].StartedAt) })
	data, err := json.Marshal(sessions)
	if err != nil {
		return err
	}
	return security.WriteSealedFile(manager.filePath, data)
}

func eventsFromRun(sessionID string, run agent.AgentRun) []CoderEvent {
	result := make([]CoderEvent, 0, len(run.Trace))
	steps := make(map[string]agent.RunStep, len(run.Steps))
	for _, step := range run.Steps {
		steps[step.ID] = step
	}
	for _, trace := range run.Trace {
		step := steps[trace.StepID]
		eventType := eventTypeForTrace(trace.Event, step.ToolName)
		status := eventStatusForTrace(trace.Event)
		if step.ID != "" {
			switch step.Status {
			case agent.StepVerified:
				status = EventSuccess
			case agent.StepFailed:
				status = EventFailed
			case agent.StepRunning:
				status = EventRunning
			case agent.StepWaitingApproval:
				status = EventQueued
			case agent.StepPending:
				status = EventQueued
			}
		}
		if trace.Event == "model_turn" {
			for index := len(result) - 1; index >= 0; index-- {
				if result[index].Type == "ANALYSIS" && result[index].Status == EventRunning {
					result[index].Status = EventSuccess
					break
				}
			}
		}
		event := CoderEvent{ID: fmt.Sprintf("%s-event-%d", sessionID, trace.Sequence), SessionID: sessionID, Timestamp: trace.Timestamp, Type: eventType, Status: status, Title: eventTitle(trace.Event, step), Summary: redactSecrets(trace.Detail)}
		if step.ToolName == "fs_read_file" || step.ToolName == "fs_write_file" || step.ToolName == "fs_replace_file_content" {
			event.Path = extractJSONText(step.ToolArgs, "path")
		}
		if step.ToolName == "sys_exec_cmd" {
			event.Command = redactedCommand(step.ToolArgs)
		}
		if !step.StartedAt.IsZero() && !step.FinishedAt.IsZero() {
			event.DurationMS = step.FinishedAt.Sub(step.StartedAt).Milliseconds()
		}
		result = append(result, event)
	}
	return result
}

func validationsFromRun(run agent.AgentRun) []CoderValidationResult {
	result := []CoderValidationResult{}
	for _, step := range run.Steps {
		if step.ToolName != "sys_exec_cmd" {
			continue
		}
		command := redactedCommand(step.ToolArgs)
		lower := strings.ToLower(command)
		name := "Command"
		if strings.Contains(lower, "test") {
			name = "Tests"
		} else if strings.Contains(lower, "lint") || strings.Contains(lower, "vet") {
			name = "Lint"
		} else if strings.Contains(lower, "build") {
			name = "Build"
		} else {
			continue
		}
		status := "QUEUED"
		failed, passed := 0, 0
		if step.Status == agent.StepVerified {
			status, passed = "PASS", 1
		} else if step.Status == agent.StepFailed {
			status, failed = "FAIL", 1
		} else if step.Status == agent.StepRunning {
			status = "RUNNING"
		}
		duration := int64(0)
		if !step.StartedAt.IsZero() && !step.FinishedAt.IsZero() {
			duration = step.FinishedAt.Sub(step.StartedAt).Milliseconds()
		}
		result = append(result, CoderValidationResult{ID: step.ID, Name: name, Command: command, Status: status, Passed: passed, Failed: failed, DurationMS: duration, Output: redactSecrets(clamp(firstNonEmpty(step.Result, step.Error), 2000))})
	}
	return result
}

func statsForSession(session CoderSession, run agent.AgentRun) CoderStats {
	stats := CoderStats{FilesChanged: len(session.ChangedFiles), DurationMS: session.DurationMS}
	for _, file := range session.ChangedFiles {
		stats.LinesAdded += file.Additions
		stats.LinesDeleted += file.Deletions
		switch file.Status {
		case "ADDED":
			stats.FilesAdded++
		case "DELETED":
			stats.FilesDeleted++
		default:
			stats.FilesModified++
		}
	}
	for _, step := range run.Steps {
		if step.ToolName == "sys_exec_cmd" {
			stats.CommandsRun++
		}
	}
	for _, validation := range session.ValidationResults {
		stats.TestsPassed += validation.Passed
		stats.TestsFailed += validation.Failed
	}
	return stats
}

func mapRunStatus(run agent.AgentRun) SessionStatus {
	switch run.Status {
	case agent.RunQueued:
		return StatusPlanning
	case agent.RunRunning:
		if run.AgentTurn == 0 {
			return StatusPlanning
		}
		return StatusRunning
	case agent.RunWaitingApproval, agent.RunPaused:
		return StatusWaitingApproval
	case agent.RunCompleted:
		return StatusCompleted
	case agent.RunCancelled:
		return StatusCancelled
	case agent.RunFailed:
		return StatusFailed
	default:
		return StatusCreated
	}
}

func eventTypeForTrace(event, tool string) string {
	switch tool {
	case "fs_read_file", "fs_list_dir":
		return "READ_FILE"
	case "fs_write_file":
		return "CREATE_FILE"
	case "fs_replace_file_content":
		return "EDIT_FILE"
	case "sys_exec_cmd":
		return "COMMAND"
	}
	switch {
	case event == "model_started":
		return "ANALYSIS"
	case event == "model_turn":
		return "DECISION"
	case strings.Contains(event, "approval"):
		return "APPROVAL"
	case strings.Contains(event, "failed"), strings.Contains(event, "error"):
		return "ERROR"
	case event == "completed":
		return "FINAL"
	case event == "created", strings.Contains(event, "planned"):
		return "PLAN"
	case strings.Contains(event, "verified"):
		return "CHECKPOINT"
	default:
		return "SEARCH"
	}
}

func eventStatusForTrace(event string) EventStatus {
	switch {
	case strings.Contains(event, "failed"), strings.Contains(event, "error"):
		return EventFailed
	case strings.Contains(event, "cancel"):
		return EventCancelled
	case strings.Contains(event, "started"), strings.Contains(event, "running"):
		return EventRunning
	case strings.Contains(event, "planned"), strings.Contains(event, "created"):
		return EventQueued
	default:
		return EventSuccess
	}
}

func eventTitle(event string, step agent.RunStep) string {
	if step.Title != "" {
		return step.Title
	}
	return strings.ReplaceAll(strings.Title(strings.ReplaceAll(event, "_", " ")), "  ", " ") //nolint:staticcheck -- stable presentation only.
}

func redactedCommand(raw json.RawMessage) string {
	var input struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if json.Unmarshal(raw, &input) != nil {
		return ""
	}
	return redactSecrets(strings.Join(append([]string{input.Command}, input.Args...), " "))
}

func extractJSONText(raw json.RawMessage, key string) string {
	var input map[string]interface{}
	if json.Unmarshal(raw, &input) != nil {
		return ""
	}
	value, _ := input[key].(string)
	return value
}

func redactSecrets(value string) string {
	patterns := []string{"authorization:", "api_key", "apikey", "password", "private key", "token="}
	lower := strings.ToLower(value)
	for _, pattern := range patterns {
		if index := strings.Index(lower, pattern); index >= 0 {
			end := strings.IndexAny(value[index:], " \r\n\t")
			if end < 0 {
				end = len(value) - index
			}
			value = value[:index] + "[REDACTED]" + value[index+end:]
			lower = strings.ToLower(value)
		}
	}
	return clamp(value, 3000)
}

func terminalStatus(status SessionStatus) bool {
	return status == StatusCompleted || status == StatusFailed || status == StatusCancelled
}

func durationMilliseconds(start time.Time, finish *time.Time) int64 {
	end := time.Now().UTC()
	if finish != nil {
		end = finish.UTC()
	}
	return max64(0, end.Sub(start).Milliseconds())
}

func cloneSession(session CoderSession) CoderSession {
	data, _ := json.Marshal(session)
	var clone CoderSession
	_ = json.Unmarshal(data, &clone)
	return clone
}

func newSessionID() (string, error) {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "coder_" + hex.EncodeToString(raw), nil
}

func filepathBase(path string) string {
	path = strings.TrimRight(strings.ReplaceAll(path, "\\", "/"), "/")
	if index := strings.LastIndex(path, "/"); index >= 0 {
		return path[index+1:]
	}
	return path
}

func appendUnique(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func clamp(value string, maximum int) string {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > maximum {
		return string([]rune(value)[:maximum]) + "…"
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func max64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}

// osErrNotExist keeps errors.Is semantics without exposing storage details.
func osErrNotExist() error { return os.ErrNotExist }
