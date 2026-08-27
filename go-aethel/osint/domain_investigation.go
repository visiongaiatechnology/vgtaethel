package osint

// STATUS: DIAMANT VGT SUPREME

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"go-aethel/intelligence/connectors"
	"golang.org/x/net/idna"
)

const ianaRDAPDNSBootstrap = "https://data.iana.org/rdap/dns.json"
const certSpotterIssuancesAPI = "https://api.certspotter.com/v1/issuances"

var domainInvestigationRate = struct {
	sync.Mutex
	StartedAt time.Time
	Requests  int
}{}

type DNSFinding struct {
	Addresses   []string `json:"addresses"`
	NameServers []string `json:"name_servers"`
	MailServers []string `json:"mail_servers"`
	TXTRecords  []string `json:"txt_records"`
}

type RDAPFinding struct {
	AuthoritativeURL string            `json:"authoritative_url"`
	Handle           string            `json:"handle,omitempty"`
	LDHName          string            `json:"ldh_name,omitempty"`
	UnicodeName      string            `json:"unicode_name,omitempty"`
	Status           []string          `json:"status,omitempty"`
	NameServers      []string          `json:"name_servers,omitempty"`
	Events           map[string]string `json:"events,omitempty"`
	Notices          []string          `json:"notices,omitempty"`
}

type CertificateTransparencyFinding struct {
	ID         string   `json:"id"`
	TBSSHA256  string   `json:"tbs_sha256,omitempty"`
	CertSHA256 string   `json:"cert_sha256,omitempty"`
	DNSNames   []string `json:"dns_names"`
	IssuerName string   `json:"issuer_name,omitempty"`
	NotBefore  string   `json:"not_before,omitempty"`
	NotAfter   string   `json:"not_after,omitempty"`
	Revoked    bool     `json:"revoked"`
}

type DomainInvestigation struct {
	Domain             string                           `json:"domain"`
	DNS                DNSFinding                       `json:"dns"`
	RDAP               RDAPFinding                      `json:"rdap"`
	Certificates       []CertificateTransparencyFinding `json:"certificate_transparency"`
	Policy             connectors.Policy                `json:"policy"`
	CTPolicy           connectors.Policy                `json:"certificate_transparency_policy"`
	CollectionWarnings []string                         `json:"collection_warnings,omitempty"`
	CollectedAt        time.Time                        `json:"collected_at"`
}

func DomainInvestigationPolicy() connectors.Policy {
	return connectors.PublicOSINTPolicy("IANA-RDAP-registry-and-authoritative-provider-terms", "https://www.iana.org/help/terms-of-service", 10, time.Minute, 180*24*time.Hour)
}

func CertificateTransparencyPolicy() connectors.Policy {
	return connectors.PublicOSINTPolicy("SSLmate-CertSpotter-API-terms", "https://sslmate.com/terms/", 10, time.Minute, 180*24*time.Hour)
}

func InvestigateDomain(ctx context.Context, rawDomain string) (DomainInvestigation, error) {
	if err := authorizeDomainInvestigation(time.Now().UTC()); err != nil {
		return DomainInvestigation{}, err
	}
	domain, err := canonicalDomain(rawDomain)
	if err != nil {
		return DomainInvestigation{}, err
	}
	dns, err := resolvePublicDNS(ctx, domain)
	if err != nil {
		return DomainInvestigation{}, err
	}
	bootstrap, _, err := fetchBoundedPublicJSON(ianaRDAPDNSBootstrap)
	if err != nil {
		return DomainInvestigation{}, err
	}
	baseURL, err := rdapBaseForDomain(bootstrap, domain)
	if err != nil {
		return DomainInvestigation{}, err
	}
	rdapURL := strings.TrimRight(baseURL, "/") + "/domain/" + url.PathEscape(domain)
	body, _, err := fetchBoundedPublicJSON(rdapURL)
	if err != nil {
		return DomainInvestigation{}, err
	}
	rdap, err := parseRDAPDomain(body, rdapURL)
	if err != nil {
		return DomainInvestigation{}, err
	}
	result := DomainInvestigation{Domain: domain, DNS: dns, RDAP: rdap, Policy: DomainInvestigationPolicy(), CTPolicy: CertificateTransparencyPolicy(), CollectedAt: time.Now().UTC()}
	certificates, ctErr := fetchCertificateTransparency(ctx, domain)
	if ctErr != nil {
		result.CollectionWarnings = append(result.CollectionWarnings, "certificate transparency source unavailable")
	} else {
		result.Certificates = certificates
	}
	return result, nil
}

func fetchCertificateTransparency(ctx context.Context, domain string) ([]CertificateTransparencyFinding, error) {
	query := url.Values{}
	query.Set("domain", domain)
	query.Set("include_subdomains", "true")
	query["expand"] = []string{"dns_names", "issuer"}
	query.Set("match_wildcards", "true")
	requestURL := certSpotterIssuancesAPI + "?" + query.Encode()
	body, _, err := fetchBoundedPublicJSONContext(ctx, requestURL)
	if err != nil {
		return nil, err
	}
	return parseCertificateTransparency(body, domain)
}

func parseCertificateTransparency(body []byte, domain string) ([]CertificateTransparencyFinding, error) {
	var records []struct {
		ID         string   `json:"id"`
		TBSSHA256  string   `json:"tbs_sha256"`
		CertSHA256 string   `json:"cert_sha256"`
		DNSNames   []string `json:"dns_names"`
		Issuer     struct {
			Name string `json:"name"`
		} `json:"issuer"`
		NotBefore string `json:"not_before"`
		NotAfter  string `json:"not_after"`
		Revoked   bool   `json:"revoked"`
	}
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, errors.New("certificate transparency response is invalid")
	}
	result := make([]CertificateTransparencyFinding, 0, min(len(records), 250))
	seen := make(map[string]bool, len(records))
	for _, record := range records {
		if record.ID == "" || seen[record.ID] {
			continue
		}
		names := make([]string, 0, len(record.DNSNames))
		for _, rawName := range record.DNSNames {
			name := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(rawName)), "*.")
			if name == domain || strings.HasSuffix(name, "."+domain) {
				names = append(names, rawName)
			}
		}
		if len(names) == 0 {
			continue
		}
		seen[record.ID] = true
		result = append(result, CertificateTransparencyFinding{ID: record.ID, TBSSHA256: record.TBSSHA256, CertSHA256: record.CertSHA256, DNSNames: uniqueNonEmptyStrings(names), IssuerName: record.Issuer.Name, NotBefore: record.NotBefore, NotAfter: record.NotAfter, Revoked: record.Revoked})
		if len(result) == 250 {
			break
		}
	}
	return result, nil
}

func authorizeDomainInvestigation(now time.Time) error {
	domainInvestigationRate.Lock()
	defer domainInvestigationRate.Unlock()
	policy := DomainInvestigationPolicy()
	if domainInvestigationRate.StartedAt.IsZero() || now.Sub(domainInvestigationRate.StartedAt) >= policy.Rate.Window {
		domainInvestigationRate.StartedAt = now
		domainInvestigationRate.Requests = 0
	}
	if domainInvestigationRate.Requests >= policy.Rate.Requests {
		return errors.New("domain investigation rate policy exceeded")
	}
	domainInvestigationRate.Requests++
	return nil
}

func canonicalDomain(raw string) (string, error) {
	value := strings.TrimSpace(strings.TrimSuffix(strings.ToLower(raw), "."))
	if value == "" || len(value) > 253 || strings.ContainsAny(value, "/\\:@?#\x00\r\n") {
		return "", errors.New("domain is invalid")
	}
	ascii, err := idna.Lookup.ToASCII(value)
	if err != nil || !isCanonicalCollectorHostname(ascii) || net.ParseIP(ascii) != nil {
		return "", errors.New("domain is invalid")
	}
	return ascii, nil
}

func isCanonicalCollectorHostname(host string) bool {
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, character := range label {
			if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
				return false
			}
		}
	}
	return strings.Contains(host, ".")
}

func resolvePublicDNS(ctx context.Context, domain string) (DNSFinding, error) {
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, domain)
	if err != nil || len(addresses) == 0 {
		return DNSFinding{}, errors.New("domain address resolution failed")
	}
	result := DNSFinding{}
	for _, address := range addresses {
		if isBlockedCollectorIP(address.IP) {
			return DNSFinding{}, errors.New("domain resolved to a blocked address")
		}
		result.Addresses = append(result.Addresses, address.IP.String())
	}
	if values, lookupErr := net.DefaultResolver.LookupNS(ctx, domain); lookupErr == nil {
		for _, value := range values {
			result.NameServers = append(result.NameServers, strings.TrimSuffix(strings.ToLower(value.Host), "."))
		}
	}
	if values, lookupErr := net.DefaultResolver.LookupMX(ctx, domain); lookupErr == nil {
		for _, value := range values {
			result.MailServers = append(result.MailServers, strings.TrimSuffix(strings.ToLower(value.Host), "."))
		}
	}
	if values, lookupErr := net.DefaultResolver.LookupTXT(ctx, domain); lookupErr == nil {
		for _, value := range values {
			if len(value) <= 1024 {
				result.TXTRecords = append(result.TXTRecords, value)
			}
		}
	}
	sort.Strings(result.Addresses)
	sort.Strings(result.NameServers)
	sort.Strings(result.MailServers)
	sort.Strings(result.TXTRecords)
	return result, nil
}

func rdapBaseForDomain(body []byte, domain string) (string, error) {
	var bootstrap struct {
		Services [][][]string `json:"services"`
	}
	if err := json.Unmarshal(body, &bootstrap); err != nil {
		return "", errors.New("IANA RDAP bootstrap is invalid")
	}
	parts := strings.Split(domain, ".")
	tld := parts[len(parts)-1]
	for _, service := range bootstrap.Services {
		if len(service) != 2 {
			continue
		}
		matched := false
		for _, candidate := range service[0] {
			matched = matched || strings.EqualFold(candidate, tld)
		}
		if !matched || len(service[1]) == 0 {
			continue
		}
		base := service[1][0]
		if err := validatePublicCollectorURL(base); err != nil {
			return "", errors.New("authoritative RDAP URL failed policy")
		}
		return base, nil
	}
	return "", errors.New("no authoritative RDAP service registered for TLD")
}

func parseRDAPDomain(body []byte, authoritativeURL string) (RDAPFinding, error) {
	var document struct {
		Handle      string   `json:"handle"`
		LDHName     string   `json:"ldhName"`
		UnicodeName string   `json:"unicodeName"`
		Status      []string `json:"status"`
		NameServers []struct {
			LDHName string `json:"ldhName"`
		} `json:"nameservers"`
		Events []struct {
			Action string `json:"eventAction"`
			Date   string `json:"eventDate"`
		} `json:"events"`
		Notices []struct {
			Title       string   `json:"title"`
			Description []string `json:"description"`
		} `json:"notices"`
	}
	decoder := json.NewDecoder(io.LimitReader(strings.NewReader(string(body)), 2<<20))
	if err := decoder.Decode(&document); err != nil || document.LDHName == "" {
		return RDAPFinding{}, errors.New("authoritative RDAP response is invalid")
	}
	finding := RDAPFinding{AuthoritativeURL: authoritativeURL, Handle: document.Handle, LDHName: document.LDHName, UnicodeName: document.UnicodeName, Status: uniqueNonEmptyStrings(document.Status), Events: make(map[string]string)}
	for _, nameServer := range document.NameServers {
		if nameServer.LDHName != "" {
			finding.NameServers = append(finding.NameServers, strings.ToLower(nameServer.LDHName))
		}
	}
	for _, event := range document.Events {
		if event.Action != "" && event.Date != "" {
			finding.Events[event.Action] = event.Date
		}
	}
	for _, notice := range document.Notices {
		text := strings.TrimSpace(notice.Title + ": " + strings.Join(notice.Description, " "))
		if text != ":" && len(text) <= 2000 {
			finding.Notices = append(finding.Notices, text)
		}
	}
	sort.Strings(finding.NameServers)
	return finding, nil
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
