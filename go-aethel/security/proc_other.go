//go:build !windows

package security

import "syscall"

// HideWindowSysProcAttr returns nil on non-Windows platforms where HideWindow does not exist.
func HideWindowSysProcAttr() *syscall.SysProcAttr {
	return nil
}
