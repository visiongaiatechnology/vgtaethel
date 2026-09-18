// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/sphere/research/research_engine.go
// Purpose: Research Desk, Evidence Collection, Provenance Tracking, and Briefing Synthesis

package research

import (
	"encoding/json"
	"errors"
	"fmt"
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

// CreateProject initializes a new ResearchProject.
func (e *Engine) CreateProject(title string, description string, tags []string) (*sphere.ResearchProject, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("research project title is required")
	}
	now := time.Now().UTC()
	proj := &sphere.ResearchProject{
		ID:          sphere.GenerateID("RES"),
		Title:       title,
		Description: description,
		ItemCount:   0,
		Tags:        tags,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	data, err := json.Marshal(proj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal research project: %w", err)
	}

	obj := &sphere.SphereObject{
		ID:        proj.ID,
		Type:      sphere.TypeProject,
		Title:     proj.Title,
		Summary:   fmt.Sprintf("%s · %d Quellen", proj.Description, proj.ItemCount),
		Desktop:   sphere.DesktopResearch,
		Tags:      append([]string{"research", "sources"}, tags...),
		Data:      data,
		CreatedAt: now,
		UpdatedAt: now,
		SourceApp: "research",
	}

	if err := e.store.Put(obj); err != nil {
		return nil, err
	}
	return proj, nil
}

// AddClip adds an extracted evidence snippet or citation to a research project.
func (e *Engine) AddClip(projectID string, title string, sourceURL string, sourceTitle string, extractedText string, keyTakeaway string, provenance string) (*sphere.ResearchItem, error) {
	if strings.TrimSpace(title) == "" {
		title = "Recherche-Auszug"
	}
	if strings.TrimSpace(extractedText) == "" {
		return nil, errors.New("extracted text cannot be empty")
	}
	if provenance == "" {
		provenance = "Aethel Research Desk"
	}

	now := time.Now().UTC()
	item := &sphere.ResearchItem{
		ID:            sphere.GenerateID("CLIP"),
		ProjectID:     projectID,
		Title:         title,
		SourceURL:     sourceURL,
		SourceTitle:   sourceTitle,
		ExtractedText: extractedText,
		KeyTakeaway:   keyTakeaway,
		Provenance:    provenance,
		CapturedAt:    now,
		Confidence:    "HIGH",
	}

	data, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal research item: %w", err)
	}

	obj := &sphere.SphereObject{
		ID:        item.ID,
		Type:      sphere.TypeResearchItem,
		Title:     item.Title,
		Summary:   fmt.Sprintf("Quelle: %s · %s", item.Provenance, item.SourceTitle),
		Desktop:   sphere.DesktopResearch,
		Tags:      []string{"clip", "evidence", "research"},
		Data:      data,
		CreatedAt: now,
		UpdatedAt: now,
		SourceApp: "research",
	}

	if err := e.store.Put(obj); err != nil {
		return nil, err
	}

	// Update parent project item count if projectID exists
	if projectID != "" {
		if projObj, exists := e.store.Get(projectID); exists {
			var proj sphere.ResearchProject
			if err := json.Unmarshal(projObj.Data, &proj); err == nil {
				proj.ItemCount++
				proj.UpdatedAt = now
				if pData, err2 := json.Marshal(proj); err2 == nil {
					projObj.Data = pData
					projObj.Summary = fmt.Sprintf("%s · %d Quellen", proj.Description, proj.ItemCount)
					_ = e.store.Put(projObj)
				}
			}
		}
	}

	return item, nil
}

// ListItems returns all research items, optionally filtered by projectID.
func (e *Engine) ListItems(projectID string) ([]*sphere.ResearchItem, error) {
	objects := e.store.List(sphere.TypeResearchItem, "", "")
	items := make([]*sphere.ResearchItem, 0, len(objects))
	for _, obj := range objects {
		var item sphere.ResearchItem
		if err := json.Unmarshal(obj.Data, &item); err == nil {
			if projectID == "" || item.ProjectID == projectID {
				items = append(items, &item)
			}
		}
	}
	return items, nil
}

// GenerateBriefing synthesizes all research items of a project into structured Markdown.
func (e *Engine) GenerateBriefing(projectID string) (string, error) {
	items, err := e.ListItems(projectID)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("# Forschungs- & Lagebericht\n\n")
	b.WriteString(fmt.Sprintf("**Erstellt:** %s · **Quellen:** %d\n\n", time.Now().Format("02.01.2006 15:04"), len(items)))
	b.WriteString("---\n\n")

	if len(items) == 0 {
		b.WriteString("Keine Recherche-Elemente in diesem Projekt gefunden.\n")
		return b.String(), nil
	}

	b.WriteString("## Wichtigste Erkenntnisse\n\n")
	for i, item := range items {
		if item.KeyTakeaway != "" {
			b.WriteString(fmt.Sprintf("%d. **%s**: %s\n", i+1, item.Title, item.KeyTakeaway))
		}
	}
	b.WriteString("\n## Detaillierte Auszüge & Provenienz\n\n")
	for _, item := range items {
		b.WriteString(fmt.Sprintf("### %s\n", item.Title))
		if item.SourceURL != "" {
			b.WriteString(fmt.Sprintf("*Quelle: [%s](%s) (%s)*\n\n", item.SourceTitle, item.SourceURL, item.Provenance))
		} else {
			b.WriteString(fmt.Sprintf("*Herkunft: %s*\n\n", item.Provenance))
		}
		b.WriteString(fmt.Sprintf("> %s\n\n", strings.ReplaceAll(item.ExtractedText, "\n", "\n> ")))
	}

	return b.String(), nil
}
