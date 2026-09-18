// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/sphere/service.go
// Purpose: Master Sphere 2.0 Service Orchestrator uniting all sub-engines, Context Bus, and Global Watch Bridge

package sphere

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go-aethel/security"
)

// Service is the central orchestrator for Aethel Sphere 2.0.
type Service struct {
	mu          sync.RWMutex
	store       *ObjectStore
	workspaceDir string
}

// NewService initializes the Sphere 2.0 service.
func NewService(workspaceDir string) (*Service, error) {
	if strings.TrimSpace(workspaceDir) == "" {
		workspaceDir = security.WorkspaceDir
	}
	sphereStoreDir := fmt.Sprintf("%s/sphere", workspaceDir)
	store, err := NewObjectStore(sphereStoreDir)
	if err != nil {
		return nil, fmt.Errorf("failed to init sphere object store: %w", err)
	}

	svc := &Service{
		store:        store,
		workspaceDir: workspaceDir,
	}

	security.LogKernelActivity("SPHERE_SERVICE_INIT", "VGT Sphere 2.0 Kernel Active", "SUCCESS")
	return svc, nil
}

// GetStore exposes the underlying ObjectStore.
func (s *Service) GetStore() *ObjectStore {
	return s.store
}

// ResolveContext inspects active entities, trips, documents, and research items to construct a rich contextual picture for the operator or agent.
func (s *Service) ResolveContext(focusedApp string, query string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := map[string]interface{}{
		"active_desktop": s.store.GetWorkspaceState().ActiveDesktop,
		"focused_app":    focusedApp,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}

	// Fetch active trips
	trips := s.store.List(TypeTrip, "", "")
	if len(trips) > 0 {
		tripSummaries := make([]map[string]interface{}, 0, len(trips))
		for _, t := range trips {
			var tripObj TripObject
			if err := json.Unmarshal(t.Data, &tripObj); err == nil {
				tripSummaries = append(tripSummaries, map[string]interface{}{
					"id":          tripObj.ID,
					"title":       tripObj.Title,
					"destination": tripObj.Destination,
					"dates":       fmt.Sprintf("%s – %s", tripObj.StartDate, tripObj.EndDate),
					"status":      tripObj.Status,
					"risks_count": len(tripObj.GlobalWatchRisks),
				})
			}
		}
		result["trips"] = tripSummaries
	}

	// Fetch recent documents
	docs := s.store.List(TypeDocument, "", "")
	if len(docs) > 0 {
		docSummaries := make([]map[string]interface{}, 0, len(docs))
		for _, d := range docs {
			docSummaries = append(docSummaries, map[string]interface{}{
				"id":            d.ID,
				"title":         d.Title,
				"summary":       d.Summary,
				"last_modified": d.UpdatedAt,
			})
		}
		result["documents"] = docSummaries
	}

	// Fetch active plans
	plans := s.store.List(TypePlan, "", "")
	if len(plans) > 0 {
		planSummaries := make([]map[string]interface{}, 0, len(plans))
		for _, p := range plans {
			planSummaries = append(planSummaries, map[string]interface{}{
				"id":      p.ID,
				"title":   p.Title,
				"summary": p.Summary,
			})
		}
		result["plans"] = planSummaries
	}

	// If query is provided, execute fuzzy search
	if strings.TrimSpace(query) != "" {
		matches := s.store.Search(query, 10)
		result["search_matches"] = matches
	}

	return result, nil
}

// DispatchSendTo executes a universal cross-app entity transfer.
func (s *Service) DispatchSendTo(payload *SendToPayload) (*SphereObject, error) {
	if payload == nil {
		return nil, errors.New("nil send_to payload")
	}
	if err := payload.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	targetApp := strings.ToLower(strings.TrimSpace(payload.TargetApp))

	switch targetApp {
	case "writer":
		// Create or append to a document
		contentHTML := payload.Content
		if !strings.HasPrefix(contentHTML, "<") {
			contentHTML = fmt.Sprintf("<h2>%s</h2><p>%s</p>", payload.Title, payload.Content)
		}
		words, chars, readTime := len(strings.Fields(payload.Content)), len([]rune(payload.Content)), 1
		doc := &DocumentObject{
			ID:           GenerateID("DOC"),
			Title:        payload.Title,
			ContentHTML:  contentHTML,
			WordCount:    words,
			CharCount:    chars,
			ReadTimeMin:  readTime,
			Author:       fmt.Sprintf("Transferred from %s", payload.SourceApp),
			LastModified: now,
			Versions:     make([]DocumentVersion, 0),
		}
		data, _ := json.Marshal(doc)
		obj := &SphereObject{
			ID:        doc.ID,
			Type:      TypeDocument,
			Title:     doc.Title,
			Summary:   fmt.Sprintf("Empfangen von %s", payload.SourceApp),
			Desktop:   DesktopWork,
			Tags:      []string{"writer", "transferred", strings.ToLower(payload.SourceApp)},
			Data:      data,
			CreatedAt: now,
			UpdatedAt: now,
			SourceApp: payload.SourceApp,
		}
		if err := s.store.Put(obj); err != nil {
			return nil, err
		}
		security.LogKernelActivity("SPHERE_SEND_TO", fmt.Sprintf("%s -> writer (%s)", payload.SourceApp, doc.ID), "SUCCESS")
		return obj, nil

	case "research":
		// Add as research item
		item := &ResearchItem{
			ID:            GenerateID("CLIP"),
			Title:         payload.Title,
			ExtractedText: payload.Content,
			Provenance:    payload.SourceApp,
			CapturedAt:    now,
			Confidence:    "HIGH",
		}
		if url, ok := payload.Metadata["url"].(string); ok {
			item.SourceURL = url
		}
		if srcTitle, ok := payload.Metadata["source_title"].(string); ok {
			item.SourceTitle = srcTitle
		}
		data, _ := json.Marshal(item)
		obj := &SphereObject{
			ID:        item.ID,
			Type:      TypeResearchItem,
			Title:     item.Title,
			Summary:   fmt.Sprintf("Auszug von %s", payload.SourceApp),
			Desktop:   DesktopResearch,
			Tags:      []string{"clip", "research", strings.ToLower(payload.SourceApp)},
			Data:      data,
			CreatedAt: now,
			UpdatedAt: now,
			SourceApp: payload.SourceApp,
		}
		if err := s.store.Put(obj); err != nil {
			return nil, err
		}
		security.LogKernelActivity("SPHERE_SEND_TO", fmt.Sprintf("%s -> research (%s)", payload.SourceApp, item.ID), "SUCCESS")
		return obj, nil

	case "trip", "travel":
		// If object_id is a trip, add item to trip notes or documents
		if payload.ObjectID != "" {
			if tripObj, exists := s.store.Get(payload.ObjectID); exists && tripObj.Type == TypeTrip {
				var trip TripObject
				if err := json.Unmarshal(tripObj.Data, &trip); err == nil {
					trip.Notes = fmt.Sprintf("%s\n\n[Aus %s]: %s\n%s", trip.Notes, payload.SourceApp, payload.Title, payload.Content)
					trip.UpdatedAt = now
					tData, _ := json.Marshal(trip)
					tripObj.Data = tData
					tripObj.UpdatedAt = now
					_ = s.store.Put(tripObj)
					return tripObj, nil
				}
			}
		}
		// Otherwise create a candidate trip
		trip := &TripObject{
			ID:           GenerateID("TRIP"),
			Title:        payload.Title,
			Destination:  payload.Title,
			Status:       "planning",
			Notes:        payload.Content,
			Currency:     "EUR",
			DurationDays: 5,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		tData, _ := json.Marshal(trip)
		obj := &SphereObject{
			ID:        trip.ID,
			Type:      TypeTrip,
			Title:     trip.Title,
			Summary:   fmt.Sprintf("%s (Entwurf aus %s)", trip.Title, payload.SourceApp),
			Desktop:   DesktopTravel,
			Tags:      []string{"travel", "trip", "draft"},
			Data:      tData,
			CreatedAt: now,
			UpdatedAt: now,
			SourceApp: payload.SourceApp,
		}
		if err := s.store.Put(obj); err != nil {
			return nil, err
		}
		return obj, nil

	case "task", "planner":
		// Create a task or milestone
		plan := &PlanObject{
			ID:          GenerateID("PLAN"),
			Title:       payload.Title,
			Objective:   payload.Content,
			Status:      "active",
			ProgressPct: 0,
			Tasks: []PlanTask{
				{
					ID:       GenerateID("TASK"),
					Title:    payload.Title,
					Status:   "pending",
					Assignee: "OPERATOR",
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		}
		pData, _ := json.Marshal(plan)
		obj := &SphereObject{
			ID:        plan.ID,
			Type:      TypePlan,
			Title:     plan.Title,
			Summary:   fmt.Sprintf("Plan erstellt aus %s", payload.SourceApp),
			Desktop:   DesktopWork,
			Tags:      []string{"plan", "task", strings.ToLower(payload.SourceApp)},
			Data:      pData,
			CreatedAt: now,
			UpdatedAt: now,
			SourceApp: payload.SourceApp,
		}
		if err := s.store.Put(obj); err != nil {
			return nil, err
		}
		return obj, nil

	default:
		return nil, fmt.Errorf("unsupported target application: %s", targetApp)
	}
}
