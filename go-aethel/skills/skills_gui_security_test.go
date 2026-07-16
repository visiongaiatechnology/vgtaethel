package skills

import (
	"strings"
	"testing"
)

func TestWindowControlRejectsScriptAndGeometryInjectionShapes(t *testing.T) {
	invalid := []GUIWindowControlArgs{
		{Action: "focus", WindowID: "1;Stop-Process -Id 1"},
		{Action: "focus", TitlePattern: "Editor\nStop-Process"},
		{Action: "move", WindowID: "123", Width: 0, Height: 800},
		{Action: "move", WindowID: "123", Width: 800, Height: 600, X: 70000},
		{Action: "close"},
		{Action: "execute", WindowID: "123"},
	}
	for _, input := range invalid {
		if err := validateWindowControlArgs(input); err == nil {
			t.Errorf("unsafe window-control input accepted: %+v", input)
		}
	}
	valid := []GUIWindowControlArgs{
		{Action: "list"},
		{Action: "focus", WindowID: "1234"},
		{Action: "focus", TitlePattern: `Visual Studio Code - Aethel "Beta"`},
		{Action: "move", WindowID: "0x00ABCDEF", X: -100, Y: 10, Width: 1280, Height: 720},
	}
	for _, input := range valid {
		if err := validateWindowControlArgs(input); err != nil {
			t.Errorf("valid window-control input rejected: %+v: %v", input, err)
		}
	}
}

func TestAppleScriptWindowTitleEscaping(t *testing.T) {
	escaped := escapeAppleScriptString(`Aethel "Beta" \\ Workspace`)
	if escaped != `Aethel \"Beta\" \\\\ Workspace` {
		t.Fatalf("AppleScript escaping mismatch: %q", escaped)
	}
}

func TestLiteralTypingCannotBecomeSendKeysShortcut(t *testing.T) {
	escaped := escapeSendKeysLiteral("hello %{F4} + ^ ~ (x)\n")
	if escaped == "hello %{F4} + ^ ~ (x){ENTER}" {
		t.Fatal("literal text retained active SendKeys metacharacters")
	}
	for _, required := range []string{"{%}", "{{}", "{}}", "{+}", "{^}", "{~}", "{(}", "{)}", "{ENTER}"} {
		if !strings.Contains(escaped, required) {
			t.Fatalf("escaped literal is missing %q: %q", required, escaped)
		}
	}
}
