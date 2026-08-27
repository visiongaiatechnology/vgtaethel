package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/parquet-go/parquet-go"
)

type AnalyticsRow struct {
	RecordType  string `parquet:"record_type,dict" json:"record_type"`
	RecordID    string `parquet:"record_id,dict" json:"record_id"`
	CaseID      string `parquet:"case_id,dict" json:"case_id,omitempty"`
	SourceID    string `parquet:"source_id,dict" json:"source_id,omitempty"`
	Domain      string `parquet:"domain,dict" json:"domain,omitempty"`
	TimestampUS int64  `parquet:"timestamp_us,timestamp(microsecond)" json:"timestamp_us"`
	Confidence  int32  `parquet:"confidence" json:"confidence,omitempty"`
	Quarantined bool   `parquet:"quarantined" json:"quarantined"`
}

type AnalyticsAggregate struct {
	RecordType string `json:"record_type"`
	Domain     string `json:"domain"`
	Count      int64  `json:"count"`
	Quarantine int64  `json:"quarantined_count"`
}

type AnalyticsExport struct {
	Path      string               `json:"-"`
	Artifact  string               `json:"artifact"`
	Revision  uint64               `json:"revision"`
	Rows      int                  `json:"rows"`
	SHA256    string               `json:"sha256"`
	CreatedAt time.Time            `json:"created_at"`
	Summary   []AnalyticsAggregate `json:"summary"`
	Engine    string               `json:"engine"`
}

func (s *Store) ExportAnalytics() (AnalyticsExport, error) {
	s.mu.RLock()
	rows := analyticsRows(s.state)
	revision := s.state.Revision
	s.mu.RUnlock()
	directory, err := filepath.Abs(s.path + ".analytics")
	if err != nil {
		return AnalyticsExport{}, err
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return AnalyticsExport{}, err
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return AnalyticsExport{}, err
	}
	filename := fmt.Sprintf("intelligence-r%020d.parquet", revision)
	destination := filepath.Join(directory, filename)
	if !strings.HasPrefix(destination, directory+string(filepath.Separator)) {
		return AnalyticsExport{}, errors.New("analytics path escaped jail")
	}
	if _, statErr := os.Stat(destination); os.IsNotExist(statErr) {
		temporary := destination + ".partial"
		if err := parquet.WriteFile(temporary, rows); err != nil {
			return AnalyticsExport{}, err
		}
		if err := os.Chmod(temporary, 0o600); err != nil {
			return AnalyticsExport{}, err
		}
		if err := os.Rename(temporary, destination); err != nil {
			return AnalyticsExport{}, err
		}
	} else if statErr != nil {
		return AnalyticsExport{}, statErr
	}
	raw, err := os.ReadFile(destination)
	if err != nil {
		return AnalyticsExport{}, err
	}
	summary, engine, err := queryAnalyticsParquet(destination)
	if err != nil {
		return AnalyticsExport{}, err
	}
	return AnalyticsExport{Path: destination, Artifact: filename, Revision: revision, Rows: len(rows), SHA256: contentSHA256(string(raw)), CreatedAt: time.Now().UTC(), Summary: summary, Engine: engine}, nil
}

func analyticsRows(state StoreState) []AnalyticsRow {
	rows := make([]AnalyticsRow, 0, len(state.Observations)+len(state.Events)+len(state.Claims)+len(state.Evidence))
	for _, item := range state.Observations {
		rows = append(rows, AnalyticsRow{RecordType: "observation", RecordID: item.ID, SourceID: item.SourceID, Domain: item.Domain, TimestampUS: item.ObservedAt.UnixMicro(), Quarantined: item.Quarantined})
	}
	for _, item := range state.Events {
		rows = append(rows, AnalyticsRow{RecordType: "event", RecordID: item.ID, SourceID: item.SourceID, Domain: item.Domain, TimestampUS: item.ObservedAt.UnixMicro(), Confidence: int32(item.Confidence)})
	}
	for _, item := range state.Claims {
		rows = append(rows, AnalyticsRow{RecordType: "claim", RecordID: item.ID, CaseID: item.CaseID, SourceID: item.AssertingSourceID, TimestampUS: item.CreatedAt.UnixMicro(), Confidence: int32(item.Confidence)})
	}
	for _, item := range state.Evidence {
		rows = append(rows, AnalyticsRow{RecordType: "evidence", RecordID: item.ID, CaseID: item.CaseID, SourceID: item.SourceID, TimestampUS: item.CollectedAt.UnixMicro()})
	}
	return rows
}
