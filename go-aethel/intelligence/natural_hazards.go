package intelligence

// STATUS: DIAMANT VGT SUPREME

import "strings"

const (
	HazardEarthquake = "earthquake"
	HazardVolcano    = "volcano"
)

// DetectNaturalHazard classifies machine-originated natural-hazard records.
// Explicit collector/source markers prevent ordinary reporting from being
// misclassified solely because an article discusses a natural hazard.
func DetectNaturalHazard(source, title, summary string) string {
	sourceKey := strings.ToLower(strings.TrimSpace(source))
	text := strings.ToLower(strings.TrimSpace(title + " " + summary))

	if strings.Contains(sourceKey, "usgs-earthquake") ||
		strings.Contains(sourceKey, "earthquake-geojson") ||
		strings.Contains(text, "[earthquake]") {
		return HazardEarthquake
	}
	if strings.Contains(sourceKey, "nasa-eonet-volcano") ||
		strings.Contains(sourceKey, "volcano-eonet") ||
		strings.Contains(sourceKey, "builtin-volcano") ||
		strings.Contains(text, "[volcano") {
		return HazardVolcano
	}
	return ""
}
