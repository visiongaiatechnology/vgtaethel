package connectors

import (
	"errors"
	"time"

	"go-aethel/intelligence"
)

// Controlled Connector model per spec §13.
// Trust classes: BuiltInTrusted (1) / LocallyApproved (2) / CommunityUnverified (3) / Blocked (0).
// Each declares Name, Version, RateLimits, HealthCheck, TrustTier, Sandbox rules.
// No arbitrary code execution. Validation + explicit activation required.
// External content is untrusted (prompt injection guard at ingest).

type TrustClass int

const (
	TrustBlocked TrustClass = 0
	TrustBuiltIn TrustClass = 1
	TrustLocal   TrustClass = 2
	TrustCommunity TrustClass = 3
)

// Descriptor is the static registration metadata for a connector (no code exec).
type Descriptor struct {
	Name             string
	Version          string
	SourceTypes      []string
	Permissions      []string
	RequiredSecrets  []string
	PollingInterval  time.Duration
	RateLimitPerMin  int
	Regions          []string
	LicenseInfo      string
	TrustTier       TrustClass
	Activated        bool
}

// Connector produces raw Observations only (never free-form verified facts).
type Connector interface {
	Descriptor() Descriptor
	HealthCheck() error
	Fetch() ([]intelligence.Observation, error)
}

var Registry = map[string]Connector{}

// Register only after sandbox + policy check. Blocked trust is refused.
func Register(c Connector) error {
	if c == nil {
		return errors.New("nil connector")
	}
	d := c.Descriptor()
	if d.Name == "" {
		return errors.New("connector name required")
	}
	if d.TrustTier == TrustBlocked {
		return errors.New("blocked connectors cannot register")
	}
	if !d.Activated && d.TrustTier == TrustCommunity {
		return errors.New("community connectors require explicit activation")
	}
	Registry[d.Name] = c
	return nil
}

// BuiltinRSSDescriptor documents the in-tree RSS path (OSINT collector → Observation → Store.Ingest).
// Actual fetch remains in main package; this registry entry is metadata for Source Policy / UI.
func BuiltinRSSDescriptor() Descriptor {
	return Descriptor{
		Name:            "builtin-rss",
		Version:         "1.0.0",
		SourceTypes:     []string{"rss", "atom"},
		Permissions:     []string{"network.fetch.public"},
		PollingInterval: 15 * time.Minute,
		RateLimitPerMin: 30,
		LicenseInfo:     "in-tree AETHEL; no WorldWideView/WorldMonitor code",
		TrustTier:      TrustBuiltIn,
		Activated:       true,
	}
}
