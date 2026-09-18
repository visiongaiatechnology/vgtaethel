// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/sphere/travel/travel_engine.go
// Purpose: Trip Planning, Itinerary Generation, Budgeting, and Global Watch Intelligence Correlation

package travel

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-aethel/sphere"
)

// Engine orchestrates trip logic and Global Watch intelligence correlation.
type Engine struct {
	store *sphere.ObjectStore
}

func NewEngine(store *sphere.ObjectStore) *Engine {
	return &Engine{store: store}
}

// CreateTrip initializes and persists a new TripObject.
func (e *Engine) CreateTrip(trip *sphere.TripObject) (*sphere.TripObject, error) {
	if trip == nil {
		return nil, errors.New("trip object cannot be nil")
	}
	trip.Title = strings.TrimSpace(trip.Title)
	trip.Destination = strings.TrimSpace(trip.Destination)
	if trip.Title == "" {
		return nil, errors.New("trip title is required")
	}
	if trip.Destination == "" {
		return nil, errors.New("trip destination is required")
	}

	if trip.ID == "" {
		trip.ID = sphere.GenerateID("TRIP")
	}
	if trip.Status == "" {
		trip.Status = "planning"
	}
	if trip.Currency == "" {
		trip.Currency = "EUR"
	}
	now := time.Now().UTC()
	trip.CreatedAt = now
	trip.UpdatedAt = now

	// Recalculate duration if dates are given
	if trip.StartDate != "" && trip.EndDate != "" {
		if s, err1 := time.Parse("2006-01-02", trip.StartDate); err1 == nil {
			if end, err2 := time.Parse("2006-01-02", trip.EndDate); err2 == nil && end.After(s) {
				trip.DurationDays = int(end.Sub(s).Hours()/24) + 1
			}
		}
	}

	// Persist as Universal SphereObject
	data, err := json.Marshal(trip)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal trip data: %w", err)
	}

	obj := &sphere.SphereObject{
		ID:        trip.ID,
		Type:      sphere.TypeTrip,
		Title:     trip.Title,
		Summary:   fmt.Sprintf("%s · %d Tage · %s %.2f", trip.Destination, trip.DurationDays, trip.Currency, trip.BudgetTotalEUR),
		Desktop:   sphere.DesktopTravel,
		Tags:      append([]string{"travel", "trip", strings.ToLower(trip.Destination)}, trip.Destinations...),
		Data:      data,
		CreatedAt: now,
		UpdatedAt: now,
		SourceApp: "travel",
	}

	if err := e.store.Put(obj); err != nil {
		return nil, fmt.Errorf("failed to persist trip: %w", err)
	}
	return trip, nil
}

// GetTrip loads a TripObject by ID.
func (e *Engine) GetTrip(id string) (*sphere.TripObject, error) {
	obj, exists := e.store.Get(id)
	if !exists || obj.Type != sphere.TypeTrip {
		return nil, fmt.Errorf("trip %s not found", id)
	}
	var trip sphere.TripObject
	if err := json.Unmarshal(obj.Data, &trip); err != nil {
		return nil, fmt.Errorf("failed to deserialize trip data: %w", err)
	}
	return &trip, nil
}

// ListTrips loads all trips.
func (e *Engine) ListTrips() ([]*sphere.TripObject, error) {
	objects := e.store.List(sphere.TypeTrip, "", "")
	trips := make([]*sphere.TripObject, 0, len(objects))
	for _, obj := range objects {
		var trip sphere.TripObject
		if err := json.Unmarshal(obj.Data, &trip); err == nil {
			trips = append(trips, &trip)
		}
	}
	return trips, nil
}

// UpdateTrip updates trip attributes, recalculates spent budgets, and persists.
func (e *Engine) UpdateTrip(trip *sphere.TripObject) error {
	if trip == nil || trip.ID == "" {
		return errors.New("invalid trip object")
	}
	trip.UpdatedAt = time.Now().UTC()

	// Calculate spent budget from flights and lodgings
	var spent float64
	for _, f := range trip.FlightOptions {
		if f.Status == "booked" {
			spent += f.PriceEUR
		}
	}
	for _, l := range trip.LodgingOptions {
		if l.Status == "booked" || l.Status == "reserved" {
			spent += l.TotalPrice
		}
	}
	for _, day := range trip.ItineraryDays {
		for _, slot := range day.Schedule {
			if slot.Confirmed {
				spent += slot.CostEUR
			}
		}
	}
	trip.BudgetSpentEUR = spent

	data, err := json.Marshal(trip)
	if err != nil {
		return fmt.Errorf("failed to marshal trip data: %w", err)
	}

	obj, exists := e.store.Get(trip.ID)
	if !exists {
		obj = &sphere.SphereObject{
			ID:        trip.ID,
			Type:      sphere.TypeTrip,
			CreatedAt: trip.CreatedAt,
			Desktop:   sphere.DesktopTravel,
			SourceApp: "travel",
		}
	}
	obj.Title = trip.Title
	obj.Summary = fmt.Sprintf("%s · %d Tage · %s %.2f", trip.Destination, trip.DurationDays, trip.Currency, trip.BudgetTotalEUR)
	obj.Tags = append([]string{"travel", "trip", strings.ToLower(trip.Destination)}, trip.Destinations...)
	obj.Data = data
	obj.UpdatedAt = trip.UpdatedAt

	return e.store.Put(obj)
}

// DeleteTrip removes a trip by ID.
func (e *Engine) DeleteTrip(id string) (bool, error) {
	return e.store.Delete(id)
}

// CorrelateGlobalWatchIntelligence checks active trips against given Global Watch risk events.
func (e *Engine) CorrelateGlobalWatchIntelligence(tripID string, regionName string, signal string, severity string, confidence string) (*sphere.GlobalWatchImpact, error) {
	trip, err := e.GetTrip(tripID)
	if err != nil {
		return nil, err
	}

	// Match region against destination and sub-destinations
	isMatch := false
	destinations := append([]string{trip.Destination}, trip.Destinations...)
	for _, d := range destinations {
		if strings.Contains(strings.ToLower(d), strings.ToLower(regionName)) ||
			strings.Contains(strings.ToLower(regionName), strings.ToLower(d)) {
			isMatch = true
			break
		}
	}

	if !isMatch {
		return nil, nil // No geographical relevance
	}

	impact := sphere.GlobalWatchImpact{
		AlertID:         fmt.Sprintf("ALERT-%d", time.Now().UnixNano()%1000000),
		Region:          regionName,
		Signal:          signal,
		Confidence:      confidence,
		PotentialImpact: "Mögliche Flugverspätungen oder Einschränkungen am Reiseziel",
		AffectedDates:   fmt.Sprintf("%s – %s", trip.StartDate, trip.EndDate),
		SourceCount:     3,
		Severity:        severity,
		DetectedAt:      time.Now().UTC(),
		Dismissed:      false,
	}

	// Add impact to trip object and save
	trip.GlobalWatchRisks = append(trip.GlobalWatchRisks, impact)
	if err := e.UpdateTrip(trip); err != nil {
		return nil, err
	}
	return &impact, nil
}
