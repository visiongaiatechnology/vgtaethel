package security

import (
	"context"
	"strings"
	"testing"
)

func TestResolvePublicHTTPSDestinationRejectsLocalAndAmbiguousTargets(t *testing.T) {
	t.Parallel()
	invalid := []string{
		"http://example.com",
		"https://user:secret@example.com",
		"https://localhost/admin",
		"https://127.0.0.1/admin",
		"https://[::1]/admin",
		"https://100.64.0.1/",
		"https://192.0.2.1/",
		"https://bad_host.example/",
		"https://example.com:8443/",
	}
	for _, raw := range invalid {
		if _, err := ResolvePublicHTTPSDestination(context.Background(), raw); err == nil {
			t.Errorf("unsafe target accepted: %s", raw)
		}
	}
}

func TestResolvePublicHTTPSDestinationPinsLiteralPublicAddress(t *testing.T) {
	t.Parallel()
	target, err := ResolvePublicHTTPSDestination(context.Background(), "https://1.1.1.1/path#fragment")
	if err != nil {
		t.Fatal(err)
	}
	if target.URL != "https://1.1.1.1/path" || target.Address.String() != "1.1.1.1" {
		t.Fatalf("unexpected canonical target: %+v", target)
	}
	rules := ChromiumHostResolverRules(target)
	if rules != "MAP 1.1.1.1 1.1.1.1, MAP * ~NOTFOUND" || strings.ContainsAny(rules, "\r\n") {
		t.Fatalf("unexpected resolver rules: %q", rules)
	}
}

func TestResolvePublicHTTPSDestinationPreservesIPv6URLBrackets(t *testing.T) {
	t.Parallel()
	target, err := ResolvePublicHTTPSDestination(context.Background(), "https://[2606:4700:4700::1111]/dns-query")
	if err != nil {
		t.Fatal(err)
	}
	if target.URL != "https://[2606:4700:4700::1111]/dns-query" {
		t.Fatalf("IPv6 URL was corrupted: %q", target.URL)
	}
	if rules := ChromiumHostResolverRules(target); rules != "MAP [2606:4700:4700::1111] [2606:4700:4700::1111], MAP * ~NOTFOUND" {
		t.Fatalf("unexpected IPv6 resolver rules: %q", rules)
	}
}
