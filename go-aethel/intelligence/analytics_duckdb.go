//go:build !windows

package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

func queryAnalyticsParquet(path string) ([]AnalyticsAggregate, string, error) {
	database, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, "duckdb", err
	}
	defer database.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	for _, setting := range []string{"SET autoinstall_known_extensions=false", "SET autoload_known_extensions=false", "SET threads=2", "SET memory_limit='256MB'"} {
		if _, err := database.ExecContext(ctx, setting); err != nil {
			return nil, "duckdb", err
		}
	}
	rows, err := database.QueryContext(ctx, `SELECT record_type, domain, COUNT(*) AS count, SUM(CASE WHEN quarantined THEN 1 ELSE 0 END) AS quarantined_count FROM read_parquet(?) GROUP BY record_type, domain ORDER BY record_type, domain`, path)
	if err != nil {
		return nil, "duckdb", err
	}
	defer rows.Close()
	result := make([]AnalyticsAggregate, 0)
	for rows.Next() {
		var item AnalyticsAggregate
		if err := rows.Scan(&item.RecordType, &item.Domain, &item.Count, &item.Quarantine); err != nil {
			return nil, "duckdb", err
		}
		result = append(result, item)
	}
	return result, "duckdb", rows.Err()
}
