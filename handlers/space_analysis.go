// STATUS: DIAMANT VGT SUPREME
// Multi-step Space weather analysis pipeline (packages → intermediate reports → final risk report).
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// SpaceAnalysisPackage is one discrete telemetry domain analyzed before final synthesis.
type SpaceAnalysisPackage struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Focus       string            `json:"focus"`
	Facts       map[string]string `json:"facts"`
	RiskDomain  string            `json:"risk_domain"` // G | R | S | IMF | AURORA
	RiskLevel   int               `json:"risk_level"`  // 0-5 domain scale where applicable
	RiskLabel   string            `json:"risk_label"`
	PromptBrief string            `json:"prompt_brief"`
}

// SpaceRiskClasses are explicitly labeled risk tiers for the final report.
type SpaceRiskClasses struct {
	Overall      string `json:"overall"`       // LOW | ELEVATED | HIGH | SEVERE | EXTREME
	OverallScore int    `json:"overall_score"` // 0-100
	RadioR       string `json:"radio_r"`       // R0-R5
	RadiationS   string `json:"radiation_s"`   // S0-S5
	GeomagG      string `json:"geomag_g"`      // G0-G5
	WindIMF      string `json:"wind_imf"`      // LOW|MODERATE|HIGH|SEVERE
	Aurora       string `json:"aurora"`        // QUIET|ACTIVE|STORM|EXTREME
	Summary      string `json:"summary"`
}

// SpaceAnalysisRequest is the body for POST /v1/space/analysis.
type SpaceAnalysisRequest struct {
	ModelID  string                `json:"model_id"`
	Snapshot *SpaceWeatherResponse `json:"snapshot,omitempty"`
	Language string                `json:"language,omitempty"`
}

// SpaceIntermediateReport is retained after each package step for final synthesis.
type SpaceIntermediateReport struct {
	PackageID string `json:"package_id"`
	Title     string `json:"title"`
	RiskLabel string `json:"risk_label"`
	Body      string `json:"body"`
	Source    string `json:"source"` // model | deterministic
}

// BuildSpaceAnalysisPackages returns ordered domain packages from a weather snapshot.
// Order is fixed so the UI pipeline is deterministic and testable.
func BuildSpaceAnalysisPackages(snap SpaceWeatherResponse) []SpaceAnalysisPackage {
	gLabel := fmt.Sprintf("G%d", clampScale(snap.GScale))
	rLabel := fmt.Sprintf("R%d", clampScale(snap.RScale))
	sLabel := fmt.Sprintf("S%d", clampScale(snap.SScale))

	return []SpaceAnalysisPackage{
		{
			ID:         "geomag",
			Title:      "Geomagnetik · Kp / G / Dst",
			Focus:      "Geomagnetische Sturmstufe, Dst und Feldstatus",
			RiskDomain: "G",
			RiskLevel:  clampScale(snap.GScale),
			RiskLabel:  gLabel,
			Facts: map[string]string{
				"kp_index":          fmt.Sprintf("%.1f", snap.KpIndex),
				"kp_status":         snap.KpStatus,
				"g_scale":           gLabel,
				"dst_nt":            fmt.Sprintf("%.1f", snap.DstIndex),
				"dst_status":        snap.DstStatus,
				"geomagnetic_field": snap.GeomagneticField,
			},
			PromptBrief: "Bewerte Kp, G-Skala und Dst. Nenne Auswirkungen auf GPS, Netze und HF. 4-7 Sätze, Deutsch, faktenbasiert.",
		},
		{
			ID:         "flares",
			Title:      "Flares · X-Ray / R-Skala",
			Focus:      "Röntgenfluss, Flare-Historie und Radio-Blackout-Risiko",
			RiskDomain: "R",
			RiskLevel:  clampScale(snap.RScale),
			RiskLabel:  rLabel,
			Facts: map[string]string{
				"xray_class":       snap.SolarXRayFlux,
				"xray_flux":        fmt.Sprintf("%.3e", snap.XRayFlux),
				"r_scale":          rLabel,
				"flare_max_72h":    snap.FlareMax72h,
				"flare_last_class": snap.FlareLastClass,
				"flare_last_time":  snap.FlareLastTime,
				"sunspot_count":    fmt.Sprintf("%d", snap.SunspotCount),
			},
			PromptBrief: "Bewerte aktuellen X-Ray-Fluss, R-Skala und Flare-Historie. Radio-Blackout-Risiko klar benennen. 4-7 Sätze, Deutsch.",
		},
		{
			ID:         "protons",
			Title:      "Protonen · S-Skala",
			Focus:      "Solar energetic particles und Radiation Storms",
			RiskDomain: "S",
			RiskLevel:  clampScale(snap.SScale),
			RiskLabel:  sLabel,
			Facts: map[string]string{
				"proton_flux_10mev": fmt.Sprintf("%.2f", snap.ProtonFlux10MeV),
				"s_scale":           sLabel,
				"prob_proton":       fmt.Sprintf("%d%%", snap.ProbProton),
			},
			PromptBrief: "Bewerte Protonenfluss ≥10 MeV und S-Skala. Satelliten-/Polarflug-Risiken. 4-7 Sätze, Deutsch.",
		},
		{
			ID:         "wind_imf",
			Title:      "Sonnenwind · IMF Bt/Bz",
			Focus:      "Solar wind speed/density und interplanetares Magnetfeld",
			RiskDomain: "IMF",
			RiskLevel:  imfRiskLevel(snap),
			RiskLabel:  imfRiskLabel(snap),
			Facts: map[string]string{
				"wind_km_s": fmt.Sprintf("%.0f", snap.SolarWindSpeed),
				"density":   fmt.Sprintf("%.1f", snap.SolarWindDensity),
				"bt_nt":     fmt.Sprintf("%.1f", snap.BtTotal),
				"bz_nt":     fmt.Sprintf("%.1f", snap.BzVector),
			},
			PromptBrief: "Bewerte Wind, Dichte, Bt und Bz (Süd/Nord). Coupling-Potenzial mit Magnetosphäre. 4-7 Sätze, Deutsch.",
		},
		{
			ID:         "aurora",
			Title:      "Aurora · Sichtbarkeit",
			Focus:      "Polarlicht-Oval, Hemispheric Power und Mitteleuropa-Outlook",
			RiskDomain: "AURORA",
			RiskLevel:  auroraRiskLevel(snap),
			RiskLabel:  auroraRiskLabel(snap),
			Facts: map[string]string{
				"activity":       snap.AuroraActivity,
				"min_lat":        snap.AuroraMinLat,
				"confidence":     fmt.Sprintf("%d%%", snap.AuroraConfidence),
				"hemispheric_gw": fmt.Sprintf("%.1f", snap.AuroraHemispheric),
				"geo_forecast":   snap.GeoForecast,
			},
			PromptBrief: "Bewerte Aurora-Lage und Sichtbarkeit für hohe/mittlere Breiten inkl. Mitteleuropa-Chance. 4-7 Sätze, Deutsch.",
		},
	}
}

// DeriveSpaceRiskClasses computes explicit risk tiers from the snapshot scales/telemetry.
func DeriveSpaceRiskClasses(snap SpaceWeatherResponse) SpaceRiskClasses {
	r := clampScale(snap.RScale)
	s := clampScale(snap.SScale)
	g := clampScale(snap.GScale)
	imf := imfRiskLevel(snap)

	score := r*12 + s*12 + g*14 + imf*8 + auroraRiskLevel(snap)*6
	if score > 100 {
		score = 100
	}
	overall := "LOW"
	switch {
	case score >= 80 || g >= 5 || r >= 4 || s >= 4:
		overall = "EXTREME"
	case score >= 60 || g >= 3 || r >= 3 || s >= 3:
		overall = "SEVERE"
	case score >= 40 || g >= 2 || r >= 2 || s >= 2 || imf >= 3:
		overall = "HIGH"
	case score >= 20 || g >= 1 || r >= 1 || s >= 1 || imf >= 2 || auroraRiskLevel(snap) >= 2:
		overall = "ELEVATED"
	}

	summary := fmt.Sprintf("Gesamt %s (%d/100) · %s · %s · %s · IMF %s · Aurora %s",
		overall, score, fmt.Sprintf("R%d", r), fmt.Sprintf("S%d", s), fmt.Sprintf("G%d", g),
		imfRiskLabel(snap), auroraRiskLabel(snap))

	return SpaceRiskClasses{
		Overall:      overall,
		OverallScore: score,
		RadioR:       fmt.Sprintf("R%d", r),
		RadiationS:   fmt.Sprintf("S%d", s),
		GeomagG:      fmt.Sprintf("G%d", g),
		WindIMF:      imfRiskLabel(snap),
		Aurora:       auroraRiskLabel(snap),
		Summary:      summary,
	}
}

// DeterministicPackageNote is the fallback narrative when no LLM is available.
func DeterministicPackageNote(pkg SpaceAnalysisPackage, snap SpaceWeatherResponse) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s\n", pkg.Title)
	fmt.Fprintf(&b, "**Domänenrisiko:** %s (Stufe %d)\n\n", pkg.RiskLabel, pkg.RiskLevel)
	fmt.Fprintf(&b, "%s\n\n", pkg.Focus)
	b.WriteString("**Messwerte:**\n")
	// Stable order for tests: sort-like by writing known keys first then rest is ok via range
	keys := make([]string, 0, len(pkg.Facts))
	for k := range pkg.Facts {
		keys = append(keys, k)
	}
	// simple insertion sort for stable output
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	for _, k := range keys {
		fmt.Fprintf(&b, "- `%s`: %s\n", k, pkg.Facts[k])
	}
	b.WriteString("\n_Deterministische Auswertung aus dem aktuellen Weltraumwetter-Snapshot (ohne Modellnarrativ)._\n")
	return b.String()
}

// BuildFinalReportMarkdown assembles intermediate package reports + risk classes into one document.
func BuildFinalReportMarkdown(snap SpaceWeatherResponse, risks SpaceRiskClasses, intermediates []SpaceIntermediateReport, modelSynthesis string) string {
	var b strings.Builder
	b.WriteString("# Weltraumwetter · Finaler Lagebericht\n\n")
	fmt.Fprintf(&b, "**Stand Snapshot:** %s  \n", snap.Timestamp)
	fmt.Fprintf(&b, "**Gesamtrisiko:** `%s` · Score **%d/100**  \n", risks.Overall, risks.OverallScore)
	fmt.Fprintf(&b, "**NOAA-Skalen:** `%s` · `%s` · `%s`  \n", risks.RadioR, risks.RadiationS, risks.GeomagG)
	fmt.Fprintf(&b, "**IMF/Wind:** `%s` · **Aurora:** `%s`  \n\n", risks.WindIMF, risks.Aurora)
	fmt.Fprintf(&b, "> %s\n\n", risks.Summary)

	b.WriteString("## Risikoklassen\n\n")
	b.WriteString("| Domäne | Klasse |\n|---|---|\n")
	fmt.Fprintf(&b, "| Gesamt | **%s** (%d/100) |\n", risks.Overall, risks.OverallScore)
	fmt.Fprintf(&b, "| Radio Blackouts | **%s** |\n", risks.RadioR)
	fmt.Fprintf(&b, "| Solar Radiation | **%s** |\n", risks.RadiationS)
	fmt.Fprintf(&b, "| Geomagnetic Storms | **%s** |\n", risks.GeomagG)
	fmt.Fprintf(&b, "| Solar Wind / IMF | **%s** |\n", risks.WindIMF)
	fmt.Fprintf(&b, "| Aurora | **%s** |\n\n", risks.Aurora)

	b.WriteString("## Zwischenberichte der Datenpakete\n\n")
	for i, ir := range intermediates {
		fmt.Fprintf(&b, "### %d. %s · `%s`\n\n", i+1, ir.Title, ir.RiskLabel)
		b.WriteString(strings.TrimSpace(ir.Body))
		b.WriteString("\n\n")
	}

	if strings.TrimSpace(modelSynthesis) != "" {
		b.WriteString("## Synthese (KI)\n\n")
		b.WriteString(strings.TrimSpace(modelSynthesis))
		b.WriteString("\n")
	} else {
		b.WriteString("## Synthese\n\n")
		b.WriteString("Die Domänenberichte oben fassen die aktuelle Lage. ")
		fmt.Fprintf(&b, "Maßgeblich ist das Gesamtrisiko **%s** mit G/R/S = %s/%s/%s. ",
			risks.Overall, risks.GeomagG, risks.RadioR, risks.RadiationS)
		b.WriteString("Handlungsempfehlung: kritische Infrastruktur und Navigation bei HIGH+ beobachten; Aurora-Chancen an Min-Breite und Kp koppeln.\n")
	}
	return b.String()
}

func clampScale(v int) int {
	if v < 0 {
		return 0
	}
	if v > 5 {
		return 5
	}
	return v
}

func imfRiskLevel(snap SpaceWeatherResponse) int {
	level := 0
	if snap.SolarWindSpeed >= 500 {
		level = 1
	}
	if snap.SolarWindSpeed >= 600 || snap.BtTotal >= 15 {
		level = 2
	}
	if snap.SolarWindSpeed >= 700 || snap.BtTotal >= 25 || snap.BzVector <= -10 {
		level = 3
	}
	if snap.BzVector <= -15 && snap.BtTotal >= 20 {
		level = 4
	}
	if snap.BzVector <= -20 && snap.SolarWindSpeed >= 700 {
		level = 5
	}
	return level
}

func imfRiskLabel(snap SpaceWeatherResponse) string {
	switch imfRiskLevel(snap) {
	case 0, 1:
		return "LOW"
	case 2:
		return "MODERATE"
	case 3:
		return "HIGH"
	default:
		return "SEVERE"
	}
}

func auroraRiskLevel(snap SpaceWeatherResponse) int {
	if snap.GScale >= 4 || snap.KpIndex >= 7 {
		return 4
	}
	if snap.GScale >= 2 || snap.KpIndex >= 5 {
		return 3
	}
	if snap.GScale >= 1 || snap.KpIndex >= 4 {
		return 2
	}
	if snap.KpIndex >= 3 {
		return 1
	}
	return 0
}

func auroraRiskLabel(snap SpaceWeatherResponse) string {
	switch auroraRiskLevel(snap) {
	case 0:
		return "QUIET"
	case 1, 2:
		return "ACTIVE"
	case 3:
		return "STORM"
	default:
		return "EXTREME"
	}
}

func factsBlock(facts map[string]string) string {
	keys := make([]string, 0, len(facts))
	for k := range facts {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "- %s: %s\n", k, facts[k])
	}
	return b.String()
}

func marshalSpaceChatRequest(modelID, systemPrompt, userPrompt string) ([]byte, error) {
	msgObj := map[string]string{"role": "user", "content": userPrompt}
	msgBytes, err := json.Marshal(msgObj)
	if err != nil {
		return nil, err
	}
	if state != nil && state.providers != nil {
		if selected, _ := state.providers.SelectAvailable(modelID, state, false, false); selected.ID != "" {
			modelID = selected.ID
		}
	}
	chatReq := ChatRequest{
		ModelID:             modelID,
		SystemPrompt:        systemPrompt,
		Messages:            []json.RawMessage{json.RawMessage(msgBytes)},
		Temperature:         0.2,
		UseTools:            false,
		ReasoningEffort:     "low",
		ReasoningVisibility: "hidden",
	}
	return json.Marshal(chatReq)
}

func invokeSpaceChatRaw(bodyBytes []byte) (string, error) {
	if len(bodyBytes) == 0 {
		return "", fmt.Errorf("empty chat request")
	}
	if state == nil || state.providers == nil {
		return "", fmt.Errorf("chat core unavailable")
	}
	req, err := http.NewRequest(http.MethodPost, "/v1/chat", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleChat(rec, req)
	if rec.Code >= 400 {
		snippet := rec.Body.String()
		if len(snippet) > 200 {
			snippet = snippet[:200] + "…"
		}
		return "", fmt.Errorf("chat status %d: %s", rec.Code, snippet)
	}
	return parseInternalSSE(rec.Body.String())
}

func writeAnalysisSSE(w http.ResponseWriter, flusher http.Flusher, eventType string, payload interface{}) {
	body, err := json.Marshal(map[string]interface{}{
		"type": eventType,
		"data": payload,
		"ts":   time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", body)
	if flusher != nil {
		flusher.Flush()
	}
}

// HandleSpaceAnalysis runs the multi-package pipeline and streams SSE progress events.
func HandleSpaceAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var req SpaceAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	var snap SpaceWeatherResponse
	if req.Snapshot != nil {
		snap = *req.Snapshot
	} else {
		snap = buildSpaceWeatherLive()
	}
	if strings.TrimSpace(snap.Timestamp) == "" {
		snap.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	packages := BuildSpaceAnalysisPackages(snap)
	risks := DeriveSpaceRiskClasses(snap)
	modelID := strings.TrimSpace(req.ModelID)

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, _ := w.(http.Flusher)

	writeAnalysisSSE(w, flusher, "pipeline", map[string]interface{}{
		"packages":     packages,
		"risk_classes": risks,
		"timestamp":    snap.Timestamp,
	})

	systemPkg := "Du bist Aethel Space Analyst. Antworte auf Deutsch, präzise, ohne Tools, ohne erfundene Messwerte. Nutze nur die gelieferten Fakten."
	intermediates := make([]SpaceIntermediateReport, 0, len(packages))

	for _, pkg := range packages {
		writeAnalysisSSE(w, flusher, "step_start", map[string]interface{}{
			"id":         pkg.ID,
			"title":      pkg.Title,
			"risk_label": pkg.RiskLabel,
			"focus":      pkg.Focus,
		})

		userPrompt := fmt.Sprintf("DATENPAKET: %s\nFOKUS: %s\nRISIKO: %s\nFAKTEN:\n%s\n\n%s",
			pkg.Title, pkg.Focus, pkg.RiskLabel, factsBlock(pkg.Facts), pkg.PromptBrief)

		body, src := "", "deterministic"
		if bodyBytes, err := marshalSpaceChatRequest(modelID, systemPkg, userPrompt); err == nil {
			if text, err := invokeSpaceChatRaw(bodyBytes); err == nil && strings.TrimSpace(text) != "" {
				body = strings.TrimSpace(text)
				src = "model"
			} else {
				body = DeterministicPackageNote(pkg, snap)
				if err != nil {
					body += fmt.Sprintf("\n\n_(Modell-Hinweis: %s — deterministischer Fallback.)_\n", truncateSpaceErr(err.Error(), 120))
				}
			}
		} else {
			body = DeterministicPackageNote(pkg, snap)
		}

		ir := SpaceIntermediateReport{
			PackageID: pkg.ID,
			Title:     pkg.Title,
			RiskLabel: pkg.RiskLabel,
			Body:      body,
			Source:    src,
		}
		intermediates = append(intermediates, ir)
		writeAnalysisSSE(w, flusher, "step_done", ir)
	}

	writeAnalysisSSE(w, flusher, "step_start", map[string]interface{}{
		"id":         "synthesis",
		"title":      "Finale Synthese",
		"risk_label": risks.Overall,
		"focus":      "Zusammenführung aller Domänenberichte",
	})

	var synthBuilder strings.Builder
	for _, ir := range intermediates {
		fmt.Fprintf(&synthBuilder, "## %s (%s)\n%s\n\n", ir.Title, ir.RiskLabel, ir.Body)
	}
	synthPrompt := fmt.Sprintf(`Erstelle eine knappe Executive-Synthese (max 12 Sätze) auf Deutsch aus den Domänenberichten.
Gesamtrisiko: %s (%d/100). Skalen: %s %s %s. IMF: %s. Aurora: %s.

Domänenberichte:
%s

Strukturiere: 1) Lagebild 2) Kritische Risiken 3) Betroffene Systeme 4) Beobachtungspunkte.
Erfinde keine neuen Messwerte.`, risks.Overall, risks.OverallScore, risks.RadioR, risks.RadiationS, risks.GeomagG, risks.WindIMF, risks.Aurora, synthBuilder.String())

	modelSynth := ""
	synthSource := "deterministic"
	if bodyBytes, err := marshalSpaceChatRequest(modelID, systemPkg, synthPrompt); err == nil {
		if text, err := invokeSpaceChatRaw(bodyBytes); err == nil && strings.TrimSpace(text) != "" {
			modelSynth = strings.TrimSpace(text)
			synthSource = "model"
		}
	}

	finalMD := BuildFinalReportMarkdown(snap, risks, intermediates, modelSynth)
	writeAnalysisSSE(w, flusher, "step_done", SpaceIntermediateReport{
		PackageID: "synthesis",
		Title:     "Finale Synthese",
		RiskLabel: risks.Overall,
		Body:      modelSynth,
		Source:    synthSource,
	})
	writeAnalysisSSE(w, flusher, "final", map[string]interface{}{
		"report":           finalMD,
		"risk_classes":     risks,
		"intermediates":    intermediates,
		"synthesis_source": synthSource,
	})
	fmt.Fprint(w, "data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}

func truncateSpaceErr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
