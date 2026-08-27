package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-aethel/security"
)

func TestSphereWorkspaceIndexExposesOnlySafeMetadata(t *testing.T) {
	root := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	ws := filepath.Join(root, "vgt_workspace")
	if err := os.MkdirAll(filepath.Join(ws, "notes"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "notes", "brief.md"), []byte("safe"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "vault.key"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, ".env"), []byte("TOKEN=no"), 0600); err != nil {
		t.Fatal(err)
	}

	// handleSphereWorkspace walks security.WorkspaceDir (./vgt_workspace relative to CWD).
	_ = security.WorkspaceDir
	recorder := httptest.NewRecorder()
	handleSphereWorkspace(recorder, httptest.NewRequest(http.MethodGet, "/v1/sphere/workspace", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("workspace index returned %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Entries []sphereWorkspaceEntry `json:"entries"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	joined := make([]string, 0, len(payload.Entries))
	for _, entry := range payload.Entries {
		joined = append(joined, entry.Path)
	}
	visible := strings.Join(joined, "\n")
	if !strings.Contains(visible, "notes") || strings.Contains(visible, "vault.key") || strings.Contains(visible, ".env") {
		t.Fatalf("unsafe workspace index: %s", visible)
	}
}
