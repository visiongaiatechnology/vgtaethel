package intelligence

import (
	"strings"
)

// IsPointInPolygon checks if a point is inside a polygon using ray casting.
func IsPointInPolygon(p Point, polygon []Point) bool {
	if len(polygon) < 3 {
		return false
	}

	inside := false
	j := len(polygon) - 1
	for i := 0; i < len(polygon); i++ {
		// Lon is X, Lat is Y
		if ((polygon[i].Lat > p.Lat) != (polygon[j].Lat > p.Lat)) &&
			(p.Lon < (polygon[j].Lon-polygon[i].Lon)*(p.Lat-polygon[i].Lat)/(polygon[j].Lat-polygon[i].Lat)+polygon[i].Lon) {
			inside = !inside
		}
		j = i
	}
	return inside
}

// RegionCentroid returns the average vertex of the first ring (deterministic, no external deps).
func RegionCentroid(r Region) Point {
	if len(r.Polygons) == 0 || len(r.Polygons[0]) == 0 {
		return Point{}
	}
	ring := r.Polygons[0]
	n := len(ring)
	if n > 1 && ring[0].Lat == ring[n-1].Lat && ring[0].Lon == ring[n-1].Lon {
		n--
	}
	if n < 1 {
		return Point{}
	}
	var sumLat, sumLon float64
	for i := 0; i < n; i++ {
		sumLat += ring[i].Lat
		sumLon += ring[i].Lon
	}
	return Point{Lat: sumLat / float64(n), Lon: sumLon / float64(n)}
}

// RegionEngine manages polygon geofencing lookup.
type RegionEngine struct {
	regions []Region
}

func NewRegionEngine(regions []Region) *RegionEngine {
	return &RegionEngine{regions: regions}
}

// MatchPoint returns all regions containing the given coordinates.
func (re *RegionEngine) MatchPoint(lat, lon float64) []Region {
	p := Point{Lon: lon, Lat: lat}
	matched := []Region{}
	for _, r := range re.regions {
		for _, poly := range r.Polygons {
			if IsPointInPolygon(p, poly) {
				matched = append(matched, r)
				break
			}
		}
	}
	return matched
}

// SearchRegions filters by name/type/ID substring and optionally by centroid distance to a center point.
// Uses package haversineDistance (store.go) for radius filtering — no placeholders.
func (re *RegionEngine) SearchRegions(query string, center *Point, radiusKm float64) []Region {
	var out []Region
	q := strings.ToLower(strings.TrimSpace(query))
	seen := map[string]bool{}

	for _, r := range re.regions {
		nameHit := q == "" || strings.Contains(strings.ToLower(r.Name), q) ||
			strings.Contains(strings.ToLower(r.Type), q) ||
			strings.Contains(strings.ToLower(r.ID), q)
		if !nameHit {
			continue
		}
		if center != nil && radiusKm > 0 {
			c := RegionCentroid(r)
			if haversineDistance(center.Lat, center.Lon, c.Lat, c.Lon) > radiusKm {
				continue
			}
		}
		if !seen[r.ID] {
			out = append(out, r)
			seen[r.ID] = true
		}
	}

	if center != nil && radiusKm > 0 && q == "" {
		for _, r := range re.regions {
			if seen[r.ID] {
				continue
			}
			c := RegionCentroid(r)
			if haversineDistance(center.Lat, center.Lon, c.Lat, c.Lon) <= radiusKm {
				out = append(out, r)
				seen[r.ID] = true
			}
		}
	}
	return out
}
