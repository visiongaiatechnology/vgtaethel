package geoint

import (
	"time"
)

// GeoEntityType classifies the domain of a geospatial entity
type GeoEntityType string

const (
	TypeAircraft             GeoEntityType = "AIRCRAFT"
	TypeMilAircraft          GeoEntityType = "MIL_AIRCRAFT"
	TypeVessel               GeoEntityType = "VESSEL"
	TypeSatellite            GeoEntityType = "SATELLITE"
	TypeCamera               GeoEntityType = "CAMERA"
	TypeFire                 GeoEntityType = "FIRE"
	TypeEarthquake           GeoEntityType = "EARTHQUAKE"
	TypeVolcano              GeoEntityType = "VOLCANO"
	TypeInfrastructure       GeoEntityType = "INFRASTRUCTURE"
	TypeMilitaryInstallation GeoEntityType = "MILITARY_INSTALLATION"
	TypeConflict             GeoEntityType = "CONFLICT"
	TypeCyber                GeoEntityType = "CYBER"
	TypePolitical            GeoEntityType = "POLITICAL"
	TypeEconomic             GeoEntityType = "ECONOMIC"
	TypeEnergy               GeoEntityType = "ENERGY"
	TypeSpaceEvent           GeoEntityType = "SPACE_EVENT"
	TypeNewsEvent            GeoEntityType = "NEWS_EVENT"
	TypeCustom               GeoEntityType = "CUSTOM"
)

type GeoAssessmentStatus string

const (
	AssessmentRaw          GeoAssessmentStatus = "RAW"
	AssessmentObserved     GeoAssessmentStatus = "OBSERVED"
	AssessmentCorrelated   GeoAssessmentStatus = "CORRELATED"
	AssessmentAssessed     GeoAssessmentStatus = "ASSESSED"
	AssessmentSupported    GeoAssessmentStatus = "SUPPORTED"
	AssessmentConfirmed    GeoAssessmentStatus = "CONFIRMED"
	AssessmentContradicted GeoAssessmentStatus = "CONTRADICTED"
	AssessmentRetracted    GeoAssessmentStatus = "RETRACTED"
)

type GeoLocationPrecision string

const (
	PrecisionExact       GeoLocationPrecision = "EXACT"
	PrecisionCity        GeoLocationPrecision = "CITY"
	PrecisionRegion      GeoLocationPrecision = "REGION"
	PrecisionCountry     GeoLocationPrecision = "COUNTRY"
	PrecisionApproximate GeoLocationPrecision = "APPROXIMATE"
	PrecisionUnknown     GeoLocationPrecision = "UNKNOWN"
)

type GeoRelationCategory string

const (
	RelationKinetic  GeoRelationCategory = "KINETIC"
	RelationCyber    GeoRelationCategory = "CYBER"
	RelationEconomic GeoRelationCategory = "ECONOMIC"
	RelationSupport  GeoRelationCategory = "SUPPORT"
	RelationMaritime GeoRelationCategory = "MARITIME"
	RelationAir      GeoRelationCategory = "AIR"
	RelationSpace    GeoRelationCategory = "SPACE"
)

// GeoPosition holds spatial coordinates and kinematics
type GeoPosition struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	Altitude  float64 `json:"alt,omitempty"`        // Meters above sea level
	Heading   float64 `json:"heading,omitempty"`    // 0-360 degrees
	Velocity  float64 `json:"velocity,omitempty"`   // m/s ground speed
	ClimbRate float64 `json:"climb_rate,omitempty"` // m/s vertical rate
	Course    float64 `json:"course,omitempty"`     // Track / course
}

// GeoEntity is the universal entity model in Aethel GEOINT
type GeoEntity struct {
	ID             string               `json:"id"`
	Type           GeoEntityType        `json:"type"`
	Source         string               `json:"source"`
	Label          string               `json:"label"`
	Position       GeoPosition          `json:"position"`
	Timestamp      time.Time            `json:"timestamp"`
	Freshness      float64              `json:"freshness"`  // 0.0 to 1.0
	Confidence     int                  `json:"confidence"` // 0 to 100
	Classification string               `json:"classification,omitempty"`
	Metadata       map[string]any       `json:"metadata,omitempty"`
	Trail          []GeoPosition        `json:"trail,omitempty"`
	Evidence       []string             `json:"evidence,omitempty"`
	SourceIDs      []string             `json:"source_ids,omitempty"`
	Assessment     GeoAssessmentStatus  `json:"assessment_status,omitempty"`
	Location       GeoLocationPrecision `json:"location_precision,omitempty"`
	ExpiresAt      time.Time            `json:"expires_at,omitempty"`
}

// GeoRelation is an evidence-bound directed relationship. It is kept separate
// from point entities so a renderer never has to infer direction or attribution.
type GeoRelation struct {
	ID                string               `json:"id"`
	ReportID          string               `json:"report_id"`
	OriginName        string               `json:"origin_name"`
	TargetName        string               `json:"target_name"`
	Origin            GeoPosition          `json:"origin"`
	Target            GeoPosition          `json:"target"`
	OriginPrecision   GeoLocationPrecision `json:"origin_precision"`
	TargetPrecision   GeoLocationPrecision `json:"target_precision"`
	Action            string               `json:"action"`
	Category          GeoRelationCategory  `json:"category"`
	Timestamp         time.Time            `json:"timestamp"`
	Freshness         float64              `json:"freshness"`
	Confidence        int                  `json:"confidence"`
	EvidenceIDs       []string             `json:"evidence_ids"`
	SourceIDs         []string             `json:"source_ids,omitempty"`
	AssessmentStatus  GeoAssessmentStatus  `json:"assessment_status"`
	AttributionStatus string               `json:"attribution_status,omitempty"`
	Assessment        string               `json:"assessment"`
}

// CorrelatedEvent represents cross-modal fusion of multiple entity observations
type CorrelatedEvent struct {
	ID                    string         `json:"id"`
	Title                 string         `json:"title"`
	Summary               string         `json:"summary"`
	CenterLat             float64        `json:"center_lat"`
	CenterLon             float64        `json:"center_lon"`
	Confidence            float64        `json:"confidence"` // 0.0 - 1.0
	TemporalWindowMinutes float64        `json:"temporal_window_minutes"`
	PrimaryType           GeoEntityType  `json:"primary_type"`
	EntityIDs             []string       `json:"entity_ids"`
	Evidence              []string       `json:"evidence"`
	Assessment            string         `json:"assessment"`
	CreatedAt             time.Time      `json:"created_at"`
	Metadata              map[string]any `json:"metadata,omitempty"`
}

// ContactItem is a localized contact relative to an observer or focal point
type ContactItem struct {
	EntityID       string         `json:"entity_id"`
	Type           GeoEntityType  `json:"type"`
	Label          string         `json:"label"`
	Classification string         `json:"classification"`
	DistanceKm     float64        `json:"distance_km"`
	BearingDeg     float64        `json:"bearing_deg"`
	Position       GeoPosition    `json:"position"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// ContactRoster represents a localized 250km+ contact breakdown
type ContactRoster struct {
	CenterLat      float64       `json:"center_lat"`
	CenterLon      float64       `json:"center_lon"`
	RadiusKm       float64       `json:"radius_km"`
	TotalContacts  int           `json:"total_contacts"`
	AirContacts    []ContactItem `json:"air_contacts"`
	MilAirContacts []ContactItem `json:"mil_air_contacts"`
	SeaContacts    []ContactItem `json:"sea_contacts"`
	SpaceContacts  []ContactItem `json:"space_contacts"`
	CameraContacts []ContactItem `json:"camera_contacts"`
	HazardContacts []ContactItem `json:"hazard_contacts"`
	OsintContacts  []ContactItem `json:"osint_contacts"`
	Timestamp      time.Time     `json:"timestamp"`
}

// CollectorStatus captures the operational health of a GEOINT collector
type CollectorStatus struct {
	Name         string    `json:"name"`
	Enabled      bool      `json:"enabled"`
	ActiveCount  int       `json:"active_count"`
	LastRefresh  time.Time `json:"last_refresh"`
	LastError    string    `json:"last_error,omitempty"`
	SourceStatus string    `json:"source_status"` // OK, DEGRADED, RATE_LIMITED, OFFLINE
}

// GeoIntSystemStatus gives an overview of the entire GEOINT engine
type GeoIntSystemStatus struct {
	TotalEntities  int                        `json:"total_entities"`
	TypeCounts     map[GeoEntityType]int      `json:"type_counts"`
	Collectors     map[string]CollectorStatus `json:"collectors"`
	Correlations   int                        `json:"correlations"`
	BusSubscribers int                        `json:"bus_subscribers"`
	LastUpdated    time.Time                  `json:"last_updated"`
	SnapshotRevision uint64                   `json:"snapshot_revision"`
	ShadowRelations  int                      `json:"shadow_relations"`
	ShadowSyncAt     time.Time                `json:"shadow_sync_at,omitempty"`
}
