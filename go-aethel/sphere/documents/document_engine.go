// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/sphere/documents/document_engine.go
// Purpose: Document Versioning, Snapshot Restore Points, and Track Changes Diff Engine for VGT Writer 2.0

package documents

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"go-aethel/sphere"
)

type Engine struct {
	store *sphere.ObjectStore
}

func NewEngine(store *sphere.ObjectStore) *Engine {
	return &Engine{store: store}
}

// CalculateMetrics extracts word count, character count, and reading time from HTML/text.
func CalculateMetrics(text string) (int, int, int) {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return 0, 0, 0
	}
	words := len(strings.Fields(clean))
	chars := len([]rune(clean))
	readTime := int(math.Ceil(float64(words) / 220.0))
	if readTime == 0 && words > 0 {
		readTime = 1
	}
	return words, chars, readTime
}

// CreateDocument creates and stores a new rich DocumentObject.
func (e *Engine) CreateDocument(title string, contentHTML string, author string) (*sphere.DocumentObject, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Neues Dokument"
	}
	if author == "" {
		author = "OPERATOR"
	}
	words, chars, readTime := CalculateMetrics(contentHTML)
	now := time.Now().UTC()

	doc := &sphere.DocumentObject{
		ID:           sphere.GenerateID("DOC"),
		Title:        title,
		ContentHTML:  contentHTML,
		WordCount:    words,
		CharCount:    chars,
		ReadTimeMin:  readTime,
		Author:       author,
		LastModified: now,
		Versions: []sphere.DocumentVersion{
			{
				VersionID:   fmt.Sprintf("V1-%d", now.Unix()),
				Timestamp:   now,
				Summary:     "Initialversion erstellt",
				ContentHTML: contentHTML,
				Author:      author,
			},
		},
		PendingDiffs: make([]sphere.TrackChangeProposal, 0),
	}

	if err := e.SaveDocument(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// SaveDocument persists a DocumentObject as a Universal SphereObject.
func (e *Engine) SaveDocument(doc *sphere.DocumentObject) error {
	if doc == nil || doc.ID == "" {
		return errors.New("invalid document")
	}
	doc.LastModified = time.Now().UTC()
	doc.WordCount, doc.CharCount, doc.ReadTimeMin = CalculateMetrics(doc.ContentHTML)

	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	obj := &sphere.SphereObject{
		ID:        doc.ID,
		Type:      sphere.TypeDocument,
		Title:     doc.Title,
		Summary:   fmt.Sprintf("%d Wörter · %d Zeichen · %d Min Lesezeit", doc.WordCount, doc.CharCount, doc.ReadTimeMin),
		Desktop:   sphere.DesktopWork,
		Tags:      []string{"writer", "document", "text"},
		Data:      data,
		UpdatedAt: doc.LastModified,
		SourceApp: "writer",
	}
	return e.store.Put(obj)
}

// GetDocument loads a DocumentObject by ID.
func (e *Engine) GetDocument(id string) (*sphere.DocumentObject, error) {
	obj, exists := e.store.Get(id)
	if !exists || obj.Type != sphere.TypeDocument {
		return nil, fmt.Errorf("document %s not found", id)
	}
	var doc sphere.DocumentObject
	if err := json.Unmarshal(obj.Data, &doc); err != nil {
		return nil, fmt.Errorf("failed to deserialize document data: %w", err)
	}
	return &doc, nil
}

// ListDocuments lists all DocumentObjects.
func (e *Engine) ListDocuments() ([]*sphere.DocumentObject, error) {
	objects := e.store.List(sphere.TypeDocument, "", "")
	docs := make([]*sphere.DocumentObject, 0, len(objects))
	for _, obj := range objects {
		var doc sphere.DocumentObject
		if err := json.Unmarshal(obj.Data, &doc); err == nil {
			docs = append(docs, &doc)
		}
	}
	return docs, nil
}

// ProposeChange creates a TrackChangeProposal without overwriting the document.
func (e *Engine) ProposeChange(docID string, explanation string, originalSnippet string, proposedSnippet string, proposedBy string) (*sphere.TrackChangeProposal, error) {
	doc, err := e.GetDocument(docID)
	if err != nil {
		return nil, err
	}
	if proposedBy == "" {
		proposedBy = "AETHEL // AI Core"
	}
	diff := sphere.TrackChangeProposal{
		DiffID:       fmt.Sprintf("DIFF-%d", time.Now().UnixNano()%1000000),
		ProposedBy:   proposedBy,
		Timestamp:    time.Now().UTC(),
		Explanation:  explanation,
		OriginalText: originalSnippet,
		ProposedText: proposedSnippet,
		Status:       "pending",
	}

	doc.PendingDiffs = append(doc.PendingDiffs, diff)
	if err := e.SaveDocument(doc); err != nil {
		return nil, err
	}
	return &diff, nil
}

// ApplyChange accepts or rejects a pending TrackChangeProposal.
func (e *Engine) ApplyChange(docID string, diffID string, accept bool) error {
	doc, err := e.GetDocument(docID)
	if err != nil {
		return err
	}

	foundIdx := -1
	for i, d := range doc.PendingDiffs {
		if d.DiffID == diffID {
			foundIdx = i
			break
		}
	}
	if foundIdx < 0 {
		return fmt.Errorf("diff %s not found in document %s", diffID, docID)
	}

	targetDiff := doc.PendingDiffs[foundIdx]
	if accept {
		// Snapshot current version first
		now := time.Now().UTC()
		doc.Versions = append(doc.Versions, sphere.DocumentVersion{
			VersionID:   fmt.Sprintf("V%d-%d", len(doc.Versions)+1, now.Unix()),
			Timestamp:   now,
			Summary:     fmt.Sprintf("Änderung akzeptiert: %s", targetDiff.Explanation),
			ContentHTML: doc.ContentHTML,
			Author:      "OPERATOR",
		})

		// Apply textual replacement
		if targetDiff.OriginalText != "" && strings.Contains(doc.ContentHTML, targetDiff.OriginalText) {
			doc.ContentHTML = strings.Replace(doc.ContentHTML, targetDiff.OriginalText, targetDiff.ProposedText, 1)
		} else {
			// Append or wrap if snippet not found directly
			doc.ContentHTML = doc.ContentHTML + "\n" + targetDiff.ProposedText
		}
	}

	// Remove applied diff from pending list
	doc.PendingDiffs = append(doc.PendingDiffs[:foundIdx], doc.PendingDiffs[foundIdx+1:]...)
	return e.SaveDocument(doc)
}

// CreateSnapshot manually creates a named restore point for a document.
func (e *Engine) CreateSnapshot(docID string, summary string) error {
	doc, err := e.GetDocument(docID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	doc.Versions = append(doc.Versions, sphere.DocumentVersion{
		VersionID:   fmt.Sprintf("SNAP-%d", now.Unix()),
		Timestamp:   now,
		Summary:     summary,
		ContentHTML: doc.ContentHTML,
		Author:      doc.Author,
	})
	return e.SaveDocument(doc)
}
