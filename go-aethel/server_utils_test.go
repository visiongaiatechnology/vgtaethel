package main

// STATUS: DIAMANT VGT SUPREME

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLocalAPIHeadersHaveStrictCSPWithoutUnsafeDirectives(t *testing.T) {
	recorder := httptest.NewRecorder()
	setLocalAPIHeaders(recorder)
	policy := recorder.Header().Get("Content-Security-Policy")
	if policy == "" || strings.Contains(policy, "unsafe-inline") || strings.Contains(policy, "unsafe-eval") {
		t.Fatalf("unsafe CSP shipped: %q", policy)
	}
	for _, directive := range []string{"script-src 'self'", "style-src 'self'", "object-src 'none'", "frame-ancestors 'none'", "base-uri 'none'", "form-action 'self'"} {
		if !strings.Contains(policy, directive) {
			t.Fatalf("CSP directive missing: %s in %q", directive, policy)
		}
	}
}
