// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/sphere/documents/document_engine_test.go

package documents

import (
	"strings"
	"testing"

	"go-aethel/sphere"
)

func TestDocumentEngineTrackChanges(t *testing.T) {
	dir := t.TempDir()
	store, err := sphere.NewObjectStore(dir)
	if err != nil {
		t.Fatalf("NewObjectStore failed: %v", err)
	}

	engine := NewEngine(store)

	doc, err := engine.CreateDocument("Aethel Release Plan", "<h1>Aethel Release</h1><p>Geplanter Release im August.</p>", "OPERATOR")
	if err != nil {
		t.Fatalf("CreateDocument failed: %v", err)
	}
	if doc.WordCount == 0 {
		t.Fatal("expected non-zero word count")
	}

	// Propose AI Change
	diff, err := engine.ProposeChange(doc.ID, "Release auf Oktober verschieben", "August", "Oktober", "AETHEL // AI Core")
	if err != nil {
		t.Fatalf("ProposeChange failed: %v", err)
	}
	if diff.Status != "pending" {
		t.Fatalf("expected pending status, got %s", diff.Status)
	}

	// Apply change (Accept)
	if err := engine.ApplyChange(doc.ID, diff.DiffID, true); err != nil {
		t.Fatalf("ApplyChange failed: %v", err)
	}

	// Verify document content updated
	updated, err := engine.GetDocument(doc.ID)
	if err != nil {
		t.Fatalf("GetDocument failed: %v", err)
	}
	if !strings.Contains(updated.ContentHTML, "Oktober") {
		t.Fatalf("expected content to contain Oktober, got %s", updated.ContentHTML)
	}
	if len(updated.Versions) < 2 {
		t.Fatalf("expected snapshot in versions history, got %d versions", len(updated.Versions))
	}
	if len(updated.PendingDiffs) != 0 {
		t.Fatalf("expected 0 pending diffs after accept, got %d", len(updated.PendingDiffs))
	}
}
