package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/87nehal/vengo/config"
	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

func TestApiKeyMiddleware(t *testing.T) {
	middleware := ApiKeyMiddleware([]string{"key1", "key2"}, "X-API-Key")
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Missing API Key
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
	}

	// Case 2: Invalid API Key
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
	}

	// Case 3: Valid API Key in Header
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "key2")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	// Case 4: Valid API Key in Query Param
	req = httptest.NewRequest("GET", "/?api_key=key1", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
}

func TestJwtMiddleware(t *testing.T) {
	secret := "my-secret-key-12345"
	middleware := JwtMiddleware(secret, "vengo-issuer")
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			t.Error("claims not found in context")
		}
		w.Write([]byte(claims["sub"].(string)))
	}))

	// Case 1: Missing Authorization Header
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	// Case 2: Valid JWT Token
	claims := map[string]any{
		"sub": "user123",
		"iss": "vengo-issuer",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token, err := GenerateToken(claims, secret)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "user123" {
		t.Fatalf("expected user123, got %q", rec.Body.String())
	}

	// Case 3: Expired JWT Token
	claimsExpired := map[string]any{
		"sub": "user123",
		"iss": "vengo-issuer",
		"exp": time.Now().Add(-time.Hour).Unix(),
	}
	tokenExpired, _ := GenerateToken(claimsExpired, secret)
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenExpired)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for expired token, got %d", rec.Code)
	}

	// Case 4: Invalid signature
	tokenInvalidSig, _ := GenerateToken(claims, "wrong-secret")
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenInvalidSig)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for invalid signature, got %d", rec.Code)
	}
}

func TestCookieSessionStore(t *testing.T) {
	secret := "secret-keys-for-testing-cookie-session-store"
	store := NewCookieSessionStore(secret)

	// Get session from empty request
	req := httptest.NewRequest("GET", "/", nil)
	session, err := store.Get(req, "test_sess")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if session == nil {
		t.Fatal("session is nil")
	}
	if session.Name != "test_sess" {
		t.Fatalf("session.Name = %q, want test_sess", session.Name)
	}

	session.Set("username", "alice")
	if !session.Dirty {
		t.Fatal("session should be dirty after set")
	}

	// Save session
	rec := httptest.NewRecorder()
	err = store.Save(req, rec, session)
	if err != nil {
		t.Fatalf("save session: %v", err)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != "test_sess" {
		t.Fatalf("cookie.Name = %q, want test_sess", cookie.Name)
	}

	// Load session back
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.AddCookie(cookie)
	session2, err := store.Get(req2, "test_sess")
	if err != nil {
		t.Fatalf("get session2: %v", err)
	}
	usernameVal, ok := session2.Get("username")
	if !ok {
		t.Fatal("username not found in loaded session")
	}
	if usernameVal.(string) != "alice" {
		t.Fatalf("username = %v, want alice", usernameVal)
	}
}

func TestInMemorySessionStore(t *testing.T) {
	store := NewInMemorySessionStore()

	// Get session from empty request
	req := httptest.NewRequest("GET", "/", nil)
	session, err := store.Get(req, "test_sess")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	session.Set("roles", []any{"admin", "user"})

	// Save session
	rec := httptest.NewRecorder()
	err = store.Save(req, rec, session)
	if err != nil {
		t.Fatalf("save session: %v", err)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	cookie := cookies[0]

	// Load session back
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.AddCookie(cookie)
	session2, err := store.Get(req2, "test_sess")
	if err != nil {
		t.Fatalf("get session2: %v", err)
	}

	rolesVal, ok := session2.Get("roles")
	if !ok {
		t.Fatal("roles not found in loaded session")
	}
	rolesSlice := rolesVal.([]any)
	if len(rolesSlice) != 2 || rolesSlice[0].(string) != "admin" {
		t.Fatalf("invalid loaded roles: %v", rolesVal)
	}
}

func TestSecureHeadersMiddleware(t *testing.T) {
	cfg := Config{
		HeadersEnabled:           true,
		HeadersFrameOptions:      "SAMEORIGIN",
		HeadersContentTypeOption: "nosniff",
		HeadersXssProtection:     "1; mode=block",
	}

	middleware := SecureHeadersMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	if res.Header.Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Errorf("X-Frame-Options = %q, want SAMEORIGIN", res.Header.Get("X-Frame-Options"))
	}
	if res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", res.Header.Get("X-Content-Type-Options"))
	}
	if res.Header.Get("X-XSS-Protection") != "1; mode=block" {
		t.Errorf("X-XSS-Protection = %q, want 1; mode=block", res.Header.Get("X-XSS-Protection"))
	}
}

func TestCorsMiddleware(t *testing.T) {
	cfg := Config{
		CorsEnabled:        true,
		CorsAllowedOrigins: "http://example.com, http://test.com",
		CorsAllowedMethods: "GET,POST",
		CorsAllowedHeaders: "Content-Type",
		CorsMaxAge:         3600,
	}

	middleware := CorsMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Preflight OPTIONS request from allowed origin
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 NoContent, got %d", rec.Code)
	}
	res := rec.Result()
	if res.Header.Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want http://example.com", res.Header.Get("Access-Control-Allow-Origin"))
	}
	if res.Header.Get("Access-Control-Allow-Methods") != "GET,POST" {
		t.Errorf("Access-Control-Allow-Methods = %q, want GET,POST", res.Header.Get("Access-Control-Allow-Methods"))
	}

	// Case 2: Request from non-allowed origin
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://malicious.com")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Result().Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("expected empty Access-Control-Allow-Origin header for unauthorized origin")
	}
}

func TestCsrfMiddleware(t *testing.T) {
	cfg := Config{
		CsrfEnabled:    true,
		CsrfCookieName: "CSRF-COOKIE",
		CsrfHeaderName: "X-CSRF-TOKEN",
	}

	middleware := CsrfMiddleware(cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Safe method GET should set the CSRF cookie
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "CSRF-COOKIE" {
		t.Fatalf("expected cookie CSRF-COOKIE, got %v", cookies)
	}
	token := cookies[0].Value

	// Case 2: POST request without CSRF token in header
	req = httptest.NewRequest("POST", "/", nil)
	req.AddCookie(cookies[0])
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden without CSRF header, got %d", rec.Code)
	}

	// Case 3: POST request with matching CSRF token in header
	req = httptest.NewRequest("POST", "/", nil)
	req.AddCookie(cookies[0])
	req.Header.Set("X-CSRF-TOKEN", token)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with correct CSRF token, got %d", rec.Code)
	}
}

func TestAuthAndRoleMiddleware(t *testing.T) {
	authMw := AuthMiddleware()
	roleMw := RequireRole("admin")

	handler := authMw(roleMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	// Case 1: Unauthenticated
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
	}

	// Case 2: Authenticated but missing admin role
	req = httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), jwtClaimsContextKey, map[string]any{
		"sub":   "user1",
		"roles": []any{"user"},
	})
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req.WithContext(ctx))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", rec.Code)
	}

	// Case 3: Authenticated with correct role
	req = httptest.NewRequest("GET", "/", nil)
	ctx = context.WithValue(req.Context(), jwtClaimsContextKey, map[string]any{
		"sub":   "admin1",
		"roles": []any{"admin", "user"},
	})
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req.WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
}

func TestSecurityModuleIntegration(t *testing.T) {
	app := core.New("test-app")
	webServer := web.New(":0")
	webServer.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if sess, ok := SessionFromContext(r.Context()); ok {
			sess.Set("touched", "true")
		}
		w.WriteHeader(http.StatusOK)
	})
	app.Use(webServer)

	// Define configuration
	cfgMap := map[string]string{
		"security.headers.enabled":              "true",
		"security.headers.frame-options":        "DENY",
		"security.headers.content-type-options": "nosniff",
		"security.jwt.enabled":                  "true",
		"security.jwt.secret":                   "test-secret-key-123456",
		"security.session.enabled":              "true",
		"security.session.secret":               "session-secret-key-987654",
	}
	cfg, err := config.Load(context.Background(), config.NewMapSource("test", cfgMap))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	app.SetConfig(cfg)

	// Register Security Module
	secMod := New()
	app.Use(secMod)

	// Start App (triggers configure)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := app.Start(ctx); err != nil {
		t.Fatalf("failed to start app: %v", err)
	}
	defer app.Stop(ctx)

	// Verify that the web server applies middlewares correctly.
	// We can test this by requesting through the Server handler.
	serverHandler := webServer.Handler()

	claims := map[string]any{
		"sub": "user123",
		"iss": "vengo-issuer",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token, err := GenerateToken(claims, "test-secret-key-123456")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	serverHandler.ServeHTTP(rec, req)

	t.Logf("Integration test response code: %d, body: %q, headers: %v", rec.Code, rec.Body.String(), rec.Header())

	res := rec.Result()
	if res.Header.Get("X-Frame-Options") != "DENY" {
		t.Errorf("X-Frame-Options = %q, want DENY", res.Header.Get("X-Frame-Options"))
	}
	if res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", res.Header.Get("X-Content-Type-Options"))
	}

	// It should also set a session cookie since Session is enabled
	sessionCookieFound := false
	for _, cookie := range res.Cookies() {
		if cookie.Name == "vengo_session" {
			sessionCookieFound = true
		}
	}
	if !sessionCookieFound {
		t.Error("vengo_session cookie was not set by session middleware")
	}
}

func TestSecuritySessionSecretRequired(t *testing.T) {
	app := core.New("test-app")
	webServer := web.New(":0")
	app.Use(webServer)

	cfgMap := map[string]string{
		"security.session.enabled": "true",
		"security.session.secret":  "", // Empty secret
	}
	cfg, err := config.Load(context.Background(), config.NewMapSource("test", cfgMap))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	app.SetConfig(cfg)

	secMod := New()
	app.Use(secMod)

	if err := app.Start(context.Background()); err == nil {
		_ = app.Stop(context.Background())
		t.Fatal("expected error when starting app with sessions enabled but empty secret")
	}
}

type dummySessionStore struct{}

func (d *dummySessionStore) Get(r *http.Request, name string) (*Session, error) {
	return NewSession("dummy"), nil
}

func (d *dummySessionStore) Save(r *http.Request, w http.ResponseWriter, session *Session) error {
	return nil
}

func TestCustomSessionStoreResolution(t *testing.T) {
	app := core.New("test-app")
	webServer := web.New(":0")
	app.Use(webServer)

	cfgMap := map[string]string{
		"security.session.enabled": "true",
		"security.session.secret":  "", // Empty secret, but custom store is provided
	}
	cfg, err := config.Load(context.Background(), config.NewMapSource("test", cfgMap))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	app.SetConfig(cfg)

	customStore := &dummySessionStore{}
	if err := app.Register("my-custom-store", customStore); err != nil {
		t.Fatalf("failed to register custom store: %v", err)
	}

	secMod := New()
	app.Use(secMod)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("failed to start app with custom SessionStore: %v", err)
	}
	defer app.Stop(context.Background())
}
