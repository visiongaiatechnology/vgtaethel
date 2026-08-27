package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"time"
)

type WebsiteMonitor struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	URL                 string    `json:"url"`
	SourceID            string    `json:"source_id"`
	Domain              string    `json:"domain"`
	IntervalMinutes     int       `json:"interval_minutes"`
	LicenseID           string    `json:"license_id"`
	TermsURL            string    `json:"terms_url,omitempty"`
	AllowedUse          string    `json:"allowed_use"`
	RetentionDays       int       `json:"retention_days"`
	Classification      string    `json:"classification"`
	AuthenticationMode  string    `json:"authentication_mode"`
	Geography           string    `json:"geography"`
	Redistribution      string    `json:"redistribution"`
	Enabled             bool      `json:"enabled"`
	LastRawSHA256       string    `json:"last_raw_sha256,omitempty"`
	LastSnapshotID      string    `json:"last_snapshot_id,omitempty"`
	LastHTTPStatus      int       `json:"last_http_status,omitempty"`
	LastETag            string    `json:"last_etag,omitempty"`
	LastModified        string    `json:"last_modified,omitempty"`
	LastCheckedAt       time.Time `json:"last_checked_at,omitempty"`
	LastChangedAt       time.Time `json:"last_changed_at,omitempty"`
	NextCheckAt         time.Time `json:"next_check_at"`
	CreatedAt           time.Time `json:"created_at"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastError           string    `json:"last_error,omitempty"`
}

type WebsiteChange struct {
	ID             string    `json:"id"`
	MonitorID      string    `json:"monitor_id"`
	PreviousSHA256 string    `json:"previous_sha256"`
	CurrentSHA256  string    `json:"current_sha256"`
	SnapshotID     string    `json:"snapshot_id"`
	ObservationIDs []string  `json:"observation_ids"`
	DetectedAt     time.Time `json:"detected_at"`
}

func (s *Store) AddWebsiteMonitor(candidate WebsiteMonitor) (WebsiteMonitor, error) {
	candidate.Name, candidate.URL, candidate.SourceID, candidate.Domain = strings.TrimSpace(candidate.Name), strings.TrimSpace(candidate.URL), strings.TrimSpace(candidate.SourceID), strings.TrimSpace(candidate.Domain)
	parsed, err := url.Parse(candidate.URL)
	host := strings.ToLower(parsed.Hostname())
	if err != nil || parsed.Scheme != "https" || host == "" || parsed.User != nil || net.ParseIP(host) != nil || host == "localhost" || strings.HasSuffix(host, ".localhost") || candidate.Name == "" || candidate.SourceID == "" {
		return WebsiteMonitor{}, errors.New("website monitor metadata is invalid")
	}
	if candidate.IntervalMinutes < 5 || candidate.IntervalMinutes > 10080 || candidate.RetentionDays < 1 || candidate.RetentionDays > 3650 {
		return WebsiteMonitor{}, errors.New("website monitor schedule or retention is outside policy")
	}
	if candidate.TermsURL != "" {
		terms, termsErr := url.Parse(candidate.TermsURL)
		if termsErr != nil || terms.Scheme != "https" || terms.Hostname() == "" {
			return WebsiteMonitor{}, errors.New("website monitor terms URL is invalid")
		}
	}
	if candidate.LicenseID == "" || candidate.AllowedUse == "" || candidate.AuthenticationMode != "none" || candidate.Geography == "" || candidate.Redistribution == "" || (candidate.Classification != "public" && candidate.Classification != "restricted") {
		return WebsiteMonitor{}, errors.New("website monitor machine policy is incomplete")
	}
	id, err := newIntelID("web-monitor")
	if err != nil {
		return WebsiteMonitor{}, err
	}
	now := time.Now().UTC()
	candidate.ID, candidate.Enabled, candidate.CreatedAt, candidate.NextCheckAt = id, true, now, now
	candidate.LastRawSHA256, candidate.LastSnapshotID, candidate.LastETag, candidate.LastModified, candidate.LastError = "", "", "", "", ""
	candidate.LastHTTPStatus, candidate.ConsecutiveFailures = 0, 0
	candidate.LastCheckedAt, candidate.LastChangedAt = time.Time{}, time.Time{}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.state.WebsiteMonitors {
		if strings.EqualFold(existing.URL, candidate.URL) {
			return WebsiteMonitor{}, errors.New("website monitor URL already exists")
		}
	}
	s.state.WebsiteMonitors = append(s.state.WebsiteMonitors, candidate)
	sourceExists := false
	for _, source := range s.state.Sources {
		sourceExists = sourceExists || source.ID == candidate.SourceID
	}
	if !sourceExists {
		s.state.Sources = append(s.state.Sources, Source{ID: candidate.SourceID, Name: candidate.Name, URL: candidate.URL, OriginalURL: candidate.URL, SourceType: "website-monitor", Publisher: parsed.Hostname(), TrustTier: 2, PermissionStatus: candidate.LicenseID, Region: candidate.Geography, AvailabilityStatus: "pending", ParserVersion: "website-monitor-v1"})
	}
	s.state.Audits = append(s.state.Audits, AuditEvent{At: now, Action: "website-monitor.created", Actor: "operator", Detail: candidate.ID})
	if err := s.save(); err != nil {
		return WebsiteMonitor{}, err
	}
	return candidate, nil
}

func (s *Store) GetWebsiteMonitor(id string) (WebsiteMonitor, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, monitor := range s.state.WebsiteMonitors {
		if monitor.ID == id {
			return monitor, true
		}
	}
	return WebsiteMonitor{}, false
}

func (s *Store) RecordWebsiteMonitorSuccess(id string, imported ImportedDocument, status int, etag, modified string) (WebsiteMonitor, *WebsiteChange, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for index := range s.state.WebsiteMonitors {
		monitor := &s.state.WebsiteMonitors[index]
		if monitor.ID != id {
			continue
		}
		previous := monitor.LastRawSHA256
		monitor.LastRawSHA256, monitor.LastSnapshotID, monitor.LastHTTPStatus = imported.RawSHA256, imported.SnapshotID, status
		monitor.LastETag, monitor.LastModified, monitor.LastCheckedAt = etag, modified, now
		monitor.NextCheckAt = now.Add(time.Duration(monitor.IntervalMinutes) * time.Minute)
		monitor.ConsecutiveFailures, monitor.LastError = 0, ""
		for sourceIndex := range s.state.Sources {
			if s.state.Sources[sourceIndex].ID == monitor.SourceID {
				s.state.Sources[sourceIndex].FetchedAt = now
				s.state.Sources[sourceIndex].ContentHash = imported.RawSHA256
				s.state.Sources[sourceIndex].AvailabilityStatus = "ok"
				break
			}
		}
		var change *WebsiteChange
		if previous != "" && previous != imported.RawSHA256 {
			entryID, idErr := newIntelID("web-change")
			if idErr != nil {
				return WebsiteMonitor{}, nil, idErr
			}
			entry := WebsiteChange{ID: entryID, MonitorID: id, PreviousSHA256: previous, CurrentSHA256: imported.RawSHA256, SnapshotID: imported.SnapshotID, ObservationIDs: append([]string(nil), imported.ObservationIDs...), DetectedAt: now}
			s.state.WebsiteChanges = append(s.state.WebsiteChanges, entry)
			monitor.LastChangedAt, change = now, &entry
			s.publish("website.changed", entry.ID, entry)
		}
		if err := s.save(); err != nil {
			return WebsiteMonitor{}, nil, err
		}
		return *monitor, change, nil
	}
	return WebsiteMonitor{}, nil, errors.New("website monitor not found")
}

func (s *Store) RecordWebsiteMonitorFailure(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for index := range s.state.WebsiteMonitors {
		monitor := &s.state.WebsiteMonitors[index]
		if monitor.ID != id {
			continue
		}
		monitor.ConsecutiveFailures++
		monitor.LastCheckedAt = now
		monitor.NextCheckAt = now.Add(time.Duration(1<<min(monitor.ConsecutiveFailures-1, 6)) * time.Minute)
		monitor.LastError = "collection failed"
		for sourceIndex := range s.state.Sources {
			if s.state.Sources[sourceIndex].ID == monitor.SourceID {
				s.state.Sources[sourceIndex].FetchedAt = now
				s.state.Sources[sourceIndex].AvailabilityStatus = "degraded"
				break
			}
		}
		s.state.Audits = append(s.state.Audits, AuditEvent{At: now, Action: "website-monitor.failed", Actor: "collector", Detail: id})
		return s.save()
	}
	return errors.New("website monitor not found")
}
