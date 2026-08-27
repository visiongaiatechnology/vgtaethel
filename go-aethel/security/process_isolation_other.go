//go:build !windows

package security

// STATUS: PLATIN

import "os/exec"

type processJob struct{}

func configureRestrictedProcess(*exec.Cmd) error                  { return nil }
func releaseRestrictedProcess(*exec.Cmd)                          {}
func attachProcessLimits(int, ProcessLimits) (*processJob, error) { return &processJob{}, nil }
func (*processJob) Close() error                                  { return nil }
