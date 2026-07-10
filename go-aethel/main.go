package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed frontend/*
var frontendFS embed.FS

const (
	groqURL     = "https://api.groq.com/openai/v1/chat/completions"
	deepseekURL = "https://api.deepseek.com/chat/completions"
	openaiURL   = "https://api.openai.com/v1/chat/completions"
	geminiURL   = "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"
	claudeURL   = "https://api.anthropic.com/v1/messages"
)

func main() {
	defer func() {
		if recovered := recover(); recovered != nil {
			if state != nil && state.release != nil {
				state.release.CapturePanic(recovered)
			}
			panic(recovered)
		}
	}()
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

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := "SETUP_REQUIRED"
	if state.isConfigured() {
		status = "READY"
	}

	oReady := "false"
	if state.getOpenAIKey() != "" {
		oReady = "true"
	}

	dsReady := "false"
	if state.getDeepSeekKey() != "" {
		dsReady = "true"
	}

	ollamaReady := "false"
	if models := getLocalOllamaModels(); len(models) > 0 {
		ollamaReady = "true"
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":         status,
		"mode":           "STREAMING",
		"core":           "GO-CORTEX",
		"openai_ready":   oReady,
		"deepseek_ready": dsReady,
		"ollama_ready":   ollamaReady,
		"os":             runtime.GOOS,
	})
}

func getPowerShellPath() string {
	if path, err := exec.LookPath("powershell.exe"); err == nil {
		return path
	}
	if path, err := exec.LookPath("powershell"); err == nil {
		return path
	}
	return "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"
}
