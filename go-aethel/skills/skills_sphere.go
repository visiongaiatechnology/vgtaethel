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

	xhtml "golang.org/x/net/html"
)

// SphereWriteDocumentSkill is a narrow, provider-independent Writer contract.
// It can mutate exactly one application-owned document and no arbitrary path.
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

var sphereAllowedTags = map[string]bool{
	"h1": true, "h2": true, "h3": true, "p": true, "div": true,
	"strong": true, "b": true, "em": true, "i": true, "u": true,
	"ul": true, "ol": true, "li": true, "blockquote": true,
	"pre": true, "code": true, "br": true, "hr": true,
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
