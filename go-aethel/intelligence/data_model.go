package intelligence

import (
	"time"
)

// Source describes the origin of intelligence data
type Source struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	URL                string    `json:"url"`
	SourceType         string    `json:"source_type"` // rss, api, local, database
	Publisher          string    `json:"publisher"`
	TrustTier          int       `json:"trust_tier"` // 1=verified, 2=community, 3=unverified
	PermissionStatus   string    `json:"permission_status"`
	Region             string    `json:"region"`
	Language           string    `json:"language"`
	FetchedAt          time.Time `json:"fetched_at"`
	PublishedAt        time.Time `json:"published_at"`
	ParserVersion      string    `json:"parser_version"`
	ContentHash        string    `json:"content_hash"`
	Freshness          float64   `json:"freshness"`
	AvailabilityStatus string    `json:"availability_status"`
}

// Observation is raw data ingested directly from a source
type Observation struct {
	ID          string    `json:"id"`
	SourceID    string    `json:"source_id"`
	RawText     string    `json:"raw_text"`
	ObservedAt  time.Time `json:"observed_at"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	ContentHash string    `json:"content_hash"`
	Domain      string    `json:"domain"`
}

// Evidence represents sealed context inside a case
type Evidence struct {
	ID               string    `json:"id"`
	CaseID           string    `json:"case_id"`
	SourceID         string    `json:"source_id"`
	URL              string    `json:"url,omitempty"`
	Excerpt          string    `json:"excerpt"`
	SHA256           string    `json:"sha256"`
	CollectedAt      time.Time `json:"collected_at"`
	Sealed           bool      `json:"sealed"`
	ValidationStatus string    `json:"validation_status"` // pending, verified, disputed, rejected
	ChainOfCustodyID string    `json:"chain_of_custody_id"`
	SnapshotPath     string    `json:"snapshot_path,omitempty"`
}

// Entity maps actors, organisations, or locations (pseudonymized if personal)
type Entity struct {
	ID         string `json:"id"` // GX-PER-HASH for pseudonymized persons
	Label      string `json:"label"`
	Kind       string `json:"kind"`       // person, organisation, location, asset, event
	Confidence int    `json:"confidence"` // 0-100
}

// Relation is a semantic link between two entities
type Relation struct {
	FromEntity   string    `json:"from_entity_id"`
	ToEntity     string    `json:"to_entity_id"`
	RelationType string    `json:"relation_type"`
	EvidenceIDs  []string  `json:"evidence_ids"`
	Confidence   int       `json:"confidence"` // 0-100
	ValidFrom    time.Time `json:"valid_from"`
	ValidUntil   time.Time `json:"valid_until"`
	Verified     bool      `json:"verified"`
}

// Event is a classified geopol, cyber, eco or humanitarian occurrence
type Event struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Summary    string    `json:"summary"`
	Domain     string    `json:"domain"` // geo, cyber, economic, humanitarian, general
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Severity   string    `json:"severity"`   // low, medium, high
	Confidence int       `json:"confidence"` // 0-100
	ObservedAt time.Time `json:"observed_at"`
}

// Region is a bounding polygon for risk scoring
type Region struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`     // state, city, economic, custom
	Polygons [][]Point `json:"polygons"` // array of coordinate rings
}

// Point helper coordinate
type Point struct {
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
}

// Signal represents an indicator of change
type Signal struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	SourceID  string    `json:"source_id"`
	Timestamp time.Time `json:"timestamp"`
}

// Assessment is an interpretation by LLM, operator or rules
type Assessment struct {
	ID                    string    `json:"id"`
	Statement             string    `json:"statement"`
	Confidence            int       `json:"confidence"`
	EvidenceIDs           []string  `json:"evidence_ids"`
	GeneratedBy           string    `json:"generated_by"` // model, operator, rule
	ModelVersion          string    `json:"model_version,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	Status                string    `json:"status"` // hypothesis, unverified, corroborated, verified, disputed, rejected
	ContradictingEvidence []string  `json:"contradicting_evidence,omitempty"`
	ReviewedBy            string    `json:"reviewed_by,omitempty"`
}

// RiskScore aggregates evaluated regional risks
type RiskScore struct {
	OverallRisk            float64   `json:"overall_risk"`
	GeopoliticalRisk       float64   `json:"geopolitical_risk"`
	ConflictRisk           float64   `json:"conflict_risk"`
	CyberRisk              float64   `json:"cyber_risk"`
	InfrastructureRisk     float64   `json:"infrastructure_risk"`
	EconomicRisk           float64   `json:"economic_risk"`
	FinancialRisk          float64   `json:"financial_risk"`
	EnergyRisk             float64   `json:"energy_risk"`
	SupplyChainRisk        float64   `json:"supply_chain_risk"`
	ClimateRisk            float64   `json:"climate_risk"`
	PublicSafetyRisk       float64   `json:"public_safety_risk"`
	InformationReliability float64   `json:"information_reliability"`
	DataFreshness          float64   `json:"data_freshness"`
	Confidence             int       `json:"confidence"`
	Trend                  string    `json:"trend"` // up, stable, down
	LastUpdated            time.Time `json:"last_updated"`
	PrimaryDrivers         []string  `json:"primary_drivers"`
	MissingData            []string  `json:"missing_data,omitempty"`
}

// Alert is a prioritized operator notification
type Alert struct {
	ID              string             `json:"id"`
	Severity        string             `json:"severity"` // low, medium, high
	Confidence      int                `json:"confidence"`
	Region          string             `json:"region"`
	AffectedDomains []string           `json:"affected_domains"`
	EvidenceIDs     []string           `json:"evidence_ids"`
	Reason          string             `json:"reason"`
	CreatedAt       time.Time          `json:"created_at"`
	ExpiresAt       time.Time          `json:"expires_at"`
	Acknowledged    bool               `json:"acknowledged"`
	EscalationState string             `json:"escalation_state"`
	AIAssessment    *AlertAIAssessment `json:"ai_assessment,omitempty"`
}

// AlertAIAssessment is an explicitly unverified model interpretation layered
// on top of the deterministic alert. It never overwrites rule-derived fields.
type AlertAIAssessment struct {
	ModelID            string    `json:"model_id"`
	Language           string    `json:"language"`
	Severity           string    `json:"severity"`
	Confidence         int       `json:"confidence"`
	Summary            string    `json:"summary"`
	Rationale          string    `json:"rationale"`
	Uncertainties      []string  `json:"uncertainties"`
	RecommendedActions []string  `json:"recommended_actions"`
	EvaluatedAt        time.Time `json:"evaluated_at"`
	Status             string    `json:"status"`
}

// Briefing defines a generated report structure
type Briefing struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	ReportType     string    `json:"report_type"`
	Content        string    `json:"content"` // Markdown report content
	CreatedAt      time.Time `json:"created_at"`
	SourceCount    int       `json:"source_count"`
	AlertsIncluded []string  `json:"alerts_included,omitempty"`
}

// Case isolates target context and audits
type Case struct {
	ID             string       `json:"id"`
	Title          string       `json:"title"`
	Purpose        string       `json:"purpose"`
	Classification string       `json:"classification"`
	AllowedSources []string     `json:"allowed_sources"`
	RetentionRules string       `json:"retention_rules,omitempty"`
	PseudonymKey   []byte       `json:"-"`
	Evidence       []Evidence   `json:"evidence"`
	Entities       []Entity     `json:"entities"`
	Relations      []Relation   `json:"relations"`
	Audit          []AuditEvent `json:"audit"`
}

// Watchlist defines operator-tracked targets
type Watchlist struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	RegionIDs []string `json:"region_ids"`
	Keywords  []string `json:"keywords"`
}

// AlertRule is an operator-defined proactive threshold on regional risk (shared model).
// CreatedAt and CreatedBy are assigned by Store.AddAlertRule — names must match.
type AlertRule struct {
	ID             string    `json:"id"`
	RegionID       string    `json:"region_id"`
	MinSeverity    string    `json:"min_severity"` // low|medium|high
	MinOverallRisk float64   `json:"min_overall_risk"`
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      string    `json:"created_by"`
}

// AuditEvent logs operator actions
type AuditEvent struct {
	At     time.Time `json:"at"`
	Action string    `json:"action"`
	Actor  string    `json:"actor"`
	Detail string    `json:"detail"`
}

// AgentAction logs controlled autonomous or semi-autonomous actions (with approval trail)
type AgentAction struct {
	ID           string    `json:"id"`
	Skill        string    `json:"skill"`
	Args         string    `json:"args"` // JSON
	Result       string    `json:"result,omitempty"`
	ApprovedBy   string    `json:"approved_by,omitempty"`
	ExecutedAt   time.Time `json:"executed_at"`
	Status       string    `json:"status"` // requested, approved, completed, rejected, failed
	AuditEventID string    `json:"audit_event_id,omitempty"`
}

// PersonalContext holds operator-declared interests (separate trust class from World State).
type PersonalContext struct {
	OperatorID         string    `json:"operator_id"`
	Interests          []string  `json:"interests"`
	Projects           []string  `json:"projects"`
	PreferredRegions   []string  `json:"preferred_regions"`
	WatchlistIDs       []string  `json:"watchlist_ids"`
	RiskTolerance      string    `json:"risk_tolerance"` // low, medium, high
	Goals              []string  `json:"goals"`
	LastUpdated        time.Time `json:"last_updated"`
	CommunicationStyle string    `json:"communication_style,omitempty"`
	LocationCity       string    `json:"location_city,omitempty"`
	LocationCountry    string    `json:"location_country,omitempty"`
}

// WorldStateSnapshot is a derived view (Events/Risks/Alerts). Correlation at query time.
type WorldStateSnapshot struct {
	AsOf          time.Time            `json:"as_of"`
	ActiveRegions []string             `json:"active_regions"`
	RiskOverview  map[string]RiskScore `json:"risk_overview"`
	RecentEvents  []Event              `json:"recent_events"`
	Alerts        []Alert              `json:"alerts"`
	Notes         string               `json:"notes"`
}

// IdentityProfile is technical continuity (not consciousness).
type IdentityProfile struct {
	Name            string    `json:"name"`
	Version         string    `json:"version"`
	LastUpdated     time.Time `json:"last_updated"`
	LastWarnedAt    time.Time `json:"last_warned_at,omitempty"`
	PendingAlertIDs []string  `json:"pending_alert_ids,omitempty"`
	UnresolvedQs    []string  `json:"unresolved_questions,omitempty"`
	CapabilityNotes []string  `json:"capability_notes,omitempty"`
	DiagnosticsNote string    `json:"diagnostics_note,omitempty"`
}

// Relocated from osint_engine.go
type OSINTDomain string

const (
	DomainGeo          OSINTDomain = "geo"
	DomainCyber        OSINTDomain = "cyber"
	DomainEconomic     OSINTDomain = "economic"
	DomainHumanitarian OSINTDomain = "humanitarian"
	DomainGeneral      OSINTDomain = "general"
)

type OSINTEvent struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	Summary    string      `json:"summary"`
	Source     string      `json:"source"`
	SourceURL  string      `json:"source_url"`
	Domain     OSINTDomain `json:"domain"`
	Lat        float64     `json:"lat"`
	Lon        float64     `json:"lon"`
	HasGeo     bool        `json:"has_geo"`
	Timestamp  time.Time   `json:"timestamp"`
	Confidence float64     `json:"confidence"` // 0.0–1.0
	Status     string      `json:"status"`     // raw, proposed, verified, disputed, alert
	URL        string      `json:"url"`
}
