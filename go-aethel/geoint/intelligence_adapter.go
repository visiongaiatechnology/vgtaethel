package geoint

// STATUS: DIAMANT VGT SUPREME

import (
	"sort"
	"strings"
	"sync"
	"time"

	"go-aethel/intelligence"
)

const intelligenceGeoWindow = 7 * 24 * time.Hour
const maximumIntelligenceGeoEvents = 1000

type IntelligenceSnapshotSource interface {
	GetSnapshot() intelligence.StoreState
}

// IntelligenceGeoAdapter projects the canonical local Intelligence Store into
// bounded GEOINT markers. It never performs collection and never creates links.
type IntelligenceGeoAdapter struct {
	mu        sync.Mutex
	bus       *GeoEntityBus
	source    IntelligenceSnapshotSource
	entityIDs map[string]struct{}
	revision  uint64
}

func NewIntelligenceGeoAdapter(bus *GeoEntityBus, source IntelligenceSnapshotSource) *IntelligenceGeoAdapter {
	return &IntelligenceGeoAdapter{bus: bus, source: source, entityIDs: make(map[string]struct{})}
}

func (a *IntelligenceGeoAdapter) Sync(now time.Time) {
	if a == nil || a.bus == nil || a.source == nil {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	snapshot := a.source.GetSnapshot()
	a.mu.Lock()
	defer a.mu.Unlock()
	if snapshot.Revision == a.revision {
		return
	}
	events := append([]intelligence.Event(nil), snapshot.Events...)
	sort.SliceStable(events, func(i, j int) bool { return events[i].ObservedAt.After(events[j].ObservedAt) })
	if len(events) > maximumIntelligenceGeoEvents {
		events = events[:maximumIntelligenceGeoEvents]
	}
	currentIDs := make(map[string]struct{}, len(events))
	entities := make([]GeoEntity, 0, len(events))
	for _, event := range events {
		if strings.TrimSpace(event.ID) == "" || event.ObservedAt.IsZero() || event.ObservedAt.Before(now.Add(-intelligenceGeoWindow)) || !validGeoCoordinate(event.Latitude, event.Longitude) {
			continue
		}
		id := "intel:event:" + strings.TrimSpace(event.ID)
		currentIDs[id] = struct{}{}
		entities = append(entities, GeoEntity{
			ID: id, Type: intelligenceDomainType(event.Domain), Source: "INTELLIGENCE_STORE",
			SourceIDs: cleanGeoStrings([]string{event.SourceID}), Label: strings.TrimSpace(event.Title),
			Position: GeoPosition{Latitude: event.Latitude, Longitude: event.Longitude}, Timestamp: event.ObservedAt.UTC(),
			Freshness: geoFreshness(event.ObservedAt, now, intelligenceGeoWindow), Confidence: event.Confidence,
			Classification: strings.ToUpper(strings.TrimSpace(event.Severity)),
			Metadata: map[string]any{"summary": strings.TrimSpace(event.Summary), "domain": strings.TrimSpace(event.Domain)},
			Assessment: AssessmentCorrelated, Location: PrecisionApproximate, ExpiresAt: event.ObservedAt.Add(intelligenceGeoWindow),
		})
	}
	a.bus.UpsertBatch(entities)
	for id := range a.entityIDs {
		if _, exists := currentIDs[id]; !exists {
			a.bus.Delete(id)
		}
	}
	a.entityIDs = currentIDs
	a.revision = snapshot.Revision
}

func intelligenceDomainType(domain string) GeoEntityType {
	switch strings.ToLower(strings.TrimSpace(domain)) {
	case "cyber":
		return TypeCyber
	case "economic", "economy", "finance":
		return TypeEconomic
	case "political", "politics":
		return TypePolitical
	case "energy":
		return TypeEnergy
	case "space":
		return TypeSpaceEvent
	case "conflict", "military", "geopolitics":
		return TypeConflict
	case "geo":
		return TypeNewsEvent
	default:
		return TypeNewsEvent
	}
}
