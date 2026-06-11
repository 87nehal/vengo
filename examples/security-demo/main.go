package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/87nehal/vengo/config"
	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/security"
	"github.com/87nehal/vengo/web"
)

type AppConfig struct {
	Port int `config:"server.port" default:"8080"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Load config
	cfg, err := config.LoadDefaults(ctx, config.ActiveProfile())
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	var appCfg AppConfig
	if err := config.Bind(cfg, &appCfg); err != nil {
		log.Fatalf("failed to bind config: %v", err)
	}
	jwtSecret := "demo-signing-secret-key-12345"
	if configuredSecret, exists := cfg.Get("security.jwt.secret"); exists {
		jwtSecret = configuredSecret
	}
	jwtIssuer := "vengo-demo"
	if configuredIssuer, exists := cfg.Get("security.jwt.issuer"); exists {
		jwtIssuer = configuredIssuer
	}

	// 2. Create web server
	addr := fmt.Sprintf(":%d", appCfg.Port)
	server := web.New(addr)

	// 3. Define routes
	server.HandleFunc("GET /public", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message": "This is a public endpoint accessible by anyone."}`))
	})

	// Login endpoint to establish a cookie session
	server.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		sess, ok := security.SessionFromContext(r.Context())
		if !ok {
			http.Error(w, "session unavailable", http.StatusInternalServerError)
			return
		}
		sess.Set("user_id", "user123")
		sess.Set("roles", []any{"admin", "user"})

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message": "Logged in successfully! Session established."}`))
	})

	// Private admin-only session-secured route
	privateGroup := server.Group("/private", security.AuthMiddleware(), security.RequireRole("admin"))
	privateGroup.HandleFunc("GET /dashboard", func(w http.ResponseWriter, r *http.Request) {
		user, _ := security.UserFromContext(r.Context())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fmt.Sprintf(`{"message": "Welcome to the admin dashboard!", "userId": %q}`, user.ID)))
	})

	// Token generator for API / JWT demo
	server.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		claims := map[string]any{
			"sub":   "api-client-99",
			"iss":   jwtIssuer,
			"roles": []string{"api-role"},
			"exp":   time.Now().Add(time.Hour).Unix(),
		}
		token, err := security.GenerateToken(claims, jwtSecret)
		if err != nil {
			http.Error(w, "token generation failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fmt.Sprintf(`{"token": %q}`, token)))
	})

	// API route protected by JWT
	apiGroup := server.Group("/api", security.JwtMiddleware(jwtSecret, jwtIssuer), security.AuthMiddleware())
	apiGroup.HandleFunc("GET /data", func(w http.ResponseWriter, r *http.Request) {
		user, _ := security.UserFromContext(r.Context())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fmt.Sprintf(`{"message": "Secure API data retrieved", "clientId": %q}`, user.ID)))
	})

	// 4. Initialize app and register modules
	app := core.New("security-demo", server, security.New())
	app.SetConfig(cfg)

	// 5. Start app
	if err := app.Start(ctx); err != nil {
		log.Fatalf("failed to start app: %v", err)
	}
	log.Printf("Security demo listening on %s", server.Addr())

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Stop(shutdownCtx); err != nil {
		log.Fatalf("failed to stop app: %v", err)
	}
}
