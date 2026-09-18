//go:build !windows

// STATUS: DIAMANT VGT SUPREME
package coder

import "os/exec"

func configureBackgroundProcess(*exec.Cmd) {}
