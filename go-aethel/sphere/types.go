// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/sphere/types.go
// Purpose: Universal Sphere Object Model definitions, data contracts, and validation

package sphere

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ObjectType defines the universal taxonomy of Sphere entities.
type ObjectType string

const (
	TypeDocument     ObjectType = "document"
	TypeTrip         ObjectType = "trip"
	TypePlan         ObjectType = "plan"
	TypeTask         ObjectType = "task"
	TypeProject      ObjectType = "project"
	TypeResearchItem ObjectType = "research_item"
	TypePlace        ObjectType = "place"
	TypePerson       ObjectType = "person"
	TypeSource       ObjectType = "source"
	TypeEvent        ObjectType = "event"
	TypeAlert        ObjectType = "alert"
	TypeFile         ObjectType = "file"
	TypeBooking      ObjectType = "booking"
	TypeConversation ObjectType = "conversation"
	TypeRun          ObjectType = "run"
)

// VirtualDesktop represents one of the isolated virtual desktop environments in Sphere.
type VirtualDesktop string

const (
	DesktopPersonal VirtualDesktop = "PERSONAL"
	DesktopWork     VirtualDesktop = "WORK"
	DesktopResearch VirtualDesktop = "RESEARCH"
	DesktopTravel   VirtualDesktop = "TRAVEL"
	DesktopProject  VirtualDesktop = "PROJECT"
	DesktopIncident VirtualDesktop = "INCIDENT"
)

// SphereObject is the universal container for any first-class Sphere entity.
type SphereObject struct {
	ID        string                 `json:"id"`
	Type      ObjectType             `json:"type"`
	Title     string                 `json:"title"`
	Summary   string                 `json:"summary,omitempty"`
	Desktop   VirtualDesktop         `json:"desktop,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Data      json.RawMessage        `json:"data"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Version   int                    `json:"version"`
	LinkedIDs []string               `json:"linked_ids,omitempty"`
	SourceApp string                 `json:"source_app,omitempty"`
}

func (o *SphereObject) Validate() error {
	if strings.TrimSpace(o.ID) == "" {
		return errors.New("sphere object ID cannot be empty")
	}
	if strings.TrimSpace(string(o.Type)) == "" {
		return errors.New("sphere object type cannot be empty")
	}
	if strings.TrimSpace(o.Title) == "" {
		return errors.New("sphere object title cannot be empty")
	}
	if len(o.Title) > 300 {
		return errors.New("sphere object title exceeds 300 characters")
	}
	return nil
}

// DocumentObject models rich documents with versioning and track changes in VGT Writer 2.0.
type DocumentObject struct {
	ID           string                `json:"id"`
	Title        string                `json:"title"`
	ContentHTML  string                `json:"content_html"`
	ContentMD    string                `json:"content_md,omitempty"`
	WordCount    int                   `json:"word_count"`
	CharCount    int                   `json:"char_count"`
	ReadTimeMin  int                   `json:"read_time_min"`
	Author       string                `json:"author,omitempty"`
	LastModified time.Time             `json:"last_modified"`
	Versions     []DocumentVersion     `json:"versions,omitempty"`
	PendingDiffs []TrackChangeProposal `json:"pending_diffs,omitempty"`
	TemplateID   string                `json:"template_id,omitempty"`
	LinkedTripID string                `json:"linked_trip_id,omitempty"`
	LinkedPlanID string                `json:"linked_plan_id,omitempty"`
}

type DocumentVersion struct {
	VersionID   string    `json:"version_id"`
	Timestamp   time.Time `json:"timestamp"`
	Summary     string    `json:"summary"`
	ContentHTML string    `json:"content_html"`
	Author      string    `json:"author"`
}

type TrackChangeProposal struct {
	DiffID       string    `json:"diff_id"`
	ProposedBy   string    `json:"proposed_by"` // e.g. "AETHEL // AI Core"
	Timestamp    time.Time `json:"timestamp"`
	Explanation  string    `json:"explanation"`
	OriginalText string    `json:"original_text"`
	ProposedText string    `json:"proposed_text"`
	Status       string    `json:"status"` // "pending", "accepted", "rejected"
}

// TripObject models full persistent trips with Global Watch intelligence bridge.
type TripObject struct {
	ID               string              `json:"id"`
	Title            string              `json:"title"`        // e.g. "Reise Japan 2027"
	Destination      string              `json:"destination"`  // e.g. "Tokyo, Japan"
	Destinations     []string            `json:"destinations,omitempty"`
	StartDate        string              `json:"start_date"`   // YYYY-MM-DD
	EndDate          string              `json:"end_date"`     // YYYY-MM-DD
	DurationDays     int                 `json:"duration_days"`
	BudgetTotalEUR   float64             `json:"budget_total_eur"`
	BudgetSpentEUR   float64             `json:"budget_spent_eur"`
	Currency         string              `json:"currency"`     // EUR, USD, JPY, etc.
	Status           string              `json:"status"`       // "planning", "booked", "active", "completed", "cancelled"
	ItineraryDays    []TripDayItinerary  `json:"itinerary_days,omitempty"`
	LodgingOptions   []TripLodging       `json:"lodging_options,omitempty"`
	FlightOptions    []TripFlight        `json:"flight_options,omitempty"`
	Checklist        []TripChecklistItem `json:"checklist,omitempty"`
	TravelDocuments  []TripDocument      `json:"travel_documents,omitempty"`
	GlobalWatchRisks []GlobalWatchImpact `json:"global_watch_risks,omitempty"`
	WeatherSummary   string              `json:"weather_summary,omitempty"`
	Notes            string              `json:"notes,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

type TripDayItinerary struct {
	DayNumber  int        `json:"day_number"`
	Date       string     `json:"date"` // YYYY-MM-DD
	Location   string     `json:"location"`
	Title      string     `json:"title"`
	Highlights []string   `json:"highlights,omitempty"`
	Schedule   []TripSlot `json:"schedule,omitempty"`
	BudgetEst  float64    `json:"budget_est,omitempty"`
}

type TripSlot struct {
	Time       string  `json:"time"` // "09:00"
	Activity   string  `json:"activity"`
	Location   string  `json:"location,omitempty"`
	CostEUR    float64 `json:"cost_eur,omitempty"`
	BookingRef string  `json:"booking_ref,omitempty"`
	Confirmed  bool    `json:"confirmed"`
}

type TripLodging struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Location      string  `json:"location"`
	PricePerNight float64 `json:"price_per_night"`
	TotalPrice    float64 `json:"total_price"`
	Rating        float64 `json:"rating,omitempty"`
	Status        string  `json:"status"` // "candidate", "reserved", "booked"
	URL           string  `json:"url,omitempty"`
	Notes         string  `json:"notes,omitempty"`
}

type TripFlight struct {
	ID        string  `json:"id"`
	Airline   string  `json:"airline"`
	FlightNo  string  `json:"flight_no,omitempty"`
	Departure string  `json:"departure"` // "FRA 14:00"
	Arrival   string  `json:"arrival"`   // "HND 08:30+1"
	Date      string  `json:"date"`
	PriceEUR  float64 `json:"price_eur"`
	IsDirect  bool    `json:"is_direct"`
	Status    string  `json:"status"` // "candidate", "booked"
	URL       string  `json:"url,omitempty"`
}

type TripChecklistItem struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
	Category  string `json:"category,omitempty"` // "documents", "packing", "booking", "health"
}

type TripDocument struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Type     string `json:"type"` // "passport", "visa", "ticket", "insurance", "booking_confirmation"
	FilePath string `json:"file_path,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

type GlobalWatchImpact struct {
	AlertID         string    `json:"alert_id"`
	Region          string    `json:"region"`
	Signal          string    `json:"signal"`
	Confidence      string    `json:"confidence"` // "HIGH", "MEDIUM", "LOW"
	PotentialImpact string    `json:"potential_impact"`
	AffectedDates   string    `json:"affected_dates"`
	SourceCount     int       `json:"source_count"`
	Severity        string    `json:"severity"` // "NORMAL", "WATCH", "ELEVATED", "CRITICAL"
	DetectedAt      time.Time `json:"detected_at"`
	Dismissed       bool      `json:"dismissed"`
}

// PlanObject models structured plans with goals, milestones, dependencies, and agent runs.
type PlanObject struct {
	ID           string           `json:"id"`
	Title        string           `json:"title"`
	Objective    string           `json:"objective"`
	TargetDate   string           `json:"target_date,omitempty"`
	Status       string           `json:"status"` // "draft", "active", "in_progress", "completed", "paused"
	Milestones   []PlanMilestone  `json:"milestones,omitempty"`
	Tasks        []PlanTask       `json:"tasks,omitempty"`
	Dependencies []PlanDependency `json:"dependencies,omitempty"`
	LinkedRunIDs []string         `json:"linked_run_ids,omitempty"`
	ProgressPct  int              `json:"progress_pct"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

type PlanMilestone struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
	Completed   bool   `json:"completed"`
	Order       int    `json:"order"`
}

type PlanTask struct {
	ID           string   `json:"id"`
	MilestoneID  string   `json:"milestone_id,omitempty"`
	Title        string   `json:"title"`
	Status       string   `json:"status"` // "pending", "in_progress", "blocked", "completed"
	Assignee     string   `json:"assignee,omitempty"` // e.g. "AETHEL // Agent", "OPERATOR"
	LinkedRunID  string   `json:"linked_run_id,omitempty"`
	DependsOnIDs []string `json:"depends_on_ids,omitempty"`
}

type PlanDependency struct {
	FromTaskID string `json:"from_task_id"`
	ToTaskID   string `json:"to_task_id"`
	Type       string `json:"type"` // "blocks", "relates_to"
}

// ResearchItem models evidence clippings, citations, and notes collected into Research Projects.
type ResearchItem struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"project_id"`
	Title         string    `json:"title"`
	SourceURL     string    `json:"source_url,omitempty"`
	SourceTitle   string    `json:"source_title,omitempty"`
	ExtractedText string    `json:"extracted_text"`
	KeyTakeaway   string    `json:"key_takeaway,omitempty"`
	Provenance    string    `json:"provenance,omitempty"` // e.g. "SHADOW Collector", "Aethel Browser", "Global Watch"
	CapturedAt    time.Time `json:"captured_at"`
	Tags          []string  `json:"tags,omitempty"`
	Confidence    string    `json:"confidence,omitempty"`
}

type ResearchProject struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ItemCount   int       `json:"item_count"`
	Tags        []string  `json:"tags,omitempty"`
	Status      string    `json:"status"` // "active", "archived", "briefing_ready"
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BrowserTabState models persistent browser sessions and tabs in Sphere Browser 2.0.
type BrowserTabState struct {
	TabID         string    `json:"tab_id"`
	URL           string    `json:"url"`
	Title         string    `json:"title"`
	Favicon       string    `json:"favicon,omitempty"`
	IsActive      bool      `json:"is_active"`
	IsLoading     bool      `json:"is_loading"`
	AIControlling bool      `json:"ai_controlling"`
	LastUpdated   time.Time `json:"last_updated"`
}

// DesktopWindowState models persistent window geometry and states per virtual desktop.
type DesktopWindowState struct {
	AppID       string `json:"app_id"`
	Desktop     string `json:"desktop"`
	Left        int    `json:"left"`
	Top         int    `json:"top"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	ZIndex      int    `json:"z_index"`
	IsMinimized bool   `json:"is_minimized"`
	IsMaximized bool   `json:"is_maximized"`
	IsOpen      bool   `json:"is_open"`
}

type DesktopWorkspaceState struct {
	ActiveDesktop VirtualDesktop                `json:"active_desktop"`
	AmbientLevel  int                           `json:"ambient_level"`
	Windows       map[string]DesktopWindowState `json:"windows"`
	LastSaved     time.Time                     `json:"last_saved"`
}

// SendToPayload models the cross-app entity transfer contract.
type SendToPayload struct {
	SourceApp  string                 `json:"source_app"`
	TargetApp  string                 `json:"target_app"`
	ObjectType ObjectType             `json:"object_type"`
	ObjectID   string                 `json:"object_id,omitempty"`
	Title      string                 `json:"title"`
	Content    string                 `json:"content"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

func (p *SendToPayload) Validate() error {
	if strings.TrimSpace(p.SourceApp) == "" {
		return errors.New("source_app is required")
	}
	if strings.TrimSpace(p.TargetApp) == "" {
		return errors.New("target_app is required")
	}
	if strings.TrimSpace(p.Title) == "" && strings.TrimSpace(p.Content) == "" {
		return errors.New("title or content is required")
	}
	return nil
}

// GenerateID produces a typed, collision-resistant identifier.
func GenerateID(prefix string) string {
	ts := time.Now().UTC().Format("20060102-150405")
	nano := time.Now().UTC().UnixNano() % 1000000
	return fmt.Sprintf("%s-%s-%06d", strings.ToUpper(prefix), ts, nano)
}
