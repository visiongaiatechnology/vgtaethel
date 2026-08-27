package main

import (
	"embed"
	"io/fs"
	"log"

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
	if !isBindingsBuild() {
		runtimeDir, err := configureRuntimeWorkingDirectory()
		if err != nil {
			log.Fatalf("AETHEL runtime workspace unavailable: %v", err)
		}
		log.Printf("[RUNTIME] Persistent workspace root: %s", runtimeDir)
	}
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
