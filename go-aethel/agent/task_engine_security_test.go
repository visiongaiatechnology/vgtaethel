package agent

// STATUS: DIAMANT VGT SUPREME

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go-aethel/security"
)

func TestScheduledTaskRejectsReusableOrMutatingAuthority(t *testing.T) {
	engine := NewTaskEngine(filepath.Join(t.TempDir(), "tasks.enc"))
	base := TaskItem{ID: "task-safe", Text: "Read status", Objective: "Read status", RequiredCapabilities: []string{string(security.CapIntelRead)}}
	withApproval := base
	withApproval.PreApprovedCapabilities = []string{string(security.CapFsWrite)}
	if err := engine.Add(withApproval); err == nil {
		t.Fatal("scheduled task accepted reusable pre-approval")
	}
	mutating := base
	mutating.PreApprovedCapabilities = nil
	mutating.RequiredCapabilities = []string{string(security.CapFsWrite)}
	if err := engine.Add(mutating); err == nil {
		t.Fatal("scheduled task accepted mutating capability")
	}
	if !taskDeclaresCapability(base, security.CapIntelRead) || taskDeclaresCapability(base, security.CapIntelSources) {
		t.Fatal("scheduled capability matching is not exact")
	}
}

func TestRestrictedCronSchedulerIsDeterministicAndBounded(t *testing.T) {
	after := time.Date(2026, 8, 24, 10, 7, 31, 0, time.UTC)
	next, err := nextCronTime("*/15 9 * * 1", after)
	if err != nil {
		t.Fatal(err)
	}
	expected := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Fatalf("cron next occurrence=%s expected=%s", next, expected)
	}
	for _, invalid := range []string{"* * *", "0 0 * * * *", "*/0 * * * *", "60 * * * *", "* * * * 7", "1,2 * * * *"} {
		if _, err := nextCronTime(invalid, after); err == nil {
			t.Fatalf("unsafe or unsupported cron expression accepted: %s", invalid)
		}
	}
}

func TestTaskJournalIsSealedAndMigratesLegacyAuthority(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.enc")
	engine := NewTaskEngine(path)
	objective := "sensitive operator project objective"
	if err := engine.Add(TaskItem{ID: "task-one", Text: "Status", Objective: objective, RequiredCapabilities: []string{string(security.CapIntelRead)}}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(objective)) {
		t.Fatal("task journal persisted operator context in plaintext")
	}
	loaded := NewTaskEngine(path)
	if err := loaded.Load(); err != nil || len(loaded.GetAll()) != 1 {
		t.Fatalf("sealed task journal failed round trip: %v", err)
	}
}
