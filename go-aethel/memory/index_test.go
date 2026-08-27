package memory

// STATUS: DIAMANT VGT SUPREME

import "testing"

func TestMemorySearchUsesIncrementalUnicodeIndex(t *testing.T) {
	store := &LocalMemoryStore{entries: []MemoryEntry{
		{ID: "one", Content: "München Infrastruktur Entscheidung", Category: "decision", Tags: []string{"münchen"}},
		{ID: "two", Content: "Athens unrelated reference", Category: "reference", Tags: []string{"athens"}},
	}}
	store.rebuildIndexLocked()
	if len(store.index) != 2 || store.df["münchen"] == 0 {
		t.Fatalf("incremental index was not built: %+v", store.df)
	}
	before := len(store.index)
	results := store.Search("München Infrastruktur")
	if len(results) == 0 || results[0].ID != "one" {
		t.Fatalf("unicode memory recall failed: %+v", results)
	}
	if len(store.index) != before {
		t.Fatal("read-time search rebuilt or mutated the persistent index")
	}
}
