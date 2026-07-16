//go:build !windows

package security

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
)

// Non-Windows builds retain a restricted local key file until an OS key-store
// backend is added for that platform. Desktop release targets use Windows DPAPI.
func getPlatformSecretKey() ([]byte, error) {
	const keyPath = "./vgt_workspace/config.key"
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return nil, err
	}
	key, err := os.ReadFile(keyPath)
	if err == nil {
		if len(key) != 32 {
			return nil, errors.New("invalid configuration key length")
		}
		return key, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	key = make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return nil, err
	}
	return key, nil
}
