package security

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestSecretVaultMigratesLegacyPlaintextKeyToProtectedEnvelope(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "vault.key")
	legacyKey := make([]byte, 32)
	if _, err := rand.Read(legacyKey); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, legacyKey, 0600); err != nil {
		t.Fatal(err)
	}
	vault, err := NewSecretVault(filepath.Join(dir, "vault.enc"), legacyPath)
	if err != nil {
		t.Fatalf("vault migration failed: %v", err)
	}
	if len(vault.key) != 32 {
		t.Fatal("migrated vault key missing")
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy plaintext key remains: %v", err)
	}
	protected, err := os.ReadFile(legacyPath + ".protected")
	if err != nil || len(protected) == 0 {
		t.Fatalf("protected vault envelope missing: %v", err)
	}
	for index := 0; index+len(legacyKey) <= len(protected); index++ {
		match := true
		for offset := range legacyKey {
			if protected[index+offset] != legacyKey[offset] {
				match = false
				break
			}
		}
		if match {
			t.Fatal("protected vault envelope contains raw key material")
		}
	}
}
