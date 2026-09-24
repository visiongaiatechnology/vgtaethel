package security

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestServerAuthManagerLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "security", "server_auth.seal")

	mgr, err := NewServerAuthManager(storePath, true)
	if err != nil {
		t.Fatalf("NewServerAuthManager failed: %v", err)
	}

	if !mgr.IsAuthRequired() {
		t.Fatalf("expected IsAuthRequired() to be true")
	}
	if mgr.IsConfigured() {
		t.Fatalf("expected IsConfigured() to be false before password setup")
	}

	// 1. Reject weak passwords
	if err := mgr.SetPassword("short"); err == nil {
		t.Fatalf("expected weak password to be rejected")
	}

	// 2. Configure valid strong password
	validPassword := "Suprem3-Aethel-Server-Key!"
	if err := mgr.SetPassword(validPassword); err != nil {
		t.Fatalf("SetPassword failed: %v", err)
	}
	if !mgr.IsConfigured() {
		t.Fatalf("expected IsConfigured() to be true after SetPassword")
	}

	// 3. Reload from sealed file on disk
	mgrReloaded, err := NewServerAuthManager(storePath, true)
	if err != nil {
		t.Fatalf("NewServerAuthManager reload failed: %v", err)
	}
	if !mgrReloaded.IsConfigured() {
		t.Fatalf("expected reloaded manager to be configured")
	}

	// 4. Verify wrong password fails
	if _, _, err := mgrReloaded.Authenticate("WrongPassword123!", "192.0.2.10:54321", "TestAgent"); err == nil {
		t.Fatalf("expected wrong password to fail")
	}

	// 5. Verify correct password issues session + CSRF token
	sessionToken, csrfToken, err := mgrReloaded.Authenticate(validPassword, "192.0.2.10:54321", "TestAgent")
	if err != nil {
		t.Fatalf("Authenticate with valid password failed: %v", err)
	}
	if sessionToken == "" || csrfToken == "" {
		t.Fatalf("expected non-empty sessionToken and csrfToken")
	}

	// 6. Validate GET request with session cookie
	getReq := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	getReq.AddCookie(&http.Cookie{Name: ServerSessionCookieName, Value: sessionToken})
	sess, ok := mgrReloaded.ValidateRequest(getReq)
	if !ok || sess == nil {
		t.Fatalf("expected GET request with valid session cookie to succeed")
	}

	// 7. Validate POST request without CSRF header fails
	postReqNoCSRF := httptest.NewRequest(http.MethodPost, "/v1/chat", nil)
	postReqNoCSRF.AddCookie(&http.Cookie{Name: ServerSessionCookieName, Value: sessionToken})
	if _, ok := mgrReloaded.ValidateRequest(postReqNoCSRF); ok {
		t.Fatalf("expected POST request without CSRF token to be rejected")
	}

	// 8. Validate POST request with valid CSRF header succeeds
	postReqWithCSRF := httptest.NewRequest(http.MethodPost, "/v1/chat", nil)
	postReqWithCSRF.AddCookie(&http.Cookie{Name: ServerSessionCookieName, Value: sessionToken})
	postReqWithCSRF.Header.Set(ServerCSRFHeaderName, csrfToken)
	if _, ok := mgrReloaded.ValidateRequest(postReqWithCSRF); !ok {
		t.Fatalf("expected POST request with valid CSRF token to succeed")
	}

	// 9. Logout revokes session
	logoutRec := httptest.NewRecorder()
	mgrReloaded.Logout(logoutRec, postReqWithCSRF)
	if _, ok := mgrReloaded.ValidateRequest(getReq); ok {
		t.Fatalf("expected session to be invalid after Logout")
	}
}

func TestServerAuthRateLimiting(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "security", "server_auth.seal")

	mgr, err := NewServerAuthManager(storePath, true)
	if err != nil {
		t.Fatalf("NewServerAuthManager failed: %v", err)
	}
	validPassword := "Suprem3-Aethel-Server-Key!"
	if err := mgr.SetPassword(validPassword); err != nil {
		t.Fatalf("SetPassword failed: %v", err)
	}

	clientAddr := "198.51.100.77:12345"
	for i := 0; i < MaxFailedLoginsPerWindow; i++ {
		_, _, err := mgr.Authenticate("InvalidAttemptPassword!", clientAddr, "BruteBot")
		if err != ErrInvalidCredentials {
			t.Fatalf("attempt %d: expected ErrInvalidCredentials, got %v", i+1, err)
		}
	}

	// Next attempt should be rate-limited even with the valid password!
	_, _, err = mgr.Authenticate(validPassword, clientAddr, "BruteBot")
	if err != ErrAuthRateLimited {
		t.Fatalf("expected ErrAuthRateLimited after %d failed attempts, got %v", MaxFailedLoginsPerWindow, err)
	}

	// A different IP should still be able to authenticate
	_, _, err = mgr.Authenticate(validPassword, "198.51.100.88:12345", "LegitOperator")
	if err != nil {
		t.Fatalf("expected different IP to authenticate cleanly, got %v", err)
	}

	// Simulate lockout expiration
	mgr.mu.Lock()
	if bucket, ok := mgr.failedByIP["198.51.100.77"]; ok {
		bucket.lockedUntil = time.Now().Add(-1 * time.Second)
		bucket.failures = nil
	}
	mgr.mu.Unlock()

	if _, _, err := mgr.Authenticate(validPassword, clientAddr, "LegitAfterExpiry"); err != nil {
		t.Fatalf("expected login to succeed after lockout expiry, got %v", err)
	}
}
