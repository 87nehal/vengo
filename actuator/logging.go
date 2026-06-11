package actuator

import (
	"log/slog"
	"os"

	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

const LoggingServiceName = "actuator.logging"

type LoggingModule struct {
	logger        *slog.Logger
	level         slog.Level
	enabled       bool
	useMiddleware bool
}

type LoggingOption func(*LoggingModule)

func NewLogging(options ...LoggingOption) *LoggingModule {
	module := &LoggingModule{
		level:         slog.LevelInfo,
		enabled:       true,
		useMiddleware: true,
	}
	for _, opt := range options {
		if opt != nil {
			opt(module)
		}
	}
	if module.enabled && module.logger == nil {
		module.logger = slog.New(
			slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: module.level}),
		)
	}
	return module
}

func WithLogger(logger *slog.Logger) LoggingOption {
	return func(m *LoggingModule) {
		m.logger = logger
	}
}

func WithLogLevel(level slog.Level) LoggingOption {
	return func(m *LoggingModule) {
		m.level = level
	}
}

func WithLoggingEnabled(enabled bool) LoggingOption {
	return func(m *LoggingModule) {
		m.enabled = enabled
	}
}

func WithRequestLogging(enabled bool) LoggingOption {
	return func(m *LoggingModule) {
		m.useMiddleware = enabled
	}
}

func (m *LoggingModule) Name() string {
	return "actuator.logging"
}

func (m *LoggingModule) Logger() *slog.Logger {
	return m.logger
}

func (m *LoggingModule) Configure(app *core.App) error {
	if !m.enabled {
		return nil
	}

	if err := app.Register(LoggingServiceName, m); err != nil {
		return err
	}

	if m.useMiddleware {
		server, err := core.Get[*web.Server](app, web.ServiceName)
		if err != nil {
			return nil
		}
		server.Use(web.RequestLogger(m.logger))
	}

	return nil
}

func LoggerFromApp(app *core.App) (*slog.Logger, bool) {
	mod, err := core.Get[*LoggingModule](app, LoggingServiceName)
	if err != nil {
		return nil, false
	}
	return mod.Logger(), true
}
