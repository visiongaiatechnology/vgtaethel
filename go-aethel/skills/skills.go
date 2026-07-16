package skills

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"go-aethel/security"
)

// VgtSkill is the interface every local skill must implement
type VgtSkill interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	RiskLevel() security.RiskLevel
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


var forbiddenShellMeta = regexp.MustCompile(`[;&|><` + "`" + `$]|\$\(|\r|\n`)

// trustedExecutable returns only fixed operating-system installation paths.
// Never delegate executable resolution to PATH: a user-writable PATH entry can
// otherwise turn an approved tool call into an arbitrary binary execution.
func TrustedExecutable(name string) string {
	switch runtime.GOOS {
	case "windows":
		switch strings.ToLower(name) {
		case "powershell", "powershell.exe":
			return `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`
		case "cmd", "cmd.exe":
			return `C:\Windows\System32\cmd.exe`
		case "rundll32", "rundll32.exe":
			return `C:\Windows\System32\rundll32.exe`
		case "where", "where.exe":
			return `C:\Windows\System32\where.exe`
		case "whoami", "whoami.exe":
			return `C:\Windows\System32\whoami.exe`
		case "tasklist", "tasklist.exe":
			return `C:\Windows\System32\tasklist.exe`
		case "git":
			return `C:\Program Files\Git\cmd\git.exe`
		case "go":
			return `C:\Program Files\Go\bin\go.exe`
		case "node":
			return `C:\Program Files\nodejs\node.exe`
		case "npm":
			return `C:\Program Files\nodejs\npm.cmd`
		case "npx":
			return `C:\Program Files\nodejs\npx.cmd`
		case "python", "python3":
			return `C:\Program Files\Python312\python.exe`
		}
	case "darwin":
		switch name {
		case "osascript":
			return "/usr/bin/osascript"
		case "open":
			return "/usr/bin/open"
		}
	default:
		switch name {
		case "xdotool":
			return "/usr/bin/xdotool"
		case "wmctrl":
			return "/usr/bin/wmctrl"
		case "xdg-open":
			return "/usr/bin/xdg-open"
		case "scrot":
			return "/usr/bin/scrot"
		case "import":
			return "/usr/bin/import"
		case "git":
			return "/usr/bin/git"
		case "go":
			return "/usr/local/go/bin/go"
		case "node":
			return "/usr/bin/node"
		case "npm":
			return "/usr/bin/npm"
		case "npx":
			return "/usr/bin/npx"
		case "python", "python3":
			return "/usr/bin/python3"
		}
	}
	return ""
}





// canonicalTarget resolves the closest existing parent before checking a file
// target. Windows may expose the same path through a short 8.3 alias, so raw
// string-prefix comparisons are not a safe authorization boundary.


func validatePath(pathStr string) (string, error) {
	return security.ValidatePathForAccess(pathStr, security.MountRead)
}





func resolveCommandPath(command string) string {
	return TrustedExecutable(strings.ToLower(strings.TrimSpace(command)))
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
		cmd := exec.Command(TrustedExecutable("scrot"), "-z", "-q", "70", screenshotPath)
		if err := cmd.Run(); err == nil {
			return os.Chmod(screenshotPath, 0600)
		}
		cmdFallback := exec.Command(TrustedExecutable("import"), "-window", "root", "-quality", "70", screenshotPath)
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

	cmd := exec.Command(security.GetPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
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
