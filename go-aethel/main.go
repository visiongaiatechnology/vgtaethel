// STATUS: DIAMANT VGT SUPREME
package main

import (
	"embed"
	"io/fs"
	"log"
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

	opts := parseServerRuntimeOptions()

	if !isBindingsBuild() {
		runtimeDir, err := configureRuntimeWorkingDirectory()
		if err != nil {
			log.Fatalf("AETHEL runtime workspace unavailable: %v", err)
		}
		log.Printf("[RUNTIME] Persistent workspace root: %s", runtimeDir)

		exitAfterProvision, err := handlePasswordProvisionCLI(opts)
		if err != nil {
			log.Fatalf("AETHEL password configuration failed: %v", err)
		}
		if exitAfterProvision {
			return
		}

		if err := initServerAuth(opts.ServerMode); err != nil {
			log.Fatalf("AETHEL server authentication subsystem failed: %v", err)
		}
	}

	app := NewApp()

	sub, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		log.Fatalf("Failed to load embedded frontend: %v", err)
	}

	if opts.ServerMode && !isBindingsBuild() {
		log.Println("🛡️ VGT AETHEL :: INITIALISIERUNG (SERVER MODUS)...")
		if err := runHTTPServer(app, sub, opts); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
		return
	}

	log.Println("🛡️ VGT AETHEL :: INITIALISIERUNG (WAILS DESKTOP)...")
	if err := runDesktopWindow(app, sub); err != nil {
		log.Fatalf("Desktop runtime failed: %v", err)
	}
}
