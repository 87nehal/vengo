package security

// Config holds the security settings for the vengo application.
type Config struct {
	// API Key Configuration
	ApiKeyEnabled bool   `config:"security.api-key.enabled" default:"false"`
	ApiKeyKeys    string `config:"security.api-key.keys" default:""` // comma-separated
	ApiKeyHeader  string `config:"security.api-key.header" default:"X-API-Key"`

	// JWT Configuration
	JwtEnabled bool   `config:"security.jwt.enabled" default:"false"`
	JwtSecret  string `config:"security.jwt.secret" default:""`
	JwtIssuer  string `config:"security.jwt.issuer" default:""`

	// Session Configuration
	SessionEnabled bool   `config:"security.session.enabled" default:"false"`
	SessionSecret  string `config:"security.session.secret" default:""`
	SessionCookie  string `config:"security.session.cookie" default:"vengo_session"`

	// Secure Headers Configuration
	HeadersEnabled           bool   `config:"security.headers.enabled" default:"false"`
	HeadersFrameOptions      string `config:"security.headers.frame-options" default:"DENY"`
	HeadersContentTypeOption string `config:"security.headers.content-type-options" default:"nosniff"`
	HeadersXssProtection     string `config:"security.headers.xss-protection" default:"1; mode=block"`
	HeadersReferrerPolicy    string `config:"security.headers.referrer-policy" default:"strict-origin-when-cross-origin"`
	HeadersCsp               string `config:"security.headers.content-security-policy" default:""`
	HeadersHsts              string `config:"security.headers.hsts" default:"max-age=63072000; includeSubDomains"`

	// CORS Configuration
	CorsEnabled          bool   `config:"security.cors.enabled" default:"false"`
	CorsAllowedOrigins   string `config:"security.cors.allowed-origins" default:"*"`
	CorsAllowedMethods   string `config:"security.cors.allowed-methods" default:"GET,POST,PUT,DELETE,OPTIONS"`
	CorsAllowedHeaders   string `config:"security.cors.allowed-headers" default:"Content-Type,Authorization"`
	CorsAllowCredentials bool   `config:"security.cors.allow-credentials" default:"false"`
	CorsMaxAge           int    `config:"security.cors.max-age" default:"1800"`

	// CSRF Configuration
	CsrfEnabled    bool   `config:"security.csrf.enabled" default:"false"`
	CsrfCookieName string `config:"security.csrf.cookie-name" default:"XSRF-TOKEN"`
	CsrfHeaderName string `config:"security.csrf.header-name" default:"X-XSRF-TOKEN"`
}
