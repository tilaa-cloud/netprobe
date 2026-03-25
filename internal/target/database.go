package target

import (
	"context"
	"database/sql"
	"fmt"
)

// DatabaseSource implements TargetSource by fetching targets from a SQL database
type DatabaseSource struct {
	db              *sql.DB
	query           string   // Configurable SQL query
	dimensionLabels []string // Column names for dimensions
}

// NewDatabaseSource creates a new database source
func NewDatabaseSource(db *sql.DB, query string, dimensionLabels []string) *DatabaseSource {
	return &DatabaseSource{
		db:              db,
		query:           query,
		dimensionLabels: dimensionLabels,
	}
}

// Fetch retrieves all targets from the database using the configured query
func (d *DatabaseSource) Fetch(ctx context.Context) ([]Target, error) {
	rows, err := d.db.QueryContext(ctx, d.query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var targets []Target
	for rows.Next() {
		// Scan the first column as destination_ip, rest as dimensions
		values := make([]interface{}, len(d.dimensionLabels)+1)
		var destIP string
		values[0] = &destIP

		dimensionValues := make([]string, len(d.dimensionLabels))
		for i := range d.dimensionLabels {
			values[i+1] = &dimensionValues[i]
		}

		if err := rows.Scan(values...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Build dimensions map from the configurable labels
		dims := make(map[string]string)
		for i, label := range d.dimensionLabels {
			dims[label] = dimensionValues[i]
		}

		target := Target{
			DestinationIP: destIP,
			Dimensions:    dims,
		}
		targets = append(targets, target)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading rows: %w", err)
	}

	return targets, nil
}
