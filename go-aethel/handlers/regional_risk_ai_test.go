package handlers

import (
	"strings"
	"testing"
	"time"

	"go-aethel/intelligence"
)

func TestParseAIRegionalRiskJSON_WrappedAndArray(t *testing.T) {
	wrapped := `{"regions":[{"region_id":"GERMANY","overall_risk":42,"geopolitical_risk":30,"conflict_risk":10,"cyber_risk":20,"infrastructure_risk":15,"economic_risk":12,"primary_drivers":["x"],"trend":"up","narrative":"Lage ruhig.","confidence":80}]}`
	dtos, err := ParseAIRegionalRiskJSON(wrapped)
	if err != nil || len(dtos) != 1 {
		t.Fatalf("wrapped: %#v %v", dtos, err)
	}
	if dtos[0].RegionID != "GERMANY" || dtos[0].OverallRisk != 42 {
		t.Fatalf("dto: %+v", dtos[0])
	}

	arr := `[{"region_id":"UK","overall_risk":11,"trend":"stable","narrative":"ok"}]`
	dtos2, err := ParseAIRegionalRiskJSON(arr)
	if err != nil || len(dtos2) != 1 || dtos2[0].RegionID != "UK" {
		t.Fatalf("array: %#v %v", dtos2, err)
	}

	// Markdown fence (common DeepSeek/Groq wrapper)
	fenced := "```json\n" + wrapped + "\n```"
	dtos3, err := ParseAIRegionalRiskJSON(fenced)
	if err != nil || len(dtos3) != 1 {
		t.Fatalf("fenced: %#v %v", dtos3, err)
	}
}

func TestMergeAIRegionalScores_TTLAndClamp(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	dtos := []aiRegionScoreDTO{{
		RegionID: "germany", OverallRisk: 150, CyberRisk: -5,
		Trend: "rising", Narrative: "Test", Confidence: 200,
		PrimaryDrivers: []string{"a", "b"},
	}}
	scores := MergeAIRegionalScores(dtos, "model-x", "ai", now)
	rs, ok := scores["GERMANY"]
	if !ok {
		t.Fatal("missing GERMANY")
	}
	if rs.OverallRisk != 100 || rs.CyberRisk != 0 {
		t.Fatalf("clamp failed: %+v", rs)
	}
	if rs.Trend != "up" {
		t.Fatalf("trend: %s", rs.Trend)
	}
	if rs.EvaluationSource != "ai" || rs.AIModelID != "model-x" {
		t.Fatalf("meta: %+v", rs)
	}
	if !rs.NextRefreshAt.Equal(now.Add(intelligence.RegionalAIRiskTTL)) {
		t.Fatalf("next refresh: %v want +5h", rs.NextRefreshAt)
	}
	if rs.Confidence != 100 {
		t.Fatalf("conf: %d", rs.Confidence)
	}
}

func TestRegionalAIRiskTTLIsFiveHours(t *testing.T) {
	if intelligence.RegionalAIRiskTTL != 5*time.Hour {
		t.Fatalf("TTL must be 5h, got %s", intelligence.RegionalAIRiskTTL)
	}
}

func TestBuildRegionalRiskContext_EmptyStoreSafe(t *testing.T) {
	// nil store
	if BuildRegionalRiskContext(nil, nil) != "{}" {
		t.Fatal("nil store")
	}
	// An empty store must not leak algorithmic baseline regions into AI context.
	st := intelligence.NewStore(t.TempDir()+"/intel.json", nil)
	ctx := BuildRegionalRiskContext(st, []intelligence.RegionalRiskData{{RegionID: "GERMANY", RegionName: "Germany", OverallRisk: 12}})
	if ctx != "[]" || strings.Contains(ctx, "deterministic_baseline") {
		t.Fatalf("empty context must contain no synthetic regions: %s", ctx)
	}
}

func TestBuildRegionalRiskContextExcludesNaturalHazards(t *testing.T) {
	store := intelligence.NewStore(t.TempDir()+"/hazard-isolation.json", nil)
	now := time.Now().UTC()
	store.IngestObservation(intelligence.Observation{
		ID: "news-berlin", SourceID: "reuters-world", RawText: "Government update in Berlin",
		Latitude: 52.52, Longitude: 13.405, ObservedAt: now, Domain: "geo",
	})
	store.IngestObservation(intelligence.Observation{
		ID: "quake-berlin", SourceID: "usgs-earthquakes", RawText: "[earthquake] M 4.2 near Berlin",
		Latitude: 52.52, Longitude: 13.405, ObservedAt: now, Domain: "geo",
	})
	context := BuildRegionalRiskContext(store, nil)
	if !strings.Contains(context, "Government update in Berlin") {
		t.Fatalf("news signal missing from regional context: %s", context)
	}
	if strings.Contains(context, "quake-berlin") || strings.Contains(context, "[earthquake]") {
		t.Fatalf("natural hazard leaked into automatic regional AI context: %s", context)
	}
}

func TestRiskScoresToRegionalData_PublishesOnlyAISubset(t *testing.T) {
	now := time.Date(2026, 7, 29, 16, 0, 0, 0, time.UTC)
	scores := MergeAIRegionalScores([]aiRegionScoreDTO{
		{RegionID: "RUSSIA", OverallRisk: 77, Trend: "up", Narrative: "Kriegskontext", Confidence: 80},
		{RegionID: "IR", OverallRisk: 71, Trend: "up", Narrative: "Eskalation", Confidence: 75}, // alias → IRAN
	}, "deepseek/deepseek-v4-flash", "ai", now)

	baseline := []intelligence.RegionalRiskData{}
	for _, e := range intelligence.RegionalRiskCatalog() {
		baseline = append(baseline, intelligence.RegionalRiskData{
			RegionID: e.ID, RegionName: e.Name, OverallRisk: 11, Trend: "stable",
			EvaluationSource: "deterministic",
		})
	}
	scores["GERMANY"] = intelligence.RiskScore{OverallRisk: 11, EvaluationSource: "hybrid"}

	out := riskScoresToRegionalData(scores, baseline)
	if len(out) != 2 {
		t.Fatalf("API slice len=%d want two explicit AI regions", len(out))
	}
	found := map[string]intelligence.RegionalRiskData{}
	for _, r := range out {
		found[r.RegionID] = r
		if r.RegionName == "" {
			t.Fatalf("empty name for %s", r.RegionID)
		}
	}
	if found["RUSSIA"].OverallRisk != 77 || found["RUSSIA"].AINarrative != "Kriegskontext" {
		t.Fatalf("RUSSIA: %+v", found["RUSSIA"])
	}
	if found["IRAN"].OverallRisk != 71 {
		t.Fatalf("IRAN alias merge: %+v", found["IRAN"])
	}
	if _, exists := found["GERMANY"]; exists {
		t.Fatalf("legacy hybrid score leaked into payload: %+v", found["GERMANY"])
	}
}

func TestFinalizeRegionalRiskPayload_BaselineURLUpgradesTitleOnlyCollect(t *testing.T) {
	// SharedIntelStore Event has no SourceURL — Collect yields title-only.
	// Baseline (IntelligenceEvent path) has the same title + real URL → merge must keep URL.
	st := intelligence.NewStore(t.TempDir()+"/refs-upgrade.json", nil)
	// Inject Iran-bbox event (title only path via Collect)
	st.IngestObservation(intelligence.Observation{
		ID: "obs1", SourceID: "feed-a", RawText: "Strike reported near Tehran",
		Latitude: 35.7, Longitude: 51.4, ObservedAt: time.Now().UTC(),
	})
	// Also put a classified Event so Collect hits Events path
	// Use Ingest if available — otherwise apply via GetSnapshot-style: score apply not needed.
	// Store.IngestObservation promotes to event in many pipelines; Collect also scans Observations.

	baseline := []intelligence.RegionalRiskData{{
		RegionID: "IRAN", RegionName: "Iran", OverallRisk: 70, Trend: "up",
		EvaluationSource: "deterministic",
		References: []intelligence.RiskReference{{
			Title:  "Strike reported near Tehran",
			URL:    "https://www.example.com/iran-strike",
			Source: "tagesschau",
		}},
	}}
	// Fill full catalog shells so finalize doesn't panic on empty
	for _, e := range intelligence.RegionalRiskCatalog() {
		if e.ID == "IRAN" {
			continue
		}
		baseline = append(baseline, intelligence.RegionalRiskData{
			RegionID: e.ID, RegionName: e.Name, Trend: "stable", EvaluationSource: "deterministic",
		})
	}
	scores := map[string]intelligence.RiskScore{
		"IRAN": {OverallRisk: 70, EvaluationSource: "ai", Trend: "up"},
	}
	out := finalizeRegionalRiskPayload(scores, baseline, st)
	var iran *intelligence.RegionalRiskData
	for i := range out {
		if out[i].RegionID == "IRAN" {
			iran = &out[i]
			break
		}
	}
	if iran == nil {
		t.Fatal("IRAN missing from payload")
	}
	foundURL := false
	for _, ref := range iran.References {
		if strings.Contains(strings.ToLower(ref.Title), "strike reported near tehran") {
			if ref.URL == "https://www.example.com/iran-strike" {
				foundURL = true
			}
		}
	}
	if !foundURL {
		t.Fatalf("baseline SourceURL must appear after title-only Collect merge; refs=%+v", iran.References)
	}
}

func TestEnsureAIRegionalRisks_FailsClosedWithoutAI(t *testing.T) {
	// No state/providers: no score is safer than an algorithmic substitute.
	prev := intelligence.SharedIntelStore
	st := intelligence.NewStore(t.TempDir()+"/intel-risk.json", nil)
	intelligence.SharedIntelStore = st
	t.Cleanup(func() { intelligence.SharedIntelStore = prev })

	risks, err := EnsureAIRegionalRisks(true, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(risks) != 0 {
		t.Fatalf("expected no unauthorised fallback scores, got %v", regionIDs(risks))
	}
}

func TestEvidenceBoundRegionalDTOs_RequiresSameRegionEvidence(t *testing.T) {
	store := intelligence.NewStore(t.TempDir()+"/evidence-bound.json", nil)
	now := time.Now().UTC()
	store.IngestObservation(intelligence.Observation{
		ID: "berlin-news", SourceID: "news", RawText: "Policy crisis in Berlin",
		Latitude: 52.52, Longitude: 13.405, ObservedAt: now,
	})
	input := []aiRegionScoreDTO{
		{RegionID: "GERMANY", OverallRisk: 44, EvidenceIDs: []string{"berlin-news"}},
		{RegionID: "IRAN", OverallRisk: 90, EvidenceIDs: []string{"berlin-news"}},
		{RegionID: "RUSSIA", OverallRisk: 80},
	}
	bound := evidenceBoundRegionalDTOs(input, store)
	if len(bound) != 1 || bound[0].RegionID != "GERMANY" || len(bound[0].EvidenceIDs) != 1 {
		t.Fatalf("unexpected evidence binding: %+v", bound)
	}
}

func regionIDs(risks []intelligence.RegionalRiskData) []string {
	out := make([]string, len(risks))
	for i, r := range risks {
		out[i] = r.RegionID
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
