// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/handlers/sphere_handlers_test.go
// Purpose: Comprehensive HTTP handler integration tests for Sphere 2.0 endpoints

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-aethel/sphere"
)

func initTestSphereEnv(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	_, err := InitSphereServices(dir)
	if err != nil {
		t.Fatalf("InitSphereServices failed: %v", err)
	}
}

func TestHandleSphereStateHTTP(t *testing.T) {
	initTestSphereEnv(t)

	// 1. GET state
	req := httptest.NewRequest(http.MethodGet, "/v1/sphere/state", nil)
	rec := httptest.NewRecorder()
	HandleSphereState(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. POST updated state
	newState := sphere.DesktopWorkspaceState{
		ActiveDesktop: sphere.DesktopResearch,
		AmbientLevel:  22,
		Windows: map[string]sphere.DesktopWindowState{
			"browser": {AppID: "browser", Desktop: "RESEARCH", Left: 50, Top: 50, Width: 800, Height: 600, IsOpen: true},
		},
	}
	body, _ := json.Marshal(newState)
	postReq := httptest.NewRequest(http.MethodPost, "/v1/sphere/state", bytes.NewReader(body))
	postRec := httptest.NewRecorder()
	HandleSphereState(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on post, got %d: %s", postRec.Code, postRec.Body.String())
	}
}

func TestHandleSphereObjectsHTTP(t *testing.T) {
	initTestSphereEnv(t)

	// 1. POST object
	obj := sphere.SphereObject{
		ID:      "DOC-TEST-HTTP",
		Type:    sphere.TypeDocument,
		Title:   "HTTP Test Document",
		Desktop: sphere.DesktopWork,
		Data:    json.RawMessage(`{"content":"test"}`),
	}
	body, _ := json.Marshal(obj)
	postReq := httptest.NewRequest(http.MethodPost, "/v1/sphere/objects", bytes.NewReader(body))
	postRec := httptest.NewRecorder()
	HandleSphereObjects(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", postRec.Code)
	}

	// 2. GET object
	getReq := httptest.NewRequest(http.MethodGet, "/v1/sphere/objects?type=document", nil)
	getRec := httptest.NewRecorder()
	HandleSphereObjects(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRec.Code)
	}
}

func TestHandleSphereSendToHTTP(t *testing.T) {
	initTestSphereEnv(t)

	payload := sphere.SendToPayload{
		SourceApp:  "browser",
		TargetApp:  "writer",
		ObjectType: sphere.TypeDocument,
		Title:      "Artikel über Quantenüberlegenheit",
		Content:    "<p>Google und IBM erzielen neue Meilensteine in der Qubit-Skalierung.</p>",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/v1/sphere/send_to", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	HandleSphereSendTo(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSphereTripsAndImpactHTTP(t *testing.T) {
	initTestSphereEnv(t)

	// 1. Create Trip
	trip := sphere.TripObject{
		Title:          "Japan 2027",
		Destination:    "Tokyo, Japan",
		StartDate:      "2027-04-01",
		EndDate:        "2027-04-14",
		BudgetTotalEUR: 4000,
	}
	body, _ := json.Marshal(trip)
	postReq := httptest.NewRequest(http.MethodPost, "/v1/sphere/trips", bytes.NewReader(body))
	postRec := httptest.NewRecorder()
	HandleSphereTrips(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", postRec.Code, postRec.Body.String())
	}

	var created sphere.TripObject
	_ = json.Unmarshal(postRec.Body.Bytes(), &created)

	// 2. Correlate Risk Impact
	impactPayload := map[string]string{
		"trip_id":    created.ID,
		"region":     "Tokyo",
		"signal":     "Starkes Erdbeben M6.0 gemeldet",
		"severity":   "CRITICAL",
		"confidence": "HIGH",
	}
	iBody, _ := json.Marshal(impactPayload)
	iReq := httptest.NewRequest(http.MethodPost, "/v1/sphere/trips/impact", bytes.NewReader(iBody))
	iRec := httptest.NewRecorder()
	HandleSphereTripImpact(iRec, iReq)

	if iRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", iRec.Code, iRec.Body.String())
	}
}

func TestHandleSphereDocumentsTrackChangesHTTP(t *testing.T) {
	initTestSphereEnv(t)

	// 1. Create Document
	createPayload := map[string]string{
		"title":        "Sicherheitskonzept Aethel",
		"content_html": "<p>Standard Authentifizierung aktiv.</p>",
		"author":       "OPERATOR",
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/v1/sphere/documents", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	HandleSphereDocuments(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var doc sphere.DocumentObject
	_ = json.Unmarshal(rec.Body.Bytes(), &doc)

	// 2. Propose Change
	propPayload := map[string]string{
		"document_id":      doc.ID,
		"explanation":      "Zero-Trust MFA erzwingen",
		"original_snippet": "Standard Authentifizierung",
		"proposed_snippet": "Zero-Trust MFA & Hardware Key",
		"proposed_by":      "AETHEL // Guard",
	}
	pBody, _ := json.Marshal(propPayload)
	pReq := httptest.NewRequest(http.MethodPost, "/v1/sphere/documents/propose_change", bytes.NewReader(pBody))
	pRec := httptest.NewRecorder()
	HandleSphereDocumentProposeChange(pRec, pReq)

	if pRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on propose, got %d: %s", pRec.Code, pRec.Body.String())
	}
	var diff sphere.TrackChangeProposal
	_ = json.Unmarshal(pRec.Body.Bytes(), &diff)

	// 3. Apply Change
	applyPayload := map[string]interface{}{
		"document_id": doc.ID,
		"diff_id":     diff.DiffID,
		"accept":      true,
	}
	aBody, _ := json.Marshal(applyPayload)
	aReq := httptest.NewRequest(http.MethodPost, "/v1/sphere/documents/apply_change", bytes.NewReader(aBody))
	aRec := httptest.NewRecorder()
	HandleSphereDocumentApplyChange(aRec, aReq)

	if aRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on apply, got %d", aRec.Code)
	}
}

func TestHandleSphereBrowserTabsHTTP(t *testing.T) {
	initTestSphereEnv(t)

	// 1. GET tabs
	getReq := httptest.NewRequest(http.MethodGet, "/v1/sphere/browser/tabs", nil)
	getRec := httptest.NewRecorder()
	HandleSphereBrowserTabs(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRec.Code)
	}

	// 2. POST new tab
	newTab := map[string]string{
		"url":   "https://github.com",
		"title": "GitHub",
	}
	body, _ := json.Marshal(newTab)
	postReq := httptest.NewRequest(http.MethodPost, "/v1/sphere/browser/tabs", bytes.NewReader(body))
	postRec := httptest.NewRecorder()
	HandleSphereBrowserTabs(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", postRec.Code)
	}
}
