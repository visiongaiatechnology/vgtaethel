package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileSnapshot struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	Existed   bool      `json:"existed"`
	Content   []byte    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
type FileSnapshotStore struct {
	mu  sync.Mutex
	dir string
}

type FileSnapshotArtifact struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	Existed   bool      `json:"existed"`
	SizeBytes int       `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

var defaultFileSnapshots = NewFileSnapshotStore("./vgt_workspace/file_snapshots")

func NewFileSnapshotStore(dir string) *FileSnapshotStore { return &FileSnapshotStore{dir: dir} }
func (s *FileSnapshotStore) Create(path string) (string, error) {
	data, err := os.ReadFile(path)
	existed := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if len(data) > 5<<20 {
		return "", errors.New("file exceeds 5 MiB snapshot boundary")
	}
	raw := make([]byte, 12)
	if _, err = rand.Read(raw); err != nil {
		return "", err
	}
	id := "snap_" + hex.EncodeToString(raw)
	payload, err := json.Marshal(FileSnapshot{ID: id, Path: path, Existed: existed, Content: data, CreatedAt: time.Now().UTC()})
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err = os.MkdirAll(s.dir, 0700); err != nil {
		return "", err
	}
	return id, writeSealedFile(filepath.Join(s.dir, id+".sealed"), payload)
}
func (s *FileSnapshotStore) Restore(id string) error {
	if len(id) != 29 {
		return errors.New("invalid snapshot identifier")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, _, err := readSealedFile(filepath.Join(s.dir, id+".sealed"))
	if err != nil {
		return err
	}
	var snap FileSnapshot
	if err = json.Unmarshal(data, &snap); err != nil {
		return err
	}
	safe, err := validateWritePath(snap.Path)
	if err != nil || safe != snap.Path {
		return errors.New("snapshot path is no longer authorized")
	}
	if !snap.Existed {
		return os.Remove(safe)
	}
	return os.WriteFile(safe, snap.Content, 0600)
}

// ListArtifacts exposes only restore-relevant metadata. Snapshot contents stay
// sealed and are never returned to the browser.
func (s *FileSnapshotStore) ListArtifacts() ([]FileSnapshotArtifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, os.ErrNotExist) {
		return []FileSnapshotArtifact{}, nil
	}
	if err != nil {
		return nil, err
	}
	artifacts := make([]FileSnapshotArtifact, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sealed" {
			continue
		}
		data, _, readErr := readSealedFile(filepath.Join(s.dir, entry.Name()))
		if readErr != nil {
			continue
		}
		var snapshot FileSnapshot
		if json.Unmarshal(data, &snapshot) != nil || snapshot.ID == "" {
			continue
		}
		artifacts = append(artifacts, FileSnapshotArtifact{ID: snapshot.ID, Path: snapshot.Path, Existed: snapshot.Existed, SizeBytes: len(snapshot.Content), CreatedAt: snapshot.CreatedAt})
	}
	return artifacts, nil
}

type RestoreSnapshotSkill struct{ Store *FileSnapshotStore }

func (s *RestoreSnapshotSkill) Name() string { return "fs_restore_snapshot" }
func (s *RestoreSnapshotSkill) Description() string {
	return "Stellt einen versiegelten Datei-Snapshot nach Freigabe wieder her."
}
func (s *RestoreSnapshotSkill) RiskLevel() RiskLevel { return RiskCritical }
func (s *RestoreSnapshotSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"snapshot_id": map[string]interface{}{"type": "string"}}, "required": []string{"snapshot_id"}}
}
func (s *RestoreSnapshotSkill) Execute(args json.RawMessage) (string, error) {
	var v struct {
		ID string `json:"snapshot_id"`
	}
	if err := json.Unmarshal(args, &v); err != nil {
		return "", err
	}
	if err := s.Store.Restore(v.ID); err != nil {
		return "", err
	}
	return "Snapshot restored: " + v.ID, nil
}
