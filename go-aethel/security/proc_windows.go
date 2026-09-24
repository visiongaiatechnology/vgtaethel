//go:build windows

package security

import "syscall"

// HideWindowSysProcAttr returns a Windows SysProcAttr configured to hide the child console window.
func HideWindowSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}
