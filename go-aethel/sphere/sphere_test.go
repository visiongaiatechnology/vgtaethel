// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/sphere/sphere_test.go
// Purpose: Exhaustive unit test suite for Sphere 2.0 Engines, Object Store, and Cross-App Workflows

package sphere

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupTestStore(t *testing.T) (*ObjectStore, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := NewObjectStore(dir)
	if err != nil {
		t.Fatalf("NewObjectStore failed: %v", err)
	}
	return store, dir
}

func TestObjectStoreCRUD(t *testing.T) {
	store, _ := setupTestStore(t)

	obj := &SphereObject{
		ID:        "TEST-1",
		Type:      TypeDocument,
		Title:     "Test Dokument",
		Summary:   "Zusammenfassung",
		Desktop:   DesktopWork,
		Tags:      []string{"test", "doc"},
		Data:      json.RawMessage(`{"text":"hello world"}`),
		CreatedAt: time.Now().UTC(),
	}

	// 1. Put
	if err := store.Put(obj); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// 2. Get
	fetched, ok := store.Get("TEST-1")
	if !ok {
		t.Fatal("expected object TEST-1 to exist")
	}
	if fetched.Title != "Test Dokument" || fetched.Version != 1 {
		t.Fatalf("unexpected fetched object: %+v", fetched)
	}

	// 3. Search
	results := store.Search("Dokument", 10)
	if len(results) != 1 || results[0].ID != "TEST-1" {
		t.Fatalf("Search failed, got %d results", len(results))
	}

	// 4. List by Desktop
	list := store.List("", DesktopWork, "")
	if len(list) != 1 {
		t.Fatalf("List by desktop failed, got %d", len(list))
	}

	// 5. Delete
	deleted, err := store.Delete("TEST-1")
	if err != nil || !deleted {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, ok := store.Get("TEST-1"); ok {
		t.Fatal("expected object to be deleted")
	}
}

func TestWorkspaceStatePersistence(t *testing.T) {
	store, dir := setupTestStore(t)

	state := DesktopWorkspaceState{
		ActiveDesktop: DesktopTravel,
		AmbientLevel:  18,
		Windows: map[string]DesktopWindowState{
			"editor": {
				AppID:   "editor",
				Desktop: "TRAVEL",
				Left:    120,
				Top:     80,
				Width:   600,
				Height:  450,
				IsOpen:  true,
			},
		},
		LastSaved: time.Now().UTC(),
	}

	if err := store.SaveWorkspaceState(state); err != nil {
		t.Fatalf("SaveWorkspaceState failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, "workspace_state.json")); err != nil {
		t.Fatalf("workspace_state.json not created on disk: %v", err)
	}

	// Re-load into new store instance
	newStore, err := NewObjectStore(dir)
	if err != nil {
		t.Fatalf("reloading store failed: %v", err)
	}
	reloaded := newStore.GetWorkspaceState()
	if reloaded.ActiveDesktop != DesktopTravel || reloaded.AmbientLevel != 18 {
		t.Fatalf("reloaded state mismatch: %+v", reloaded)
	}
	if win, ok := reloaded.Windows["editor"]; !ok || win.Left != 120 {
		t.Fatalf("window state not preserved: %+v", win)
	}
}

func TestDispatchSendToWriter(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	payload := &SendToPayload{
		SourceApp:  "browser",
		TargetApp:  "writer",
		ObjectType: TypeDocument,
		Title:      "Tokyo Reiseführer 2027",
		Content:    "Hier sind die besten Hotels in Shinjuku und Shibuya.",
		Metadata:   map[string]interface{}{"url": "https://japan-guide.com"},
	}

	obj, err := svc.DispatchSendTo(payload)
	if err != nil {
		t.Fatalf("DispatchSendTo failed: %v", err)
	}
	if obj.Type != TypeDocument || !strings.Contains(obj.Title, "Tokyo") {
		t.Fatalf("unexpected sent object: %+v", obj)
	}

	// Check that it exists in store
	found, ok := svc.GetStore().Get(obj.ID)
	if !ok {
		t.Fatalf("transferred object not found in store")
	}
	if !strings.Contains(found.Summary, "browser") {
		t.Fatalf("summary should note provenance, got %s", found.Summary)
	}
}

func TestDispatchSendToResearch(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	payload := &SendToPayload{
		SourceApp:  "global_watch",
		TargetApp:  "research",
		ObjectType: TypeResearchItem,
		Title:      "Erdbebenmeldung Tokio",
		Content:    "Magnitude 5.2 vor der Küste registriert. Keine Tsunami-Gefahr.",
		Metadata: map[string]interface{}{
			"source_title": "JMA Live Feed",
			"url":          "https://jma.go.jp",
		},
	}

	obj, err := svc.DispatchSendTo(payload)
	if err != nil {
		t.Fatalf("DispatchSendTo failed: %v", err)
	}
	if obj.Type != TypeResearchItem {
		t.Fatalf("expected research item type, got %s", obj.Type)
	}
}
