package data

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/87nehal/vengo/actuator"
	"github.com/87nehal/vengo/config"
	"github.com/87nehal/vengo/core"
)

const (
	// DBServiceName is the named service key for the standard database/sql handle.
	DBServiceName = "data.db"
	// InstrumentedDBServiceName is registered when slow query logging is enabled.
	InstrumentedDBServiceName = "data.instrumented-db"
)

// Module integrates database/sql with a vengo app.
type Module struct {
	cfg             Config
	explicitConfig  bool
	db              *sql.DB
	migrationsFS    fs.FS
	logger          *slog.Logger
	enabled         bool
	instrumentedDB  *InstrumentedDB
	createdDatabase bool
}

// Option customizes a data module.
type Option func(*Module)

// New creates a data module.
func New(options ...Option) *Module {
	m := &Module{enabled: true}
	for _, opt := range options {
		if opt != nil {
			opt(m)
		}
	}
	return m
}

// WithConfig uses cfg instead of binding database settings from the app config.
func WithConfig(cfg Config) Option {
	return func(m *Module) {
		m.cfg = cfg
		m.explicitConfig = true
	}
}

// WithDB uses an externally managed database handle. The module will not close it.
func WithDB(db *sql.DB) Option {
	return func(m *Module) {
		m.db = db
	}
}

// WithMigrations configures the filesystem that contains SQL migrations.
func WithMigrations(fsys fs.FS) Option {
	return func(m *Module) {
		m.migrationsFS = fsys
	}
}

// WithLogger configures the logger used by slow query logging.
func WithLogger(logger *slog.Logger) Option {
	return func(m *Module) {
		m.logger = logger
	}
}

// WithEnabled enables or disables the module.
func WithEnabled(enabled bool) Option {
	return func(m *Module) {
		m.enabled = enabled
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "data.sql"
}

// Configure binds configuration, registers services, and hooks database lifecycle into the app.
func (m *Module) Configure(app *core.App) error {
	if m == nil || !m.enabled {
		return nil
	}

	if !m.explicitConfig {
		if cfg, err := config.FromApp(app); err == nil {
			if err := config.Bind(cfg, &m.cfg); err != nil {
				return fmt.Errorf("bind database config: %w", err)
			}
		}
	}

	logger := m.logger
	if logger == nil {
		if appLogger, ok := actuator.LoggerFromApp(app); ok {
			logger = appLogger
		} else {
			logger = slog.Default()
		}
	}

	db := m.db
	created := false
	if db == nil {
		if m.cfg.Driver == "" {
			return fmt.Errorf("database.driver is required")
		}
		if m.cfg.DSN == "" {
			return fmt.Errorf("database.dsn is required")
		}
		opened, err := sql.Open(m.cfg.Driver, m.cfg.DSN)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		db = opened
		created = true
	}

	applyPoolConfig(db, m.cfg)

	if err := app.Register(DBServiceName, db); err != nil {
		if created {
			_ = db.Close()
		}
		return err
	}

	if m.cfg.SlowQueryThreshold > 0 {
		m.instrumentedDB = NewInstrumentedDB(db, logger, m.cfg.SlowQueryThreshold)
		if err := app.Register(InstrumentedDBServiceName, m.instrumentedDB); err != nil {
			if created {
				_ = db.Close()
			}
			return err
		}
	}

	m.db = db
	m.createdDatabase = created

	app.RegisterHook(core.Hook{
		Name: m.Name(),
		Start: func(ctx context.Context) error {
			if err := db.PingContext(ctx); err != nil {
				return fmt.Errorf("ping database: %w", err)
			}
			if m.migrationsFS != nil {
				if err := ApplyMigrations(ctx, db, m.migrationsFS, MigrationOptions{Table: m.cfg.MigrationsTable, Prefix: m.cfg.MigrationsPathPrefix}); err != nil {
					return fmt.Errorf("apply migrations: %w", err)
				}
			}
			return nil
		},
		Stop: func(context.Context) error {
			if created {
				return db.Close()
			}
			return nil
		},
	})

	return nil
}

func applyPoolConfig(db *sql.DB, cfg Config) {
	if cfg.MaxOpenConns >= 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns >= 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime >= 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime >= 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}
}

// DB returns the configured database handle after Configure has run.
func (m *Module) DB() *sql.DB {
	if m == nil {
		return nil
	}
	return m.db
}

// InstrumentedDB returns the slow-query wrapper when slow query logging is enabled.
func (m *Module) InstrumentedDB() *InstrumentedDB {
	if m == nil {
		return nil
	}
	return m.instrumentedDB
}
