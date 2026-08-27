package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-aethel/security"
)

type StateJournalEntry struct {
	Revision     uint64    `json:"revision"`
	StateSHA256  string    `json:"state_sha256"`
	PreviousHash string    `json:"previous_hash,omitempty"`
	EventHash    string    `json:"event_hash"`
	RecordedAt   time.Time `json:"recorded_at"`
	Anchor       bool      `json:"anchor"`
}

func appendStateJournal(storePath string, revision uint64, state []byte) error {
	directory := storePath + ".journal"
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return err
	}
	stateDigest := sha256.Sum256(state)
	stateHash := hex.EncodeToString(stateDigest[:])
	path := filepath.Join(directory, fmt.Sprintf("%020d.entry", revision))
	if existing, err := readStateJournalEntry(path); err == nil {
		if existing.Revision == revision && existing.StateSHA256 == stateHash {
			return nil
		}
		return errors.New("state journal append would overwrite a conflicting revision")
	} else if !os.IsNotExist(err) {
		return err
	}
	previousPath := filepath.Join(directory, fmt.Sprintf("%020d.entry", revision-1))
	previous, previousErr := readStateJournalEntry(previousPath)
	anchor := os.IsNotExist(previousErr)
	if previousErr != nil && !anchor {
		return previousErr
	}
	if anchor {
		files, err := os.ReadDir(directory)
		if err != nil {
			return err
		}
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".entry") {
				return errors.New("state journal revision sequence is invalid")
			}
		}
	}
	entry := StateJournalEntry{Revision: revision, StateSHA256: stateHash, RecordedAt: time.Now().UTC(), Anchor: anchor}
	if !anchor {
		entry.PreviousHash = previous.EventHash
	}
	entry.EventHash = hashStateJournalEntry(entry)
	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return security.WriteSealedFile(path, payload)
}

func readStateJournalEntry(path string) (StateJournalEntry, error) {
	payload, sealed, err := security.ReadSealedFile(path)
	if err != nil {
		return StateJournalEntry{}, err
	}
	if !sealed {
		return StateJournalEntry{}, errors.New("state journal entry is not sealed")
	}
	var entry StateJournalEntry
	if json.Unmarshal(payload, &entry) != nil {
		return StateJournalEntry{}, errors.New("state journal entry is invalid")
	}
	return entry, nil
}

func VerifyStateJournal(storePath string, stateRevision uint64, state []byte) error {
	entries, err := readStateJournal(storePath + ".journal")
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return errors.New("state journal is empty")
	}
	previousHash := ""
	for index, entry := range entries {
		if index == 0 {
			if !entry.Anchor || entry.PreviousHash != "" {
				return errors.New("state journal anchor is invalid")
			}
		} else if entry.Revision != entries[index-1].Revision+1 || entry.PreviousHash != previousHash || entry.Anchor {
			return errors.New("state journal chain is invalid")
		}
		if entry.EventHash == "" || entry.EventHash != hashStateJournalEntry(entry) {
			return errors.New("state journal event hash is invalid")
		}
		previousHash = entry.EventHash
	}
	latest := entries[len(entries)-1]
	if latest.Revision != stateRevision {
		return errors.New("state snapshot and journal revisions diverge")
	}
	digest := sha256.Sum256(state)
	if latest.StateSHA256 != hex.EncodeToString(digest[:]) {
		return errors.New("state snapshot digest does not match journal")
	}
	return nil
}

func readStateJournal(directory string) ([]StateJournalEntry, error) {
	files, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(files))
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".entry") {
			continue
		}
		base := strings.TrimSuffix(file.Name(), ".entry")
		if len(base) != 20 {
			return nil, errors.New("state journal contains an invalid filename")
		}
		if _, err := strconv.ParseUint(base, 10, 64); err != nil {
			return nil, errors.New("state journal contains an invalid revision")
		}
		paths = append(paths, filepath.Join(directory, file.Name()))
	}
	sort.Strings(paths)
	entries := make([]StateJournalEntry, 0, len(paths))
	for _, path := range paths {
		payload, sealed, err := security.ReadSealedFile(path)
		if err != nil || !sealed {
			return nil, errors.New("state journal entry failed sealed-file verification")
		}
		var entry StateJournalEntry
		if json.Unmarshal(payload, &entry) != nil {
			return nil, errors.New("state journal entry is invalid")
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func hashStateJournalEntry(entry StateJournalEntry) string {
	canonical, _ := json.Marshal(struct {
		Revision     uint64    `json:"revision"`
		StateSHA256  string    `json:"state_sha256"`
		PreviousHash string    `json:"previous_hash"`
		RecordedAt   time.Time `json:"recorded_at"`
		Anchor       bool      `json:"anchor"`
	}{entry.Revision, entry.StateSHA256, entry.PreviousHash, entry.RecordedAt, entry.Anchor})
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:])
}
