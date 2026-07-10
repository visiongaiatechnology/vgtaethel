package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// --- 1. SKILL: EXECUTE SYSTEM COMMAND ---

type ExecuteCommandSkill struct{}

type ExecArgs struct {
	Command    string   `json:"command"`
	Args       []string `json:"args"`
	Background bool     `json:"background,omitempty"`
}

func (s *ExecuteCommandSkill) Name() string { return "sys_exec_cmd" }
func (s *ExecuteCommandSkill) Description() string {
	return "Führt einen Systembefehl auf dem Host aus. Sicherheitskritisch."
}
func (s *ExecuteCommandSkill) RiskLevel() RiskLevel { return RiskCritical }

func (s *ExecuteCommandSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command":    map[string]interface{}{"type": "string", "description": "Der Befehl (z.B. 'dir', 'git', 'ping')"},
			"args":       map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Argumente"},
			"background": map[string]interface{}{"type": "boolean", "description": "Ob der Befehl im Hintergrund gestartet werden soll (wichtig für GUI-Apps wie Browser-Tabs, Spotify, Notepad, um Hänger zu vermeiden)"},
		},
		"required": []string{"command", "args"},
	}
}

func (s *ExecuteCommandSkill) Execute(args json.RawMessage) (string, error) {
	var input ExecArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	command := strings.ToLower(strings.TrimSpace(input.Command))
	if !approvedSystemCommands[command] {
		return "", fmt.Errorf("VGT SECURITY INTERVENTION: command is not in the approved system command allowlist")
	}
	if len(input.Args) > 32 || len([]rune(command)) == 0 || len([]rune(command)) > 80 {
		return "", errors.New("invalid command shape")
	}
	for _, arg := range input.Args {
		if len([]rune(arg)) > 2048 || forbiddenShellMeta.MatchString(arg) {
			return "", fmt.Errorf("VGT SECURITY INTERVENTION: shell metacharacters are forbidden in command arguments")
		}
	}
	if input.Background && !approvedBackgroundCommands[command] {
		return "", errors.New("background execution is not allowed for this command")
	}

	cmdPath, err := exec.LookPath(command)
	if err != nil {
		return "", fmt.Errorf("approved command is unavailable on this system")
	}
	fullCmdStr := command
	if len(input.Args) > 0 {
		fullCmdStr += " " + strings.Join(input.Args, " ")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cmdPath, input.Args...)

	if input.Background {
		if err := cmd.Start(); err != nil {
			LogKernelActivity("EXEC_FAILED", fullCmdStr, "ERROR")
			return "", fmt.Errorf("background command failed")
		}
		LogKernelActivity("EXEC_BG", fullCmdStr, "SUCCESS")
		return fmt.Sprintf("Approved background command started (PID: %d)", cmd.Process.Pid), nil
	}

	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		LogKernelActivity("EXEC_TIMEOUT", fullCmdStr, "ERROR")
		return "", errors.New("command timed out after 60 seconds")
	}
	if err != nil {
		LogKernelActivity("EXEC_FAILED", fullCmdStr, "ERROR")
		return "", fmt.Errorf("command failed: %s", clampRunDetail(string(output)))
	}
	LogKernelActivity("EXEC", fullCmdStr, "SUCCESS")
	return clampRunDetail(string(output)), nil
}

var approvedSystemCommands = map[string]bool{
	"git": true, "go": true, "node": true, "npm": true, "npx": true, "wails": true,
	"cargo": true, "rustc": true, "python": true, "python3": true, "pip": true, "pip3": true,
	"rg": true, "where": true, "whoami": true, "tasklist": true,
}

var approvedBackgroundCommands = map[string]bool{"node": true, "npm": true, "wails": true}
