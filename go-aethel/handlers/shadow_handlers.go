package handlers

// STATUS: DIAMANT VGT SUPREME

import (
	"bytes"
	"context"
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
	"go-aethel/skills"
)

var shadowService *osint.ShadowService

func InitShadowService(service *osint.ShadowService) { shadowService = service }

func RunShadowAutoAnalysis() error {
	if shadowService == nil {
		return errors.New("SHADOW mode unavailable")
	}
	for completed := 0; completed < 8; completed++ {
		enabled, modelID := shadowService.Autonomy()
		if !enabled || shadowService.Status().PendingItems < osint.ShadowBatchMin {
			return nil
		}
		items, prompt, err := shadowService.PrepareBatch(modelID)
		if err != nil {
			return err
		}
		payload, marketSnapshot := buildShadowAnalysisInput(items)
		report, err := executeShadowModel(modelID, prompt, "SHADOW_ANALYSIS_INPUT_JSON="+string(payload))
		if err != nil {
			shadowService.FailBatch(err)
			return err
		}
		report.MarketSnapshot = marketSnapshot
		if _, err = shadowService.CompleteBatch(items, report); err != nil {
			shadowService.FailBatch(err)
			return err
		}
	}
	return nil
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
	case "autonomy":
		if r.Method != http.MethodPut {
			shadowMethodNotAllowed(w)
			return
		}
		var input struct {
			Enabled bool   `json:"enabled"`
			ModelID string `json:"model_id"`
		}
		if err := decodeShadowJSON(w, r, &input, 4<<10); err != nil {
			return
		}
		if err := shadowService.SetAutonomy(input.Enabled, input.ModelID); err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(shadowService.Status())
		if input.Enabled {
			go func() {
				cycleContext, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				_, _ = shadowService.Collect(cycleContext, 8)
				_ = RunShadowAutoAnalysis()
			}()
		}
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
		marketSnapshot := currentShadowMarketSnapshot()
		payload, _ := json.Marshal(map[string]any{"daily_reports": reports, "market_pulse": marketSnapshot, "forecast_horizon_hours": 72})
		report, err := executeShadowModel(input.ModelID, prompt, "SHADOW_DAILY_ANALYSIS_INPUT_JSON="+string(payload))
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		report.MarketSnapshot = marketSnapshot
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
	items, prompt, err := shadowService.PrepareBatch(input.ModelID)
	if err != nil {
		intelligence.WriteIntelJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	defer shadowService.AbortBatch()
	payload, marketSnapshot := buildShadowAnalysisInput(items)
	report, err := executeShadowModel(input.ModelID, prompt, "SHADOW_ANALYSIS_INPUT_JSON="+string(payload))
	if err != nil {
		shadowService.FailBatch(err)
		intelligence.WriteIntelJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	report.MarketSnapshot = marketSnapshot
	saved, err := shadowService.CompleteBatch(items, report)
	if err != nil {
		shadowService.FailBatch(err)
		intelligence.WriteIntelJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(saved)
}

func buildShadowAnalysisInput(items []osint.ShadowIntelItem) ([]byte, []osint.ShadowMarketPoint) {
	marketSnapshot := currentShadowMarketSnapshot()
	payload, _ := json.Marshal(map[string]any{
		"intel_items": items, "market_pulse": marketSnapshot, "forecast_horizon_hours": 72,
	})
	return payload, marketSnapshot
}

func currentShadowMarketSnapshot() []osint.ShadowMarketPoint {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	quotes, err := skills.LookupMarketQuotes(ctx, []string{"BTC", "ETH", "GOLD", "BRENT", "WTI", "SP500", "NASDAQ", "DAX", "EURUSD", "USDJPY"})
	if err != nil {
		return []osint.ShadowMarketPoint{}
	}
	result := make([]osint.ShadowMarketPoint, 0, len(quotes))
	for _, quote := range quotes {
		if quote.Price <= 0 || quote.ObservedAt.IsZero() {
			continue
		}
		result = append(result, osint.ShadowMarketPoint{
			Symbol: quote.Symbol, Name: quote.Name, Category: quote.Category, Currency: quote.Currency,
			Price: quote.Price, Change24H: quote.Change24H, ObservedAt: quote.ObservedAt, Source: quote.Source,
		})
	}
	return result
}

func executeShadowModel(modelID, systemPrompt, userContent string) (osint.ShadowReport, error) {
	if strings.TrimSpace(modelID) == "" {
		modelID = "openai/gpt-oss-120b"
	}
	message, _ := json.Marshal(map[string]string{"role": "user", "content": userContent})
	request := ChatRequest{ModelID: modelID, Messages: []json.RawMessage{message}, SystemPrompt: systemPrompt, Temperature: 0.2, UseTools: false, ReasoningEffort: "high", ReasoningVisibility: "hidden", TextOnly: true, StructuredJSON: true}
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
	return decodeShadowModelReport(parsed.Text)
}

func decodeShadowModelReport(content string) (osint.ShadowReport, error) {
	content = strings.TrimSpace(content)
	if len(content) == 0 || len(content) > 512<<10 {
		return osint.ShadowReport{}, errors.New("SHADOW model response outside size boundary")
	}
	candidates := extractJSONObjectCandidates(content, 16)
	var lastErr error
	for _, candidate := range candidates {
		var report osint.ShadowReport
		decoder := json.NewDecoder(strings.NewReader(candidate))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&report); err != nil {
			lastErr = err
			continue
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			lastErr = errors.New("trailing structured data")
			continue
		}
		if strings.TrimSpace(report.ThreatLevel) == "" || strings.TrimSpace(report.Summary) == "" {
			lastErr = errors.New("missing report identity fields")
			continue
		}
		return report, nil
	}
	if lastErr != nil {
		message := lastErr.Error()
		if len(message) > 180 {
			message = message[:180]
		}
		return osint.ShadowReport{}, fmt.Errorf("SHADOW model returned invalid structured JSON: %s", message)
	}
	return osint.ShadowReport{}, errors.New("SHADOW model returned no structured JSON object")
}

func extractJSONObjectCandidates(content string, maximum int) []string {
	if maximum < 1 {
		return nil
	}
	result := make([]string, 0, min(maximum, 4))
	start, depth := -1, 0
	inString, escaped := false, false
	for index := 0; index < len(content); index++ {
		char := content[index]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '"' {
				inString = false
			}
			continue
		}
		if char == '"' && depth > 0 {
			inString = true
			continue
		}
		switch char {
		case '{':
			if depth == 0 {
				start = index
			}
			depth++
		case '}':
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 && start >= 0 {
				result = append(result, content[start:index+1])
				start = -1
				if len(result) == maximum {
					return result
				}
			}
		}
	}
	return result
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
	b.WriteString("\n## 72h Forecast Matrix\n")
	for _, forecast := range report.Forecasts {
		fmt.Fprintf(&b, "- **%s / %s / %s** probability=%d instruments=%s — %s\n", forecast.Sector, forecast.Horizon, forecast.Direction, forecast.Probability, strings.Join(forecast.Instruments, ", "), forecast.Prediction)
	}
	b.WriteString("\n## Sphere Market Pulse Snapshot\n")
	for _, point := range report.MarketSnapshot {
		fmt.Fprintf(&b, "- **%s** %.6f %s change24h=%+.2f%% observed=%s source=%s\n", point.Symbol, point.Price, point.Currency, point.Change24H, point.ObservedAt.Format(time.RFC3339), point.Source)
	}
	return b.String()
}
