package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go-aethel/security"
	_ "modernc.org/sqlite"
)

const maximumIndexedTermsPerRecord = 4096

type encryptedSearchIndex struct {
	mu       sync.Mutex
	path     string
	key      []byte
	revision uint64
}

type indexRecord struct{ recordType, recordID, content string }

func newEncryptedSearchIndex(storePath string) (*encryptedSearchIndex, error) {
	if strings.TrimSpace(storePath) == "" {
		return nil, errors.New("search index requires a canonical store path")
	}
	absolute, err := filepath.Abs(storePath + ".fts.sqlite")
	if err != nil {
		return nil, err
	}
	keyPath := storePath + ".fts.key"
	key, _, err := security.ReadSealedFile(keyPath)
	if err != nil {
		key = make([]byte, 32)
		if _, err = rand.Read(key); err != nil {
			return nil, err
		}
		if err = security.WriteSealedFile(keyPath, key); err != nil {
			return nil, err
		}
	}
	if len(key) != 32 {
		return nil, errors.New("search index key has an invalid size")
	}
	index := &encryptedSearchIndex{path: absolute, key: append([]byte(nil), key...)}
	if err := index.initialize(); err != nil {
		return nil, err
	}
	return index, nil
}

func (index *encryptedSearchIndex) open() (*sql.DB, error) {
	database, err := sql.Open("sqlite", index.path)
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(1)
	database.SetConnMaxLifetime(time.Minute)
	return database, nil
}

func (index *encryptedSearchIndex) initialize() error {
	index.mu.Lock()
	defer index.mu.Unlock()
	database, err := index.open()
	if err != nil {
		return err
	}
	defer database.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	statements := []string{
		"PRAGMA journal_mode=WAL", "PRAGMA synchronous=FULL", "PRAGMA secure_delete=ON",
		"PRAGMA trusted_schema=OFF", "PRAGMA busy_timeout=5000",
		`CREATE TABLE IF NOT EXISTS search_meta (name TEXT PRIMARY KEY, value INTEGER NOT NULL) STRICT`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS search_terms USING fts5(record_type UNINDEXED, record_id UNINDEXED, tokens, tokenize='unicode61')`,
	}
	for _, statement := range statements {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	_ = os.Chmod(index.path, 0o600)
	return nil
}

func (index *encryptedSearchIndex) sync(state StoreState) error {
	index.mu.Lock()
	defer index.mu.Unlock()
	if index.revision == state.Revision {
		return nil
	}
	database, err := index.open()
	if err != nil {
		return err
	}
	defer database.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	transaction, err := database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	if _, err := transaction.ExecContext(ctx, "DELETE FROM search_terms"); err != nil {
		return err
	}
	statement, err := transaction.PrepareContext(ctx, "INSERT INTO search_terms(record_type, record_id, tokens) VALUES(?, ?, ?)")
	if err != nil {
		return err
	}
	defer statement.Close()
	for _, record := range searchableIndexRecords(state) {
		tokens := index.tokenizeContent(record.content)
		if tokens == "" {
			continue
		}
		if _, err := statement.ExecContext(ctx, record.recordType, record.recordID, tokens); err != nil {
			return err
		}
	}
	if _, err := transaction.ExecContext(ctx, "INSERT INTO search_meta(name, value) VALUES('revision', ?) ON CONFLICT(name) DO UPDATE SET value=excluded.value", state.Revision); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return err
	}
	index.revision = state.Revision
	return nil
}

func (index *encryptedSearchIndex) query(terms []string, limit int) (map[string]bool, error) {
	index.mu.Lock()
	defer index.mu.Unlock()
	if len(terms) == 0 {
		return nil, errors.New("search index query has no terms")
	}
	if limit < 1 || limit > 1000 {
		limit = 500
	}
	queryTokens := make([]string, 0, len(terms))
	for _, term := range terms {
		queryTokens = append(queryTokens, index.token(term))
	}
	database, err := index.open()
	if err != nil {
		return nil, err
	}
	defer database.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := database.QueryContext(ctx, "SELECT record_type, record_id FROM search_terms WHERE search_terms MATCH ? LIMIT ?", strings.Join(queryTokens, " OR "), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]bool)
	for rows.Next() {
		var recordType, recordID string
		if err := rows.Scan(&recordType, &recordID); err != nil {
			return nil, err
		}
		result[recordType+":"+recordID] = true
	}
	return result, rows.Err()
}

func (index *encryptedSearchIndex) tokenizeContent(content string) string {
	terms := searchTerms(content)
	if len(terms) > maximumIndexedTermsPerRecord {
		terms = terms[:maximumIndexedTermsPerRecord]
	}
	tokens := make([]string, 0, len(terms))
	for _, term := range terms {
		tokens = append(tokens, index.token(term))
	}
	return strings.Join(tokens, " ")
}

func (index *encryptedSearchIndex) token(term string) string {
	mac := hmac.New(sha256.New, index.key)
	_, _ = mac.Write([]byte(strings.ToLower(strings.TrimSpace(term))))
	return "t" + hex.EncodeToString(mac.Sum(nil))
}

func searchableIndexRecords(state StoreState) []indexRecord {
	records := make([]indexRecord, 0, len(state.Observations)+len(state.Passages)+len(state.Events)+len(state.Claims)+len(state.Cases)*2)
	for _, item := range state.Observations {
		records = append(records, indexRecord{"observation", item.ID, item.RawText})
	}
	for _, item := range state.Passages {
		records = append(records, indexRecord{"passage", item.ID, item.Text})
	}
	for _, item := range state.Events {
		records = append(records, indexRecord{"event", item.ID, item.Title + " " + item.Summary})
	}
	for _, item := range state.Claims {
		records = append(records, indexRecord{"claim", item.ID, item.Subject + " " + item.Predicate + " " + item.Object + " " + item.Statement})
	}
	for _, item := range state.Cases {
		records = append(records, indexRecord{"case", item.ID, item.Title + " " + item.Purpose})
		for _, evidence := range item.Evidence {
			records = append(records, indexRecord{"evidence", evidence.ID, evidence.Excerpt})
		}
		for _, entity := range item.Entities {
			records = append(records, indexRecord{"entity", entity.ID, entity.Label + " " + entity.Kind})
		}
	}
	return records
}

func (index *encryptedSearchIndex) verifyNoPlaintext(forbidden string) error {
	forbidden = strings.TrimSpace(forbidden)
	if forbidden == "" {
		return nil
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		raw, err := os.ReadFile(index.path + suffix)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if strings.Contains(string(raw), forbidden) {
			return fmt.Errorf("search index leaked plaintext into %s", filepath.Base(index.path+suffix))
		}
	}
	return nil
}
