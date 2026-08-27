package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareRuntimeDirectoryCopiesLegacyWorkspaceWithoutDeletingSource(t *testing.T) {
	installDir := t.TempDir()
	dataRoot := t.TempDir()
	legacyWorkspace := filepath.Join(installDir, runtimeWorkspaceName)
	if err := os.MkdirAll(filepath.Join(legacyWorkspace, "cases"), 0o700); err != nil {
		t.Fatal(err)
	}
	legacyFile := filepath.Join(legacyWorkspace, "cases", "operator.json")
	if err := os.WriteFile(legacyFile, []byte(`{"case":"preserved"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	runtimeDir, err := prepareRuntimeDirectory(filepath.Join(installDir, "AETHEL.exe"), dataRoot)
	if err != nil {
		t.Fatal(err)
	}
	persistentFile := filepath.Join(runtimeDir, runtimeWorkspaceName, "cases", "operator.json")
	content, err := os.ReadFile(persistentFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != `{"case":"preserved"}` {
		t.Fatalf("unexpected migrated content: %q", content)
	}
	if _, err := os.Stat(legacyFile); err != nil {
		t.Fatalf("legacy recovery source was removed: %v", err)
	}
}

func TestPrepareRuntimeDirectoryNeverOverwritesPersistentWorkspace(t *testing.T) {
	installDir := t.TempDir()
	dataRoot := t.TempDir()
	legacyWorkspace := filepath.Join(installDir, runtimeWorkspaceName)
	persistentWorkspace := filepath.Join(dataRoot, runtimeVendorDirectory, runtimeAppDirectory, runtimeWorkspaceName)
	if err := os.MkdirAll(legacyWorkspace, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(persistentWorkspace, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyWorkspace, "state.json"), []byte("legacy"), 0o600); err != nil {
		t.Fatal(err)
	}
	persistentFile := filepath.Join(persistentWorkspace, "state.json")
	if err := os.WriteFile(persistentFile, []byte("persistent"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := prepareRuntimeDirectory(filepath.Join(installDir, "AETHEL.exe"), dataRoot); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(persistentFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "persistent" {
		t.Fatalf("persistent state was overwritten: %q", content)
	}
}
