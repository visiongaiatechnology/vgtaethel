//go:build windows

package security

// STATUS: DIAMANT VGT SUPREME

import (
	"errors"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	disableMaxPrivilege = 0x1
	luaToken            = 0x4
)

var createRestrictedTokenProc = windows.NewLazySystemDLL("advapi32.dll").NewProc("CreateRestrictedToken")

type processJob struct {
	handle windows.Handle
}

func configureRestrictedProcess(cmd *exec.Cmd) error {
	var current windows.Token
	access := uint32(windows.TOKEN_DUPLICATE | windows.TOKEN_QUERY | windows.TOKEN_ASSIGN_PRIMARY)
	if err := windows.OpenProcessToken(windows.CurrentProcess(), access, &current); err != nil {
		return err
	}
	defer current.Close()
	var restricted windows.Token
	result, _, callErr := createRestrictedTokenProc.Call(
		uintptr(current), disableMaxPrivilege|luaToken, 0, 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(&restricted)),
	)
	if result == 0 {
		if callErr != nil {
			return callErr
		}
		return errors.New("CreateRestrictedToken failed")
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, Token: syscall.Token(restricted)}
	return nil
}

func releaseRestrictedProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.SysProcAttr == nil || cmd.SysProcAttr.Token == 0 {
		return
	}
	_ = windows.CloseHandle(windows.Handle(cmd.SysProcAttr.Token))
	cmd.SysProcAttr.Token = 0
}

func attachProcessLimits(pid int, limits ProcessLimits) (*processJob, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	cleanup := func() { _ = windows.CloseHandle(job) }
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE |
		windows.JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION |
		windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS |
		windows.JOB_OBJECT_LIMIT_PROCESS_TIME |
		windows.JOB_OBJECT_LIMIT_PROCESS_MEMORY
	info.BasicLimitInformation.PerProcessUserTimeLimit = requestDuration100Nanoseconds(limits.MaximumCPUTime)
	info.BasicLimitInformation.ActiveProcessLimit = limits.MaximumProcesses
	info.ProcessMemoryLimit = uintptr(limits.MemoryBytes)
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		cleanup()
		return nil, err
	}
	ui := windows.JOBOBJECT_BASIC_UI_RESTRICTIONS{UIRestrictionsClass: windows.JOB_OBJECT_UILIMIT_HANDLES | windows.JOB_OBJECT_UILIMIT_READCLIPBOARD | windows.JOB_OBJECT_UILIMIT_WRITECLIPBOARD | windows.JOB_OBJECT_UILIMIT_SYSTEMPARAMETERS | windows.JOB_OBJECT_UILIMIT_DESKTOP | windows.JOB_OBJECT_UILIMIT_DISPLAYSETTINGS | windows.JOB_OBJECT_UILIMIT_EXITWINDOWS | windows.JOB_OBJECT_UILIMIT_GLOBALATOMS}
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectBasicUIRestrictions, uintptr(unsafe.Pointer(&ui)), uint32(unsafe.Sizeof(ui))); err != nil {
		cleanup()
		return nil, err
	}
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		cleanup()
		return nil, err
	}
	defer windows.CloseHandle(process)
	if err := windows.AssignProcessToJobObject(job, process); err != nil {
		cleanup()
		return nil, err
	}
	return &processJob{handle: job}, nil
}

func requestDuration100Nanoseconds(value time.Duration) int64 {
	return int64(value / (100 * time.Nanosecond))
}

func (job *processJob) Close() error {
	if job == nil || job.handle == 0 {
		return nil
	}
	err := windows.CloseHandle(job.handle)
	job.handle = 0
	return err
}
