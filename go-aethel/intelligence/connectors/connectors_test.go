package connectors

// STATUS: DIAMANT VGT SUPREME

import (
	"errors"
	"testing"
	"time"

	"go-aethel/intelligence"
)

type policyTestConnector struct{ descriptor Descriptor }

func (connector policyTestConnector) Descriptor() Descriptor { return connector.descriptor }
func (policyTestConnector) HealthCheck() error               { return nil }
func (policyTestConnector) Fetch() ([]intelligence.Observation, error) {
	return nil, errors.New("fetch is not used by policy tests")
}

func TestRegisterRejectsIncompleteMachinePolicy(t *testing.T) {
	descriptor := Descriptor{
		Name: "invalid-policy", Version: "1.0.0", SourceTypes: []string{"api"},
		PollingInterval: time.Minute, RateLimitPerMin: 1, TrustTier: TrustBuiltIn, Activated: true,
	}
	if err := Register(policyTestConnector{descriptor: descriptor}); err == nil {
		t.Fatal("connector without a machine-readable policy was registered")
	}
}

func TestAuthorizeFetchEnforcesUseGeographyAndRate(t *testing.T) {
	descriptor := Descriptor{
		Name: "policy-enforcement-test", Version: "1.0.0", SourceTypes: []string{"api"},
		PollingInterval: time.Minute, RateLimitPerMin: 1, TrustTier: TrustBuiltIn, Activated: true,
		Policy: PublicOSINTPolicy("test-license", "https://example.test/terms", 1, time.Minute, 24*time.Hour),
	}
	descriptor.Policy.AllowedUses = []string{"research"}
	descriptor.Policy.Geography.AllowedRegions = []string{"eu"}
	if err := Register(policyTestConnector{descriptor: descriptor}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := AuthorizeFetch(descriptor.Name, "case-support", "eu", now); err == nil {
		t.Fatal("disallowed use was authorized")
	}
	if err := AuthorizeFetch(descriptor.Name, "research", "us", now); err == nil {
		t.Fatal("disallowed geography was authorized")
	}
	if err := AuthorizeFetch(descriptor.Name, "research", "eu", now); err != nil {
		t.Fatal(err)
	}
	if err := AuthorizeFetch(descriptor.Name, "research", "eu", now.Add(time.Second)); err == nil {
		t.Fatal("policy rate limit was not enforced")
	}
	if err := AuthorizeFetch(descriptor.Name, "research", "eu", now.Add(time.Minute)); err != nil {
		t.Fatalf("rate window did not reset: %v", err)
	}
}
