//go:build windows

package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

const dpapiKeyPath = "./vgt_workspace/config.key.dpapi"

func getPlatformSecretKey() ([]byte, error) {
	if err := os.MkdirAll(filepath.Dir(dpapiKeyPath), 0700); err != nil {
		return nil, err
	}
	if protected, err := os.ReadFile(dpapiKeyPath); err == nil {
		key, err := dpapiUnprotect(protected)
		if err != nil {
			return nil, fmt.Errorf("DPAPI configuration key cannot be opened: %w", err)
		}
		if len(key) != 32 {
			return nil, errors.New("invalid DPAPI configuration key length")
		}
		return key, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	// One-time migration of the old local key. The legacy file is removed only
	// after Windows has durably protected an equivalent DPAPI value.
	legacyPath := "./vgt_workspace/config.key"
	if legacy, err := os.ReadFile(legacyPath); err == nil {
		if len(legacy) != 32 {
			return nil, errors.New("invalid legacy configuration key length")
		}
		protected, err := dpapiProtect(legacy)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(dpapiKeyPath, protected, 0600); err != nil {
			return nil, err
		}
		if err := os.Remove(legacyPath); err != nil {
			return nil, fmt.Errorf("DPAPI migration completed but legacy key removal failed: %w", err)
		}
		return legacy, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	protected, err := dpapiProtect(key)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(dpapiKeyPath, protected, 0600); err != nil {
		return nil, err
	}
	return key, nil
}

func dpapiProtect(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errors.New("cannot protect empty data")
	}
	in := windows.DataBlob{Size: uint32(len(plaintext)), Data: &plaintext[0]}
	var out windows.DataBlob
	description, err := windows.UTF16PtrFromString("VGT AETHEL configuration key")
	if err != nil {
		return nil, err
	}
	if err := windows.CryptProtectData(&in, description, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return []byte(base64.StdEncoding.EncodeToString(unsafe.Slice(out.Data, out.Size))), nil
}

func dpapiUnprotect(encoded []byte) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil || len(ciphertext) == 0 {
		return nil, errors.New("invalid DPAPI payload")
	}
	in := windows.DataBlob{Size: uint32(len(ciphertext)), Data: &ciphertext[0]}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}
