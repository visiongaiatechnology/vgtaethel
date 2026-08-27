package osint

// STATUS: DIAMANT VGT SUPREME

import (
	"testing"
	"time"

	"go-aethel/intelligence/connectors"
)

func TestCanonicalDomainRejectsInjectionAndNormalizesIDN(t *testing.T) {
	for _, value := range []string{"", "localhost", "127.0.0.1", "example.com/path", "example.com:443", "-bad.example"} {
		if _, err := canonicalDomain(value); err == nil {
			t.Fatalf("invalid domain accepted: %q", value)
		}
	}
	canonical, err := canonicalDomain("BÜCHER.example.")
	if err != nil || canonical != "xn--bcher-kva.example" {
		t.Fatalf("IDN normalization failed: %q %v", canonical, err)
	}
}

func TestRDAPBootstrapSelectsAuthoritativeService(t *testing.T) {
	bootstrap := []byte(`{"services":[[["com","net"],["https://rdap.example.test/"]],[["de"],["https://rdap.de.example.test/"]]]}`)
	base, err := rdapBaseForDomain(bootstrap, "example.com")
	if err != nil || base != "https://rdap.example.test/" {
		t.Fatalf("authoritative service selection failed: %q %v", base, err)
	}
	if _, err := rdapBaseForDomain(bootstrap, "example.invalid"); err == nil {
		t.Fatal("unregistered TLD received an RDAP service")
	}
}

func TestRDAPParserReturnsBoundedAnalyticFields(t *testing.T) {
	payload := []byte(`{"handle":"EXAMPLE","ldhName":"example.com","unicodeName":"example.com","status":["active"],"nameservers":[{"ldhName":"NS1.EXAMPLE.COM"}],"events":[{"eventAction":"registration","eventDate":"2020-01-01T00:00:00Z"}],"notices":[{"title":"Terms","description":["Public RDAP data"]}]}`)
	finding, err := parseRDAPDomain(payload, "https://rdap.example.test/domain/example.com")
	if err != nil || finding.LDHName != "example.com" || finding.Events["registration"] == "" || len(finding.NameServers) != 1 || finding.NameServers[0] != "ns1.example.com" {
		t.Fatalf("RDAP analytic projection failed: %+v %v", finding, err)
	}
	if err := connectors.ValidatePolicy(DomainInvestigationPolicy()); err != nil {
		t.Fatalf("domain policy invalid: %v", err)
	}
}

func TestDomainInvestigationRatePolicy(t *testing.T) {
	domainInvestigationRate.Lock()
	domainInvestigationRate.StartedAt = time.Time{}
	domainInvestigationRate.Requests = 0
	domainInvestigationRate.Unlock()
	now := time.Now().UTC()
	for index := 0; index < DomainInvestigationPolicy().Rate.Requests; index++ {
		if err := authorizeDomainInvestigation(now); err != nil {
			t.Fatal(err)
		}
	}
	if err := authorizeDomainInvestigation(now); err == nil {
		t.Fatal("domain rate policy was not enforced")
	}
}
