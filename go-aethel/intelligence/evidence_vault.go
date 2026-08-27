package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go-aethel/security"
)

const maxSnapshotBytes = 32 << 20

var snapshotIDPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

type EvidenceVault struct {
	root string
}

func NewEvidenceVault(root string) (*EvidenceVault, error) {
	root = filepath.Clean(root)
	if root == "." || root == "" {
		return nil, errors.New("evidence vault root is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absolute, 0700); err != nil {
		return nil, err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, err
	}
	return &EvidenceVault{root: resolved}, nil
}

func (vault *EvidenceVault) Capture(sourceID, originalURL, finalURL, mimeType string, headers map[string]string, fetchedAt time.Time, raw []byte) (SnapshotRecord, error) {
	if vault == nil || vault.root == "" {
		return SnapshotRecord{}, errors.New("evidence vault unavailable")
	}
	if strings.TrimSpace(sourceID) == "" {
		return SnapshotRecord{}, errors.New("snapshot source is required")
	}
	if len(raw) == 0 || len(raw) > maxSnapshotBytes {
		return SnapshotRecord{}, errors.New("snapshot size boundary violation")
	}
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	id := contentSHA256(string(raw))
	objectPath, err := vault.safePath("objects", id[:2], id+".snapshot")
	if err != nil {
		return SnapshotRecord{}, err
	}
	manifestPath, err := vault.safePath("manifests", id[:2], id+".json")
	if err != nil {
		return SnapshotRecord{}, err
	}
	if err := os.MkdirAll(filepath.Dir(objectPath), 0700); err != nil {
		return SnapshotRecord{}, err
	}
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0700); err != nil {
		return SnapshotRecord{}, err
	}
	if _, err := os.Stat(objectPath); os.IsNotExist(err) {
		if err := security.WriteSealedFile(objectPath, raw); err != nil {
			return SnapshotRecord{}, err
		}
	} else if err != nil {
		return SnapshotRecord{}, err
	}
	record := SnapshotRecord{
		ID:              id,
		SourceID:        strings.TrimSpace(sourceID),
		OriginalURL:     strings.TrimSpace(originalURL),
		FinalURL:        strings.TrimSpace(finalURL),
		MIMEType:        strings.TrimSpace(mimeType),
		FetchedAt:       fetchedAt.UTC(),
		RawSHA256:       id,
		SizeBytes:       int64(len(raw)),
		ResponseHeaders: boundedResponseHeaders(headers),
		ObjectPath:      filepath.ToSlash(filepath.Join("objects", id[:2], id+".snapshot")),
		ManifestPath:    filepath.ToSlash(filepath.Join("manifests", id[:2], id+".json")),
	}
	manifest, err := json.Marshal(record)
	if err != nil {
		return SnapshotRecord{}, err
	}
	if err := security.WriteSealedFile(manifestPath, manifest); err != nil {
		return SnapshotRecord{}, err
	}
	return record, nil
}

func (vault *EvidenceVault) Read(snapshotID string) ([]byte, SnapshotRecord, error) {
	if !snapshotIDPattern.MatchString(snapshotID) {
		return nil, SnapshotRecord{}, errors.New("invalid snapshot identifier")
	}
	objectPath, err := vault.safePath("objects", snapshotID[:2], snapshotID+".snapshot")
	if err != nil {
		return nil, SnapshotRecord{}, err
	}
	manifestPath, err := vault.safePath("manifests", snapshotID[:2], snapshotID+".json")
	if err != nil {
		return nil, SnapshotRecord{}, err
	}
	raw, _, err := security.ReadSealedFile(objectPath)
	if err != nil {
		return nil, SnapshotRecord{}, err
	}
	manifest, _, err := security.ReadSealedFile(manifestPath)
	if err != nil {
		return nil, SnapshotRecord{}, err
	}
	var record SnapshotRecord
	if err := json.Unmarshal(manifest, &record); err != nil {
		return nil, SnapshotRecord{}, err
	}
	if record.ID != snapshotID || record.RawSHA256 != contentSHA256(string(raw)) || int64(len(raw)) != record.SizeBytes {
		return nil, SnapshotRecord{}, errors.New("snapshot integrity verification failed")
	}
	return raw, record, nil
}

func (vault *EvidenceVault) safePath(parts ...string) (string, error) {
	joined := filepath.Join(append([]string{vault.root}, parts...)...)
	relative, err := filepath.Rel(vault.root, joined)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("snapshot path escaped vault jail")
	}
	return joined, nil
}

func boundedResponseHeaders(headers map[string]string) map[string]string {
	allowed := map[string]bool{"content-type": true, "content-length": true, "etag": true, "last-modified": true, "date": true, "cache-control": true}
	result := make(map[string]string)
	for name, value := range headers {
		key := strings.ToLower(strings.TrimSpace(name))
		if !allowed[key] {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) > 1024 {
			value = value[:1024]
		}
		result[key] = value
	}
	return result
}
