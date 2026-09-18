// STATUS: DIAMANT VGT SUPREME
package skills

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"go-aethel/security"
)


func TestSecurityPoCPathTraversal(t *testing.T) {
	// Setup workspace environment for test
	tempWorkspace := "./vgt_workspace"
	_ = os.MkdirAll(tempWorkspace, 0700)
	defer func() {
		// Keep structural workspace intact, do not break running server setup
	}()

	payloads := []struct {
		input string
		want  bool // true if it should be blocked
	}{
		{"..", true},
		{"../", true},
		{"../../etc/passwd", true},
		{"..\\..\\Windows\\System32\\cmd.exe", true},
		{"/absolute/path/outside", true},
		{"C:\\Windows\\win.ini", true},
		{"./normal_file.txt", false},
		{"normal_file.txt", false},
		{"subfolder/file.txt", false},
	}

	for _, tc := range payloads {
		_, err := validatePath(tc.input)
		isBlocked := err != nil
		if isBlocked != tc.want {
			t.Errorf("Path validation failed for %q: expected blocked=%t, got blocked=%t (err: %v)", tc.input, tc.want, isBlocked, err)
		}
	}
}

func TestSecurityPoCCommandInjection(t *testing.T) {
	skill := &ExecuteCommandSkill{}

	payloads := []struct {
		command string
		args    []string
		want    bool // true if it should be blocked
	}{
		{"git", []string{"status"}, false},
		{"git", []string{"status; echo hacked"}, true},
		{"git", []string{"status&echo hacked"}, true},
		{"git", []string{"status|echo hacked"}, true},
		{"git", []string{"status`echo hacked`"}, true},
		{"git", []string{"$(echo hacked)"}, true},
		{"git", []string{"status\nhello"}, true},
		{"invalid_cmd_not_in_allowlist", []string{"args"}, true},
	}

	for _, tc := range payloads {
		req := ExecArgs{
			Command: tc.command,
			Args:    tc.args,
		}
		argsJSON, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("JSON marshal failed: %v", err)
		}

		_, err = skill.Execute(argsJSON)
		isBlocked := err != nil && (strings.Contains(err.Error(), "VGT SECURITY INTERVENTION") || strings.Contains(err.Error(), "not in the approved"))
		if isBlocked != tc.want {
			t.Errorf("Command injection check failed for %s %v: expected blocked=%t, got blocked=%t (err: %v)", tc.command, tc.args, tc.want, isBlocked, err)
		}
	}
}

func TestSecurityPoCResourceIDValidation(t *testing.T) {
	payloads := []struct {
		id   string
		want bool // true if it should be blocked
	}{
		{"run_12345", false},
		{"run-12345", false},
		{"run_12345; DROP TABLE runs;", true},
		{"../../escaped", true},
		{"run_123*&^", true},
	}

	for _, tc := range payloads {
		err := security.ValidateResourceID(tc.id)
		isBlocked := err != nil
		if isBlocked != tc.want {
			t.Errorf("Resource ID validation check failed for %q: expected blocked=%t, got blocked=%t", tc.id, tc.want, isBlocked)
		}
	}
}

func TestMountFolderSkillWriteAndDuration(t *testing.T) {
	tmpDir := t.TempDir()
	var mountedDir string
	var mountedAccess security.MountAccess
	var mountedDuration time.Duration

	InitState(nil, nil, nil, nil, nil,
		func() []security.MountGrant {
			return []security.MountGrant{{Path: mountedDir, Access: mountedAccess}}
		},
		func(dir string, access security.MountAccess, duration time.Duration) error {
			mountedDir = dir
			mountedAccess = access
			mountedDuration = duration
			return nil
		},
		nil, nil,
	)

	mountSkill := &MountFolderSkill{}
	argsJSON, err := json.Marshal(map[string]interface{}{
		"path":             tmpDir,
		"access":           "write",
		"duration_minutes": 1440,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := mountSkill.Execute(argsJSON)
	if err != nil {
		t.Fatalf("expected write mount to succeed, got error: %v", err)
	}
	if mountedAccess != security.MountWrite {
		t.Fatalf("expected mounted access write, got %v", mountedAccess)
	}
	if mountedDuration != 1440*time.Minute {
		t.Fatalf("expected 1440m duration, got %v", mountedDuration)
	}
	if !strings.Contains(result, "erfolgreich eingehängt") {
		t.Fatalf("unexpected result: %s", result)
	}
}

