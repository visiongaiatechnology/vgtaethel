// STATUS: DIAMANT VGT SUPREME
// AI regional risk evaluation for Global Watch — refresh at most every 5 hours.
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"go-aethel/intelligence"
)

// Regional risk AI evaluation is serialized so concurrent HUD polls do not stampede the model.
var regionalAIEvalMu sync.Mutex
var regionalAILastFailedAttempt time.Time

type aiRegionScoreDTO struct {
	RegionID           string   `json:"region_id"`
	OverallRisk        float64  `json:"overall_risk"`
	GeopoliticalRisk   float64  `json:"geopolitical_risk"`
	ConflictRisk       float64  `json:"conflict_risk"`
	CyberRisk          float64  `json:"cyber_risk"`
	InfrastructureRisk float64  `json:"infrastructure_risk"`
	EconomicRisk       float64  `json:"economic_risk"`
	PrimaryDrivers     []string `json:"primary_drivers"`
	Trend              string   `json:"trend"`
	Narrative          string   `json:"narrative"`
	Confidence         int      `json:"confidence"`
	EvidenceIDs        []string `json:"evidence_ids"`
}

type aiRegionalRiskResponse struct {
	Regions []aiRegionScoreDTO `json:"regions"`
}

// defaultAIRegions is an alias over the shared catalog (HUD + AI + baseline must not diverge).
func defaultAIRegions() []intelligence.RegionalRiskCatalogEntry {
	return intelligence.RegionalRiskCatalog()
}

func clampRisk01(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func normalizeTrend(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "up", "rising", "increasing":
		return "up"
	case "down", "falling", "decreasing":
		return "down"
	default:
		return "stable"
	}
}

// BuildRegionalRiskContext packs only current source evidence per catalog region.
// The baseline argument remains for API compatibility but is intentionally ignored.
func BuildRegionalRiskContext(store *intelligence.Store, baseline []intelligence.RegionalRiskData) string {
	if store == nil {
		return "{}"
	}
	snap := store.GetSnapshot()
	type regionPack struct {
		RegionID   string           `json:"region_id"`
		RegionName string           `json:"region_name"`
		Events     []map[string]any `json:"events"`
	}
	_ = baseline
	packs := make([]regionPack, 0, len(defaultAIRegions()))
	now := time.Now().UTC()
	for _, r := range defaultAIRegions() {
		evs := make([]map[string]any, 0, 10)
		for _, ev := range snap.Events {
			if intelligence.DetectNaturalHazard(ev.SourceID, ev.Title, ev.Summary) != "" {
				continue
			}
			if !intelligence.PointInCatalogBBox(r, ev.Latitude, ev.Longitude) {
				continue
			}
			if !ev.ObservedAt.IsZero() && now.Sub(ev.ObservedAt) > 7*24*time.Hour {
				continue
			}
			evs = append(evs, map[string]any{
				"evidence_id": ev.ID,
				"title":       intelligence.TruncateIntel(ev.Title, 160),
				"summary":     intelligence.TruncateIntel(ev.Summary, 220),
				"severity":    ev.Severity,
				"domain":      ev.Domain,
				"lat":         ev.Latitude,
				"lon":         ev.Longitude,
				"observed":    ev.ObservedAt.UTC().Format(time.RFC3339),
				"confidence":  ev.Confidence,
			})
			if len(evs) >= 10 {
				break
			}
		}
		if len(evs) < 6 {
			for _, obs := range snap.Observations {
				if intelligence.DetectNaturalHazard(obs.SourceID, obs.RawText, "") != "" {
					continue
				}
				if !intelligence.PointInCatalogBBox(r, obs.Latitude, obs.Longitude) {
					continue
				}
				if !obs.ObservedAt.IsZero() && now.Sub(obs.ObservedAt) > 7*24*time.Hour {
					continue
				}
				evs = append(evs, map[string]any{
					"evidence_id": obs.ID,
					"title":       intelligence.TruncateIntel(obs.RawText, 120),
					"summary":     intelligence.TruncateIntel(obs.RawText, 200),
					"source":      obs.SourceID,
					"lat":         obs.Latitude,
					"lon":         obs.Longitude,
					"observed":    obs.ObservedAt.UTC().Format(time.RFC3339),
					"raw":         true,
				})
				if len(evs) >= 10 {
					break
				}
			}
		}
		if len(evs) > 0 {
			packs = append(packs, regionPack{RegionID: r.ID, RegionName: r.Name, Events: evs})
		}
	}
	raw, _ := json.Marshal(packs)
	return string(raw)
}

// catalogRegionIDListCSV for the AI prompt.
func catalogRegionIDListCSV() string {
	ids := intelligence.RegionalRiskCatalogIDs()
	return strings.Join(ids, ", ")
}

// ParseAIRegionalRiskJSON extracts the regions array from model text (shipped path).
func ParseAIRegionalRiskJSON(text string) ([]aiRegionScoreDTO, error) {
	text = strings.TrimSpace(text)
	// Strip common markdown fences from Groq/DeepSeek chatty wrappers.
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```JSON")
	text = strings.TrimPrefix(text, "```")
	if idx := strings.LastIndex(text, "```"); idx > 0 {
		text = strings.TrimSpace(text[:idx])
	}
	payload, err := regionalRiskJSONValue(text)
	if err != nil {
		return nil, err
	}
	var wrapped aiRegionalRiskResponse
	if err := json.Unmarshal(payload, &wrapped); err == nil && len(wrapped.Regions) > 0 {
		return wrapped.Regions, nil
	}
	// Allow bare array
	var arr []aiRegionScoreDTO
	if err := json.Unmarshal(payload, &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}
	// Nested loose search: find "regions" key object
	if i := strings.Index(string(payload), `"regions"`); i >= 0 {
		sub := string(payload[i:])
		if lb := strings.Index(sub, "["); lb >= 0 {
			if rb := strings.LastIndex(sub, "]"); rb > lb {
				var arr2 []aiRegionScoreDTO
				if err := json.Unmarshal([]byte(sub[lb:rb+1]), &arr2); err == nil && len(arr2) > 0 {
					return arr2, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("model did not return regional risk array")
}

// regionalRiskJSONValue extracts exactly one bounded JSON object or array from
// optional model prose. Array roots must be considered before the first object
// member; otherwise a bare array is truncated to its first element.
func regionalRiskJSONValue(text string) ([]byte, error) {
	objectStart := strings.Index(text, "{")
	arrayStart := strings.Index(text, "[")
	start := objectStart
	if arrayStart >= 0 && (start < 0 || arrayStart < start) {
		start = arrayStart
	}
	if start < 0 {
		return nil, fmt.Errorf("model did not return regional risk JSON")
	}
	decoder := json.NewDecoder(strings.NewReader(text[start:]))
	var payload json.RawMessage
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode regional risk JSON: %w", err)
	}
	if len(payload) == 0 || len(payload) > 64<<10 {
		return nil, fmt.Errorf("regional risk JSON size boundary violation")
	}
	return payload, nil
}

// pickConfiguredRiskModel chooses a usable chat model: preferred → any configured provider model.
func pickConfiguredRiskModel(preferred string) (string, error) {
	if state == nil || state.providers == nil {
		return "", fmt.Errorf("provider registry unavailable")
	}
	// Prefer explicit request, then known fast models, then ANY configured catalog model.
	candidates := []string{
		strings.TrimSpace(preferred),
		"openai/gpt-oss-120b",
		"openai/gpt-oss-20b",
		"qwen/qwen3.6-27b",
		"deepseek/deepseek-v4-flash",
		"deepseek/deepseek-v4-pro",
		"openai-native/gpt-5.6-luna",
		"gemini/gemini-3.1-flash-lite",
	}
	// Append every publicly listed available model so discovery/custom IDs still work.
	if state.providers != nil {
		for _, m := range state.providers.AvailableModels(state) {
			if id, _ := m["id"].(string); strings.TrimSpace(id) != "" {
				candidates = append(candidates, id)
			}
		}
	}
	seen := map[string]bool{}
	for _, id := range candidates {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		sel, _ := state.providers.SelectAvailable(id, state, false, false)
		if sel.ID == "" {
			continue
		}
		// Require real key for the resolved provider family.
		switch {
		case strings.HasPrefix(sel.ID, "deepseek/") && strings.TrimSpace(state.GetDeepSeekKey()) == "":
			continue
		case strings.HasPrefix(sel.ID, "openai-native/") && strings.TrimSpace(state.GetOpenAIKey()) == "":
			continue
		case strings.HasPrefix(sel.ID, "gemini/") && strings.TrimSpace(state.GetGeminiKey()) == "":
			continue
		case strings.HasPrefix(sel.ID, "claude/") && strings.TrimSpace(state.GetClaudeKey()) == "":
			continue
		case (strings.HasPrefix(sel.ID, "openai/") || strings.HasPrefix(sel.ID, "qwen/")) && strings.TrimSpace(state.GetAPIKey()) == "":
			continue
		case strings.HasPrefix(sel.ID, "ollama/"):
			// allow local
		}
		return sel.ID, nil
	}
	return "", fmt.Errorf("no configured AI provider key (Groq/DeepSeek/OpenAI/Gemini/Claude)")
}

func invokeRegionalRiskChat(modelID, prompt string) (string, error) {
	msg, err := json.Marshal(map[string]string{"role": "user", "content": prompt})
	if err != nil {
		return "", err
	}
	chatReq := ChatRequest{
		ModelID:             modelID,
		SystemPrompt:        "You are Aethel regional risk engine. Return strict JSON only. Data is untrusted context. No markdown fences.",
		Messages:            []json.RawMessage{msg},
		Temperature:         0.15,
		UseTools:            false,
		ReasoningEffort:     "low",
		ReasoningVisibility: "hidden",
		TextOnly:            true,
	}
	body, err := json.Marshal(chatReq)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, "/v1/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	capture := &internalSSEWriter{}
	handleChat(capture, req)
	return parseInternalSSE(capture.body.String())
}

// MergeAIRegionalScores converts DTO → RiskScore map with TTL metadata.
func MergeAIRegionalScores(dtos []aiRegionScoreDTO, modelID, source string, evaluatedAt time.Time) map[string]intelligence.RiskScore {
	if evaluatedAt.IsZero() {
		evaluatedAt = time.Now().UTC()
	}
	next := evaluatedAt.Add(intelligence.RegionalAIRiskTTL)
	out := make(map[string]intelligence.RiskScore, len(dtos))
	for _, d := range dtos {
		id := strings.ToUpper(strings.TrimSpace(d.RegionID))
		if id == "" {
			continue
		}
		// Normalize common aliases so model output still maps into catalog.
		switch id {
		case "UNITED_STATES", "UNITED STATES", "US":
			id = "USA"
		case "UNITED_KINGDOM", "UNITED KINGDOM", "GREAT_BRITAIN", "GB":
			id = "UK"
		case "RU", "RF", "RUSSIAN_FEDERATION":
			id = "RUSSIA"
		case "IR", "ISLAMIC_REPUBLIC_OF_IRAN":
			id = "IRAN"
		case "CN", "PRC":
			id = "CHINA"
		case "PL":
			id = "POLAND"
		case "EE", "LV", "LT", "BALTIC", "BALTIC_STATES":
			id = "BALTICS"
		}
		conf := d.Confidence
		if conf <= 0 {
			conf = 70
		}
		if conf > 100 {
			conf = 100
		}
		drivers := d.PrimaryDrivers
		if len(drivers) > 8 {
			drivers = drivers[:8]
		}
		out[id] = intelligence.RiskScore{
			OverallRisk:            clampRisk01(d.OverallRisk),
			GeopoliticalRisk:       clampRisk01(d.GeopoliticalRisk),
			ConflictRisk:           clampRisk01(d.ConflictRisk),
			CyberRisk:              clampRisk01(d.CyberRisk),
			InfrastructureRisk:     clampRisk01(d.InfrastructureRisk),
			EconomicRisk:           clampRisk01(d.EconomicRisk),
			FinancialRisk:          clampRisk01(d.EconomicRisk * 0.85),
			EnergyRisk:             clampRisk01(d.InfrastructureRisk * 0.65),
			SupplyChainRisk:        clampRisk01(d.EconomicRisk*0.75 + d.InfrastructureRisk*0.3),
			ClimateRisk:            10,
			PublicSafetyRisk:       clampRisk01(d.ConflictRisk * 0.55),
			DataFreshness:          80,
			InformationReliability: float64(conf),
			Confidence:             conf,
			Trend:                  normalizeTrend(d.Trend),
			LastUpdated:            evaluatedAt,
			PrimaryDrivers:         drivers,
			MissingData:            []string{"AI inference over open sources; not operator-verified"},
			EvaluationSource:       source,
			AINarrative:            strings.TrimSpace(d.Narrative),
			AIModelID:              modelID,
			AIEvaluatedAt:          evaluatedAt,
			NextRefreshAt:          next,
		}
	}
	return out
}

func baselineRegionalRisks() []intelligence.RegionalRiskData {
	if state != nil && state.intel != nil {
		base := state.intel.ComputeAllRegionalRisks()
		// Ensure full catalog even if ComputeAll is older binary path — re-fill.
		baseByID := map[string]intelligence.RegionalRiskData{}
		for _, b := range base {
			baseByID[strings.ToUpper(b.RegionID)] = b
		}
		return intelligence.FillCatalogFromScores(nil, baseByID, time.Now().UTC())
	}
	out := make([]intelligence.RegionalRiskData, 0, len(defaultAIRegions()))
	for _, r := range defaultAIRegions() {
		out = append(out, intelligence.RegionalRiskData{
			RegionID: r.ID, RegionName: r.Name, Trend: "stable",
			EvaluationSource: "deterministic",
		})
	}
	return out
}

func baselineByIDMap(baseline []intelligence.RegionalRiskData) map[string]intelligence.RegionalRiskData {
	m := make(map[string]intelligence.RegionalRiskData, len(baseline))
	for _, b := range baseline {
		m[strings.ToUpper(b.RegionID)] = b
	}
	return m
}

// riskScoresToRegionalData publishes only explicit AI assessments. Missing
// regions remain unscored instead of receiving synthetic zero/baseline values.
func riskScoresToRegionalData(scores map[string]intelligence.RiskScore, baseline []intelligence.RegionalRiskData) []intelligence.RegionalRiskData {
	now := time.Now().UTC()
	_ = baseline
	nameByID := make(map[string]string, len(defaultAIRegions()))
	for _, entry := range defaultAIRegions() {
		nameByID[entry.ID] = entry.Name
	}
	out := make([]intelligence.RegionalRiskData, 0, len(scores))
	for id, rs := range scores {
		id = strings.ToUpper(strings.TrimSpace(id))
		name, exists := nameByID[id]
		if !exists || rs.EvaluationSource != "ai" {
			continue
		}
		age := 0.0
		if !rs.AIEvaluatedAt.IsZero() {
			age = now.Sub(rs.AIEvaluatedAt).Hours()
		}
		out = append(out, intelligence.RegionalRiskData{
			RegionID: id, RegionName: name,
			OverallRisk: rs.OverallRisk, GeopoliticalRisk: rs.GeopoliticalRisk,
			ConflictRisk: rs.ConflictRisk, CyberRisk: rs.CyberRisk,
			InfrastructureRisk: rs.InfrastructureRisk, EconomicRisk: rs.EconomicRisk,
			PrimaryDrivers: append([]string(nil), rs.PrimaryDrivers...), Trend: rs.Trend,
			EvaluationSource: "ai", AINarrative: rs.AINarrative, AIModelID: rs.AIModelID,
			AIEvaluatedAt: rs.AIEvaluatedAt, NextRefreshAt: rs.NextRefreshAt, CacheAgeHours: age,
		})
	}
	// Sort by overall risk desc for HUD
	sort.Slice(out, func(i, j int) bool { return out[i].OverallRisk > out[j].OverallRisk })
	return out
}

func canonicalRegionalID(id string) string {
	id = strings.ToUpper(strings.TrimSpace(id))
	switch id {
	case "UNITED_STATES", "UNITED STATES", "US":
		return "USA"
	case "UNITED_KINGDOM", "UNITED KINGDOM", "GREAT_BRITAIN", "GB":
		return "UK"
	case "RU", "RF", "RUSSIAN_FEDERATION":
		return "RUSSIA"
	case "IR", "ISLAMIC_REPUBLIC_OF_IRAN":
		return "IRAN"
	case "CN", "PRC":
		return "CHINA"
	case "PL":
		return "POLAND"
	case "EE", "LV", "LT", "BALTIC", "BALTIC_STATES":
		return "BALTICS"
	default:
		return id
	}
}

// evidenceBoundRegionalDTOs rejects model scores that cannot cite at least one
// supplied event/observation ID belonging to the same catalog region.
func evidenceBoundRegionalDTOs(dtos []aiRegionScoreDTO, store *intelligence.Store) []aiRegionScoreDTO {
	if store == nil {
		return nil
	}
	snap := store.GetSnapshot()
	allowed := make(map[string]map[string]struct{}, len(defaultAIRegions()))
	now := time.Now().UTC()
	for _, region := range defaultAIRegions() {
		ids := map[string]struct{}{}
		for _, event := range snap.Events {
			if event.ID != "" && intelligence.DetectNaturalHazard(event.SourceID, event.Title, event.Summary) == "" &&
				intelligence.PointInCatalogBBox(region, event.Latitude, event.Longitude) &&
				(event.ObservedAt.IsZero() || now.Sub(event.ObservedAt) <= 7*24*time.Hour) {
				ids[event.ID] = struct{}{}
			}
		}
		for _, observation := range snap.Observations {
			if observation.ID != "" && intelligence.DetectNaturalHazard(observation.SourceID, observation.RawText, "") == "" &&
				intelligence.PointInCatalogBBox(region, observation.Latitude, observation.Longitude) &&
				(observation.ObservedAt.IsZero() || now.Sub(observation.ObservedAt) <= 7*24*time.Hour) {
				ids[observation.ID] = struct{}{}
			}
		}
		allowed[region.ID] = ids
	}
	out := make([]aiRegionScoreDTO, 0, len(dtos))
	for _, dto := range dtos {
		id := canonicalRegionalID(dto.RegionID)
		regionIDs, exists := allowed[id]
		if !exists || len(dto.EvidenceIDs) == 0 {
			continue
		}
		validIDs := make([]string, 0, len(dto.EvidenceIDs))
		seen := map[string]struct{}{}
		for _, evidenceID := range dto.EvidenceIDs {
			evidenceID = strings.TrimSpace(evidenceID)
			if _, valid := regionIDs[evidenceID]; !valid {
				continue
			}
			if _, duplicate := seen[evidenceID]; duplicate {
				continue
			}
			seen[evidenceID] = struct{}{}
			validIDs = append(validIDs, evidenceID)
		}
		if len(validIDs) == 0 {
			continue
		}
		dto.RegionID = id
		dto.EvidenceIDs = validIDs
		out = append(out, dto)
	}
	return out
}

// CollectRegionalRiskReferences gathers event/observation titles for popup references.
// SharedIntelStore Event has no SourceURL; URLs come from Evidence (by SourceID on Observations,
// or by title/excerpt match for Events). Never invents URLs.
func CollectRegionalRiskReferences(store *intelligence.Store) map[string][]intelligence.RiskReference {
	out := map[string][]intelligence.RiskReference{}
	if store == nil {
		return out
	}
	snap := store.GetSnapshot()
	now := time.Now().UTC()
	// Evidence URLs by source_id and by lowercased title fragment for Event upgrade.
	urlBySource := map[string]string{}
	urlByTitleKey := map[string]string{}
	for _, ev := range snap.Evidence {
		u := strings.TrimSpace(ev.URL)
		if u == "" {
			continue
		}
		if sid := strings.TrimSpace(ev.SourceID); sid != "" {
			urlBySource[sid] = u
		}
		// Index excerpt/title tokens for Event matching (Event has no SourceURL field).
		ex := strings.TrimSpace(ev.Excerpt)
		if ex != "" {
			key := strings.ToLower(intelligence.TruncateIntel(ex, 160))
			urlByTitleKey[key] = u
		}
	}
	// Helper: best-effort URL for a title from evidence excerpt containment / exact key.
	urlForTitle := func(title string) string {
		t := strings.TrimSpace(title)
		if t == "" {
			return ""
		}
		key := strings.ToLower(intelligence.TruncateIntel(t, 160))
		if u := urlByTitleKey[key]; u != "" {
			return u
		}
		// Containment: evidence excerpt contains title or vice versa (min length guard).
		if len(key) < 12 {
			return ""
		}
		for ek, u := range urlByTitleKey {
			if strings.Contains(ek, key) || strings.Contains(key, ek) {
				return u
			}
		}
		return ""
	}

	for _, r := range defaultAIRegions() {
		var eventRefs, obsRefs []intelligence.RiskReference
		for _, ev := range snap.Events {
			if intelligence.DetectNaturalHazard(ev.SourceID, ev.Title, ev.Summary) != "" {
				continue
			}
			if !intelligence.PointInCatalogBBox(r, ev.Latitude, ev.Longitude) {
				continue
			}
			if !ev.ObservedAt.IsZero() && now.Sub(ev.ObservedAt) > 14*24*time.Hour {
				continue
			}
			title := strings.TrimSpace(ev.Title)
			if title == "" {
				title = strings.TrimSpace(ev.Summary)
			}
			if title == "" {
				continue
			}
			ref := intelligence.RiskReference{
				Title:  intelligence.TruncateIntel(title, 160),
				Source: strings.TrimSpace(ev.Domain),
			}
			// Event has no SourceURL — attach Evidence URL when excerpt matches title.
			if u := urlForTitle(title); u != "" {
				ref.URL = intelligence.TruncateIntel(u, 1024)
			}
			eventRefs = append(eventRefs, ref)
			if len(eventRefs) >= 12 {
				break
			}
		}
		for _, obs := range snap.Observations {
			if intelligence.DetectNaturalHazard(obs.SourceID, obs.RawText, "") != "" {
				continue
			}
			if !intelligence.PointInCatalogBBox(r, obs.Latitude, obs.Longitude) {
				continue
			}
			if !obs.ObservedAt.IsZero() && now.Sub(obs.ObservedAt) > 14*24*time.Hour {
				continue
			}
			title := strings.TrimSpace(obs.RawText)
			if title == "" {
				continue
			}
			ref := intelligence.RiskReference{
				Title:  intelligence.TruncateIntel(title, 160),
				Source: strings.TrimSpace(obs.SourceID),
			}
			if u := urlBySource[obs.SourceID]; u != "" {
				ref.URL = intelligence.TruncateIntel(u, 1024)
			} else if u := urlForTitle(title); u != "" {
				ref.URL = intelligence.TruncateIntel(u, 1024)
			}
			obsRefs = append(obsRefs, ref)
			if len(obsRefs) >= 12 {
				break
			}
		}
		// Merge event + obs with URL upgrade (not first-wins title-only).
		merged := intelligence.MergeRiskReferences(12, eventRefs, obsRefs)
		if len(merged) > 0 {
			out[r.ID] = merged
		}
	}
	return out
}

func finalizeRegionalRiskPayload(scores map[string]intelligence.RiskScore, baseline []intelligence.RegionalRiskData, store *intelligence.Store) []intelligence.RegionalRiskData {
	data := riskScoresToRegionalData(scores, baseline)
	collected := CollectRegionalRiskReferences(store)
	// Fold baseline references AFTER collect so same-title SourceURLs upgrade title-only Events.
	refs := map[string][]intelligence.RiskReference{}
	for id, list := range collected {
		refs[id] = list
	}
	for _, b := range baseline {
		id := strings.ToUpper(b.RegionID)
		if len(b.References) == 0 {
			continue
		}
		refs[id] = intelligence.MergeRiskReferences(12, refs[id], b.References)
	}
	return intelligence.AttachCatalogReferences(data, refs)
}

// EnsureAIRegionalRisks returns regional risks, refreshing AI evaluation when cache is older than 5h.
// force=true bypasses TTL (operator refresh). Pure deterministic fallbacks are NOT treated as AI-fresh.
// Regions are returned only after an evidence-bound AI assessment.
func EnsureAIRegionalRisks(force bool, modelID string) ([]intelligence.RegionalRiskData, error) {
	regionalAIEvalMu.Lock()
	defer regionalAIEvalMu.Unlock()

	store := intelligence.SharedIntelStore
	baseline := baselineRegionalRisks()

	if store != nil && !force {
		if fresh, _ := store.AIRegionalRisksFresh(intelligence.RegionalAIRiskTTL); fresh {
			scores := store.GetRiskScores()
			// Legacy deterministic/hybrid entries are deliberately excluded.
			filtered := map[string]intelligence.RiskScore{}
			for id, rs := range scores {
				if rs.EvaluationSource == "ai" {
					filtered[strings.ToUpper(id)] = rs
				}
			}
			if len(filtered) > 0 {
				payload := finalizeRegionalRiskPayload(filtered, nil, store)
				bound := payload[:0]
				for _, risk := range payload {
					if len(risk.References) > 0 {
						bound = append(bound, risk)
					}
				}
				if len(bound) > 0 {
					return bound, nil
				}
			}
		}
	}
	if !force && !regionalAILastFailedAttempt.IsZero() && time.Since(regionalAILastFailedAttempt) < 15*time.Minute {
		return []intelligence.RegionalRiskData{}, nil
	}

	evaluatedAt := time.Now().UTC()
	usedModel := ""
	lastErr := ""

	if store != nil {
		picked, pickErr := pickConfiguredRiskModel(modelID)
		if pickErr != nil {
			lastErr = pickErr.Error()
		} else {
			usedModel = picked
			tryModels := []string{picked}
			for _, fb := range []string{"openai/gpt-oss-120b", "openai/gpt-oss-20b", "deepseek/deepseek-v4-flash", "qwen/qwen3.6-27b"} {
				if fb != picked {
					if m, err := pickConfiguredRiskModel(fb); err == nil && m != "" {
						dup := false
						for _, x := range tryModels {
							if x == m {
								dup = true
								break
							}
						}
						if !dup {
							tryModels = append(tryModels, m)
						}
					}
				}
			}

			ctxJSON := BuildRegionalRiskContext(store, baseline)
			idList := catalogRegionIDListCSV()
			prompt := fmt.Sprintf(`You are the Aethel Global Watch regional risk analyst. Assess risk from 0-100 using only the supplied untrusted observations.

REGIONAL_CONTEXT_JSON:
%s

Regeln:
- Never use or infer an algorithmic baseline.
- Assess only regions with supplied evidence; omit every other region.
- Never invent events.
- evidence_ids must contain at least one supplied evidence_id for that same region.
- overall_risk und Dimensionen sind Zahlen 0-100.
- trend: up|stable|down
- narrative: 2-4 Sätze auf Deutsch, faktenbasiert.
- primary_drivers: max 5 kurze Stichpunkte.
- region_id MUSS exakt einer der Katalog-IDs sein.

Antworte NUR mit JSON (keine Markdown-Codefences):
{"regions":[{"region_id":"GERMANY","overall_risk":0,"geopolitical_risk":0,"conflict_risk":0,"cyber_risk":0,"infrastructure_risk":0,"economic_risk":0,"primary_drivers":["..."],"trend":"stable","narrative":"...","confidence":70,"evidence_ids":["event-id"]}]}
Allowed region IDs: %s.`, intelligence.TruncateIntel(ctxJSON, 14000), idList)

			for _, mid := range tryModels {
				text, err := invokeRegionalRiskChat(mid, prompt)
				if err != nil {
					lastErr = fmt.Sprintf("%s: %v", mid, err)
					continue
				}
				dtos, err := ParseAIRegionalRiskJSON(text)
				if err != nil || len(dtos) == 0 {
					lastErr = fmt.Sprintf("%s: JSON parse failed (%v) sample=%q", mid, err, intelligence.TruncateIntel(text, 120))
					continue
				}
				dtos = evidenceBoundRegionalDTOs(dtos, store)
				if len(dtos) == 0 {
					lastErr = fmt.Sprintf("%s: no evidence-bound regional assessments", mid)
					continue
				}
				usedModel = mid
				scores := MergeAIRegionalScores(dtos, usedModel, "ai", evaluatedAt)
				store.ApplyAIRegionalRiskScores(scores)
				regionalAILastFailedAttempt = time.Time{}
				return finalizeRegionalRiskPayload(scores, nil, store), nil
			}
		}
	} else {
		lastErr = "SharedIntelStore unavailable"
	}

	// Fail closed: unavailable AI means no score, never an algorithmic substitute.
	regionalAILastFailedAttempt = evaluatedAt
	_ = lastErr
	return []intelligence.RegionalRiskData{}, nil
}

// handleRegionalRiskGET serves AI-evaluated regional risks (5h cache).
func handleRegionalRiskGET(w http.ResponseWriter, r *http.Request) {
	force := r.URL.Query().Get("refresh") == "1"
	modelID := strings.TrimSpace(r.URL.Query().Get("model_id"))
	risks, err := EnsureAIRegionalRisks(force, modelID)
	if err != nil {
		intelligence.WriteIntelJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if risks == nil {
		risks = []intelligence.RegionalRiskData{}
	}
	// Array form for /v1/intelligence/risk (legacy HUD)
	if strings.HasSuffix(r.URL.Path, "/risk") {
		intelligence.WriteIntelJSON(w, http.StatusOK, risks)
		return
	}
	// Object form for /v1/intelligence/risks
	intelligence.WriteIntelJSON(w, http.StatusOK, map[string]any{
		"risks":                risks,
		"catalog_ids":          intelligence.RegionalRiskCatalogIDs(),
		"evaluation_ttl_hours": intelligence.RegionalAIRiskTTL.Hours(),
		"refresh_hint":         "Pass ?refresh=1 to force AI re-evaluation (otherwise max every 5h).",
	})
}
