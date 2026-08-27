package security

// STATUS: DIAMANT VGT SUPREME

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type ProcessLimits struct {
	Timeout          time.Duration
	MaximumCPUTime   time.Duration
	MemoryBytes      uint64
	MaximumProcesses uint32
	MaximumOutput    int
	AllowNetwork     bool
	EgressHosts      []string
}

type ProcessRequest struct {
	Executable  string
	Arguments   []string
	WorkingDir  string
	Environment []string
	Limits      ProcessLimits
}

type ProcessResult struct {
	PID       int
	Output    string
	ExitError string
	StartedAt time.Time
	EndedAt   time.Time
}

type ProcessBroker struct {
	mu       sync.Mutex
	active   map[int]*os.Process
	disabled atomic.Bool
}

var DefaultProcessBroker = NewProcessBroker()

func NewProcessBroker() *ProcessBroker {
	return &ProcessBroker{active: make(map[int]*os.Process)}
}

func (broker *ProcessBroker) Execute(ctx context.Context, request ProcessRequest) (ProcessResult, error) {
	if broker == nil || broker.disabled.Load() {
		return ProcessResult{}, errors.New("process broker kill switch is active")
	}
	request, err := validateProcessRequest(request)
	if err != nil {
		return ProcessResult{}, err
	}
	isolatedTemp, err := os.MkdirTemp("", "aethel-process-*")
	if err != nil {
		return ProcessResult{}, errors.New("isolated process temp directory unavailable")
	}
	defer os.RemoveAll(isolatedTemp)
	if err := os.Chmod(isolatedTemp, 0700); err != nil {
		return ProcessResult{}, err
	}
	timeoutContext, cancel := context.WithTimeout(ctx, request.Limits.Timeout)
	defer cancel()
	cmd := exec.CommandContext(timeoutContext, request.Executable, request.Arguments...)
	cmd.Dir = request.WorkingDir
	cmd.Env = append(sanitizeProcessEnvironment(request.Environment), "TEMP="+isolatedTemp, "TMP="+isolatedTemp, "AETHEL_NETWORK="+networkPolicyValue(request.Limits))
	if err := configureRestrictedProcess(cmd); err != nil {
		return ProcessResult{}, err
	}
	output := &boundedProcessBuffer{limit: request.Limits.MaximumOutput}
	cmd.Stdout, cmd.Stderr = output, output
	startedAt := time.Now().UTC()
	if err := cmd.Start(); err != nil {
		releaseRestrictedProcess(cmd)
		return ProcessResult{}, errors.New("isolated process failed to start")
	}
	releaseRestrictedProcess(cmd)
	job, err := attachProcessLimits(cmd.Process.Pid, request.Limits)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return ProcessResult{}, fmt.Errorf("process isolation failed closed: %w", err)
	}
	defer job.Close()
	broker.mu.Lock()
	if broker.disabled.Load() {
		broker.mu.Unlock()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return ProcessResult{}, errors.New("process broker kill switch activated during launch")
	}
	broker.active[cmd.Process.Pid] = cmd.Process
	broker.mu.Unlock()
	waitErr := cmd.Wait()
	broker.mu.Lock()
	delete(broker.active, cmd.Process.Pid)
	broker.mu.Unlock()
	result := ProcessResult{PID: cmd.Process.Pid, Output: output.String(), StartedAt: startedAt, EndedAt: time.Now().UTC()}
	if timeoutContext.Err() == context.DeadlineExceeded {
		result.ExitError = "deadline exceeded"
		return result, errors.New("isolated process deadline exceeded")
	}
	if waitErr != nil {
		result.ExitError = waitErr.Error()
		return result, errors.New("isolated process exited unsuccessfully")
	}
	return result, nil
}

func (broker *ProcessBroker) KillAll() int {
	if broker == nil {
		return 0
	}
	broker.disabled.Store(true)
	broker.mu.Lock()
	defer broker.mu.Unlock()
	killed := 0
	for pid, process := range broker.active {
		if process.Kill() == nil {
			killed++
		}
		delete(broker.active, pid)
	}
	return killed
}

func (broker *ProcessBroker) Enable() {
	if broker != nil {
		broker.disabled.Store(false)
	}
}
func (broker *ProcessBroker) Disabled() bool { return broker == nil || broker.disabled.Load() }

func validateProcessRequest(request ProcessRequest) (ProcessRequest, error) {
	request.Executable = filepath.Clean(strings.TrimSpace(request.Executable))
	if request.Executable == "." || !filepath.IsAbs(request.Executable) {
		return ProcessRequest{}, errors.New("process executable must be an absolute allowlisted path")
	}
	if len(request.Arguments) > 32 {
		return ProcessRequest{}, errors.New("process argument limit exceeded")
	}
	if request.Limits.Timeout <= 0 || request.Limits.Timeout > time.Minute {
		return ProcessRequest{}, errors.New("process timeout is outside safety boundary")
	}
	if request.Limits.MaximumCPUTime == 0 {
		request.Limits.MaximumCPUTime = request.Limits.Timeout
	}
	if request.Limits.MaximumCPUTime <= 0 || request.Limits.MaximumCPUTime > request.Limits.Timeout {
		return ProcessRequest{}, errors.New("process CPU budget is outside safety boundary")
	}
	if request.Limits.MemoryBytes < 32<<20 || request.Limits.MemoryBytes > 2<<30 {
		return ProcessRequest{}, errors.New("process memory limit is outside safety boundary")
	}
	if request.Limits.MaximumProcesses < 1 || request.Limits.MaximumProcesses > 16 {
		return ProcessRequest{}, errors.New("process count limit is outside safety boundary")
	}
	if request.Limits.MaximumOutput < 1024 || request.Limits.MaximumOutput > 4<<20 {
		return ProcessRequest{}, errors.New("process output limit is outside safety boundary")
	}
	if request.Limits.AllowNetwork {
		return ProcessRequest{}, errors.New("external processes cannot receive network authority; use the restricted network broker")
	}
	if len(request.Limits.EgressHosts) != 0 {
		return ProcessRequest{}, errors.New("egress hosts require the restricted network broker")
	}
	for _, host := range request.Limits.EgressHosts {
		if !validEgressHost(host) {
			return ProcessRequest{}, errors.New("process egress host is invalid")
		}
	}
	if request.WorkingDir == "" {
		request.WorkingDir = WorkspaceDir
	}
	resolved, err := filepath.Abs(request.WorkingDir)
	if err != nil {
		return ProcessRequest{}, errors.New("process working directory is invalid")
	}
	request.WorkingDir = resolved
	return request, nil
}

func validEgressHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" || strings.ContainsAny(host, "/\\:@?#\x00\r\n") || strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return false
			}
		}
	}
	return true
}

func sanitizeProcessEnvironment(values []string) []string {
	allowed := map[string]bool{"SYSTEMROOT": true, "WINDIR": true, "PATH": true, "GOTOOLCHAIN": true, "GOENV": true, "GOFLAGS": true, "LANG": true}
	result := make([]string, 0, len(values)+2)
	for _, value := range values {
		key, _, found := strings.Cut(value, "=")
		if found && allowed[strings.ToUpper(strings.TrimSpace(key))] && !strings.ContainsAny(value, "\x00\r\n") {
			result = append(result, value)
		}
	}
	return result
}

func networkPolicyValue(limits ProcessLimits) string {
	if !limits.AllowNetwork {
		return "deny"
	}
	return "allow:" + strings.Join(limits.EgressHosts, ",")
}

type boundedProcessBuffer struct {
	buffer bytes.Buffer
	limit  int
}

func (buffer *boundedProcessBuffer) Write(payload []byte) (int, error) {
	originalLength := len(payload)
	remaining := buffer.limit - buffer.buffer.Len()
	if remaining > 0 {
		if len(payload) > remaining {
			payload = payload[:remaining]
		}
		_, _ = buffer.buffer.Write(payload)
	}
	return originalLength, nil
}

func (buffer *boundedProcessBuffer) String() string { return buffer.buffer.String() }
