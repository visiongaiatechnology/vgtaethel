package handlers

// STATUS: DIAMANT VGT SUPREME

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-aethel/intelligence"
	"go-aethel/osint"
	"go-aethel/security"
)

const analysisRequestLimit = 64 << 10

func handleAnalysisAPI(w http.ResponseWriter, r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/v1/intelligence/analysis") {
		return false
	}
	store := intelligence.SharedIntelStore
	if store == nil {
		intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "canonical intelligence store unavailable"})
		return true
	}
	switch {
	case r.URL.Path == "/v1/intelligence/analysis" && r.Method == http.MethodGet:
		snapshot := store.GetSnapshot()
		journalError := store.VerifyStateJournal()
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{
			"schema_version": snapshot.SchemaVersion,
			"documents":      len(snapshot.Documents), "claims": len(snapshot.Claims),
			"source_lineage": len(snapshot.SourceLineage), "hypotheses": len(snapshot.Hypotheses),
			"information_gaps": len(snapshot.InformationGaps), "collection_plans": len(snapshot.CollectionPlans),
			"resolved_entities": len(snapshot.ResolvedEntities), "resolution_candidates": len(snapshot.ResolutionCandidates),
			"custody_events": len(snapshot.CustodyEvents), "custody_chain_valid": intelligence.VerifyCustodyChain(snapshot.CustodyEvents),
			"state_journal_valid": journalError == nil,
		})
		return true
	case r.URL.Path == "/v1/intelligence/analysis/claims" && r.Method == http.MethodGet:
		snapshot := store.GetSnapshot()
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"claims": snapshot.Claims})
		return true
	case r.URL.Path == "/v1/intelligence/analysis/workspace" && r.Method == http.MethodGet:
		snapshot := store.GetSnapshot()
		intelligence.WriteIntelJSON(w, http.StatusOK, buildOperatorWorkspace(snapshot))
		return true
	case r.URL.Path == "/v1/intelligence/analysis/search" && r.Method == http.MethodPost:
		var request intelligence.SearchRequest
		if !decodeAnalysisRequest(w, r, &request) {
			return true
		}
		hits, err := store.Search(request)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"hits": hits, "count": len(hits)})
		return true
	case r.URL.Path == "/v1/intelligence/analysis/images/match" && r.Method == http.MethodPost:
		handleImageMatch(w, r, store)
		return true
	case r.URL.Path == "/v1/intelligence/analysis/import" && r.Method == http.MethodPost:
		handleDocumentImport(w, r, store)
		return true
	case r.URL.Path == "/v1/intelligence/analysis/analytics/export" && r.Method == http.MethodPost:
		export, err := store.ExportAnalytics()
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusInternalServerError, map[string]string{"error": "analytics export failed"})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusCreated, export)
		return true
	case r.URL.Path == "/v1/intelligence/analysis/search-monitors" && r.Method == http.MethodGet:
		snapshot := store.GetSnapshot()
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"monitors": snapshot.SavedSearches, "alerts": snapshot.SearchAlerts})
		return true
	case r.URL.Path == "/v1/intelligence/analysis/web-monitors" && r.Method == http.MethodGet:
		snapshot := store.GetSnapshot()
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"monitors": snapshot.WebsiteMonitors, "changes": snapshot.WebsiteChanges})
		return true
	case r.URL.Path == "/v1/intelligence/analysis/web-monitors" && r.Method == http.MethodPost:
		var request intelligence.WebsiteMonitor
		if !decodeAnalysisRequest(w, r, &request) {
			return true
		}
		monitor, err := store.AddWebsiteMonitor(request)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusCreated, monitor)
		return true
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/analysis/web-monitors/") && strings.HasSuffix(r.URL.Path, "/run") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/intelligence/analysis/web-monitors/"), "/run")
		if id == "" || strings.Contains(id, "/") {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid website monitor identifier"})
			return true
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		monitor, change, err := osint.RunWebsiteMonitor(ctx, store, id)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"monitor": monitor, "change": change})
		return true
	case r.URL.Path == "/v1/intelligence/analysis/search-monitors" && r.Method == http.MethodPost:
		var request struct {
			Name         string                     `json:"name"`
			Search       intelligence.SearchRequest `json:"search"`
			MinimumScore int                        `json:"minimum_score"`
		}
		if !decodeAnalysisRequest(w, r, &request) {
			return true
		}
		monitor, err := store.CreateSearchMonitor(request.Name, request.Search, request.MinimumScore)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusCreated, monitor)
		return true
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/analysis/search-monitors/") && strings.HasSuffix(r.URL.Path, "/run") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/intelligence/analysis/search-monitors/"), "/run")
		if id == "" || strings.Contains(id, "/") {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid search monitor identifier"})
			return true
		}
		alerts, err := store.RunSearchMonitor(id)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"alerts": alerts, "count": len(alerts)})
		return true
	case r.URL.Path == "/v1/intelligence/analysis/claims" && r.Method == http.MethodPost:
		var request intelligence.Claim
		if !decodeAnalysisRequest(w, r, &request) {
			return true
		}
		claim, err := store.AddClaim(request)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusCreated, claim)
		return true
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/analysis/claims/") && strings.HasSuffix(r.URL.Path, "/review") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/intelligence/analysis/claims/"), "/review")
		if id == "" || strings.Contains(id, "/") {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid claim identifier"})
			return true
		}
		var request struct {
			Status string `json:"status"`
			Actor  string `json:"actor"`
			Reason string `json:"reason"`
		}
		if !decodeAnalysisRequest(w, r, &request) {
			return true
		}
		claim, err := store.ReviewClaim(id, request.Status, request.Actor, request.Reason)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, claim)
		return true
	case r.URL.Path == "/v1/intelligence/analysis/lineage" && r.Method == http.MethodGet:
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"source_lineage": store.GetSnapshot().SourceLineage})
		return true
	case r.URL.Path == "/v1/intelligence/analysis/lineage" && r.Method == http.MethodPost:
		var request intelligence.SourceLineage
		if !decodeAnalysisRequest(w, r, &request) {
			return true
		}
		lineage, err := store.AddSourceLineage(request)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusCreated, lineage)
		return true
	case r.URL.Path == "/v1/intelligence/analysis/hypotheses" && r.Method == http.MethodGet:
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"hypotheses": store.GetSnapshot().Hypotheses})
		return true
	case r.URL.Path == "/v1/intelligence/analysis/hypotheses" && r.Method == http.MethodPost:
		var request intelligence.Hypothesis
		if !decodeAnalysisRequest(w, r, &request) {
			return true
		}
		hypothesis, err := store.CreateHypothesis(request)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusCreated, hypothesis)
		return true
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/analysis/hypotheses/") && strings.HasSuffix(r.URL.Path, "/confidence") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/intelligence/analysis/hypotheses/"), "/confidence")
		if id == "" || strings.Contains(id, "/") {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid hypothesis identifier"})
			return true
		}
		var request struct {
			Confidence int    `json:"confidence"`
			Reason     string `json:"reason"`
			Actor      string `json:"actor"`
		}
		if !decodeAnalysisRequest(w, r, &request) {
			return true
		}
		hypothesis, err := store.UpdateHypothesisConfidence(id, request.Confidence, request.Reason, request.Actor)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, hypothesis)
		return true
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/analysis/hypotheses/") && strings.HasSuffix(r.URL.Path, "/framework") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/intelligence/analysis/hypotheses/"), "/framework")
		if id == "" || strings.Contains(id, "/") {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid hypothesis identifier"})
			return true
		}
		var request struct {
			Indicators       []intelligence.HypothesisIndicator `json:"indicators"`
			AlternativeIDs   []string                           `json:"alternative_hypothesis_ids"`
			InformationGaps  []string                           `json:"information_gap_ids"`
			ChangeConditions []string                           `json:"change_conditions"`
			Actor            string                             `json:"actor"`
		}
		if !decodeAnalysisRequest(w, r, &request) {
			return true
		}
		hypothesis, err := store.UpdateHypothesisFramework(id, request.Indicators, request.AlternativeIDs, request.InformationGaps, request.ChangeConditions, request.Actor)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, hypothesis)
		return true
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/analysis/hypotheses/") && strings.HasSuffix(r.URL.Path, "/evidence") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/intelligence/analysis/hypotheses/"), "/evidence")
		if id == "" || strings.Contains(id, "/") {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid hypothesis identifier"})
			return true
		}
		var request struct {
			EvidenceID    string `json:"evidence_id"`
			Compatibility int    `json:"compatibility"`
			Diagnosticity int    `json:"diagnosticity"`
			Reason        string `json:"reason"`
			Actor         string `json:"actor"`
		}
		if !decodeAnalysisRequest(w, r, &request) {
			return true
		}
		assessment, err := store.AssessHypothesisEvidence(id, request.EvidenceID, request.Compatibility, request.Diagnosticity, request.Reason, request.Actor)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusCreated, assessment)
		return true
	case r.URL.Path == "/v1/intelligence/analysis/ach" && r.Method == http.MethodGet:
		matrix, err := store.BuildACHMatrix(r.URL.Query().Get("case_id"))
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, matrix)
		return true
	case r.URL.Path == "/v1/intelligence/analysis/gaps" && r.Method == http.MethodGet:
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"information_gaps": store.GetSnapshot().InformationGaps})
		return true
	case r.URL.Path == "/v1/intelligence/analysis/gaps" && r.Method == http.MethodPost:
		var request intelligence.InformationGap
		if !decodeAnalysisRequest(w, r, &request) {
			return true
		}
		gap, err := store.CreateInformationGap(request)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusCreated, gap)
		return true
	case r.URL.Path == "/v1/intelligence/analysis/collection-plans" && r.Method == http.MethodGet:
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"collection_plans": store.GetSnapshot().CollectionPlans})
		return true
	case r.URL.Path == "/v1/intelligence/analysis/collection-plans" && r.Method == http.MethodPost:
		var request intelligence.CollectionPlan
		if !decodeAnalysisRequest(w, r, &request) {
			return true
		}
		plan, err := store.CreateCollectionPlan(request)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusCreated, plan)
		return true
	case r.URL.Path == "/v1/intelligence/analysis/custody" && r.Method == http.MethodGet:
		events := store.GetSnapshot().CustodyEvents
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"events": events, "valid": intelligence.VerifyCustodyChain(events)})
		return true
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/analysis/case-exports/") && r.Method == http.MethodGet:
		caseID := strings.TrimPrefix(r.URL.Path, "/v1/intelligence/analysis/case-exports/")
		if !security.SafeResourceIDPattern.MatchString(caseID) || strings.Contains(caseID, "/") {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid case identifier"})
			return true
		}
		archive, manifest, err := store.ExportCaseEvidence(caseID)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="aethel-evidence-export.zip"`)
		w.Header().Set("X-Aethel-Export-Signature", manifest.ManifestSignature)
		w.Header().Set("Content-Length", strconv.Itoa(len(archive)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archive)
		return true
	case r.URL.Path == "/v1/intelligence/analysis/domain-investigation" && r.Method == http.MethodPost:
		var request struct {
			Domain string `json:"domain"`
		}
		if !decodeAnalysisRequest(w, r, &request) {
			return true
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		finding, err := osint.InvestigateDomain(ctx, request.Domain)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, finding)
		return true
	case r.URL.Path == "/v1/intelligence/analysis/entity-resolution" && r.Method == http.MethodGet:
		snapshot := store.GetSnapshot()
		intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{
			"resolved_entities": snapshot.ResolvedEntities, "candidates": snapshot.ResolutionCandidates,
			"decisions": snapshot.ResolutionDecisions, "versions": snapshot.EntityVersions,
		})
		return true
	case r.URL.Path == "/v1/intelligence/analysis/entity-resolution/candidates" && r.Method == http.MethodPost:
		var request struct {
			CaseID        string `json:"case_id"`
			LeftEntityID  string `json:"left_entity_id"`
			RightEntityID string `json:"right_entity_id"`
		}
		if !decodeAnalysisRequest(w, r, &request) {
			return true
		}
		candidate, err := store.ProposeEntityResolution(request.CaseID, request.LeftEntityID, request.RightEntityID)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusCreated, candidate)
		return true
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/analysis/entity-resolution/candidates/") && strings.HasSuffix(r.URL.Path, "/review") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/intelligence/analysis/entity-resolution/candidates/"), "/review")
		if id == "" || strings.Contains(id, "/") {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid resolution candidate identifier"})
			return true
		}
		var request struct {
			Action string `json:"action"`
			Actor  string `json:"actor"`
			Reason string `json:"reason"`
		}
		if !decodeAnalysisRequest(w, r, &request) {
			return true
		}
		decision, err := store.ReviewEntityResolution(id, request.Action, request.Actor, request.Reason)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, decision)
		return true
	case strings.HasPrefix(r.URL.Path, "/v1/intelligence/analysis/entity-resolution/resolved/") && strings.HasSuffix(r.URL.Path, "/aliases") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/intelligence/analysis/entity-resolution/resolved/"), "/aliases")
		if id == "" || strings.Contains(id, "/") {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid resolved entity identifier"})
			return true
		}
		var request struct {
			Alias intelligence.EntityAlias `json:"alias"`
			Actor string                   `json:"actor"`
		}
		if !decodeAnalysisRequest(w, r, &request) {
			return true
		}
		resolved, err := store.AddResolvedEntityAlias(id, request.Alias, request.Actor)
		if err != nil {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return true
		}
		intelligence.WriteIntelJSON(w, http.StatusOK, resolved)
		return true
	default:
		intelligence.WriteIntelJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "analysis endpoint or method not allowed"})
		return true
	}
}

func buildOperatorWorkspace(snapshot intelligence.StoreState) map[string]any {
	const maximumRecords = 250
	evidence := make([]map[string]any, 0, maximumRecords)
	cases := make([]map[string]any, 0, len(snapshot.Cases))
	for caseIndex := len(snapshot.Cases) - 1; caseIndex >= 0 && len(cases) < maximumRecords; caseIndex-- {
		caseRecord := snapshot.Cases[caseIndex]
		cases = append(cases, map[string]any{
			"id": caseRecord.ID, "title": caseRecord.Title, "purpose": caseRecord.Purpose,
			"classification": caseRecord.Classification, "status": caseRecord.Status,
			"created_at": caseRecord.CreatedAt, "evidence_count": len(caseRecord.Evidence),
			"entity_count": len(caseRecord.Entities), "relation_count": len(caseRecord.Relations),
		})
	}
	for caseIndex := len(snapshot.Cases) - 1; caseIndex >= 0 && len(evidence) < maximumRecords; caseIndex-- {
		caseRecord := snapshot.Cases[caseIndex]
		for evidenceIndex := len(caseRecord.Evidence) - 1; evidenceIndex >= 0 && len(evidence) < maximumRecords; evidenceIndex-- {
			item := caseRecord.Evidence[evidenceIndex]
			evidence = append(evidence, map[string]any{
				"id": item.ID, "case_id": caseRecord.ID, "source_id": item.SourceID,
				"excerpt": item.Excerpt, "sha256": item.SHA256, "raw_sha256": item.RawSHA256,
				"normalized_sha256": item.NormalizedSHA256, "capture_scope": item.CaptureScope,
				"validation_status": item.ValidationStatus, "sealed": item.Sealed,
				"collected_at": item.CollectedAt, "snapshot_id": item.SnapshotID,
			})
		}
	}
	return map[string]any{
		"sources":          tailRecords(snapshot.Sources, maximumRecords),
		"events":           tailRecords(snapshot.Events, maximumRecords),
		"claims":           tailRecords(snapshot.Claims, maximumRecords),
		"cases":            tailRecords(cases, maximumRecords),
		"alerts":           tailRecords(snapshot.Alerts, maximumRecords),
		"collection_plans": tailRecords(snapshot.CollectionPlans, maximumRecords),
		"lineage":          tailRecords(snapshot.SourceLineage, maximumRecords),
		"evidence":         evidence,
		"website_monitors": tailRecords(snapshot.WebsiteMonitors, maximumRecords),
		"website_changes":  tailRecords(snapshot.WebsiteChanges, maximumRecords),
	}
}

func tailRecords[T any](records []T, maximum int) []T {
	if len(records) <= maximum {
		return records
	}
	return records[len(records)-maximum:]
}

func decodeAnalysisRequest(w http.ResponseWriter, r *http.Request, destination any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, analysisRequestLimit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid analysis payload"})
		return false
	}
	return true
}

func handleImageMatch(w http.ResponseWriter, r *http.Request, store *intelligence.Store) {
	r.Body = http.MaxBytesReader(w, r.Body, (8<<20)+(64<<10))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid or oversized multipart image payload"})
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, _, err := r.FormFile("image")
	if err != nil {
		intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "image file is required"})
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, (8<<20)+1))
	if err != nil || len(raw) > 8<<20 {
		intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "image size boundary violation"})
		return
	}
	index := r.FormValue("index") != "false"
	fingerprint, matches, err := store.MatchImage(raw, r.FormValue("case_id"), r.FormValue("source_id"), r.FormValue("label"), index)
	if err != nil {
		intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{"query": fingerprint, "matches": matches, "indexed": index})
}

func handleDocumentImport(w http.ResponseWriter, r *http.Request, store *intelligence.Store) {
	r.Body = http.MaxBytesReader(w, r.Body, (8<<20)+(64<<10))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid or oversized document payload"})
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, err := r.FormFile("document")
	if err != nil {
		intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "document file is required"})
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, (8<<20)+1))
	if err != nil || len(raw) > 8<<20 {
		intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "document size boundary violation"})
		return
	}
	filename := r.FormValue("filename")
	if filename == "" {
		filename = header.Filename
	}
	result, err := store.ImportDocument(raw, r.FormValue("format"), r.FormValue("source_id"), filename, r.FormValue("domain"))
	if err != nil {
		intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	intelligence.WriteIntelJSON(w, http.StatusCreated, result)
}
