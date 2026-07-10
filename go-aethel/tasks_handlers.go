package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	currentChecklist   []map[string]interface{}
	currentChecklistMu sync.RWMutex
)

func handleChecklist(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		currentChecklistMu.RLock()
		defer currentChecklistMu.RUnlock()
		if currentChecklist == nil {
			json.NewEncoder(w).Encode([]interface{}{})
		} else {
			json.NewEncoder(w).Encode(currentChecklist)
		}
		return
	}

	if r.Method == http.MethodPost {
		var req []map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		currentChecklistMu.Lock()
		currentChecklist = req
		currentChecklistMu.Unlock()
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}
}

func handleKernelTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		list := state.tasks.GetAll()
		if list == nil {
			list = []TaskItem{}
		}
		json.NewEncoder(w).Encode(list)
		return
	} else if r.Method == http.MethodPost {
		var req struct {
			Text                 string   `json:"text"`
			Objective            string   `json:"objective"`
			ScheduleType         string   `json:"schedule_type"`
			IntervalSeconds      int      `json:"interval_seconds"`
			RequiredCapabilities []string `json:"required_capabilities"`
			RiskLevel            string   `json:"risk_level"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Text == "" {
			http.Error(w, "Task description is required", http.StatusBadRequest)
			return
		}

		id := fmt.Sprintf("task_%d", time.Now().UnixNano())
		objective := req.Objective
		if objective == "" {
			objective = req.Text
		}
		scheduleType := req.ScheduleType
		if scheduleType == "" {
			scheduleType = "once"
		}
		riskLevel := req.RiskLevel
		if riskLevel == "" {
			riskLevel = "Moderate"
		}

		task := TaskItem{
			ID:                   id,
			Text:                 req.Text,
			Objective:            objective,
			Done:                 false,
			Status:               "pending",
			ScheduleType:         scheduleType,
			IntervalSeconds:      req.IntervalSeconds,
			RequiredCapabilities: req.RequiredCapabilities,
			RiskLevel:            riskLevel,
			LimitSteps:           5,
			LimitToolCalls:       10,
		}

		err := state.tasks.Add(task)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "success", "id": id})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleKernelTasksPath(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := r.URL.Path
	path = strings.TrimPrefix(path, "/v1/kernel/tasks/")

	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		handleKernelTasks(w, r)
		return
	}

	id := parts[0]

	if len(parts) == 2 {
		action := parts[1]
		if action == "run" && r.Method == http.MethodPost {
			err := state.tasks.TriggerManual(id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "success"})
			return
		}
		if action == "pause" && r.Method == http.MethodPost {
			err := state.tasks.Pause(id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "success"})
			return
		}
	}

	if r.Method == http.MethodDelete {
		err := state.tasks.Delete(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}
