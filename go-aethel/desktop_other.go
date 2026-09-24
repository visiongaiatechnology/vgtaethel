//go:build !windows

// STATUS: DIAMANT VGT SUPREME
package main

import (
	"context"
	"errors"
	"io/fs"
)

func runDesktopWindow(app *App, sub fs.FS) error {
	return runHTTPServer(app, sub, parseServerRuntimeOptions())
}

func platformHideWindow(_ context.Context) {}

func platformShowWindow(_ context.Context) {}

func platformOpenDirectoryDialog(_ context.Context, _ string) (string, error) {
	return "", errors.New("native directory dialog is unavailable in server mode")
}

func platformWindowMinimise(_ context.Context) {}

func platformWindowToggleMaximise(_ context.Context) {}

func platformWindowClose(_ context.Context) {}

func platformWindowIsMaximised(_ context.Context) bool {
	return false
}
