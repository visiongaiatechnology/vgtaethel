// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/handlers/sphere_document_handlers.go
// Purpose: HTTP Handlers for VGT Writer 2.0 Rich Documents, Version Snapshots, and Track Changes Diffs

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go-aethel/sphere"
)

// HandleSphereDocuments handles listing, loading, creating, and updating VGT Writer documents.
func HandleSphereDocuments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	getSphereService()
	eng := globalDocEng

	switch r.Method {
	case http.MethodGet:
		id := r.URL.Query().Get("id")
		if id != "" {
			doc, err := eng.GetDocument(id)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(doc)
			return
		}

		docs, err := eng.ListDocuments()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"documents": docs, "count": len(docs)})

	case http.MethodPost:
		var incoming struct {
			Title       string `json:"title"`
			ContentHTML string `json:"content_html"`
			Author      string `json:"author"`
		}
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Invalid document payload: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		created, err := eng.CreateDocument(incoming.Title, incoming.ContentHTML, incoming.Author)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(created)

	case http.MethodPut:
		var doc sphere.DocumentObject
		if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Invalid document payload: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		if err := eng.SaveDocument(&doc); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// HandleSphereDocumentProposeChange handles AI change proposals as Track Changes diffs.
func HandleSphereDocumentProposeChange(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	getSphereService()
	eng := globalDocEng

	var req struct {
		DocumentID      string `json:"document_id"`
		Explanation     string `json:"explanation"`
		OriginalSnippet string `json:"original_snippet"`
		ProposedSnippet string `json:"proposed_snippet"`
		ProposedBy      string `json:"proposed_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Invalid proposal payload: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	diff, err := eng.ProposeChange(req.DocumentID, req.Explanation, req.OriginalSnippet, req.ProposedSnippet, req.ProposedBy)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(diff)
}

// HandleSphereDocumentApplyChange handles Accept / Reject for a pending Track Changes diff.
func HandleSphereDocumentApplyChange(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	getSphereService()
	eng := globalDocEng

	var req struct {
		DocumentID string `json:"document_id"`
		DiffID     string `json:"diff_id"`
		Accept     bool   `json:"accept"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Invalid payload: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	if err := eng.ApplyChange(req.DocumentID, req.DiffID, req.Accept); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"accepted": req.Accept,
		"diff_id":  req.DiffID,
	})
}
