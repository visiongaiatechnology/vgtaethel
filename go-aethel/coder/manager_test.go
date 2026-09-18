// STATUS: DIAMANT VGT SUPREME
package coder

import (
	"path/filepath"
	"testing"
)

func TestManagerPersistsSessionAndPublicViewHidesBaselines(t *testing.T) {
	root := initializeTestRepository(t)
	store := filepath.Join(t.TempDir(), "coder_sessions.sealed")
	manager := NewManager(store)
	session, err := manager.Create(root, "Implement a verified change", "worker-model", "orchestrator-model")
	if err != nil {
		t.Fatal(err)
	}
	reloaded := NewManager(store)
	stored, ok := reloaded.Get(session.SessionID)
	if !ok || stored.WorkerModel != "worker-model" || stored.OrchestratorModel != "orchestrator-model" {
		t.Fatalf("persisted session mismatch: %#v", stored)
	}
	public := PublicSession(stored)
	if public.BaselineChanges != nil || public.BaselineFiles != nil {
		t.Fatal("repository baselines must never be transported to the UI")
	}
}
