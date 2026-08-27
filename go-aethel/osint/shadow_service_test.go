package osint

// STATUS: DIAMANT VGT SUPREME

import (
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
	for _, required := range []string{"BETA V3 MANDATORY CONFLICT CONTRACT", "attacker_name", "target_name", "evidence_id"} {
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

	items, _, err := service.PrepareBatch()
	if err != nil || len(items) != ShadowBatchMax {
		t.Fatalf("expected bounded 60-item batch: len=%d err=%v", len(items), err)
	}
	invalid := validShadowTestReport(items[0].ID)
	invalid.Regions[0].EvidenceIDs = []string{"outside-batch"}
	if _, err := service.CompleteBatch(items, invalid); err == nil {
		t.Fatal("outside-batch regional evidence was accepted")
	}
	service.AbortBatch()

	items, _, err = service.PrepareBatch()
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
