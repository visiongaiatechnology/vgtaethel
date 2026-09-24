//go:build !windows

package voice

import "os/exec"

func hideCommandWindow(_ *exec.Cmd) {}
