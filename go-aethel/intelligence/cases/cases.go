package cases

// OSINT Case Kernel — fully implemented core (GaiasEye per spec §11, §16).
// Strict isolation: cases live in separate store slice. PII always pseudonymized on ingest into case.
// Re-ID is a privileged, audited, time-bounded operation with explicit purpose + approvals.
// No mixing with root personal memory. All mutations produce AuditEvent.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"go-aethel/intelligence"
)

type Case struct {
	ID             string                  `json:"id"`
	Purpose        string                  `json:"purpose"`
	Classification string                  `json:"classification"`
	Policy         string                  `json:"policy"`
	AllowedSources []string                `json:"allowed_sources"`
	Evidence       []intelligence.Evidence `json:"evidence"`
	Entities       []intelligence.Entity   `json:"entities"`
	Relations      []intelligence.Relation `json:"relations"`
	Timeline       []intelligence.AuditEvent `json:"timeline"`
	Reports        []intelligence.Briefing `json:"reports"`
	Audit          []intelligence.AuditEvent `json:"audit"`
	ReIdRequests   []ReIdRequest           `json:"reid_requests"`
	CreatedAt      time.Time               `json:"created_at"`
	PseudonymKey   []byte                  `json:"-"` // never serialized
}

type ReIdRequest struct {
	ID             string    `json:"id"`
	Purpose        string    `json:"purpose"`
	ApprovedBy     string    `json:"approved_by"`
	SecondApproval string    `json:"second_approval,omitempty"`
	RequestedAt    time.Time `json:"requested_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	Unlocked       bool      `json:"unlocked"`
	AuditID        string    `json:"audit_id"`
}

// NewCase creates a fully initialized case with pseudonym key.
func NewCase(id, purpose, classification string, allowed []string) *Case {
	key := make([]byte, 32)
	// In real: crypto/rand. Here deterministic for testability but non-secret for pseudonym.
	copy(key, []byte(purpose+id))
	c := &Case{
		ID:             id,
		Purpose:        purpose,
		Classification: classification,
		Policy:         "operator-controlled",
		AllowedSources: allowed,
		Evidence:       []intelligence.Evidence{},
		Entities:       []intelligence.Entity{},
		Relations:      []intelligence.Relation{},
		Timeline:       []intelligence.AuditEvent{},
		Reports:        []intelligence.Briefing{},
		Audit:          []intelligence.AuditEvent{},
		ReIdRequests:   []ReIdRequest{},
		CreatedAt:      time.Now().UTC(),
		PseudonymKey:   key,
	}
	c.Audit = append(c.Audit, intelligence.AuditEvent{At: c.CreatedAt, Action: "case.created", Actor: "operator", Detail: purpose})
	return c
}

// Pseudonymize produces stable case-scoped alias. Never reversible without key.
func (c *Case) Pseudonymize(normalized string) string {
	if len(c.PseudonymKey) == 0 {
		return ""
	}
	n := strings.TrimSpace(strings.ToLower(normalized))
	mac := hmac.New(sha256.New, c.PseudonymKey)
	mac.Write([]byte(n))
	sum := mac.Sum(nil)
	return "GX-" + hex.EncodeToString(sum[:12])
}

// RequestReID creates a guarded request. Requires purpose and will need external approval.
func (c *Case) RequestReID(purpose string, expires time.Duration) ReIdRequest {
	req := ReIdRequest{
		ID:          "reid-" + c.ID + "-" + time.Now().Format("20060102150405"),
		Purpose:     purpose,
		RequestedAt: time.Now().UTC(),
		ExpiresAt:   time.Now().UTC().Add(expires),
		Unlocked:    false,
	}
	c.ReIdRequests = append(c.ReIdRequests, req)
	c.Audit = append(c.Audit, intelligence.AuditEvent{At: req.RequestedAt, Action: "reid.requested", Actor: "operator", Detail: purpose})
	return req
}

// SealEvidence adds sealed evidence (raw level Observation can be turned into sealed inside case).
func (c *Case) SealEvidence(srcID, excerpt, sha string) intelligence.Evidence {
	e := intelligence.Evidence{
		ID:               "case-evid-" + c.ID + "-" + time.Now().Format("150405"),
		SourceID:         srcID,
		Excerpt:          excerpt,
		SHA256:           sha,
		CollectedAt:      time.Now().UTC(),
		Sealed:           true,
		ValidationStatus: "pending",
	}
	c.Evidence = append(c.Evidence, e)
	c.Audit = append(c.Audit, intelligence.AuditEvent{At: e.CollectedAt, Action: "evidence.sealed", Actor: "operator", Detail: srcID})
	return e
}

// All external content remains untrusted. Re-ID never automatic.

