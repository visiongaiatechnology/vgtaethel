package osint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGlobalWatchMonitorMigratesLegacyPolicyAndPrunesReports(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config", "schedule.json")
	reportDir := filepath.Join(root, "reports")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	legacy, err := json.Marshal(GlobalWatchSchedule{Enabled: false, IntervalMinutes: 60})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	monitor := NewGlobalWatchMonitor(nil, configPath, reportDir)
	snapshot := monitor.Snapshot()
	if snapshot.RetentionDays != 30 || snapshot.MaxReports != 100 {
		t.Fatalf("legacy defaults not migrated: %+v", snapshot)
	}
	if err := monitor.ConfigurePolicy(false, 30, 7, 2); err != nil {
		t.Fatalf("configure policy: %v", err)
	}
	if err := os.MkdirAll(reportDir, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for index := 0; index < 5; index++ {
		name := filepath.Join(reportDir, "global-watch-20260712-12000"+string(rune('0'+index))+".md")
		if err := os.WriteFile(name, []byte("report"), 0o600); err != nil {
			t.Fatal(err)
		}
		stamp := now.Add(-time.Duration(index) * time.Hour)
		if err := os.Chtimes(name, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	oldName := filepath.Join(reportDir, "global-watch-20200101-000000.md")
	if err := os.WriteFile(oldName, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldStamp := now.Add(-10 * 24 * time.Hour)
	if err := os.Chtimes(oldName, oldStamp, oldStamp); err != nil {
		t.Fatal(err)
	}
	if err := monitor.pruneReports(); err != nil {
		t.Fatalf("prune reports: %v", err)
	}
	entries, err := os.ReadDir(reportDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected newest two reports, got %d", len(entries))
	}
	if entries[0].Name() != "global-watch-20260712-120000.md" || entries[1].Name() != "global-watch-20260712-120001.md" {
		t.Fatalf("unexpected retained reports: %s, %s", entries[0].Name(), entries[1].Name())
	}
}

func TestGlobalWatchMonitorRejectsUnsafePolicyBounds(t *testing.T) {
	monitor := NewGlobalWatchMonitor(nil, filepath.Join(t.TempDir(), "schedule.json"), t.TempDir())
	invalid := [][3]int{{14, 30, 100}, {60, 0, 100}, {60, 30, 0}, {1441, 30, 100}, {60, 3651, 100}, {60, 30, 5001}}
	for _, policy := range invalid {
		if err := monitor.ConfigurePolicy(false, policy[0], policy[1], policy[2]); err == nil {
			t.Fatalf("invalid policy accepted: %+v", policy)
		}
	}
}
