//go:build windows

package main

import (
	"bytes"
	"testing"
)

func TestDPAPIRoundTrip(t *testing.T) {
	plain := []byte("aethel-dpapi-test-value")
	protected, err := dpapiProtect(plain)
	if err != nil {
		t.Fatalf("DPAPI protect failed: %v", err)
	}
	if bytes.Contains(protected, plain) {
		t.Fatal("DPAPI payload contains plaintext")
	}
	opened, err := dpapiUnprotect(protected)
	if err != nil {
		t.Fatalf("DPAPI unprotect failed: %v", err)
	}
	if !bytes.Equal(opened, plain) {
		t.Fatalf("DPAPI round trip mismatch: %q", opened)
	}
}
