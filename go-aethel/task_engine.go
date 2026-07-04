package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type TaskItem struct {
	ID                   string   `json:"id"`
	Text                 string   `json:"text"` // Display text / title
	Objective            string   `json:"objective"`
	Done                 bool     `json:"done"`
	Status               string   `json:"status"` // "pending" | "running" | "waiting_for_user" | "blocked" | "done" | "failed"
	ScheduleType         string   `json:"schedule_type"` // "once" | "interval" | "cron"
	IntervalSeconds      int      `json:"interval_seconds,omitempty"`
	CronExpression       string   `json:"cron_expression,omitempty"`
	NextRunAt            string   `json:"next_run_at,omitempty"`
	RequiredCapabilities []string `json:"required_capabilities"`
	RiskLevel            string   `json:"risk_level"`
	LimitSteps           int      `json:"limit_steps"`
	LimitToolCalls       int      `json:"limit_tool_calls"`
	CreatedAt            string   `json:"created_at"`
	UpdatedAt            string   `json:"updated_at"`
	LastRunAt            string   `json:"last_run_at"`
	LastReport           string   `json:"last_report"`
	AuditRefs            []string `json:"audit_refs"`
}

type TaskEngine struct {
	mu        sync.Mutex
	filePath  string
	tasks     []TaskItem
	stopChan  chan struct{}
	isRunning bool
}

func NewTaskEngine(filePath string) *TaskEngine {
	return &TaskEngine{
		filePath: filePath,
		stopChan: make(chan struct{}),
	}
}

func (te *TaskEngine) Load() error {
	te.mu.Lock()
	defer te.mu.Unlock()

	data, err := os.ReadFile(te.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			te.tasks = []TaskItem{}
			return nil
		}
		return err
	}

	var rawItems []json.RawMessage
	if err := json.Unmarshal(data, &rawItems); err != nil {
		return err
	}

	te.tasks = []TaskItem{}
	for _, raw := range rawItems {
		var item TaskItem
		// Parse basic fields
		if err := json.Unmarshal(raw, &item); err == nil {
			// Backwards compatibility defaults
			if item.Status == "" {
				if item.Done {
					item.Status = "done"
				} else {
					item.Status = "pending"
				}
			}
			if item.ScheduleType == "" {
				item.ScheduleType = "once"
			}
			if item.LimitSteps == 0 {
				item.LimitSteps = 5
			}
			if item.LimitToolCalls == 0 {
				item.LimitToolCalls = 10
			}
			te.tasks = append(te.tasks, item)
		}
	}

	return nil
}

func (te *TaskEngine) Save() error {
	data, err := json.MarshalIndent(te.tasks, "", "  ")
	if err != nil {
		return err
	}

	_ = os.MkdirAll(filepath.Dir(te.filePath), 0755)
	return os.WriteFile(te.filePath, data, 0644)
}

func (te *TaskEngine) Add(item TaskItem) error {
	te.mu.Lock()
	defer te.mu.Unlock()

	item.CreatedAt = time.Now().Format(time.RFC3339)
	item.UpdatedAt = item.CreatedAt
	item.LastRunAt = "never"
	item.LastReport = "Created task."
	
	if item.ScheduleType == "interval" && item.IntervalSeconds > 0 {
		item.NextRunAt = time.Now().Add(time.Duration(item.IntervalSeconds) * time.Second).Format(time.RFC3339)
	} else {
		item.NextRunAt = time.Now().Format(time.RFC3339) // Run immediately
	}

	te.tasks = append(te.tasks, item)
	return te.Save()
}

func (te *TaskEngine) Delete(id string) error {
	te.mu.Lock()
	defer te.mu.Unlock()

	found := false
	var updated []TaskItem
	for _, t := range te.tasks {
		if t.ID == id {
			found = true
			continue
		}
		updated = append(updated, t)
	}

	if !found {
		return fmt.Errorf("task not found")
	}

	te.tasks = updated
	return te.Save()
}

func (te *TaskEngine) GetAll() []TaskItem {
	te.mu.Lock()
	defer te.mu.Unlock()
	return te.tasks
}

func (te *TaskEngine) Start() {
	te.mu.Lock()
	if te.isRunning {
		te.mu.Unlock()
		return
	}
	te.isRunning = true
	te.mu.Unlock()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-te.stopChan:
				return
			case <-ticker.C:
				te.checkAndRunTasks()
			}
		}
	}()
}

func (te *TaskEngine) Stop() {
	te.mu.Lock()
	defer te.mu.Unlock()
	if te.isRunning {
		close(te.stopChan)
		te.isRunning = false
	}
}

func (te *TaskEngine) TriggerManual(id string) error {
	te.mu.Lock()
	var target *TaskItem
	for i := range te.tasks {
		if te.tasks[i].ID == id {
			target = &te.tasks[i]
			break
		}
	}
	te.mu.Unlock()

	if target == nil {
		return fmt.Errorf("task not found")
	}

	go te.runTask(target)
	return nil
}

func (te *TaskEngine) Pause(id string) error {
	te.mu.Lock()
	defer te.mu.Unlock()

	for i := range te.tasks {
		if te.tasks[i].ID == id {
			te.tasks[i].Status = "blocked"
			te.tasks[i].LastReport = "Paused by operator."
			return te.Save()
		}
	}
	return fmt.Errorf("task not found")
}

func (te *TaskEngine) checkAndRunTasks() {
	te.mu.Lock()
	now := time.Now()
	var dueTasks []*TaskItem

	for i := range te.tasks {
		t := &te.tasks[i]
		if t.Done || t.Status == "running" || t.Status == "blocked" {
			continue
		}

		if t.NextRunAt != "" {
			parsedTime, err := time.Parse(time.RFC3339, t.NextRunAt)
			if err == nil && now.After(parsedTime) {
				t.Status = "running"
				dueTasks = append(dueTasks, t)
			}
		}
	}
	_ = te.Save()
	te.mu.Unlock()

	for _, task := range dueTasks {
		go te.runTask(task)
	}
}

func (te *TaskEngine) runTask(task *TaskItem) {
	defer func() {
		te.mu.Lock()
		task.UpdatedAt = time.Now().Format(time.RFC3339)
		if task.ScheduleType == "interval" && task.IntervalSeconds > 0 {
			task.Status = "pending"
			task.NextRunAt = time.Now().Add(time.Duration(task.IntervalSeconds) * time.Second).Format(time.RFC3339)
		} else {
			task.Done = true
			task.Status = "done"
			task.NextRunAt = ""
		}
		_ = te.Save()
		te.mu.Unlock()
	}()

	LogKernelActivity("TASK_START", task.ID, "RUNNING")
	
	// Deriving background execution loop
	apiKey := state.getAPIKey()
	if apiKey == "" {
		task.LastReport = "Fehlgeschlagen: Kein API-Schlüssel konfiguriert."
		task.Status = "failed"
		return
	}

	task.LastRunAt = time.Now().Format(time.RFC3339)
	task.LastReport = "Executing task agent loop..."

	// Simple background step execution simulated using LLM completion
	messages := []map[string]string{
		{"role": "system", "content": "Du bist VGT AETHEL, ein autonomer Task-Agent im Hintergrund. Du hast das Ziel: " + task.Objective + "\nVerwende die verfügbaren Skills."},
		{"role": "user", "content": "Führe die nächste Aktion aus, um das Ziel zu erreichen. Antworte in JSON mit {\"action\": \"tool_name\", \"args\": {}} oder {\"report\": \"Zusammenfassung des Ergebnisses\"}."},
	}

	stepCount := 0
	toolCallCount := 0

	for stepCount < task.LimitSteps && toolCallCount < task.LimitToolCalls {
		stepCount++
		
		payload := map[string]interface{}{
			"model":       "llama3-70b-8192",
			"messages":    messages,
			"temperature": 0.1,
		}
		
		jsonBytes, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPost, groqURL, bytes.NewBuffer(jsonBytes))
		if err != nil {
			break
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			break
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			break
		}

		var apiResult struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		
		_ = json.NewDecoder(resp.Body).Decode(&apiResult)
		if len(apiResult.Choices) == 0 {
			break
		}

		content := apiResult.Choices[0].Message.Content
		messages = append(messages, map[string]string{"role": "assistant", "content": content})

		var responseParse struct {
			Action string                 `json:"action"`
			Args   map[string]interface{} `json:"args"`
			Report string                 `json:"report"`
		}

		if err := json.Unmarshal([]byte(content), &responseParse); err != nil {
			// Fallback: search for simple report text
			task.LastReport = content
			break
		}

		if responseParse.Report != "" {
			task.LastReport = responseParse.Report
			break
		}

		if responseParse.Action != "" {
			toolCallCount++
			
			argsBytes, _ := json.Marshal(responseParse.Args)
			argsStr := string(argsBytes)

			// Intercept with policy engine
			allowed, decision, report := state.policy.Evaluate(responseParse.Action, argsStr, false)
			
			if !allowed {
				task.Status = "blocked"
				task.LastReport = fmt.Sprintf("Blocked by Security Firewall (%s): missing lease for capability '%s' or threat warning detected: %v", decision, report.Capability, report.Threats)
				LogKernelActivity("TASK_BLOCKED", task.ID, "BLOCKED")
				return
			}

			// Execute tool call safely
			skill, ok := state.skills.Get(responseParse.Action)
			if !ok {
				messages = append(messages, map[string]string{"role": "user", "content": "Fehler: Skill nicht gefunden."})
				continue
			}

			result, err := skill.Execute(argsBytes)
			var resultSummary string
			if err != nil {
				resultSummary = "Error: " + err.Error()
			} else {
				resultSummary = fmt.Sprintf("%v", result)
			}

			// Log to cryptographic audit logger
			auditID, _ := state.audit.Log("aethel", responseParse.Action, task.ID, report.RiskLevel, "", "allowed", "Task automation lease bypass", argsStr)
			task.AuditRefs = append(task.AuditRefs, auditID)

			messages = append(messages, map[string]string{"role": "user", "content": "Tool Output: " + resultSummary})
		}
	}

	LogKernelActivity("TASK_COMPLETE", task.ID, "SUCCESS")
}
