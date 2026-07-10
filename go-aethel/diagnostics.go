// STATUS: DIAMANT VGT SUPREME
package main

import (
	"archive/zip"
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

type diagnosticRun struct {
	ID        string    `json:"id"`
	Status    RunStatus `json:"status"`
	ProfileID string    `json:"profile_id"`
	ModelID   string    `json:"model_id,omitempty"`
	Steps     int       `json:"steps"`
	ToolCalls int       `json:"tool_calls"`
}

type diagnosticSnapshot struct {
	Product      string          `json:"product"`
	Version      string          `json:"version"`
	GeneratedAt  time.Time       `json:"generated_at"`
	OperatingSys string          `json:"operating_system"`
	Architecture string          `json:"architecture"`
	GoVersion    string          `json:"go_version"`
	Runs         []diagnosticRun `json:"runs"`
}

// handleDiagnosticsExport emits a deliberately minimal support bundle. It
// excludes prompts, messages, objectives, tool arguments/results, memory,
// secrets, paths, logs and machine identifiers by construction.
func handleDiagnosticsExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	runs := state.runs.List()
	summaries := make([]diagnosticRun, 0, len(runs))
	for _, run := range runs {
		summaries = append(summaries, diagnosticRun{
			ID: run.ID, Status: run.Status, ProfileID: run.ProfileID,
			ModelID: run.ModelID, Steps: len(run.Steps), ToolCalls: run.ToolCalls,
		})
	}
	snapshot := diagnosticSnapshot{
		Product: ProductName, Version: ProductVersion, GeneratedAt: time.Now().UTC(),
		OperatingSys: runtime.GOOS, Architecture: runtime.GOARCH, GoVersion: runtime.Version(), Runs: summaries,
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="aethel-diagnostics.zip"`)
	w.Header().Set("Cache-Control", "no-store")
	archive := zip.NewWriter(w)
	entry, err := archive.CreateHeader(&zip.FileHeader{Name: "diagnostics.json", Method: zip.Deflate})
	if err != nil {
		_ = archive.Close()
		return
	}
	encoder := json.NewEncoder(entry)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(snapshot)
	_ = archive.Close()
}
