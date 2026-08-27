package security

// STATUS: DIAMANT VGT SUPREME

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestProcessBrokerRejectsUnboundedNetworkAndKillSwitch(t *testing.T) {
	request := ProcessRequest{
		Executable: filepath.Join(t.TempDir(), "tool.exe"),
		Limits:     ProcessLimits{Timeout: time.Second, MemoryBytes: 64 << 20, MaximumProcesses: 1, MaximumOutput: 4096, AllowNetwork: true},
	}
	if _, err := validateProcessRequest(request); err == nil {
		t.Fatal("network process without an egress allowlist was accepted")
	}
	request.Limits.EgressHosts = []string{"api.example.test"}
	if _, err := validateProcessRequest(request); err == nil || !strings.Contains(err.Error(), "network broker") {
		t.Fatal("process broker accepted network authority that belongs to the restricted network broker")
	}
	request.Limits.AllowNetwork = false
	request.Limits.EgressHosts = nil
	broker := NewProcessBroker()
	broker.KillAll()
	if _, err := broker.Execute(context.Background(), request); err == nil || !strings.Contains(err.Error(), "kill switch") {
		t.Fatal("process broker kill switch did not fail closed")
	}
}

func TestProcessBrokerExecutesWithResourceIsolation(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows restricted-token and Job Object contract")
	}
	broker := NewProcessBroker()
	result, err := broker.Execute(context.Background(), ProcessRequest{
		Executable:  `C:\Windows\System32\whoami.exe`,
		WorkingDir:  t.TempDir(),
		Environment: []string{"PATH=C:\\Windows\\System32", "SECRET_SHOULD_NOT_LEAK=value"},
		Limits:      ProcessLimits{Timeout: 10 * time.Second, MemoryBytes: 64 << 20, MaximumProcesses: 1, MaximumOutput: 4096},
	})
	if err != nil || result.PID <= 0 || strings.TrimSpace(result.Output) == "" {
		t.Fatalf("isolated process failed: result=%+v err=%v", result, err)
	}
	if strings.Contains(result.Output, "SECRET_SHOULD_NOT_LEAK") {
		t.Fatal("secret environment data leaked into isolated process output")
	}
}
