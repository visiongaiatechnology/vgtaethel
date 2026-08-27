package personal

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOperationsStoreLifecycle(t *testing.T) {
	store := NewOperationsStore(filepath.Join(t.TempDir(), "operations.enc"))
	notice, err := store.Enqueue(OperationNotice{
		Kind: "daily_plan", Title: "Plan", Body: "First action", Source: "test",
	})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if pending := store.Pending(time.Now().UTC()); len(pending) != 1 || pending[0].ID != notice.ID {
		t.Fatalf("unexpected pending operations: %+v", pending)
	}
	if err := store.Snooze(notice.ID, 15*time.Minute); err != nil {
		t.Fatalf("snooze: %v", err)
	}
	if pending := store.Pending(time.Now().UTC()); len(pending) != 0 {
		t.Fatalf("snoozed operation was delivered: %+v", pending)
	}
	if err := store.Acknowledge(notice.ID); err != nil {
		t.Fatalf("acknowledge: %v", err)
	}
	reloaded := NewOperationsStore(store.filePath)
	if err := reloaded.Load(); err != nil {
		t.Fatalf("reload sealed inbox: %v", err)
	}
	if pending := reloaded.Pending(time.Now().UTC().Add(24 * time.Hour)); len(pending) != 0 {
		t.Fatalf("acknowledged operation returned after reload: %+v", pending)
	}
}

func TestOperationsStoreRejectsInvalidAndUnknownItems(t *testing.T) {
	store := NewOperationsStore(filepath.Join(t.TempDir(), "operations.enc"))
	if _, err := store.Enqueue(OperationNotice{Kind: "weather"}); err == nil {
		t.Fatal("invalid notice accepted")
	}
	if err := store.Acknowledge("op_missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected missing-item error: %v", err)
	}
}

func TestClockDueAllowsResumeGraceWithoutEarlyDelivery(t *testing.T) {
	location := time.FixedZone("test", 3600)
	if clockDue(time.Date(2026, 8, 24, 6, 59, 0, 0, location), "07:00", 3*time.Hour) {
		t.Fatal("alarm delivered before configured time")
	}
	if !clockDue(time.Date(2026, 8, 24, 8, 15, 0, 0, location), "07:00", 3*time.Hour) {
		t.Fatal("alarm was not recovered inside resume grace")
	}
	if clockDue(time.Date(2026, 8, 24, 12, 0, 0, 0, location), "07:00", 3*time.Hour) {
		t.Fatal("stale alarm delivered outside resume grace")
	}
}
