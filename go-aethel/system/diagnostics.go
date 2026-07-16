// STATUS: DIAMANT VGT SUPREME
package system

import (
	"time"

	
)

type DiagnosticRun struct {
	ID        string          `json:"id"`
	Status    string          `json:"status"`
	ProfileID string          `json:"profile_id"`
	ModelID   string          `json:"model_id,omitempty"`
	Steps     int             `json:"steps"`
	ToolCalls int             `json:"tool_calls"`
}

type DiagnosticSnapshot struct {
	Product      string          `json:"product"`
	Version      string          `json:"version"`
	GeneratedAt  time.Time       `json:"generated_at"`
	OperatingSys string          `json:"operating_system"`
	Architecture string          `json:"architecture"`
	GoVersion    string          `json:"go_version"`
	Runs         []DiagnosticRun `json:"runs"`
}
