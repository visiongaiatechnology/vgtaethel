package geoint

import (
	"context"
	"log"
	"sync"
	"time"

	"go-aethel/osint"
	"go-aethel/intelligence"
)

// GeoIntService is the master GEOINT engine service in Aethel
type GeoIntService struct {
	bus         *GeoEntityBus
	aircraft    *AircraftCollector
	satellites  *SatelliteCollector
	vessels     *VesselCollector
	cctv        *CCTVCollector
	hazards     *HazardsCollector
	correlation *CorrelationEngine
	shadow      *ShadowGeoAdapter
	intelligence *IntelligenceGeoAdapter
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.Mutex
	running     bool
}

func (s *GeoIntService) AttachIntelligence(store *intelligence.Store) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.intelligence = NewIntelligenceGeoAdapter(s.bus, store)
}

// AttachShadow binds the existing SHADOW evidence store to GEOINT without
// starting another collector or model request.
func (s *GeoIntService) AttachShadow(shadow *osint.ShadowService) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shadow = NewShadowGeoAdapter(s.bus, shadow)
}

// Relations returns the current evidence-bound SHADOW relationships.
func (s *GeoIntService) Relations() ([]GeoRelation, time.Time, string) {
	s.mu.Lock()
	adapter := s.shadow
	s.mu.Unlock()
	if adapter == nil {
		return []GeoRelation{}, time.Time{}, ""
	}
	return adapter.Relations()
}

func (s *GeoIntService) Status() GeoIntSystemStatus {
	status := s.bus.GetStatus()
	relations, synchronizedAt, _ := s.Relations()
	status.ShadowRelations = len(relations)
	status.ShadowSyncAt = synchronizedAt
	return status
}

// NewGeoIntService instantiates all GEOINT components
func NewGeoIntService(workspaceDir string, firmsMapKey string) *GeoIntService {
	bus := NewGeoEntityBus()
	aircraft := NewAircraftCollector(bus)
	satellites := NewSatelliteCollector(bus, workspaceDir+"/satellites")
	vessels := NewVesselCollector(bus)
	cctv := NewCCTVCollector(bus)
	hazards := NewHazardsCollector(bus, firmsMapKey)
	correlation := NewCorrelationEngine(bus)

	return &GeoIntService{
		bus:         bus,
		aircraft:    aircraft,
		satellites:  satellites,
		vessels:     vessels,
		cctv:        cctv,
		hazards:     hazards,
		correlation: correlation,
	}
}

// Bus returns the central GeoEntityBus
func (s *GeoIntService) Bus() *GeoEntityBus {
	return s.bus
}

// Aircraft returns the AircraftCollector
func (s *GeoIntService) Aircraft() *AircraftCollector {
	return s.aircraft
}

// Satellites returns the SatelliteCollector
func (s *GeoIntService) Satellites() *SatelliteCollector {
	return s.satellites
}

// Vessels returns the VesselCollector
func (s *GeoIntService) Vessels() *VesselCollector {
	return s.vessels
}

// CCTV returns the CCTVCollector
func (s *GeoIntService) CCTV() *CCTVCollector {
	return s.cctv
}

// Hazards returns the HazardsCollector
func (s *GeoIntService) Hazards() *HazardsCollector {
	return s.hazards
}

// Correlation returns the CorrelationEngine
func (s *GeoIntService) Correlation() *CorrelationEngine {
	return s.correlation
}

// Start launches all background collection and correlation workers
func (s *GeoIntService) Start(parentCtx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.ctx, s.cancel = context.WithCancel(parentCtx)
	s.running = true
	s.mu.Unlock()

	log.Println("🛰️ AETHEL GEOINT :: ENGINE STARTED")

	// Immediate initial load
	go func() {
		ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
		defer cancel()
		_, _ = s.cctv.Refresh(ctx)
		_, _ = s.vessels.Refresh(ctx)
		_, _ = s.hazards.Refresh(ctx)
		_, _ = s.satellites.Refresh(ctx)
		_, _ = s.aircraft.Refresh(ctx)
		s.correlation.AnalyzeCrossModalPatterns(ctx)
		s.syncShadow()
	}()

	// 1. Fast loop: Aircraft (every 30s)
	go s.runWorker(30*time.Second, func(ctx context.Context) {
		_, _ = s.aircraft.Refresh(ctx)
	})

	// 2. Medium loop: AIS Vessels & Hazards (every 60s)
	go s.runWorker(60*time.Second, func(ctx context.Context) {
		_, _ = s.vessels.Refresh(ctx)
		_, _ = s.hazards.Refresh(ctx)
	})

	// 3. Correlation loop (every 90s)
	go s.runWorker(90*time.Second, func(ctx context.Context) {
		s.correlation.AnalyzeCrossModalPatterns(ctx)
	})

	// SHADOW projection is local and never invokes collection or an AI model.
	go s.runWorker(30*time.Second, func(_ context.Context) {
		s.syncShadow()
		s.bus.PruneStale(time.Now().UTC())
	})

	// 4. Satellite positions are propagated locally; network TLE refresh remains cached for six hours.
	go s.runWorker(10*time.Second, func(ctx context.Context) {
		_, _ = s.satellites.Refresh(ctx)
	})

	// 5. Static public CCTV catalog refresh.
	go s.runWorker(10*time.Minute, func(ctx context.Context) {
		_, _ = s.cctv.Refresh(ctx)
	})
}

func (s *GeoIntService) syncShadow() {
	s.mu.Lock()
	adapter := s.shadow
	intelAdapter := s.intelligence
	s.mu.Unlock()
	if adapter != nil {
		adapter.Sync(time.Now().UTC())
	}
	if intelAdapter != nil {
		intelAdapter.Sync(time.Now().UTC())
	}
}

// Stop terminates all background workers
func (s *GeoIntService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	if s.cancel != nil {
		s.cancel()
	}
	s.running = false
	log.Println("🔴 AETHEL GEOINT :: ENGINE STOPPED")
}

func (s *GeoIntService) runWorker(interval time.Duration, task func(ctx context.Context)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(s.ctx, 45*time.Second)
			task(ctx)
			cancel()
		}
	}
}
