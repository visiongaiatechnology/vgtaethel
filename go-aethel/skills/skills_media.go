package skills

import (
	"go-aethel/security"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	)

// --- 5b. SKILL: MEDIA CONTROL ---

type MediaControlSkill struct{}

type MediaControlArgs struct {
	Action string `json:"action"`
}

func (s *MediaControlSkill) Name() string { return "media_control" }
func (s *MediaControlSkill) Description() string {
	return "Steuert globale Medienwiedergabe und Lautstärke: play_pause, next, previous, volume_up, volume_down, mute."
}
func (s *MediaControlSkill) RiskLevel() security.RiskLevel { return security.RiskLow }

func (s *MediaControlSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"play_pause", "next", "previous", "volume_up", "volume_down", "mute"},
				"description": "Globale Media-Aktion.",
			},
		},
		"required": []string{"action"},
	}
}

func (s *MediaControlSkill) Execute(args json.RawMessage) (string, error) {
	var input MediaControlArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	keyByAction := map[string]string{
		"play_pause":  "0xB3",
		"next":        "0xB0",
		"previous":    "0xB1",
		"volume_up":   "0xAF",
		"volume_down": "0xAE",
		"mute":        "0xAD",
	}
	vk, ok := keyByAction[input.Action]
	if !ok {
		return "", fmt.Errorf("unbekannte Media-Aktion: %s", input.Action)
	}

	if runtime.GOOS == "windows" {
		psScript := fmt.Sprintf(`
			$code = @"
			using System;
			using System.Runtime.InteropServices;
			public static class MediaKey {
				[DllImport("user32.dll", SetLastError=true)]
				public static extern void keybd_event(byte bVk, byte bScan, uint dwFlags, UIntPtr dwExtraInfo);
			}
			"@
			Add-Type -TypeDefinition $code -ErrorAction SilentlyContinue;
			[MediaKey]::keybd_event([byte]%s, 0, 0, [UIntPtr]::Zero);
			[MediaKey]::keybd_event([byte]%s, 0, 2, [UIntPtr]::Zero);
		`, vk, vk)
		cmd := exec.Command(security.GetPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if out, err := cmd.CombinedOutput(); err != nil {
			security.LogKernelActivity("MEDIA_CONTROL_FAILED", input.Action, "ERROR")
			return "", fmt.Errorf("media control failed: %v, output: %s", err, strings.TrimSpace(string(out)))
		}
		security.LogKernelActivity("MEDIA_CONTROL", input.Action, "SUCCESS")
		return fmt.Sprintf("Media-Aktion '%s' ausgeführt.", input.Action), nil
	}

	linuxKeyByAction := map[string]string{
		"play_pause":  "XF86AudioPlay",
		"next":        "XF86AudioNext",
		"previous":    "XF86AudioPrev",
		"volume_up":   "XF86AudioRaiseVolume",
		"volume_down": "XF86AudioLowerVolume",
		"mute":        "XF86AudioMute",
	}
	cmd := exec.Command(TrustedExecutable("xdotool"), "key", linuxKeyByAction[input.Action])
	if err := cmd.Run(); err != nil {
		security.LogKernelActivity("MEDIA_CONTROL_FAILED", input.Action, "ERROR")
		return "", err
	}
	security.LogKernelActivity("MEDIA_CONTROL", input.Action, "SUCCESS")
	return fmt.Sprintf("Media-Aktion '%s' ausgeführt.", input.Action), nil
}

// --- 5c. SKILL: YOUTUBE CONTROL ---

type YouTubeControlSkill struct{}

type YouTubeControlArgs struct {
	Action string `json:"action"`
	Query  string `json:"query,omitempty"`
}

func (s *YouTubeControlSkill) Name() string { return "youtube_control" }
func (s *YouTubeControlSkill) Description() string {
	return "Steuert YouTube im sichtbaren Browser: search, open_home, next_video, play_pause, fullscreen, captions."
}
func (s *YouTubeControlSkill) RiskLevel() security.RiskLevel { return security.RiskLow }

func (s *YouTubeControlSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"search", "open_home", "next_video", "play_pause", "fullscreen", "captions"},
				"description": "YouTube-Aktion.",
			},
			"query": map[string]interface{}{"type": "string", "description": "Suchbegriff für action=search."},
		},
		"required": []string{"action"},
	}
}

func openVisibleURL(targetURL string) error {
	if runtime.GOOS == "windows" {
		cmd := exec.Command(TrustedExecutable("rundll32.exe"), "url.dll,FileProtocolHandler", targetURL)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		return cmd.Start()
	}
	if runtime.GOOS == "darwin" {
		return exec.Command(TrustedExecutable("open"), targetURL).Start()
	}
	return exec.Command(TrustedExecutable("xdg-open"), targetURL).Start()
}

func sendWindowsKeys(sendKeys string) error {
	psScript := fmt.Sprintf(`
		Add-Type -AssemblyName System.Windows.Forms;
		[System.Windows.Forms.SendKeys]::SendWait('%s');
	`, strings.ReplaceAll(sendKeys, "'", "''"))
	cmd := exec.Command(security.GetPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("send keys failed: %v, output: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *YouTubeControlSkill) Execute(args json.RawMessage) (string, error) {
	var input YouTubeControlArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	switch input.Action {
	case "open_home":
		if err := openVisibleURL("https://www.youtube.com/"); err != nil {
			return "", err
		}
	case "search":
		query := strings.TrimSpace(input.Query)
		if query == "" {
			return "", errors.New("youtube search benötigt query")
		}
		if len([]rune(query)) > 200 {
			return "", errors.New("youtube query zu lang")
		}
		targetURL := "https://www.youtube.com/results?search_query=" + url.QueryEscape(query)
		if err := openVisibleURL(targetURL); err != nil {
			return "", err
		}
	case "next_video":
		if runtime.GOOS == "windows" {
			if err := sendWindowsKeys("+N"); err != nil {
				return "", err
			}
		} else {
			if err := exec.Command(TrustedExecutable("xdotool"), "key", "shift+n").Run(); err != nil {
				return "", err
			}
		}
	case "play_pause":
		return (&MediaControlSkill{}).Execute([]byte(`{"action":"play_pause"}`))
	case "fullscreen":
		if runtime.GOOS == "windows" {
			if err := sendWindowsKeys("f"); err != nil {
				return "", err
			}
		} else {
			if err := exec.Command(TrustedExecutable("xdotool"), "key", "f").Run(); err != nil {
				return "", err
			}
		}
	case "captions":
		if runtime.GOOS == "windows" {
			if err := sendWindowsKeys("c"); err != nil {
				return "", err
			}
		} else {
			if err := exec.Command(TrustedExecutable("xdotool"), "key", "c").Run(); err != nil {
				return "", err
			}
		}
	default:
		return "", fmt.Errorf("unbekannte YouTube-Aktion: %s", input.Action)
	}

	security.LogKernelActivity("YOUTUBE_CONTROL", input.Action, "SUCCESS")
	return fmt.Sprintf("YouTube-Aktion '%s' ausgeführt.", input.Action), nil
}
