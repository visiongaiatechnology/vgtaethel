package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const sealedStorePrefix = "AETHEL-SEAL-v1:"

// readSealedFile returns plaintext and whether the source already used the
// sealed format. Plaintext is accepted only as a migration input and is sealed
// during the next write.
func readSealedFile(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	encoded := strings.TrimPrefix(string(data), sealedStorePrefix)
	if encoded == string(data) {
		return data, false, nil
	}
	plain, err := decryptGCM(encoded)
	if err != nil {
		return nil, true, errors.New("encrypted local store cannot be opened")
	}
	return []byte(plain), true, nil
}

func writeSealedFile(path string, plaintext []byte) error {
	sealed, err := encryptGCM(string(plaintext))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(sealedStorePrefix+sealed), 0600)
}
