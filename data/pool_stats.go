package data

import (
	"context"
	"database/sql"

	"github.com/87nehal/vengo/actuator"
)

// PoolStatsIndicator exposes database/sql connection pool statistics as actuator health details.
type PoolStatsIndicator struct {
	name string
	db   *sql.DB
}

// NewPoolStatsIndicator creates a health indicator that reports connection pool statistics.
func NewPoolStatsIndicator(db *sql.DB) *PoolStatsIndicator {
	return &PoolStatsIndicator{name: "database-pool", db: db}
}

// HealthIndicatorName returns the health details key for this indicator.
func (p *PoolStatsIndicator) HealthIndicatorName() string {
	if p == nil || p.name == "" {
		return "database-pool"
	}
	return p.name
}

// Health reports pool statistics as part of the health check.
func (p *PoolStatsIndicator) Health(ctx context.Context) actuator.Health {
	if p == nil || p.db == nil {
		return actuator.Health{Status: actuator.StatusDown, Details: map[string]any{"error": "database is nil"}}
	}
	if err := p.db.PingContext(ctx); err != nil {
		return actuator.Health{Status: actuator.StatusDown, Details: map[string]any{"error": err.Error()}}
	}
	stats := p.db.Stats()
	return actuator.Health{
		Status: actuator.StatusUp,
		Details: map[string]any{
			"max_open_connections": stats.MaxOpenConnections,
			"open_connections":     stats.OpenConnections,
			"in_use":               stats.InUse,
			"idle":                 stats.Idle,
			"wait_count":           stats.WaitCount,
			"wait_duration":        stats.WaitDuration.String(),
			"max_idle_closed":      stats.MaxIdleClosed,
			"max_idle_time_closed": stats.MaxIdleTimeClosed,
			"max_lifetime_closed":  stats.MaxLifetimeClosed,
		},
	}
}

// DBStats returns the current sql.DBStats for programmatic access.
func DBStats(db *sql.DB) sql.DBStats {
	if db == nil {
		return sql.DBStats{}
	}
	return db.Stats()
}
