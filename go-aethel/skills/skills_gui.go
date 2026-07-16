package skills

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"unicode"

	"go-aethel/security"
)

// --- 5d. SKILL: VISION CONTEXT ---

type VisionContextSkill struct{}

type VisionContextArgs struct {
	Action string `json:"action"`
}

func (s *VisionContextSkill) Name() string { return "vision_context" }
func (s *VisionContextSkill) Description() string {
	return "Liefert Kontext zum sichtbaren Desktop: screenshot, status oder windows. Für echte Bildanalyse wird der aktuelle Screenshot automatisch an visionfähige Chat-Modelle gehängt."
}
func (s *VisionContextSkill) RiskLevel() security.RiskLevel { return security.RiskModerate }

func (s *VisionContextSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"screenshot", "status", "windows"},
				"description": "Kontext-Aktion: screenshot aktualisieren, viewport status, oder sichtbare Fenster listen.",
			},
		},
		"required": []string{"action"},
	}
}

func (s *VisionContextSkill) Execute(args json.RawMessage) (string, error) {
	var input VisionContextArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	switch input.Action {
	case "screenshot":
		if err := CapturePrimaryDisplay(); err != nil {
			security.LogKernelActivity("VISION_SCREENSHOT_FAILED", "", "ERROR")
			return "", err
		}
		data, err := GetLatestScreenshot()
		if err != nil {
			return "", err
		}
		security.LogKernelActivity("VISION_SCREENSHOT", fmt.Sprintf("%d bytes", len(data)), "SUCCESS")
		return fmt.Sprintf("Desktop-Screenshot aktualisiert (%d Bytes). Wenn das aktuelle Modell Vision unterstützt, wird der Screenshot bei visuellen Fragen automatisch an die nächste Modellanfrage angehängt.", len(data)), nil
	case "status":
		data, err := GetLatestScreenshot()
		if err != nil {
			return "Viewport aktiv, aber Screenshot konnte nicht gelesen werden.", nil
		}
		return fmt.Sprintf("Viewport aktiv. Letzter Screenshot verfügbar (%d Bytes).", len(data)), nil
	case "windows":
		return (&GUIWindowControlSkill{}).Execute([]byte(`{"action":"list"}`))
	default:
		return "", fmt.Errorf("unbekannte Vision-Aktion: %s", input.Action)
	}
}

// --- 6. SKILL: GUI CONTROL ---

type GUIControlSkill struct{}

type GUIControlArgs struct {
	Action string `json:"action"` // "move", "click", "type", "press", "position"
	X      int    `json:"x,omitempty"`
	Y      int    `json:"y,omitempty"`
	Button string `json:"button,omitempty"` // "left", "right", "double"
	Text   string `json:"text,omitempty"`
	Keys   string `json:"keys,omitempty"` // e.g. "{ENTER}", "{TAB}", "^t", "^w"
}

func escapeSendKeysLiteral(value string) string {
	var escaped strings.Builder
	escaped.Grow(len(value) + 16)
	for _, character := range value {
		switch character {
		case '\r':
			continue
		case '\n':
			escaped.WriteString("{ENTER}")
		case '\t':
			escaped.WriteString("{TAB}")
		case '{':
			escaped.WriteString("{{}")
		case '}':
			escaped.WriteString("{}}")
		case '+', '^', '%', '~', '(', ')', '[', ']':
			escaped.WriteByte('{')
			escaped.WriteRune(character)
			escaped.WriteByte('}')
		default:
			escaped.WriteRune(character)
		}
	}
	return escaped.String()
}

func (s *GUIControlSkill) Name() string { return "gui_control" }
func (s *GUIControlSkill) Description() string {
	return "Steuert Maus, Tastatur und GUI-Elemente (Tabs, Tastenanschläge, Klicks). Sicherheitskritisch."
}
func (s *GUIControlSkill) RiskLevel() security.RiskLevel { return security.RiskCritical }

func (s *GUIControlSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"move", "click", "type", "press", "position"},
				"description": "Die auszuführende GUI-Aktion: 'move' (Maus bewegen), 'click' (Klicken), 'type' (Text tippen), 'press' (Spezialtasten/Shortcuts drücken), 'position' (Maus-Koordinaten und Auflösung abfragen)",
			},
			"x":      map[string]interface{}{"type": "integer", "description": "Absolute X-Koordinate (für 'move')"},
			"y":      map[string]interface{}{"type": "integer", "description": "Absolute Y-Koordinate (für 'move')"},
			"button": map[string]interface{}{"type": "string", "enum": []string{"left", "right", "double"}, "description": "Maustaste (für 'click')"},
			"text":   map[string]interface{}{"type": "string", "description": "Text, der getippt werden soll (für 'type')"},
			"keys":   map[string]interface{}{"type": "string", "description": "Tastenbezeichnung oder Tastenkombination wie '{ENTER}', '{TAB}', '^t' (Strg+T für neuen Tab), '^w' (Strg+W), '%{TAB}' (Alt+Tab) (für 'press')"},
		},
		"required": []string{"action"},
	}
}

func (s *GUIControlSkill) Execute(args json.RawMessage) (string, error) {
	var input GUIControlArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	if input.X < 0 || input.Y < 0 || input.X > 16384 || input.Y > 16384 {
		return "", errors.New("GUI coordinates are outside the allowed desktop range")
	}
	if len([]rune(input.Text)) > 4096 || len([]rune(input.Keys)) > 64 {
		return "", errors.New("GUI input exceeds the allowed size limit")
	}
	if strings.IndexFunc(input.Text, func(r rune) bool { return unicode.IsControl(r) && r != '\r' && r != '\n' && r != '\t' }) >= 0 || strings.IndexFunc(input.Keys, unicode.IsControl) >= 0 {
		return "", errors.New("GUI input contains forbidden control characters")
	}
	switch input.Action {
	case "move", "click", "type", "press", "position":
	default:
		return "", fmt.Errorf("unbekannte GUI-Aktion: %s", input.Action)
	}
	if input.Action == "click" && input.Button != "" && input.Button != "left" && input.Button != "right" && input.Button != "double" {
		return "", errors.New("unsupported mouse button")
	}

	defer func() {
		_ = CapturePrimaryDisplay()
	}()

	if runtime.GOOS == "linux" {
		switch input.Action {
		case "move":
			cmd := exec.Command(TrustedExecutable("xdotool"), "mousemove", fmt.Sprintf("%d", input.X), fmt.Sprintf("%d", input.Y))
			if err := cmd.Run(); err != nil {
				security.LogKernelActivity("GUI_MOVE_FAILED", fmt.Sprintf("x:%d, y:%d", input.X, input.Y), "ERROR")
				return "", err
			}
			security.LogKernelActivity("GUI_MOVE", fmt.Sprintf("x:%d, y:%d", input.X, input.Y), "SUCCESS")
			return fmt.Sprintf("Mauszeiger erfolgreich zu X:%d Y:%d bewegt.", input.X, input.Y), nil

		case "click":
			if input.X != 0 || input.Y != 0 {
				if err := exec.Command(TrustedExecutable("xdotool"), "mousemove", fmt.Sprintf("%d", input.X), fmt.Sprintf("%d", input.Y)).Run(); err != nil {
					return "", err
				}
			}
			var clickBtn string
			if input.Button == "right" {
				clickBtn = "3"
			} else {
				clickBtn = "1"
			}
			var cmd *exec.Cmd
			if input.Button == "double" {
				cmd = exec.Command(TrustedExecutable("xdotool"), "doubleclick", "1")
			} else {
				cmd = exec.Command(TrustedExecutable("xdotool"), "click", clickBtn)
			}
			if err := cmd.Run(); err != nil {
				security.LogKernelActivity("GUI_CLICK_FAILED", input.Button, "ERROR")
				return "", err
			}
			security.LogKernelActivity("GUI_CLICK", input.Button, "SUCCESS")
			return fmt.Sprintf("Klick mit Taste '%s' erfolgreich ausgeführt (X:%d Y:%d).", input.Button, input.X, input.Y), nil

		case "type":
			if input.Text == "" {
				return "", errors.New("kein text zum tippen übergeben")
			}
			cmd := exec.Command(TrustedExecutable("xdotool"), "type", input.Text)
			if err := cmd.Run(); err != nil {
				security.LogKernelActivity("GUI_TYPE_FAILED", input.Text, "ERROR")
				return "", err
			}
			security.LogKernelActivity("GUI_TYPE", input.Text, "SUCCESS")
			return "Text erfolgreich getippt.", nil

		case "press":
			if input.Keys == "" {
				return "", errors.New("keine tastenbezeichnung zum drücken übergeben")
			}
			keyStr := input.Keys
			keyStr = strings.ReplaceAll(keyStr, "{ENTER}", "Return")
			keyStr = strings.ReplaceAll(keyStr, "{TAB}", "Tab")
			keyStr = strings.ReplaceAll(keyStr, "{ESC}", "Escape")
			if strings.HasPrefix(keyStr, "^") && len(keyStr) == 2 {
				keyStr = "ctrl+" + strings.ToLower(string(keyStr[1]))
			}
			cmd := exec.Command(TrustedExecutable("xdotool"), "key", keyStr)
			if err := cmd.Run(); err != nil {
				security.LogKernelActivity("GUI_PRESS_FAILED", input.Keys, "ERROR")
				return "", err
			}
			security.LogKernelActivity("GUI_PRESS", input.Keys, "SUCCESS")
			return fmt.Sprintf("Taste/Shortcut '%s' erfolgreich gedrückt.", input.Keys), nil

		case "position":
			outLoc, errLoc := exec.Command(TrustedExecutable("xdotool"), "getmouselocation", "--shell").Output()
			if errLoc != nil {
				security.LogKernelActivity("GUI_POSITION_FAILED", "", "ERROR")
				return "", errLoc
			}
			outGeom, errGeom := exec.Command(TrustedExecutable("xdotool"), "getdisplaygeometry").Output()
			if errGeom != nil {
				security.LogKernelActivity("GUI_POSITION_FAILED", "", "ERROR")
				return "", errGeom
			}

			var posX, posY string
			for _, line := range strings.Split(string(outLoc), "\n") {
				if strings.HasPrefix(line, "X=") {
					posX = strings.TrimPrefix(line, "X=")
				} else if strings.HasPrefix(line, "Y=") {
					posY = strings.TrimPrefix(line, "Y=")
				}
			}

			geomParts := strings.Fields(string(outGeom))
			var width, height string
			if len(geomParts) >= 2 {
				width = geomParts[0]
				height = geomParts[1]
			} else {
				width = "1920"
				height = "1080"
			}

			security.LogKernelActivity("GUI_POSITION", fmt.Sprintf("x:%s, y:%s", posX, posY), "SUCCESS")
			return fmt.Sprintf("Maus-Position: X:%s, Y:%s | Bildschirmauflösung: %sx%s", posX, posY, width, height), nil
		}
		return "", fmt.Errorf("unbekannte GUI-Aktion: %s", input.Action)
	}

	switch input.Action {
	case "move":
		psScript := fmt.Sprintf(`
			Add-Type -AssemblyName System.Windows.Forms;
			[System.Windows.Forms.Cursor]::Position = New-Object System.Drawing.Point(%d, %d);
			$pos = [System.Windows.Forms.Cursor]::Position;
			Write-Output ($pos.X.ToString() + ';' + $pos.Y.ToString())
		`, input.X, input.Y)
		cmd := exec.Command(security.GetPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		out, err := cmd.CombinedOutput()
		if err != nil {
			security.LogKernelActivity("GUI_MOVE_FAILED", fmt.Sprintf("x:%d, y:%d", input.X, input.Y), "ERROR")
			return "", err
		}
		parts := strings.Split(strings.TrimSpace(string(out)), ";")
		if len(parts) >= 2 && parts[0] == fmt.Sprintf("%d", input.X) && parts[1] == fmt.Sprintf("%d", input.Y) {
			security.LogKernelActivity("GUI_MOVE", fmt.Sprintf("x:%d, y:%d", input.X, input.Y), "SUCCESS")
			return fmt.Sprintf("Mauszeiger erfolgreich zu X:%d Y:%d bewegt. [SEMANTIC VERIFICATION PASSED: Mouse coordinates match]", input.X, input.Y), nil
		}
		security.LogKernelActivity("GUI_MOVE", fmt.Sprintf("x:%d, y:%d", input.X, input.Y), "SUCCESS")
		return fmt.Sprintf("Mauszeiger erfolgreich zu X:%d Y:%d bewegt.", input.X, input.Y), nil

	case "click":
		var mouseEventCode string
		switch input.Button {
		case "right":
			mouseEventCode = "[Mouse]::mouse_event(0x0008, 0, 0, 0, 0); [Mouse]::mouse_event(0x0010, 0, 0, 0, 0);"
		case "double":
			mouseEventCode = "[Mouse]::mouse_event(0x0002, 0, 0, 0, 0); [Mouse]::mouse_event(0x0004, 0, 0, 0, 0); [Mouse]::mouse_event(0x0002, 0, 0, 0, 0); [Mouse]::mouse_event(0x0004, 0, 0, 0, 0);"
		default: // "left"
			mouseEventCode = "[Mouse]::mouse_event(0x0002, 0, 0, 0, 0); [Mouse]::mouse_event(0x0004, 0, 0, 0, 0);"
		}

		moveToTarget := ""
		if input.X != 0 || input.Y != 0 {
			moveToTarget = fmt.Sprintf("[System.Windows.Forms.Cursor]::Position = New-Object System.Drawing.Point(%d, %d);", input.X, input.Y)
		}
		psScript := fmt.Sprintf(`
			Add-Type -TypeDefinition @'
			using System;
			using System.Runtime.InteropServices;
			public class Mouse {
				[DllImport("user32.dll")]
				public static extern void mouse_event(int dwFlags, int dx, int dy, int cButtons, int dwExtraInfo);
			}
'@;
			Add-Type -AssemblyName System.Windows.Forms;
			%s
			%s
		`, moveToTarget, mouseEventCode)
		cmd := exec.Command(security.GetPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		err := cmd.Run()
		if err != nil {
			security.LogKernelActivity("GUI_CLICK_FAILED", input.Button, "ERROR")
			return "", err
		}
		security.LogKernelActivity("GUI_CLICK", input.Button, "SUCCESS")
		return fmt.Sprintf("Klick mit Taste '%s' erfolgreich ausgeführt (X:%d Y:%d).", input.Button, input.X, input.Y), nil

	case "type":
		if input.Text == "" {
			return "", errors.New("kein text zum tippen übergeben")
		}
		escapedText := strings.ReplaceAll(escapeSendKeysLiteral(input.Text), "'", "''")
		psScript := fmt.Sprintf(`
			Add-Type -AssemblyName System.Windows.Forms;
			[System.Windows.Forms.SendKeys]::SendWait('%s');
		`, escapedText)
		cmd := exec.Command(security.GetPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		err := cmd.Run()
		if err != nil {
			security.LogKernelActivity("GUI_TYPE_FAILED", input.Text, "ERROR")
			return "", err
		}
		security.LogKernelActivity("GUI_TYPE", input.Text, "SUCCESS")
		return "Text erfolgreich getippt.", nil

	case "press":
		if input.Keys == "" {
			return "", errors.New("keine tastenbezeichnung zum drücken übergeben")
		}
		escapedKeys := strings.ReplaceAll(input.Keys, "'", "''")
		psScript := fmt.Sprintf(`
			Add-Type -AssemblyName System.Windows.Forms;
			[System.Windows.Forms.SendKeys]::SendWait('%s');
		`, escapedKeys)
		cmd := exec.Command(security.GetPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		err := cmd.Run()
		if err != nil {
			security.LogKernelActivity("GUI_PRESS_FAILED", input.Keys, "ERROR")
			return "", err
		}
		security.LogKernelActivity("GUI_PRESS", input.Keys, "SUCCESS")
		return fmt.Sprintf("Taste/Shortcut '%s' erfolgreich gedrückt.", input.Keys), nil

	case "position":
		psScript := `
			Add-Type -AssemblyName System.Windows.Forms;
			$pos = [System.Windows.Forms.Cursor]::Position;
			$scr = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds;
			Write-Output ($pos.X.ToString() + ';' + $pos.Y.ToString() + ';' + $scr.Width.ToString() + ';' + $scr.Height.ToString())
		`
		cmd := exec.Command(security.GetPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		out, err := cmd.Output()
		if err != nil {
			security.LogKernelActivity("GUI_POSITION_FAILED", "", "ERROR")
			return "", err
		}
		parts := strings.Split(strings.TrimSpace(string(out)), ";")
		if len(parts) < 4 {
			return "", errors.New("ungültige rückgabe von mouse/screen info")
		}
		security.LogKernelActivity("GUI_POSITION", "", "SUCCESS")
		return fmt.Sprintf("Maus-Position: X:%s, Y:%s | Bildschirmauflösung: %sx%s", parts[0], parts[1], parts[2], parts[3]), nil
	}
	return "", fmt.Errorf("unbekannte GUI-Aktion: %s", input.Action)
}

// --- 6b. SKILL: GUI WINDOW CONTROL ---

type GUIWindowControlSkill struct{}

type GUIWindowControlArgs struct {
	Action       string `json:"action"`                  // "list", "focus", "move", "close"
	WindowID     string `json:"window_id,omitempty"`     // Target Window ID (PID in Windows / Window ID in Linux)
	TitlePattern string `json:"title_pattern,omitempty"` // Title search pattern (fallback if no ID is passed)
	X            int    `json:"x,omitempty"`
	Y            int    `json:"y,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
}

var (
	windowIDPattern   = regexp.MustCompile(`^(?:0x[0-9A-Fa-f]{1,16}|[0-9]{1,20})$`)
	windowsPIDPattern = regexp.MustCompile(`^[0-9]{1,10}$`)
)

func validateWindowControlArgs(input GUIWindowControlArgs) error {
	switch input.Action {
	case "list", "focus", "move", "close":
	default:
		return errors.New("unsupported window action")
	}
	if input.WindowID != "" && !windowIDPattern.MatchString(input.WindowID) {
		return errors.New("window identifier has an invalid shape")
	}
	if len([]rune(input.TitlePattern)) > 160 || strings.IndexFunc(input.TitlePattern, unicode.IsControl) >= 0 {
		return errors.New("window title pattern is invalid")
	}
	if input.Action != "list" && input.WindowID == "" && strings.TrimSpace(input.TitlePattern) == "" {
		return errors.New("window action requires an identifier or title pattern")
	}
	if input.Action == "move" && (input.Width < 1 || input.Height < 1 || input.Width > 32768 || input.Height > 32768 || input.X < -65536 || input.X > 65536 || input.Y < -65536 || input.Y > 65536) {
		return errors.New("window geometry is outside the supported desktop boundary")
	}
	return nil
}

func findWindowsPIDByTitle(pattern string) (string, error) {
	const script = "Get-Process | Where-Object { $_.MainWindowTitle } | ForEach-Object { Write-Output ($_.Id.ToString() + \"`t\" + ($_.MainWindowTitle -replace \"[\\r\\n\\t]\", \" \")) }"
	cmd := exec.Command(security.GetPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.Output()
	if err != nil {
		return "", errors.New("window title inventory failed")
	}
	needle := strings.ToLower(strings.TrimSpace(pattern))
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "\t", 2)
		if len(parts) == 2 && windowIDPattern.MatchString(parts[0]) && strings.Contains(strings.ToLower(parts[1]), needle) {
			return parts[0], nil
		}
	}
	return "", errors.New("window title not found")
}

func escapeAppleScriptString(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `"`, `\"`)
}

func (s *GUIWindowControlSkill) Name() string { return "gui_window_control" }
func (s *GUIWindowControlSkill) Description() string {
	return "Erkennt, listet auf, fokussiert (Vordergrund), verschiebt oder schließt geöffnete Anwendungsfenster (z.B. Chrome, VS Code) auf dem Desktop."
}
func (s *GUIWindowControlSkill) RiskLevel() security.RiskLevel { return security.RiskCritical }

func (s *GUIWindowControlSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"list", "focus", "move", "close"},
				"description": "Die auszuführende Aktion: 'list' (alle offenen Fenster auflisten), 'focus' (ein Fenster in den Vordergrund bringen), 'move' (ein Fenster positionieren/Größe ändern), 'close' (ein Fenster schließen)",
			},
			"window_id":     map[string]interface{}{"type": "string", "description": "Die ID (PID/Handle) des Ziel-Fensters (wird bei 'list' zurückgegeben)"},
			"title_pattern": map[string]interface{}{"type": "string", "description": "Suchmuster für den Fenstertitel (z.B. 'chrome', 'editor') als Fallback/Filter"},
			"x":             map[string]interface{}{"type": "integer", "description": "Neue X-Koordinate für 'move'"},
			"y":             map[string]interface{}{"type": "integer", "description": "Neue Y-Koordinate für 'move'"},
			"width":         map[string]interface{}{"type": "integer", "description": "Neue Breite für 'move'"},
			"height":        map[string]interface{}{"type": "integer", "description": "Neue Höhe für 'move'"},
		},
		"required": []string{"action"},
	}
}

func (s *GUIWindowControlSkill) Execute(args json.RawMessage) (string, error) {
	var input GUIWindowControlArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	if err := validateWindowControlArgs(input); err != nil {
		return "", err
	}

	defer func() {
		_ = CapturePrimaryDisplay()
	}()

	if runtime.GOOS == "windows" {
		psHeader := `
			$code = @"
			using System;
			using System.Runtime.InteropServices;
			public class Window {
				[DllImport("user32.dll")]
				[return: MarshalAs(UnmanagedType.Bool)]
				public static extern bool GetWindowRect(IntPtr hWnd, out RECT lpRect);
 
				[DllImport("user32.dll")]
				[return: MarshalAs(UnmanagedType.Bool)]
				public static extern bool SetForegroundWindow(IntPtr hWnd);
 
				[DllImport("user32.dll")]
				public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
 
				[DllImport("user32.dll")]
				public static extern bool MoveWindow(IntPtr hWnd, int X, int Y, int nWidth, int nHeight, bool bRepaint);
 
				[StructLayout(LayoutKind.Sequential)]
				public struct RECT {
					public int Left;
					public int Top;
					public int Right;
					public int Bottom;
				}
			}
"@
			Add-Type -TypeDefinition $code -ErrorAction SilentlyContinue
		`

		switch input.Action {
		case "list":
			psScript := psHeader + `
				$processes = Get-Process | Where-Object { $_.MainWindowTitle }
				$results = @()
				foreach ($p in $processes) {
					$rect = New-Object Window+RECT
					if ([Window]::GetWindowRect($p.MainWindowHandle, [ref]$rect)) {
						$width = $rect.Right - $rect.Left
						$height = $rect.Bottom - $rect.Top
						$results += [PSCustomObject]@{
							id = $p.Id.ToString()
							process = $p.ProcessName
							title = $p.MainWindowTitle
							x = $rect.Left
							y = $rect.Top
							width = $width
							height = $height
						}
					}
				}
				$results | ConvertTo-Json
			`
			cmd := exec.Command(security.GetPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			out, err := cmd.CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("PowerShell windows list failed: %v, output: %s", err, string(out))
			}
			return string(out), nil

		case "focus", "move", "close":
			// Find window handle by ID or TitlePattern
			var targetID string
			if input.WindowID != "" {
				if !windowsPIDPattern.MatchString(input.WindowID) {
					return "", errors.New("Windows window identifier must be a numeric process ID")
				}
				targetID = input.WindowID
			} else if input.TitlePattern != "" {
				if foundID, findErr := findWindowsPIDByTitle(input.TitlePattern); findErr == nil {
					targetID = foundID
				}
			}

			if targetID == "" {
				return "", fmt.Errorf("Fenster konnte mit ID '%s' oder Titelmuster '%s' nicht gefunden werden", input.WindowID, input.TitlePattern)
			}

			if input.Action == "focus" {
				psScript := psHeader + fmt.Sprintf(`
					$p = Get-Process -Id %s -ErrorAction SilentlyContinue
					if ($p -and $p.MainWindowHandle -ne [IntPtr]::Zero) {
						[Window]::ShowWindow($p.MainWindowHandle, 9) # Restore if minimized (9 = SW_RESTORE)
						[Window]::SetForegroundWindow($p.MainWindowHandle) | Out-Null
						Start-Sleep -Milliseconds 250
						
						Add-Type -TypeDefinition @"
						using System;
						using System.Runtime.InteropServices;
						public class Win {
							[DllImport("user32.dll")]
							public static extern IntPtr GetForegroundWindow();
							[DllImport("user32.dll")]
							public static extern int GetWindowThreadProcessId(IntPtr hWnd, out int lpdwProcessId);
						}
"@ -ErrorAction SilentlyContinue
						$fg = [Win]::GetForegroundWindow()
						$activePid = 0
						[Win]::GetWindowThreadProcessId($fg, [ref]$activePid)
						
						if ($activePid -eq %s) {
							Write-Output "VERIFIED"
						} else {
							Write-Output "SUCCESS"
						}
					} else {
						Write-Output "NOT_FOUND"
					}
				`, targetID, targetID)
				cmd := exec.Command(security.GetPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
				out, err := cmd.CombinedOutput()
				outStr := strings.TrimSpace(string(out))
				if err != nil || outStr == "NOT_FOUND" {
					return "", fmt.Errorf("Fokus auf Fenster %s fehlgeschlagen: %v, output: %s", targetID, err, outStr)
				}
				if outStr == "VERIFIED" {
					return fmt.Sprintf("Fenster %s erfolgreich in den Vordergrund gebracht. [SEMANTIC VERIFICATION PASSED: Window focused]", targetID), nil
				}
				return fmt.Sprintf("Fenster %s erfolgreich in den Vordergrund gebracht.", targetID), nil
			}

			if input.Action == "move" {
				psScript := psHeader + fmt.Sprintf(`
					$p = Get-Process -Id %s -ErrorAction SilentlyContinue
					if ($p -and $p.MainWindowHandle -ne [IntPtr]::Zero) {
						[Window]::ShowWindow($p.MainWindowHandle, 9)
						[Window]::MoveWindow($p.MainWindowHandle, %d, %d, %d, %d, $true) | Out-Null
						[Window]::SetForegroundWindow($p.MainWindowHandle) | Out-Null
						Start-Sleep -Milliseconds 150
						
						$rect = New-Object Window+RECT
						if ([Window]::GetWindowRect($p.MainWindowHandle, [ref]$rect)) {
							$width = $rect.Right - $rect.Left
							$height = $rect.Bottom - $rect.Top
							if ($rect.Left -eq %d -and $rect.Top -eq %d -and $width -eq %d -and $height -eq %d) {
								Write-Output "VERIFIED"
							} else {
								Write-Output "SUCCESS"
							}
						} else {
							Write-Output "SUCCESS"
						}
					} else {
						Write-Output "NOT_FOUND"
					}
				`, targetID, input.X, input.Y, input.Width, input.Height, input.X, input.Y, input.Width, input.Height)
				cmd := exec.Command(security.GetPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
				out, err := cmd.CombinedOutput()
				outStr := strings.TrimSpace(string(out))
				if err != nil || outStr == "NOT_FOUND" {
					return "", fmt.Errorf("Fenster %s konnte nicht bewegt/skaliert werden: %v, output: %s", targetID, err, outStr)
				}
				if outStr == "VERIFIED" {
					return fmt.Sprintf("Fenster %s erfolgreich auf X:%d Y:%d (%dx%d) verschoben. [SEMANTIC VERIFICATION PASSED: Window repositioned]", targetID, input.X, input.Y, input.Width, input.Height), nil
				}
				return fmt.Sprintf("Fenster %s erfolgreich auf X:%d Y:%d (%dx%d) verschoben.", targetID, input.X, input.Y, input.Width, input.Height), nil
			}

			if input.Action == "close" {
				psScript := fmt.Sprintf(`
					$p = Get-Process -Id %s -ErrorAction SilentlyContinue
					if ($p) {
						$p.CloseMainWindow() | Out-Null
						Write-Output "SUCCESS"
					} else {
						Write-Output "NOT_FOUND"
					}
				`, targetID)
				cmd := exec.Command(security.GetPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
				out, err := cmd.CombinedOutput()
				if err != nil || strings.TrimSpace(string(out)) != "SUCCESS" {
					return "", fmt.Errorf("Schließen von Fenster %s fehlgeschlagen: %v", targetID, err)
				}
				return fmt.Sprintf("Fenster %s wurde zum Schließen aufgefordert.", targetID), nil
			}
		}
	} else if runtime.GOOS == "linux" {
		switch input.Action {
		case "list":
			cmd := exec.Command(TrustedExecutable("wmctrl"), "-l", "-G")
			out, err := cmd.CombinedOutput()
			if err != nil {
				cmd2 := exec.Command(TrustedExecutable("xdotool"), "search", "--onlyvisible", "--name", ".*")
				out2, err2 := cmd2.CombinedOutput()
				if err2 != nil {
					return "", fmt.Errorf("Linux window listing failed: %v", err2)
				}
				return string(out2), nil
			}
			return string(out), nil
		case "focus":
			var focusCmd *exec.Cmd
			if input.WindowID != "" {
				focusCmd = exec.Command(TrustedExecutable("xdotool"), "windowactivate", input.WindowID)
			} else if input.TitlePattern != "" {
				focusCmd = exec.Command(TrustedExecutable("wmctrl"), "-a", input.TitlePattern)
			} else {
				return "", errors.New("focus requires window_id or title_pattern")
			}
			out, err := focusCmd.CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("focus failed: %v, output: %s", err, string(out))
			}
			return "Fenster erfolgreich fokussiert.", nil
		case "move":
			if input.WindowID == "" {
				return "", errors.New("move action on Linux requires window_id")
			}
			cmd1 := exec.Command(TrustedExecutable("xdotool"), "windowmove", input.WindowID, fmt.Sprintf("%d", input.X), fmt.Sprintf("%d", input.Y))
			_ = cmd1.Run()
			cmd2 := exec.Command(TrustedExecutable("xdotool"), "windowsize", input.WindowID, fmt.Sprintf("%d", input.Width), fmt.Sprintf("%d", input.Height))
			_ = cmd2.Run()
			return "Fenster verschoben und skaliert.", nil
		case "close":
			var killCmd *exec.Cmd
			if input.WindowID != "" {
				killCmd = exec.Command(TrustedExecutable("xdotool"), "windowkill", input.WindowID)
			} else if input.TitlePattern != "" {
				killCmd = exec.Command(TrustedExecutable("wmctrl"), "-c", input.TitlePattern)
			} else {
				return "", errors.New("close requires window_id or title_pattern")
			}
			_ = killCmd.Run()
			return "Fenster wurde zum Schließen aufgefordert.", nil
		}
	} else if runtime.GOOS == "darwin" {
		switch input.Action {
		case "list":
			script := `tell application "System Events" to get the title of every window of every process whose visible is true`
			cmd := exec.Command(TrustedExecutable("osascript"), "-e", script)
			out, err := cmd.CombinedOutput()
			if err != nil {
				return "", err
			}
			return string(out), nil
		case "focus":
			if input.TitlePattern == "" {
				return "", errors.New("focus on macOS requires title_pattern")
			}
			script := fmt.Sprintf(`tell application "%s" to activate`, escapeAppleScriptString(input.TitlePattern))
			cmd := exec.Command(TrustedExecutable("osascript"), "-e", script)
			_ = cmd.Run()
			return "Fenster fokussiert.", nil
		case "move":
			return "", errors.New("move is not fully implemented on macOS")
		case "close":
			if input.TitlePattern == "" {
				return "", errors.New("close on macOS requires title_pattern")
			}
			script := fmt.Sprintf(`tell application "%s" to quit`, escapeAppleScriptString(input.TitlePattern))
			cmd := exec.Command(TrustedExecutable("osascript"), "-e", script)
			_ = cmd.Run()
			return "Fenster geschlossen.", nil
		}
	}

	return "", fmt.Errorf("Aktion %s wird auf OS %s nicht unterstützt", input.Action, runtime.GOOS)
}
