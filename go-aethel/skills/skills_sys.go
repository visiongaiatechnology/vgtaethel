package skills

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"go-aethel/security"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type ExecuteCommandSkill struct{}

type ExecArgs struct {
	Command    string   `json:"command"`
	Args       []string `json:"args"`
	Background bool     `json:"background,omitempty"`
}

func (s *ExecuteCommandSkill) Name() string { return "sys_exec_cmd" }
func (s *ExecuteCommandSkill) Description() string {
	return "Führt ein fest installiertes, freigegebenes Entwicklerwerkzeug mit getrennten Argumenten aus. Shells und Inline-Code sind gesperrt; jede Ausführung ist sicherheitskritisch."
}
func (s *ExecuteCommandSkill) RiskLevel() security.RiskLevel { return security.RiskCritical }

func (s *ExecuteCommandSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command":    map[string]interface{}{"type": "string", "enum": []string{"git", "go", "node", "npm", "python", "rg", "where", "whoami", "tasklist"}, "description": "Fest installiertes Werkzeug."},
			"args":       map[string]interface{}{"type": "array", "maxItems": 32, "items": map[string]interface{}{"type": "string", "maxLength": 2048}, "description": "Getrennte Argumente ohne Shell-Syntax."},
			"background": map[string]interface{}{"type": "boolean", "description": "Nur für explizit freigegebene Node- oder npm-Prozesse."},
		},
		"required":             []string{"command", "args"},
		"additionalProperties": false,
	}
}

func (s *ExecuteCommandSkill) Execute(args json.RawMessage) (string, error) {
	var input ExecArgs
	decoder := json.NewDecoder(strings.NewReader(string(args)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return "", err
	}
	command := strings.ToLower(strings.TrimSpace(input.Command))
	if !approvedSystemCommands[command] {
		return "", errors.New("VGT SECURITY INTERVENTION: command is not in the approved system command allowlist")
	}
	if err := validateCommandArguments(command, input.Args); err != nil {
		return "", err
	}
	if input.Background && !approvedBackgroundCommands[command] {
		return "", errors.New("background execution is not allowed for this command")
	}

	cmdPath := TrustedExecutable(command)
	if cmdPath == "" {
		return "", errors.New("approved command is unavailable on this system")
	}
	auditTarget := commandArgumentDigest(command, input.Args)

	if input.Background {
		cmd := exec.Command(cmdPath, input.Args...) // #nosec G204 -- executable is a fixed allowlisted absolute path; arguments passed without a shell.
		cmd.Env = restrictedCommandEnvironment()
		if err := cmd.Start(); err != nil {
			security.LogKernelActivity("EXEC_FAILED", auditTarget, "ERROR")
			return "", errors.New("background command failed")
		}
		pid := cmd.Process.Pid
		go func() {
			_ = cmd.Wait()
		}()
		security.LogKernelActivity("EXEC_BG", auditTarget, "SUCCESS")
		return fmt.Sprintf("Approved background command started (PID: %d)", pid), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cmdPath, input.Args...) // #nosec G204 -- executable is a fixed allowlisted absolute path; arguments passed without a shell.
	cmd.Env = restrictedCommandEnvironment()
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		security.LogKernelActivity("EXEC_TIMEOUT", auditTarget, "ERROR")
		return "", errors.New("command timed out after 60 seconds")
	}
	if err != nil {
		security.LogKernelActivity("EXEC_FAILED", auditTarget, "ERROR")
		return "", fmt.Errorf("command failed: %s", clampRunDetail(string(output)))
	}
	security.LogKernelActivity("EXEC", auditTarget, "SUCCESS")
	return clampRunDetail(string(output)), nil
}

var approvedSystemCommands = map[string]bool{
	"git": true, "go": true, "node": true, "npm": true, "python": true,
	"rg": true, "where": true, "whoami": true, "tasklist": true,
}

var approvedBackgroundCommands = map[string]bool{"node": true, "npm": true}

var forbiddenInlineFlags = map[string]map[string]bool{
	"node": {
		"-e": true, "--eval": true, "-p": true, "--print": true,
		"-r": true, "--require": true, "--import": true, "--experimental-loader": true,
	},
	"python": {"-c": true, "-m": true, "-": true},
}

func validateCommandArguments(command string, args []string) error {
	if len(args) > 32 || command == "" || len(command) > 80 {
		return errors.New("invalid command shape")
	}
	for _, arg := range args {
		if len([]rune(arg)) > 2048 || strings.IndexByte(arg, 0) >= 0 || forbiddenShellMeta.MatchString(arg) {
			return errors.New("VGT SECURITY INTERVENTION: unsafe command argument rejected")
		}
		flag := strings.ToLower(strings.SplitN(arg, "=", 2)[0])
		if forbiddenInlineFlags[command][flag] {
			return fmt.Errorf("VGT SECURITY INTERVENTION: inline execution flag %s is forbidden", flag)
		}
	}
	return nil
}

func commandArgumentDigest(command string, args []string) string {
	digest := sha256.Sum256([]byte(strings.Join(append([]string{command}, args...), "\x00")))
	return fmt.Sprintf("%s sha256:%x argc:%d", command, digest, len(args))
}

func restrictedCommandEnvironment() []string {
	get := func(key string) string { return strings.TrimSpace(os.Getenv(key)) }
	values := make([]string, 0, 12)
	add := func(key, value string) {
		if value != "" && !strings.ContainsAny(value, "\r\n\x00") {
			values = append(values, key+"="+value)
		}
	}
	if runtime.GOOS == "windows" {
		add("SystemRoot", `C:\Windows`)
		add("WINDIR", `C:\Windows`)
		add("PATH", `C:\Windows\System32;C:\Program Files\Git\cmd;C:\Program Files\Go\bin;C:\Program Files\nodejs;C:\Program Files\Python312`)
		add("USERPROFILE", get("USERPROFILE"))
		add("APPDATA", get("APPDATA"))
		add("LOCALAPPDATA", get("LOCALAPPDATA"))
	} else {
		add("PATH", "/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin")
		add("HOME", get("HOME"))
	}
	add("TEMP", os.TempDir())
	add("TMP", os.TempDir())
	add("GOTOOLCHAIN", "go1.26.5+auto")
	add("GOENV", "off")
	add("GOFLAGS", "")
	return values
}

func clampRunDetail(detail string) string {
	detail = strings.TrimSpace(detail)
	if len([]rune(detail)) > 6000 {
		return string([]rune(detail)[:6000]) + "..."
	}
	return detail
}
