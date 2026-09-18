package geoint

// STATUS: DIAMANT VGT SUPREME

import (
	"net"
	"net/url"
	"testing"
)

func TestValidateCameraFrameURLUsesStrictAllowlist(t *testing.T) {
	allowed, _ := url.Parse("https://cwwp2.dot.ca.gov/data/d4/cctv/image/example.jpg")
	if err := validateCameraFrameURL(allowed); err != nil {
		t.Fatalf("expected built-in camera host to be accepted: %v", err)
	}
	rejected := []string{
		"http://127.0.0.1/admin", "http://169.254.169.254/latest/meta-data", "file:///etc/passwd",
		"https://user:pass@cwwp2.dot.ca.gov/frame.jpg", "https://cwwp2.dot.ca.gov:8443/frame.jpg",
		"https://cwwp2.dot.ca.gov.evil.example/frame.jpg",
	}
	for _, raw := range rejected {
		candidate, _ := url.Parse(raw)
		if err := validateCameraFrameURL(candidate); err == nil {
			t.Fatalf("expected URL to be rejected: %s", raw)
		}
	}
}

func TestIsPublicCameraIPRejectsInternalNetworks(t *testing.T) {
	rejected := []string{"127.0.0.1", "10.0.0.1", "172.16.0.1", "192.168.1.1", "169.254.169.254", "::1", "fc00::1"}
	for _, raw := range rejected {
		if isPublicCameraIP(net.ParseIP(raw)) {
			t.Fatalf("expected private address to be rejected: %s", raw)
		}
	}
	if !isPublicCameraIP(net.ParseIP("1.1.1.1")) {
		t.Fatal("expected public address to be accepted")
	}
}

func TestCameraImageTypeAllowlist(t *testing.T) {
	for _, allowed := range []string{"image/jpeg", "image/png", "image/gif", "image/webp"} {
		if !isAllowedCameraImageType(allowed) {
			t.Fatalf("expected %s to be accepted", allowed)
		}
	}
	for _, rejected := range []string{"text/html", "image/svg+xml", "application/octet-stream"} {
		if isAllowedCameraImageType(rejected) {
			t.Fatalf("expected %s to be rejected", rejected)
		}
	}
}

