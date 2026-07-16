package intelligence

// GetDefaultRegionEngine returns a region engine loaded with approximate coordinates of major states
func GetDefaultRegionEngine() *RegionEngine {
	germany := Region{
		ID:   "GERMANY",
		Name: "Germany",
		Type: "state",
		Polygons: [][]Point{{
			{Lon: 5.8, Lat: 47.2},
			{Lon: 15.0, Lat: 47.2},
			{Lon: 15.0, Lat: 55.0},
			{Lon: 5.8, Lat: 55.0},
			{Lon: 5.8, Lat: 47.2},
		}},
	}
	
	france := Region{
		ID:   "FRANCE",
		Name: "France",
		Type: "state",
		Polygons: [][]Point{{
			{Lon: -5.0, Lat: 42.3},
			{Lon: 9.5, Lat: 42.3},
			{Lon: 9.5, Lat: 51.1},
			{Lon: -5.0, Lat: 51.1},
			{Lon: -5.0, Lat: 42.3},
		}},
	}

	usa := Region{
		ID:   "USA",
		Name: "United States",
		Type: "state",
		Polygons: [][]Point{{
			{Lon: -125.0, Lat: 24.5},
			{Lon: -66.9, Lat: 24.5},
			{Lon: -66.9, Lat: 49.0},
			{Lon: -125.0, Lat: 49.0},
			{Lon: -125.0, Lat: 24.5},
		}},
	}

	ukraine := Region{
		ID:   "UKRAINE",
		Name: "Ukraine",
		Type: "state",
		Polygons: [][]Point{{
			{Lon: 22.0, Lat: 44.3},
			{Lon: 40.2, Lat: 44.3},
			{Lon: 40.2, Lat: 52.4},
			{Lon: 22.0, Lat: 52.4},
			{Lon: 22.0, Lat: 44.3},
		}},
	}

	uk := Region{
		ID:   "UK",
		Name: "United Kingdom",
		Type: "state",
		Polygons: [][]Point{{
			{Lon: -8.6, Lat: 49.9},
			{Lon: 1.7, Lat: 49.9},
			{Lon: 1.7, Lat: 60.8},
			{Lon: -8.6, Lat: 60.8},
			{Lon: -8.6, Lat: 49.9},
		}},
	}

	berlin := Region{
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
	}

	// Approximate bounding boxes — deterministic scoring anchors, not political claims.
	poland := Region{
		ID:   "POLAND",
		Name: "Poland",
		Type: "state",
		Polygons: [][]Point{{
			{Lon: 14.1, Lat: 49.0},
			{Lon: 24.2, Lat: 49.0},
			{Lon: 24.2, Lat: 54.9},
			{Lon: 14.1, Lat: 54.9},
			{Lon: 14.1, Lat: 49.0},
		}},
	}
	baltics := Region{
		ID:   "BALTICS",
		Name: "Baltics (EE/LV/LT approx)",
		Type: "state",
		Polygons: [][]Point{{
			{Lon: 20.9, Lat: 53.9},
			{Lon: 28.3, Lat: 53.9},
			{Lon: 28.3, Lat: 59.7},
			{Lon: 20.9, Lat: 59.7},
			{Lon: 20.9, Lat: 53.9},
		}},
	}
	taiwan := Region{
		ID:   "TAIWAN",
		Name: "Taiwan",
		Type: "state",
		Polygons: [][]Point{{
			{Lon: 119.3, Lat: 21.8},
			{Lon: 122.1, Lat: 21.8},
			{Lon: 122.1, Lat: 25.4},
			{Lon: 119.3, Lat: 25.4},
			{Lon: 119.3, Lat: 21.8},
		}},
	}
	israel := Region{
		ID:   "ISRAEL",
		Name: "Israel",
		Type: "state",
		Polygons: [][]Point{{
			{Lon: 34.2, Lat: 29.4},
			{Lon: 35.9, Lat: 29.4},
			{Lon: 35.9, Lat: 33.4},
			{Lon: 34.2, Lat: 33.4},
			{Lon: 34.2, Lat: 29.4},
		}},
	}

	return NewRegionEngine([]Region{germany, france, usa, ukraine, uk, berlin, poland, baltics, taiwan, israel})
}
