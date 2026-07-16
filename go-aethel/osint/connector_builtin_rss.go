package osint

// BuiltIn RSS connector — registry + Fetch that produces Observations from the
// OSINT engine cache (and optional live refresh). Ingest is explicit via
// intelligence_connector_fetch so network work stays operator-triggered when desired.
// No arbitrary code execution; no WWV/WM code.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go-aethel/intelligence"
	"go-aethel/intelligence/connectors"
)

type builtinRSSConnector struct {
	mu       sync.Mutex
	lastFetch time.Time
}

func (c *builtinRSSConnector) Descriptor() connectors.Descriptor {
	return connectors.BuiltinRSSDescriptor()
}

func (c *builtinRSSConnector) HealthCheck() error {
	if state == nil || state.osint == nil {
		return errors.New("osint engine not started")
	}
	cfgs := state.osint.GetConfigs()
	enabled := 0
	for _, cfg := range cfgs {
		if cfg.Enabled {
			enabled++
		}
	}
	if enabled == 0 {
		return errors.New("no enabled RSS collectors")
	}
	return nil
}

// Fetch returns Observations derived from the local OSINT event cache.
// It does not invent sources: only cached collector output is mapped.
// Optional live refresh is rate-limited to the descriptor polling interval.
func (c *builtinRSSConnector) Fetch() ([]intelligence.Observation, error) {
	if err := c.HealthCheck(); err != nil {
		return nil, err
	}
	d := c.Descriptor()
	c.mu.Lock()
	since := time.Since(c.lastFetch)
	doRefresh := c.lastFetch.IsZero() || since >= d.PollingInterval
	if doRefresh {
		c.lastFetch = time.Now().UTC()
	}
	c.mu.Unlock()

	if doRefresh {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		_ = state.osint.RefreshNow(ctx)
		cancel()
	}

	events := state.osint.GetEvents("all")
	if len(events) == 0 {
		// Domain filter "all" may be empty if engine expects empty domain for all — try general empty
		events = state.osint.GetEvents("")
	}
	limit := d.RateLimitPerMin
	if limit < 1 {
		limit = 30
	}
	out := make([]intelligence.Observation, 0, minInt(limit, len(events)))
	for i, ev := range events {
		if i >= limit {
			break
		}
		src := strings.TrimSpace(ev.Source)
		if src == "" {
			src = "rss-unknown"
		}
		srcID := "rss-" + strings.ReplaceAll(src, " ", "_")
		raw := strings.TrimSpace(ev.Title + " " + ev.Summary)
		if raw == "" {
			continue
		}
		id := strings.TrimSpace(ev.ID)
		if id == "" {
			id = fmt.Sprintf("conn-%d-%d", time.Now().UnixNano(), i)
		}
		out = append(out, intelligence.Observation{
			ID:         "conn-" + id,
			SourceID:   srcID,
			RawText:    raw,
			ObservedAt: ev.Timestamp,
			Latitude:   ev.Lat,
			Longitude:  ev.Lon,
			Domain:     string(ev.Domain),
		})
	}
	return out, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// connectorRegistrySummary is used by identity/source health surfaces.
func ConnectorRegistrySummary() map[string]any {
	out := make([]map[string]any, 0, len(connectors.Registry))
	for name, c := range connectors.Registry {
		d := c.Descriptor()
		health := "ok"
		if err := c.HealthCheck(); err != nil {
			health = err.Error()
		}
		out = append(out, map[string]any{
			"name": name, "version": d.Version, "trust_tier": int(d.TrustTier),
			"activated": d.Activated, "rate_limit_per_min": d.RateLimitPerMin,
			"polling": d.PollingInterval.String(), "health": health,
		})
	}
	return map[string]any{
		"count":      len(out),
		"connectors": out,
		"as_of":      time.Now().UTC().Format(time.RFC3339),
	}
}

// runConnectorFetchIngest executes Fetch on a named connector and ingests into intelligence.SharedIntelStore.
func RunConnectorFetchIngest(name string) (fetched int, ingested int, err error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "builtin-rss"
	}
	c, ok := connectors.Registry[name]
	if !ok {
		return 0, 0, fmt.Errorf("connector %q not registered", name)
	}
	if intelligence.SharedIntelStore == nil {
		return 0, 0, errors.New("intelligence.SharedIntelStore unavailable")
	}
	obs, err := c.Fetch()
	if err != nil {
		return 0, 0, err
	}
	n := intelligence.SharedIntelStore.IngestObservationsBatch(obs)
	return len(obs), n, nil
}
