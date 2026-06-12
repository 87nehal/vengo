# Release Notes - Vengo v0.1.0-beta

Welcome to the first public beta release of the **Vengo** framework! Vengo is a lightweight, dependency-injection-driven web and command-line application framework for Go.

This beta release represents the completion of the core feature set across phases 0 through 9.

---

## What's New in v0.1.0-beta

### 1. Core Application & DI Container
- **App Lifecycle**: Managed startup and stop hooks for graceful shutdown.
- **Dependency Container**: Constructor-based auto-injection (reflection-driven), named providers, cycle detection, and dependency graph serialization.
- **Dependency Overrides**: Swap out registered providers and services at runtime (crucial for testing).

### 2. Configuration System
- **Precedence Loading**: Dot-key flat mapping supporting env vars, map sources, TOML, and JSON files.
- **Struct Binding**: Auto-binding to Go structs with default values and validation tags (`required`, `nonzero`).
- **Redaction**: Automatic masking of sensitive properties (passwords, tokens, keys) in reports.

### 3. Web & Middlewares
- **Web Routing**: Go 1.22+ pattern routing, group-level prefixes, and structured route registration.
- **Middlewares**: Session stores, JWT validation, API Key protection, CORS, CSRF, and Secure Headers.
- **Robust Error Handling**: Standardized JSON response binding with validation checks.

### 4. CLI Tooling
- **Command Set**: Scaffolding (`vengo new`), dependency inspector (`vengo deps`), configuration viewer (`vengo config`), routes auditor (`vengo routes`), doctor diagnostic tool (`vengo doctor`), and file watching live-reload runner (`vengo run`).

### 5. Testing Toolkit (`testutil`)
- **App Test Harness**: Automatic lifecycle teardown and ephemeral port mapping to ensure zero port collisions.
- **HTTP client helpers**: Convenient methods for GET/POST requests and response parsing (Status, BodyString, JSON).
- **Config Helper**: Support for directory overrides (`SetupConfig`).

---

## Quick Start

Create a new application in seconds using the CLI:

```bash
go install github.com/87nehal/vengo/cmd/vengo@latest
vengo new my-app --modules=web,data
cd my-app
go run main.go
```
