package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"runtime"
)

func GetSecretKey() ([]byte, error) {
	return getPlatformSecretKey()
}

func getLegacySecretKey() ([]byte, error) {
	hn, err := os.Hostname()
	if err != nil {
		hn = "aethel_default_host"
	}
	legacySalt := "VGT_AETHEL_HUD_PREMIUM_SALT_2026"
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s", hn, runtime.GOOS, legacySalt)))
	return hash[:], nil
}

func EncryptGCM(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key, err := GetSecretKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptGCM(cryptoText string) (string, error) {
	if cryptoText == "" {
		return "", nil
	}
	key, err := GetSecretKey()
	if err != nil {
		return "", err
	}
	plaintext, err := decryptGCMWithKey(cryptoText, key)
	if err == nil {
		return plaintext, nil
	}
	// Existing installations used a hostname-derived key. Retain read-only
	// migration support; the next settings save upgrades the ciphertext.
	legacyKey, legacyErr := getLegacySecretKey()
	if legacyErr != nil {
		return "", err
	}
	return decryptGCMWithKey(cryptoText, legacyKey)
}

func decryptGCMWithKey(cryptoText string, key []byte) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(cryptoText)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
