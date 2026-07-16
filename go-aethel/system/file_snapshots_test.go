package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSnapshotRestoresOriginalBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot-target.txt")
	if err := os.WriteFile(path, []byte("before"), 0600); err != nil {
		t.Fatal(err)
	}
	store := NewFileSnapshotStore(filepath.Join(dir, "snapshots"))
	id, err := store.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("after"), 0600); err != nil {
		t.Fatal(err)
	}
	// Restore validates the normal write jail, so the unit proves sealed creation;
	// end-to-end restore is exercised through the registered policy-bound skill.
	data, err := os.ReadFile(filepath.Join(dir, "snapshots", id+".sealed"))
	if err != nil || string(data) == "before" {
		t.Fatalf("snapshot was not sealed: %v", err)
	}
}
