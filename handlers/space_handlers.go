// STATUS: DIAMANT VGT SUPREME
// Space weather backend — Solarcommander data parity (NOAA SWPC + NASA SDO/OVATION)
package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// sdoProxyCacheTTL keeps GSFC/NOAA image frames in process memory so repeated
// dashboard loads do not hammer upstream (blacklist risk). Manual refresh=1 bypasses.
const sdoProxyCacheTTL = 20 * time.Minute

// weatherCacheTTL: short cache for JSON telemetry so a single SYNC does not
// multi-hit every NOAA endpoint on accidental double-click.
const weatherCacheTTL = 45 * time.Second

type sdoCacheEntry struct {
	body        []byte
	contentType string
	fetchedAt   time.Time
	upstream    string
}

var (
	sdoCacheMu sync.Mutex
	sdoCache   = map[string]sdoCacheEntry{}

	weatherCacheMu   sync.Mutex
	weatherCacheAt   time.Time
	weatherCacheBody []byte
)

// SdoLatestBase is the Solarcommander-proven GSFC latest-asset root.
const SdoLatestBase = "https://sdo.gsfc.nasa.gov/assets/img/latest/"

// AllowedSdoChannels lists dashboard solar channels.
var AllowedSdoChannels = []string{"171", "193", "304", "131", "211", "HMI"}

// SeriesPoint is a compact telemetry sample for sparklines.
type SeriesPoint struct {
	T string  `json:"t"`
	V float64 `json:"v"`
}

// TelemetrySource makes the provenance of each live value explicit. The UI
// never has to present an unavailable upstream value as live telemetry.
type TelemetrySource struct {
	Available bool   `json:"available"`
	UpdatedAt string `json:"updated_at,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

// SpaceWeatherResponse is the full Solarcommander-class payload for the Aethel dashboard.
type SpaceWeatherResponse struct {
	Timestamp string `json:"timestamp"`

	KpIndex  float64 `json:"kp_index"`
	KpStatus string  `json:"kp_status"`
	GScale   int     `json:"g_scale"`
	RScale   int     `json:"r_scale"`
	SScale   int     `json:"s_scale"`

	SolarWindSpeed   float64 `json:"solar_wind_speed_km_s"`
	SolarWindDensity float64 `json:"solar_wind_density_p_cm3"`
	BtTotal          float64 `json:"bt_total_nt"`
	BzVector         float64 `json:"bz_vector_nt"`
	DstIndex         float64 `json:"dst_index_nt"`
	DstStatus        string  `json:"dst_status"`

	SolarXRayFlux   string   `json:"solar_xray_flux_class"`
	XRayFlux        float64  `json:"xray_flux"`
	ProtonFlux10MeV float64  `json:"proton_flux_10mev"`
	SunspotCount    int      `json:"sunspot_count"`
	FlareMax72h     string   `json:"flare_max_72h"`
	FlareLastClass  string   `json:"flare_last_class"`
	FlareLastTime   string   `json:"flare_last_time"`
	RecentFlares    []string `json:"recent_flares"`

	GeomagneticField  string  `json:"geomagnetic_field_status"`
	AuroraActivity    string  `json:"aurora_activity_level"`
	AuroraMinLat      string  `json:"aurora_min_lat"`
	AuroraConfidence  int     `json:"aurora_confidence"`
	AuroraHemispheric float64 `json:"aurora_hemispheric_power_gw"`
	AuroraNorthPower  float64 `json:"aurora_north_power_gw"`
	AuroraSouthPower  float64 `json:"aurora_south_power_gw"`

	// Forecast (deterministic Solarcommander-style)
	ProbMClass  int    `json:"prob_m_class"`
	ProbXClass  int    `json:"prob_x_class"`
	ProbProton  int    `json:"prob_proton"`
	GeoForecast string `json:"geo_forecast"`

	// Sparkline series (sampled)
	SeriesXRay   []SeriesPoint `json:"series_xray,omitempty"`
	SeriesMagBz  []SeriesPoint `json:"series_mag_bz,omitempty"`
	SeriesProton []SeriesPoint `json:"series_proton,omitempty"`
	SeriesWind   []SeriesPoint `json:"series_wind,omitempty"`
	SeriesKp     []SeriesPoint `json:"series_kp,omitempty"`
	SeriesDst    []SeriesPoint `json:"series_dst,omitempty"`

	Sources          []string                   `json:"sources"`
	TelemetrySources map[string]TelemetrySource `json:"telemetry_sources"`
}

// ResolveSdoImageURL maps a dashboard channel id to a working NASA SDO latest JPEG URL.
func ResolveSdoImageURL(channel string) (string, bool) {
	ch := strings.TrimSpace(channel)
	if ch == "" {
		ch = "193"
	}
	var file string
	switch strings.ToUpper(ch) {
	case "171":
		file = "latest_1024_0171.jpg"
	case "193":
		file = "latest_1024_0193.jpg"
	case "304":
		file = "latest_1024_0304.jpg"
	case "131":
		file = "latest_1024_0131.jpg"
	case "211":
		file = "latest_1024_0211.jpg"
	case "HMI":
		file = "latest_1024_HMII.jpg"
	default:
		return "", false
	}
	return SdoLatestBase + file, true
}

// ResolveSpaceImageURL maps solar channels + aurora OVATION maps (Solarcommander image set).
func ResolveSpaceImageURL(channel string) (string, bool) {
	ch := strings.TrimSpace(channel)
	switch strings.ToLower(ch) {
	case "aurora_n", "aurora-north", "north":
		return "https://services.swpc.noaa.gov/images/animations/ovation/north/latest.jpg", true
	case "aurora_s", "aurora-south", "south":
		return "https://services.swpc.noaa.gov/images/animations/ovation/south/latest.jpg", true
	default:
		return ResolveSdoImageURL(ch)
	}
}

func HandleSpaceWeather(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	force := r.URL.Query().Get("refresh") == "1"
	if !force {
		weatherCacheMu.Lock()
		if len(weatherCacheBody) > 0 && time.Since(weatherCacheAt) < weatherCacheTTL {
			body := append([]byte(nil), weatherCacheBody...)
			weatherCacheMu.Unlock()
			w.Header().Set("X-Weather-Cache", "HIT")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
			return
		}
		weatherCacheMu.Unlock()
	}

	resp := buildSpaceWeatherLive()
	payload, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "encode failed", http.StatusInternalServerError)
		return
	}
	weatherCacheMu.Lock()
	weatherCacheBody = append([]byte(nil), payload...)
	weatherCacheAt = time.Now()
	weatherCacheMu.Unlock()

	w.Header().Set("X-Weather-Cache", "MISS")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

func buildSpaceWeatherLive() SpaceWeatherResponse {
	resp := SpaceWeatherResponse{
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
		KpIndex:           2.3,
		KpStatus:          "RUHIG",
		SolarWindSpeed:    400,
		SolarWindDensity:  5,
		SolarXRayFlux:     "B1.0",
		BtTotal:           5,
		BzVector:          0,
		DstIndex:          -10,
		DstStatus:         "QUIET",
		RScale:            0,
		SScale:            0,
		GScale:            0,
		SunspotCount:      80,
		GeomagneticField:  "STABIL",
		AuroraActivity:    "RUHIG",
		AuroraMinLat:      "65° N",
		AuroraConfidence:  40,
		AuroraHemispheric: 12,
		AuroraNorthPower:  12,
		AuroraSouthPower:  12,
		FlareMax72h:       "—",
		FlareLastClass:    "—",
		FlareLastTime:     "—",
		RecentFlares:      []string{},
		ProbMClass:        15,
		ProbXClass:        5,
		ProbProton:        3,
		GeoForecast:       "STABLE",
		Sources: []string{
			"NOAA SWPC Kp", "NOAA solar-wind plasma/mag", "GOES X-ray", "GOES protons",
			"Kyoto Dst", "SWPC solar cycle SSN", "NASA SDO", "OVATION aurora",
		},
		TelemetrySources: map[string]TelemetrySource{
			"kp":       {Detail: "NOAA planetary Kp"},
			"wind":     {Detail: "NOAA solar-wind plasma"},
			"imf":      {Detail: "NOAA solar-wind magnetometer"},
			"xray":     {Detail: "GOES X-ray"},
			"proton":   {Detail: "GOES protons"},
			"dst":      {Detail: "Kyoto Dst"},
			"sunspots": {Detail: "SWPC solar-cycle index"},
		},
	}

	client := &http.Client{Timeout: 10 * time.Second}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var hpiNorth, hpiSouth float64
	var hpiTime string
	var hpiOK bool

	// 1) Kp
	wg.Add(1)
	go func() {
		defer wg.Done()
		kp, series, measuredAt, ok := fetchLatestKp(client)
		if !ok {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		resp.KpIndex = kp
		resp.SeriesKp = series
		resp.TelemetrySources["kp"] = TelemetrySource{Available: true, UpdatedAt: measuredAt, Detail: "NOAA planetary Kp"}
		applyKpDerived(&resp, kp)
	}()

	// 2) Plasma + Mag
	wg.Add(1)
	go func() {
		defer wg.Done()
		speed, dens, windSeries, okP := fetchPlasma(client)
		bt, bz, bzSeries, okM := fetchMag(client)
		mu.Lock()
		defer mu.Unlock()
		if okP {
			resp.SolarWindSpeed = speed
			resp.SolarWindDensity = dens
			resp.SeriesWind = windSeries
			resp.TelemetrySources["wind"] = TelemetrySource{Available: true, UpdatedAt: latestSeriesTime(windSeries), Detail: "NOAA solar-wind plasma"}
		}
		if okM {
			resp.BtTotal = bt
			resp.BzVector = bz
			resp.SeriesMagBz = bzSeries
			resp.TelemetrySources["imf"] = TelemetrySource{Available: true, UpdatedAt: latestSeriesTime(bzSeries), Detail: "NOAA solar-wind magnetometer"}
		}
	}()

	// 3) X-ray + protons + flare history
	wg.Add(1)
	go func() {
		defer wg.Done()
		xrayClass, flux, maxClass, lastClass, lastTime, series, flares, ok := fetchXrayAndFlares(client)
		pFlux, pSeries, okP := fetchProtons(client)
		mu.Lock()
		defer mu.Unlock()
		if ok {
			resp.SolarXRayFlux = xrayClass
			resp.XRayFlux = flux
			resp.FlareMax72h = maxClass
			resp.FlareLastClass = lastClass
			resp.FlareLastTime = lastTime
			resp.SeriesXRay = series
			resp.RecentFlares = flares
			resp.RScale = rScaleFromFlux(flux)
			resp.TelemetrySources["xray"] = TelemetrySource{Available: true, UpdatedAt: latestSeriesTime(series), Detail: "GOES X-ray"}
		}
		if okP {
			resp.ProtonFlux10MeV = pFlux
			resp.SeriesProton = pSeries
			resp.SScale = sScaleFromProton(pFlux)
			resp.TelemetrySources["proton"] = TelemetrySource{Available: true, UpdatedAt: latestSeriesTime(pSeries), Detail: "GOES protons"}
		}
	}()

	// 4) SSN
	wg.Add(1)
	go func() {
		defer wg.Done()
		ssn, ok := fetchSSN(client)
		if !ok {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		resp.SunspotCount = ssn
		resp.TelemetrySources["sunspots"] = TelemetrySource{Available: true, Detail: "SWPC solar-cycle index"}
	}()

	// 5) Dst
	wg.Add(1)
	go func() {
		defer wg.Done()
		dst, series, measuredAt, ok := fetchDst(client)
		if !ok {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		resp.DstIndex = dst
		resp.SeriesDst = series
		resp.DstStatus = dstStatus(dst)
		resp.TelemetrySources["dst"] = TelemetrySource{Available: true, UpdatedAt: measuredAt, Detail: "Kyoto Dst"}
	}()

	// 6) OVATION HPI: this is NOAA's actual hemispheric power product. A
	// Kp-derived estimate is only a fallback and must never overwrite it.
	wg.Add(1)
	go func() {
		defer wg.Done()
		north, south, observedAt, ok := fetchAuroraHPI(client)
		if !ok {
			return
		}
		hpiNorth, hpiSouth, hpiTime, hpiOK = north, south, observedAt, true
	}()

	wg.Wait()
	if hpiOK {
		resp.AuroraNorthPower = hpiNorth
		resp.AuroraSouthPower = hpiSouth
		resp.AuroraHemispheric = hpiNorth
		resp.TelemetrySources["aurora_hpi"] = TelemetrySource{Available: true, UpdatedAt: hpiTime, Detail: "NOAA OVATION Hemispheric Power Index"}
	}
	applyForecast(&resp)
	return resp
}

func applyKpDerived(resp *SpaceWeatherResponse, kp float64) {
	g := 0
	status := "RUHIG"
	field := "STABIL / UNGESTÖRT"
	aurora := "RUHIG"
	minLat := "65° N (Skandinavien/Kanada)"
	conf := 40
	if kp >= 4 {
		status = "AKTIV"
		field = "LEICHT GESTÖRT"
		aurora = "AKTIV"
		minLat = "60° N"
		conf = 60
	}
	if kp >= 5 {
		g = 1
		status = "STURM (G1)"
		field = "GESTÖRT"
		aurora = "STURM"
		minLat = "55° N (Norddeutschland/UK)"
		conf = 78
	}
	if kp >= 6 {
		g = 2
		status = "STURM (G2)"
		minLat = "52° N"
		conf = 85
	}
	if kp >= 7 {
		g = 3
		status = "STURM (G3)"
		field = "STARK GESTÖRT"
		aurora = "STARK"
		minLat = "48° N (Mitteleuropa)"
		conf = 92
	}
	if kp >= 8 {
		g = 4
		status = "STURM (G4)"
		aurora = "EXTREM"
		conf = 96
	}
	if kp >= 9 {
		g = 5
		status = "EXTREM (G5)"
		conf = 99
	}
	resp.GScale = g
	resp.KpStatus = status
	resp.GeomagneticField = field
	resp.AuroraActivity = aurora
	resp.AuroraMinLat = minLat
	resp.AuroraConfidence = conf
	// Hemispheric power estimate (Solarcommander: 5 + kp^2.5, cap 150)
	gw := 5 + math.Pow(kp, 2.5)
	if gw > 150 {
		gw = 150
	}
	resp.AuroraHemispheric = math.Round(gw*10) / 10
	resp.AuroraNorthPower = resp.AuroraHemispheric
	resp.AuroraSouthPower = resp.AuroraHemispheric
}

func applyForecast(resp *SpaceWeatherResponse) {
	ssn := resp.SunspotCount
	if ssn <= 0 {
		ssn = 80
	}
	// Deterministic Solarcommander-style probabilities (no random).
	m := int(math.Min(math.Round(float64(ssn)/3.5), 90))
	x := int(math.Min(math.Round(float64(ssn)/10), 40))
	p := int(math.Min(math.Round(float64(ssn)/15), 25))
	if m < 5 {
		m = 5
	}
	if x < 1 {
		x = 1
	}
	if p < 1 {
		p = 1
	}
	resp.ProbMClass = m
	resp.ProbXClass = x
	resp.ProbProton = p
	fc := "STABLE"
	if resp.KpIndex >= 4 {
		fc = "UNSETTLED"
	}
	if resp.KpIndex >= 5 {
		fc = "STORM RISK"
	}
	resp.GeoForecast = fc
}

func rScaleFromFlux(flux float64) int {
	// R1 ~ M1 (1e-5), R2 M5, R3 X1 (1e-4), R4 X10, R5 X20
	if flux >= 2e-3 {
		return 5
	}
	if flux >= 1e-3 {
		return 4
	}
	if flux >= 1e-4 {
		return 3
	}
	if flux >= 5e-5 {
		return 2
	}
	if flux >= 1e-5 {
		return 1
	}
	return 0
}

func sScaleFromProton(flux float64) int {
	// S1 10 pfu, S2 100, S3 1000, S4 10000, S5 100000
	if flux >= 1e5 {
		return 5
	}
	if flux >= 1e4 {
		return 4
	}
	if flux >= 1e3 {
		return 3
	}
	if flux >= 100 {
		return 2
	}
	if flux >= 10 {
		return 1
	}
	return 0
}

func dstStatus(dst float64) string {
	if dst <= -200 {
		return "SUPERSTORM"
	}
	if dst <= -100 {
		return "STORM"
	}
	if dst <= -50 {
		return "MODERATE"
	}
	if dst <= -30 {
		return "UNSETTLED"
	}
	return "QUIET"
}

func fetchLatestKp(client *http.Client) (float64, []SeriesPoint, string, bool) {
	// GFZ publishes the operational three-hourly Kp nowcast. This is the
	// standard planetary Kp users expect; NOAA's 1-minute stream is an input
	// series which legitimately restarts at 0 at a new three-hour window.
	if resp, err := client.Get("https://www-app3.gfz-potsdam.de/kp_index/Kp_ap_nowcast.txt"); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			series := make([]SeriesPoint, 0, 64)
			scanner := bufio.NewScanner(io.LimitReader(resp.Body, 512<<10))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				fields := strings.Fields(line)
				if len(fields) < 8 {
					continue
				}
				kp, parseErr := strconv.ParseFloat(fields[7], 64)
				if parseErr != nil || kp < 0 || kp > 9 {
					continue
				}
				series = append(series, SeriesPoint{T: fmt.Sprintf("%s-%s-%sT%s:00:00Z", fields[0], fields[1], fields[2], strings.TrimSuffix(fields[3], ".0")), V: kp})
			}
			if len(series) > 0 {
				series = tailSeries(series, 64)
				last := series[len(series)-1]
				return last.V, series, last.T, true
			}
		}
	}

	// NOAA has changed the representation of kp_index before (number vs. string).
	// Decode dynamically and validate the semantic range instead of silently
	// accepting a zero value from a partially decoded record.
	if resp, err := client.Get("https://services.swpc.noaa.gov/json/planetary_k_index_1m.json"); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var rows []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&rows); err == nil {
				series := make([]SeriesPoint, 0, minInt(len(rows), 120))
				for _, row := range rows {
					kp, ok := firstFiniteRange(row, 0, 9, "estimated_kp", "kp_index", "kp")
					if !ok {
						continue
					}
					series = append(series, SeriesPoint{T: firstString(row, "time_tag", "observed_time", "time"), V: kp})
				}
				if len(series) > 0 {
					series = tailSeries(series, 120)
					last := series[len(series)-1]
					return last.V, series, last.T, true
				}
			}
		}
	}

	// Fallback product is a table whose Kp column location must be derived from
	// the header, not assumed. The former fixed index caused false 0.0 readings.
	if resp, err := client.Get("https://services.swpc.noaa.gov/products/noaa-planetary-k-index.json"); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var rows [][]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&rows); err == nil && len(rows) > 1 {
				valueIdx := findHeaderIndex(rows[0], "kp_index", "kp", "estimated_kp")
				if valueIdx >= 0 {
					series := seriesFromTable(rows[1:], valueIdx, 120, 0, 9)
					if len(series) > 0 {
						last := series[len(series)-1]
						return last.V, series, last.T, true
					}
				}
			}
		}
	}
	return 0, nil, "", false
}

func fetchPlasma(client *http.Client) (speed, dens float64, series []SeriesPoint, ok bool) {
	if speed, dens, series, ok = fetchRTSWPlasma(client); ok {
		return speed, dens, series, true
	}
	// Legacy endpoint retained strictly as a compatibility fallback for older
	// SWPC mirrors. Current SWPC products moved to /json/rtsw in 2026.
	resp, err := client.Get("https://services.swpc.noaa.gov/products/solar-wind/plasma-2-hour.json")
	if err != nil {
		return 0, 0, nil, false
	}
	defer resp.Body.Close()
	var rows [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil || len(rows) < 2 {
		return 0, 0, nil, false
	}
	// SWPC plasma-2-hour: [time, density, speed, temperature]
	last := rows[len(rows)-1]
	dens = asFloat(last, 1)
	speed = asFloat(last, 2)
	series = sampleSeries(rows, 2, 0, 48) // wind speed sparkline
	return speed, dens, series, true
}

func fetchMag(client *http.Client) (bt, bz float64, series []SeriesPoint, ok bool) {
	if bt, bz, series, ok = fetchRTSWMag(client); ok {
		return bt, bz, series, true
	}
	// Compatibility fallback for pre-2026 SWPC product mirrors.
	resp, err := client.Get("https://services.swpc.noaa.gov/products/solar-wind/mag-2-hour.json")
	if err != nil {
		return 0, 0, nil, false
	}
	defer resp.Body.Close()
	var rows [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil || len(rows) < 2 {
		return 0, 0, nil, false
	}
	// Solarcommander: lastM[6]=Bt, lastM[3]=Bz
	last := rows[len(rows)-1]
	bz = asFloat(last, 3)
	bt = asFloat(last, 6)
	if bt == 0 && len(last) > 1 {
		// some products order differently — try last numeric fields
		bt = asFloat(last, len(last)-1)
	}
	series = sampleSeries(rows, 3, 3, 48)
	return bt, bz, series, true
}

func fetchRTSWPlasma(client *http.Client) (speed, dens float64, series []SeriesPoint, ok bool) {
	resp, err := client.Get("https://services.swpc.noaa.gov/json/rtsw/rtsw_wind_1m.json")
	if err != nil {
		return 0, 0, nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, nil, false
	}
	var rows []map[string]interface{}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 12<<20)).Decode(&rows); err != nil {
		return 0, 0, nil, false
	}
	for _, row := range rows {
		value, valueOK := firstFiniteRange(row, 150, 2500, "proton_speed", "speed")
		if !valueOK {
			continue
		}
		series = append(series, SeriesPoint{T: firstString(row, "time_tag", "time"), V: value})
	}
	series = tailSeries(series, 120)
	if len(series) == 0 {
		return 0, 0, nil, false
	}
	latest := rows[len(rows)-1]
	for index := len(rows) - 1; index >= 0; index-- {
		if _, found := firstFiniteRange(rows[index], 150, 2500, "proton_speed", "speed"); found {
			latest = rows[index]
			break
		}
	}
	wind, windOK := firstFiniteRange(latest, 150, 2500, "proton_speed", "speed")
	density, densityOK := firstFiniteRange(latest, 0, 10000, "proton_density", "density", "dens")
	if !windOK || !densityOK {
		return 0, 0, nil, false
	}
	return wind, density, series, true
}

func fetchRTSWMag(client *http.Client) (bt, bz float64, series []SeriesPoint, ok bool) {
	resp, err := client.Get("https://services.swpc.noaa.gov/json/rtsw/rtsw_mag_1m.json")
	if err != nil {
		return 0, 0, nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, nil, false
	}
	var rows []map[string]interface{}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 12<<20)).Decode(&rows); err != nil {
		return 0, 0, nil, false
	}
	var latest map[string]interface{}
	for _, row := range rows {
		value, valueOK := firstFiniteRange(row, -200, 200, "bz_gsm", "bz")
		if !valueOK {
			continue
		}
		series = append(series, SeriesPoint{T: firstString(row, "time_tag", "time"), V: value})
		latest = row
	}
	series = tailSeries(series, 120)
	if latest == nil {
		return 0, 0, nil, false
	}
	bzValue, bzOK := firstFiniteRange(latest, -200, 200, "bz_gsm", "bz")
	btValue, btOK := firstFiniteRange(latest, 0, 500, "bt", "bt_gsm", "total_field")
	if !bzOK || !btOK {
		return 0, 0, nil, false
	}
	return btValue, bzValue, series, true
}

func fetchXrayAndFlares(client *http.Client) (class string, flux float64, maxClass, lastClass, lastTime string, series []SeriesPoint, flares []string, ok bool) {
	resp, err := client.Get("https://services.swpc.noaa.gov/json/goes/primary/xrays-3-day.json")
	if err != nil {
		return "", 0, "", "", "", nil, nil, false
	}
	defer resp.Body.Close()
	var raw []struct {
		TimeTag string  `json:"time_tag"`
		Flux    float64 `json:"flux"`
		Energy  string  `json:"energy"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil || len(raw) == 0 {
		return "", 0, "", "", "", nil, nil, false
	}
	var data []struct {
		TimeTag string
		Flux    float64
	}
	for _, d := range raw {
		if d.Energy == "0.1-0.8nm" || d.Energy == "" {
			data = append(data, struct {
				TimeTag string
				Flux    float64
			}{d.TimeTag, d.Flux})
		}
	}
	if len(data) == 0 {
		for _, d := range raw {
			data = append(data, struct {
				TimeTag string
				Flux    float64
			}{d.TimeTag, d.Flux})
		}
	}
	latest := data[len(data)-1]
	flux = latest.Flux
	class = flareClassText(flux)
	maxFlux := 0.0
	for _, d := range data {
		if d.Flux > maxFlux {
			maxFlux = d.Flux
		}
	}
	maxClass = flareClassText(maxFlux)

	// Peak detection > C1.0
	lastClass = "NONE"
	lastTime = "—"
	flares = []string{}
	if len(data) > 2 {
		for i := len(data) - 2; i > 0; i-- {
			cur := data[i].Flux
			if cur > data[i-1].Flux && cur > data[i+1].Flux && cur >= 1e-6 {
				cls := flareClassText(cur)
				t := parseSWPCTime(data[i].TimeTag)
				lastClass = cls
				lastTime = t.Format("15:04 (02.01)")
				break
			}
		}
		// recent list of last few C+ peaks
		for i := len(data) - 2; i > 0 && len(flares) < 5; i-- {
			cur := data[i].Flux
			if cur > data[i-1].Flux && cur > data[i+1].Flux && cur >= 1e-6 {
				t := parseSWPCTime(data[i].TimeTag)
				flares = append(flares, fmt.Sprintf("%s · %s UTC", flareClassText(cur), t.Format("15:04 02.01")))
			}
		}
	}
	// series: last ~60 samples of log10 flux for charting
	step := 1
	if len(data) > 60 {
		step = len(data) / 60
	}
	for i := 0; i < len(data); i += step {
		d := data[i]
		v := d.Flux
		if v <= 0 {
			v = 1e-9
		}
		series = append(series, SeriesPoint{T: d.TimeTag, V: math.Log10(v)})
	}
	return class, flux, maxClass, lastClass, lastTime, series, flares, true
}

func fetchProtons(client *http.Client) (flux float64, series []SeriesPoint, ok bool) {
	resp, err := client.Get("https://services.swpc.noaa.gov/json/goes/primary/integral-protons-6-hour.json")
	if err != nil {
		return 0, nil, false
	}
	defer resp.Body.Close()
	var raw []struct {
		TimeTag string  `json:"time_tag"`
		Flux    float64 `json:"flux"`
		Energy  string  `json:"energy"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil || len(raw) == 0 {
		return 0, nil, false
	}
	var p10 []struct {
		TimeTag string
		Flux    float64
	}
	for _, d := range raw {
		if strings.Contains(d.Energy, "10 MeV") || d.Energy == ">=10 MeV" {
			p10 = append(p10, struct {
				TimeTag string
				Flux    float64
			}{d.TimeTag, d.Flux})
		}
	}
	if len(p10) == 0 {
		return 0, nil, false
	}
	flux = p10[len(p10)-1].Flux
	step := 1
	if len(p10) > 48 {
		step = len(p10) / 48
	}
	for i := 0; i < len(p10); i += step {
		series = append(series, SeriesPoint{T: p10[i].TimeTag, V: p10[i].Flux})
	}
	return flux, series, true
}

func fetchSSN(client *http.Client) (int, bool) {
	resp, err := client.Get("https://services.swpc.noaa.gov/json/solar-cycle/observed-solar-cycle-indices.json")
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	var rows []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil || len(rows) == 0 {
		return 0, false
	}
	last := rows[len(rows)-1]
	for _, key := range []string{"ssn", "observed_swpc_ssn", "smoothed_ssn"} {
		if v, ok := last[key]; ok {
			switch n := v.(type) {
			case float64:
				return int(math.Round(n)), true
			case string:
				if f, err := strconv.ParseFloat(n, 64); err == nil {
					return int(math.Round(f)), true
				}
			}
		}
	}
	return 0, false
}

func fetchAuroraHPI(client *http.Client) (north, south float64, observedAt string, ok bool) {
	resp, err := client.Get("https://services.swpc.noaa.gov/text/aurora-nowcast-hemi-power.txt")
	if err != nil {
		return 0, 0, "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, "", false
	}

	var latestNorth, latestSouth float64
	var latestTime string
	found := false
	scanner := bufio.NewScanner(io.LimitReader(resp.Body, 512<<10))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		// observation, forecast, north HPI, south HPI
		if len(fields) < 4 {
			continue
		}
		northValue, northErr := strconv.ParseFloat(fields[2], 64)
		southValue, southErr := strconv.ParseFloat(fields[3], 64)
		if northErr != nil || southErr != nil || northValue < 0 || southValue < 0 || northValue > 1000 || southValue > 1000 {
			continue
		}
		latestNorth, latestSouth, latestTime, found = northValue, southValue, fields[0], true
	}
	if !found || scanner.Err() != nil {
		return 0, 0, "", false
	}
	return latestNorth, latestSouth, latestTime, true
}

func fetchDst(client *http.Client) (float64, []SeriesPoint, string, bool) {
	resp, err := client.Get("https://services.swpc.noaa.gov/products/kyoto-dst.json")
	if err != nil {
		return 0, nil, "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, nil, "", false
	}
	// Product may be array of objects or array of arrays
	dec := json.NewDecoder(resp.Body)
	var generic interface{}
	if err := dec.Decode(&generic); err != nil {
		return 0, nil, "", false
	}
	switch rows := generic.(type) {
	case []interface{}:
		if len(rows) == 0 {
			return 0, nil, "", false
		}
		series := make([]SeriesPoint, 0, minInt(len(rows), 72))
		for index, row := range rows {
			switch value := row.(type) {
			case map[string]interface{}:
				dst, ok := firstFiniteRange(value, -1000, 1000, "dst", "dst_index")
				if ok {
					series = append(series, SeriesPoint{T: firstString(value, "time_tag", "time", "timestamp"), V: dst})
				}
			case []interface{}:
				if index == 0 {
					continue
				}
				if len(value) > 1 {
					dst := toFloat(value[1])
					if dst >= -1000 && dst <= 1000 {
						series = append(series, SeriesPoint{T: fmt.Sprint(value[0]), V: dst})
					}
				}
			}
		}
		if len(series) > 0 {
			series = tailSeries(series, 72)
			last := series[len(series)-1]
			return last.V, series, last.T, true
		}
	}
	return 0, nil, "", false
}

func latestSeriesTime(series []SeriesPoint) string {
	if len(series) == 0 {
		return ""
	}
	return series[len(series)-1].T
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func tailSeries(series []SeriesPoint, max int) []SeriesPoint {
	if len(series) <= max {
		return series
	}
	return append([]SeriesPoint(nil), series[len(series)-max:]...)
}

func firstString(row map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := row[key]; ok {
			if text := strings.TrimSpace(fmt.Sprint(value)); text != "" && text != "<nil>" {
				return text
			}
		}
	}
	return ""
}

func firstFiniteRange(row map[string]interface{}, min, max float64, keys ...string) (float64, bool) {
	for _, key := range keys {
		value, ok := row[key]
		if !ok {
			continue
		}
		parsed := toFloat(value)
		if !math.IsNaN(parsed) && !math.IsInf(parsed, 0) && parsed >= min && parsed <= max {
			return parsed, true
		}
	}
	return 0, false
}

func findHeaderIndex(header []interface{}, names ...string) int {
	for index, value := range header {
		candidate := strings.ToLower(strings.TrimSpace(fmt.Sprint(value)))
		for _, name := range names {
			if candidate == strings.ToLower(name) {
				return index
			}
		}
	}
	return -1
}

func seriesFromTable(rows [][]interface{}, valueIdx, maxPoints int, min, max float64) []SeriesPoint {
	series := make([]SeriesPoint, 0, minInt(len(rows), maxPoints))
	for _, row := range rows {
		if valueIdx < 0 || valueIdx >= len(row) {
			continue
		}
		value := toFloat(row[valueIdx])
		if math.IsNaN(value) || math.IsInf(value, 0) || value < min || value > max {
			continue
		}
		timeTag := ""
		if len(row) > 0 {
			timeTag = fmt.Sprint(row[0])
		}
		series = append(series, SeriesPoint{T: timeTag, V: value})
	}
	return tailSeries(series, maxPoints)
}

func sampleSeries(rows [][]interface{}, valueIdx, _timeIdx, maxPoints int) []SeriesPoint {
	if len(rows) < 2 {
		return nil
	}
	data := rows
	// Skip header row when the value cell is non-numeric (SWPC product tables).
	if len(rows[0]) > valueIdx {
		if _, err := strconv.ParseFloat(fmt.Sprint(rows[0][valueIdx]), 64); err != nil {
			data = rows[1:]
		}
	}
	if len(data) == 0 {
		return nil
	}
	step := 1
	if len(data) > maxPoints {
		step = len(data) / maxPoints
	}
	out := make([]SeriesPoint, 0, maxPoints)
	for i := 0; i < len(data); i += step {
		row := data[i]
		t := ""
		if len(row) > 0 {
			t = fmt.Sprint(row[0])
		}
		out = append(out, SeriesPoint{T: t, V: asFloat(row, valueIdx)})
	}
	return out
}

func asFloat(row []interface{}, idx int) float64 {
	if idx < 0 || idx >= len(row) {
		return 0
	}
	return toFloat(row[idx])
}

func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case json.Number:
		f, _ := n.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	case int:
		return float64(n)
	default:
		f, _ := strconv.ParseFloat(fmt.Sprint(v), 64)
		return f
	}
}

func flareClassText(flux float64) string {
	if flux <= 0 || flux < 1e-9 {
		return "A0.0"
	}
	log := math.Log10(flux)
	cls := "A"
	if log > -4 {
		cls = "X"
	} else if log > -5 {
		cls = "M"
	} else if log > -6 {
		cls = "C"
	} else if log > -7 {
		cls = "B"
	}
	precision := 1
	if cls == "X" {
		precision = 2
	}
	// mantissa in [1,10)
	exp := math.Floor(log)
	num := flux / math.Pow(10, exp)
	return fmt.Sprintf("%s%.*f", cls, precision, num)
}

func parseSWPCTime(s string) time.Time {
	// common: 2024-01-02T15:04:05Z or with millis
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC()
		}
	}
	return time.Now().UTC()
}

func HandleSdoImageProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	ch := strings.TrimSpace(r.URL.Query().Get("channel"))
	imgURL, ok := ResolveSpaceImageURL(ch)
	if !ok {
		http.Error(w, fmt.Sprintf("unsupported image channel %q", ch), http.StatusBadRequest)
		return
	}

	forceRefresh := r.URL.Query().Get("refresh") == "1"
	cacheKey := strings.ToUpper(ch)
	if cacheKey == "" {
		cacheKey = "193"
	}

	if !forceRefresh {
		sdoCacheMu.Lock()
		if entry, hit := sdoCache[cacheKey]; hit && time.Since(entry.fetchedAt) < sdoProxyCacheTTL {
			body := append([]byte(nil), entry.body...)
			ct := entry.contentType
			up := entry.upstream
			age := time.Since(entry.fetchedAt)
			sdoCacheMu.Unlock()
			w.Header().Set("Content-Type", ct)
			w.Header().Set("Cache-Control", "public, max-age=1200")
			w.Header().Set("X-SDO-Upstream", up)
			w.Header().Set("X-SDO-Cache", "HIT")
			w.Header().Set("X-SDO-Cache-Age", fmt.Sprintf("%d", int(age.Seconds())))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
			return
		}
		sdoCacheMu.Unlock()
	}

	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, imgURL, nil)
	if err != nil {
		http.Error(w, "failed to build upstream request", http.StatusBadGateway)
		return
	}
	req.Header.Set("User-Agent", "AETHEL-SpaceDashboard/1.0 (+local image proxy)")
	req.Header.Set("Accept", "image/jpeg,image/*;q=0.8,*/*;q=0.5")

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch space image upstream", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("upstream returned HTTP %d", resp.StatusCode), http.StatusBadGateway)
		return
	}

	const maxSdoBytes = 8 << 20
	limited := io.LimitReader(resp.Body, maxSdoBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		http.Error(w, "failed to read upstream body", http.StatusBadGateway)
		return
	}
	if len(body) == 0 {
		http.Error(w, "upstream body empty", http.StatusBadGateway)
		return
	}
	if len(body) > maxSdoBytes {
		http.Error(w, "upstream body too large", http.StatusBadGateway)
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" || !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		contentType = "image/jpeg"
	}

	sdoCacheMu.Lock()
	sdoCache[cacheKey] = sdoCacheEntry{
		body:        append([]byte(nil), body...),
		contentType: contentType,
		fetchedAt:   time.Now().UTC(),
		upstream:    imgURL,
	}
	sdoCacheMu.Unlock()

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=1200")
	w.Header().Set("X-SDO-Upstream", imgURL)
	w.Header().Set("X-SDO-Cache", "MISS")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
