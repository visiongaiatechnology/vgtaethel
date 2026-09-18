package geoint

// STATUS: DIAMANT VGT SUPREME

import (
	"testing"
	"time"

	"go-aethel/osint"
)

type shadowSnapshotFixture struct {
	state osint.ShadowState
}

func (f shadowSnapshotFixture) Snapshot() osint.ShadowState { return f.state }

func TestShadowGeoAdapterProjectsOnlyEvidenceBoundRelations(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	state := osint.ShadowState{
		Buffer: []osint.ShadowIntelItem{{
			ID: "ev-1", SourceID: "source-official", PublishedAt: now.Add(-2 * time.Hour),
		}},
		Reports: []osint.ShadowReport{{
			ID: "report-1", Kind: "DAILY", CreatedAt: now.Add(-time.Hour), ContentSHA256: "hash-1",
			Regions: osint.ShadowRegions{{
				RegionID: "region-a", RegionName: "Region A", Latitude: 50, Longitude: 30,
				Confidence: 82, SecurityScore: 24, ConflictLevel: "WAR", Trend: "DETERIORATING",
				EvidenceIDs: osint.ShadowStringList{"ev-1"}, Assessment: "Evidence-backed regional assessment.",
			}},
			ConflictLinks: osint.ShadowConflictLinks{{
				AttackerName: "Actor A", TargetName: "Actor B", AttackerLatitude: 50, AttackerLongitude: 30,
				TargetLatitude: 49, TargetLongitude: 31, Action: "STRIKE", Confidence: 82,
				EvidenceIDs: osint.ShadowStringList{"ev-1"}, Assessment: "Reported strike direction.",
			}},
		}},
	}

	bus := NewGeoEntityBus()
	adapter := NewShadowGeoAdapter(bus, shadowSnapshotFixture{state: state})
	adapter.Sync(now)

	relations, synchronizedAt, version := adapter.Relations()
	if len(relations) != 1 {
		t.Fatalf("expected one evidence-bound relation, got %d", len(relations))
	}
	if synchronizedAt != now || version == "" {
		t.Fatalf("missing synchronization metadata: %v %q", synchronizedAt, version)
	}
	relation := relations[0]
	if relation.Category != RelationKinetic || relation.AssessmentStatus != AssessmentSupported {
		t.Fatalf("unexpected relation classification: %#v", relation)
	}
	if len(relation.EvidenceIDs) != 1 || relation.EvidenceIDs[0] != "ev-1" || len(relation.SourceIDs) != 1 {
		t.Fatalf("relation lost provenance: %#v", relation)
	}
	entities := bus.ListByType(TypeConflict, 10)
	if len(entities) != 1 || entities[0].Location != PrecisionRegion || entities[0].Assessment != AssessmentSupported {
		t.Fatalf("expected one assessed regional entity, got %#v", entities)
	}
}

func TestShadowGeoAdapterCyberAttributionRemainsReported(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	state := osint.ShadowState{Reports: []osint.ShadowReport{{
		ID: "cyber-report", CreatedAt: now.Add(-time.Hour),
		ConflictLinks: osint.ShadowConflictLinks{{
			AttackerName: "Reported Actor", TargetName: "Target", AttackerLatitude: 55, AttackerLongitude: 37,
			TargetLatitude: 52, TargetLongitude: 13, Action: "CYBER_ATTACK", Confidence: 95,
			EvidenceIDs: osint.ShadowStringList{"ev-cyber"}, Assessment: "Source-reported attribution.",
		}},
	}}}
	adapter := NewShadowGeoAdapter(NewGeoEntityBus(), shadowSnapshotFixture{state: state})
	adapter.Sync(now)
	relations, _, _ := adapter.Relations()
	if len(relations) != 1 {
		t.Fatalf("expected one cyber relation, got %d", len(relations))
	}
	if relations[0].Category != RelationCyber || relations[0].AttributionStatus != "REPORTED_ATTRIBUTION" {
		t.Fatalf("cyber attribution was overstated: %#v", relations[0])
	}
	if relations[0].AssessmentStatus == AssessmentConfirmed {
		t.Fatal("model-derived cyber attribution must never be promoted to CONFIRMED")
	}
}

func TestShadowGeoAdapterRejectsInvalidStaleAndDuplicateRelations(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	newer := osint.ShadowReport{
		ID: "newer", CreatedAt: now.Add(-time.Hour),
		ConflictLinks: osint.ShadowConflictLinks{
			{AttackerName: "A", TargetName: "B", AttackerLatitude: 10, AttackerLongitude: 20, TargetLatitude: 30, TargetLongitude: 40, Action: "INVASION", Confidence: 75, EvidenceIDs: osint.ShadowStringList{"ev-new"}},
			{AttackerName: "Invalid", TargetName: "Nowhere", AttackerLatitude: 0, AttackerLongitude: 0, TargetLatitude: 20, TargetLongitude: 20, Action: "STRIKE", Confidence: 90, EvidenceIDs: osint.ShadowStringList{"ev-invalid"}},
		},
	}
	olderDuplicate := newer
	olderDuplicate.ID = "older-duplicate"
	olderDuplicate.CreatedAt = now.Add(-2 * time.Hour)
	olderDuplicate.ConflictLinks = osint.ShadowConflictLinks{{
		AttackerName: "A", TargetName: "B", AttackerLatitude: 11, AttackerLongitude: 21,
		TargetLatitude: 31, TargetLongitude: 41, Action: "INVASION", Confidence: 20, EvidenceIDs: osint.ShadowStringList{"ev-old"},
	}}
	stale := newer
	stale.ID = "stale"
	stale.CreatedAt = now.Add(-8 * 24 * time.Hour)
	stale.ConflictLinks = osint.ShadowConflictLinks{{
		AttackerName: "Stale", TargetName: "Target", AttackerLatitude: 1, AttackerLongitude: 1,
		TargetLatitude: 2, TargetLongitude: 2, Action: "STRIKE", Confidence: 99, EvidenceIDs: osint.ShadowStringList{"ev-stale"},
	}}
	adapter := NewShadowGeoAdapter(NewGeoEntityBus(), shadowSnapshotFixture{state: osint.ShadowState{
		Reports: []osint.ShadowReport{olderDuplicate, stale, newer},
	}})
	adapter.Sync(now)
	relations, _, _ := adapter.Relations()
	if len(relations) != 1 {
		t.Fatalf("expected exactly one current, valid, deduplicated relation; got %#v", relations)
	}
	if relations[0].ReportID != "newer" || relations[0].Confidence != 75 {
		t.Fatalf("newest relation did not win deduplication: %#v", relations[0])
	}
}
