package osint

// STATUS: PLATIN
// Local-only, opt-in report scheduling. No data leaves the workstation.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"go-aethel/intelligence"
)

type GlobalWatchSchedule struct {
	Enabled         bool      `json:"enabled"`
	IntervalMinutes int       `json:"interval_minutes"`
	RetentionDays   int       `json:"retention_days"`
	MaxReports      int       `json:"max_reports"`
	LastReportAt    time.Time `json:"last_report_at,omitempty"`
}
type GlobalWatchMonitor struct {
	mu         sync.Mutex
	store      *intelligence.IntelligenceStore
	config     GlobalWatchSchedule
	configPath string
	reportDir  string
	cancel     chan struct{}
	running    bool
}

func NewGlobalWatchMonitor(store *intelligence.IntelligenceStore, configPath, reportDir string) *GlobalWatchMonitor {
	m := &GlobalWatchMonitor{store: store, configPath: configPath, reportDir: reportDir, config: GlobalWatchSchedule{IntervalMinutes: 60, RetentionDays: 30, MaxReports: 100}}
	m.load()
	return m
}
func (m *GlobalWatchMonitor) load() {
	if raw, err := os.ReadFile(m.configPath); err == nil {
		var cfg GlobalWatchSchedule
		if json.Unmarshal(raw, &cfg) != nil {
			return
		}
		if cfg.RetentionDays == 0 {
			cfg.RetentionDays = 30
		}
		if cfg.MaxReports == 0 {
			cfg.MaxReports = 100
		}
		if validSchedule(cfg.IntervalMinutes, cfg.RetentionDays, cfg.MaxReports) {
			m.config = cfg
		}
	}
}
func (m *GlobalWatchMonitor) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(m.configPath), ".global-watch-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(raw)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, m.configPath)
}
func (m *GlobalWatchMonitor) Configure(enabled bool, minutes int) error {
	m.mu.Lock()
	retentionDays := m.config.RetentionDays
	maxReports := m.config.MaxReports
	m.mu.Unlock()
	return m.ConfigurePolicy(enabled, minutes, retentionDays, maxReports)
}

func validSchedule(minutes, retentionDays, maxReports int) bool {
	return minutes >= 15 && minutes <= 1440 && retentionDays >= 1 && retentionDays <= 3650 && maxReports >= 1 && maxReports <= 5000
}

func (m *GlobalWatchMonitor) ConfigurePolicy(enabled bool, minutes, retentionDays, maxReports int) error {
	if !validSchedule(minutes, retentionDays, maxReports) {
		return errors.New("report policy outside allowed boundaries")
	}
	m.mu.Lock()
	m.config.Enabled = enabled
	m.config.IntervalMinutes = minutes
	m.config.RetentionDays = retentionDays
	m.config.MaxReports = maxReports
	err := m.saveLocked()
	if m.running {
		close(m.cancel)
		m.running = false
	}
	if err == nil && enabled {
		m.startLocked()
	}
	m.mu.Unlock()
	return err
}
func (m *GlobalWatchMonitor) startLocked() {
	m.cancel = make(chan struct{})
	m.running = true
	stop := m.cancel
	go func() {
		for {
			m.mu.Lock()
			interval := time.Duration(m.config.IntervalMinutes) * time.Minute
			m.mu.Unlock()
			timer := time.NewTimer(interval)
			select {
			case <-timer.C:
				_, _ = m.GenerateReport()
			case <-stop:
				timer.Stop()
				return
			}
		}
	}()
}
func (m *GlobalWatchMonitor) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.config.Enabled && !m.running {
		m.startLocked()
	}
}
func (m *GlobalWatchMonitor) Snapshot() GlobalWatchSchedule {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.config
}
func (m *GlobalWatchMonitor) GenerateReport() (string, error) {
	// U1: prefer intelligence.SharedIntelStore structured report (same truth as chat/map)
	var report string
	if intelligence.SharedIntelStore != nil {
		report = intelligence.SharedIntelStore.GenerateReport("Daily Global Brief")
	} else if m.store != nil {
		report = m.store.Briefing()
	} else {
		return "", errors.New("intelligence store unavailable")
	}
	if err := os.MkdirAll(m.reportDir, 0700); err != nil {
		return "", err
	}
	name := filepath.Join(m.reportDir, "global-watch-"+time.Now().UTC().Format("20060102-150405")+".md")
	tmp, err := os.CreateTemp(m.reportDir, ".briefing-*.tmp")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.WriteString(report)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}
	if err = os.Rename(tmpName, name); err != nil {
		return "", err
	}
	if err = m.pruneReports(); err != nil {
		return "", err
	}
	m.mu.Lock()
	m.config.LastReportAt = time.Now().UTC()
	saveErr := m.saveLocked()
	m.mu.Unlock()
	if saveErr != nil {
		return "", saveErr
	}
	return name, nil
}

type reportFile struct {
	name    string
	modTime time.Time
}

func (m *GlobalWatchMonitor) pruneReports() error {
	m.mu.Lock()
	retentionDays := m.config.RetentionDays
	maxReports := m.config.MaxReports
	m.mu.Unlock()
	entries, err := os.ReadDir(m.reportDir)
	if err != nil {
		return err
	}
	cutoff := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	reports := make([]reportFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "global-watch-") || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		if info.ModTime().UTC().Before(cutoff) {
			if removeErr := os.Remove(filepath.Join(m.reportDir, entry.Name())); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				return removeErr
			}
			continue
		}
		reports = append(reports, reportFile{name: entry.Name(), modTime: info.ModTime()})
	}
	sort.Slice(reports, func(i, j int) bool { return reports[i].modTime.After(reports[j].modTime) })
	if len(reports) <= maxReports {
		return nil
	}
	for _, report := range reports[maxReports:] {
		if err := os.Remove(filepath.Join(m.reportDir, report.name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
