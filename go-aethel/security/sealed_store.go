package security

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const sealedStorePrefix = "AETHEL-SEAL-v1:"

var ErrUnsealedAuthorityStore = errors.New("authority store is not authenticated")

// ReadSealedFile returns plaintext and whether the source already used the
// sealed format. Plaintext is accepted only as a migration input and is sealed
// during the next write.
func ReadSealedFile(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	encoded := strings.TrimPrefix(string(data), sealedStorePrefix)
	if encoded == string(data) {
		return data, false, nil
	}
	plain, err := DecryptGCM(encoded)
	if err != nil {
		return nil, true, errors.New("encrypted local store cannot be opened")
	}
	return []byte(plain), true, nil
}

func WriteSealedFile(path string, plaintext []byte) error {
	sealed, err := EncryptGCM(string(plaintext))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return writePrivateFileAtomic(path, []byte(sealedStorePrefix+sealed))
}

// ReadAuthorityFile refuses legacy plaintext. Permission and approval state is
// transient and must fail closed instead of trusting a forgeable migration.
func ReadAuthorityFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(string(data), sealedStorePrefix) {
		return nil, ErrUnsealedAuthorityStore
	}
	plain, _, err := ReadSealedFile(path)
	return plain, err
}

func writePrivateFileAtomic(path string, data []byte) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".aethel-sealed-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0600); err != nil {
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := replacePrivateFile(temporaryPath, path); err != nil {
		return err
	}
	committed = true
	return nil
}
