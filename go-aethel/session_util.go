package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var safeSessionIDPattern = regexp.MustCompile(`^session_[A-Za-z0-9_-]{1,80}$`)
var safeResourceIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,79}$`)

func validateSessionID(id string) error {
	if !safeSessionIDPattern.MatchString(id) {
		return fmt.Errorf("invalid session id")
	}
	return nil
}

func validateResourceID(id string) error {
	if !safeResourceIDPattern.MatchString(id) {
		return fmt.Errorf("invalid resource id")
	}
	return nil
}

func sessionFilePath(id string) (string, error) {
	if err := validateSessionID(id); err != nil {
		return "", err
	}
	sessionsDir := filepath.Join(".", "vgt_workspace", "sessions")
	baseDir, err := filepath.Abs(sessionsDir)
	if err != nil {
		return "", err
	}
	target, err := filepath.Abs(filepath.Join(baseDir, "session_"+id+".json"))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(baseDir, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("session path escaped jail")
	}
	return target, nil
}

type FileChange struct {
	Path    string `json:"path"`
	File    string `json:"file"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
}

var (
	currentSessionChanges   []FileChange
	currentSessionChangesMu sync.Mutex
)

func recordFileChange(path string, added, removed int) {
	currentSessionChangesMu.Lock()
	defer currentSessionChangesMu.Unlock()

	for i, change := range currentSessionChanges {
		if change.Path == path {
			currentSessionChanges[i].Added += added
			currentSessionChanges[i].Removed += removed
			return
		}
	}

	currentSessionChanges = append(currentSessionChanges, FileChange{
		Path:    path,
		File:    filepath.Base(path),
		Added:   added,
		Removed: removed,
	})
}
