package security

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/87nehal/vengo/web"
)

type contextKey string

const (
	sessionContextKey   contextKey = "session"
	jwtClaimsContextKey contextKey = "jwt_claims"
	authUserContextKey  contextKey = "auth_user"
)

// User represents the authenticated identity.
type User struct {
	ID    string
	Roles []string
}

// CorsMiddleware implements Cross-Origin Resource Sharing.
func CorsMiddleware(cfg Config) web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				allowedOrigins := strings.Split(cfg.CorsAllowedOrigins, ",")
				allowed := false
				for _, ao := range allowedOrigins {
					ao = strings.TrimSpace(ao)
					if ao == "*" || ao == origin {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						allowed = true
						break
					}
				}

				if allowed {
					if cfg.CorsAllowCredentials {
						w.Header().Set("Access-Control-Allow-Credentials", "true")
					}
					if cfg.CorsAllowedMethods != "" {
						w.Header().Set("Access-Control-Allow-Methods", cfg.CorsAllowedMethods)
					}
					if cfg.CorsAllowedHeaders != "" {
						w.Header().Set("Access-Control-Allow-Headers", cfg.CorsAllowedHeaders)
					}
					if cfg.CorsMaxAge > 0 {
						w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.CorsMaxAge))
					}
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecureHeadersMiddleware injects common security headers.
func SecureHeadersMiddleware(cfg Config) web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.HeadersFrameOptions != "" {
				w.Header().Set("X-Frame-Options", cfg.HeadersFrameOptions)
			}
			if cfg.HeadersContentTypeOption != "" {
				w.Header().Set("X-Content-Type-Options", cfg.HeadersContentTypeOption)
			}
			if cfg.HeadersXssProtection != "" {
				w.Header().Set("X-XSS-Protection", cfg.HeadersXssProtection)
			}
			if cfg.HeadersReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", cfg.HeadersReferrerPolicy)
			}
			if cfg.HeadersCsp != "" {
				w.Header().Set("Content-Security-Policy", cfg.HeadersCsp)
			}
			if cfg.HeadersHsts != "" && (r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https") {
				w.Header().Set("Strict-Transport-Security", cfg.HeadersHsts)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CsrfMiddleware protects against CSRF using the double-submit cookie pattern.
func CsrfMiddleware(cfg Config) web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookieName := cfg.CsrfCookieName
			if cookieName == "" {
				cookieName = "XSRF-TOKEN"
			}
			headerName := cfg.CsrfHeaderName
			if headerName == "" {
				headerName = "X-XSRF-TOKEN"
			}

			var csrfToken string
			cookie, err := r.Cookie(cookieName)
			if err == nil {
				csrfToken = cookie.Value
			}

			if csrfToken == "" {
				csrfToken = generateSessionID()
				http.SetCookie(w, &http.Cookie{
					Name:     cookieName,
					Value:    csrfToken,
					Path:     "/",
					HttpOnly: false, // Javascript needs to read this cookie for double-submit
					Secure:   r.TLS != nil,
				})
			}

			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
				reqToken := r.Header.Get(headerName)
				if reqToken == "" {
					reqToken = r.FormValue("_csrf")
				}
				if reqToken == "" || reqToken != csrfToken {
					http.Error(w, "CSRF token validation failed", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SessionMiddleware exposes session state through request context.
func SessionMiddleware(store SessionStore, cookieName string) web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := store.Get(r, cookieName)
			if err != nil {
				http.Error(w, "failed to load session", http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), sessionContextKey, session)
			r = r.WithContext(ctx)

			rw := &sessionResponseWriter{ResponseWriter: w, req: r, store: store, session: session}
			next.ServeHTTP(rw, r)

			_ = rw.Save()
		})
	}
}

type sessionResponseWriter struct {
	http.ResponseWriter
	req     *http.Request
	store   SessionStore
	session *Session
	written bool
}

func (s *sessionResponseWriter) WriteHeader(status int) {
	_ = s.Save()
	s.ResponseWriter.WriteHeader(status)
}

func (s *sessionResponseWriter) Write(b []byte) (int, error) {
	_ = s.Save()
	return s.ResponseWriter.Write(b)
}

func (s *sessionResponseWriter) Save() error {
	if s.written {
		return nil
	}
	s.written = true
	return s.store.Save(s.req, s.ResponseWriter, s.session)
}

// SessionFromContext extracts a session from the context.
func SessionFromContext(ctx context.Context) (*Session, bool) {
	sess, ok := ctx.Value(sessionContextKey).(*Session)
	return sess, ok
}

// JwtMiddleware validates HS256 tokens and injects claims.
func JwtMiddleware(secret string, expectedIssuer string) web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "missing or invalid authorization header", http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := VerifyToken(tokenStr, secret)
			if err != nil {
				http.Error(w, "invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}

			if expectedIssuer != "" {
				iss, _ := claims["iss"].(string)
				if iss != expectedIssuer {
					http.Error(w, "invalid token issuer", http.StatusUnauthorized)
					return
				}
			}

			if expVal, exists := claims["exp"]; exists {
				var expSec int64
				switch v := expVal.(type) {
				case float64:
					expSec = int64(v)
				case int64:
					expSec = v
				default:
					http.Error(w, "invalid token expiration claim", http.StatusUnauthorized)
					return
				}
				if time.Now().Unix() > expSec {
					http.Error(w, "token has expired", http.StatusUnauthorized)
					return
				}
			}

			ctx := context.WithValue(r.Context(), jwtClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext extracts JWT claims from context.
func ClaimsFromContext(ctx context.Context) (map[string]any, bool) {
	claims, ok := ctx.Value(jwtClaimsContextKey).(map[string]any)
	return claims, ok
}

// ApiKeyMiddleware validates simple API keys.
func ApiKeyMiddleware(allowedKeys []string, headerName string) web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get(headerName)
			if apiKey == "" {
				apiKey = r.URL.Query().Get("api_key")
			}

			if apiKey == "" {
				http.Error(w, "missing API key", http.StatusUnauthorized)
				return
			}

			valid := false
			for _, k := range allowedKeys {
				if apiKey == k {
					valid = true
					break
				}
			}

			if !valid {
				http.Error(w, "invalid API key", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthMiddleware extracts authentication information to create a User context.
func AuthMiddleware() web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if claims, ok := ClaimsFromContext(r.Context()); ok {
				sub, _ := claims["sub"].(string)
				var roles []string
				if rVal, ok := claims["roles"]; ok {
					if rList, ok := rVal.([]any); ok {
						for _, role := range rList {
							if roleStr, ok := role.(string); ok {
								roles = append(roles, roleStr)
							}
						}
					}
				}
				user := &User{ID: sub, Roles: roles}
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authUserContextKey, user)))
				return
			}

			if sess, ok := SessionFromContext(r.Context()); ok {
				if userIDVal, ok := sess.Get("user_id"); ok {
					if userID, ok := userIDVal.(string); ok {
						var roles []string
						if rolesVal, ok := sess.Get("roles"); ok {
							if rList, ok := rolesVal.([]any); ok {
								for _, role := range rList {
									if roleStr, ok := role.(string); ok {
										roles = append(roles, roleStr)
									}
								}
							}
						}
						user := &User{ID: userID, Roles: roles}
						next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authUserContextKey, user)))
						return
					}
				}
			}

			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		})
	}
}

// UserFromContext retrieves the authenticated User from context.
func UserFromContext(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(authUserContextKey).(*User)
	return u, ok
}

// RequireRole restricts access to users with a specific role.
func RequireRole(role string) web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			hasRole := false
			for _, r := range user.Roles {
				if r == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// VerifyToken decodes and verifies an HS256 JWT string.
func VerifyToken(tokenStr string, secret string) (map[string]any, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, errors.New("token must have 3 parts")
	}

	headerPart, payloadPart, signaturePart := parts[0], parts[1], parts[2]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(headerPart + "." + payloadPart))
	expectedSig := mac.Sum(nil)
	actualSig, err := base64.RawURLEncoding.DecodeString(signaturePart)
	if err != nil {
		return nil, errors.New("invalid signature encoding")
	}

	if !hmac.Equal(actualSig, expectedSig) {
		return nil, errors.New("signature is invalid")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return nil, errors.New("invalid payload encoding")
	}

	var claims map[string]any
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, errors.New("invalid payload JSON")
	}

	return claims, nil
}

// GenerateToken helper creates a signed HS256 JWT string.
func GenerateToken(claims map[string]any, secret string) (string, error) {
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	headerB64 := base64.RawURLEncoding.EncodeToString(headerBytes)

	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)

	signingInput := headerB64 + "." + payloadB64
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature, nil
}
