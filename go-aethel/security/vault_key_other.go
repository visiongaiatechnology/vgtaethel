//go:build !windows

package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

var vaultKeyAAD = []byte("vgt-aethel-vault-key-v1")

func protectVaultDataKey(key []byte) ([]byte, error) {
	master, err := getPlatformSecretKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(master)
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
	return gcm.Seal(nonce, nonce, key, vaultKeyAAD), nil
}

func unprotectVaultDataKey(protected []byte) ([]byte, error) {
	master, err := getPlatformSecretKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(master)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(protected) < gcm.NonceSize() {
		return nil, fmt.Errorf("protected vault key is truncated")
	}
	return gcm.Open(nil, protected[:gcm.NonceSize()], protected[gcm.NonceSize():], vaultKeyAAD)
}
