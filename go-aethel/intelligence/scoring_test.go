package intelligence

import (
	"testing"
	"time"
)

func TestComputeRiskScore(t *testing.T) {
	se := NewScoringEngine(0.02)
	
	now := time.Now().UTC()
	events := []Event{
		{
			ID:         "ev1",
			Title:      "Cyberattack on Grid",
			Domain:     "cyber",
			Severity:   "high",
			Confidence: 90,
			ObservedAt: now,
		},
		{
			ID:         "ev2",
			Title:      "Local Strike",
			Domain:     "general", // infrastructure
			Severity:   "medium",
			Confidence: 80,
			ObservedAt: now.Add(-10 * time.Hour), // 10 hours ago
		},
	}
	
	score := se.ComputeRiskScore(events, now)
	
	// Cyber Risk calculation: Weight 12 * Severity 2.5 * Conf 0.9 * Freshness 1.0 = 27.0
	if score.CyberRisk != 27.0 {
		t.Errorf("Expected CyberRisk to be 27.0, got %v", score.CyberRisk)
	}
	
	if score.OverallRisk <= 0 {
		t.Errorf("Expected OverallRisk to be greater than 0, got %v", score.OverallRisk)
	}
	
	if len(score.PrimaryDrivers) == 0 {
		t.Error("Expected primary drivers to be populated for high-severity fresh event")
	}
}
