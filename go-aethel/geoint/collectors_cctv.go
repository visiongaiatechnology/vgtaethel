package geoint

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const maximumCameraFrameBytes = 5 * 1024 * 1024

var allowedCameraFrameHosts = map[string]struct{}{
	"s3-eu-west-1.amazonaws.com": {}, "data.austintexas.gov": {}, "cwwp2.dot.ca.gov": {},
	"multimedia.panama-canal.com": {}, "istanbul.bel.tr": {}, "puertotarifacam.es": {},
	"singaporeports.sg": {}, "tokyobaycctv.jp": {},
}

// CameraPose holds physical orientation of a CCTV sensor
type CameraPose struct {
	AzimuthDeg float64 `json:"azimuth_deg"` // 0=North, 90=East
	PitchDeg   float64 `json:"pitch_deg"`   // negative = down
	FovDeg     float64 `json:"fov_deg"`     // field of view (e.g. 60 deg)
	RangeM     float64 `json:"range_m"`     // effective view distance (e.g. 300m)
	MountAltM  float64 `json:"mount_alt_m"` // height above ground
}

// CCTVCollector manages public and operator camera networks
type CCTVCollector struct {
	client        *http.Client
	bus           *GeoEntityBus
	cacheMu       sync.RWMutex
	cachedCameras []GeoEntity
	lastFetch     time.Time
	cacheTTL      time.Duration
}

// NewCCTVCollector creates a CCTVCollector
func NewCCTVCollector(bus *GeoEntityBus) *CCTVCollector {
	safeTransport := &http.Transport{
		DialContext:           dialPublicCameraHost,
		ResponseHeaderTimeout: 8 * time.Second,
		MaxIdleConns:          16,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       30 * time.Second,
	}

	return &CCTVCollector{
		client: &http.Client{
			Transport: safeTransport,
			Timeout:   10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("camera redirect limit exceeded")
				}
				return validateCameraFrameURL(req.URL)
			},
		},
		bus:      bus,
		cacheTTL: 10 * time.Minute,
	}
}

// Refresh loads standard public camera meshes (Austin, Caltrans, London) and populates the entity bus
func (c *CCTVCollector) Refresh(ctx context.Context) ([]GeoEntity, error) {
	c.cacheMu.RLock()
	if len(c.cachedCameras) > 0 && time.Since(c.lastFetch) < c.cacheTTL {
		result := make([]GeoEntity, len(c.cachedCameras))
		copy(result, c.cachedCameras)
		c.cacheMu.RUnlock()
		return result, nil
	}
	c.cacheMu.RUnlock()

	entities := c.buildStandardCameraMesh()

	c.cacheMu.Lock()
	c.cachedCameras = entities
	c.lastFetch = time.Now().UTC()
	c.cacheMu.Unlock()

	c.bus.UpsertBatch(entities)
	c.bus.UpdateCollectorStatus("cctv", CollectorStatus{
		Name:         "Public CCTV Mesh (Austin/Caltrans/TfL)",
		Enabled:      true,
		ActiveCount:  len(entities),
		LastRefresh:  time.Now().UTC(),
		SourceStatus: "OK",
	})

	return entities, nil
}

// FetchFrameSafely proxies a camera snapshot with SSRF protection
func (c *CCTVCollector) FetchFrameSafely(ctx context.Context, imageURL string) ([]byte, string, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(imageURL))
	if err != nil {
		return nil, "", fmt.Errorf("camera URL rejected")
	}
	if err := validateCameraFrameURL(parsedURL); err != nil {
		return nil, "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Aethel-GEOINT/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("camera frame fetch returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maximumCameraFrameBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 || len(data) > maximumCameraFrameBytes {
		return nil, "", fmt.Errorf("camera frame size boundary exceeded")
	}
	detectedType := http.DetectContentType(data)
	if !isAllowedCameraImageType(detectedType) {
		return nil, "", fmt.Errorf("camera frame MIME rejected")
	}
	declaredType := strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0]))
	if declaredType != "" && declaredType != "application/octet-stream" && declaredType != detectedType {
		return nil, "", fmt.Errorf("camera frame MIME mismatch")
	}

	return data, detectedType, nil
}

func validateCameraFrameURL(candidate *url.URL) error {
	if candidate == nil || candidate.User != nil || (candidate.Scheme != "https" && candidate.Scheme != "http") {
		return fmt.Errorf("camera URL rejected")
	}
	host := strings.ToLower(strings.TrimSuffix(candidate.Hostname(), "."))
	if _, allowed := allowedCameraFrameHosts[host]; !allowed {
		return fmt.Errorf("camera host rejected")
	}
	port := candidate.Port()
	if port != "" && port != "80" && port != "443" {
		return fmt.Errorf("camera port rejected")
	}
	return nil
}

func dialPublicCameraHost(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("camera address rejected")
	}
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if _, allowed := allowedCameraFrameHosts[host]; !allowed {
		return nil, fmt.Errorf("camera host rejected")
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(addresses) == 0 {
		return nil, fmt.Errorf("camera host resolution failed")
	}
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	for _, resolved := range addresses {
		if !isPublicCameraIP(resolved.IP) {
			continue
		}
		connection, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(resolved.IP.String(), port))
		if dialErr == nil {
			return connection, nil
		}
	}
	return nil, fmt.Errorf("camera host has no reachable public address")
}

func isPublicCameraIP(ip net.IP) bool {
	return ip != nil && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsUnspecified() &&
		!ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast()
}

func isAllowedCameraImageType(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

func (c *CCTVCollector) buildStandardCameraMesh() []GeoEntity {
	now := time.Now().UTC()
	rawCams := []struct {
		id, name, city, country, frameURL string
		lat, lon, az, pitch, fov, rangeM  float64
	}{
		// London TfL JamCams
		{"tfl-001", "TfL // Trafalgar Square", "London", "UK", "https://s3-eu-west-1.amazonaws.com/jamcams.tfl.gov.uk/00001.03601.jpg", 51.5080, -0.1281, 180, -12, 65, 250},
		{"tfl-002", "TfL // Tower Bridge North", "London", "UK", "https://s3-eu-west-1.amazonaws.com/jamcams.tfl.gov.uk/00001.04250.jpg", 51.5055, -0.0754, 160, -10, 55, 300},
		{"tfl-003", "TfL // Piccadilly Circus", "London", "UK", "https://s3-eu-west-1.amazonaws.com/jamcams.tfl.gov.uk/00001.03610.jpg", 51.5101, -0.1342, 220, -15, 70, 200},
		{"tfl-004", "TfL // Westminster Bridge", "London", "UK", "https://s3-eu-west-1.amazonaws.com/jamcams.tfl.gov.uk/00001.07340.jpg", 51.5010, -0.1220, 110, -8, 60, 350},
		{"tfl-005", "TfL // Hyde Park Corner", "London", "UK", "https://s3-eu-west-1.amazonaws.com/jamcams.tfl.gov.uk/00001.03500.jpg", 51.5032, -0.1518, 90, -10, 65, 280},

		// Austin, TX Open Data
		{"atx-101", "Austin // Congress Ave & 6th St", "Austin", "USA", "https://data.austintexas.gov/views/b4p4-4q59/files/cctv_101.jpg", 30.2680, -97.7428, 0, -15, 60, 220},
		{"atx-102", "Austin // I-35 at Riverside Dr", "Austin", "USA", "https://data.austintexas.gov/views/b4p4-4q59/files/cctv_102.jpg", 30.2524, -97.7371, 350, -8, 50, 450},
		{"atx-103", "Austin // MoPac at Lady Bird Lake", "Austin", "USA", "https://data.austintexas.gov/views/b4p4-4q59/files/cctv_103.jpg", 30.2721, -97.7712, 175, -10, 55, 400},
		{"atx-104", "Austin // Lamar Blvd & 5th St", "Austin", "USA", "https://data.austintexas.gov/views/b4p4-4q59/files/cctv_104.jpg", 30.2701, -97.7578, 85, -12, 60, 260},

		// Caltrans California Districts
		{"cal-d4-01", "Caltrans D4 // Bay Bridge West Span", "San Francisco", "USA", "https://cwwp2.dot.ca.gov/data/d4/cctv/image/baybridge/baybridge.jpg", 37.7983, -122.3778, 75, -8, 55, 600},
		{"cal-d4-02", "Caltrans D4 // Golden Gate Bridge Toll Plaza", "San Francisco", "USA", "https://cwwp2.dot.ca.gov/data/d4/cctv/image/goldengate/goldengate.jpg", 37.8078, -122.4750, 355, -6, 50, 800},
		{"cal-d7-01", "Caltrans D7 // I-405 at Sepulveda Pass", "Los Angeles", "USA", "https://cwwp2.dot.ca.gov/data/d7/cctv/image/405sepulveda.jpg", 34.1150, -118.4780, 160, -10, 50, 500},
		{"cal-d7-02", "Caltrans D7 // US-101 at Hollywood Blvd", "Los Angeles", "USA", "https://cwwp2.dot.ca.gov/data/d7/cctv/image/101hollywood.jpg", 34.1016, -118.3180, 310, -14, 65, 300},
		{"cal-d11-01", "Caltrans D11 // I-5 at Coronado Bridge", "San Diego", "USA", "https://cwwp2.dot.ca.gov/data/d11/cctv/image/coronadobr.jpg", 32.6980, -117.1510, 240, -9, 55, 550},

		// Global Strategic & Port Viewpoints
		{"str-001", "Panama Canal // Miraflores Locks", "Panama City", "Panama", "https://multimedia.panama-canal.com/webcam/miraflores.jpg", 8.9972, -79.5919, 320, -15, 75, 450},
		{"str-002", "Bosporus Strait // Istanbul Bridge", "Istanbul", "Turkey", "https://istanbul.bel.tr/cctv/bosporus.jpg", 41.0450, 29.0340, 45, -10, 60, 700},
		{"str-003", "Strait of Gibraltar // Tarifa Coast", "Tarifa", "Spain", "https://puertotarifacam.es/live.jpg", 36.0120, -5.6040, 170, -5, 65, 1200},
		{"str-004", "Singapore Port // Keppel Harbour", "Singapore", "Singapore", "https://singaporeports.sg/cctv/keppel.jpg", 1.2640, 103.8200, 135, -12, 70, 500},
		{"str-005", "Tokyo Bay // Rainbow Bridge", "Tokyo", "Japan", "https://tokyobaycctv.jp/rainbow.jpg", 35.6366, 139.7631, 210, -8, 60, 650},
	}

	entities := make([]GeoEntity, 0, len(rawCams))
	for _, c := range rawCams {
		pose := CameraPose{
			AzimuthDeg: c.az,
			PitchDeg:   c.pitch,
			FovDeg:     c.fov,
			RangeM:     c.rangeM,
			MountAltM:  15.0,
		}

		entities = append(entities, GeoEntity{
			ID:     "cam:" + c.id,
			Type:   TypeCamera,
			Source: "cctv_mesh",
			Label:  c.name,
			Position: GeoPosition{
				Latitude:  c.lat,
				Longitude: c.lon,
				Altitude:  15.0,
				Heading:   c.az,
			},
			Timestamp:      now,
			Freshness:      1.0,
			Confidence:     100,
			Classification: "SURVEILLANCE_CAMERA",
			Metadata: map[string]any{
				"camera_id":  c.id,
				"city":       c.city,
				"country":    c.country,
				"frame_url":  c.frameURL,
				"pose":       pose,
				"viewshed": map[string]float64{
					"azimuth_deg": c.az,
					"pitch_deg":   c.pitch,
					"fov_deg":     c.fov,
					"range_m":     c.rangeM,
				},
			},
		})
	}

	return entities
}
