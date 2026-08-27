package intelligence

import (
	"strings"
	"time"
)

// RegionalRiskCatalogEntry is one scored Global Watch region (shared by AI, baseline, HUD, globe).
// Polygons are approximate boxes/outlines for scoring and overlay — not political boundary claims.
type RegionalRiskCatalogEntry struct {
	ID             string
	Name           string
	MinLat, MaxLat float64
	MinLon, MaxLon float64
	// Centroid for globe pin placement
	Lat, Lon float64
	// Ring is a closed lon/lat ring for semi-transparent overlay.
	Ring []Point
}

// RiskReference is a source citation shown in the Regional Risk detail popup.
type RiskReference struct {
	Title  string `json:"title"`
	URL    string `json:"url,omitempty"`
	Source string `json:"source,omitempty"`
}

// RegionalRiskCatalog is the single source of truth for which regions appear in
// "Regionales Risiko". Every ensure/merge path must return this full set.
func RegionalRiskCatalog() []RegionalRiskCatalogEntry {
	return []RegionalRiskCatalogEntry{
		box("GERMANY", "Germany", 47.2, 55.0, 5.8, 15.0),
		box("FRANCE", "France", 42.3, 51.1, -5.0, 9.5),
		box("USA", "United States", 24.5, 49.0, -125.0, -66.9),
		box("UKRAINE", "Ukraine", 44.3, 52.4, 22.0, 40.2),
		box("UK", "United Kingdom", 49.9, 60.8, -8.6, 1.7),
		// Expanded conflict / geopolitical anchors
		box("RUSSIA", "Russia", 41.2, 77.7, 19.6, 180.0),
		box("IRAN", "Iran", 25.0, 39.8, 44.0, 63.3),
		box("CHINA", "China", 18.2, 53.6, 73.5, 134.8),
		box("ISRAEL", "Israel", 29.4, 33.4, 34.2, 35.9),
		box("TAIWAN", "Taiwan", 21.8, 25.4, 119.3, 122.1),
		box("POLAND", "Poland", 49.0, 54.9, 14.1, 24.2),
		box("BALTICS", "Baltics (EE/LV/LT)", 53.9, 59.7, 20.9, 28.3),
	}
}

// RegionalRiskCatalogIDs returns uppercase region IDs in catalog order.
func RegionalRiskCatalogIDs() []string {
	cat := RegionalRiskCatalog()
	out := make([]string, len(cat))
	for i, e := range cat {
		out[i] = e.ID
	}
	return out
}

// LookupRegionalRiskCatalog returns the entry for id (case-insensitive) if present.
func LookupRegionalRiskCatalog(id string) (RegionalRiskCatalogEntry, bool) {
	want := strings.ToUpper(strings.TrimSpace(id))
	for _, e := range RegionalRiskCatalog() {
		if e.ID == want {
			return e, true
		}
	}
	return RegionalRiskCatalogEntry{}, false
}

func box(id, name string, minLat, maxLat, minLon, maxLon float64) RegionalRiskCatalogEntry {
	ring := []Point{
		{Lon: minLon, Lat: minLat},
		{Lon: maxLon, Lat: minLat},
		{Lon: maxLon, Lat: maxLat},
		{Lon: minLon, Lat: maxLat},
		{Lon: minLon, Lat: minLat},
	}
	return RegionalRiskCatalogEntry{
		ID: id, Name: name,
		MinLat: minLat, MaxLat: maxLat, MinLon: minLon, MaxLon: maxLon,
		Lat: (minLat + maxLat) / 2, Lon: (minLon + maxLon) / 2,
		Ring: ring,
	}
}

// PointInCatalogBBox reports whether lat/lon fall inside the catalog entry bbox.
func PointInCatalogBBox(e RegionalRiskCatalogEntry, lat, lon float64) bool {
	return lat >= e.MinLat && lat <= e.MaxLat && lon >= e.MinLon && lon <= e.MaxLon
}

// CatalogRegionAsEngineRegion converts a catalog entry to a Region for the region engine.
func CatalogRegionAsEngineRegion(e RegionalRiskCatalogEntry) Region {
	return Region{
		ID: e.ID, Name: e.Name, Type: "state",
		Polygons: [][]Point{append([]Point(nil), e.Ring...)},
	}
}

// FillCatalogFromScores merges AI/hybrid/deterministic scores with the full catalog.
// Missing region IDs are filled from baselineByID (if present) or zero shells so the HUD never shrinks.
// now is used for CacheAgeHours when AIEvaluatedAt is set.
func FillCatalogFromScores(scores map[string]RiskScore, baselineByID map[string]RegionalRiskData, now time.Time) []RegionalRiskData {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cat := RegionalRiskCatalog()
	out := make([]RegionalRiskData, 0, len(cat))
	for _, e := range cat {
		if rs, ok := scores[e.ID]; ok {
			age := 0.0
			if !rs.AIEvaluatedAt.IsZero() {
				age = now.Sub(rs.AIEvaluatedAt).Hours()
			}
			rd := RegionalRiskData{
				RegionID:           e.ID,
				RegionName:         e.Name,
				OverallRisk:        rs.OverallRisk,
				GeopoliticalRisk:   rs.GeopoliticalRisk,
				ConflictRisk:       rs.ConflictRisk,
				CyberRisk:          rs.CyberRisk,
				InfrastructureRisk: rs.InfrastructureRisk,
				EconomicRisk:       rs.EconomicRisk,
				PrimaryDrivers:     append([]string(nil), rs.PrimaryDrivers...),
				Trend:              rs.Trend,
				EvaluationSource:   rs.EvaluationSource,
				AINarrative:        rs.AINarrative,
				AIModelID:          rs.AIModelID,
				AIEvaluatedAt:      rs.AIEvaluatedAt,
				NextRefreshAt:      rs.NextRefreshAt,
				CacheAgeHours:      age,
			}
			if rd.Trend == "" {
				rd.Trend = "stable"
			}
			if rd.EvaluationSource == "" {
				rd.EvaluationSource = "deterministic"
			}
			if b, ok := baselineByID[e.ID]; ok && len(rd.PrimaryDrivers) == 0 {
				rd.PrimaryDrivers = append([]string(nil), b.PrimaryDrivers...)
			}
			out = append(out, rd)
			continue
		}
		if b, ok := baselineByID[e.ID]; ok {
			b.RegionID = e.ID
			if b.RegionName == "" {
				b.RegionName = e.Name
			}
			if b.EvaluationSource == "" {
				b.EvaluationSource = "deterministic"
			}
			if b.Trend == "" {
				b.Trend = "stable"
			}
			out = append(out, b)
			continue
		}
		out = append(out, RegionalRiskData{
			RegionID: e.ID, RegionName: e.Name, Trend: "stable",
			EvaluationSource: "deterministic",
			AINarrative:      "Keine Score-Daten für diese Region — leere Baseline.",
		})
	}
	return out
}

// MergeRiskReferences merges reference lists by title (case-insensitive).
// Later candidates with a non-empty URL upgrade earlier title-only entries
// (so Collect's SharedIntelStore Event titles do not block baseline/feed URLs).
// Empty titles are skipped. Cap is applied after merge (0 or negative → 12).
func MergeRiskReferences(capN int, lists ...[]RiskReference) []RiskReference {
	if capN <= 0 {
		capN = 12
	}
	type slot struct {
		ref   RiskReference
		order int
	}
	byTitle := map[string]*slot{}
	order := 0
	for _, list := range lists {
		for _, r := range list {
			title := strings.TrimSpace(r.Title)
			if title == "" {
				continue
			}
			key := strings.ToLower(title)
			url := strings.TrimSpace(r.URL)
			src := strings.TrimSpace(r.Source)
			if existing, ok := byTitle[key]; ok {
				// Upgrade empty URL; never drop a real URL for a later empty one.
				if strings.TrimSpace(existing.ref.URL) == "" && url != "" {
					existing.ref.URL = url
				}
				if strings.TrimSpace(existing.ref.Source) == "" && src != "" {
					existing.ref.Source = src
				}
				// Prefer longer title only if equal key already set; keep first display title.
				continue
			}
			byTitle[key] = &slot{
				ref: RiskReference{
					Title:  TruncateIntel(title, 160),
					URL:    TruncateIntel(url, 1024),
					Source: TruncateIntel(src, 120),
				},
				order: order,
			}
			order++
		}
	}
	// Stable order by first-seen
	ordered := make([]*slot, 0, len(byTitle))
	for _, s := range byTitle {
		ordered = append(ordered, s)
	}
	// small insertion sort by order (no import sort if avoidable — use sort)
	for i := 1; i < len(ordered); i++ {
		j := i
		for j > 0 && ordered[j].order < ordered[j-1].order {
			ordered[j], ordered[j-1] = ordered[j-1], ordered[j]
			j--
		}
	}
	out := make([]RiskReference, 0, len(ordered))
	for _, s := range ordered {
		out = append(out, s.ref)
		if len(out) >= capN {
			break
		}
	}
	return out
}

// AttachCatalogReferences fills References for each region from shared-store style event titles
// (and optional URLs when present on the event map). Callers pass pre-filtered candidates.
func AttachCatalogReferences(risks []RegionalRiskData, refsByRegion map[string][]RiskReference) []RegionalRiskData {
	if len(risks) == 0 || len(refsByRegion) == 0 {
		return risks
	}
	for i := range risks {
		id := strings.ToUpper(risks[i].RegionID)
		if refs, ok := refsByRegion[id]; ok && len(refs) > 0 {
			// Cap + normalize via merge (single list)
			risks[i].References = MergeRiskReferences(12, refs)
		}
	}
	return risks
}
