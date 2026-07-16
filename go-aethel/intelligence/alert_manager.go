package intelligence

import "time"

// AlertManager enforces spam prevention for proactive alerts (dedup, cooldown,
// min-confidence, min-sources). Lives in package intelligence to avoid a circular
// import with intelligence/alerts (which re-exports the same rules for external use).
type AlertManager struct {
	Cooldown      time.Duration
	MinConfidence int
	MinSources    int
}

// NewAlertManager returns the default local spam-prevention policy.
func NewAlertManager() *AlertManager {
	return &AlertManager{
		Cooldown:      30 * time.Minute,
		MinConfidence: 50,
		MinSources:    1,
	}
}

// CreateAlert applies full spam prevention and returns the alert or nil if suppressed.
func (m *AlertManager) CreateAlert(candidate Alert, existing []Alert, sourceCount int) *Alert {
	if m == nil {
		m = NewAlertManager()
	}
	if candidate.Confidence < m.MinConfidence {
		return nil
	}
	if sourceCount < m.MinSources {
		return nil
	}
	for _, ex := range existing {
		if ex.Region == candidate.Region && !ex.Acknowledged && time.Since(ex.CreatedAt) < m.Cooldown {
			// same region recent unacked — dedup
			return nil
		}
	}
	if candidate.Severity == "high" {
		candidate.EscalationState = "escalated"
	} else if candidate.EscalationState == "" {
		candidate.EscalationState = "new"
	}
	now := time.Now().UTC()
	candidate.CreatedAt = now
	candidate.ExpiresAt = now.Add(6 * time.Hour)
	return &candidate
}

// ClusterAlertsByRegion groups alerts for summary (simple, deterministic).
func ClusterAlertsByRegion(alerts []Alert) map[string][]Alert {
	out := make(map[string][]Alert)
	for _, a := range alerts {
		out[a.Region] = append(out[a.Region], a)
	}
	return out
}
