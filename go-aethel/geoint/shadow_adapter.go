package geoint

// STATUS: DIAMANT VGT SUPREME

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"

	"go-aethel/osint"
)

const shadowGeoHistoryWindow = 7 * 24 * time.Hour

// ShadowGeoAdapter projects the existing SHADOW evidence store into GEOINT.
// It never performs collection and never infers a relationship absent from a
// validated SHADOW conflict_link.
type ShadowGeoAdapter struct {
	mu          sync.RWMutex
	bus         *GeoEntityBus
	shadow      ShadowSnapshotSource
	relations   []GeoRelation
	entityIDs   map[string]struct{}
	lastSync    time.Time
	lastVersion string
}

type ShadowSnapshotSource interface {
	Snapshot() osint.ShadowState
}

func NewShadowGeoAdapter(bus *GeoEntityBus, shadow ShadowSnapshotSource) *ShadowGeoAdapter {
	return &ShadowGeoAdapter{bus: bus, shadow: shadow, entityIDs: make(map[string]struct{})}
}

func (a *ShadowGeoAdapter) Sync(now time.Time) {
	if a == nil || a.bus == nil || a.shadow == nil {
		return
	}
	now = now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	snapshot := a.shadow.Snapshot()
	reports := append([]osint.ShadowReport(nil), snapshot.Reports...)
	sort.SliceStable(reports, func(i, j int) bool { return reports[i].CreatedAt.After(reports[j].CreatedAt) })

	items := make(map[string]osint.ShadowIntelItem, len(snapshot.Buffer))
	for _, item := range snapshot.Buffer {
		items[item.ID] = item
	}

	entities := make([]GeoEntity, 0, 64)
	relations := make([]GeoRelation, 0, 64)
	seenRegions := make(map[string]struct{})
	seenRelations := make(map[string]struct{})
	versionMaterial := strings.Builder{}

	for _, report := range reports {
		if report.CreatedAt.IsZero() || report.CreatedAt.Before(now.Add(-shadowGeoHistoryWindow)) || report.CreatedAt.After(now.Add(10*time.Minute)) {
			continue
		}
		versionMaterial.WriteString(report.ID)
		versionMaterial.WriteString(report.ContentSHA256)
		for _, region := range report.Regions {
			key := normalizedGeoKey(region.RegionID, region.RegionName)
			if key == "" {
				continue
			}
			if _, exists := seenRegions[key]; exists {
				continue
			}
			if !validGeoCoordinate(region.Latitude, region.Longitude) {
				continue
			}
			seenRegions[key] = struct{}{}
			evidenceIDs := cleanGeoStrings([]string(region.EvidenceIDs))
			if len(evidenceIDs) == 0 {
				continue
			}
			entities = append(entities, GeoEntity{
				ID:             "shadow:region:" + shortGeoHash(key),
				Type:           TypeConflict,
				Source:         "SHADOW",
				SourceIDs:      sourceIDsForEvidence(evidenceIDs, items),
				Label:          strings.TrimSpace(region.RegionName),
				Position:       GeoPosition{Latitude: region.Latitude, Longitude: region.Longitude},
				Timestamp:      report.CreatedAt.UTC(),
				Freshness:      geoFreshness(report.CreatedAt, now, shadowGeoHistoryWindow),
				Confidence:     int(region.Confidence),
				Classification: strings.TrimSpace(region.ConflictLevel),
				Metadata: map[string]any{
					"report_id": report.ID, "report_kind": report.Kind, "trend": region.Trend,
					"security_score": int(region.SecurityScore), "assessment": string(region.Assessment),
				},
				Evidence:   evidenceIDs,
				Assessment: supportedShadowStatus(int(region.Confidence)),
				Location:   PrecisionRegion,
				ExpiresAt:  report.CreatedAt.Add(shadowGeoHistoryWindow),
			})
		}

		for _, link := range report.ConflictLinks {
			key := normalizedGeoKey(link.AttackerName, link.TargetName, link.Action)
			if key == "" {
				continue
			}
			if _, exists := seenRelations[key]; exists {
				continue
			}
			if !validGeoCoordinate(link.AttackerLatitude, link.AttackerLongitude) || !validGeoCoordinate(link.TargetLatitude, link.TargetLongitude) {
				continue
			}
			if link.AttackerLatitude == link.TargetLatitude && link.AttackerLongitude == link.TargetLongitude {
				continue
			}
			seenRelations[key] = struct{}{}
			evidenceIDs := cleanGeoStrings([]string(link.EvidenceIDs))
			if len(evidenceIDs) == 0 {
				continue
			}
			timestamp := newestEvidenceTimestamp(evidenceIDs, items, report.CreatedAt)
			category := shadowRelationCategory(link.Action)
			attribution := "ASSESSED_DIRECTION"
			if category == RelationCyber {
				attribution = "REPORTED_ATTRIBUTION"
			}
			relations = append(relations, GeoRelation{
				ID: "shadow:relation:" + shortGeoHash(key), ReportID: report.ID,
				OriginName: strings.TrimSpace(link.AttackerName), TargetName: strings.TrimSpace(link.TargetName),
				Origin:          GeoPosition{Latitude: link.AttackerLatitude, Longitude: link.AttackerLongitude},
				Target:          GeoPosition{Latitude: link.TargetLatitude, Longitude: link.TargetLongitude},
				OriginPrecision: PrecisionCountry, TargetPrecision: PrecisionCountry,
				Action: strings.TrimSpace(link.Action), Category: category, Timestamp: timestamp,
				Freshness: geoFreshness(timestamp, now, shadowGeoHistoryWindow), Confidence: int(link.Confidence),
				EvidenceIDs: evidenceIDs, SourceIDs: sourceIDsForEvidence(evidenceIDs, items),
				AssessmentStatus: supportedShadowStatus(int(link.Confidence)), AttributionStatus: attribution,
				Assessment: strings.TrimSpace(string(link.Assessment)),
			})
		}
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	currentIDs := make(map[string]struct{}, len(entities))
	for _, entity := range entities {
		currentIDs[entity.ID] = struct{}{}
		a.bus.Upsert(entity)
	}
	for id := range a.entityIDs {
		if _, exists := currentIDs[id]; !exists {
			a.bus.Delete(id)
		}
	}
	a.entityIDs = currentIDs
	a.relations = relations
	a.lastSync = now
	sum := sha256.Sum256([]byte(versionMaterial.String()))
	a.lastVersion = hex.EncodeToString(sum[:8])
}

func (a *ShadowGeoAdapter) Relations() ([]GeoRelation, time.Time, string) {
	if a == nil {
		return []GeoRelation{}, time.Time{}, ""
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return append([]GeoRelation(nil), a.relations...), a.lastSync, a.lastVersion
}

func validGeoCoordinate(lat, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180 && !(lat == 0 && lon == 0)
}

func normalizedGeoKey(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.Join(strings.Fields(value), " "))
		if value == "" {
			return ""
		}
		parts = append(parts, value)
	}
	return strings.Join(parts, "|")
}

func shortGeoHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:10])
}

func cleanGeoStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sourceIDsForEvidence(evidenceIDs []string, items map[string]osint.ShadowIntelItem) []string {
	values := make([]string, 0, len(evidenceIDs))
	for _, id := range evidenceIDs {
		if item, exists := items[id]; exists {
			values = append(values, item.SourceID)
		}
	}
	return cleanGeoStrings(values)
}

func newestEvidenceTimestamp(evidenceIDs []string, items map[string]osint.ShadowIntelItem, fallback time.Time) time.Time {
	latest := fallback.UTC()
	for _, id := range evidenceIDs {
		item, exists := items[id]
		if !exists {
			continue
		}
		stamp := item.PublishedAt
		if stamp.IsZero() {
			stamp = item.CollectedAt
		}
		if stamp.After(latest) {
			latest = stamp.UTC()
		}
	}
	return latest
}

func geoFreshness(timestamp, now time.Time, lifetime time.Duration) float64 {
	if timestamp.IsZero() || lifetime <= 0 {
		return 0
	}
	age := now.Sub(timestamp)
	if age <= 0 {
		return 1
	}
	if age >= lifetime {
		return 0
	}
	return 1 - float64(age)/float64(lifetime)
}

func supportedShadowStatus(confidence int) GeoAssessmentStatus {
	if confidence >= 70 {
		return AssessmentSupported
	}
	return AssessmentAssessed
}

func shadowRelationCategory(action string) GeoRelationCategory {
	switch strings.ToUpper(strings.TrimSpace(action)) {
	case "CYBER_ATTACK":
		return RelationCyber
	case "MILITARY_SUPPORT":
		return RelationSupport
	case "BLOCKADE":
		return RelationMaritime
	default:
		return RelationKinetic
	}
}
