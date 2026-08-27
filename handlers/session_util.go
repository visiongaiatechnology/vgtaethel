package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"go-aethel/security"
)

type sealedSessionEnvelope struct {
	Version    int    `json:"version"`
	Ciphertext string `json:"ciphertext"`
}

func sealSessionMessages(messages []json.RawMessage) ([]byte, error) {
	plaintext, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}
	ciphertext, err := security.EncryptGCM(string(plaintext))
	if err != nil {
		return nil, err
	}
	return json.Marshal(sealedSessionEnvelope{Version: 1, Ciphertext: ciphertext})
}

func openSessionMessages(data []byte) ([]json.RawMessage, error) {
	var envelope sealedSessionEnvelope
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Version == 1 && envelope.Ciphertext != "" {
		plaintext, decryptErr := security.DecryptGCM(envelope.Ciphertext)
		if decryptErr != nil {
			return nil, fmt.Errorf("session decryption failed")
		}
		var messages []json.RawMessage
		if err := json.Unmarshal([]byte(plaintext), &messages); err != nil {
			return nil, err
		}
		return messages, nil
	}
	var messages []json.RawMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

var safeSessionIDPattern = regexp.MustCompile(`^session_[A-Za-z0-9_-]{1,80}$`)

const sessionsDirectory = "./vgt_workspace/sessions"

func validateSessionID(id string) error {
	if !safeSessionIDPattern.MatchString(id) {
		return fmt.Errorf("invalid session id")
	}
	return nil
}

func sessionFileName(id string) (string, error) {
	if err := validateSessionID(id); err != nil {
		return "", err
	}
	return "session_" + id + ".json", nil
}

func openSessionsRoot() (*os.Root, error) {
	if err := os.MkdirAll(sessionsDirectory, 0700); err != nil {
		return nil, err
	}
	return os.OpenRoot(sessionsDirectory)
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

func RecordFileChange(path string, added, removed int) {
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
