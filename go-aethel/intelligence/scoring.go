package intelligence

import (
	"math"
	"time"
)

// ScoringEngine computes regional risk matrices using freshness decay and weights
type ScoringEngine struct {
	decayLambda float64 // decay rate per hour
}

func NewScoringEngine(decayLambda float64) *ScoringEngine {
	if decayLambda <= 0 {
		decayLambda = 0.02 // half-life ~35 hours default
	}
	return &ScoringEngine{decayLambda: decayLambda}
}

// ComputeRiskScore processes a list of events and outputs risk aggregates
func (se *ScoringEngine) ComputeRiskScore(events []Event, lastUpdated time.Time) RiskScore {
	score := RiskScore{
		LastUpdated: time.Now().UTC(),
	}
	
	if len(events) == 0 {
		return score
	}

	var geopotRisk, conflictRisk, cyberRisk, infraRisk, ecoRisk float64
	var geopotCount, conflictCount, cyberCount, infraCount, ecoCount int
	
	drivers := []string{}
	now := time.Now().UTC()

	for _, ev := range events {
		dt := now.Sub(ev.ObservedAt).Hours()
		if dt < 0 {
			dt = 0
		}
		
		// Freshness decay factor: F = e^(-lambda * dt)
		freshness := math.Exp(-se.decayLambda * dt)
		
		// Severity factor: Low=1.0, Medium=1.5, High=2.5
		sevMultiplier := 1.0
		switch ev.Severity {
		case "medium":
			sevMultiplier = 1.5
		case "high":
			sevMultiplier = 2.5
		}
		
		// Confidence multiplier (0.0 to 1.0)
		confMultiplier := float64(ev.Confidence) / 100.0
		if confMultiplier <= 0 {
			confMultiplier = 0.5
		}
		
		// Event score contributions
		weight := 10.0
		
		switch ev.Domain {
		case "geo":
			weight = 10.0
			geopotRisk += weight * sevMultiplier * freshness * confMultiplier
			geopotCount++
			if ev.Severity == "high" && freshness > 0.5 {
				drivers = append(drivers, "Geopolitical: "+ev.Title)
			}
		case "cyber":
			weight = 12.0
			cyberRisk += weight * sevMultiplier * freshness * confMultiplier
			cyberCount++
			if ev.Severity == "high" && freshness > 0.5 {
				drivers = append(drivers, "Cyber: "+ev.Title)
			}
		case "economic":
			weight = 8.0
			ecoRisk += weight * sevMultiplier * freshness * confMultiplier
			ecoCount++
			if ev.Severity == "high" && freshness > 0.5 {
				drivers = append(drivers, "Economic: "+ev.Title)
			}
		case "humanitarian":
			weight = 15.0
			conflictRisk += weight * sevMultiplier * freshness * confMultiplier
			conflictCount++
			if ev.Severity == "high" && freshness > 0.5 {
				drivers = append(drivers, "Conflict: "+ev.Title)
			}
		default: // infrastructure / general
			weight = 10.0
			infraRisk += weight * sevMultiplier * freshness * confMultiplier
			infraCount++
			if ev.Severity == "high" && freshness > 0.5 {
				drivers = append(drivers, "Infrastructure: "+ev.Title)
			}
		}
	}

	score.GeopoliticalRisk = math.Min(100.0, geopotRisk)
	score.ConflictRisk = math.Min(100.0, conflictRisk)
	score.CyberRisk = math.Min(100.0, cyberRisk)
	score.InfrastructureRisk = math.Min(100.0, infraRisk)
	score.EconomicRisk = math.Min(100.0, ecoRisk)
	
	// Fill additional spec dimensions (conservative propagation from existing signals)
	score.FinancialRisk = math.Min(100.0, score.EconomicRisk*0.85)
	score.EnergyRisk = math.Min(100.0, score.InfrastructureRisk*0.65)
	score.SupplyChainRisk = math.Min(100.0, score.EconomicRisk*0.75+score.InfrastructureRisk*0.3)
	score.ClimateRisk = math.Min(100.0, 8.0) // baseline; overridden on climate signals in ingest classify
	score.PublicSafetyRisk = math.Min(100.0, score.ConflictRisk*0.55)
	score.InformationReliability = 72.0 // default until source tiers implemented
	score.DataFreshness = 78.0

	// Aggregated overall score bias formula: Bias towards highest risk dimension
	maxRisk := score.GeopoliticalRisk
	if score.ConflictRisk > maxRisk {
		maxRisk = score.ConflictRisk
	}
	if score.CyberRisk > maxRisk {
		maxRisk = score.CyberRisk
	}
	if score.InfrastructureRisk > maxRisk {
		maxRisk = score.InfrastructureRisk
	}
	if score.EconomicRisk > maxRisk {
		maxRisk = score.EconomicRisk
	}
	if score.EnergyRisk > maxRisk {
		maxRisk = score.EnergyRisk
	}
	
	avgRisk := (score.GeopoliticalRisk + score.ConflictRisk + score.CyberRisk + score.InfrastructureRisk + score.EconomicRisk +
		score.EnergyRisk + score.SupplyChainRisk) / 7.0
	
	// S_overall = 0.6 * max(S_dim) + 0.4 * avg(S_dim)
	score.OverallRisk = math.Min(100.0, 0.6*maxRisk+0.4*avgRisk)
	score.Confidence = 82
	score.PrimaryDrivers = drivers
	if len(score.PrimaryDrivers) == 0 {
		score.PrimaryDrivers = []string{"No high-severity recent signals in region"}
	}
	score.MissingData = []string{"Limited source diversity", "No operator-provided signals yet"}
	
	// Default trend; Ingest overrides based on risk delta when prior score exists.
	score.Trend = "stable"
	
	return score
}
