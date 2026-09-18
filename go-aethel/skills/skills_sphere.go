// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/skills/skills_sphere.go
// Purpose: AI Agent Tool Definitions for Sphere 2.0 (Writer, Track Changes, Trips, Plans, Research, Send To)

package skills

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"go-aethel/security"
	"go-aethel/sphere"
	"go-aethel/sphere/documents"
	"go-aethel/sphere/planner"
	"go-aethel/sphere/research"
	"go-aethel/sphere/travel"

	xhtml "golang.org/x/net/html"
)

// SphereWriteDocumentSkill is a provider-independent Writer contract.
type SphereWriteDocumentSkill struct{}

type sphereWriteDocumentArgs struct {
	Content string `json:"content"`
	Format  string `json:"format,omitempty"`
}

func (s *SphereWriteDocumentSkill) Name() string { return "sphere_write_document" }
func (s *SphereWriteDocumentSkill) Description() string {
	return "Ersetzt den sichtbaren Inhalt des AETHEL Writers. Nutze dies für Gedichte, Geschichten, Artikel und Dokumententwürfe in Sphere."
}
func (s *SphereWriteDocumentSkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *SphereWriteDocumentSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"content": map[string]interface{}{"type": "string", "minLength": 1, "maxLength": 500000},
			"format":  map[string]interface{}{"type": "string", "enum": []string{"plain", "markdown", "html"}},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}
}

func (s *SphereWriteDocumentSkill) Execute(args json.RawMessage) (string, error) {
	var input sphereWriteDocumentArgs
	decoder := json.NewDecoder(bytes.NewReader(args))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return "", errors.New("ungültige Writer-Anfrage")
	}
	input.Content = strings.TrimSpace(input.Content)
	if input.Content == "" || len([]rune(input.Content)) > 500000 {
		return "", errors.New("Writer-Inhalt muss zwischen 1 und 500000 Zeichen enthalten")
	}
	format := strings.ToLower(strings.TrimSpace(input.Format))
	if format == "" {
		format = "html"
	}
	var document string
	switch format {
	case "plain", "markdown":
		document = "<pre>" + html.EscapeString(input.Content) + "</pre>"
	case "html":
		document = sanitizeSphereHTML(input.Content)
	default:
		return "", errors.New("Writer-Format wird nicht unterstützt")
	}
	if strings.TrimSpace(document) == "" {
		return "", errors.New("Writer-Sanitizer entfernte den vollständigen Inhalt")
	}
	if err := writeSphereDocumentAtomically(document); err != nil {
		return "", err
	}
	security.LogKernelActivity("SPHERE_DOCUMENT_WRITE", "sphere_document.html", "SUCCESS")
	return fmt.Sprintf("Writer-Dokument sichtbar aktualisiert (%d Zeichen, Format %s).", len([]rune(input.Content)), format), nil
}

// SphereProposeDocumentChangeSkill allows the AI to propose structured Track Changes diffs.
type SphereProposeDocumentChangeSkill struct {
	DocEngine *documents.Engine
}

type sphereProposeChangeArgs struct {
	DocumentID      string `json:"document_id,omitempty"`
	Explanation     string `json:"explanation"`
	OriginalSnippet string `json:"original_snippet"`
	ProposedSnippet string `json:"proposed_snippet"`
}

func (s *SphereProposeDocumentChangeSkill) Name() string { return "sphere_propose_document_change" }
func (s *SphereProposeDocumentChangeSkill) Description() string {
	return "Schlägt eine Änderung im Aethel Writer als Track-Changes-Diff vor. Der Operator kann diesen Vorschlag mit einem Klick annehmen oder ablehnen."
}
func (s *SphereProposeDocumentChangeSkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *SphereProposeDocumentChangeSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"document_id":      map[string]interface{}{"type": "string", "description": "Optionale ID des Ziel-Dokuments."},
			"explanation":      map[string]interface{}{"type": "string", "description": "Begründung für die vorgeschlagene Änderung."},
			"original_snippet": map[string]interface{}{"type": "string", "description": "Der zu ersetzende Textabschnitt."},
			"proposed_snippet": map[string]interface{}{"type": "string", "description": "Der neue Textabschnitt."},
		},
		"required":             []string{"explanation", "original_snippet", "proposed_snippet"},
		"additionalProperties": false,
	}
}

func (s *SphereProposeDocumentChangeSkill) Execute(args json.RawMessage) (string, error) {
	var input sphereProposeChangeArgs
	decoder := json.NewDecoder(bytes.NewReader(args))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return "", errors.New("ungültige Änderungsvorschlag-Anfrage")
	}

	docID := input.DocumentID
	if docID == "" {
		docID = "DOC-DEFAULT"
	}
	if s.DocEngine != nil {
		diff, err := s.DocEngine.ProposeChange(docID, input.Explanation, input.OriginalSnippet, input.ProposedSnippet, "AETHEL // AI Core")
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Änderungsvorschlag registriert (Diff-ID: %s). Begründung: %s", diff.DiffID, input.Explanation), nil
	}
	return fmt.Sprintf("Änderungsvorschlag vorgemerkt: '%s' -> '%s' (%s)", input.OriginalSnippet, input.ProposedSnippet, input.Explanation), nil
}

// SphereManageTripSkill allows the AI to create or update persistent Trip objects.
type SphereManageTripSkill struct {
	TravelEngine *travel.Engine
}

type sphereManageTripArgs struct {
	Title          string   `json:"title"`
	Destination    string   `json:"destination"`
	Destinations   []string `json:"destinations,omitempty"`
	StartDate      string   `json:"start_date,omitempty"`
	EndDate        string   `json:"end_date,omitempty"`
	BudgetTotalEUR float64  `json:"budget_total_eur,omitempty"`
	Notes          string   `json:"notes,omitempty"`
}

func (s *SphereManageTripSkill) Name() string { return "sphere_manage_trip" }
func (s *SphereManageTripSkill) Description() string {
	return "Erstellt oder aktualisiert eine strukturierte Reise in Sphere Trip Planner (Ziele, Reisedaten, Budget, Notizen)."
}
func (s *SphereManageTripSkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *SphereManageTripSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"title":            map[string]interface{}{"type": "string", "description": "Titel der Reise, z.B. 'Reise Japan 2027'."},
			"destination":      map[string]interface{}{"type": "string", "description": "Hauptreiseziel, z.B. 'Tokyo, Japan'."},
			"destinations":     map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Weitere Stationen."},
			"start_date":       map[string]interface{}{"type": "string", "description": "Startdatum YYYY-MM-DD."},
			"end_date":         map[string]interface{}{"type": "string", "description": "Enddatum YYYY-MM-DD."},
			"budget_total_eur": map[string]interface{}{"type": "number", "description": "Gesamtbudget in Euro."},
			"notes":            map[string]interface{}{"type": "string", "description": "Notizen oder Vorlieben."},
		},
		"required":             []string{"title", "destination"},
		"additionalProperties": false,
	}
}

func (s *SphereManageTripSkill) Execute(args json.RawMessage) (string, error) {
	var input sphereManageTripArgs
	decoder := json.NewDecoder(bytes.NewReader(args))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return "", errors.New("ungültige Reise-Anfrage")
	}

	trip := &sphere.TripObject{
		Title:          input.Title,
		Destination:    input.Destination,
		Destinations:   input.Destinations,
		StartDate:      input.StartDate,
		EndDate:        input.EndDate,
		BudgetTotalEUR: input.BudgetTotalEUR,
		Notes:          input.Notes,
	}

	if s.TravelEngine != nil {
		created, err := s.TravelEngine.CreateTrip(trip)
		if err != nil {
			return "", err
		}
		security.LogKernelActivity("SPHERE_TRIP_CREATED", created.ID, "SUCCESS")
		return fmt.Sprintf("Reise '%s' (%s, %d Tage, Budget €%.2f) erfolgreich in Sphere Trip Planner angelegt.", created.Title, created.Destination, created.DurationDays, created.BudgetTotalEUR), nil
	}
	return fmt.Sprintf("Reise '%s' nach %s vorbereitet.", input.Title, input.Destination), nil
}

// SphereManagePlanSkill allows the AI to create or update plans with milestones.
type SphereManagePlanSkill struct {
	PlannerEngine *planner.Engine
}

type sphereManagePlanArgs struct {
	Title      string   `json:"title"`
	Objective  string   `json:"objective"`
	TargetDate string   `json:"target_date,omitempty"`
	Milestones []string `json:"milestones,omitempty"`
}

func (s *SphereManagePlanSkill) Name() string { return "sphere_manage_plan" }
func (s *SphereManagePlanSkill) Description() string {
	return "Erstellt einen übergeordneten Plan mit Meilensteinen im Sphere Master Planner."
}
func (s *SphereManagePlanSkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *SphereManagePlanSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"title":       map[string]interface{}{"type": "string", "description": "Titel des Plans."},
			"objective":   map[string]interface{}{"type": "string", "description": "Hauptziel des Plans."},
			"target_date": map[string]interface{}{"type": "string", "description": "Zieldatum YYYY-MM-DD."},
			"milestones":  map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Liste von Meilensteinen."},
		},
		"required":             []string{"title", "objective"},
		"additionalProperties": false,
	}
}

func (s *SphereManagePlanSkill) Execute(args json.RawMessage) (string, error) {
	var input sphereManagePlanArgs
	decoder := json.NewDecoder(bytes.NewReader(args))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return "", errors.New("ungültige Plan-Anfrage")
	}

	plan := &sphere.PlanObject{
		Title:      input.Title,
		Objective:  input.Objective,
		TargetDate: input.TargetDate,
	}
	for i, m := range input.Milestones {
		plan.Milestones = append(plan.Milestones, sphere.PlanMilestone{
			ID:        fmt.Sprintf("M%d", i+1),
			Title:     m,
			Completed: false,
			Order:     i + 1,
		})
	}

	if s.PlannerEngine != nil {
		created, err := s.PlannerEngine.CreatePlan(plan)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Plan '%s' mit %d Meilensteinen erfolgreich im Master Planner registriert.", created.Title, len(created.Milestones)), nil
	}
	return fmt.Sprintf("Plan '%s' erstellt.", input.Title), nil
}

// SphereResearchCaptureSkill allows the AI to capture research snippets into the Research Desk.
type SphereResearchCaptureSkill struct {
	ResearchEngine *research.Engine
}

type sphereResearchCaptureArgs struct {
	ProjectID     string `json:"project_id,omitempty"`
	Title         string `json:"title"`
	SourceURL     string `json:"source_url,omitempty"`
	SourceTitle   string `json:"source_title,omitempty"`
	ExtractedText string `json:"extracted_text"`
	KeyTakeaway   string `json:"key_takeaway,omitempty"`
	Provenance    string `json:"provenance,omitempty"`
}

func (s *SphereResearchCaptureSkill) Name() string { return "sphere_research_capture" }
func (s *SphereResearchCaptureSkill) Description() string {
	return "Speichert ein Recherche-Ergebnis, Zitat oder Beweisstück mit Herkunftsnachweis im Sphere Research Desk."
}
func (s *SphereResearchCaptureSkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *SphereResearchCaptureSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"project_id":     map[string]interface{}{"type": "string", "description": "Optionale Projekt-ID."},
			"title":          map[string]interface{}{"type": "string", "description": "Titel des Auszugs."},
			"source_url":     map[string]interface{}{"type": "string", "description": "URL der Quelle."},
			"source_title":   map[string]interface{}{"type": "string", "description": "Name oder Titel der Quelle."},
			"extracted_text": map[string]interface{}{"type": "string", "description": "Der extrahierte Text oder das Zitat."},
			"key_takeaway":   map[string]interface{}{"type": "string", "description": "Kernaussage."},
			"provenance":     map[string]interface{}{"type": "string", "description": "Herkunft, z.B. 'Aethel Web Browser'."},
		},
		"required":             []string{"title", "extracted_text"},
		"additionalProperties": false,
	}
}

func (s *SphereResearchCaptureSkill) Execute(args json.RawMessage) (string, error) {
	var input sphereResearchCaptureArgs
	decoder := json.NewDecoder(bytes.NewReader(args))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return "", errors.New("ungültige Recherche-Anfrage")
	}

	if s.ResearchEngine != nil {
		item, err := s.ResearchEngine.AddClip(input.ProjectID, input.Title, input.SourceURL, input.SourceTitle, input.ExtractedText, input.KeyTakeaway, input.Provenance)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Recherche-Element '%s' (ID: %s) erfolgreich im Research Desk gesichert.", item.Title, item.ID), nil
	}
	return fmt.Sprintf("Recherche-Element '%s' gesichert.", input.Title), nil
}

// SphereSendToSkill executes universal cross-app entity transfers.
type SphereSendToSkill struct {
	Service *sphere.Service
}

type sphereSendToArgs struct {
	SourceApp  string `json:"source_app"`
	TargetApp  string `json:"target_app"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	ObjectID   string `json:"object_id,omitempty"`
}

func (s *SphereSendToSkill) Name() string { return "sphere_send_to" }
func (s *SphereSendToSkill) Description() string {
	return "Überträgt ein Objekt zwischen Sphere-Anwendungen (z.B. Browser-Artikel -> Writer / Research / Trip, Mail -> Task / Trip)."
}
func (s *SphereSendToSkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *SphereSendToSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"source_app": map[string]interface{}{"type": "string", "description": "Quell-Anwendung, z.B. 'browser', 'mail', 'global_watch'."},
			"target_app": map[string]interface{}{"type": "string", "enum": []string{"writer", "research", "trip", "task"}, "description": "Ziel-Anwendung."},
			"title":      map[string]interface{}{"type": "string", "description": "Titel des Elements."},
			"content":    map[string]interface{}{"type": "string", "description": "Inhalt des Elements."},
			"object_id":  map[string]interface{}{"type": "string", "description": "Optionale Ziel-Objekt-ID (z.B. Trip-ID)."},
		},
		"required":             []string{"source_app", "target_app", "title", "content"},
		"additionalProperties": false,
	}
}

func (s *SphereSendToSkill) Execute(args json.RawMessage) (string, error) {
	var input sphereSendToArgs
	decoder := json.NewDecoder(bytes.NewReader(args))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return "", errors.New("ungültige Send-To Anfrage")
	}

	payload := &sphere.SendToPayload{
		SourceApp:  input.SourceApp,
		TargetApp:  input.TargetApp,
		Title:      input.Title,
		Content:    input.Content,
		ObjectID:   input.ObjectID,
		ObjectType: sphere.TypeDocument,
	}

	if s.Service != nil {
		obj, err := s.Service.DispatchSendTo(payload)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Erfolgreich von %s an %s übertragen (Objekt-ID: %s).", input.SourceApp, input.TargetApp, obj.ID), nil
	}
	return fmt.Sprintf("Objekt '%s' von %s an %s gesendet.", input.Title, input.SourceApp, input.TargetApp), nil
}

var sphereAllowedTags = map[string]bool{
	"h1": true, "h2": true, "h3": true, "p": true, "div": true,
	"strong": true, "b": true, "em": true, "i": true, "u": true,
	"ul": true, "ol": true, "li": true, "blockquote": true,
	"pre": true, "code": true, "br": true, "hr": true,
	"table": true, "thead": true, "tbody": true, "tr": true, "th": true, "td": true,
}

func sanitizeSphereHTML(raw string) string {
	root, err := xhtml.Parse(strings.NewReader(raw))
	if err != nil {
		return ""
	}
	var out strings.Builder
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		switch node.Type {
		case xhtml.TextNode:
			out.WriteString(html.EscapeString(node.Data))
		case xhtml.ElementNode:
			tag := strings.ToLower(node.Data)
			if tag == "script" || tag == "style" || tag == "iframe" || tag == "object" || tag == "svg" || tag == "form" {
				return
			}
			allowed := sphereAllowedTags[tag]
			if allowed {
				out.WriteByte('<')
				out.WriteString(tag)
				out.WriteByte('>')
			}
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				walk(child)
			}
			if allowed && tag != "br" && tag != "hr" {
				out.WriteString("</")
				out.WriteString(tag)
				out.WriteByte('>')
			}
		default:
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				walk(child)
			}
		}
	}
	walk(root)
	return strings.TrimSpace(out.String())
}

func writeSphereDocumentAtomically(content string) error {
	if err := os.MkdirAll(security.WorkspaceDir, 0700); err != nil {
		return errors.New("Writer-Verzeichnis konnte nicht vorbereitet werden")
	}
	temp, err := os.CreateTemp(security.WorkspaceDir, ".sphere-document-*.tmp")
	if err != nil {
		return errors.New("temporäres Writer-Dokument konnte nicht erstellt werden")
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0600); err != nil {
		temp.Close()
		return errors.New("Writer-Dateirechte konnten nicht gesetzt werden")
	}
	if _, err := temp.WriteString(content); err != nil {
		temp.Close()
		return errors.New("Writer-Dokument konnte nicht geschrieben werden")
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return errors.New("Writer-Dokument konnte nicht synchronisiert werden")
	}
	if err := temp.Close(); err != nil {
		return errors.New("Writer-Dokument konnte nicht abgeschlossen werden")
	}
	target := filepath.Join(security.WorkspaceDir, "sphere_document.html")
	if err := os.Rename(tempPath, target); err != nil {
		return errors.New("Writer-Dokument konnte nicht atomar ersetzt werden")
	}
	return nil
}
