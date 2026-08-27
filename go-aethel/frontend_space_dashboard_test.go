// STATUS: DIAMANT VGT SUPREME
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Structural contract for the Space Dashboard front-end (Solarcommander layout in Aethel style).
func TestSpaceDashboardUIStructure(t *testing.T) {
	htmlBytes, err := os.ReadFile(filepath.Join("frontend", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	jsBytes, err := os.ReadFile(filepath.Join("frontend", "modules", "space_dashboard.js"))
	if err != nil {
		t.Fatal(err)
	}
	cssBytes, err := os.ReadFile(filepath.Join("frontend", "space-dashboard.css"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(htmlBytes)
	js := string(jsBytes)
	css := string(cssBytes)

	// Shell + stylesheet linkage
	for _, needle := range []string{
		`id="view-space"`,
		`id="space-dashboard-content"`,
		`href="space-dashboard.css"`,
	} {
		if !strings.Contains(html, needle) {
			t.Errorf("index.html missing %q", needle)
		}
	}

	// Multi-region mission layout builders
	for _, needle := range []string{
		"space-mission-grid",
		"buildImager",
		"buildScales",
		"buildAurora",
		"buildAnalysisConsole",
		"sdoProxyURL",
		"/v1/space/sdo_image?channel=",
		"activeSdoChannel",
		"space-channel-btn",
		"space-img-error",
		"space-img-loader",
		"btn-refresh-space",
		"btn-generate-space-analysis",
		"/v1/space/weather",
		"fetchSpaceWeatherData",
		"generateSpaceAnalysis",
		"STALE_AFTER_MS",
		"forceImages",
		"space-stale-banner",
		"space-data-age",
		"MANUELLER SYNC",
		"buildAuroraMap",
		"buildForecast",
		"buildCharts",
		"buildFlareHistory",
		"buildWindMag",
		"buildDstCard",
		"buildKpCard",
		"aurora_n",
		"space-telemetry-chart",
		"space-chart-tooltip",
		"series_kp",
		"series_dst",
		"fetchSpaceSnapshotWithRetry",
		"INITIAL_UPLINK_RETRIES_MS",
		"ResizeObserver",
		"bindChartInteractions",
		"prob_m_class",
		"SYNC_COOLDOWN_MS",
		"space-analysis-modal",
		"/v1/space/analysis",
		"openAnalysisModal",
		"handleAnalysisEvent",
		"space-analysis-risk-row",
		"space-analysis-steps",
		"space-analysis-final-body",
		"setFinalReport",
		"space-analysis-tabs",
		"space-analysis-orbit",
		"startThinkingSimulation",
		"PACKAGE_VISUALS",
		"switchAnalysisTab",
		"space-step-progress",
		"setStepProgress",
		"space-orbit-pct",
		"overallPipelinePercent",
	} {
		if !strings.Contains(js, needle) {
			t.Errorf("space_dashboard.js missing contract token %q", needle)
		}
	}

	// Must NOT auto-poll weather/images on a short interval (NASA blacklist risk).
	if strings.Contains(js, "setInterval(fetchSpaceWeatherData") {
		t.Error("must not setInterval(fetchSpaceWeatherData) — images only on manual SYNC")
	}
	if strings.Contains(js, "setInterval(fetchSpaceWeatherData, 30000)") {
		t.Error("30s weather poll removed")
	}

	// Analysis must not primarily redirect to chat.
	if strings.Contains(js, "switchMode('chat')") || strings.Contains(js, "switchMode(\"chat\")") {
		t.Error("space analysis must not navigate to chat as primary path")
	}
	if strings.Contains(js, "sendMessage(") {
		t.Error("space analysis must not use sendMessage chat path as primary")
	}

	// Must use local proxy, not raw GSFC URLs in the browser
	if strings.Contains(js, "sdo.gsfc.nasa.gov") {
		t.Error("dashboard must not load SDO images cross-origin from GSFC; use local proxy")
	}

	for _, needle := range []string{
		"space-analysis-modal",
		"space-risk-chip",
		"space-analysis-panel-wide",
		"space-analysis-tabs",
		"space-analysis-orbit",
		"space-orbit-spin",
		"space-analysis-think-log",
	} {
		if !strings.Contains(css, needle) {
			t.Errorf("space-dashboard.css missing %q", needle)
		}
	}

	// Aethel-token CSS (not a Tailwind/Solarcommander port)
	for _, needle := range []string{
		"#view-space",
		".space-mission-grid",
		".space-imager",
		".space-channel-controls",
		".space-img-error",
		"var(--vgt-cyan)",
		"var(--vgt-orange)",
		"var(--vgt-purple)",
		"var(--font-mono)",
	} {
		if !strings.Contains(css, needle) {
			t.Errorf("space-dashboard.css missing %q", needle)
		}
	}
	if strings.Contains(css, "tailwind") || strings.Contains(css, "#vg-sun-root") {
		t.Error("must not wholesale-copy Solarcommander Tailwind root skin")
	}
}
