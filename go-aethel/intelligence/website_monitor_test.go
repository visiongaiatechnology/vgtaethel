package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"path/filepath"
	"testing"
	"time"
)

func TestWebsiteMonitorPersistsBaselineAndChangeWithForensicProvenance(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "intel.json"), NewEventBus())
	monitor, err := store.AddWebsiteMonitor(WebsiteMonitor{Name: "Official status", URL: "https://status.example.test/page", SourceID: "official-status", Domain: "infrastructure", IntervalMinutes: 15, LicenseID: "site-terms", AllowedUse: "situational-awareness", RetentionDays: 365, Classification: "public", AuthenticationMode: "none", Geography: "global", Redistribution: "metadata-only"})
	if err != nil {
		t.Fatal(err)
	}
	metadata := AcquisitionMetadata{OriginalURL: monitor.URL, FinalURL: monitor.URL, MIMEType: "text/html", ResponseHeaders: map[string]string{"etag": "one"}, FetchedAt: time.Now().UTC()}
	first, err := store.ImportAcquiredDocument([]byte("<html><body>service operational baseline</body></html>"), "html", monitor.SourceID, "baseline.html", monitor.Domain, metadata)
	if err != nil {
		t.Fatal(err)
	}
	if _, change, err := store.RecordWebsiteMonitorSuccess(monitor.ID, first, 200, "one", ""); err != nil || change != nil {
		t.Fatalf("baseline failed: %+v %v", change, err)
	}
	metadata.ResponseHeaders["etag"] = "two"
	second, err := store.ImportAcquiredDocument([]byte("<html><body>service degraded incident reported</body></html>"), "html", monitor.SourceID, "changed.html", monitor.Domain, metadata)
	if err != nil {
		t.Fatal(err)
	}
	updated, change, err := store.RecordWebsiteMonitorSuccess(monitor.ID, second, 200, "two", "")
	if err != nil || change == nil || change.PreviousSHA256 != first.RawSHA256 || change.CurrentSHA256 != second.RawSHA256 {
		t.Fatalf("change record invalid: %+v %v", change, err)
	}
	if updated.LastSnapshotID != second.SnapshotID || second.OriginalURL != monitor.URL || second.CaptureScope != "original" {
		t.Fatalf("forensic provenance lost: %+v %+v", updated, second)
	}
}

func TestWebsiteMonitorRequiresMachinePolicy(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "intel.json"), NewEventBus())
	if _, err := store.AddWebsiteMonitor(WebsiteMonitor{Name: "Missing policy", URL: "https://example.test", SourceID: "source", IntervalMinutes: 15, RetentionDays: 30}); err == nil {
		t.Fatal("monitor without machine-readable policy was accepted")
	}
}
