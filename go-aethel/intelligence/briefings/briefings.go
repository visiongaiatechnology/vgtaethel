package briefings

import (
	"fmt"
	"strings"
	"time"

	"go-aethel/intelligence"
)

// STATUS: PLATIN
// Structured report generator per spec §7.
// Separates RAW observation / INFERENCE / VERIFIED; labels recommendations explicitly.

// Generate builds a markdown briefing from a StoreState snapshot (pure function, testable).
func Generate(reportType string, snap intelligence.StoreState) string {
	var b strings.Builder
	now := time.Now().UTC()
	rt := strings.TrimSpace(reportType)
	if rt == "" {
		rt = "Daily Global Brief"
	}
	b.WriteString(fmt.Sprintf("# AETHEL %s\n\n", rt))
	b.WriteString(fmt.Sprintf("Generated: %s (UTC)\n", now.Format(time.RFC3339)))
	b.WriteString("**SOVEREIGN LOCAL REPORT — NO EXTERNAL TRANSMISSION**\n\n")

	b.WriteString("## 1. Executive Summary\n")
	b.WriteString(fmt.Sprintf("Raw observations: %d | Classified events: %d | Evidence: %d | Risk regions: %d | Alerts: %d\n\n",
		len(snap.Observations), len(snap.Events), len(snap.Evidence), len(snap.RiskScores), len(snap.Alerts)))

	b.WriteString("## Layer separation\n")
	b.WriteString("### RAW OBSERVATIONS (unreviewed source claims)\n")
	n := 0
	for i := len(snap.Observations) - 1; i >= 0 && n < 8; i-- {
		o := snap.Observations[i]
		excerpt := o.RawText
		if len(excerpt) > 120 {
			excerpt = excerpt[:120]
		}
		b.WriteString(fmt.Sprintf("- [%s] src=%s — %s\n", o.ID, o.SourceID, excerpt))
		n++
	}
	if n == 0 {
		b.WriteString("- (none)\n")
	}

	b.WriteString("\n### INFERENCE / ASSESSMENTS (status never auto-verified)\n")
	for i, a := range snap.Assessments {
		if i >= 8 {
			break
		}
		b.WriteString(fmt.Sprintf("- [%s] status=%s conf=%d%% — %s\n", a.ID, a.Status, a.Confidence, a.Statement))
	}
	if len(snap.Assessments) == 0 {
		b.WriteString("- (none)\n")
	}

	b.WriteString("\n### CORROBORATED (multi-source inference — still not operator-verified)\n")
	cN := 0
	for _, a := range snap.Assessments {
		if a.Status == "corroborated" {
			b.WriteString(fmt.Sprintf("- [%s] status=%s conf=%d%% — %s\n", a.ID, a.Status, a.Confidence, a.Statement))
			cN++
		}
	}
	if cN == 0 {
		b.WriteString("- (none)\n")
	}

	b.WriteString("\n### VERIFIED FINDINGS (operator/evidence review only)\n")
	v := 0
	for _, a := range snap.Assessments {
		if a.Status == "verified" {
			b.WriteString(fmt.Sprintf("- [%s] status=%s — %s\n", a.ID, a.Status, a.Statement))
			v++
		}
	}
	if v == 0 {
		b.WriteString("- none (operator/evidence review required; corroborated ≠ verified)\n")
	}

	b.WriteString("\n## Regional risk (multi-dimensional)\n")
	for id, rs := range snap.RiskScores {
		b.WriteString(fmt.Sprintf("- **%s** Overall=%.1f%% trend=%s conf=%d%% drivers=%v missing=%v\n",
			id, rs.OverallRisk, rs.Trend, rs.Confidence, rs.PrimaryDrivers, rs.MissingData))
	}
	if len(snap.RiskScores) == 0 {
		b.WriteString("- No scores computed yet.\n")
	}

	b.WriteString("\n## Uncertainties\n")
	b.WriteString("- Classifications are rule-engine inference, not confirmed facts.\n")
	b.WriteString("- Missing multi-source corroboration may understate or overstate risk.\n")
	b.WriteString("- Scores use defaults for dimensions without signals.\n")

	b.WriteString("\n## Sources & Evidence\n")
	for i, s := range snap.Sources {
		if i >= 6 {
			break
		}
		b.WriteString(fmt.Sprintf("- %s type=%s trust=%d\n", s.ID, s.SourceType, s.TrustTier))
	}
	for i, e := range snap.Evidence {
		if i >= 6 {
			break
		}
		ex := e.Excerpt
		if len(ex) > 80 {
			ex = ex[:80]
		}
		b.WriteString(fmt.Sprintf("- Evidence %s status=%s — %s\n", e.ID, e.ValidationStatus, ex))
	}

	b.WriteString("\n## Recommendations\n")
	b.WriteString("**These are recommendations, not facts.**\n")
	b.WriteString("- Monitor regions with rising risk deltas.\n")
	b.WriteString("- Review primary sources for high-confidence events.\n")
	b.WriteString("- Link personal watchlists to regions of interest.\n")

	b.WriteString("\n---\n**Provenance:** Snapshot-only generation; no LLM-invented sources. Observation strictly separated from Assessment/Risk.\n")
	return b.String()
}
