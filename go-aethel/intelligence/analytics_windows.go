//go:build windows

package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"sort"

	"github.com/parquet-go/parquet-go"
)

func queryAnalyticsParquet(path string) ([]AnalyticsAggregate, string, error) {
	rows, err := parquet.ReadFile[AnalyticsRow](path)
	if err != nil {
		return nil, "parquet-go-windows", err
	}
	aggregates := make(map[string]AnalyticsAggregate)
	for _, row := range rows {
		key := row.RecordType + "\x00" + row.Domain
		item := aggregates[key]
		item.RecordType, item.Domain = row.RecordType, row.Domain
		item.Count++
		if row.Quarantined {
			item.Quarantine++
		}
		aggregates[key] = item
	}
	result := make([]AnalyticsAggregate, 0, len(aggregates))
	for _, item := range aggregates {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].RecordType == result[j].RecordType {
			return result[i].Domain < result[j].Domain
		}
		return result[i].RecordType < result[j].RecordType
	})
	return result, "parquet-go-windows", nil
}
