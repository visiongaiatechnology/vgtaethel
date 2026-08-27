package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCanonicalAdapterOwnsCaseGraphAndReID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "canonical.json")
	canonical := NewStore(path, NewEventBus())
	adapter := NewCanonicalIntelligenceAdapter(canonical)

	caseRecord, err := adapter.CreateCase("Port disruption", "Assess operational impact and competing explanations")
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := adapter.SealEvidence(caseRecord.ID, "official notice", "https://example.test/notice", "Commercial operations suspended", "")
	if err != nil {
		t.Fatal(err)
	}
	first, err := adapter.AddEntity(caseRecord.ID, "Harbor Authority", "organisation", 90)
	if err != nil {
		t.Fatal(err)
	}
	second, err := adapter.AddEntity(caseRecord.ID, "Terminal A", "location", 85)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.LinkEntities(caseRecord.ID, first.ID, second.ID, "operates", evidence.ID, 80); err != nil {
		t.Fatal(err)
	}
	request, err := adapter.RequestReID(caseRecord.ID, "Validate whether aliases overlap this active case", "requester")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.ApproveReID(caseRecord.ID, request.ID, "approver-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.ApproveReID(caseRecord.ID, request.ID, "approver-a"); err == nil {
		t.Fatal("dual control accepted the same approver twice")
	}
	unlocked, err := adapter.ApproveReID(caseRecord.ID, request.ID, "approver-b")
	if err != nil || !unlocked.Unlocked || unlocked.Status != "unlocked" {
		t.Fatalf("second approval failed: %+v %v", unlocked, err)
	}

	snapshot := canonical.GetSnapshot()
	if len(snapshot.Cases) != 1 || len(snapshot.Cases[0].Evidence) != 1 || len(snapshot.Cases[0].Relations) != 1 || len(snapshot.Cases[0].ReIDRequests) != 1 {
		t.Fatalf("canonical case graph incomplete: %+v", snapshot.Cases)
	}
	if !VerifyCustodyChain(snapshot.CustodyEvents) {
		t.Fatal("canonical Re-ID actions broke the custody chain")
	}
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 || string(raw[:min(len(raw), len("AETHEL-SEAL-v1:"))]) != "AETHEL-SEAL-v1:" {
		t.Fatalf("canonical state is not sealed: %v", err)
	}
}

func TestLegacyMigrationIsIdempotent(t *testing.T) {
	root := t.TempDir()
	legacyPath := filepath.Join(root, "legacy.json")
	legacy := NewIntelligenceStore(legacyPath)
	caseRecord, err := legacy.CreateCase("Legacy case", "Preserve evidence and graph during migration")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.SealEvidence(caseRecord.ID, "legacy source", "https://example.test/legacy", "legacy excerpt", ""); err != nil {
		t.Fatal(err)
	}
	if err := legacy.ProposeEvent(IntelligenceEvent{
		ID: "legacy-event", Title: "Legacy event", Summary: "Migrated once", Source: "legacy-source",
		ObservedAt: time.Now().UTC(), Confidence: 70, Severity: "medium",
	}); err != nil {
		t.Fatal(err)
	}

	canonical := NewStore(filepath.Join(root, "canonical.json"), NewEventBus())
	if err := MigrateLegacyIntelligence(legacyPath, canonical); err != nil {
		t.Fatal(err)
	}
	first := canonical.GetSnapshot()
	if err := MigrateLegacyIntelligence(legacyPath, canonical); err != nil {
		t.Fatal(err)
	}
	second := canonical.GetSnapshot()
	if len(first.Cases) != 1 || len(first.Events) != 1 || len(second.Cases) != len(first.Cases) || len(second.Events) != len(first.Events) {
		t.Fatalf("migration was not idempotent: first cases/events=%d/%d second=%d/%d", len(first.Cases), len(first.Events), len(second.Cases), len(second.Events))
	}
	if !canonical.MigrationApplied(legacyIntelligenceMigrationID) {
		t.Fatal("migration marker missing")
	}
}
