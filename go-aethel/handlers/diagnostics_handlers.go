package handlers

import (
	"archive/zip"
	"go-aethel/system"
	"runtime"
	"time"
	"net/http"
	"encoding/json"
)

func HandleDiagnosticsExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if state == nil || state.runs == nil {
		http.Error(w, "diagnostics unavailable", http.StatusServiceUnavailable)
		return
	}
	runs := state.runs.List()
	summaries := make([]system.DiagnosticRun, 0, len(runs))
	for _, run := range runs {
		summaries = append(summaries, system.DiagnosticRun{
			ID: run.ID, Status: string(run.Status), ProfileID: run.ProfileID,
			ModelID: run.ModelID, Steps: len(run.Steps), ToolCalls: run.ToolCalls,
		})
	}
	snapshot := system.DiagnosticSnapshot{
		Product: system.ProductName, Version: system.ProductVersion, GeneratedAt: time.Now().UTC(),
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
