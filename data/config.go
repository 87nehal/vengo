package data

import "time"

// Config contains database/sql integration settings bound from the app config.
type Config struct {
	Driver               string        `config:"database.driver" default:""`
	DSN                  string        `config:"database.dsn" default:""`
	MaxOpenConns         int           `config:"database.max-open-conns" default:"0"`
	MaxIdleConns         int           `config:"database.max-idle-conns" default:"2"`
	ConnMaxLifetime      time.Duration `config:"database.conn-max-lifetime" default:"0s"`
	ConnMaxIdleTime      time.Duration `config:"database.conn-max-idle-time" default:"0s"`
	SlowQueryThreshold   time.Duration `config:"database.slow-query-threshold" default:"0s"`
	MigrationsTable      string        `config:"database.migrations-table" default:"schema_migrations"`
	MigrationsPathPrefix string        `config:"database.migrations-path-prefix" default:"migrations"`
}
