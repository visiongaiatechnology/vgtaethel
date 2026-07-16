package main

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed frontend/*
var frontendFS embed.FS

func main() {
	defer func() {
		if recovered := recover(); recovered != nil {
			if state != nil && state.release != nil {
				state.release.CapturePanic(recovered)
			}
			panic(recovered)
		}
	}()
	pinRuntimeWorkingDirectory()
	log.Println("🛡️ VGT AETHEL :: INITIALISIERUNG (WAILS DESKTOP)...")

	app := NewApp()

	sub, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		log.Fatalf("Failed to load embedded frontend: %v", err)
	}

	err = wails.Run(&options.App{
		Title:             "VGT AETHEL",
		Width:             1440,
		Height:            900,
		MinWidth:          1024,
		MinHeight:         700,
		DisableResize:     false,
		StartHidden:       false,
		HideWindowOnClose: false,
		BackgroundColour:  &options.RGBA{R: 8, G: 8, B: 18, A: 255},
		AssetServer: &assetserver.Options{
			Assets:  sub,
			Handler: APIHandler,
		},
		OnStartup:     app.startup,
		OnDomReady:    app.domReady,
		OnBeforeClose: app.beforeClose,
		OnShutdown:    app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})
	if err != nil {
		log.Fatalf("Wails failed: %v", err)
	}
}

// pinRuntimeWorkingDirectory keeps every relative AETHEL workspace path bound
// to the directory that contains the shipped executable. Windows shortcuts,
// terminals and launchers may otherwise inject an unrelated current directory,
// causing the UI to see a different config/key pair than the installed app.
func pinRuntimeWorkingDirectory() {
	executablePath, err := os.Executable()
	if err != nil {
		log.Printf("[RUNTIME] Executable path unavailable: %v", err)
		return
	}
	runtimeDir, ok := selectRuntimeWorkingDirectory(executablePath)
	if !ok {
		return
	}
	if err := os.Chdir(runtimeDir); err != nil {
		log.Printf("[RUNTIME] Cannot bind workspace to executable directory: %v", err)
	}
}

func selectRuntimeWorkingDirectory(executablePath string) (string, bool) {
	runtimeDir := filepath.Clean(filepath.Dir(executablePath))
	workspacePath := filepath.Join(runtimeDir, "vgt_workspace")
	info, err := os.Stat(workspacePath)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return runtimeDir, true
}
