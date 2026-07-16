package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go-aethel/security"
	"go-aethel/skills"
)

func handleViewportScreenshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := skills.GetLatestScreenshot()
	if err != nil {
		http.Error(w, fmt.Sprintf("Screenshot capture failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func handleViewportStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"active":      true,
		"width":       1280,
		"height":      720,
		"browser_url": skills.GetLastBrowserURL(),
	})
}

func handleSphereDocument(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	docPath := "./vgt_workspace/sphere_document.html"

	if r.Method == http.MethodGet {
		data, err := os.ReadFile(docPath)
		if err != nil {
			if os.IsNotExist(err) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write([]byte("<h1>Aethel Document</h1><p>Beginne mit dem Schreiben oder lass Aethel ein Dokument erstellen...</p>"))
				return
			}
			http.Error(w, fmt.Sprintf("Failed to read document: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
		return
	}

	if r.Method == http.MethodPost {
		data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		err = os.MkdirAll(filepath.Dir(docPath), 0700)
		if err != nil {
			http.Error(w, "Failed to create directory", http.StatusInternalServerError)
			return
		}

		err = os.WriteFile(docPath, data, 0600)
		if err != nil {
			http.Error(w, "Failed to write file", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

type sphereWorkspaceEntry struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
	Size int64  `json:"size,omitempty"`
}

// handleSphereWorkspace exposes a deliberately shallow, metadata-only view
// of Aethel's own workspace. File contents and secret-like names never leave
// the local skill boundary through this desktop convenience endpoint.
func handleSphereWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	entries := make([]sphereWorkspaceEntry, 0, 80)
	err := filepath.WalkDir(security.WorkspaceDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || len(entries) >= 80 {
			return walkErr
		}
		if path == security.WorkspaceDir {
			return nil
		}
		rel, err := filepath.Rel(security.WorkspaceDir, path)
		if err != nil {
			return err
		}
		if strings.Count(filepath.ToSlash(rel), "/") > 1 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		name := strings.ToLower(entry.Name())
		if strings.Contains(name, "secret") || strings.Contains(name, "vault") || strings.HasSuffix(name, ".key") || strings.HasSuffix(name, ".pem") || strings.HasPrefix(name, ".env") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		kind := "file"
		var size int64
		if entry.IsDir() {
			kind = "folder"
		} else if info, infoErr := entry.Info(); infoErr == nil {
			size = info.Size()
		}
		entries = append(entries, sphereWorkspaceEntry{Path: filepath.ToSlash(rel), Kind: kind, Size: size})
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		http.Error(w, "Workspace unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"entries": entries, "root": "vgt_workspace"})
}
