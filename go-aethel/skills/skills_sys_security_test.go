package skills

import (
	"strings"
	"testing"
)

func TestCommandArgumentsRejectInlineInterpretersAndShellShapes(t *testing.T) {
	invalid := []struct {
		command string
		args    []string
	}{
		{"node", []string{"--eval=process.exit(0)"}},
		{"node", []string{"-r", "./hook.js"}},
		{"python", []string{"-c", "print(1)"}},
		{"python", []string{"-m", "http.server"}},
		{"git", []string{"status;whoami"}},
		{"go", []string{"test\nwhoami"}},
	}
	for _, input := range invalid {
		if err := validateCommandArguments(input.command, input.args); err == nil {
			t.Fatalf("unsafe invocation accepted: %s %q", input.command, input.args)
		}
	}
	for _, input := range []struct {
		command string
		args    []string
	}{{"go", []string{"test", "./..."}}, {"git", []string{"status", "--short"}}, {"node", []string{"scripts/check.mjs"}}} {
		if err := validateCommandArguments(input.command, input.args); err != nil {
			t.Fatalf("valid invocation rejected: %s %q: %v", input.command, input.args, err)
		}
	}
}

func TestRestrictedCommandEnvironmentDoesNotInheritSecrets(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "must-not-leak")
	t.Setenv("NODE_OPTIONS", "--require=evil.js")
	environment := strings.Join(restrictedCommandEnvironment(), "\n")
	if strings.Contains(environment, "must-not-leak") || strings.Contains(environment, "NODE_OPTIONS") {
		t.Fatalf("sensitive process environment leaked: %s", environment)
	}
	if !strings.Contains(environment, "PATH=") || !strings.Contains(environment, "GOTOOLCHAIN=go1.26.5+auto") {
		t.Fatalf("required restricted environment missing: %s", environment)
	}
}
