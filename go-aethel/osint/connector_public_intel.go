package osint

// STATUS: DIAMANT VGT SUPREME

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go-aethel/intelligence"
	"go-aethel/intelligence/connectors"
)

const (
	cisaKEVURL      = "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json"
	reliefWebURL    = "https://api.reliefweb.int/v2/reports?appname=vgt-aethel&limit=40&sort[]=date:desc&fields[include][]=title&fields[include][]=body-html&fields[include][]=date&fields[include][]=url&fields[include][]=country&fields[include][]=source"
	publicJSONLimit = 8 << 20
)

type cisaKEVConnector struct{}

func (*cisaKEVConnector) Descriptor() connectors.Descriptor {
	return connectors.Descriptor{
		Name: "builtin-cisa-kev", Version: "1.0.0", SourceTypes: []string{"json", "vulnerability-catalog"},
		Permissions: []string{"network.fetch.public"}, PollingInterval: 6 * time.Hour, RateLimitPerMin: 2,
		Regions: []string{"global"}, LicenseInfo: "CISA Known Exploited Vulnerabilities catalog",
		TrustTier: connectors.TrustBuiltIn, Activated: true,
		Policy: connectors.PublicOSINTPolicy("CISA-public-information", "https://www.cisa.gov/about/website-policies", 2, time.Minute, 365*24*time.Hour),
	}
}

func (*cisaKEVConnector) HealthCheck() error { return nil }
func (*cisaKEVConnector) Fetch() ([]intelligence.Observation, error) {
	body, fetchedAt, err := fetchBoundedPublicJSON(cisaKEVURL)
	if err != nil {
		return nil, err
	}
	return parseCISAKEV(body, fetchedAt)
}

type reliefWebConnector struct{}

func (*reliefWebConnector) Descriptor() connectors.Descriptor {
	return connectors.Descriptor{
		Name: "builtin-reliefweb", Version: "1.0.0", SourceTypes: []string{"json", "humanitarian-reports"},
		Permissions: []string{"network.fetch.public"}, PollingInterval: 30 * time.Minute, RateLimitPerMin: 4,
		Regions: []string{"global"}, LicenseInfo: "UN OCHA ReliefWeb API; source attribution retained",
		TrustTier: connectors.TrustBuiltIn, Activated: true,
		Policy: connectors.PublicOSINTPolicy("ReliefWeb-API-terms", "https://reliefweb.int/terms-conditions", 4, time.Minute, 365*24*time.Hour),
	}
}

func (*reliefWebConnector) HealthCheck() error { return nil }
func (*reliefWebConnector) Fetch() ([]intelligence.Observation, error) {
	body, fetchedAt, err := fetchBoundedPublicJSON(reliefWebURL)
	if err != nil {
		return nil, err
	}
	return parseReliefWeb(body, fetchedAt)
}

func fetchBoundedPublicJSON(rawURL string) ([]byte, time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return fetchBoundedPublicJSONContext(ctx, rawURL)
}

func fetchBoundedPublicJSONContext(ctx context.Context, rawURL string) ([]byte, time.Time, error) {
	if err := validatePublicCollectorURL(rawURL); err != nil {
		return nil, time.Time{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, time.Time{}, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "VGT-AETHEL-OSINT/2.0")
	response, err := newSafeCollectorHTTPClient().Do(request)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, time.Time{}, fmt.Errorf("public intelligence endpoint returned status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, publicJSONLimit+1))
	if err != nil {
		return nil, time.Time{}, err
	}
	if len(body) > publicJSONLimit {
		return nil, time.Time{}, errors.New("public intelligence response exceeds size boundary")
	}
	return body, time.Now().UTC(), nil
}

func parseCISAKEV(body []byte, fetchedAt time.Time) ([]intelligence.Observation, error) {
	var catalog struct {
		Vulnerabilities []struct {
			CVEID              string `json:"cveID"`
			VendorProject      string `json:"vendorProject"`
			Product            string `json:"product"`
			VulnerabilityName  string `json:"vulnerabilityName"`
			DateAdded          string `json:"dateAdded"`
			ShortDescription   string `json:"shortDescription"`
			RequiredAction     string `json:"requiredAction"`
			KnownRansomwareUse string `json:"knownRansomwareCampaignUse"`
		} `json:"vulnerabilities"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	if err := decoder.Decode(&catalog); err != nil {
		return nil, errors.New("CISA KEV payload is invalid")
	}
	limit := minInt(100, len(catalog.Vulnerabilities))
	result := make([]intelligence.Observation, 0, limit)
	for index := 0; index < limit; index++ {
		item := catalog.Vulnerabilities[index]
		if !strings.HasPrefix(item.CVEID, "CVE-") || strings.TrimSpace(item.VulnerabilityName) == "" {
			continue
		}
		publishedAt, _ := time.Parse("2006-01-02", item.DateAdded)
		raw := strings.Join([]string{item.CVEID, item.VendorProject, item.Product, item.VulnerabilityName, item.ShortDescription, item.RequiredAction, "ransomware=" + item.KnownRansomwareUse}, " | ")
		result = append(result, intelligence.Observation{
			ID: "cisa-kev-" + strings.ToLower(item.CVEID), SourceID: "cisa-kev", RawText: raw,
			ObservedAt: publishedAt, PublishedAt: publishedAt, FetchedAt: fetchedAt,
			OriginalURL: cisaKEVURL, FinalURL: "https://www.cisa.gov/known-exploited-vulnerabilities-catalog",
			Domain: "cyber", MIMEType: "application/json", ParserVersion: "cisa-kev-v1",
		})
	}
	return result, nil
}

func parseReliefWeb(body []byte, fetchedAt time.Time) ([]intelligence.Observation, error) {
	var response struct {
		Data []struct {
			ID     int `json:"id"`
			Fields struct {
				Title string `json:"title"`
				Body  string `json:"body-html"`
				URL   string `json:"url"`
				Date  struct {
					Created string `json:"created"`
				} `json:"date"`
				Countries []struct {
					Name string `json:"name"`
				} `json:"country"`
				Sources []struct {
					Name string `json:"name"`
				} `json:"source"`
			} `json:"fields"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, errors.New("ReliefWeb payload is invalid")
	}
	result := make([]intelligence.Observation, 0, len(response.Data))
	for _, item := range response.Data {
		if item.ID <= 0 || strings.TrimSpace(item.Fields.Title) == "" {
			continue
		}
		publishedAt, _ := time.Parse(time.RFC3339, item.Fields.Date.Created)
		countries, sources := make([]string, 0, len(item.Fields.Countries)), make([]string, 0, len(item.Fields.Sources))
		for _, country := range item.Fields.Countries {
			countries = append(countries, country.Name)
		}
		for _, source := range item.Fields.Sources {
			sources = append(sources, source.Name)
		}
		raw := strings.TrimSpace(item.Fields.Title + " | countries=" + strings.Join(countries, ", ") + " | sources=" + strings.Join(sources, ", ") + " | " + stripHTML(item.Fields.Body))
		result = append(result, intelligence.Observation{
			ID: fmt.Sprintf("reliefweb-%d", item.ID), SourceID: "reliefweb", RawText: raw,
			ObservedAt: publishedAt, PublishedAt: publishedAt, FetchedAt: fetchedAt,
			OriginalURL: reliefWebURL, FinalURL: item.Fields.URL, Domain: "humanitarian",
			MIMEType: "application/json", ParserVersion: "reliefweb-v1",
		})
	}
	return result, nil
}
