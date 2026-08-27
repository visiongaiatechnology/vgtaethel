// STATUS: DIAMANT VGT SUPREME
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fixtureStormSnapshot() SpaceWeatherResponse {
	return SpaceWeatherResponse{
		Timestamp:          "2026-07-29T12:00:00Z",
		KpIndex:            6.3,
		KpStatus:           "STURM (G2)",
		GScale:             2,
		RScale:             1,
		SScale:             0,
		SolarWindSpeed:     620,
		SolarWindDensity:   12,
		BtTotal:            18,
		BzVector:           -12,
		DstIndex:           -85,
		DstStatus:          "STORM",
		SolarXRayFlux:      "M1.2",
		XRayFlux:           1.2e-5,
		ProtonFlux10MeV:    2.5,
		SunspotCount:       140,
		FlareMax72h:        "M2.1",
		FlareLastClass:     "M1.2",
		FlareLastTime:      "11:40 (29.07)",
		GeomagneticField:   "GESTÖRT",
		AuroraActivity:     "STURM",
		AuroraMinLat:       "55° N",
		AuroraConfidence:   80,
		AuroraHemispheric:  45,
		GeoForecast:        "STORM RISK",
		ProbMClass:         40,
		ProbXClass:         12,
		ProbProton:         8,
	}
}

func TestBuildSpaceAnalysisPackages_OrderedDomains(t *testing.T) {
	pkgs := BuildSpaceAnalysisPackages(fixtureStormSnapshot())
	wantIDs := []string{"geomag", "flares", "protons", "wind_imf", "aurora"}
	if len(pkgs) != len(wantIDs) {
		t.Fatalf("expected %d packages, got %d", len(wantIDs), len(pkgs))
	}
	for i, id := range wantIDs {
		if pkgs[i].ID != id {
			t.Fatalf("package %d: want %s got %s", i, id, pkgs[i].ID)
		}
		if pkgs[i].Title == "" || pkgs[i].RiskLabel == "" || len(pkgs[i].Facts) == 0 {
			t.Fatalf("package %s incomplete: %+v", id, pkgs[i])
		}
	}
	if pkgs[0].RiskLabel != "G2" {
		t.Fatalf("geomag risk label want G2 got %s", pkgs[0].RiskLabel)
	}
	if pkgs[1].RiskLabel != "R1" {
		t.Fatalf("flares risk label want R1 got %s", pkgs[1].RiskLabel)
	}
}

func TestDeriveSpaceRiskClasses_FromFixture(t *testing.T) {
	risks := DeriveSpaceRiskClasses(fixtureStormSnapshot())
	if risks.GeomagG != "G2" || risks.RadioR != "R1" || risks.RadiationS != "S0" {
		t.Fatalf("scale labels wrong: %+v", risks)
	}
	if risks.Overall == "" || risks.OverallScore <= 0 {
		t.Fatalf("overall risk not derived: %+v", risks)
	}
	// Storm Kp/G2 + elevated wind/Bz should be at least ELEVATED
	if risks.Overall == "LOW" {
		t.Fatalf("storm fixture must not be LOW overall: %+v", risks)
	}
	if !strings.Contains(risks.Summary, risks.Overall) {
		t.Fatalf("summary must mention overall class")
	}
}

func TestDeterministicPackageNoteAndFinalReportRetainIntermediates(t *testing.T) {
	snap := fixtureStormSnapshot()
	pkgs := BuildSpaceAnalysisPackages(snap)
	risks := DeriveSpaceRiskClasses(snap)
	intermediates := make([]SpaceIntermediateReport, 0, len(pkgs))
	for _, pkg := range pkgs {
		note := DeterministicPackageNote(pkg, snap)
		if !strings.Contains(note, pkg.Title) {
			t.Fatalf("note missing title for %s", pkg.ID)
		}
		intermediates = append(intermediates, SpaceIntermediateReport{
			PackageID: pkg.ID,
			Title:     pkg.Title,
			RiskLabel: pkg.RiskLabel,
			Body:      note,
			Source:    "deterministic",
		})
	}
	if len(intermediates) != 5 {
		t.Fatalf("expected 5 intermediate reports, got %d", len(intermediates))
	}
	final := BuildFinalReportMarkdown(snap, risks, intermediates, "")
	for _, id := range []string{"Geomagnetik", "Flares", "Protonen", "Sonnenwind", "Aurora", "Risikoklassen", risks.Overall, risks.GeomagG, risks.RadioR} {
		if !strings.Contains(final, id) {
			t.Fatalf("final report missing %q", id)
		}
	}
	// All package ids appear in intermediates section via titles
	if !strings.Contains(final, "Zwischenberichte") {
		t.Fatal("final report must list intermediate section")
	}
}

func TestHandleSpaceAnalysis_SSEPipelineWithoutModel(t *testing.T) {
	// state may be nil → deterministic fallbacks only; pipeline must still complete.
	snap := fixtureStormSnapshot()
	body, _ := json.Marshal(SpaceAnalysisRequest{ModelID: "", Snapshot: &snap})
	req := httptest.NewRequest(http.MethodPost, "/v1/space/analysis", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	HandleSpaceAnalysis(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("expected SSE content-type, got %q", ct)
	}
	out := rec.Body.String()
	if !strings.Contains(out, `"type":"pipeline"`) {
		t.Fatal("missing pipeline event")
	}
	if !strings.Contains(out, `"type":"step_start"`) {
		t.Fatal("missing step_start")
	}
	if !strings.Contains(out, `"type":"step_done"`) {
		t.Fatal("missing step_done")
	}
	if !strings.Contains(out, `"type":"final"`) {
		t.Fatal("missing final event")
	}
	if !strings.Contains(out, "data: [DONE]") {
		t.Fatal("missing DONE")
	}
	// Risk classes in pipeline/final
	if !strings.Contains(out, "risk_classes") && !strings.Contains(out, "overall") {
		t.Fatal("risk classes not present in stream")
	}
	// Five domain packages + synthesis
	if strings.Count(out, `"package_id":"geomag"`) < 1 {
		// step_done embeds package_id
		if !strings.Contains(out, "geomag") {
			t.Fatal("geomag package not in stream")
		}
	}
}

func TestHandleSpaceAnalysis_EmptyBody400(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/space/analysis", bytes.NewReader([]byte("not-json")))
	rec := httptest.NewRecorder()
	HandleSpaceAnalysis(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", rec.Code)
	}
}
