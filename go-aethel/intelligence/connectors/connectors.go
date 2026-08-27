package connectors

// STATUS: DIAMANT VGT SUPREME

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"go-aethel/intelligence"
)

type TrustClass int

const (
	TrustBlocked   TrustClass = 0
	TrustBuiltIn   TrustClass = 1
	TrustLocal     TrustClass = 2
	TrustCommunity TrustClass = 3
)

type AuthenticationPolicy struct {
	Mode        string   `json:"mode"`
	SecretNames []string `json:"secret_names,omitempty"`
}

type RatePolicy struct {
	Requests int           `json:"requests"`
	Window   time.Duration `json:"window"`
	Burst    int           `json:"burst"`
}

type RetentionPolicy struct {
	MaximumAge      time.Duration `json:"maximum_age"`
	StoreRawPayload bool          `json:"store_raw_payload"`
	LegalHold       bool          `json:"legal_hold"`
}

type GeographyPolicy struct {
	AllowedRegions []string `json:"allowed_regions"`
	BlockedRegions []string `json:"blocked_regions,omitempty"`
}

type Policy struct {
	LicenseID           string               `json:"license_id"`
	TermsURL            string               `json:"terms_url,omitempty"`
	AllowedUses         []string             `json:"allowed_uses"`
	Authentication      AuthenticationPolicy `json:"authentication"`
	Rate                RatePolicy           `json:"rate"`
	Retention           RetentionPolicy      `json:"retention"`
	Geography           GeographyPolicy      `json:"geography"`
	Classification      string               `json:"classification"`
	Redistribution      string               `json:"redistribution"`
	AttributionRequired bool                 `json:"attribution_required"`
}

type Descriptor struct {
	Name            string        `json:"name"`
	Version         string        `json:"version"`
	SourceTypes     []string      `json:"source_types"`
	Permissions     []string      `json:"permissions"`
	RequiredSecrets []string      `json:"required_secrets,omitempty"`
	PollingInterval time.Duration `json:"polling_interval"`
	RateLimitPerMin int           `json:"rate_limit_per_min"`
	Regions         []string      `json:"regions"`
	LicenseInfo     string        `json:"license_info"`
	TrustTier       TrustClass    `json:"trust_tier"`
	Activated       bool          `json:"activated"`
	Policy          Policy        `json:"policy"`
}

type Connector interface {
	Descriptor() Descriptor
	HealthCheck() error
	Fetch() ([]intelligence.Observation, error)
}

var registry = struct {
	sync.RWMutex
	items   map[string]Connector
	windows map[string]rateWindow
}{items: make(map[string]Connector), windows: make(map[string]rateWindow)}

type rateWindow struct {
	StartedAt time.Time
	Requests  int
}

func PublicOSINTPolicy(licenseID, termsURL string, requests int, window, retention time.Duration) Policy {
	return Policy{
		LicenseID:           licenseID,
		TermsURL:            termsURL,
		AllowedUses:         []string{"situational-awareness", "research", "case-support"},
		Authentication:      AuthenticationPolicy{Mode: "none"},
		Rate:                RatePolicy{Requests: requests, Window: window, Burst: 1},
		Retention:           RetentionPolicy{MaximumAge: retention, StoreRawPayload: true},
		Geography:           GeographyPolicy{AllowedRegions: []string{"global"}},
		Classification:      "public",
		Redistribution:      "metadata-and-derived-assessments",
		AttributionRequired: true,
	}
}

func LocalReplayPolicy() Policy {
	return Policy{
		LicenseID:      "AETHEL-internal",
		AllowedUses:    []string{"case-support", "audit", "recovery"},
		Authentication: AuthenticationPolicy{Mode: "local-process"},
		Rate:           RatePolicy{Requests: 20, Window: time.Minute, Burst: 1},
		Retention:      RetentionPolicy{MaximumAge: 3650 * 24 * time.Hour, StoreRawPayload: false, LegalHold: true},
		Geography:      GeographyPolicy{AllowedRegions: []string{"local"}},
		Classification: "operator-controlled",
		Redistribution: "prohibited",
	}
}

func Register(connector Connector) error {
	if connector == nil {
		return errors.New("nil connector")
	}
	descriptor := connector.Descriptor()
	if err := ValidateDescriptor(descriptor); err != nil {
		return err
	}
	registry.Lock()
	defer registry.Unlock()
	registry.items[descriptor.Name] = connector
	return nil
}

func Get(name string) (Connector, bool) {
	registry.RLock()
	defer registry.RUnlock()
	connector, exists := registry.items[name]
	return connector, exists
}

func AuthorizeFetch(name, intendedUse, region string, now time.Time) error {
	registry.Lock()
	defer registry.Unlock()
	connector, exists := registry.items[strings.TrimSpace(name)]
	if !exists {
		return errors.New("connector is not registered")
	}
	descriptor := connector.Descriptor()
	if err := ValidateDescriptor(descriptor); err != nil {
		return fmt.Errorf("connector policy rejected: %w", err)
	}
	if !descriptor.Activated {
		return errors.New("connector is not activated")
	}
	if !containsFold(descriptor.Policy.AllowedUses, intendedUse) {
		return errors.New("connector policy does not allow the intended use")
	}
	if !containsFold(descriptor.Policy.Geography.AllowedRegions, "global") && !containsFold(descriptor.Policy.Geography.AllowedRegions, region) {
		return errors.New("connector policy does not allow the requested geography")
	}
	if containsFold(descriptor.Policy.Geography.BlockedRegions, region) {
		return errors.New("connector policy blocks the requested geography")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	window := registry.windows[descriptor.Name]
	if window.StartedAt.IsZero() || now.Sub(window.StartedAt) >= descriptor.Policy.Rate.Window {
		window = rateWindow{StartedAt: now}
	}
	if window.Requests >= descriptor.Policy.Rate.Requests {
		return errors.New("connector policy rate limit exceeded")
	}
	window.Requests++
	registry.windows[descriptor.Name] = window
	return nil
}

func List() []Connector {
	registry.RLock()
	defer registry.RUnlock()
	names := make([]string, 0, len(registry.items))
	for name := range registry.items {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]Connector, 0, len(names))
	for _, name := range names {
		result = append(result, registry.items[name])
	}
	return result
}

func ValidateDescriptor(descriptor Descriptor) error {
	if strings.TrimSpace(descriptor.Name) == "" || strings.TrimSpace(descriptor.Version) == "" {
		return errors.New("connector name and version are required")
	}
	if descriptor.TrustTier == TrustBlocked {
		return errors.New("blocked connectors cannot register")
	}
	if !descriptor.Activated && descriptor.TrustTier == TrustCommunity {
		return errors.New("community connectors require explicit activation")
	}
	if descriptor.PollingInterval <= 0 || descriptor.RateLimitPerMin <= 0 || len(descriptor.SourceTypes) == 0 {
		return errors.New("connector polling, rate and source types are required")
	}
	return ValidatePolicy(descriptor.Policy)
}

func ValidatePolicy(policy Policy) error {
	if strings.TrimSpace(policy.LicenseID) == "" || len(policy.AllowedUses) == 0 {
		return errors.New("connector license and allowed uses are required")
	}
	if policy.TermsURL != "" {
		parsed, err := url.Parse(policy.TermsURL)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
			return errors.New("connector terms URL must be absolute HTTPS")
		}
	}
	switch policy.Authentication.Mode {
	case "none", "api-key", "oauth2", "local-process":
	default:
		return errors.New("connector authentication mode is invalid")
	}
	if (policy.Authentication.Mode == "api-key" || policy.Authentication.Mode == "oauth2") && len(policy.Authentication.SecretNames) == 0 {
		return errors.New("authenticated connectors must declare secret names")
	}
	if policy.Rate.Requests <= 0 || policy.Rate.Window <= 0 || policy.Rate.Burst <= 0 {
		return errors.New("connector rate policy is invalid")
	}
	if policy.Retention.MaximumAge <= 0 || len(policy.Geography.AllowedRegions) == 0 {
		return errors.New("connector retention and geography policy are required")
	}
	switch policy.Classification {
	case "public", "operator-controlled", "restricted":
	default:
		return errors.New("connector classification is invalid")
	}
	if strings.TrimSpace(policy.Redistribution) == "" {
		return errors.New("connector redistribution policy is required")
	}
	for _, use := range policy.AllowedUses {
		if strings.TrimSpace(use) == "" {
			return fmt.Errorf("connector allowed use contains an empty value")
		}
	}
	return nil
}

func containsFold(values []string, wanted string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(wanted)) {
			return true
		}
	}
	return false
}

func BuiltinRSSDescriptor() Descriptor {
	return Descriptor{
		Name: "builtin-rss", Version: "2.0.0", SourceTypes: []string{"rss", "atom"},
		Permissions: []string{"network.fetch.public"}, PollingInterval: 15 * time.Minute,
		RateLimitPerMin: 30, LicenseInfo: "Feed-specific terms apply; AETHEL stores source attribution",
		TrustTier: TrustBuiltIn, Activated: true,
		Policy: PublicOSINTPolicy("source-specific", "", 30, time.Minute, 180*24*time.Hour),
	}
}
