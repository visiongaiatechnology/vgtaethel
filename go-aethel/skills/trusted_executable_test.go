package skills

import (
	"runtime"
	"strings"
	"testing"

	)

func TestTrustedExecutableNeverUsesBarePATHCommand(t *testing.T) {
	for _, command := range []string{"powershell.exe", "cmd.exe", "xdotool", "osascript", "not-allowed"} {
		path := TrustedExecutable(command)
		if path != "" && !strings.Contains(path, "/") && !strings.Contains(path, `\`) {
			t.Fatalf("%s resolved to bare command %q", command, path)
		}
	}
	if runtime.GOOS == "windows" && !strings.EqualFold(TrustedExecutable("powershell.exe"), `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`) {
		t.Fatalf("unexpected PowerShell executable: %q", TrustedExecutable("powershell.exe"))
	}
}
