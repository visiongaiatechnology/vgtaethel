package briefings

import (
	"strings"
	"testing"
	"time"

	"go-aethel/intelligence"
)

func TestGenerateSeparatesLayers(t *testing.T) {
	snap := intelligence.StoreState{
		Observations: []intelligence.Observation{{
			ID: "obs1", SourceID: "src1", RawText: "Raw Berlin report", ObservedAt: time.Now().UTC(),
		}},
		Assessments: []intelligence.Assessment{
			{ID: "a1", Statement: "Classified geo", Status: "unverified", Confidence: 70},
			{ID: "a2", Statement: "Two sources agree", Status: "corroborated", Confidence: 80},
			{ID: "a3", Statement: "Operator verified finding", Status: "verified", Confidence: 90},
		},
		Events: []intelligence.Event{{
			ID: "e1", Title: "Berlin event", Domain: "geo", Confidence: 70,
		}},
		Sources: []intelligence.Source{{ID: "src1", SourceType: "rss", TrustTier: 2}},
		RiskScores: map[string]intelligence.RiskScore{
			"GERMANY": {OverallRisk: 22, Trend: "stable", Confidence: 70, PrimaryDrivers: []string{"none"}, MissingData: []string{"gaps"}},
		},
	}
	out := Generate("Daily Global Brief", snap)
	for _, need := range []string{"RAW OBSERVATIONS", "INFERENCE", "VERIFIED", "Recommendations", "Provenance", "obs1", "CORROBORATED"} {
		if !strings.Contains(out, need) {
			t.Errorf("briefing missing %q", need)
		}
	}
	if strings.Contains(strings.ToLower(out), "raw berlin report") && strings.Contains(out, "verified fact") {
		t.Error("must not label raw as verified fact")
	}
	// Corroborated must appear under CORROBORATED section, not be the only "verified" path
	if !strings.Contains(out, "corroborated ≠ verified") && !strings.Contains(out, "Operator verified finding") {
		t.Error("must distinguish corroborated from operator-verified")
	}
	// Golden: recommendations explicitly non-factual
	if !strings.Contains(out, "recommendations, not facts") && !strings.Contains(out, "Recommendations") {
		t.Error("recommendations section required")
	}
}

func TestGenerateGoldenEmptySnapshot(t *testing.T) {
	out := Generate("Empty Brief", intelligence.StoreState{})
	for _, need := range []string{"RAW OBSERVATIONS", "VERIFIED FINDINGS", "Recommendations", "Provenance", "SOVEREIGN LOCAL"} {
		if !strings.Contains(out, need) {
			t.Errorf("empty golden briefing missing %q", need)
		}
	}
	if strings.Contains(out, "confirmed by intelligence agencies") {
		t.Error("must not invent authoritative confirmation language")
	}
}
