package osint

// STATUS: DIAMANT VGT SUPREME

import (
	"strings"
	"testing"
	"time"

	"go-aethel/intelligence/connectors"
)

func TestCISAKEVParserPreservesCyberProvenance(t *testing.T) {
	fetchedAt := time.Now().UTC()
	payload := `{"vulnerabilities":[{"cveID":"CVE-2026-1234","vendorProject":"Vendor","product":"Gateway","vulnerabilityName":"Gateway flaw","dateAdded":"2026-08-20","shortDescription":"Actively exploited issue","requiredAction":"Apply vendor mitigations","knownRansomwareCampaignUse":"Known"}]}`
	observations, err := parseCISAKEV([]byte(payload), fetchedAt)
	if err != nil || len(observations) != 1 {
		t.Fatalf("CISA parser failed: %+v %v", observations, err)
	}
	observation := observations[0]
	if observation.Domain != "cyber" || observation.SourceID != "cisa-kev" || observation.OriginalURL != cisaKEVURL || !strings.Contains(observation.RawText, "CVE-2026-1234") || !observation.FetchedAt.Equal(fetchedAt) {
		t.Fatalf("CISA provenance incomplete: %+v", observation)
	}
}

func TestReliefWebParserPreservesHumanitarianProvenance(t *testing.T) {
	fetchedAt := time.Now().UTC()
	payload := `{"data":[{"id":42,"fields":{"title":"Flood response update","body-html":"<p>Verified field report.</p>","url":"https://reliefweb.int/report/test","date":{"created":"2026-08-20T12:00:00+00:00"},"country":[{"name":"Exampleland"}],"source":[{"name":"OCHA"}]}}]}`
	observations, err := parseReliefWeb([]byte(payload), fetchedAt)
	if err != nil || len(observations) != 1 {
		t.Fatalf("ReliefWeb parser failed: %+v %v", observations, err)
	}
	observation := observations[0]
	if observation.Domain != "humanitarian" || observation.SourceID != "reliefweb" || !strings.Contains(observation.RawText, "Exampleland") || strings.Contains(observation.RawText, "<p>") {
		t.Fatalf("ReliefWeb provenance/content invalid: %+v", observation)
	}
}

func TestSpecialistConnectorsDeclareCompletePolicies(t *testing.T) {
	for _, connector := range []connectors.Connector{&cisaKEVConnector{}, &reliefWebConnector{}, newOFACActionsConnector(), newFAANASConnector(), newUSCGBNMConnector()} {
		if err := connectors.ValidateDescriptor(connector.Descriptor()); err != nil {
			t.Fatalf("connector %s has invalid policy: %v", connector.Descriptor().Name, err)
		}
	}
}

func TestOfficialStatusParserProducesBoundedProvenance(t *testing.T) {
	fetchedAt := time.Now().UTC()
	payload := `<html><body><nav>menu noise</nav><main><h1>National status</h1><table><tr><td>SFO</td><td>WEATHER / LOW CEILINGS</td></tr></table></main><script>steal()</script></body></html>`
	observations, err := parseOfficialStatusPage([]byte(payload), fetchedAt, "faa-atcscc", faaNASURL, "infrastructure")
	if err != nil || len(observations) != 1 {
		t.Fatalf("official status parser failed: %+v %v", observations, err)
	}
	observation := observations[0]
	if observation.SourceID != "faa-atcscc" || observation.OriginalURL != faaNASURL || observation.ParserVersion != "official-status-html-v1" || strings.Contains(observation.RawText, "menu noise") || strings.Contains(observation.RawText, "steal()") || !strings.Contains(observation.RawText, "LOW CEILINGS") {
		t.Fatalf("official status provenance or extraction invalid: %+v", observation)
	}
}
