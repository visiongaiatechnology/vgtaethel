//go:build windows

package security

import "fmt"

// protectVaultDataKey uses the Windows user-bound DPAPI store. A copied vault
// no longer contains the material required to decrypt its own secrets.
func protectVaultDataKey(key []byte) ([]byte, error) {
	protected, err := dpapiProtect(key)
	if err != nil {
		return nil, fmt.Errorf("DPAPI vault key protection failed: %w", err)
	}
	return protected, nil
}

func unprotectVaultDataKey(protected []byte) ([]byte, error) {
	key, err := dpapiUnprotect(protected)
	if err != nil {
		return nil, fmt.Errorf("DPAPI vault key opening failed: %w", err)
	}
	return key, nil
}
