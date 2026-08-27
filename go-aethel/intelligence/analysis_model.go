package intelligence

// STATUS: DIAMANT VGT SUPREME

import "time"

// SourceDocument is the immutable acquisition envelope from which passages,
// observations and claims are derived. Raw bytes live in the content-addressed
// evidence vault; this record carries reproducible provenance only.
type SourceDocument struct {
	ID               string            `json:"id"`
	SourceID         string            `json:"source_id"`
	OriginalURL      string            `json:"original_url,omitempty"`
	FinalURL         string            `json:"final_url,omitempty"`
	Title            string            `json:"title,omitempty"`
	MIMEType         string            `json:"mime_type,omitempty"`
	Language         string            `json:"language,omitempty"`
	FetchedAt        time.Time         `json:"fetched_at"`
	PublishedAt      time.Time         `json:"published_at,omitempty"`
	RawSHA256        string            `json:"raw_sha256"`
	NormalizedSHA256 string            `json:"normalized_sha256,omitempty"`
	SnapshotID       string            `json:"snapshot_id,omitempty"`
	ParserVersion    string            `json:"parser_version"`
	ResponseHeaders  map[string]string `json:"response_headers,omitempty"`
	InstructionFlags []string          `json:"instruction_flags,omitempty"`
	Quarantined      bool              `json:"quarantined"`
}

// Passage binds extracted text to byte/rune offsets in one immutable document.
type Passage struct {
	ID          string `json:"id"`
	DocumentID  string `json:"document_id"`
	Text        string `json:"text"`
	StartOffset int    `json:"start_offset"`
	EndOffset   int    `json:"end_offset"`
}

// Claim is the atomic analytic unit. Supporting and contradicting evidence are
// stored separately so agreement can never erase contrary information.
type Claim struct {
	ID                       string    `json:"id"`
	CaseID                   string    `json:"case_id,omitempty"`
	Subject                  string    `json:"subject"`
	Predicate                string    `json:"predicate"`
	Object                   string    `json:"object"`
	Statement                string    `json:"statement"`
	Location                 string    `json:"location,omitempty"`
	OccurredAt               time.Time `json:"occurred_at,omitempty"`
	AssertingSourceID        string    `json:"asserting_source_id"`
	SourceNature             string    `json:"source_nature"`
	PassageIDs               []string  `json:"passage_ids,omitempty"`
	SupportingEvidenceIDs    []string  `json:"supporting_evidence_ids,omitempty"`
	ContradictingEvidenceIDs []string  `json:"contradicting_evidence_ids,omitempty"`
	IndependentSourceCount   int       `json:"independent_source_count"`
	Confidence               int       `json:"confidence"`
	CalibrationBasis         string    `json:"calibration_basis,omitempty"`
	Status                   string    `json:"status"`
	CreatedAt                time.Time `json:"created_at"`
	ReviewedAt               time.Time `json:"reviewed_at,omitempty"`
	ReviewedBy               string    `json:"reviewed_by,omitempty"`
}

// SourceLineage captures dependence between publications. A corroborating
// source counts as independent only when no active lineage edge links it to the
// same upstream origin.
type SourceLineage struct {
	ID               string    `json:"id"`
	UpstreamSource   string    `json:"upstream_source_id"`
	DownstreamSource string    `json:"downstream_source_id"`
	Relationship     string    `json:"relationship"`
	EvidenceIDs      []string  `json:"evidence_ids,omitempty"`
	Confidence       int       `json:"confidence"`
	DetectedBy       string    `json:"detected_by"`
	CreatedAt        time.Time `json:"created_at"`
	Reviewed         bool      `json:"reviewed"`
}

type HypothesisIndicator struct {
	ID            string   `json:"id"`
	Description   string   `json:"description"`
	Expected      bool     `json:"expected"`
	Observed      bool     `json:"observed"`
	Diagnosticity int      `json:"diagnosticity"`
	EvidenceIDs   []string `json:"evidence_ids,omitempty"`
}

type ConfidencePoint struct {
	At         time.Time `json:"at"`
	Confidence int       `json:"confidence"`
	Reason     string    `json:"reason"`
	Actor      string    `json:"actor"`
}

// Hypothesis supports explicit competing explanations and revision history.
type Hypothesis struct {
	ID                       string                `json:"id"`
	CaseID                   string                `json:"case_id"`
	Statement                string                `json:"statement"`
	AlternativeHypothesisIDs []string              `json:"alternative_hypothesis_ids,omitempty"`
	Indicators               []HypothesisIndicator `json:"indicators,omitempty"`
	SupportingEvidenceIDs    []string              `json:"supporting_evidence_ids,omitempty"`
	ContradictingEvidenceIDs []string              `json:"contradicting_evidence_ids,omitempty"`
	InformationGapIDs        []string              `json:"information_gap_ids,omitempty"`
	Confidence               int                   `json:"confidence"`
	ConfidenceHistory        []ConfidencePoint     `json:"confidence_history"`
	ChangeConditions         []string              `json:"change_conditions,omitempty"`
	Status                   string                `json:"status"`
	CreatedAt                time.Time             `json:"created_at"`
	UpdatedAt                time.Time             `json:"updated_at"`
}

type InformationGap struct {
	ID         string    `json:"id"`
	CaseID     string    `json:"case_id"`
	Question   string    `json:"question"`
	Priority   string    `json:"priority"`
	Rationale  string    `json:"rationale"`
	Status     string    `json:"status"`
	ResolvedBy []string  `json:"resolved_by_evidence_ids,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	ResolvedAt time.Time `json:"resolved_at,omitempty"`
}

type CollectionPlan struct {
	ID               string    `json:"id"`
	CaseID           string    `json:"case_id"`
	InformationGapID string    `json:"information_gap_id"`
	SourceTypes      []string  `json:"source_types"`
	Queries          []string  `json:"queries"`
	Constraints      []string  `json:"constraints,omitempty"`
	OwnerProfile     string    `json:"owner_profile"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CustodyEvent struct {
	ID           string    `json:"id"`
	EvidenceID   string    `json:"evidence_id"`
	Action       string    `json:"action"`
	Actor        string    `json:"actor"`
	Detail       string    `json:"detail,omitempty"`
	At           time.Time `json:"at"`
	PreviousHash string    `json:"previous_hash,omitempty"`
	EventHash    string    `json:"event_hash"`
}

type SnapshotRecord struct {
	ID              string            `json:"id"`
	SourceID        string            `json:"source_id"`
	OriginalURL     string            `json:"original_url,omitempty"`
	FinalURL        string            `json:"final_url,omitempty"`
	MIMEType        string            `json:"mime_type,omitempty"`
	FetchedAt       time.Time         `json:"fetched_at"`
	RawSHA256       string            `json:"raw_sha256"`
	SizeBytes       int64             `json:"size_bytes"`
	ResponseHeaders map[string]string `json:"response_headers,omitempty"`
	ObjectPath      string            `json:"object_path"`
	ManifestPath    string            `json:"manifest_path"`
}

type EntityAlias struct {
	Value      string    `json:"value"`
	Language   string    `json:"language,omitempty"`
	Script     string    `json:"script,omitempty"`
	ValidFrom  time.Time `json:"valid_from,omitempty"`
	ValidUntil time.Time `json:"valid_until,omitempty"`
}

type ResolutionSignal struct {
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

type EntityResolutionCandidate struct {
	ID            string             `json:"id"`
	CaseID        string             `json:"case_id"`
	LeftEntityID  string             `json:"left_entity_id"`
	RightEntityID string             `json:"right_entity_id"`
	Score         int                `json:"score"`
	Signals       []ResolutionSignal `json:"signals"`
	Reasons       []string           `json:"reasons"`
	Status        string             `json:"status"`
	CreatedAt     time.Time          `json:"created_at"`
	ReviewedAt    time.Time          `json:"reviewed_at,omitempty"`
	ReviewedBy    string             `json:"reviewed_by,omitempty"`
}

type EntityResolutionDecision struct {
	ID               string     `json:"id"`
	CandidateID      string     `json:"candidate_id"`
	CaseID           string     `json:"case_id"`
	Action           string     `json:"action"`
	Actor            string     `json:"actor"`
	Reason           string     `json:"reason"`
	BeforeClusterIDs [][]string `json:"before_cluster_ids"`
	AfterClusterIDs  [][]string `json:"after_cluster_ids"`
	At               time.Time  `json:"at"`
}

type ResolvedEntity struct {
	ID              string        `json:"id"`
	CaseID          string        `json:"case_id"`
	Kind            string        `json:"kind"`
	CanonicalLabel  string        `json:"canonical_label"`
	SourceEntityIDs []string      `json:"source_entity_ids"`
	Aliases         []EntityAlias `json:"aliases"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type EntityVersion struct {
	ID               string         `json:"id"`
	ResolvedEntityID string         `json:"resolved_entity_id"`
	CaseID           string         `json:"case_id"`
	Version          int            `json:"version"`
	Action           string         `json:"action"`
	Actor            string         `json:"actor"`
	Reason           string         `json:"reason"`
	Snapshot         ResolvedEntity `json:"snapshot"`
	At               time.Time      `json:"at"`
}

type HypothesisEvidenceAssessment struct {
	ID            string    `json:"id"`
	CaseID        string    `json:"case_id"`
	HypothesisID  string    `json:"hypothesis_id"`
	EvidenceID    string    `json:"evidence_id"`
	Compatibility int       `json:"compatibility"`
	Diagnosticity int       `json:"diagnosticity"`
	Reason        string    `json:"reason"`
	Actor         string    `json:"actor"`
	At            time.Time `json:"at"`
}

type ACHMatrixRow struct {
	HypothesisID       string                         `json:"hypothesis_id"`
	Statement          string                         `json:"statement"`
	Assessments        []HypothesisEvidenceAssessment `json:"assessments"`
	InconsistencyScore int                            `json:"inconsistency_score"`
	MissingEvidence    int                            `json:"missing_evidence"`
	Rank               int                            `json:"rank"`
}

type ACHMatrix struct {
	CaseID      string         `json:"case_id"`
	EvidenceIDs []string       `json:"evidence_ids"`
	Rows        []ACHMatrixRow `json:"rows"`
	GeneratedAt time.Time      `json:"generated_at"`
}
