package data

import (
	"context"
	"database/sql"

	"github.com/87nehal/vengo/actuator"
)

// DBHealthIndicator reports database availability through actuator health checks.
type DBHealthIndicator struct {
	name string
	db   *sql.DB
}

// NewHealthIndicator creates an actuator-compatible database health indicator.
func NewHealthIndicator(db *sql.DB) *DBHealthIndicator {
	return &DBHealthIndicator{name: "database", db: db}
}

// HealthIndicatorName returns the health details key for this indicator.
func (h *DBHealthIndicator) HealthIndicatorName() string {
	if h == nil || h.name == "" {
		return "database"
	}
	return h.name
}

// Health pings the database and returns UP or DOWN.
func (h *DBHealthIndicator) Health(ctx context.Context) actuator.Health {
	if h == nil || h.db == nil {
		return actuator.Health{Status: actuator.StatusDown, Details: map[string]any{"error": "database is nil"}}
	}
	if err := h.db.PingContext(ctx); err != nil {
		return actuator.Health{Status: actuator.StatusDown, Details: map[string]any{"error": err.Error()}}
	}
	return actuator.Health{Status: actuator.StatusUp}
}
