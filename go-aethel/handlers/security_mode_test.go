// STATUS: DIAMANT VGT SUPREME
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-aethel/security"
)

func TestHandleSecurityMode(t *testing.T) {
	guard := security.NewSecurityGuard()
	tmpDir := t.TempDir()
	leases := security.NewLeaseManager(tmpDir + "/leases.json")
	audit := security.NewAuditLogger(tmpDir + "/audit.json")
	policy := security.NewPolicyEngine(guard, leases, audit)

	InitState(guard, leases, audit, policy, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		func() string { return "" }, func() string { return "" }, func() string { return "" }, func() string { return "" }, func() string { return "" },
		func(a, b, c, d, e string) error { return nil }, func() []string { return nil }, func() []security.MountGrant { return nil },
	)

	// 1. Initial GET should return interactive
	req := httptest.NewRequest(http.MethodGet, "/v1/security/mode", nil)
	rec := httptest.NewRecorder()
	HandleSecurityMode(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var getResp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &getResp); err != nil {
		t.Fatal(err)
	}
	if getResp["mode"] != "interactive" || getResp["full_access"] != false {
		t.Fatalf("expected interactive mode, got %+v", getResp)
	}

	// 2. POST full_access
	body, _ := json.Marshal(map[string]string{"mode": "full_access"})
	req = httptest.NewRequest(http.MethodPost, "/v1/security/mode", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	HandleSecurityMode(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var postResp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &postResp); err != nil {
		t.Fatal(err)
	}
	if postResp["mode"] != "full_access" || postResp["full_access"] != true {
		t.Fatalf("expected full_access mode, got %+v", postResp)
	}
	if policy.GetMode() != security.PermissionModeFullAccess {
		t.Fatalf("expected policy mode full_access, got %v", policy.GetMode())
	}

	// 3. POST interactive
	body, _ = json.Marshal(map[string]string{"mode": "interactive"})
	req = httptest.NewRequest(http.MethodPost, "/v1/security/mode", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	HandleSecurityMode(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if policy.GetMode() != security.PermissionModeInteractive {
		t.Fatalf("expected policy mode interactive, got %v", policy.GetMode())
	}

	// 4. POST invalid mode
	body, _ = json.Marshal(map[string]string{"mode": "invalid_mode"})
	req = httptest.NewRequest(http.MethodPost, "/v1/security/mode", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	HandleSecurityMode(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid mode, got %d", rec.Code)
	}
}
