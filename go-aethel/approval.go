package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ApprovalGrant binds one operator decision to one exact action. Only the hash
// of the bearer token is persisted, so a restart never exposes reusable grants.
type ApprovalGrant struct {
	ID         string     `json:"id"`
	TokenHash  string     `json:"token_hash"`
	ToolName   string     `json:"tool_name"`
	ArgsHash   string     `json:"args_hash"`
	Capability Capability `json:"capability"`
	RunID      string     `json:"run_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	ConsumedAt *time.Time `json:"consumed_at,omitempty"`
}

type ApprovalManager struct {
	mu       sync.Mutex
	filePath string
	grants   map[string]ApprovalGrant
}

func NewApprovalManager(filePath string) *ApprovalManager {
	m := &ApprovalManager{filePath: filePath, grants: make(map[string]ApprovalGrant)}
	_ = m.load()
	return m
}

func (m *ApprovalManager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := os.ReadFile(m.filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var list []ApprovalGrant
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, grant := range list {
		if grant.ConsumedAt == nil && now.Before(grant.ExpiresAt) {
			m.grants[grant.ID] = grant
		}
	}
	return m.saveLocked()
}

func (m *ApprovalManager) saveLocked() error {
	list := make([]ApprovalGrant, 0, len(m.grants))
	for _, grant := range m.grants {
		list = append(list, grant)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0700); err != nil {
		return err
	}
	return os.WriteFile(m.filePath, data, 0600)
}

func digestApprovalArgs(args string) string {
	digest := sha256.Sum256([]byte(args))
	return hex.EncodeToString(digest[:])
}

func newApprovalToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

// Issue returns the only plaintext copy of a short-lived approval token.
func (m *ApprovalManager) Issue(toolName, args string, capability Capability, runID string) (ApprovalGrant, string, error) {
	token, err := newApprovalToken()
	if err != nil {
		return ApprovalGrant{}, "", err
	}
	tokenHash := sha256.Sum256([]byte(token))
	id, err := newApprovalToken()
	if err != nil {
		return ApprovalGrant{}, "", err
	}
	now := time.Now().UTC()
	grant := ApprovalGrant{
		ID:         "approval_" + id[:24],
		TokenHash:  hex.EncodeToString(tokenHash[:]),
		ToolName:   toolName,
		ArgsHash:   digestApprovalArgs(args),
		Capability: capability,
		RunID:      runID,
		CreatedAt:  now,
		ExpiresAt:  now.Add(5 * time.Minute),
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.grants[grant.ID] = grant
	if err := m.saveLocked(); err != nil {
		delete(m.grants, grant.ID)
		return ApprovalGrant{}, "", err
	}
	return grant, token, nil
}

// Consume atomically validates and invalidates a grant. A bearer token cannot
// be replayed, substituted for another action, or reused after its deadline.
func (m *ApprovalManager) Consume(token, toolName, args string, capability Capability, runID string) error {
	if len(token) != 64 {
		return errors.New("invalid approval token")
	}
	tokenHash := sha256.Sum256([]byte(token))
	hashHex := hex.EncodeToString(tokenHash[:])
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	for id, grant := range m.grants {
		if grant.TokenHash != hashHex {
			continue
		}
		if grant.ConsumedAt != nil || !now.Before(grant.ExpiresAt) {
			return errors.New("approval token expired or already consumed")
		}
		if grant.ToolName != toolName || grant.ArgsHash != digestApprovalArgs(args) || grant.Capability != capability || grant.RunID != runID {
			return errors.New("approval token does not match this action")
		}
		grant.ConsumedAt = &now
		m.grants[id] = grant
		return m.saveLocked()
	}
	return errors.New("approval token not found")
}
