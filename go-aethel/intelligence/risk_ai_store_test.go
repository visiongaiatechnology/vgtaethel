package intelligence

import (
	"strings"
	"testing"
	"time"
)

func TestAIRegionalRisksFresh_TTL(t *testing.T) {
	st := NewStore(t.TempDir()+"/ai_risk.json", nil)
	fresh, _ := st.AIRegionalRisksFresh(RegionalAIRiskTTL)
	if fresh {
		t.Fatal("empty store must not be fresh")
	}
	now := time.Now().UTC()
	st.ApplyAIRegionalRiskScores(map[string]RiskScore{
		"GERMANY": {
			OverallRisk: 33, EvaluationSource: "ai",
			AIEvaluatedAt: now, NextRefreshAt: now.Add(RegionalAIRiskTTL),
		},
	})
	fresh, at := st.AIRegionalRisksFresh(RegionalAIRiskTTL)
	if !fresh || at.IsZero() {
		t.Fatalf("just applied AI scores must be fresh, got fresh=%v at=%v", fresh, at)
	}
	// Expired AI
	st.ApplyAIRegionalRiskScores(map[string]RiskScore{
		"GERMANY": {
			OverallRisk: 33, EvaluationSource: "ai",
			AIEvaluatedAt: now.Add(-6 * time.Hour),
			NextRefreshAt: now.Add(-1 * time.Hour),
		},
	})
	fresh, _ = st.AIRegionalRisksFresh(RegionalAIRiskTTL)
	if fresh {
		t.Fatal("6h-old evaluation must not be fresh under 5h TTL")
	}
	// Deterministic must never count as AI-fresh (allows retry when Groq/DeepSeek become available)
	st.ApplyAIRegionalRiskScores(map[string]RiskScore{
		"GERMANY": {
			OverallRisk: 10, EvaluationSource: "deterministic",
			AIEvaluatedAt: now, NextRefreshAt: now.Add(RegionalAIRiskTTL),
		},
	})
	fresh, _ = st.AIRegionalRisksFresh(RegionalAIRiskTTL)
	if fresh {
		t.Fatal("deterministic fallback must not be treated as AI-fresh")
	}
}

func TestApplyAIRegionalRiskScores_ExplainIncludesNarrative(t *testing.T) {
	st := NewStore(t.TempDir()+"/ai_risk2.json", nil)
	st.ApplyAIRegionalRiskScores(map[string]RiskScore{
		"GERMANY": {
			OverallRisk: 55, EvaluationSource: "ai", AINarrative: "Spannungen steigen.",
			AIModelID: "test-model", AIEvaluatedAt: time.Now().UTC(),
			PrimaryDrivers: []string{"Cyber: probe"},
		},
	})
	ex := st.ExplainScore("GERMANY")
	if !strings.Contains(ex, "KI-Lagebewertung") || !strings.Contains(ex, "Spannungen steigen") {
		t.Fatalf("explain missing AI narrative: %s", ex)
	}
	if !strings.Contains(ex, "ai") && !strings.Contains(ex, "test-model") {
		t.Fatalf("explain missing source/model: %s", ex)
	}
}
