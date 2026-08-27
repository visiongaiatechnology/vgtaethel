package intelligence

// GetDefaultRegionEngine returns a region engine loaded with approximate coordinates of major states
// and the shared regional-risk catalog (RUSSIA, IRAN, etc.).
func GetDefaultRegionEngine() *RegionEngine {
	// Start from catalog (includes GERMANY…BALTICS with bbox polygons).
	var regions []Region
	seen := map[string]bool{}
	for _, e := range RegionalRiskCatalog() {
		regions = append(regions, CatalogRegionAsEngineRegion(e))
		seen[e.ID] = true
	}

	// City-level overlay still useful for Germany context.
	if !seen["BERLIN"] {
		regions = append(regions, Region{
			ID:   "BERLIN",
			Name: "Berlin",
			Type: "city",
			Polygons: [][]Point{{
				{Lon: 13.0, Lat: 52.3},
				{Lon: 13.8, Lat: 52.3},
				{Lon: 13.8, Lat: 52.7},
				{Lon: 13.0, Lat: 52.7},
				{Lon: 13.0, Lat: 52.3},
			}},
		})
	}

	return NewRegionEngine(regions)
}
