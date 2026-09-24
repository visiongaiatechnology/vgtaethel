//go:build windows

package voice

import (
	"os/exec"
	"syscall"
)

func hideCommandWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
