package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"go-aethel/intelligence"
	"go-aethel/osint"
	"go-aethel/personal"
)

func HandleIntelligence(w http.ResponseWriter, r *http.Request) {
	if state == nil || state.intel == nil {
		intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "intelligence core starting"})
		return
	}
	switch {
	case r.URL.Path == "/v1/intelligence/schedule" && r.Method == http.MethodGet:
		if state.intelMonitor == nil {
			intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "global watch monitor starting"})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, state.intelMonitor.Snapshot())
		return
	case r.URL.Path == "/v1/intelligence/schedule" && r.Method == http.MethodPost:
		if state.intelMonitor == nil {
			intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "global watch monitor starting"})
			return
		}
		var req struct {
			Enabled         bool `json:"enabled"`
			IntervalMinutes int  `json:"interval_minutes"`
			RetentionDays   int  `json:"retention_days"`
			MaxReports      int  `json:"max_reports"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&req) != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid schedule payload"})
			return
		}
		current := state.intelMonitor.Snapshot()
		if req.RetentionDays == 0 {
			req.RetentionDays = current.RetentionDays
		}
		if req.MaxReports == 0 {
			req.MaxReports = current.MaxReports
		}
		if err := state.intelMonitor.ConfigurePolicy(req.Enabled, req.IntervalMinutes, req.RetentionDays, req.MaxReports); err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, state.intelMonitor.Snapshot())
		return
	case r.URL.Path == "/v1/intelligence/alerts" && r.Method == http.MethodGet:
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"alerts": state.intel.Alerts()})
		return
	case r.URL.Path == "/v1/intelligence/briefing" && r.Method == http.MethodGet:
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]string{"format": "markdown", "briefing": state.intel.Briefing()})
		return
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/cases/") && strings.HasSuffix(r.URL.Path, "/reid") && r.Method == http.MethodGet:
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 5 {
			intelligence.WriteIntelJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		result, err := state.intel.ReIDStatus(parts[3])
		if err != nil {
			intelligence.WriteIntelJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, result)
		return
	case r.URL.Path == "/v1/intelligence/stream" && r.Method == http.MethodGet:
		if state.intel.Bus() == nil {
			intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "event bus starting"})
			return
		}
		flusher, ok := w.(http.Flusher)
		if !ok {
			intelligence.WriteIntelJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unavailable"})
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		id, events := state.intel.Bus().Subscribe()
		defer state.intel.Bus().Unsubscribe(id)
		fmt.Fprint(w, "event: ready\ndata: {\"type\":\"ready\"}\n\n")
		flusher.Flush()
		heartbeat := time.NewTicker(20 * time.Second)
		defer heartbeat.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case event := <-events:
				raw, err := state.intel.Bus().Encode(event)
				if err == nil {
					fmt.Fprintf(w, "event: intelligence\ndata: %s\n\n", raw)
					flusher.Flush()
				}
			case <-heartbeat.C:
				fmt.Fprint(w, ": ping\n\n")
				flusher.Flush()
			}
		}
	case r.URL.Path == "/v1/intelligence/correlations" && r.Method == http.MethodGet:
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"correlations": state.intel.Correlations()})
		return
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/cases/") && strings.HasSuffix(r.URL.Path, "/report") && r.Method == http.MethodGet:
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 5 {
			intelligence.WriteIntelJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		report, err := state.intel.CaseReport(parts[3])
		if err != nil {
			intelligence.WriteIntelJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]string{"format": "markdown", "report": report})
		return
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/cases/") && strings.HasSuffix(r.URL.Path, "/validate") && r.Method == http.MethodPost:
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 5 {
			intelligence.WriteIntelJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		var req struct {
			EvidenceID string `json:"evidence_id"`
			Decision   string `json:"decision"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req) != nil {
			intelligence.WriteIntelJSON(w, 400, map[string]string{"error": "invalid validation payload"})
			return
		}
		e, err := state.intel.ValidateEvidence(parts[3], req.EvidenceID, "operator", req.Decision)
		if err != nil {
			intelligence.WriteIntelJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, e)
		return
	case r.URL.Path == "/v1/intelligence/sources" && r.Method == http.MethodGet:
		if state.intelSources == nil {
			intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "source registry starting"})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"sources": state.intelSources.List()})
		return
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/sources/") && strings.HasSuffix(r.URL.Path, "/collect") && r.Method == http.MethodPost:
		if state.intelSources == nil {
			intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "source registry starting"})
			return
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 5 {
			intelligence.WriteIntelJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		count, err := state.intelSources.Collect(parts[3])
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"source_id": parts[3], "observations_created": count})
		return
	case r.URL.Path == "/v1/intelligence/status" && r.Method == http.MethodGet:
		if intelligence.SharedIntelStore != nil {
			snap := intelligence.SharedIntelStore.GetSnapshot()
			intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{
				"mode":         "unified-shared-model",
				"event_bus":    "active",
				"observations": len(snap.Observations),
				"events":       len(snap.Events),
				"risk_regions": len(snap.RiskScores),
				"alerts":       len(snap.Alerts),
				"sources":      len(snap.Sources),
				"raw_pii":      false,
			})
			return
		}
		d := state.intel.Snapshot()
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"mode": "local-first", "event_bus": "active", "revision": d.Revision, "events": len(d.Events), "cases": len(d.Cases), "raw_pii": false, "external_collectors": "opt-in"})
	case r.URL.Path == "/v1/intelligence/risks" && r.Method == http.MethodGet:
		if intelligence.SharedIntelStore != nil {
			intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"risks": intelligence.SharedIntelStore.GetRiskScores()})
			return
		}
		risks := state.intel.ComputeAllRegionalRisks()
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"risks": risks})
		return
	case r.URL.Path == "/v1/intelligence/alerts" && r.Method == http.MethodGet:
		if intelligence.SharedIntelStore != nil {
			intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"alerts": intelligence.SharedIntelStore.GetAlerts()})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"alerts": state.intel.Alerts()})
		return
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/alerts/") && strings.HasSuffix(r.URL.Path, "/evaluate") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/intelligence/alerts/"), "/evaluate")
		handleAlertAIEvaluation(w, r, id)
		return
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/alerts/") && strings.HasSuffix(r.URL.Path, "/ack") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/intelligence/alerts/"), "/ack")
		id = strings.Trim(id, "/")
		if intelligence.SharedIntelStore == nil {
			intelligence.WriteIntelJSON(w, 503, map[string]string{"error": "intelligence.SharedIntelStore unavailable"})
			return
		}
		if err := intelligence.SharedIntelStore.AcknowledgeAlert(id); err != nil {
			intelligence.WriteIntelJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"acknowledged": true, "id": id})
		return
	case r.URL.Path == "/v1/intelligence/sources" && r.Method == http.MethodGet:
		if intelligence.SharedIntelStore != nil {
			intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"sources": intelligence.SharedIntelStore.GetSources()})
			return
		}
		// fallback
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"sources": []string{}})
		return
	case r.URL.Path == "/v1/intelligence/briefing" && r.Method == http.MethodGet:
		if intelligence.SharedIntelStore != nil {
			intelligence.WriteIntelJSON(w, http.StatusOK, map[string]string{"format": "markdown", "briefing": intelligence.SharedIntelStore.GenerateReport("Global Brief")})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]string{"format": "markdown", "briefing": state.intel.Briefing()})
		return
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/explain/") && r.Method == http.MethodGet:
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/intelligence/explain/"), "/")
		region := "GERMANY"
		if len(parts) > 0 && parts[0] != "" {
			region = parts[0]
		}
		if intelligence.SharedIntelStore != nil {
			intelligence.WriteIntelJSON(w, http.StatusOK, map[string]string{"region": region, "explanation": intelligence.SharedIntelStore.ExplainScore(region)})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]string{"region": region, "explanation": "Shared store unavailable; using legacy."})
		return
	case r.URL.Path == "/v1/intelligence/events":
		if r.Method == http.MethodGet {
			if intelligence.SharedIntelStore != nil {
				// Use shared sub model exclusively for unified data (Observation/Event from Ingest)
				subSnap := intelligence.SharedIntelStore.GetSnapshot()
				intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"events": subSnap.Events, "revision": 0})
				return
			}
			d := state.intel.Snapshot()
			intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"events": d.Events, "revision": d.Revision})
			return
		}
		if r.Method == http.MethodPost {
			var e intelligence.IntelligenceEvent
			if json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&e) != nil {
				intelligence.WriteIntelJSON(w, 400, map[string]string{"error": "invalid event payload"})
				return
			}
			// U1: intelligence.SharedIntelStore is authoritative; root keeps bridge for cases/SSE.
			if intelligence.SharedIntelStore != nil {
				obsID := e.ID
				if obsID == "" {
					obsID = fmt.Sprintf("obs-http-%d", time.Now().UnixNano())
				}
				raw := strings.TrimSpace(e.Title + " " + e.Summary)
				src := strings.TrimSpace(e.Source)
				if src == "" {
					src = "http-post"
				}
				intelligence.SharedIntelStore.IngestObservation(intelligence.Observation{
					ID: obsID, SourceID: src, RawText: raw, ObservedAt: time.Now().UTC(),
					Latitude: e.Latitude, Longitude: e.Longitude, Domain: "geo",
				})
				e.ID = obsID
				_ = state.intel.ProposeEvent(e)
				intelligence.WriteIntelJSON(w, 201, e)
				return
			}
			if err := state.intel.ProposeEvent(e); err != nil {
				intelligence.WriteIntelJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			intelligence.WriteIntelJSON(w, 201, e)
			return
		}
	case r.URL.Path == "/v1/intelligence/cases":
		if r.Method == http.MethodGet {
			d := state.intel.Snapshot()
			// Root remains UI authority for full case graph; shared shells for id alignment.
			sharedCases := []any{}
			if intelligence.SharedIntelStore != nil {
				for _, c := range intelligence.SharedIntelStore.ListCases() {
					sharedCases = append(sharedCases, map[string]any{
						"id": c.ID, "title": c.Title, "purpose": c.Purpose,
						"evidence_count": len(c.Evidence), "entity_count": len(c.Entities), "relation_count": len(c.Relations),
					})
				}
			}
			intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{
				"cases": d.Cases, "revision": d.Revision,
				"shared_cases": sharedCases, "case_id_alignment": "root_create_mirrors_shared_same_id",
			})
			return
		}
		if r.Method == http.MethodPost {
			var req struct {
				Title   string `json:"title"`
				Purpose string `json:"purpose"`
			}
			if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req) != nil {
				intelligence.WriteIntelJSON(w, 400, map[string]string{"error": "invalid case payload"})
				return
			}
			// CreateCase mirrors Shared with the SAME case id (U5 alignment)
			c, err := state.intel.CreateCase(req.Title, req.Purpose)
			if err != nil {
				intelligence.WriteIntelJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			intelligence.WriteIntelJSON(w, 201, c)
			return
		}
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/cases/") && r.Method == http.MethodDelete:
		// DELETE /v1/intelligence/cases/{id} — reject nested paths
		rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/intelligence/cases/"), "/")
		if rest == "" || strings.Contains(rest, "/") {
			intelligence.WriteIntelJSON(w, 400, map[string]string{"error": "use DELETE /v1/intelligence/cases/{id}"})
			return
		}
		caseID := rest
		if err := state.intel.DeleteCase(caseID); err != nil {
			intelligence.WriteIntelJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		if intelligence.SharedIntelStore != nil {
			_ = intelligence.SharedIntelStore.DeleteCase(caseID)
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": caseID})
		return
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/cases/") && strings.HasSuffix(r.URL.Path, "/evidence") && r.Method == http.MethodPost:
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 5 {
			intelligence.WriteIntelJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		var req struct {
			Source        string `json:"source"`
			URL           string `json:"url"`
			Excerpt       string `json:"excerpt"`
			SourceEventID string `json:"source_event_id"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req) != nil {
			intelligence.WriteIntelJSON(w, 400, map[string]string{"error": "invalid evidence payload"})
			return
		}
		e, err := state.intel.SealEvidenceWithEvent(parts[3], req.Source, req.URL, req.Excerpt, "operator", req.SourceEventID)
		if err != nil {
			intelligence.WriteIntelJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		intelligence.WriteIntelJSON(w, 201, e)
		return
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/cases/") && strings.HasSuffix(r.URL.Path, "/entities") && r.Method == http.MethodPost:
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 5 {
			intelligence.WriteIntelJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		var req struct {
			Label      string `json:"label"`
			Kind       string `json:"kind"`
			Confidence int    `json:"confidence"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req) != nil {
			intelligence.WriteIntelJSON(w, 400, map[string]string{"error": "invalid entity payload"})
			return
		}
		ent, err := state.intel.AddEntity(parts[3], req.Label, req.Kind, req.Confidence)
		if err != nil {
			intelligence.WriteIntelJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		intelligence.WriteIntelJSON(w, 201, ent)
		return
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/cases/") && strings.HasSuffix(r.URL.Path, "/relations") && r.Method == http.MethodPost:
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 5 {
			intelligence.WriteIntelJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		var req struct {
			From       string `json:"from_entity_id"`
			To         string `json:"to_entity_id"`
			Relation   string `json:"relation"`
			EvidenceID string `json:"evidence_id"`
			Confidence int    `json:"confidence"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req) != nil {
			intelligence.WriteIntelJSON(w, 400, map[string]string{"error": "invalid relation payload"})
			return
		}
		rel, err := state.intel.LinkEntities(parts[3], req.From, req.To, req.Relation, req.EvidenceID, req.Confidence)
		if err != nil {
			intelligence.WriteIntelJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		intelligence.WriteIntelJSON(w, 201, rel)
		return
	case r.URL.Path == "/v1/intelligence/risk" && r.Method == http.MethodGet:
		// Prefer unified intelligence.SharedIntelStore so Risk HUD matches map/chat/explain.
		if intelligence.SharedIntelStore != nil {
			snap := intelligence.SharedIntelStore.GetSnapshot()
			out := make([]intelligence.RegionalRiskData, 0, len(snap.RiskScores))
			for id, rs := range snap.RiskScores {
				out = append(out, intelligence.RegionalRiskData{
					RegionID:           id,
					RegionName:         id,
					OverallRisk:        rs.OverallRisk,
					GeopoliticalRisk:   rs.GeopoliticalRisk,
					ConflictRisk:       rs.ConflictRisk,
					CyberRisk:          rs.CyberRisk,
					InfrastructureRisk: rs.InfrastructureRisk,
					EconomicRisk:       rs.EconomicRisk,
					PrimaryDrivers:     rs.PrimaryDrivers,
					Trend:              rs.Trend,
				})
			}
			sort.Slice(out, func(i, j int) bool { return out[i].OverallRisk > out[j].OverallRisk })
			intelligence.WriteIntelJSON(w, http.StatusOK, out)
			return
		}
		risks := state.intel.ComputeAllRegionalRisks()
		intelligence.WriteIntelJSON(w, http.StatusOK, risks)
		return
	case r.URL.Path == "/v1/intelligence/evaluation" && r.Method == http.MethodGet:
		eval, err := state.intel.UpdateEvaluation()
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, eval)
		return
	case r.URL.Path == "/v1/intelligence/evaluation" && r.Method == http.MethodPost:
		eval, err := state.intel.UpdateEvaluation()
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, eval)
		return
	case r.URL.Path == "/v1/intelligence/personal/context" && r.Method == http.MethodGet:
		// U3: read operator personal context mirrored into intelligence.SharedIntelStore
		if intelligence.SharedIntelStore == nil {
			intelligence.WriteIntelJSON(w, 503, map[string]string{"error": "intelligence.SharedIntelStore unavailable"})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"personal_context": intelligence.SharedIntelStore.GetPersonalContext()})
		return
	case r.URL.Path == "/v1/intelligence/personal/sync" && r.Method == http.MethodPost:
		// U3: explicit opt-in bridge PersonalStore → intelligence.SharedIntelStore.SetPersonalContext
		if intelligence.SharedIntelStore == nil {
			intelligence.WriteIntelJSON(w, 503, map[string]string{"error": "intelligence.SharedIntelStore unavailable"})
			return
		}
		if state == nil || state.personal == nil {
			intelligence.WriteIntelJSON(w, 503, map[string]string{"error": "PersonalStore unavailable"})
			return
		}
		pc := personal.BuildSharedPersonalContext(state.personal)
		intelligence.SharedIntelStore.SetPersonalContext(pc)
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{
			"synced":           true,
			"personal_context": intelligence.SharedIntelStore.GetPersonalContext(),
			"note":             "Personal memory is NOT mixed into cases. Correlation is opt-in for impact only.",
		})
		return
	case r.URL.Path == "/v1/intelligence/personal/impact" && r.Method == http.MethodGet:
		if intelligence.SharedIntelStore == nil {
			intelligence.WriteIntelJSON(w, 503, map[string]string{"error": "intelligence.SharedIntelStore unavailable"})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{
			"format":     "markdown",
			"brief":      intelligence.SharedIntelStore.PersonalImpact(),
			"disclaimer": "Empfehlungen ≠ Fakten. Only PersonalContext + World State correlation.",
		})
		return
	case r.URL.Path == "/v1/intelligence/identity" && r.Method == http.MethodGet:
		if intelligence.SharedIntelStore == nil {
			intelligence.WriteIntelJSON(w, 503, map[string]string{"error": "intelligence.SharedIntelStore unavailable"})
			return
		}
		id := intelligence.SharedIntelStore.IdentitySnapshot()
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{
			"identity": id,
			"note":     "Technical continuity only — not consciousness or agency claims.",
		})
		return
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/assessments/") && strings.HasSuffix(r.URL.Path, "/status") && r.Method == http.MethodPost:
		// POST /v1/intelligence/assessments/{id}/status
		rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/intelligence/assessments/"), "/")
		rest = strings.TrimSuffix(rest, "/status")
		rest = strings.Trim(rest, "/")
		if rest == "" || strings.Contains(rest, "/") {
			intelligence.WriteIntelJSON(w, 400, map[string]string{"error": "use POST /v1/intelligence/assessments/{id}/status"})
			return
		}
		if intelligence.SharedIntelStore == nil {
			intelligence.WriteIntelJSON(w, 503, map[string]string{"error": "intelligence.SharedIntelStore unavailable"})
			return
		}
		var req struct {
			Status   string `json:"status"`
			Reviewer string `json:"reviewer"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&req) != nil {
			intelligence.WriteIntelJSON(w, 400, map[string]string{"error": "invalid status payload"})
			return
		}
		if err := intelligence.SharedIntelStore.SetAssessmentStatus(rest, req.Status, req.Reviewer); err != nil {
			intelligence.WriteIntelJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"updated": true, "id": rest, "status": req.Status})
		return
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/cases/") && strings.HasSuffix(r.URL.Path, "/reid") && r.Method == http.MethodPost:
		// POST body: { "action": "request"|"approve", "purpose": "...", "request_id": "...", "approver": "..." }
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 5 {
			intelligence.WriteIntelJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		caseID := parts[3]
		var req struct {
			Action    string `json:"action"`
			Purpose   string `json:"purpose"`
			RequestID string `json:"request_id"`
			Approver  string `json:"approver"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req) != nil {
			intelligence.WriteIntelJSON(w, 400, map[string]string{"error": "invalid reid payload"})
			return
		}
		switch strings.ToLower(strings.TrimSpace(req.Action)) {
		case "request", "":
			rr, err := state.intel.RequestReID(caseID, req.Purpose, req.Approver)
			if err != nil {
				intelligence.WriteIntelJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			intelligence.WriteIntelJSON(w, 201, rr)
			return
		case "approve":
			rr, err := state.intel.ApproveReID(caseID, req.RequestID, req.Approver)
			if err != nil {
				intelligence.WriteIntelJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			intelligence.WriteIntelJSON(w, http.StatusOK, rr)
			return
		default:
			intelligence.WriteIntelJSON(w, 400, map[string]string{"error": "action must be request or approve"})
			return
		}
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/cases/") && strings.HasSuffix(r.URL.Path, "/timeline") && r.Method == http.MethodGet:
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 5 {
			intelligence.WriteIntelJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		tl, err := state.intel.CaseTimeline(parts[3])
		if err != nil {
			intelligence.WriteIntelJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"format": "markdown", "timeline": tl})
		return
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/cases/") && strings.HasSuffix(r.URL.Path, "/report") && r.Method == http.MethodGet:
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 5 {
			intelligence.WriteIntelJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		rep, err := state.intel.CaseReport(parts[3])
		if err != nil {
			intelligence.WriteIntelJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"format": "markdown", "report": rep})
		return
	case r.URL.Path == "/v1/intelligence/connectors" && r.Method == http.MethodGet:
		intelligence.WriteIntelJSON(w, http.StatusOK, osint.ConnectorRegistrySummary())
		return
	case r.URL.Path == "/v1/intelligence/connectors/fetch" && r.Method == http.MethodPost:
		var req struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&req)
		fetched, ingested, err := osint.RunConnectorFetchIngest(req.Name)
		if err != nil {
			intelligence.WriteIntelJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{
			"fetched": fetched, "ingested": ingested, "connector": strings.TrimSpace(req.Name),
			"note": "Observations are RAW; assessments start unverified unless multi-source corroborated.",
		})
		return
	default:
		intelligence.WriteIntelJSON(w, 404, map[string]string{"error": "intelligence endpoint not found"})
	}
}
