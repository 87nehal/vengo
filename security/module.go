package security

import (
	"fmt"
	"strings"

	"github.com/87nehal/vengo/config"
	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

const (
	ModuleName          = "security"
	SessionStoreService = "security.session-store"
)

// Module integrates security features into a vengo app.
type Module struct {
	cfg            Config
	explicitConfig bool
	enabled        bool
}

// Option customizes the security module.
type Option func(*Module)

// New creates a new security module.
func New(options ...Option) *Module {
	m := &Module{enabled: true}
	for _, opt := range options {
		if opt != nil {
			opt(m)
		}
	}
	return m
}

// WithConfig sets an explicit security configuration.
func WithConfig(cfg Config) Option {
	return func(m *Module) {
		m.cfg = cfg
		m.explicitConfig = true
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
	return ModuleName
}

// Configure binds the config and registers middlewares on the web server if present.
func (m *Module) Configure(app *core.App) error {
	if m == nil || !m.enabled {
		return nil
	}

	// 1. Bind configuration
	if !m.explicitConfig {
		if cfg, err := config.FromApp(app); err == nil {
			if err := config.Bind(cfg, &m.cfg); err != nil {
				return fmt.Errorf("bind security config: %w", err)
			}
		}
	}

	// 2. Register security config
	if err := app.Register("security.config", &m.cfg); err != nil {
		return err
	}

	// 3. Resolve or create SessionStore
	var store SessionStore
	if resolvedStore, err := core.Resolve[SessionStore](app); err == nil {
		store = resolvedStore
	} else {
		if m.cfg.SessionEnabled && m.cfg.SessionSecret == "" {
			return fmt.Errorf("security.session.secret must be configured when session management is enabled")
		}
		// Use secret from config. If empty, default to a fallback.
		secret := m.cfg.SessionSecret
		if secret == "" {
			secret = "vengo-fallback-secret-for-sessions-change-me"
		}
		store = NewCookieSessionStore(secret)
		_ = app.Register(SessionStoreService, store)
	}

	// 4. Register security middlewares on web.Server if it is registered
	if server, err := core.Get[*web.Server](app, web.ServiceName); err == nil {
		// Apply CORS first so preflight OPTIONS requests return immediately.
		if m.cfg.CorsEnabled {
			server.Use(CorsMiddleware(m.cfg))
		}

		// Apply Secure Headers
		if m.cfg.HeadersEnabled {
			server.Use(SecureHeadersMiddleware(m.cfg))
		}

		// Apply CSRF Protection
		if m.cfg.CsrfEnabled {
			server.Use(CsrfMiddleware(m.cfg))
		}

		// Apply Session Management
		if m.cfg.SessionEnabled {
			server.Use(SessionMiddleware(store, m.cfg.SessionCookie))
		}

		// Apply JWT Authentication
		if m.cfg.JwtEnabled {
			server.Use(JwtMiddleware(m.cfg.JwtSecret, m.cfg.JwtIssuer))
		}

		// Apply API Key Authentication
		if m.cfg.ApiKeyEnabled {
			var allowedKeys []string
			if m.cfg.ApiKeyKeys != "" {
				parts := strings.Split(m.cfg.ApiKeyKeys, ",")
				for _, part := range parts {
					allowedKeys = append(allowedKeys, strings.TrimSpace(part))
				}
			}
			server.Use(ApiKeyMiddleware(allowedKeys, m.cfg.ApiKeyHeader))
		}
	}

	return nil
}
