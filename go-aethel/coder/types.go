// STATUS: DIAMANT VGT SUPREME
package coder

import "time"

type SessionStatus string

const (
	StatusCreated         SessionStatus = "CREATED"
	StatusPlanning        SessionStatus = "PLANNING"
	StatusRunning         SessionStatus = "RUNNING"
	StatusWaitingApproval SessionStatus = "WAITING_APPROVAL"
	StatusReview          SessionStatus = "REVIEW"
	StatusCompleted       SessionStatus = "COMPLETED"
	StatusFailed          SessionStatus = "FAILED"
	StatusCancelled       SessionStatus = "CANCELLED"
)

type EventStatus string

const (
	EventQueued    EventStatus = "QUEUED"
	EventRunning   EventStatus = "RUNNING"
	EventSuccess   EventStatus = "SUCCESS"
	EventFailed    EventStatus = "FAILED"
	EventCancelled EventStatus = "CANCELLED"
)

type CoderEvent struct {
	ID         string                 `json:"id"`
	SessionID  string                 `json:"session_id"`
	Timestamp  time.Time              `json:"timestamp"`
	Type       string                 `json:"type"`
	Status     EventStatus            `json:"status"`
	Title      string                 `json:"title"`
	Summary    string                 `json:"summary,omitempty"`
	Path       string                 `json:"path,omitempty"`
	Command    string                 `json:"command,omitempty"`
	DurationMS int64                  `json:"duration_ms,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type CoderChangedFile struct {
	Path      string `json:"path"`
	OldPath   string `json:"old_path,omitempty"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Binary    bool   `json:"binary"`
	Language  string `json:"language"`
	Signature string `json:"-"`
}

type CoderStats struct {
	FilesChanged  int   `json:"files_changed"`
	FilesAdded    int   `json:"files_added"`
	FilesModified int   `json:"files_modified"`
	FilesDeleted  int   `json:"files_deleted"`
	LinesAdded    int   `json:"lines_added"`
	LinesDeleted  int   `json:"lines_deleted"`
	CommandsRun   int   `json:"commands_run"`
	TestsPassed   int   `json:"tests_passed"`
	TestsFailed   int   `json:"tests_failed"`
	DurationMS    int64 `json:"duration_ms"`
}

type CoderValidationResult struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Command    string `json:"command,omitempty"`
	Status     string `json:"status"`
	Passed     int    `json:"passed"`
	Failed     int    `json:"failed"`
	DurationMS int64  `json:"duration_ms"`
	Output     string `json:"output,omitempty"`
}

type CoderDiffLine struct {
	Kind      string `json:"kind"`
	OldNumber int    `json:"old_number,omitempty"`
	NewNumber int    `json:"new_number,omitempty"`
	Content   string `json:"content"`
}

type CoderDiffHunk struct {
	Header string          `json:"header"`
	Lines  []CoderDiffLine `json:"lines"`
}

type CoderDiff struct {
	Path      string          `json:"path"`
	OldPath   string          `json:"old_path,omitempty"`
	Status    string          `json:"status"`
	Binary    bool            `json:"binary"`
	Additions int             `json:"additions"`
	Deletions int             `json:"deletions"`
	Hunks     []CoderDiffHunk `json:"hunks"`
	Truncated bool            `json:"truncated"`
}

type CoderSession struct {
	SessionID         string                     `json:"session_id"`
	RunID             string                     `json:"run_id,omitempty"`
	ProjectRoot       string                     `json:"project_root"`
	RepositoryRoot    string                     `json:"repository_root"`
	ProjectName       string                     `json:"project_name"`
	Branch            string                     `json:"branch,omitempty"`
	BaseBranch        string                     `json:"base_branch,omitempty"`
	DiffMode          string                     `json:"diff_mode"`
	WorkerModel       string                     `json:"worker_model"`
	OrchestratorModel string                     `json:"orchestrator_model"`
	StartedAt         time.Time                  `json:"started_at"`
	FinishedAt        *time.Time                 `json:"finished_at,omitempty"`
	DurationMS        int64                      `json:"duration_ms"`
	Status            SessionStatus              `json:"status"`
	UserRequest       string                     `json:"user_request"`
	AgentSummary      string                     `json:"agent_summary,omitempty"`
	Events            []CoderEvent               `json:"events"`
	ChangedFiles      []CoderChangedFile         `json:"changed_files"`
	WorktreeFiles     []CoderChangedFile         `json:"worktree_files"`
	BranchFiles       []CoderChangedFile         `json:"branch_files"`
	Stats             CoderStats                 `json:"stats"`
	ValidationResults []CoderValidationResult    `json:"validation_results"`
	Warnings          []string                   `json:"warnings"`
	BaselineChanges   map[string]string          `json:"baseline_changes,omitempty"`
	BaselineFiles     map[string]FileFingerprint `json:"baseline_files,omitempty"`
	LastTrackedAt     time.Time                  `json:"last_tracked_at,omitempty"`
	LastActivityAt    time.Time                  `json:"last_activity_at,omitempty"`
	Health            string                     `json:"health,omitempty"`
	HealthMessage     string                     `json:"health_message,omitempty"`
	CanCancel         bool                       `json:"can_cancel"`
}

type FileFingerprint struct {
	Hash   string `json:"hash"`
	Size   int64  `json:"size"`
	Binary bool   `json:"binary"`
	Lines  int    `json:"lines"`
}
