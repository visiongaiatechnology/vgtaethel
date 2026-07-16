package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSelectRuntimeWorkingDirectoryRequiresShippedWorkspace(t *testing.T) {
	root := t.TempDir()
	executablePath := filepath.Join(root, "AETHEL.exe")
	if runtimeDir, ok := selectRuntimeWorkingDirectory(executablePath); ok || runtimeDir != "" {
		t.Fatalf("runtime directory selected without workspace: %q", runtimeDir)
	}
	if err := os.Mkdir(filepath.Join(root, "vgt_workspace"), 0700); err != nil {
		t.Fatal(err)
	}
	runtimeDir, ok := selectRuntimeWorkingDirectory(executablePath)
	if !ok || runtimeDir != filepath.Clean(root) {
		t.Fatalf("unexpected runtime directory: path=%q selected=%t", runtimeDir, ok)
	}
}
