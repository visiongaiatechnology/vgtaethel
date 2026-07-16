package alerts

import (
	"go-aethel/intelligence"
)

// Alerts package — thin façade over intelligence.AlertManager.
// All create paths should use intelligence.NewAlertManager().CreateAlert
// (or this package's NewManager). Zero external deps.

// Manager is an alias for the in-package spam-prevention manager.
type Manager = intelligence.AlertManager

// NewManager returns the default local spam-prevention policy.
func NewManager() *Manager {
	return intelligence.NewAlertManager()
}

// CreateAlert applies full spam prevention (delegates to intelligence.AlertManager).
func CreateAlert(m *Manager, candidate intelligence.Alert, existing []intelligence.Alert, sourceCount int) *intelligence.Alert {
	if m == nil {
		m = NewManager()
	}
	return m.CreateAlert(candidate, existing, sourceCount)
}

// ClusterByRegion groups alerts for summary (simple, deterministic).
func ClusterByRegion(alerts []intelligence.Alert) map[string][]intelligence.Alert {
	return intelligence.ClusterAlertsByRegion(alerts)
}
