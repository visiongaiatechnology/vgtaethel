//go:build windows

// STATUS: DIAMANT VGT SUPREME
package main

import (
	"context"
	"io/fs"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func runDesktopWindow(app *App, sub fs.FS) error {
	return wails.Run(&options.App{
		Title:             "VGT AETHEL",
		Width:             1440,
		Height:            900,
		MinWidth:          1024,
		MinHeight:         700,
		DisableResize:     false,
		Frameless:         true,
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
			Theme:                windows.Dark,
		},
	})
}

func platformHideWindow(ctx context.Context) {
	if ctx != nil {
		runtime.Hide(ctx)
	}
}

func platformShowWindow(ctx context.Context) {
	if ctx != nil {
		runtime.Show(ctx)
	}
}

func platformOpenDirectoryDialog(ctx context.Context, title string) (string, error) {
	if ctx == nil {
		return "", nil
	}
	return runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{
		Title: title,
	})
}

func platformWindowMinimise(ctx context.Context) {
	if ctx != nil {
		runtime.WindowMinimise(ctx)
	}
}

func platformWindowToggleMaximise(ctx context.Context) {
	if ctx != nil {
		runtime.WindowToggleMaximise(ctx)
	}
}

func platformWindowClose(ctx context.Context) {
	if ctx != nil {
		runtime.Quit(ctx)
	}
}

func platformWindowIsMaximised(ctx context.Context) bool {
	if ctx != nil {
		return runtime.WindowIsMaximised(ctx)
	}
	return false
}
