package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"go-aethel/security"
)

func TestApplyPortToBindAddr(t *testing.T) {
	if got := applyPortToBindAddr("0.0.0.0:8080", "9090"); got != "0.0.0.0:9090" {
		t.Fatalf("expected 0.0.0.0:9090, got %s", got)
	}
	if got := applyPortToBindAddr("127.0.0.1:8080", ":7070"); got != "127.0.0.1:7070" {
		t.Fatalf("expected 127.0.0.1:7070, got %s", got)
	}
}

func TestServerModeAuthGateIntegration(t *testing.T) {
	prevRouter := APIRouter
	prevAuth := ServerAuth
	defer func() {
		APIRouter = prevRouter
		ServerAuth = prevAuth
	}()

	mux := http.NewServeMux()
	registerAuthRoutes(mux)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/v1/protected-probe", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"probe":"allowed"}`))
	})
	APIRouter = mux

	sealPath := filepath.Join(t.TempDir(), "server_auth.seal")
	mgr, err := security.NewServerAuthManager(sealPath, true)
	if err != nil {
		t.Fatalf("NewServerAuthManager failed: %v", err)
	}
	ServerAuth = mgr

	// 1. Public /health must succeed without auth
	healthReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthRec := httptest.NewRecorder()
	APIHandler.ServeHTTP(healthRec, healthReq)
	if healthRec.Code != http.StatusOK {
		t.Fatalf("expected /health 200, got %d", healthRec.Code)
	}

	// 2. Public /v1/auth/status must report server mode, unconfigured, unauthenticated
	statusReq := httptest.NewRequest(http.MethodGet, "/v1/auth/status", nil)
	statusRec := httptest.NewRecorder()
	APIHandler.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("expected /v1/auth/status 200, got %d", statusRec.Code)
	}
	var statusPayload map[string]interface{}
	if err := json.Unmarshal(statusRec.Body.Bytes(), &statusPayload); err != nil {
		t.Fatalf("unmarshal status failed: %v", err)
	}
	if statusPayload["mode"] != "server" || statusPayload["auth_required"] != true || statusPayload["configured"] != false || statusPayload["authenticated"] != false {
		t.Fatalf("unexpected initial auth status: %+v", statusPayload)
	}

	// 3. Protected endpoint must return 401 before login
	probeReq := httptest.NewRequest(http.MethodGet, "/v1/protected-probe", nil)
	probeRec := httptest.NewRecorder()
	APIHandler.ServeHTTP(probeRec, probeReq)
	if probeRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated request, got %d", probeRec.Code)
	}

	// 4. Initial password setup via /v1/auth/setup
	setupBody, _ := json.Marshal(map[string]string{"password": "Aethel-Server-Password-2026!"})
	setupReq := httptest.NewRequest(http.MethodPost, "/v1/auth/setup", bytes.NewReader(setupBody))
	setupRec := httptest.NewRecorder()
	APIHandler.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusOK {
		t.Fatalf("expected /v1/auth/setup 200, got %d: %s", setupRec.Code, setupRec.Body.String())
	}
	var setupResp struct {
		CSRFToken string `json:"csrf_token"`
	}
	_ = json.Unmarshal(setupRec.Body.Bytes(), &setupResp)
	if setupResp.CSRFToken == "" {
		t.Fatalf("expected non-empty csrf_token from setup")
	}
	cookies := setupRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected session cookie from setup")
	}
	sessionCookie := cookies[0]

	// 5. Second /v1/auth/setup must be forbidden once configured
	setupReq2 := httptest.NewRequest(http.MethodPost, "/v1/auth/setup", bytes.NewReader(setupBody))
	setupRec2 := httptest.NewRecorder()
	APIHandler.ServeHTTP(setupRec2, setupReq2)
	if setupRec2.Code != http.StatusForbidden {
		t.Fatalf("expected second /v1/auth/setup to return 403, got %d", setupRec2.Code)
	}

	// 6. Protected GET endpoint with session cookie succeeds
	authProbeReq := httptest.NewRequest(http.MethodGet, "/v1/protected-probe", nil)
	authProbeReq.AddCookie(sessionCookie)
	authProbeRec := httptest.NewRecorder()
	APIHandler.ServeHTTP(authProbeRec, authProbeReq)
	if authProbeRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for authenticated GET, got %d", authProbeRec.Code)
	}

	// 7. Protected POST endpoint without CSRF header fails
	postProbeNoCSRF := httptest.NewRequest(http.MethodPost, "/v1/protected-probe", bytes.NewReader([]byte(`{}`)))
	postProbeNoCSRF.AddCookie(sessionCookie)
	postProbeNoCSRFRec := httptest.NewRecorder()
	APIHandler.ServeHTTP(postProbeNoCSRFRec, postProbeNoCSRF)
	if postProbeNoCSRFRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for POST without CSRF header, got %d", postProbeNoCSRFRec.Code)
	}

	// 8. Protected POST endpoint with CSRF header succeeds
	postProbeCSRF := httptest.NewRequest(http.MethodPost, "/v1/protected-probe", bytes.NewReader([]byte(`{}`)))
	postProbeCSRF.AddCookie(sessionCookie)
	postProbeCSRF.Header.Set(security.ServerCSRFHeaderName, setupResp.CSRFToken)
	postProbeCSRFRec := httptest.NewRecorder()
	APIHandler.ServeHTTP(postProbeCSRFRec, postProbeCSRF)
	if postProbeCSRFRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for POST with CSRF header, got %d", postProbeCSRFRec.Code)
	}
}
