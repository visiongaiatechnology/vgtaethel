package handlers

// STATUS: DIAMANT VGT SUPREME

import "testing"

func TestTextOnlyRequestNeverAttachesViewportScreenshot(t *testing.T) {
	messages := []map[string]interface{}{
		{
			"role":    "user",
			"content": `SHADOW_BATCH_JSON=[{"title":"Video shows conflict escalation","summary":"Browser report"}]`,
		},
	}

	if shouldAttachViewportScreenshot(ChatRequest{TextOnly: true}, "deepseek/deepseek-chat", messages) {
		t.Fatal("text-only SHADOW analysis attempted to attach a viewport screenshot")
	}
	if !shouldAttachViewportScreenshot(ChatRequest{}, "deepseek/deepseek-chat", messages) {
		t.Fatal("regression fixture does not exercise the automatic vision trigger")
	}
}

func TestDecodeShadowModelReportExtractsFencedJSONAfterReasoning(t *testing.T) {
	content := "<think>Analyse abgeschlossen.</think>\n```json\n" +
		`{"threat_level":"HIGH","summary":"Evidence-bound summary with enough content for report identification.","situation":"Situation","cui_bono":"Interests","strategic_reality":"Projection","divergences":"None","confirmed_vectors":"None","regions":[],"conflict_links":[],"forecast_matrix":[],"evidence_ids":["item-1"]}` + "\n```"
	report, err := decodeShadowModelReport(content)
	if err != nil {
		t.Fatal(err)
	}
	if report.ThreatLevel != "HIGH" || len(report.EvidenceIDs) != 1 {
		t.Fatalf("unexpected extracted report: %+v", report)
	}
}

func TestDecodeShadowModelReportRejectsTruncatedJSON(t *testing.T) {
	if _, err := decodeShadowModelReport(`{"threat_level":"HIGH"`); err == nil {
		t.Fatal("truncated model JSON was accepted")
	}
}
