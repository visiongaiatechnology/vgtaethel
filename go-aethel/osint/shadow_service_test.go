package osint

// STATUS: DIAMANT VGT SUPREME

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-aethel/security"
)

func TestDefaultShadowSourcesContainRequestedCatalogAndTelegram(t *testing.T) {
	sources := defaultShadowSources()
	if len(sources) < 90 {
		t.Fatalf("source catalog too small: %d", len(sources))
	}
	seenIDs := map[string]bool{}
	foundTelegram := false
	for _, source := range sources {
		if source.ID == "" || seenIDs[source.ID] {
			t.Fatalf("invalid or duplicate source id %q", source.ID)
		}
		seenIDs[source.ID] = true
		if source.Type == "telegram" && source.URL == "https://t.me/s/militaernews" {
			foundTelegram = true
		}
	}
	if !foundTelegram {
		t.Fatal("militaernews Telegram source missing")
	}
}

func TestShadowServicePersistsSealedSourceChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shadow.enc")
	service := NewShadowService(path)
	if err := service.UpsertSource(ShadowSource{
		Name: "Operator Feed", URL: "https://example.com/feed.xml", Type: "rss",
		Domain: "military", Enabled: true, Priority: 5,
	}); err != nil {
		t.Fatal(err)
	}
	_, sealed, err := security.ReadSealedFile(path)
	if err != nil || !sealed {
		t.Fatalf("SHADOW state must be sealed: sealed=%v err=%v", sealed, err)
	}
	reloaded := NewShadowService(path).Snapshot()
	found := false
	for _, source := range reloaded.Sources {
		if source.Name == "Operator Feed" && source.Priority == 5 {
			found = true
		}
	}
	if !found {
		t.Fatal("sealed source update was not restored")
	}
}

func TestShadowV3ConflictContractCannotBeRemovedByEditableDoctrine(t *testing.T) {
	service := NewShadowService(filepath.Join(t.TempDir(), "shadow.enc"))
	custom := strings.Repeat("Operator-specific strategic doctrine. ", 10)
	if err := service.SetSystemPrompt(custom); err != nil {
		t.Fatal(err)
	}
	prompt := service.AnalysisPrompt()
	for _, required := range []string{"BETA V3 MANDATORY CONFLICT CONTRACT", "attacker_name", "target_name", "evidence_id", "market_pulse", "BTC", "BRENT", "72h", "rolling 24-hour", "context_dossiers"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("mandatory runtime contract missing %q", required)
		}
	}
}

func TestShadowBatchBoundaryAndEvidenceValidation(t *testing.T) {
	service := NewShadowService(filepath.Join(t.TempDir(), "shadow.enc"))
	service.mu.Lock()
	service.state.Buffer = make([]ShadowIntelItem, ShadowBatchMax+5)
	for index := range service.state.Buffer {
		service.state.Buffer[index] = ShadowIntelItem{
			ID: fmt.Sprintf("item-%02d", index), Title: "Evidence", CollectedAt: time.Now().UTC(),
		}
	}
	service.mu.Unlock()

	items, _, err := service.PrepareBatch("deepseek/deepseek-chat")
	if err != nil || len(items) != ShadowBatchMax {
		t.Fatalf("expected bounded 60-item batch: len=%d err=%v", len(items), err)
	}
	invalid := validShadowTestReport(items[0].ID)
	invalid.Regions[0].EvidenceIDs = []string{"outside-batch"}
	if _, err := service.CompleteBatch(items, invalid); err == nil {
		t.Fatal("outside-batch regional evidence was accepted")
	}
	service.AbortBatch()

	items, _, err = service.PrepareBatch("deepseek/deepseek-chat")
	if err != nil {
		t.Fatal(err)
	}
	valid := validShadowTestReport(items[0].ID)
	saved, err := service.CompleteBatch(items, valid)
	if err != nil {
		t.Fatal(err)
	}
	if saved.ItemsAnalyzed != ShadowBatchMax || saved.ContentSHA256 == "" {
		t.Fatalf("incomplete report metadata: %+v", saved)
	}
	status := service.Status()
	if status.ProcessedItems != ShadowBatchMax || status.PendingItems != 5 || status.AnalysisRunning {
		t.Fatalf("unexpected post-analysis state: %+v", status)
	}
}

func TestShadowAutonomyIsExplicitAndPersistsModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shadow.enc")
	service := NewShadowService(path)
	if enabled, _ := service.Autonomy(); enabled {
		t.Fatal("SHADOW autonomy must be opt-in")
	}
	if err := service.SetAutonomy(true, "deepseek/deepseek-chat"); err != nil {
		t.Fatal(err)
	}
	enabled, modelID := NewShadowService(path).Autonomy()
	if !enabled || modelID != "deepseek/deepseek-chat" {
		t.Fatalf("sealed autonomy configuration was not restored: enabled=%v model=%q", enabled, modelID)
	}
	if err := service.SetAutonomy(true, "../../invalid model"); err == nil {
		t.Fatal("unsafe autonomy model identifier was accepted")
	}
}

func TestShadowConflictLinkRequiresDirectionAndBatchEvidence(t *testing.T) {
	allowed := map[string]bool{"evidence-1": true}
	report := validShadowTestReport("evidence-1")
	report.ConflictLinks[0].EvidenceIDs = []string{"foreign-evidence"}
	if err := validateShadowReport(&report, allowed); err == nil {
		t.Fatal("foreign conflict evidence was accepted")
	}
	report = validShadowTestReport("evidence-1")
	report.ConflictLinks[0].TargetName = report.ConflictLinks[0].AttackerName
	if err := validateShadowReport(&report, allowed); err == nil {
		t.Fatal("self-directed conflict link was accepted")
	}
	report = validShadowTestReport("evidence-1")
	report.ConflictLinks[0].Action = "TENSION"
	if err := validateShadowReport(&report, allowed); err == nil {
		t.Fatal("non-directional tension was accepted as a conflict action")
	}
}

func TestShadowMarketSnapshotAndForecastDirectionsAreBounded(t *testing.T) {
	report := validShadowTestReport("evidence-1")
	report.Forecasts[0].Direction = "ESCALATION"
	report.Forecasts[0].Instruments = []string{"BRENT"}
	report.MarketSnapshot = []ShadowMarketPoint{{Symbol: "BRENT", Name: "Brent", Category: "commodity", Currency: "USD", Price: 81.5, ObservedAt: time.Now().UTC(), Source: "Market Pulse"}}
	if err := validateShadowReport(&report, map[string]bool{"evidence-1": true}); err != nil {
		t.Fatal(err)
	}
	report.MarketSnapshot[0].Symbol = "UNTRUSTED"
	if err := validateShadowReport(&report, map[string]bool{"evidence-1": true}); err == nil {
		t.Fatal("untrusted market symbol was accepted")
	}
}

func TestShadowPercentAcceptsFractionIntegerAndPercentString(t *testing.T) {
	var decoded struct {
		Fraction ShadowPercent `json:"fraction"`
		Integer  ShadowPercent `json:"integer"`
		Text     ShadowPercent `json:"text"`
	}
	if err := json.Unmarshal([]byte(`{"fraction":0.7,"integer":70,"text":"85%"}`), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Fraction != 70 || decoded.Integer != 70 || decoded.Text != 85 {
		t.Fatalf("percentage normalization failed: %+v", decoded)
	}
	if err := json.Unmarshal([]byte(`{"fraction":1.2,"integer":101,"text":"85%"}`), &decoded); err == nil {
		t.Fatal("out-of-range percentage was accepted")
	}
}

func TestShadowBatchUsesOnlyLatest24HoursAndNewestItemsFirst(t *testing.T) {
	service := NewShadowService(filepath.Join(t.TempDir(), "shadow.enc"))
	now := time.Now().UTC()
	service.mu.Lock()
	service.state.Buffer = append(service.state.Buffer, ShadowIntelItem{ID: "stale", PublishedAt: now.Add(-24*time.Hour - time.Second), CollectedAt: now})
	for index := 0; index < ShadowBatchMin; index++ {
		service.state.Buffer = append(service.state.Buffer, ShadowIntelItem{ID: fmt.Sprintf("fresh-%02d", index), PublishedAt: now.Add(-time.Duration(index) * time.Minute), CollectedAt: now})
	}
	service.mu.Unlock()

	status := service.Status()
	if status.PendingItems != ShadowBatchMin || status.StaleItems != 1 || status.IntakeHours != 24 {
		t.Fatalf("incorrect rolling-window status: %+v", status)
	}
	items, _, err := service.PrepareBatch("deepseek/deepseek-chat")
	if err != nil {
		t.Fatal(err)
	}
	defer service.AbortBatch()
	if len(items) != ShadowBatchMin || items[0].ID != "fresh-00" {
		t.Fatalf("batch was not newest-first: first=%q len=%d", items[0].ID, len(items))
	}
	for _, item := range items {
		if item.ID == "stale" {
			t.Fatal("stale item escaped the 24-hour intake boundary")
		}
	}
}

func TestShadowContextPrefersLastThreeDailyDossiers(t *testing.T) {
	service := NewShadowService(filepath.Join(t.TempDir(), "shadow.enc"))
	service.mu.Lock()
	service.state.Reports = []ShadowReport{
		{ID: "batch-new", Kind: "batch"}, {ID: "daily-3", Kind: "daily"},
		{ID: "daily-2", Kind: "daily"}, {ID: "daily-1", Kind: "daily"}, {ID: "daily-old", Kind: "daily"},
	}
	service.mu.Unlock()
	reports := service.RecentContextReports(ShadowContextDossiers)
	if len(reports) != 3 || reports[0].ID != "daily-3" || reports[2].ID != "daily-1" {
		t.Fatalf("unexpected continuity context: %+v", reports)
	}
}

func TestShadowClearOperationalDataPreservesConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shadow.enc")
	service := NewShadowService(path)
	service.mu.Lock()
	service.state.SystemPrompt = "custom sealed doctrine"
	service.state.AutonomyEnabled = true
	service.state.AutonomyModelID = "deepseek/deepseek-chat"
	service.state.Buffer = []ShadowIntelItem{{ID: "intel-1", CollectedAt: time.Now().UTC()}}
	service.state.Reports = []ShadowReport{{ID: "report-1", Kind: "daily"}}
	service.state.LastCollectAt = time.Now().UTC()
	service.state.SourceCursor = 7
	service.state.Sources[0].LastState = "error"
	service.state.Sources[0].LastError = "temporary failure"
	service.state.Sources[0].LastFetch = time.Now().UTC().Format(time.RFC3339)
	service.mu.Unlock()

	if err := service.ClearOperationalData(); err != nil {
		t.Fatal(err)
	}
	reloaded := NewShadowService(path).Snapshot()
	if len(reloaded.Buffer) != 0 || len(reloaded.Reports) != 0 || !reloaded.LastCollectAt.IsZero() || reloaded.SourceCursor != 0 {
		t.Fatalf("operational data survived reset: %+v", reloaded)
	}
	if reloaded.AutonomyEnabled || reloaded.AutonomyModelID != "deepseek/deepseek-chat" {
		t.Fatalf("unsafe or lost autonomy state after reset: %+v", reloaded)
	}
	if reloaded.SystemPrompt != "custom sealed doctrine" || len(reloaded.Sources) == 0 {
		t.Fatal("source registry or doctrine was deleted")
	}
	if reloaded.Sources[0].LastState != "" || reloaded.Sources[0].LastError != "" || reloaded.Sources[0].LastFetch != "" {
		t.Fatalf("source runtime telemetry survived reset: %+v", reloaded.Sources[0])
	}
}

func TestShadowClearOperationalDataRejectsActiveOperations(t *testing.T) {
	service := NewShadowService(filepath.Join(t.TempDir(), "shadow.enc"))
	service.mu.Lock()
	service.analysisRunning = true
	service.mu.Unlock()
	if !errors.Is(service.ClearOperationalData(), ErrShadowAnalysisRunning) {
		t.Fatal("reset was accepted during active analysis")
	}
	service.mu.Lock()
	service.analysisRunning = false
	service.collectionRunning = true
	service.mu.Unlock()
	if !errors.Is(service.ClearOperationalData(), ErrShadowCollectionRunning) {
		t.Fatal("reset was accepted during active collection")
	}
}

func validShadowTestReport(evidenceID string) ShadowReport {
	return ShadowReport{
		ThreatLevel: "HIGH",
		Summary:     "Evidence-bound strategic assessment with sufficient validated report detail.",
		EvidenceIDs: []string{evidenceID},
		Regions: []ShadowRegionAssessment{{
			RegionID: "TEST", RegionName: "Test Region", Latitude: 10, Longitude: 20,
			SecurityScore: 35, ConflictLevel: "ESCALATION", Confidence: 80, Trend: "DETERIORATING",
			EvidenceIDs: []string{evidenceID}, Assessment: "Corroborated escalation indicator.",
		}},
		ConflictLinks: []ShadowConflictLink{{
			AttackerName: "Actor A", TargetName: "Actor B", AttackerLatitude: 10, AttackerLongitude: 20,
			TargetLatitude: 30, TargetLongitude: 40, Action: "STRIKE", Confidence: 75,
			EvidenceIDs: []string{evidenceID}, Assessment: "Directional action supported by batch evidence.",
		}},
		Forecasts: []ShadowForecast{{
			Sector: "military", Horizon: "24h", Prediction: "Elevated operational activity",
			Probability: 70, EvidenceIDs: []string{evidenceID},
		}},
	}
}
