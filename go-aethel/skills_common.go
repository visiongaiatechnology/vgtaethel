package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

// VgtSkill is the interface every local skill must implement
type VgtSkill interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	RiskLevel() RiskLevel
	Execute(args json.RawMessage) (string, error)
}

// SkillRegistry holds all available system tools
type SkillRegistry struct {
	skills map[string]VgtSkill
}

func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{
		skills: make(map[string]VgtSkill),
	}
}

func (sr *SkillRegistry) Register(skill VgtSkill) {
	sr.skills[skill.Name()] = skill
}

func (sr *SkillRegistry) Get(name string) (VgtSkill, bool) {
	s, ok := sr.skills[name]
	return s, ok
}

// ToToolDefinitions converts skills to the OpenAI schema format
func (sr *SkillRegistry) ToToolDefinitions() []interface{} {
	var defs []interface{}
	for _, s := range sr.skills {
		defs = append(defs, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        s.Name(),
				"description": s.Description(),
				"parameters":  s.Parameters(),
			},
		})
	}
	return defs
}

// Sandbox validation helper
const workspaceDir = "./vgt_workspace"

var forbiddenShellMeta = regexp.MustCompile(`[;&|><` + "`" + `$]|\$\(|\r|\n`)

func isPathInside(baseDir, targetPath string) bool {
	if runtime.GOOS == "windows" {
		baseDir = strings.ToLower(baseDir)
		targetPath = strings.ToLower(targetPath)
	}
	rel, err := filepath.Rel(baseDir, targetPath)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func canonicalDir(pathStr string) (string, error) {
	abs, err := filepath.Abs(pathStr)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return abs, nil
		}
		return "", err
	}
	return resolved, nil
}

// canonicalTarget resolves the closest existing parent before checking a file
// target. Windows may expose the same path through a short 8.3 alias, so raw
// string-prefix comparisons are not a safe authorization boundary.
func canonicalTarget(pathStr string) (string, error) {
	abs, err := filepath.Abs(pathStr)
	if err != nil {
		return "", err
	}
	current := abs
	var tail []string
	for {
		if resolved, err := filepath.EvalSymlinks(current); err == nil {
			parts := append([]string{resolved}, tail...)
			return filepath.Join(parts...), nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return abs, nil
		}
		tail = append([]string{filepath.Base(current)}, tail...)
		current = parent
	}
}

func validatePath(pathStr string) (string, error) {
	return validatePathForAccess(pathStr, MountRead)
}

func validateWritePath(pathStr string) (string, error) {
	return validatePathForAccess(pathStr, MountWrite)
}

func validatePathForAccess(pathStr string, access MountAccess) (string, error) {
	err := os.MkdirAll(workspaceDir, 0700)
	if err != nil {
		return "", err
	}

	absWorkspace, err := canonicalDir(workspaceDir)
	if err != nil {
		return "", err
	}

	var absTarget string
	if filepath.IsAbs(pathStr) || strings.HasPrefix(pathStr, "/") || strings.HasPrefix(pathStr, "\\") {
		absTarget, err = filepath.Abs(pathStr)
	} else {
		absTarget, err = filepath.Abs(filepath.Join(workspaceDir, pathStr))
	}
	if err != nil {
		return "", err
	}

	resolvedTarget := absTarget
	if existing, err := canonicalTarget(absTarget); err == nil {
		resolvedTarget = existing
	} else {
		return "", err
	}

	if isPathInside(absWorkspace, resolvedTarget) {
		return absTarget, nil
	}

	if state != nil && state.MountAllows(resolvedTarget, access) {
		return absTarget, nil
	}

	if runtime.GOOS == "windows" {
		return "", errors.New("SECURITY VIOLATION: path escaped the configured Windows workspace jail")
	}

	return "", errors.New("SECURITY VIOLATION: Path is not inside the workspace or any mounted directory")
}

func resolveCommandPath(command string) string {
	lowerCmd := strings.ToLower(strings.TrimSpace(command))

	if runtime.GOOS == "windows" {
		if lowerCmd == "cmd.exe" || lowerCmd == "cmd" {
			return "C:\\Windows\\System32\\cmd.exe"
		}
		if lowerCmd == "powershell.exe" || lowerCmd == "powershell" {
			return getPowerShellPath()
		}
	} else {
		// on Unix/Mac
		if lowerCmd == "cmd.exe" || lowerCmd == "cmd" {
			if path, err := exec.LookPath("sh"); err == nil {
				return path
			}
		}
		if lowerCmd == "powershell.exe" || lowerCmd == "powershell" || lowerCmd == "pwsh" {
			if path, err := exec.LookPath("pwsh"); err == nil {
				return path
			}
			if path, err := exec.LookPath("powershell"); err == nil {
				return path
			}
		}
	}

	if path, err := exec.LookPath(command); err == nil {
		return path
	}

	return command
}

func CapturePrimaryDisplay() error {
	screenshotPath, err := filepath.Abs("./vgt_workspace/screenshot.jpg")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(screenshotPath), 0700); err != nil {
		return err
	}

	if runtime.GOOS == "linux" {
		cmd := exec.Command("scrot", "-z", "-q", "70", screenshotPath)
		if err := cmd.Run(); err == nil {
			return os.Chmod(screenshotPath, 0600)
		}
		cmdFallback := exec.Command("import", "-window", "root", "-quality", "70", screenshotPath)
		if err := cmdFallback.Run(); err != nil {
			return err
		}
		return os.Chmod(screenshotPath, 0600)
	}

	psScriptTemplate := `
		[Reflection.Assembly]::LoadWithPartialName("System.Drawing") | Out-Null;
		[Reflection.Assembly]::LoadWithPartialName("System.Windows.Forms") | Out-Null;
		
		# Get monitor containing the mouse cursor
		$mousePos = [System.Windows.Forms.Cursor]::Position;
		$targetScreen = [System.Windows.Forms.Screen]::FromPoint($mousePos);
		$bounds = $targetScreen.Bounds;
		
		$bmp = New-Object System.Drawing.Bitmap($bounds.Width, $bounds.Height);
		$graphics = [System.Drawing.Graphics]::FromImage($bmp);
		$graphics.CopyFromScreen($bounds.X, $bounds.Y, 0, 0, $bmp.Size);
		
		# Downscale if wider than 1280px
		$maxWidth = 1280;
		if ($bounds.Width -gt $maxWidth) {
			$targetWidth = $maxWidth;
			$targetHeight = [int]($bounds.Height * ($targetWidth / $bounds.Width));
			$resizedBmp = New-Object System.Drawing.Bitmap($targetWidth, $targetHeight);
			$gResized = [System.Drawing.Graphics]::FromImage($resizedBmp);
			$gResized.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBilinear;
			$gResized.DrawImage($bmp, 0, 0, $targetWidth, $targetHeight);
			
			$saveBmp = $resizedBmp;
			$gResized.Dispose();
		} else {
			$saveBmp = $bmp;
		}
		
		# Compress as JPEG with 70% Quality to minimize byte size and load instantly
		$jpegCodec = [System.Drawing.Imaging.ImageCodecInfo]::GetImageEncoders() | Where-Object { $_.FormatID -eq [System.Drawing.Imaging.ImageFormat]::Jpeg.Guid };
		$encoderParams = New-Object System.Drawing.Imaging.EncoderParameters(1);
		$encoderParams.Param[0] = New-Object System.Drawing.Imaging.EncoderParameter([System.Drawing.Imaging.Encoder]::Quality, 70);
		
		$saveBmp.Save('__SCREENSHOT_PATH__', $jpegCodec, $encoderParams);
		
		if ($saveBmp -ne $bmp) {
			$saveBmp.Dispose();
		}
		$graphics.Dispose();
		$bmp.Dispose();
	`
	psScript := strings.ReplaceAll(psScriptTemplate, "__SCREENSHOT_PATH__", strings.ReplaceAll(screenshotPath, "'", "''"))

	cmd := exec.Command(getPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		return err
	}
	return os.Chmod(screenshotPath, 0600)
}

var (
	screenshotMutex  sync.Mutex
	lastCaptureTime  time.Time
	cachedScreenshot []byte
)

func GetLatestScreenshot() ([]byte, error) {
	screenshotMutex.Lock()
	defer screenshotMutex.Unlock()

	// If cache is fresh (less than 800ms old), serve directly
	if len(cachedScreenshot) > 0 && time.Since(lastCaptureTime) < 800*time.Millisecond {
		return cachedScreenshot, nil
	}

	// Capture new screenshot
	err := CapturePrimaryDisplay()
	if err != nil {
		if len(cachedScreenshot) > 0 {
			// Fallback to cache on transient capturing failures
			return cachedScreenshot, nil
		}
		return nil, err
	}

	screenshotPath, err := filepath.Abs("./vgt_workspace/screenshot.jpg")
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(screenshotPath)
	if err != nil {
		if len(cachedScreenshot) > 0 {
			return cachedScreenshot, nil
		}
		return nil, err
	}

	cachedScreenshot = data
	lastCaptureTime = time.Now()
	return cachedScreenshot, nil
}
