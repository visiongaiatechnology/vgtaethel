package geoint

import (
	"math"
	"sort"
	"sync"
	"time"
)

const (
	EarthRadiusKm = 6371.0
	DegToRad      = math.Pi / 180.0
	RadToDeg      = 180.0 / math.Pi
)

// GeoEntityBus manages in-memory geospatial entities, spatial indexing, and pub/sub
type GeoEntityBus struct {
	mu           sync.RWMutex
	entities     map[string]GeoEntity
	byType       map[GeoEntityType]map[string]struct{}
	subscribers  map[chan GeoEntity]struct{}
	correlations map[string]CorrelatedEvent
	collectors   map[string]CollectorStatus
	maxTrailLen  int
	revision     uint64
}

// NewGeoEntityBus creates a new GeoEntityBus instance
func NewGeoEntityBus() *GeoEntityBus {
	return &GeoEntityBus{
		entities:     make(map[string]GeoEntity),
		byType:       make(map[GeoEntityType]map[string]struct{}),
		subscribers:  make(map[chan GeoEntity]struct{}),
		correlations: make(map[string]CorrelatedEvent),
		collectors:   make(map[string]CollectorStatus),
		maxTrailLen:  30,
	}
}

// Upsert adds or updates a GeoEntity in the bus
func (b *GeoEntityBus) Upsert(entity GeoEntity) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.upsertLocked(entity)
}

// UpsertBatch adds or updates a slice of GeoEntities in a single lock acquisition
func (b *GeoEntityBus) UpsertBatch(entities []GeoEntity) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, entity := range entities {
		b.upsertLocked(entity)
	}
}

func (b *GeoEntityBus) upsertLocked(entity GeoEntity) {
	if entity.ID == "" {
		return
	}
	if entity.Timestamp.IsZero() {
		entity.Timestamp = time.Now().UTC()
	}
	if entity.Confidence <= 0 {
		entity.Confidence = 80
	}
	if entity.Freshness <= 0 {
		entity.Freshness = 1.0
	}

	// Preserve and append trail if previous position existed
	if prev, exists := b.entities[entity.ID]; exists {
		trail := prev.Trail
		// Append old position to trail if moved significantly (> 100m)
		distKm := HaversineDistance(prev.Position.Latitude, prev.Position.Longitude, entity.Position.Latitude, entity.Position.Longitude)
		if distKm > 0.1 {
			trail = append(trail, prev.Position)
			if len(trail) > b.maxTrailLen {
				trail = trail[len(trail)-b.maxTrailLen:]
			}
		}
		if len(entity.Trail) == 0 {
			entity.Trail = trail
		}
		// Clean old type index if type changed
		if prev.Type != entity.Type {
			if typeSet, ok := b.byType[prev.Type]; ok {
				delete(typeSet, entity.ID)
			}
		}
	}

	b.entities[entity.ID] = entity
	b.revision++
	if _, ok := b.byType[entity.Type]; !ok {
		b.byType[entity.Type] = make(map[string]struct{})
	}
	b.byType[entity.Type][entity.ID] = struct{}{}

	// Notify subscribers non-blocking
	for ch := range b.subscribers {
		select {
		case ch <- entity:
		default:
		}
	}
}

// Get returns an entity by ID
func (b *GeoEntityBus) Get(id string) (GeoEntity, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	entity, exists := b.entities[id]
	return entity, exists
}

// Delete removes an entity by ID
func (b *GeoEntityBus) Delete(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if entity, exists := b.entities[id]; exists {
		if typeSet, ok := b.byType[entity.Type]; ok {
			delete(typeSet, id)
		}
		delete(b.entities, id)
		b.revision++
	}
}

// Revision returns a monotonically increasing snapshot version for HTTP cache validation.
func (b *GeoEntityBus) Revision() uint64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.revision
}

// PruneStale evicts expired snapshots so failed collectors cannot leave ghost tracks indefinitely.
func (b *GeoEntityBus) PruneStale(now time.Time) int {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	removed := 0
	for id, entity := range b.entities {
		expired := !entity.ExpiresAt.IsZero() && !entity.ExpiresAt.After(now)
		if entity.ExpiresAt.IsZero() {
			retention := entityRetention(entity.Type)
			expired = retention > 0 && !entity.Timestamp.IsZero() && entity.Timestamp.Add(retention).Before(now)
		}
		if !expired {
			continue
		}
		if typeSet, exists := b.byType[entity.Type]; exists {
			delete(typeSet, id)
		}
		delete(b.entities, id)
		removed++
	}
	if removed > 0 {
		b.revision++
	}
	return removed
}

func entityRetention(entityType GeoEntityType) time.Duration {
	switch entityType {
	case TypeAircraft, TypeMilAircraft, TypeSatellite:
		return 2 * time.Minute
	case TypeVessel:
		return 15 * time.Minute
	case TypeCamera:
		return 30 * time.Minute
	case TypeEarthquake, TypeVolcano, TypeFire:
		return 7 * 24 * time.Hour
	default:
		return 0
	}
}

// ListAll returns all currently registered entities
func (b *GeoEntityBus) ListAll(limit int) []GeoEntity {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]GeoEntity, 0, len(b.entities))
	for _, e := range b.entities {
		result = append(result, e)
	}
	sortGeoEntities(result)
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result
}

// ListByType returns entities matching a given type
func (b *GeoEntityBus) ListByType(entityType GeoEntityType, limit int) []GeoEntity {
	b.mu.RLock()
	defer b.mu.RUnlock()

	typeSet, ok := b.byType[entityType]
	if !ok {
		return []GeoEntity{}
	}

	result := make([]GeoEntity, 0, len(typeSet))
	for id := range typeSet {
		if entity, exists := b.entities[id]; exists {
			result = append(result, entity)
		}
	}
	sortGeoEntities(result)
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result
}

// QueryRadius finds entities within radiusKm of (lat, lon)
func (b *GeoEntityBus) QueryRadius(lat, lon float64, radiusKm float64, types []GeoEntityType, limit int) []GeoEntity {
	b.mu.RLock()
	defer b.mu.RUnlock()

	typeFilter := make(map[GeoEntityType]bool)
	for _, t := range types {
		typeFilter[t] = true
	}

	type match struct {
		entity   GeoEntity
		distance float64
	}
	matches := make([]match, 0)

	for _, entity := range b.entities {
		if len(typeFilter) > 0 && !typeFilter[entity.Type] {
			continue
		}
		dist := HaversineDistance(lat, lon, entity.Position.Latitude, entity.Position.Longitude)
		if dist <= radiusKm {
			matches = append(matches, match{entity: entity, distance: dist})
		}
	}

	// Sort nearest first
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].distance < matches[j].distance
	})

	maxCount := len(matches)
	if limit > 0 && limit < maxCount {
		maxCount = limit
	}

	result := make([]GeoEntity, maxCount)
	for i := 0; i < maxCount; i++ {
		result[i] = matches[i].entity
	}
	return result
}

// QueryBBox finds entities within a geographic bounding box
func (b *GeoEntityBus) QueryBBox(minLat, minLon, maxLat, maxLon float64, types []GeoEntityType, limit int) []GeoEntity {
	b.mu.RLock()
	defer b.mu.RUnlock()

	typeFilter := make(map[GeoEntityType]bool)
	for _, t := range types {
		typeFilter[t] = true
	}

	result := make([]GeoEntity, 0)
	for _, entity := range b.entities {
		if len(typeFilter) > 0 && !typeFilter[entity.Type] {
			continue
		}
		elat := entity.Position.Latitude
		elon := entity.Position.Longitude
		if elat >= minLat && elat <= maxLat && elon >= minLon && elon <= maxLon {
			result = append(result, entity)
		}
	}
	sortGeoEntities(result)
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result
}

func sortGeoEntities(entities []GeoEntity) {
	sort.SliceStable(entities, func(i, j int) bool {
		leftPriority := geoEntityPriority(entities[i])
		rightPriority := geoEntityPriority(entities[j])
		if leftPriority != rightPriority {
			return leftPriority > rightPriority
		}
		if !entities[i].Timestamp.Equal(entities[j].Timestamp) {
			return entities[i].Timestamp.After(entities[j].Timestamp)
		}
		return entities[i].ID < entities[j].ID
	})
}

func geoEntityPriority(entity GeoEntity) int {
	weight := 100
	switch entity.Type {
	case TypeConflict, TypeCyber:
		weight = 1000
	case TypeMilAircraft, TypeMilitaryInstallation:
		weight = 900
	case TypeEarthquake, TypeVolcano, TypeFire:
		weight = 800
	case TypeAircraft:
		weight = 600
	case TypeVessel:
		weight = 550
	case TypeSatellite, TypeSpaceEvent:
		weight = 500
	}
	return weight + entity.Confidence
}

// GetContactRoster constructs a categorized 250km+ contact list relative to (lat, lon)
func (b *GeoEntityBus) GetContactRoster(lat, lon float64, radiusKm float64, perCategoryLimit int) ContactRoster {
	if radiusKm <= 0 {
		radiusKm = 250.0
	}
	if perCategoryLimit <= 0 {
		perCategoryLimit = 20
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	roster := ContactRoster{
		CenterLat:      lat,
		CenterLon:      lon,
		RadiusKm:       radiusKm,
		AirContacts:    make([]ContactItem, 0),
		MilAirContacts: make([]ContactItem, 0),
		SeaContacts:    make([]ContactItem, 0),
		SpaceContacts:  make([]ContactItem, 0),
		CameraContacts: make([]ContactItem, 0),
		HazardContacts: make([]ContactItem, 0),
		OsintContacts:  make([]ContactItem, 0),
		Timestamp:      time.Now().UTC(),
	}

	for _, entity := range b.entities {
		dist := HaversineDistance(lat, lon, entity.Position.Latitude, entity.Position.Longitude)
		if dist > radiusKm && entity.Type != TypeSatellite {
			// For satellites, we calculate subpoint distance up to 2500km visibility
			if entity.Type == TypeSatellite && dist > 2500.0 {
				continue
			} else if entity.Type != TypeSatellite {
				continue
			}
		}

		bearing := InitialBearing(lat, lon, entity.Position.Latitude, entity.Position.Longitude)
		item := ContactItem{
			EntityID:       entity.ID,
			Type:           entity.Type,
			Label:          entity.Label,
			Classification: entity.Classification,
			DistanceKm:     math.Round(dist*10) / 10,
			BearingDeg:     math.Round(bearing*10) / 10,
			Position:       entity.Position,
			Metadata:       entity.Metadata,
		}

		switch entity.Type {
		case TypeAircraft:
			roster.AirContacts = append(roster.AirContacts, item)
		case TypeMilAircraft:
			roster.MilAirContacts = append(roster.MilAirContacts, item)
		case TypeVessel:
			roster.SeaContacts = append(roster.SeaContacts, item)
		case TypeSatellite:
			roster.SpaceContacts = append(roster.SpaceContacts, item)
		case TypeCamera:
			roster.CameraContacts = append(roster.CameraContacts, item)
		case TypeFire, TypeEarthquake, TypeVolcano:
			roster.HazardContacts = append(roster.HazardContacts, item)
		case TypeConflict, TypeNewsEvent, TypeMilitaryInstallation, TypeInfrastructure:
			roster.OsintContacts = append(roster.OsintContacts, item)
		default:
			roster.OsintContacts = append(roster.OsintContacts, item)
		}
	}

	sortContacts := func(items []ContactItem) []ContactItem {
		sort.Slice(items, func(i, j int) bool {
			return items[i].DistanceKm < items[j].DistanceKm
		})
		if len(items) > perCategoryLimit {
			return items[:perCategoryLimit]
		}
		return items
	}

	roster.AirContacts = sortContacts(roster.AirContacts)
	roster.MilAirContacts = sortContacts(roster.MilAirContacts)
	roster.SeaContacts = sortContacts(roster.SeaContacts)
	roster.SpaceContacts = sortContacts(roster.SpaceContacts)
	roster.CameraContacts = sortContacts(roster.CameraContacts)
	roster.HazardContacts = sortContacts(roster.HazardContacts)
	roster.OsintContacts = sortContacts(roster.OsintContacts)

	roster.TotalContacts = len(roster.AirContacts) + len(roster.MilAirContacts) +
		len(roster.SeaContacts) + len(roster.SpaceContacts) + len(roster.CameraContacts) +
		len(roster.HazardContacts) + len(roster.OsintContacts)

	return roster
}

// RecordCorrelation adds a correlated multi-sensor finding
func (b *GeoEntityBus) RecordCorrelation(event CorrelatedEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if event.ID == "" {
		event.ID = "corr-" + time.Now().UTC().Format("20060102-150405.000")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	b.correlations[event.ID] = event
}

// ListCorrelations returns all active correlated events
func (b *GeoEntityBus) ListCorrelations(limit int) []CorrelatedEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]CorrelatedEvent, 0, len(b.correlations))
	for _, c := range b.correlations {
		result = append(result, c)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	if limit > 0 && len(result) > limit {
		return result[:limit]
	}
	return result
}

// UpdateCollectorStatus updates health metrics for a collector
func (b *GeoEntityBus) UpdateCollectorStatus(name string, status CollectorStatus) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.collectors[name] = status
}

// GetStatus returns the overall health and summary stats of the GEOINT engine
func (b *GeoEntityBus) GetStatus() GeoIntSystemStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()

	typeCounts := make(map[GeoEntityType]int)
	for t, ids := range b.byType {
		typeCounts[t] = len(ids)
	}

	collectorsCopy := make(map[string]CollectorStatus, len(b.collectors))
	for k, v := range b.collectors {
		collectorsCopy[k] = v
	}

	return GeoIntSystemStatus{
		TotalEntities:  len(b.entities),
		TypeCounts:     typeCounts,
		Collectors:     collectorsCopy,
		Correlations:   len(b.correlations),
		BusSubscribers: len(b.subscribers),
		LastUpdated:    time.Now().UTC(),
		SnapshotRevision: b.revision,
	}
}

// Subscribe creates a channel receiving all entity updates
func (b *GeoEntityBus) Subscribe() chan GeoEntity {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan GeoEntity, 128)
	b.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a subscription channel
func (b *GeoEntityBus) Unsubscribe(ch chan GeoEntity) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.subscribers, ch)
	close(ch)
}

// HaversineDistance calculates great circle distance in kilometers
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := (lat2 - lat1) * DegToRad
	dLon := (lon2 - lon1) * DegToRad
	rLat1 := lat1 * DegToRad
	rLat2 := lat2 * DegToRad

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rLat1)*math.Cos(rLat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return EarthRadiusKm * c
}

// InitialBearing calculates the forward azimuth bearing from point 1 to point 2 in degrees (0-360)
func InitialBearing(lat1, lon1, lat2, lon2 float64) float64 {
	rLat1 := lat1 * DegToRad
	rLat2 := lat2 * DegToRad
	dLon := (lon2 - lon1) * DegToRad

	y := math.Sin(dLon) * math.Cos(rLat2)
	x := math.Cos(rLat1)*math.Sin(rLat2) - math.Sin(rLat1)*math.Cos(rLat2)*math.Cos(dLon)
	brng := math.Atan2(y, x) * RadToDeg
	return math.Mod(brng+360.0, 360.0)
}
