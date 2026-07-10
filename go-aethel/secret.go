package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type SecretItem struct {
	ID               string   `json:"id"`
	Service          string   `json:"service"`
	Type             string   `json:"type"` // "api_token" | "password" | "key"
	AllowedActions   []string `json:"allowed_actions"`
	AllowedTargets   []string `json:"allowed_targets"`
	RequiresApproval bool     `json:"requires_approval"`
	Token            string   `json:"token,omitempty"` // cleartext token (cleared for frontend logs)
	CreatedAt        string   `json:"created_at"`
	LastUsedAt       string   `json:"last_used_at"`
	RotationHint     string   `json:"rotation_hint"`
}

type SecretVault struct {
	mu       sync.RWMutex
	filePath string
	keyPath  string
	key      []byte
	secrets  map[string]SecretItem
}

func NewSecretVault(filePath, keyPath string) (*SecretVault, error) {
	vault := &SecretVault{
		filePath: filePath,
		keyPath:  keyPath,
		secrets:  make(map[string]SecretItem),
	}

	err := vault.initKey()
	if err != nil {
		return nil, err
	}

	err = vault.load()
	if err != nil {
		// If encrypted vault doesn't exist yet, we ignore error
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return vault, nil
}

func (sv *SecretVault) initKey() error {
	_ = os.MkdirAll(filepath.Dir(sv.keyPath), 0700)

	key, err := os.ReadFile(sv.keyPath)
	if err == nil {
		if len(key) != 32 {
			return fmt.Errorf("invalid vault key length")
		}
		sv.key = key
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read vault key: %w", err)
	}

	// Generate random 256-bit key
	newKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, newKey); err != nil {
		return fmt.Errorf("failed to generate random key: %v", err)
	}

	err = os.WriteFile(sv.keyPath, newKey, 0600)
	if err != nil {
		return fmt.Errorf("failed to save key: %v", err)
	}

	sv.key = newKey
	return nil
}

func (sv *SecretVault) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(sv.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (sv *SecretVault) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(sv.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (sv *SecretVault) load() error {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	ciphertext, err := os.ReadFile(sv.filePath)
	if err != nil {
		return err
	}

	plaintext, err := sv.decrypt(ciphertext)
	if err != nil {
		return fmt.Errorf("failed to decrypt vault: %v (is key corrupt?)", err)
	}

	var list []SecretItem
	if err := json.Unmarshal(plaintext, &list); err != nil {
		return err
	}

	sv.secrets = make(map[string]SecretItem)
	for _, item := range list {
		sv.secrets[item.ID] = item
	}

	return nil
}

func (sv *SecretVault) save() error {
	var list []SecretItem
	for _, item := range sv.secrets {
		list = append(list, item)
	}

	plaintext, err := json.Marshal(list)
	if err != nil {
		return err
	}

	ciphertext, err := sv.encrypt(plaintext)
	if err != nil {
		return err
	}

	_ = os.MkdirAll(filepath.Dir(sv.filePath), 0700)
	return os.WriteFile(sv.filePath, ciphertext, 0600)
}

func (sv *SecretVault) Add(item SecretItem) error {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	item.CreatedAt = time.Now().Format(time.RFC3339)
	item.LastUsedAt = "never"
	sv.secrets[item.ID] = item

	return sv.save()
}

func (sv *SecretVault) Delete(id string) error {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	if _, ok := sv.secrets[id]; !ok {
		return fmt.Errorf("secret not found")
	}

	delete(sv.secrets, id)
	return sv.save()
}

func (sv *SecretVault) List() []SecretItem {
	sv.mu.RLock()
	defer sv.mu.RUnlock()

	var list []SecretItem
	for _, item := range sv.secrets {
		// Clean the token so it isn't returned in listings
		item.Token = ""
		list = append(list, item)
	}
	return list
}

func (sv *SecretVault) GetToken(id string) (string, error) {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	item, ok := sv.secrets[id]
	if !ok {
		return "", fmt.Errorf("secret not found")
	}

	item.LastUsedAt = time.Now().Format(time.RFC3339)
	sv.secrets[id] = item
	_ = sv.save() // save update last used time asynchronously/silently

	return item.Token, nil
}
