//go:build !windows

package security

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const securityKeyDirectoryEnv = "AETHEL_SECURITY_KEY_DIR"

var platformKeyMu sync.Mutex

func platformSecurityKeyDirectory() (string, error) {
	configured := os.Getenv(securityKeyDirectoryEnv)
	if configured == "" {
		if strings.HasSuffix(strings.ToLower(filepath.Base(os.Args[0])), ".test") {
			return filepath.Join(os.TempDir(), "vgt-aethel-test-keys-"+fmt.Sprint(os.Getpid())), nil
		}
		return filepath.Clean("./vgt_workspace"), nil
	}
	if !filepath.IsAbs(configured) {
		return "", errors.New("AETHEL_SECURITY_KEY_DIR must be absolute")
	}
	return filepath.Clean(configured), nil
}

// Non-Windows builds retain a restricted local key file until an OS key-store
// backend is added for that platform. Desktop release targets use Windows DPAPI.
func getPlatformSecretKey() ([]byte, error) {
	platformKeyMu.Lock()
	defer platformKeyMu.Unlock()
	keyDirectory, err := platformSecurityKeyDirectory()
	if err != nil {
		return nil, err
	}
	keyPath := filepath.Join(keyDirectory, "config.key")
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
	if err := writePrivateFileAtomic(keyPath, key); err != nil {
		return nil, err
	}
	return key, nil
}
