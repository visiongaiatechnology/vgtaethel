package intelligence

import (
	"strings"
	"testing"
	"time"
)

func TestRegionalRiskCatalog_IncludesHotspots(t *testing.T) {
	ids := map[string]bool{}
	for _, e := range RegionalRiskCatalog() {
		ids[e.ID] = true
		if e.Name == "" {
			t.Fatalf("%s missing name", e.ID)
		}
		if e.MaxLat <= e.MinLat || e.MaxLon <= e.MinLon {
			t.Fatalf("%s invalid bbox", e.ID)
		}
		if len(e.Ring) < 4 {
			t.Fatalf("%s needs ring for overlay", e.ID)
		}
	}
	for _, need := range []string{"GERMANY", "FRANCE", "USA", "UKRAINE", "UK", "RUSSIA", "IRAN", "CHINA", "ISRAEL", "TAIWAN", "POLAND", "BALTICS"} {
		if !ids[need] {
			t.Fatalf("catalog missing %s", need)
		}
	}
	if len(RegionalRiskCatalog()) < 10 {
		t.Fatalf("catalog too small: %d", len(RegionalRiskCatalog()))
	}
}

func TestFillCatalogFromScores_PartialAIKeepsFullCatalog(t *testing.T) {
	now := time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)
	// AI only returned two regions — HUD must still list every catalog ID.
	scores := map[string]RiskScore{
		"GERMANY": {
			OverallRisk: 40, EvaluationSource: "ai", AINarrative: "DE ok",
			AIEvaluatedAt: now, Trend: "stable",
		},
		"UKRAINE": {
			OverallRisk: 80, EvaluationSource: "ai", AINarrative: "conflict",
			AIEvaluatedAt: now, Trend: "up",
		},
	}
	baseline := map[string]RegionalRiskData{
		"RUSSIA": {RegionID: "RUSSIA", RegionName: "Russia", OverallRisk: 70, Trend: "up", EvaluationSource: "deterministic"},
		"IRAN":   {RegionID: "IRAN", RegionName: "Iran", OverallRisk: 65, Trend: "up", EvaluationSource: "deterministic"},
	}
	out := FillCatalogFromScores(scores, baseline, now)
	wantN := len(RegionalRiskCatalog())
	if len(out) != wantN {
		t.Fatalf("len=%d want full catalog %d", len(out), wantN)
	}
	got := map[string]RegionalRiskData{}
	for _, r := range out {
		got[r.RegionID] = r
		if r.RegionName == "" {
			t.Fatalf("%s empty region_name", r.RegionID)
		}
	}
	for _, id := range RegionalRiskCatalogIDs() {
		if _, ok := got[id]; !ok {
			t.Fatalf("missing catalog id %s after partial AI", id)
		}
	}
	if got["GERMANY"].AINarrative != "DE ok" || got["GERMANY"].EvaluationSource != "ai" {
		t.Fatalf("GERMANY AI fields: %+v", got["GERMANY"])
	}
	if got["RUSSIA"].OverallRisk != 70 || got["RUSSIA"].EvaluationSource != "deterministic" {
		t.Fatalf("RUSSIA baseline fill: %+v", got["RUSSIA"])
	}
	if got["IRAN"].OverallRisk != 65 {
		t.Fatalf("IRAN baseline fill: %+v", got["IRAN"])
	}
	// Unknown catalog slots without baseline still present with shell
	if got["TAIWAN"].RegionID != "TAIWAN" {
		t.Fatal("TAIWAN shell missing")
	}
}

func TestDefaultRegionEngine_IncludesRussiaIran(t *testing.T) {
	eng := GetDefaultRegionEngine()
	// Moscow
	hitRU := eng.MatchPoint(55.75, 37.62)
	foundRU := false
	for _, r := range hitRU {
		if r.ID == "RUSSIA" {
			foundRU = true
		}
	}
	if !foundRU {
		t.Fatalf("Moscow should match RUSSIA, got %+v", hitRU)
	}
	// Tehran
	hitIR := eng.MatchPoint(35.7, 51.4)
	foundIR := false
	for _, r := range hitIR {
		if r.ID == "IRAN" {
			foundIR = true
		}
	}
	if !foundIR {
		t.Fatalf("Tehran should match IRAN, got %+v", hitIR)
	}
}

func TestAttachCatalogReferences(t *testing.T) {
	risks := []RegionalRiskData{{RegionID: "IRAN", RegionName: "Iran"}}
	refs := map[string][]RiskReference{
		"IRAN": {{Title: "Strike reported", URL: "https://example.com/a", Source: "rss"}},
	}
	out := AttachCatalogReferences(risks, refs)
	if len(out[0].References) != 1 || out[0].References[0].Title != "Strike reported" {
		t.Fatalf("refs: %+v", out[0].References)
	}
}

// TestMergeRiskReferences_URLUpgradeSameTitle drives the shipped merge used by
// finalizeRegionalRiskPayload: title-only Collect events must not block baseline URLs.
func TestMergeRiskReferences_URLUpgradeSameTitle(t *testing.T) {
	// Simulates SharedIntelStore Event path (no SourceURL on Event type).
	collectTitleOnly := []RiskReference{
		{Title: "Strike reported near Tehran", Source: "geo"},
		{Title: "Other item", Source: "rss"},
	}
	// Baseline / IntelligenceEvent path with real SourceURL for same title.
	baselineWithURL := []RiskReference{
		{Title: "Strike reported near Tehran", URL: "https://example.com/strike", Source: "tagesschau"},
		{Title: "Brand new with url", URL: "https://example.com/new", Source: "dw"},
	}
	merged := MergeRiskReferences(12, collectTitleOnly, baselineWithURL)
	if len(merged) < 2 {
		t.Fatalf("expected merged refs, got %+v", merged)
	}
	var strike *RiskReference
	for i := range merged {
		if strings.EqualFold(merged[i].Title, "Strike reported near Tehran") {
			strike = &merged[i]
			break
		}
	}
	if strike == nil {
		t.Fatalf("missing strike title in %+v", merged)
	}
	if strike.URL != "https://example.com/strike" {
		t.Fatalf("same-title baseline URL must upgrade title-only Collect: got %q", strike.URL)
	}
	if strike.Source != "geo" && strike.Source != "tagesschau" {
		// Source may stay first-seen (geo) or be filled if empty — first-seen geo is fine.
		t.Logf("source kept as %q (ok)", strike.Source)
	}
	// Later empty URL must not wipe an existing URL
	wiped := MergeRiskReferences(12,
		[]RiskReference{{Title: "Keep me", URL: "https://example.com/keep"}},
		[]RiskReference{{Title: "Keep me", URL: "", Source: "x"}},
	)
	if len(wiped) != 1 || wiped[0].URL != "https://example.com/keep" {
		t.Fatalf("empty URL must not overwrite real URL: %+v", wiped)
	}
}
