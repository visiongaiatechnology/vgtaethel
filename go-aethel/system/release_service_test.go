// STATUS: DIAMANT VGT SUPREME
package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleasePreferencesAreSealedAndRequireHTTPS(t *testing.T) {
	root := t.TempDir()
	service := NewReleaseService(root)
	if err := service.SavePreferences(ReleasePreferences{UpdateManifestURL: "http://example.invalid/manifest.json"}); err == nil {
		t.Fatal("non-HTTPS update manifest was accepted")
	}
	if err := service.SavePreferences(ReleasePreferences{CrashReportingOptIn: true, UpdateManifestURL: "https://updates.example.invalid/manifest.json"}); err != nil {
		t.Fatal(err)
	}
	prefs, err := service.Preferences()
	if err != nil || !prefs.CrashReportingOptIn || prefs.UpdateManifestURL == "" {
		t.Fatalf("preferences were not persisted: %+v err=%v", prefs, err)
	}
	data, err := os.ReadFile(filepath.Join(root, "release_preferences.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 || strings.Contains(string(data), "https://updates.example.invalid/manifest.json") {
		t.Fatal("release preferences were persisted in plaintext")
	}
}

func TestUpdateCheckIsDisabledWithoutConfiguration(t *testing.T) {
	service := NewReleaseService(t.TempDir())
	result, err := service.CheckForUpdate()
	if err != nil || result["configured"] != false || result["current_version"] != ProductVersion {
		t.Fatalf("unexpected disabled update state: result=%+v err=%v", result, err)
	}
}
