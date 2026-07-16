package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//go:embed frontend/modules/osint_watch.js frontend/modules/globe_math.js frontend/modules/global_watch_projection.js frontend/modules/osint/*.js
var osintFrontend embed.FS

// loadShippedOSINTGraph returns the public entry plus all split modules under osint/.
func loadShippedOSINTGraph(t *testing.T) (entry, graph string) {
	t.Helper()
	entryBytes, err := osintFrontend.ReadFile("frontend/modules/osint_watch.js")
	if err != nil {
		t.Fatalf("missing osint_watch.js: %v", err)
	}
	var b strings.Builder
	b.Write(entryBytes)
	b.WriteByte('\n')
	entries, err := osintFrontend.ReadDir("frontend/modules/osint")
	if err != nil {
		t.Fatalf("read osint package: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".js") {
			continue
		}
		chunk, err := osintFrontend.ReadFile("frontend/modules/osint/" + e.Name())
		if err != nil {
			t.Fatalf("read osint/%s: %v", e.Name(), err)
		}
		b.Write(chunk)
		b.WriteByte('\n')
	}
	return string(entryBytes), b.String()
}

func TestModelRegistryHasDirectWailsFallback(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("frontend", "modules", "api.js"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if !strings.Contains(source, "GetAvailableModels") || !strings.Contains(source, "window.go?.main?.App") {
		t.Fatal("model registry lost its direct Wails fallback")
	}
}

func TestModelRegistryRetriesTransientEmptyBootstrapState(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("frontend", "modules", "ui.js"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	for _, required := range []string{
		"CORE_READINESS_RETRY_DELAYS_MS",
		"MODEL_REGISTRY_RETRY_DELAYS_MS",
		"applyModelRegistryPendingState",
		"return loadModels(retryCount + 1)",
		"return checkSystemStatus(retryCount + 1)",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("bootstrap recovery lost required marker %q", required)
		}
	}
}

// jsExportNames returns local export bindings for a JS module source.
// Handles: export function/const/let/async function/class, export { a, b as c },
// and export * from './x.js' (star re-exports are expanded via starTargets).
func jsExportNames(src string) (names map[string]bool, starTargets []string) {
	names = map[string]bool{}
	lines := strings.Split(src, "\n")
	for i := 0; i < len(lines); i++ {
		trim := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trim, "export ") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(trim, "export "))
		if strings.HasPrefix(rest, "default ") {
			continue
		}
		if strings.HasPrefix(rest, "*") {
			// export * from './x.js'
			fromIdx := strings.Index(rest, "from ")
			if fromIdx < 0 {
				continue
			}
			pathPart := strings.TrimSpace(rest[fromIdx+len("from "):])
			pathPart = strings.Trim(pathPart, ";")
			pathPart = strings.Trim(pathPart, `"'`)
			if pathPart != "" {
				starTargets = append(starTargets, pathPart)
			}
			continue
		}
		if strings.HasPrefix(rest, "{") {
			// export { a, b as c }  possibly multi-line
			block := rest
			for !strings.Contains(block, "}") && i+1 < len(lines) {
				i++
				block += " " + strings.TrimSpace(lines[i])
			}
			inner := block
			if lb := strings.Index(inner, "{"); lb >= 0 {
				if rb := strings.Index(inner[lb:], "}"); rb >= 0 {
					inner = inner[lb+1 : lb+rb]
				}
			}
			for _, part := range strings.Split(inner, ",") {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				// b as c → exported name is c; a → a
				if asIdx := strings.Index(part, " as "); asIdx >= 0 {
					part = strings.TrimSpace(part[asIdx+4:])
				}
				part = strings.TrimSpace(strings.TrimSuffix(part, ";"))
				if part != "" {
					names[part] = true
				}
			}
			continue
		}
		// export async function name / export function name / export const name / export let name / export class name
		for _, prefix := range []string{"async function ", "function ", "const ", "let ", "var ", "class "} {
			if strings.HasPrefix(rest, prefix) {
				ident := strings.TrimSpace(rest[len(prefix):])
				// cut at (, =, {, space
				for _, cut := range []string{"(", "=", "{", " ", "\t"} {
					if j := strings.Index(ident, cut); j >= 0 {
						ident = ident[:j]
					}
				}
				ident = strings.TrimSpace(ident)
				if ident != "" {
					names[ident] = true
				}
				break
			}
		}
	}
	return names, starTargets
}

// jsNamedImportsFromRelative returns map[importSpec][]localNames for relative named imports.
func jsNamedImportsFromRelative(src string) map[string][]string {
	out := map[string][]string{}
	lines := strings.Split(src, "\n")
	for i := 0; i < len(lines); i++ {
		trim := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trim, "import ") {
			continue
		}
		// Accumulate multi-line import until we see from '...'
		block := trim
		for !strings.Contains(block, " from ") && i+1 < len(lines) {
			i++
			block += " " + strings.TrimSpace(lines[i])
		}
		fromIdx := strings.Index(block, " from ")
		if fromIdx < 0 {
			continue
		}
		clause := strings.TrimSpace(block[len("import "):fromIdx])
		pathPart := strings.TrimSpace(block[fromIdx+len(" from "):])
		pathPart = strings.TrimSuffix(pathPart, ";")
		pathPart = strings.TrimSpace(pathPart)
		if len(pathPart) < 3 {
			continue
		}
		quote := pathPart[0]
		if quote != '\'' && quote != '"' {
			continue
		}
		end := strings.IndexByte(pathPart[1:], quote)
		if end < 0 {
			continue
		}
		spec := pathPart[1 : 1+end]
		if !strings.HasPrefix(spec, ".") {
			continue
		}
		// Only named imports { a, b as c }; skip default/namespace for binding check
		lb := strings.Index(clause, "{")
		rb := strings.LastIndex(clause, "}")
		if lb < 0 || rb <= lb {
			// side-effect or default import — path existence checked separately
			if _, ok := out[spec]; !ok {
				out[spec] = nil
			}
			continue
		}
		inner := clause[lb+1 : rb]
		var names []string
		for _, part := range strings.Split(inner, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			// original export name is left of " as "
			if asIdx := strings.Index(part, " as "); asIdx >= 0 {
				part = strings.TrimSpace(part[:asIdx])
			}
			if part != "" && part != "type" {
				names = append(names, part)
			}
		}
		out[spec] = append(out[spec], names...)
	}
	return out
}

// resolveJSExports expands export * from chains within root (one hop recursive with memo).
func resolveJSExports(root, file string, memo map[string]map[string]bool) map[string]bool {
	if m, ok := memo[file]; ok {
		return m
	}
	// prevent cycles
	memo[file] = map[string]bool{}
	raw, err := os.ReadFile(file)
	if err != nil {
		return memo[file]
	}
	names, stars := jsExportNames(string(raw))
	for k := range names {
		memo[file][k] = true
	}
	dir := filepath.Dir(file)
	for _, star := range stars {
		target := filepath.Clean(filepath.Join(dir, star))
		// only expand package-local stars (osint/* and parent modules under frontend/modules)
		if !strings.Contains(filepath.ToSlash(target), "/frontend/modules/") &&
			!strings.Contains(filepath.ToSlash(target), `\frontend\modules\`) {
			// still try
		}
		for k := range resolveJSExports(root, target, memo) {
			memo[file][k] = true
		}
	}
	return memo[file]
}

// TestOSINTModuleImportGraph verifies every relative import under osint/ resolves
// to an existing file AND that named import bindings exist as exports of the target
// (catches briefing_and_reader importing appendTextElement/epistemicLayer from the wrong module).
func TestOSINTModuleImportGraph(t *testing.T) {
	root := filepath.Join("frontend", "modules", "osint")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	memo := map[string]map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".js") {
			continue
		}
		path := filepath.Join(root, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		imports := jsNamedImportsFromRelative(string(raw))
		for spec, names := range imports {
			target := filepath.Clean(filepath.Join(root, spec))
			if _, err := os.Stat(target); err != nil {
				t.Errorf("%s imports missing path %s (resolved %s)", e.Name(), spec, target)
				continue
			}
			exports := resolveJSExports(root, target, memo)
			for _, name := range names {
				if !exports[name] {
					t.Errorf("%s imports %q from %s but that module does not export it", e.Name(), name, spec)
				}
			}
		}
	}
	// Entry surface
	app, err := os.ReadFile(filepath.Join("frontend", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(app), "modules/osint_watch.js") || !strings.Contains(string(app), "initGlobalWatch") {
		t.Fatal("app.js must load osint_watch.js / initGlobalWatch")
	}
	ui, err := os.ReadFile(filepath.Join("frontend", "modules", "ui.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ui), "osint_watch.js") || !strings.Contains(string(ui), "refreshOSINTFeed") {
		t.Fatal("ui.js must obtain refreshOSINTFeed from osint_watch.js")
	}
	entry, err := os.ReadFile(filepath.Join("frontend", "modules", "osint_watch.js"))
	if err != nil {
		t.Fatal(err)
	}
	es := string(entry)
	for _, sym := range []string{"initGlobalWatch", "refreshOSINTFeed", "forceGlobeResize"} {
		if !strings.Contains(es, sym) {
			t.Errorf("public entry missing re-export of %s", sym)
		}
	}
	// Explicit regression: briefing must bind appendTextElement + epistemicLayer correctly
	briefing, err := os.ReadFile(filepath.Join(root, "briefing_and_reader.js"))
	if err != nil {
		t.Fatal(err)
	}
	bImports := jsNamedImportsFromRelative(string(briefing))
	// appendTextElement from projection
	foundAppend := false
	foundEpistemic := false
	for spec, names := range bImports {
		for _, n := range names {
			if n == "appendTextElement" {
				foundAppend = true
				target := filepath.Clean(filepath.Join(root, spec))
				if !resolveJSExports(root, target, memo)["appendTextElement"] {
					t.Errorf("briefing_and_reader appendTextElement from %s not exported", spec)
				}
				if !strings.Contains(spec, "projection") {
					t.Errorf("appendTextElement should come from projection.js, got %s", spec)
				}
			}
			if n == "epistemicLayer" {
				foundEpistemic = true
				target := filepath.Clean(filepath.Join(root, spec))
				if !resolveJSExports(root, target, memo)["epistemicLayer"] {
					t.Errorf("briefing_and_reader epistemicLayer from %s not exported", spec)
				}
			}
		}
	}
	if !foundAppend {
		t.Error("briefing_and_reader must import appendTextElement")
	}
	if !foundEpistemic {
		t.Error("briefing_and_reader must import epistemicLayer")
	}

	// Hazard animation + personal impact wiring (split regressions)
	uiCtrl, err := os.ReadFile(filepath.Join(root, "ui_controls.js"))
	if err != nil {
		t.Fatal(err)
	}
	uiSrc := string(uiCtrl)
	// Must not reference unbound globeLayers in hazard scheduler
	if strings.Contains(uiSrc, "globeLayers.earthquakes") || strings.Contains(uiSrc, "globeLayers.volcanoes") {
		t.Error("scheduleGlobalWatchHazardAnimation must not use unbound globeLayers; use visibleLayers proxy")
	}
	if !strings.Contains(uiSrc, "visibleLayers.earthquakes") || !strings.Contains(uiSrc, "visibleLayers.volcanoes") {
		t.Error("hazard animation must gate on visibleLayers.earthquakes/volcanoes")
	}
	body, ok := extractJSFunctionBody(uiSrc, "initGlobalWatch")
	if !ok {
		t.Fatal("initGlobalWatch missing from ui_controls.js")
	}
	if !strings.Contains(body, "scheduleGlobalWatchHazardAnimation") {
		t.Error("initGlobalWatch must call scheduleGlobalWatchHazardAnimation (DEFAULTS.hazardAnimation=true)")
	}
	if !strings.Contains(body, "scheduleGlobalWatchAutoRefresh") {
		t.Error("initGlobalWatch must call scheduleGlobalWatchAutoRefresh")
	}
	if !strings.Contains(uiSrc, "function wirePersonalImpactControls") && !strings.Contains(uiSrc, "wirePersonalImpactControls()") {
		t.Error("wirePersonalImpactControls must exist and be invoked")
	}
	if !strings.Contains(body, "wirePersonalImpactControls") {
		t.Error("initGlobalWatch must wire personal impact controls")
	}
	if !strings.Contains(uiSrc, "gw-personal-sync") || !strings.Contains(uiSrc, "gw-personal-refresh") {
		t.Error("personal impact buttons #gw-personal-sync / #gw-personal-refresh must be bound in UI code")
	}
	if !strings.Contains(uiSrc, "syncPersonalAndRefreshImpact") || !strings.Contains(uiSrc, "loadAndRenderPersonalImpact") {
		t.Error("personal impact handlers must be referenced from ui_controls")
	}
	html, err := os.ReadFile(filepath.Join("frontend", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	htmlS := string(html)
	if !strings.Contains(htmlS, `id="gw-personal-sync"`) || !strings.Contains(htmlS, `id="gw-personal-refresh"`) {
		t.Error("index.html must still expose personal impact buttons")
	}
}

func TestOSINTFrontendGlobeNoCDNAndInlinedMath(t *testing.T) { // BETA V2 - structural proof of shipped Global Watch (split under osint/*)

	entry, watch := loadShippedOSINTGraph(t)
	mathBytes, err := osintFrontend.ReadFile("frontend/modules/globe_math.js")
	if err != nil {
		t.Fatalf("missing globe_math.js: %v", err)
	}
	projectionBytes, err := osintFrontend.ReadFile("frontend/modules/global_watch_projection.js")
	if err != nil {
		t.Fatalf("missing global_watch_projection.js: %v", err)
	}
	math := string(mathBytes)

	// Public entry must re-export the split (not still be a 4k-line monolith).
	if !strings.Contains(entry, "./osint/") || !strings.Contains(entry, "initGlobalWatch") {
		t.Error("osint_watch.js must re-export initGlobalWatch from ./osint/")
	}
	if strings.Count(entry, "\n") > 120 && strings.Contains(entry, "async function initGlobalWatch") {
		t.Error("osint_watch.js still looks like a monolith; expected thin re-export entry")
	}

	// No CDN or external three
	if strings.Contains(watch, "cdnjs") || strings.Contains(watch, "three.js") || strings.Contains(watch, "resium") || strings.Contains(watch, "cesium") {
		t.Error("CDN or three.js reference found in shipped OSINT graph")
	}
	if strings.Contains(math, "cdnjs") || strings.Contains(math, "three.js") {
		t.Error("CDN or three.js reference found in globe_math.js")
	}

	// No duplicate top-level let localGlobe* (the bug)
	dupeCount := strings.Count(watch, "\nexport let localGlobeCanvas") + strings.Count(watch, "\nlet localGlobeCanvas")
	if dupeCount > 1 {
		t.Errorf("duplicate localGlobeCanvas declarations: %d", dupeCount)
	}

	// Math helpers live in pure leaf modules (projection/hazards) after the split
	if !strings.Contains(watch, "function projectLatLon(") || !strings.Contains(watch, "function buildGlobePins(") {
		t.Error("OSINT graph missing projectLatLon / buildGlobePins definitions")
	}
	if !strings.Contains(watch, "function applyDomainFilter(") {
		t.Error("OSINT graph missing applyDomainFilter")
	}

	// Draw uses the functions directly (no GlobeMath. prefix)
	if !strings.Contains(watch, "projectLatLon(c.lat, c.lon") {
		t.Error("globe draw does not call projectLatLon for continents (part of drawPureLocalGlobe)")
	}
	if !strings.Contains(watch, "buildGlobePins(activeFeedEvents") {
		t.Error("OSINT graph does not use buildGlobePins (for drawPureLocalGlobe)")
	}
	if !strings.Contains(watch, "applyDragDelta(") || !strings.Contains(watch, "applyWheelZoom(") {
		t.Error("OSINT graph does not use drag/zoom FSM")
	}

	// globe_math.js is no longer required for the draw (kept for reference only)
	_ = math

	// no external CDN in camera layer (sovereign)
	if strings.Contains(watch, "picsum.photos") || strings.Contains(watch, "cdn.") {
		t.Error("external image cdn in OSINT for cameras")
	}

	// Broken prefs import must not reappear (was ./global_watch_preferences inside osint/)
	if strings.Contains(watch, "from './global_watch_preferences.js'") {
		t.Error("osint package must import prefs from ../global_watch_preferences.js, not a missing sibling")
	}

	// Global Watch epistemic layers + safe DOM (criterion 2 + 4)
	if !strings.Contains(watch, "function epistemicLayer(") {
		t.Error("OSINT graph missing epistemicLayer for RAW/INFERENCE/VERIFIED UI")
	}
	if !strings.Contains(watch, "textContent") {
		t.Error("osint_watch must use textContent for user/event strings")
	}
	if !strings.Contains(watch, "gw-risk-item") && !strings.Contains(watch, "loadAndRenderRegionalRisks") {
		t.Error("risk HUD renderer missing")
	}
	// Feed cards must not assign untrusted event fields via innerHTML template injection
	if strings.Contains(watch, "innerHTML = `\n                <div style=\"display:flex") {
		t.Error("legacy innerHTML event card template must not return")
	}
	// risks layer must be top-level (not nested under cameras) and draw real markers
	if !strings.Contains(watch, "cachedRiskMarkers") {
		t.Error("risks layer must use cachedRiskMarkers from SharedIntelStore-backed HUD fetch")
	}
	if strings.Contains(watch, "Stub risk overlay") || strings.Contains(watch, "placeholder circle for high risk") {
		t.Error("risks layer must not remain a decorative stub")
	}
	// Toast must not inject message via innerHTML
	if strings.Contains(watch, "toast.innerHTML = `") {
		t.Error("showAethelToast must not use innerHTML for message body")
	}
	if strings.Contains(watch, "contentHtml: `<div>Möchtest du die RSS-Quelle <strong>${name}</strong>") {
		t.Error("collector delete must not interpolate name into contentHtml")
	}

	// WWV maximal: gazetteer not in JS but geo correct via backend; has_geo respected; new layers + overlays present
	if !strings.Contains(watch, "has_geo") || !strings.Contains(watch, "visibleLayers") {
		t.Error("osint_watch missing has_geo / visibleLayers handling for correct news placement + toggles")
	}
	if !strings.Contains(watch, "citiesData") || !strings.Contains(watch, "CITIES") {
		t.Error("missing cities layer or data for graphic overlay")
	}
	if !strings.Contains(watch, "subsolarLon") || !strings.Contains(watch, "terminator") {
		t.Error("day/night terminator not using real UTC subsolar calc")
	}
	// Corrected orthographic: east = +X now lives in the single-source module
	// consumed by every Global Watch layer.
	projection := string(projectionBytes)
	if !strings.Contains(watch, "GlobeProjection.project(") || !strings.Contains(projection, "cosPhi * Math.sin(lambda)") {
		t.Error("projectLatLon must use east=+X geographic formula (was mirrored)")
	}
	if !strings.Contains(watch, "function focusGlobeOnLonLat(") && !strings.Contains(watch, "export function focusGlobeOnLonLat(") {
		t.Error("globe focus helper required for feed/risk/map linkage")
	}
	// U3 personal impact surface + U5 promote event id plumbing
	if !strings.Contains(watch, "loadAndRenderPersonalImpact") || !strings.Contains(watch, "syncPersonalAndRefreshImpact") {
		t.Error("personal impact strip handlers missing from osint_watch")
	}
	if !strings.Contains(watch, "/v1/intelligence/personal/impact") || !strings.Contains(watch, "/v1/intelligence/personal/sync") {
		t.Error("personal impact/sync API paths must be present")
	}
	if !strings.Contains(watch, "AETHEL_PROMOTE_TO_CASE(title, summary, source, pLat, pLon, eventId)") &&
		!strings.Contains(watch, "AETHEL_PROMOTE_TO_CASE(ev.title, ev.summary, ev.source, ev.lat, ev.lon, eventId)") {
		t.Error("promote must pass source event id for case audit provenance")
	}
	// Command-center UI: connectors + intel detail + clustering
	if !strings.Contains(watch, "loadAndRenderConnectors") || !strings.Contains(watch, "/v1/intelligence/connectors") {
		t.Error("connectors panel wiring missing")
	}
	if !strings.Contains(watch, "fillIntelDetailPanel") || !strings.Contains(watch, "wireRegionChips") {
		t.Error("intel detail / region chips missing")
	}
	if !strings.Contains(watch, "__globeClusters") {
		t.Error("news pin clustering expected for dense feeds")
	}
	// Earth texture path (local JPG + API + atlas bake + orthographic sample)
	if !strings.Contains(watch, "drawEarthGlobeTexture") || !strings.Contains(watch, "earth_day.jpg") {
		t.Error("earth texture globe renderer / local asset path missing")
	}
	if !strings.Contains(watch, "buildEarthTextureFromAtlas") {
		t.Error("atlas-baked earth texture fallback missing")
	}
	if !strings.Contains(watch, "/v1/assets/earth-texture") {
		t.Error("earth texture API fallback path missing in frontend loader")
	}
	// High-res local maps: must not reintroduce the old 1536px quality cap
	if strings.Contains(watch, "maxW = 1536") || strings.Contains(watch, "maxW=1536") {
		t.Error("earth texture must not downscale to 1536px; local 4k–8k maps are intended")
	}
	if !strings.Contains(watch, "EARTH_TEX_MAX_W") {
		t.Error("earth texture max width constant missing")
	}
	if !strings.Contains(watch, "sampleEarthTex") {
		t.Error("earth texture sampler missing")
	}
	// Geometry / focus / time window (goal gating)
	if !strings.Contains(watch, "computeGlobeFocusRotation") || !strings.Contains(watch, "REGION_FOCUS_TABLE") {
		t.Error("shared focus table/math missing")
	}
	if !strings.Contains(watch, "filterEventsByTimeWindow") || !strings.Contains(watch, "gw-time-window") {
		// time control is in HTML; JS must filter
		if !strings.Contains(watch, "filterEventsByTimeWindow") {
			t.Error("time window filter helper missing")
		}
	}
	if !strings.Contains(watch, "AETHEL_NAVIGATE_VIEW") || !strings.Contains(watch, "AETHEL_GW_COMMAND") {
		t.Error("AI GW control bridge missing")
	}
	// Perf + shared inverse (this goal)
	if !strings.Contains(watch, "computeEarthRasterStep") || !strings.Contains(watch, "EARTH_TEX_WORK_MAX_W") {
		t.Error("adaptive raster step / texture work cap missing")
	}
	if strings.Contains(watch, "nightPoints.forEach") || !strings.Contains(watch, "ctx.createLinearGradient(") {
		t.Error("day/night layer regressed to a visible point/cell raster")
	}
	if !strings.Contains(watch, "runGlobeIdleRotation") || !strings.Contains(watch, "setInterval") {
		t.Error("reliable idle globe rotation heartbeat missing")
	}
	if !strings.Contains(watch, "GLOBE_IDLE_ROTATION_FRAME_MS = 40") ||
		!strings.Contains(watch, "GLOBE_IDLE_ROTATION_RADIANS_PER_SECOND = 0.022") {
		t.Error("idle globe rotation must retain the smooth Beta V2 cadence and speed")
	}
	if strings.Contains(watch, "GEO:${geoCount}") {
		t.Error("duplicate canvas diagnostics overlap the single globe hint")
	}
	if !strings.Contains(watch, "presentNewGlobeEvent") || !strings.Contains(watch, "performance.now() + 10000") {
		t.Error("ten-second new-event focus/resume lifecycle missing")
	}
	if !strings.Contains(watch, "unprojectScreenToLatLon") || !strings.Contains(watch, "projectUnprojectRoundTrip") {
		t.Error("shared unproject / round-trip helpers missing for N/S drift fix")
	}
	if !strings.Contains(watch, "makeGwPanelDraggable") || !strings.Contains(watch, "clampGwPanelPosition") {
		t.Error("GW panel drag helpers missing")
	}
	if !strings.Contains(watch, "wireGwFloatPanels") {
		t.Error("wireGwFloatPanels missing")
	}
	// Hazard globe layers + article reader helpers
	if !strings.Contains(watch, "function magnitudeColorBand(") || !strings.Contains(watch, "function volcanoMarkerColor(") {
		t.Error("hazard color helpers missing")
	}
	if !strings.Contains(watch, "function formatArticleDateTime(") {
		t.Error("article datetime formatter missing")
	}
	if !strings.Contains(watch, "earthquakes:") || !strings.Contains(watch, "volcanoes:") {
		t.Error("earthquake/volcano globe layers missing")
	}
	if !strings.Contains(watch, "openGwReportReader") {
		t.Error("report/article reader open path missing")
	}
}

// extractJSFunctionBody returns the body of the first function/async function
// named name found in source (brace-balanced, string/comment aware enough for our JS).
func extractJSFunctionBody(source, name string) (string, bool) {
	markers := []string{
		"async function " + name + "(",
		"function " + name + "(",
		"export async function " + name + "(",
		"export function " + name + "(",
	}
	start := -1
	for _, m := range markers {
		if i := strings.Index(source, m); i >= 0 {
			start = i
			break
		}
	}
	if start < 0 {
		return "", false
	}
	brace := strings.Index(source[start:], "{")
	if brace < 0 {
		return "", false
	}
	i := start + brace
	depth := 0
	inSingle, inDouble, inTemplate, inLineComment, inBlockComment := false, false, false, false, false
	escape := false
	for j := i; j < len(source); j++ {
		c := source[j]
		if inLineComment {
			if c == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			if c == '*' && j+1 < len(source) && source[j+1] == '/' {
				inBlockComment = false
				j++
			}
			continue
		}
		if inSingle {
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == '\'' {
				inSingle = false
			}
			continue
		}
		if inDouble {
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == '"' {
				inDouble = false
			}
			continue
		}
		if inTemplate {
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == '`' {
				inTemplate = false
			}
			continue
		}
		if c == '/' && j+1 < len(source) {
			switch source[j+1] {
			case '/':
				inLineComment = true
				j++
				continue
			case '*':
				inBlockComment = true
				j++
				continue
			}
		}
		switch c {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '`':
			inTemplate = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[i+1 : j], true
			}
		}
	}
	return "", false
}

// briefingFataler is the minimal surface assertLagebriefingTriggerBody needs
// so negative fixtures can prove failure without failing the parent *testing.T.
type briefingFataler interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
}

type briefingFailProbe struct {
	failed bool
	msg    string
}

func (p *briefingFailProbe) Helper() {}
func (p *briefingFailProbe) Fatal(args ...any) {
	p.failed = true
	p.msg = strings.TrimSpace(strings.Join(func() []string {
		out := make([]string, len(args))
		for i, a := range args {
			out[i] = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(fmt.Sprint(a), "\n", " "), "\r", ""))
		}
		return out
	}(), " "))
}
func (p *briefingFailProbe) Fatalf(format string, args ...any) {
	p.failed = true
	p.msg = strings.TrimSpace(fmt.Sprintf(format, args...))
}

// assertLagebriefingTriggerBody enforces the structural contract for the shipped
// Global Watch briefing trigger body (extracted from real JS, or synthetic fixtures).
func assertLagebriefingTriggerBody(t briefingFataler, body string) {
	t.Helper()

	localDefIdx := strings.Index(body, "const setTxt")
	if localDefIdx < 0 {
		localDefIdx = strings.Index(body, "function setTxt")
	}
	if localDefIdx < 0 {
		localDefIdx = strings.Index(body, "let setTxt")
	}
	if localDefIdx < 0 {
		t.Fatal("triggerAIBriefing must define a local setTxt (const/let/function) before status DOM updates")
	}

	statusWrite := strings.Index(body, "setTxt('gw-briefing-status'")
	if statusWrite < 0 {
		statusWrite = strings.Index(body, `setTxt("gw-briefing-status"`)
	}
	if statusWrite < 0 {
		t.Fatal("triggerAIBriefing must update gw-briefing-status via setTxt")
	}
	if localDefIdx > statusWrite {
		t.Fatal("local setTxt definition must appear before the first gw-briefing-status write")
	}

	for _, id := range []string{"gw-briefing-window", "gw-briefing-observations", "gw-briefing-model"} {
		idx := strings.Index(body, "setTxt('"+id+"'")
		if idx < 0 {
			idx = strings.Index(body, `setTxt("`+id+`"`)
		}
		if idx < 0 {
			t.Fatalf("triggerAIBriefing must setTxt %s", id)
		}
		if idx < localDefIdx {
			t.Fatalf("setTxt(%s) must not run before local setTxt is defined", id)
		}
	}

	fetchIdx := strings.Index(body, "/v1/osint/briefing")
	if fetchIdx < 0 {
		t.Fatal("triggerAIBriefing must POST/fetch /v1/osint/briefing")
	}
	if !strings.Contains(body, "fetch(") {
		t.Fatal("triggerAIBriefing must call fetch for the briefing API")
	}
	if fetchIdx < localDefIdx || fetchIdx < statusWrite {
		t.Fatal("briefing fetch must run after local setTxt definition and status context updates")
	}

	tryIdx := strings.Index(body, "try {")
	if tryIdx < 0 {
		tryIdx = strings.Index(body, "try{")
	}
	catchIdx := strings.Index(body, "catch")
	finallyIdx := strings.Index(body, "finally")
	if tryIdx < 0 || catchIdx < 0 || finallyIdx < 0 {
		t.Fatal("triggerAIBriefing must use try/catch/finally around the briefing fetch")
	}
	if !(tryIdx < fetchIdx && fetchIdx < catchIdx && catchIdx < finallyIdx) {
		t.Fatal("fetch must sit inside try before catch/finally")
	}
	if !strings.Contains(body[catchIdx:], "replaceChildren") && !strings.Contains(body[catchIdx:], "Briefing abgebrochen") {
		t.Fatal("catch path must clear the loader/content and surface a failure message")
	}
	if !strings.Contains(body[finallyIdx:], `classList.remove("loading")`) && !strings.Contains(body[finallyIdx:], "classList.remove('loading')") {
		t.Fatal("finally must remove the briefing button loading class")
	}
	if !strings.Contains(body, "INTELLIGENCE STREAMS WERDEN KORRELIERT") {
		t.Fatal("loader correlating copy must remain on the shipped path")
	}
}

// TestLagebriefingSetTxtGuardRejectsMissingLocalDef proves the assertion fails
// on the pre-fix pattern (setTxt used without a function-local definition).
func TestLagebriefingSetTxtGuardRejectsMissingLocalDef(t *testing.T) {
	broken := `
async function triggerAIBriefing() {
    content.appendChild(loader);
    loaderText.textContent = 'INTELLIGENCE STREAMS WERDEN KORRELIERT…';
    setTxt('gw-briefing-window', '24H');
    setTxt('gw-briefing-observations', '0');
    setTxt('gw-briefing-model', 'AUTO');
    setTxt('gw-briefing-status', 'ANALYSE LÄUFT');
    try {
        await fetch(API + '/v1/osint/briefing', { method: 'POST' });
    } catch (e) {
        content.replaceChildren();
        error.textContent = 'Briefing abgebrochen';
    } finally {
        briefingBtn.classList.remove("loading");
    }
}
`
	body, ok := extractJSFunctionBody(broken, "triggerAIBriefing")
	if !ok {
		t.Fatal("fixture extract failed")
	}
	if strings.Contains(body, "const setTxt") || strings.Contains(body, "let setTxt") || strings.Contains(body, "function setTxt") {
		t.Fatal("broken fixture must not define local setTxt")
	}
	probe := &briefingFailProbe{}
	assertLagebriefingTriggerBody(probe, body)
	if !probe.failed {
		t.Fatal("assertLagebriefingTriggerBody must FAIL when local setTxt is missing")
	}
	if !strings.Contains(probe.msg, "local setTxt") {
		t.Fatalf("expected local setTxt failure message, got %q", probe.msg)
	}
}

// TestShippedLagebriefingDefinesLocalSetTxt guards the Global Watch briefing
// regression: a missing in-function setTxt threw after the correlating loader
// was shown, so POST /v1/osint/briefing never ran and the UI hung.
func TestShippedLagebriefingDefinesLocalSetTxt(t *testing.T) {
	// Shipped entry path (same module app.js dynamic-imports for Global Watch).
	appBytes, err := os.ReadFile(filepath.Join("frontend", "app.js"))
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}
	app := string(appBytes)
	if !strings.Contains(app, "modules/osint_watch.js") || !strings.Contains(app, "initGlobalWatch") {
		t.Fatal("frontend/app.js must load modules/osint_watch.js and call initGlobalWatch")
	}

	// After the split, briefing lives in osint/briefing_and_reader.js and is
	// re-exported from the public entry.
	entry, graph := loadShippedOSINTGraph(t)
	if !strings.Contains(entry, "triggerAIBriefing") {
		t.Fatal("public osint_watch.js must re-export triggerAIBriefing")
	}

	body, ok := extractJSFunctionBody(graph, "triggerAIBriefing")
	if !ok {
		t.Fatal("shipped OSINT graph missing triggerAIBriefing definition")
	}
	assertLagebriefingTriggerBody(t, body)

	// Positive synthetic fixture must also pass the same contract (proves guard is not vacuous).
	good := `
async function triggerAIBriefing() {
    const loader = document.createElement('div');
    loaderText.textContent = 'INTELLIGENCE STREAMS WERDEN KORRELIERT…';
    content.appendChild(loader);
    const setTxt = (id, v) => { const el = document.getElementById(id); if (el) el.textContent = String(v); };
    setTxt('gw-briefing-window', '24H');
    setTxt('gw-briefing-observations', '3');
    setTxt('gw-briefing-model', 'AUTO');
    setTxt('gw-briefing-status', 'ANALYSE LÄUFT');
    try {
        await fetch(API + '/v1/osint/briefing', { method: 'POST' });
    } catch (e) {
        content.replaceChildren();
        error.textContent = 'Briefing abgebrochen';
    } finally {
        briefingBtn.classList.remove("loading");
    }
}
`
	goodBody, ok := extractJSFunctionBody(good, "triggerAIBriefing")
	if !ok {
		t.Fatal("good fixture extract failed")
	}
	assertLagebriefingTriggerBody(t, goodBody)
}

func mustRead(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return b
}

// TestFullAppProductionUILayer drives shipped frontend files for the whole-app UI overhaul.
func TestFullAppProductionUILayer(t *testing.T) {
	htmlBytes, err := os.ReadFile(filepath.Join("frontend", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(htmlBytes)
	cssBase, err := os.ReadFile(filepath.Join("frontend", "style.css"))
	if err != nil {
		t.Fatal(err)
	}
	cssProd, err := os.ReadFile(filepath.Join("frontend", "aethel-ui-production.css"))
	if err != nil {
		t.Fatal(err)
	}
	css := string(cssBase) + "\n" + string(cssProd)
	appBytes, err := os.ReadFile(filepath.Join("frontend", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	modernizationBytes, err := os.ReadFile(filepath.Join("frontend", "modules", "ui_modernization.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appBytes)
	modernization := string(modernizationBytes)

	// Splash frozen
	if !strings.Contains(html, `id="startup-splash-screen"`) || !strings.Contains(css, ".startup-splash {") {
		t.Error("startup splash must remain present")
	}
	// Production stylesheet linked after style.css
	if !strings.Contains(html, `href="aethel-ui-production.css"`) {
		t.Error("production UI stylesheet must be linked from index.html")
	}
	// Design system tokens + primitives
	for _, tok := range []string{
		"--radius-md", "--space-4", "--ds-panel", "--control-h", "--ease-out",
		".ds-section-title", ".ds-page", "#view-chat .input-wrapper",
		"#view-agent #agent-workspace-grid", "#view-settings", ".vgt-view-header",
		".vgt-mobile-nav-toggle", ":focus-visible", "prefers-reduced-motion",
	} {
		if !strings.Contains(css, tok) {
			t.Errorf("design system primitive/token missing: %s", tok)
		}
	}
	if !strings.Contains(app, "setupUIModernization") || !strings.Contains(modernization, "setupResponsiveNavigation") {
		t.Error("product UI modernization must be initialized by the shipped app")
	}
	for _, view := range []string{"chat", "agent", "security", "memory", "personal", "tasks", "personas", "archive"} {
		if !strings.Contains(modernization, view+":") {
			t.Errorf("modern view header metadata missing: %s", view)
		}
	}
	if strings.Contains(modernization, "innerHTML") || strings.Contains(modernization, "insertAdjacentHTML") {
		t.Error("UI modernization must render through safe DOM construction")
	}
	// All primary views still navigable
	for _, id := range []string{
		"view-core", "view-chat", "view-agent", "view-control", "view-sphere",
		"view-memory", "view-personal", "view-global-watch", "view-tasks",
		"view-case", "view-security", "view-archive", "view-settings", "view-personas",
	} {
		if !strings.Contains(html, `id="`+id+`"`) {
			t.Errorf("primary view missing: %s", id)
		}
	}
	for _, nav := range []string{
		"nav-btn-core", "nav-btn-chat", "nav-btn-agent", "nav-btn-control",
		"nav-btn-sphere", "nav-btn-memory", "nav-btn-personal", "nav-btn-global-watch",
		"nav-btn-tasks", "nav-btn-case", "nav-btn-security", "nav-btn-archive",
		"nav-btn-settings", "nav-btn-personas",
	} {
		if !strings.Contains(html, `id="`+nav+`"`) {
			t.Errorf("nav button missing: %s", nav)
		}
	}
	// Chat / Agent / Settings depth
	if !strings.Contains(html, `id="chat-output"`) || !strings.Contains(html, `id="user-input"`) || !strings.Contains(html, `id="btn-send"`) {
		t.Error("chat composer/output structure missing")
	}
	if !strings.Contains(html, `id="model-dropdown"`) || !strings.Contains(html, `id="orchestrator-model-dropdown"`) {
		t.Error("domain-model and orchestrator-model selectors must both be visible")
	}
	if !strings.Contains(html, `id="agent-workspace-grid"`) || !strings.Contains(html, `id="agent-btn-launch-team"`) {
		t.Error("agent workspace structure missing")
	}
	if !strings.Contains(html, `id="settings-btn-save"`) || !strings.Contains(html, `id="settings-groq-input"`) {
		t.Error("settings key management structure missing")
	}
	// Chat message path (shipped JS class names used by CSS)
	chatJS, err := os.ReadFile(filepath.Join("frontend", "modules", "chat_addmessage.js"))
	if err != nil {
		t.Fatal(err)
	}
	cjs := string(chatJS)
	if !strings.Contains(cjs, "message ${role}") {
		t.Error("chat_addmessage must still emit message role classes")
	}
	if !strings.Contains(cjs, "msg-header") || !strings.Contains(cjs, "msg-body") {
		t.Error("chat message structure classes missing")
	}
	if !strings.Contains(cjs, "Aethel") || !strings.Contains(cjs, "thinking-details") {
		t.Error("chat headers/thinking UI hooks missing")
	}
	// Shell primitives
	if !strings.Contains(html, `class="glass-card header-bar"`) && !strings.Contains(html, "header-bar") {
		t.Error("header-bar shell missing")
	}
	if !strings.Contains(html, `class="glass-card sidebar"`) && !strings.Contains(html, "sidebar") {
		t.Error("sidebar shell missing")
	}

	// Cascade regression: production CSS must use :not(.hidden) and explicit .hidden rules.
	// Bare #view-chat { display:flex !important } beats .hidden and stacks panels over switchMode().
	prodOnly := string(cssProd)
	for _, sel := range []string{
		"#view-chat:not(.hidden)",
		"#view-agent:not(.hidden)",
		"#view-control:not(.hidden)",
		"#view-sphere:not(.hidden)",
		"#view-chat.hidden",
		"#view-agent.hidden",
		"#view-control.hidden",
		"#view-sphere.hidden",
	} {
		if !strings.Contains(prodOnly, sel) {
			t.Errorf("production CSS missing hide-safe selector: %s", sel)
		}
	}
	// Forbid cascade-breaking bare ID display:!important blocks
	for _, f := range []string{
		"#view-chat {\n  display: flex !important",
		"#view-agent {\n  display: flex !important",
		"#view-control {\n  display: flex !important",
	} {
		if strings.Contains(prodOnly, f) {
			t.Errorf("forbidden cascade-breaking rule present: %q", f)
		}
	}
	// Also catch single-line form
	if strings.Contains(prodOnly, "#view-chat { display: flex !important") {
		t.Error("forbidden: #view-chat display flex !important without :not(.hidden)")
	}

	// Agent role density: shared classes, no legacy 9px/4px on orchestrator select
	agentSlice := html
	if i := strings.Index(html, `id="view-agent"`); i >= 0 {
		agentSlice = html[i:]
		if j := strings.Index(agentSlice, `id="view-control"`); j > 0 {
			agentSlice = agentSlice[:j]
		}
	}
	if !strings.Contains(agentSlice, `agent-role-card`) || !strings.Contains(agentSlice, `agent-role-prompt`) {
		t.Error("agent role cards must use shared agent-role-* classes")
	}
	if strings.Contains(agentSlice, `padding: 4px; font-size: 9px`) {
		t.Error("agent role selects/textareas still use legacy 9px/4px inline density")
	}
	// Settings provider section: ds-section-title immediately before health list
	idx := strings.Index(html, `id="settings-provider-health-list"`)
	if idx < 0 {
		t.Error("settings-provider-health-list missing")
	} else {
		start := idx - 500
		if start < 0 {
			start = 0
		}
		chunk := html[start:idx]
		if !strings.Contains(chunk, `ds-section-title`) || !strings.Contains(chunk, `Provider-Gesundheit`) {
			t.Error("Provider-Gesundheit heading must use ds-section-title (not bespoke inline h4)")
		}
	}
}

// TestSidebarExpandAndSettingsModernUI pins the flex-shrink sidebar fix and modern Settings surface.
func TestSidebarExpandAndSettingsModernUI(t *testing.T) {
	htmlBytes, err := os.ReadFile(filepath.Join("frontend", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(htmlBytes)
	cssBase, err := os.ReadFile(filepath.Join("frontend", "style.css"))
	if err != nil {
		t.Fatal(err)
	}
	cssProd, err := os.ReadFile(filepath.Join("frontend", "aethel-ui-production.css"))
	if err != nil {
		t.Fatal(err)
	}
	css := string(cssBase) + "\n" + string(cssProd)
	prod := string(cssProd)

	// Sidebar expand fix: section content must not flex-shrink under flex .sidebar-scroll
	if !strings.Contains(prod, "flex-shrink: 0") {
		t.Error("production CSS must set flex-shrink: 0 on sidebar section blocks")
	}
	if !strings.Contains(prod, ".nav-section-content.collapsed") {
		t.Error("production CSS must keep .nav-section-content.collapsed hide rule")
	}
	// Hard requirement: direct child flex-shrink rule on section content (the clip fix)
	if !strings.Contains(prod, ".sidebar-scroll > .nav-section-content") {
		t.Error("sidebar expand fix requires .sidebar-scroll > .nav-section-content { flex-shrink:0 }")
	}
	// Expanded content must allow full height (max-height none or equivalent)
	if !strings.Contains(prod, "max-height: none") && !strings.Contains(string(cssBase), "max-height: none") {
		t.Error("expanded .nav-section-content must use max-height: none (not clip to one row)")
	}
	// Collapsed still hides
	if !strings.Contains(css, ".nav-section-content.collapsed") {
		t.Error("collapsed rule for nav-section-content missing")
	}
	if !strings.Contains(prod, "max-height: 0 !important") && !strings.Contains(string(cssBase), "max-height: 0") {
		t.Error("collapsed max-height:0 missing")
	}

	// All primary nav IDs intact
	for _, id := range []string{
		"nav-btn-core", "nav-btn-chat", "nav-btn-personas", "nav-btn-agent", "nav-btn-control",
		"nav-btn-sphere", "nav-btn-memory", "nav-btn-personal",
		"nav-btn-global-watch", "nav-btn-tasks",
		"nav-btn-case", "nav-btn-security", "nav-btn-archive", "nav-btn-settings",
	} {
		if !strings.Contains(html, `id="`+id+`"`) {
			t.Errorf("nav button missing: %s", id)
		}
	}
	// Section content hosts multiple buttons (assistant has chat+agent etc.)
	for _, pair := range []struct{ section, btn string }{
		{"nav-section-assistant", "nav-btn-chat"},
		{"nav-section-assistant", "nav-btn-agent"},
		{"nav-section-workspace", "nav-btn-memory"},
		{"nav-section-global-watch", "nav-btn-tasks"},
		{"nav-section-case-workspace", "nav-btn-security"},
	} {
		si := strings.Index(html, `id="`+pair.section+`"`)
		if si < 0 {
			t.Errorf("section missing: %s", pair.section)
			continue
		}
		// next section or sidebar-controls ends the block roughly
		end := si + 2500
		if end > len(html) {
			end = len(html)
		}
		chunk := html[si:end]
		if !strings.Contains(chunk, `id="`+pair.btn+`"`) {
			t.Errorf("expected %s inside/near %s", pair.btn, pair.section)
		}
	}

	// Modern Settings surface
	if !strings.Contains(html, `id="view-settings"`) || !strings.Contains(html, `settings-view`) {
		t.Error("view-settings must use settings-view class")
	}
	if !strings.Contains(html, `settings-page-header`) || !strings.Contains(html, `settings-layout`) {
		t.Error("settings modern page header/layout missing")
	}
	for _, needle := range []string{
		`settings-card`,
		`id="settings-groq-input"`,
		`id="settings-btn-save"`,
		`id="settings-btn-force-setup"`,
		`id="settings-btn-reset"`,
		`id="settings-provider-health-list"`,
		`id="settings-mounted-dirs"`,
		`ds-section-title`,
	} {
		if !strings.Contains(html, needle) {
			t.Errorf("settings structure missing: %s", needle)
		}
	}
	if !strings.Contains(prod, ".settings-layout") || !strings.Contains(prod, ".settings-page-title") {
		t.Error("settings production CSS layout rules missing")
	}

	// Splash frozen
	if !strings.Contains(html, `id="startup-splash-screen"`) || !strings.Contains(css, ".startup-splash {") {
		t.Error("splash must remain present")
	}
}

func TestGlobalWatchChromeAndSphereAndNeuralCoreStructure(t *testing.T) {
	htmlBytes, err := os.ReadFile(filepath.Join("frontend", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(htmlBytes)
	for _, needle := range []string{
		`id="gw-time-window"`,
		`id="gw-settings-modal"`,
		`id="gw-ai-command"`,
		`id="gw-report-reader"`,
		`id="gw-feed-panel"`,
		`data-focus="germany"`,
		`id="neural-core-greeting"`,
		`id="neural-core-first-run"`,
		`id="editor-btn-export-html"`,
		`id="editor-btn-export-md"`,
		`sphere-window-editor`,
		`class="sphere-window glass-card hidden" id="sphere-window-editor"`,
		`class="sphere-window glass-card hidden" id="sphere-window-browser"`,
		`id="gw-risk-hud-collapse"`,
		`id="gw-risk-hud-body"`,
		`gw-float-drag-handle`,
		// UI overhaul: settings hosts Data Configuration + Alerts; feed right-docked; research reader
		`id="gw-intel-detail"`,
		`id="gw-alerts-center"`,
		`id="gw-reader-date"`,
		`id="gw-reader-time"`,
		`id="gw-reader-source"`,
		`data-layer="earthquakes"`,
		`data-layer="volcanoes"`,
		`gw-feed-dock`,
		`id="startup-splash-screen"`,
	} {
		if !strings.Contains(html, needle) {
			t.Errorf("index.html missing structure: %s", needle)
		}
	}
	// Data Configuration + Alerts must live under settings modal (not main right rail clutter)
	settingsIdx := strings.Index(html, `id="gw-settings-modal"`)
	intelIdx := strings.Index(html, `id="gw-intel-detail"`)
	alertsIdx := strings.Index(html, `id="gw-alerts-center"`)
	if settingsIdx < 0 || intelIdx < settingsIdx || alertsIdx < settingsIdx {
		t.Error("Data Configuration and Alerts must be nested after gw-settings-modal open")
	}
	// Sphere must not force full-desktop editor geometry with !important
	cssBytes, err := os.ReadFile(filepath.Join("frontend", "style.css"))
	if err != nil {
		t.Fatal(err)
	}
	css := string(cssBytes)
	if strings.Contains(css, "#sphere-window-editor { left:clamp") || strings.Contains(css, "height:calc(100% - 56px) !important") {
		t.Error("sphere window must not force full-desktop !important layout")
	}
	if !strings.Contains(css, "gw-feed-dock") || !strings.Contains(css, "neural-core-hero") {
		t.Error("GW right-docked feed / neural core styles missing")
	}
	// Design system tokens (app-wide, not GW-only hacks)
	for _, tok := range []string{"--radius-md", "--space-4", "--ds-panel", "--hz-green", "--hz-orange", "--hz-red"} {
		if !strings.Contains(css, tok) {
			t.Errorf("design system token missing: %s", tok)
		}
	}
	// Full-scale reader (not tiny popup footprint)
	if !strings.Contains(css, "gw-report-reader-shell") || !strings.Contains(css, "min(920px") {
		t.Error("full-scale article reader styles missing")
	}
	// Splash identity preserved
	if !strings.Contains(css, ".startup-splash {") {
		t.Error("startup splash CSS must remain")
	}
	sphereBytes, err := os.ReadFile(filepath.Join("frontend", "modules", "sphere.js"))
	if err != nil {
		t.Fatal(err)
	}
	sphere := string(sphereBytes)
	if !strings.Contains(sphere, "exportSphereDocument") || !strings.Contains(sphere, "buildSphereDocumentExport") {
		t.Error("sphere document export helpers missing")
	}
	if !strings.Contains(sphere, "populateNotesApp") {
		t.Error("sphere notes workspace app missing")
	}
	personalBytes, err := os.ReadFile(filepath.Join("frontend", "modules", "personal_mode.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(personalBytes), "refreshNeuralCoreHome") || !strings.Contains(string(personalBytes), "Willkommen zurück") {
		t.Error("neural core welcome-back path missing")
	}
	startupGreetingBytes, err := os.ReadFile(filepath.Join("frontend", "modules", "startup_greeting.js"))
	if err != nil {
		t.Fatal(err)
	}
	startupGreeting := string(startupGreetingBytes)
	if !strings.Contains(startupGreeting, "api.getPersonalStatus()") || !strings.Contains(startupGreeting, "profile?.display_name") {
		t.Error("spoken startup greeting must resolve the saved Personal Core display name")
	}
	if !strings.Contains(startupGreeting, "greetingSpoken") || !strings.Contains(startupGreeting, "greeting.fallback") {
		t.Error("spoken startup greeting must be idempotent and retain an offline fallback")
	}
	appBytes, err := os.ReadFile(filepath.Join("frontend", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	appSource := string(appBytes)
	if !strings.Contains(appSource, "speakPersonalizedStartupGreeting") {
		t.Error("startup disclosure acceptance must trigger the personalized greeting")
	}
	if strings.Contains(appSource, "Core initialisiert. Willkommen bei Äthel.") {
		t.Error("legacy hard-coded spoken startup greeting must not remain in app.js")
	}
}
