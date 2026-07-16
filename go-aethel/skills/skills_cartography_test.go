package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodeCartographyBuildsArchitectureMapAndExcludesSensitiveFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "api"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "ignored"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"ok\") }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "api", "handler.go"), []byte("package api\nfunc HandleRequest() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SECRET=must-not-appear"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "node_modules", "ignored", "lib.js"), []byte("export const ignored = true"), 0600); err != nil {
		t.Fatal(err)
	}

	files, skipped, err := collectCartographyFiles(root, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || skipped < 1 {
		t.Fatalf("unexpected cartography result: files=%+v skipped=%d", files, skipped)
	}
	report := renderCartographyReport(root, files, skipped, 50)
	for _, expected := range []string{"# Code Kartografie", "### `main.go`", "### `api/handler.go`", "`fmt`", "HandleRequest"} {
		if !strings.Contains(report, expected) {
			t.Fatalf("report missing %q: %s", expected, report)
		}
	}
	if strings.Contains(report, "SECRET=must-not-appear") || strings.Contains(report, "node_modules/ignored") {
		t.Fatal("sensitive or ignored content leaked into cartography report")
	}
}
