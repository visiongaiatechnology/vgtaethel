package handlers

// STATUS: DIAMANT VGT SUPREME

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"

	"go-aethel/agent"
	"go-aethel/intelligence"
	"go-aethel/osint"
)

var shadowService *osint.ShadowService

func InitShadowService(service *osint.ShadowService) { shadowService = service }

func RunShadowAutoAnalysis() error {
	if shadowService == nil {
		return errors.New("SHADOW mode unavailable")
	}
	items, prompt, err := shadowService.PrepareBatch()
	if err != nil {
		return err
	}
	defer shadowService.AbortBatch()
	payload, _ := json.Marshal(items)
	report, err := executeShadowModel("", prompt, "SHADOW_BATCH_JSON="+string(payload))
	if err != nil {
		return err
	}
	_, err = shadowService.CompleteBatch(items, report)
	return err
}

func handleShadow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if shadowService == nil {
		intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "SHADOW mode unavailable"})
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/shadow/"), "/")
	switch path {
	case "status":
		if r.Method != http.MethodGet {
			shadowMethodNotAllowed(w)
			return
		}
		_ = json.NewEncoder(w).Encode(shadowService.Status())
	case "snapshot":
		if r.Method != http.MethodGet {
			shadowMethodNotAllowed(w)
			return
		}
		snapshot := shadowService.Snapshot()
		snapshot.Buffer = latestShadowItems(snapshot.Buffer, 120)
		_ = json.NewEncoder(w).Encode(snapshot)
	case "sources":
		handleShadowSources(w, r)
	case "collect":
		if r.Method != http.MethodPost {
			shadowMethodNotAllowed(w)
			return
		}
		var input struct {
			SourceLimit int `json:"source_limit"`
		}
		if err := decodeShadowJSON(w, r, &input, 4<<10); err != nil {
			return
		}
		added, err := shadowService.Collect(r.Context(), input.SourceLimit)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"added": added, "status": shadowService.Status()})
	case "analyze":
		if r.Method != http.MethodPost {
			shadowMethodNotAllowed(w)
			return
		}
		handleShadowAnalyze(w, r, false)
	case "daily":
		if r.Method != http.MethodPost {
			shadowMethodNotAllowed(w)
			return
		}
		handleShadowAnalyze(w, r, true)
	case "reports":
		if r.Method != http.MethodGet {
			shadowMethodNotAllowed(w)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"reports": shadowService.Reports()})
	case "regions":
		if r.Method != http.MethodGet {
			shadowMethodNotAllowed(w)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"regions": shadowService.LatestRegions(), "conflict_links": shadowService.LatestConflictLinks(),
			"authority": "ai_evidence_bound",
		})
	case "prompt":
		handleShadowPrompt(w, r)
	case "export":
		handleShadowExport(w, r)
	default:
		http.NotFound(w, r)
	}
}

func shadowMethodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func decodeShadowJSON(w http.ResponseWriter, r *http.Request, target any, limit int64) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		http.Error(w, "invalid SHADOW request", http.StatusBadRequest)
		return err
	}
	return nil
}

func latestShadowItems(items []osint.ShadowIntelItem, limit int) []osint.ShadowIntelItem {
	if len(items) <= limit {
		return items
	}
	return append([]osint.ShadowIntelItem(nil), items[len(items)-limit:]...)
}

func handleShadowSources(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		_ = json.NewEncoder(w).Encode(map[string]any{"sources": shadowService.Snapshot().Sources})
	case http.MethodPost, http.MethodPut:
		var source osint.ShadowSource
		if err := decodeShadowJSON(w, r, &source, 16<<10); err != nil {
			return
		}
		if err := shadowService.UpsertSource(source); err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "saved"})
	case http.MethodDelete:
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" || len(id) > 100 {
			http.Error(w, "invalid source id", http.StatusBadRequest)
			return
		}
		if err := shadowService.DeleteSource(id); err != nil {
			intelligence.WriteIntelJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	default:
		shadowMethodNotAllowed(w)
	}
}

func handleShadowPrompt(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		_ = json.NewEncoder(w).Encode(map[string]string{"system_prompt": shadowService.Snapshot().SystemPrompt})
		return
	}
	if r.Method != http.MethodPut {
		shadowMethodNotAllowed(w)
		return
	}
	var input struct {
		SystemPrompt string `json:"system_prompt"`
	}
	if err := decodeShadowJSON(w, r, &input, 32<<10); err != nil {
		return
	}
	if err := shadowService.SetSystemPrompt(input.SystemPrompt); err != nil {
		intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "saved"})
}

func handleShadowAnalyze(w http.ResponseWriter, r *http.Request, daily bool) {
	var input struct {
		ModelID string `json:"model_id"`
	}
	if err := decodeShadowJSON(w, r, &input, 4<<10); err != nil {
		return
	}
	if len(input.ModelID) > 256 {
		http.Error(w, "invalid model", http.StatusBadRequest)
		return
	}
	if daily {
		reports := shadowService.ReportsForDay(time.Now())
		if len(reports) < 2 {
			intelligence.WriteIntelJSON(w, http.StatusConflict, map[string]string{"error": "daily synthesis requires at least two batch reports today"})
			return
		}
		prompt := shadowService.AnalysisPrompt() + "\n\nMETA-SYNTHESE: Verdichte die folgenden Tagesdossiers. Bewahre Evidence-IDs. Setze divergences und confirmed_vectors explizit."
		payload, _ := json.Marshal(reports)
		report, err := executeShadowModel(input.ModelID, prompt, "SHADOW_DAILY_REPORTS_JSON="+string(payload))
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		report.ItemsAnalyzed = 0
		for _, entry := range reports {
			report.ItemsAnalyzed += entry.ItemsAnalyzed
		}
		saved, err := shadowService.SaveDailyReport(report)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(saved)
		return
	}
	items, prompt, err := shadowService.PrepareBatch()
	if err != nil {
		intelligence.WriteIntelJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	defer shadowService.AbortBatch()
	payload, _ := json.Marshal(items)
	report, err := executeShadowModel(input.ModelID, prompt, "SHADOW_BATCH_JSON="+string(payload))
	if err != nil {
		intelligence.WriteIntelJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	saved, err := shadowService.CompleteBatch(items, report)
	if err != nil {
		intelligence.WriteIntelJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(saved)
}

func executeShadowModel(modelID, systemPrompt, userContent string) (osint.ShadowReport, error) {
	if strings.TrimSpace(modelID) == "" {
		modelID = "openai/gpt-oss-120b"
	}
	message, _ := json.Marshal(map[string]string{"role": "user", "content": userContent})
	request := ChatRequest{ModelID: modelID, Messages: []json.RawMessage{message}, SystemPrompt: systemPrompt, Temperature: 0.2, UseTools: false, ReasoningEffort: "high", ReasoningVisibility: "hidden", TextOnly: true}
	body, err := json.Marshal(request)
	if err != nil {
		return osint.ShadowReport{}, err
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handleChat(rec, req)
	if rec.Code >= 400 {
		return osint.ShadowReport{}, fmt.Errorf("SHADOW model returned HTTP %d", rec.Code)
	}
	parsed := agent.ParseAgentSSE(rec.Body.String())
	if parsed.Err != nil {
		return osint.ShadowReport{}, parsed.Err
	}
	content := strings.TrimSpace(parsed.Text)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	if len(content) == 0 || len(content) > 512<<10 {
		return osint.ShadowReport{}, errors.New("SHADOW model response outside size boundary")
	}
	var report osint.ShadowReport
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return osint.ShadowReport{}, errors.New("SHADOW model returned invalid structured JSON")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return osint.ShadowReport{}, errors.New("SHADOW model returned trailing structured data")
	}
	return report, nil
}

func handleShadowExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		shadowMethodNotAllowed(w)
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "json"
	}
	var found *osint.ShadowReport
	for _, report := range shadowService.Reports() {
		if report.ID == id {
			copy := report
			found = &copy
			break
		}
	}
	if found == nil {
		http.Error(w, "report not found", http.StatusNotFound)
		return
	}
	filename := "shadow-" + strconv.FormatInt(found.CreatedAt.Unix(), 10)
	if format == "markdown" {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`.md"`)
		_, _ = w.Write([]byte(shadowReportMarkdown(*found)))
		return
	}
	if format != "json" {
		http.Error(w, "invalid export format", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`.json"`)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(found)
}

func shadowReportMarkdown(report osint.ShadowReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# SHADOW INTELLIGENCE DOSSIER\n\n**Threat:** %s  \n**Created:** %s  \n**SHA-256:** `%s`\n\n## Executive Summary\n\n%s\n\n## Tactical Situation\n\n%s\n\n## Cui Bono\n\n%s\n\n## Strategic Reality\n\n%s\n\n## Divergences\n\n%s\n\n## Confirmed Vectors\n\n%s\n\n## Regional Assessments\n", report.ThreatLevel, report.CreatedAt.Format(time.RFC3339), report.ContentSHA256, report.Summary, report.Situation, report.CuiBono, report.StrategicReality, report.Divergences, report.ConfirmedVectors)
	for _, region := range report.Regions {
		fmt.Fprintf(&b, "- **%s** security=%d confidence=%d conflict=%s — %s\n", region.RegionName, region.SecurityScore, region.Confidence, region.ConflictLevel, region.Assessment)
	}
	b.WriteString("\n## Directed Conflict Vectors\n")
	for _, link := range report.ConflictLinks {
		fmt.Fprintf(&b, "- **%s → %s** action=%s confidence=%d — %s\n", link.AttackerName, link.TargetName, link.Action, link.Confidence, link.Assessment)
	}
	return b.String()
}
