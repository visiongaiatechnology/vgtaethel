package intelligence

import (
	"testing"
)

func TestIsPointInPolygon(t *testing.T) {
	// Simple square polygon
	poly := []Point{
		{Lon: 5, Lat: 45},
		{Lon: 15, Lat: 45},
		{Lon: 15, Lat: 55},
		{Lon: 5, Lat: 55},
		{Lon: 5, Lat: 45},
	}

	// Test inside point (Berlin / Germany area approximation)
	pIn := Point{Lon: 10, Lat: 50}
	if !IsPointInPolygon(pIn, poly) {
		t.Error("Expected point [10, 50] to be inside polygon")
	}

	// Test outside point (far outside)
	pOut := Point{Lon: 20, Lat: 50}
	if IsPointInPolygon(pOut, poly) {
		t.Error("Expected point [20, 50] to be outside polygon")
	}
}

func TestSearchRegionsByNameAndRadius(t *testing.T) {
	eng := GetDefaultRegionEngine()
	hits := eng.SearchRegions("german", nil, 0)
	if len(hits) == 0 {
		t.Fatal("expected GERMANY match for query german")
	}
	found := false
	for _, r := range hits {
		if r.ID == "GERMANY" {
			found = true
		}
	}
	if !found {
		t.Fatal("GERMANY not in search results")
	}
	// Radius around Berlin centroid should include BERLIN and often GERMANY
	center := &Point{Lat: 52.52, Lon: 13.405}
	near := eng.SearchRegions("", center, 800)
	if len(near) == 0 {
		t.Fatal("expected regions near Berlin within 800km")
	}
	c := RegionCentroid(hits[0])
	if c.Lat == 0 && c.Lon == 0 {
		t.Error("RegionCentroid should not be zero for polygon region")
	}
}
