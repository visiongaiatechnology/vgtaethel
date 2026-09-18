// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/sphere/object_store.go
// Purpose: High-performance, concurrent, atomic persistence layer for Universal Sphere Objects

package sphere

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go-aethel/security"
)

// ObjectStore manages all persisted Sphere objects and desktop state.
type ObjectStore struct {
	mu       sync.RWMutex
	baseDir  string
	objects  map[string]*SphereObject
	state    DesktopWorkspaceState
	isLoaded bool
}

// NewObjectStore initializes a store rooted in the secure workspace.
func NewObjectStore(baseDir string) (*ObjectStore, error) {
	if strings.TrimSpace(baseDir) == "" {
		baseDir = filepath.Join(security.WorkspaceDir, "sphere")
	}
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to initialize sphere storage directory: %w", err)
	}
	store := &ObjectStore{
		baseDir: baseDir,
		objects: make(map[string]*SphereObject),
		state: DesktopWorkspaceState{
			ActiveDesktop: DesktopPersonal,
			AmbientLevel:  12,
			Windows:       make(map[string]DesktopWindowState),
			LastSaved:     time.Now().UTC(),
		},
	}
	if err := store.loadFromDisk(); err != nil {
		// Log and continue with fresh store if load failed
		security.LogKernelActivity("SPHERE_STORE_INIT_WARN", err.Error(), "WARNING")
	}
	return store, nil
}

// loadFromDisk loads all saved objects and workspace state from the sphere directory.
func (s *ObjectStore) loadFromDisk() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Load Workspace State
	stateFile := filepath.Join(s.baseDir, "workspace_state.json")
	if data, err := os.ReadFile(stateFile); err == nil {
		var state DesktopWorkspaceState
		if err := json.Unmarshal(data, &state); err == nil {
			if state.Windows == nil {
				state.Windows = make(map[string]DesktopWindowState)
			}
			if state.ActiveDesktop == "" {
				state.ActiveDesktop = DesktopPersonal
			}
			s.state = state
		}
	}

	// 2. Load Sphere Objects
	objectsFile := filepath.Join(s.baseDir, "sphere_objects.json")
	if data, err := os.ReadFile(objectsFile); err == nil {
		var list []*SphereObject
		if err := json.Unmarshal(data, &list); err == nil {
			for _, obj := range list {
				if obj != nil && obj.ID != "" {
					s.objects[obj.ID] = obj
				}
			}
		}
	}

	s.isLoaded = true
	return nil
}

// saveObjectsAtomic writes all in-memory objects to disk atomically.
func (s *ObjectStore) saveObjectsAtomic() error {
	list := make([]*SphereObject, 0, len(s.objects))
	for _, obj := range s.objects {
		list = append(list, obj)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize sphere objects: %w", err)
	}

	targetFile := filepath.Join(s.baseDir, "sphere_objects.json")
	return writeAtomic(targetFile, data)
}

// saveStateAtomic writes the desktop workspace state to disk atomically.
func (s *ObjectStore) saveStateAtomic() error {
	s.state.LastSaved = time.Now().UTC()
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize workspace state: %w", err)
	}

	targetFile := filepath.Join(s.baseDir, "workspace_state.json")
	return writeAtomic(targetFile, data)
}

// writeAtomic writes data to a temporary file in the same directory, flushes, syncs, chmods 0600, and renames.
func writeAtomic(targetFile string, data []byte) error {
	dir := filepath.Dir(targetFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, ".sphere_tmp_*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	if err := tmpFile.Chmod(0600); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to set chmod on temp file: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Rename temp file to target file (atomic on POSIX and modern Windows)
	if err := os.Rename(tmpName, targetFile); err != nil {
		// On Windows, if target exists, remove then rename
		_ = os.Remove(targetFile)
		if err := os.Rename(tmpName, targetFile); err != nil {
			return fmt.Errorf("failed to replace %s: %w", targetFile, err)
		}
	}
	return nil
}

// Put stores or updates a SphereObject.
func (s *ObjectStore) Put(obj *SphereObject) error {
	if obj == nil {
		return errors.New("cannot store nil object")
	}
	if err := obj.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	if existing, exists := s.objects[obj.ID]; exists {
		obj.CreatedAt = existing.CreatedAt
		obj.Version = existing.Version + 1
	} else {
		if obj.CreatedAt.IsZero() {
			obj.CreatedAt = now
		}
		obj.Version = 1
	}
	obj.UpdatedAt = now

	s.objects[obj.ID] = obj
	return s.saveObjectsAtomic()
}

// Get retrieves a SphereObject by its ID.
func (s *ObjectStore) Get(id string) (*SphereObject, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	obj, exists := s.objects[id]
	if !exists {
		return nil, false
	}
	// Return a copy to avoid caller mutation without Put()
	copyObj := *obj
	return &copyObj, true
}

// Delete removes a SphereObject by its ID.
func (s *ObjectStore) Delete(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.objects[id]; !exists {
		return false, nil
	}
	delete(s.objects, id)
	err := s.saveObjectsAtomic()
	return true, err
}

// List returns all objects optionally filtered by type, desktop, or tag.
func (s *ObjectStore) List(filterType ObjectType, filterDesktop VirtualDesktop, filterTag string) []*SphereObject {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*SphereObject, 0, len(s.objects))
	for _, obj := range s.objects {
		if filterType != "" && obj.Type != filterType {
			continue
		}
		if filterDesktop != "" && obj.Desktop != "" && obj.Desktop != filterDesktop {
			continue
		}
		if filterTag != "" {
			matched := false
			for _, tag := range obj.Tags {
				if strings.EqualFold(tag, filterTag) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		copyObj := *obj
		result = append(result, &copyObj)
	}
	return result
}

// Search performs a case-insensitive search across title, summary, and tags.
func (s *ObjectStore) Search(query string, limit int) []*SphereObject {
	s.mu.RLock()
	defer s.mu.RUnlock()

	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return s.List("", "", "")
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}

	result := make([]*SphereObject, 0, limit)
	for _, obj := range s.objects {
		if len(result) >= limit {
			break
		}
		titleMatch := strings.Contains(strings.ToLower(obj.Title), q)
		summaryMatch := strings.Contains(strings.ToLower(obj.Summary), q)
		tagMatch := false
		for _, tag := range obj.Tags {
			if strings.Contains(strings.ToLower(tag), q) {
				tagMatch = true
				break
			}
		}

		if titleMatch || summaryMatch || tagMatch {
			copyObj := *obj
			result = append(result, &copyObj)
		}
	}
	return result
}

// GetWorkspaceState returns the current desktop layout state.
func (s *ObjectStore) GetWorkspaceState() DesktopWorkspaceState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// SaveWorkspaceState updates the current desktop layout state.
func (s *ObjectStore) SaveWorkspaceState(state DesktopWorkspaceState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if state.Windows == nil {
		state.Windows = make(map[string]DesktopWindowState)
	}
	if state.ActiveDesktop == "" {
		state.ActiveDesktop = DesktopPersonal
	}
	s.state = state
	return s.saveStateAtomic()
}
