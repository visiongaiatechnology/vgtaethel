package security

import (
	"sync"
	"time"
)

type Activity struct {
	Timestamp string `json:"timestamp"`
	Op        string `json:"op"`
	Target    string `json:"target"`
	Status    string `json:"status"`
}

var (
	kernelLogs   []Activity
	kernelLogsMu sync.RWMutex
)

func LogKernelActivity(op, target, status string) {
	kernelLogsMu.Lock()
	defer kernelLogsMu.Unlock()

	if len(target) > 100 {
		target = target[:97] + "..."
	}

	logItem := Activity{
		Timestamp: time.Now().Format("15:04:05"),
		Op:        op,
		Target:    target,
		Status:    status,
	}

	kernelLogs = append([]Activity{logItem}, kernelLogs...)
	if len(kernelLogs) > 50 {
		kernelLogs = kernelLogs[:50]
	}
}

// KernelLogs returns a copy of recent kernel activity entries for the UI.
func KernelLogs() []Activity {
	kernelLogsMu.RLock()
	defer kernelLogsMu.RUnlock()
	if kernelLogs == nil {
		return []Activity{}
	}
	out := make([]Activity, len(kernelLogs))
	copy(out, kernelLogs)
	return out
}


