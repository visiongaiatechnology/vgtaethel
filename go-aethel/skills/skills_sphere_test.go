package skills

import (
	"strings"
	"testing"
)

func TestSanitizeSphereHTMLPreservesWriterMarkupAndRemovesActiveContent(t *testing.T) {
	clean := sanitizeSphereHTML(`<h1 onclick="evil()">Titel</h1><script>alert(1)</script><p style="position:fixed">Text <strong>fett</strong></p><iframe src="https://evil.test"></iframe>`)
	for _, forbidden := range []string{"script", "alert", "onclick", "style=", "iframe", "evil.test"} {
		if strings.Contains(strings.ToLower(clean), forbidden) {
			t.Fatalf("active content survived Writer sanitizer: %q", clean)
		}
	}
	if !strings.Contains(clean, "<h1>Titel</h1>") || !strings.Contains(clean, "<strong>fett</strong>") {
		t.Fatalf("safe Writer markup was lost: %q", clean)
	}
}

func TestSphereWriterToolHasClosedSchema(t *testing.T) {
	schema := (&SphereWriteDocumentSkill{}).Parameters()
	if schema["additionalProperties"] != false {
		t.Fatal("Writer tool schema must reject unknown fields")
	}
}
