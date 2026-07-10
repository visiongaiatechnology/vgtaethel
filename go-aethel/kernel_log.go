package main

import (
	"encoding/json"
	"net/http"
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

func handleKernelLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	kernelLogsMu.RLock()
	defer kernelLogsMu.RUnlock()

	if kernelLogs == nil {
		kernelLogs = []Activity{}
	}

	json.NewEncoder(w).Encode(kernelLogs)
}
