// STATUS: DIAMANT VGT SUPREME
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"go-aethel/handlers"
	"go-aethel/security"
)

const (
	defaultLinuxBindAddr   = "0.0.0.0:8080"
	defaultWindowsBindAddr = "127.0.0.1:8080"
	serverAuthSealPath     = "./vgt_workspace/security/server_auth.seal"
)

type serverRuntimeOptions struct {
	ServerMode       bool
	BindAddr         string
	SetPasswordStdin bool
	SetPasswordValue string
	TLSCertPath      string
	TLSKeyPath       string
}

// ServerAuth coordinates authentication when Aethel runs in HTTP Server Mode.
var ServerAuth *security.ServerAuthManager

func parseServerRuntimeOptions() serverRuntimeOptions {
	opts := serverRuntimeOptions{
		ServerMode:  runtime.GOOS != "windows",
		BindAddr:    defaultBindAddrForOS(),
		TLSCertPath: strings.TrimSpace(os.Getenv("AETHEL_TLS_CERT")),
		TLSKeyPath:  strings.TrimSpace(os.Getenv("AETHEL_TLS_KEY")),
	}

	envMode := strings.ToLower(strings.TrimSpace(os.Getenv("AETHEL_SERVER_MODE")))
	if envMode == "1" || envMode == "true" || envMode == "yes" || envMode == "server" {
		opts.ServerMode = true
	} else if (envMode == "0" || envMode == "false" || envMode == "desktop") && runtime.GOOS == "windows" {
		opts.ServerMode = false
	}

	if envAddr := strings.TrimSpace(os.Getenv("AETHEL_BIND_ADDR")); envAddr != "" {
		opts.BindAddr = envAddr
	}
	if envPort := strings.TrimSpace(os.Getenv("AETHEL_PORT")); envPort != "" {
		opts.BindAddr = applyPortToBindAddr(opts.BindAddr, envPort)
	}
	if initPw := strings.TrimSpace(os.Getenv("AETHEL_INIT_PASSWORD")); initPw != "" {
		opts.SetPasswordValue = initPw
	}

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch {
		case arg == "--server" || arg == "-server":
			opts.ServerMode = true
		case arg == "--desktop" && runtime.GOOS == "windows":
			opts.ServerMode = false
		case arg == "--set-password-stdin":
			opts.SetPasswordStdin = true
		case arg == "--set-password" && i+1 < len(args):
			i++
			opts.SetPasswordValue = args[i]
		case strings.HasPrefix(arg, "--set-password="):
			opts.SetPasswordValue = strings.TrimPrefix(arg, "--set-password=")
		case (arg == "--addr" || arg == "--bind") && i+1 < len(args):
			i++
			opts.BindAddr = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--addr="):
			opts.BindAddr = strings.TrimSpace(strings.TrimPrefix(arg, "--addr="))
		case strings.HasPrefix(arg, "--bind="):
			opts.BindAddr = strings.TrimSpace(strings.TrimPrefix(arg, "--bind="))
		case (arg == "--port" || arg == "-p") && i+1 < len(args):
			i++
			opts.BindAddr = applyPortToBindAddr(opts.BindAddr, strings.TrimSpace(args[i]))
		case strings.HasPrefix(arg, "--port="):
			opts.BindAddr = applyPortToBindAddr(opts.BindAddr, strings.TrimSpace(strings.TrimPrefix(arg, "--port=")))
		case arg == "--tls-cert" && i+1 < len(args):
			i++
			opts.TLSCertPath = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--tls-cert="):
			opts.TLSCertPath = strings.TrimSpace(strings.TrimPrefix(arg, "--tls-cert="))
		case arg == "--tls-key" && i+1 < len(args):
			i++
			opts.TLSKeyPath = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--tls-key="):
			opts.TLSKeyPath = strings.TrimSpace(strings.TrimPrefix(arg, "--tls-key="))
		}
	}

	return opts
}

func defaultBindAddrForOS() string {
	if runtime.GOOS == "windows" {
		return defaultWindowsBindAddr
	}
	return defaultLinuxBindAddr
}

func applyPortToBindAddr(addr, port string) string {
	port = strings.TrimPrefix(strings.TrimSpace(port), ":")
	if port == "" {
		return addr
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return "0.0.0.0:" + port
	}
	if host == "" {
		host = "0.0.0.0"
	}
	return net.JoinHostPort(host, port)
}

func handlePasswordProvisionCLI(opts serverRuntimeOptions) (bool, error) {
	if !opts.SetPasswordStdin && opts.SetPasswordValue == "" {
		return false, nil
	}

	var password string
	if opts.SetPasswordStdin {
		raw, err := io.ReadAll(io.LimitReader(os.Stdin, 1024))
		if err != nil {
			return true, fmt.Errorf("failed to read password from stdin: %w", err)
		}
		password = strings.TrimRight(string(raw), "\r\n")
	} else {
		password = opts.SetPasswordValue
	}

	mgr, err := security.NewServerAuthManager(serverAuthSealPath, true)
	if err != nil {
		return true, fmt.Errorf("failed to initialize server auth store: %w", err)
	}
	if err := mgr.SetPassword(password); err != nil {
		return true, err
	}

	if opts.SetPasswordStdin {
		log.Println("✅ [SECURITY] Server-Operator-Passwort erfolgreich versiegelt (Argon2id + AES-256-GCM).")
		return true, nil
	}
	log.Println("✅ [SECURITY] Initiales Server-Operator-Passwort übernommen.")
	return false, nil
}

func initServerAuth(requireAuth bool) error {
	mgr, err := security.NewServerAuthManager(serverAuthSealPath, requireAuth)
	if err != nil {
		return err
	}
	ServerAuth = mgr
	return nil
}

func isPublicAuthPath(p string) bool {
	switch p {
	case "/health",
		"/v1/auth/status",
		"/v1/auth/login",
		"/v1/auth/logout",
		"/v1/auth/setup":
		return true
	default:
		return false
	}
}

func registerAuthRoutes(mux *http.ServeMux) {
	if mux == nil {
		return
	}
	mux.HandleFunc("/v1/auth/status", handleAuthStatus)
	mux.HandleFunc("/v1/auth/login", handleAuthLogin)
	mux.HandleFunc("/v1/auth/logout", handleAuthLogout)
	mux.HandleFunc("/v1/auth/setup", handleAuthSetup)
	mux.HandleFunc("/v1/code/workspace/project", handleCodeWorkspaceProject)
}

func handleCodeWorkspaceProject(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 8192)).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Ungültige Anfrage."})
			return
		}
		dir := strings.TrimSpace(req.Path)
		if dir == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Projektpfad darf nicht leer sein."})
			return
		}
		if state == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Aethel Core ist noch nicht bereit."})
			return
		}
		if err := state.AddMount(dir, security.MountWrite, 24*time.Hour); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Projekt konnte nicht freigegeben werden: " + err.Error()})
			return
		}
		if err := handlers.SetCodeWorkspaceRoot(dir); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Projekt konnte nicht aktiviert werden: " + err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
			"name":   filepath.Base(filepath.Clean(dir)),
			"path":   filepath.Clean(dir),
		})
	case http.MethodDelete:
		handlers.ClearCodeWorkspaceRoot()
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")

	mode := "desktop"
	authRequired := false
	configured := true
	authenticated := true
	csrfToken := ""

	if ServerAuth != nil && ServerAuth.IsAuthRequired() {
		mode = "server"
		authRequired, configured, authenticated, csrfToken = ServerAuth.SessionStatus(r)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"mode":          mode,
		"os":            runtime.GOOS,
		"auth_required": authRequired,
		"configured":    configured,
		"authenticated": authenticated,
		"csrf_token":    csrfToken,
	})
}

func handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")

	if ServerAuth == nil || !ServerAuth.IsAuthRequired() {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":        "ok",
			"authenticated": true,
			"csrf_token":    "",
		})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 8192)).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Ungültige Anfrage."})
		return
	}

	sessionToken, csrfToken, err := ServerAuth.Authenticate(req.Password, r.RemoteAddr, r.UserAgent())
	if err != nil {
		if state != nil && state.audit != nil {
			_, _ = state.audit.Log("operator", "SERVER_AUTH_LOGIN", "server_gate", security.RiskHigh, "", "blocked", err.Error(), r.RemoteAddr)
		}
		switch {
		case errors.Is(err, security.ErrAuthRateLimited):
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Zu viele fehlgeschlagene Anmeldeversuche. Bitte 5 Minuten warten.",
			})
		case errors.Is(err, security.ErrAuthNotConfigured):
			w.WriteHeader(http.StatusPreconditionRequired)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":      "Kein Server-Passwort konfiguriert. Bitte Initial-Passwort festlegen.",
				"configured": false,
			})
		default:
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Ungültiges Operator-Passwort.",
			})
		}
		return
	}

	ServerAuth.WriteSessionCookie(w, r, sessionToken)
	if state != nil && state.audit != nil {
		_, _ = state.audit.Log("operator", "SERVER_AUTH_LOGIN", "server_gate", security.RiskLow, "", "allowed", "Operator authenticated", r.RemoteAddr)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "ok",
		"authenticated": true,
		"csrf_token":    csrfToken,
	})
}

func handleAuthSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")

	if ServerAuth == nil || !ServerAuth.IsAuthRequired() {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Server-Modus ist nicht aktiv."})
		return
	}
	if ServerAuth.IsConfigured() {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Server-Passwort ist bereits konfiguriert. Bitte regulär anmelden.",
		})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 8192)).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Ungültige Anfrage."})
		return
	}

	if err := ServerAuth.SetPassword(req.Password); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Passwort muss mindestens 10 Zeichen lang sein.",
		})
		return
	}

	sessionToken, csrfToken, err := ServerAuth.Authenticate(req.Password, r.RemoteAddr, r.UserAgent())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Initiale Sitzung konnte nicht erstellt werden."})
		return
	}

	ServerAuth.WriteSessionCookie(w, r, sessionToken)
	if state != nil && state.audit != nil {
		_, _ = state.audit.Log("operator", "SERVER_AUTH_SETUP", "server_gate", security.RiskModerate, "", "allowed", "Initial server operator password sealed", r.RemoteAddr)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "ok",
		"configured":    true,
		"authenticated": true,
		"csrf_token":    csrfToken,
	})
}

func handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")

	if ServerAuth != nil {
		ServerAuth.Logout(w, r)
	}
	if state != nil && state.audit != nil {
		_, _ = state.audit.Log("operator", "SERVER_AUTH_LOGOUT", "server_gate", security.RiskSafe, "", "allowed", "Operator logged out", r.RemoteAddr)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "ok",
		"authenticated": false,
	})
}

func runHTTPServer(app *App, sub fs.FS, opts serverRuntimeOptions) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if ServerAuth == nil {
		if err := initServerAuth(true); err != nil {
			return fmt.Errorf("failed to initialize server auth: %w", err)
		}
	}

	app.startup(ctx)
	app.domReady(ctx)
	defer app.shutdown(ctx)

	if !ServerAuth.IsConfigured() {
		log.Println("⚠️ [SECURITY] Kein Server-Passwort konfiguriert! Beim ersten Aufruf im Browser oder per '--set-password-stdin' festlegen.")
	} else {
		log.Println("🔒 [SECURITY] Login-Schutz aktiv (Argon2id + AES-256-GCM Session Gate).")
	}

	staticFileServer := http.FileServer(http.FS(sub))

	rootHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAPIPath(r.URL.Path) || r.URL.Path == "/assets/earth_day.jpg" {
			APIHandler.ServeHTTP(w, r)
			return
		}
		sw := secureResponseWriter{ResponseWriter: w}
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			sw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		staticFileServer.ServeHTTP(sw, r)
	})

	srv := &http.Server{
		Addr:              opts.BindAddr,
		Handler:           rootHandler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	errCh := make(chan error, 1)
	go func() {
		if opts.TLSCertPath != "" && opts.TLSKeyPath != "" {
			srv.TLSConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
			log.Printf("🌐 [SERVER] VGT AETHEL läuft im HTTPS Server-Modus auf https://%s", opts.BindAddr)
			if err := srv.ListenAndServeTLS(opts.TLSCertPath, opts.TLSKeyPath); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
			return
		}
		log.Printf("🌐 [SERVER] VGT AETHEL läuft im HTTP Server-Modus auf http://%s", opts.BindAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("🔴 [SERVER] Signal %v empfangen — fahre AETHEL Server kontrolliert herunter...", sig)
	case err := <-errCh:
		return err
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	return srv.Shutdown(shutdownCtx)
}
