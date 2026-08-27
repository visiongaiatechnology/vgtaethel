package handlers

// STATUS: DIAMANT VGT SUPREME

import (
	"encoding/json"
	"strings"
	"testing"

	"go-aethel/intelligence"
)

func TestBuildOperatorWorkspaceUsesBoundedSafeProjections(t *testing.T) {
	state := intelligence.StoreState{
		Cases: []intelligence.Case{{
			ID: "case-1", Title: "Case One", Purpose: "Test", Classification: "restricted", Status: "active",
			Evidence: []intelligence.Evidence{{
				ID: "evidence-1", CaseID: "case-1", Excerpt: "public excerpt", SHA256: strings.Repeat("a", 64),
				SnapshotPath: `C:\\private\\vault\\object`, SnapshotID: "snapshot-1", Sealed: true,
			}},
		}},
	}
	projection := buildOperatorWorkspace(state)
	raw, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(raw)
	if strings.Contains(serialized, "SnapshotPath") || strings.Contains(serialized, `private\\vault`) {
		t.Fatalf("workspace projection exposed internal vault path: %s", serialized)
	}
	if !strings.Contains(serialized, "snapshot-1") || !strings.Contains(serialized, "evidence_count") {
		t.Fatalf("workspace projection lost required operator provenance: %s", serialized)
	}
}
