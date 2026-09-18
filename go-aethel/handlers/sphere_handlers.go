// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/handlers/sphere_handlers.go
// Purpose: HTTP Handlers for Sphere 2.0 Desktop Workspace, Universal Object Store, Context Bus, and Cross-App Workflows

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"go-aethel/security"
	"go-aethel/sphere"
	"go-aethel/sphere/browser"
	"go-aethel/sphere/documents"
	"go-aethel/sphere/planner"
	"go-aethel/sphere/research"
	"go-aethel/sphere/travel"
)

var (
	sphereSvcMu      sync.RWMutex
	globalSphereSvc  *sphere.Service
	globalTravelEng  *travel.Engine
	globalDocEng     *documents.Engine
	globalPlanEng    *planner.Engine
	globalResEng     *research.Engine
	globalBrowserMgr *browser.SessionManager
)

// InitSphereServices initializes all backend Sphere engines.
func InitSphereServices(workspaceDir string) (*sphere.Service, error) {
	sphereSvcMu.Lock()
	defer sphereSvcMu.Unlock()

	svc, err := sphere.NewService(workspaceDir)
	if err != nil {
		return nil, err
	}
	globalSphereSvc = svc
	store := svc.GetStore()
	globalTravelEng = travel.NewEngine(store)
	globalDocEng = documents.NewEngine(store)
	globalPlanEng = planner.NewEngine(store)
	globalResEng = research.NewEngine(store)
	globalBrowserMgr = browser.NewSessionManager()

	return svc, nil
}

func getSphereService() *sphere.Service {
	sphereSvcMu.RLock()
	svc := globalSphereSvc
	sphereSvcMu.RUnlock()
	if svc == nil {
		var err error
		svc, err = InitSphereServices(security.WorkspaceDir)
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize Sphere service: %v", err))
		}
	}
	return svc
}

// HandleSphereState handles GET and POST for desktop layout, virtual desktops, and window coordinates.
func HandleSphereState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	svc := getSphereService()
	store := svc.GetStore()

	if r.Method == http.MethodGet {
		state := store.GetWorkspaceState()
		_ = json.NewEncoder(w).Encode(state)
		return
	}

	if r.Method == http.MethodPost {
		var incoming sphere.DesktopWorkspaceState
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&incoming); err != nil {
			http.Error(w, `{"error":"Invalid workspace state JSON"}`, http.StatusBadRequest)
			return
		}

		if err := store.SaveWorkspaceState(incoming); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Failed to save state: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}

	http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
}

// HandleSphereObjects provides Universal Object Store CRUD.
func HandleSphereObjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	svc := getSphereService()
	store := svc.GetStore()

	switch r.Method {
	case http.MethodGet:
		// Filter by query parameters
		objType := sphere.ObjectType(r.URL.Query().Get("type"))
		desktop := sphere.VirtualDesktop(r.URL.Query().Get("desktop"))
		tag := r.URL.Query().Get("tag")
		searchQuery := r.URL.Query().Get("q")

		if strings.TrimSpace(searchQuery) != "" {
			results := store.Search(searchQuery, 50)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"objects": results, "count": len(results)})
			return
		}

		results := store.List(objType, desktop, tag)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"objects": results, "count": len(results)})

	case http.MethodPost:
		var obj sphere.SphereObject
		if err := json.NewDecoder(r.Body).Decode(&obj); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Invalid object payload: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		if obj.ID == "" {
			prefix := string(obj.Type)
			if prefix == "" {
				prefix = "OBJ"
			}
			obj.ID = sphere.GenerateID(prefix)
		}
		if err := store.Put(&obj); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Failed to store object: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(obj)

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, `{"error":"id query parameter is required"}`, http.StatusBadRequest)
			return
		}
		deleted, err := store.Delete(id)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Failed to delete object: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "deleted": deleted})

	default:
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// HandleSphereContext resolves rich context for the active desktop / focused app.
func HandleSphereContext(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	svc := getSphereService()
	focusedApp := r.URL.Query().Get("app")
	query := r.URL.Query().Get("q")

	ctx, err := svc.ResolveContext(focusedApp, query)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Context resolution failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(ctx)
}

// HandleSphereSendTo handles universal cross-app entity dispatching.
func HandleSphereSendTo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var payload sphere.SendToPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Invalid send_to payload: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	svc := getSphereService()
	createdObj, err := svc.DispatchSendTo(&payload)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Send to failed: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "success",
		"target_app": payload.TargetApp,
		"object":     createdObj,
	})
}

// HandleSphereSearch executes instant fuzzy search across all Sphere objects and apps.
func HandleSphereSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query().Get("q")
	svc := getSphereService()
	results := svc.GetStore().Search(q, 30)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"query":   q,
		"results": results,
		"count":   len(results),
	})
}
