package config

import (
	"context"
	"testing"
	"time"
)

func TestBindBasicTypes(t *testing.T) {
	cfg, err := Load(context.Background(), NewMapSource("test", map[string]string{
		"server.port":    "9090",
		"server.host":    "0.0.0.0",
		"server.debug":   "true",
		"server.timeout": "30s",
		"server.rate":    "1.5",
		"server.workers": "4",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	type ServerConfig struct {
		Port    int           `config:"server.port"`
		Host    string        `config:"server.host"`
		Debug   bool          `config:"server.debug"`
		Timeout time.Duration `config:"server.timeout"`
		Rate    float64       `config:"server.rate"`
		Workers uint          `config:"server.workers"`
	}

	var s ServerConfig
	if err := Bind(cfg, &s); err != nil {
		t.Fatalf("bind: %v", err)
	}

	if s.Port != 9090 {
		t.Fatalf("port = %d, want 9090", s.Port)
	}
	if s.Host != "0.0.0.0" {
		t.Fatalf("host = %q, want 0.0.0.0", s.Host)
	}
	if !s.Debug {
		t.Fatal("debug = false, want true")
	}
	if s.Timeout != 30*time.Second {
		t.Fatalf("timeout = %v, want 30s", s.Timeout)
	}
	if s.Rate != 1.5 {
		t.Fatalf("rate = %f, want 1.5", s.Rate)
	}
	if s.Workers != 4 {
		t.Fatalf("workers = %d, want 4", s.Workers)
	}
}

func TestBindDefaultValues(t *testing.T) {
	cfg, err := Load(context.Background(), NewMapSource("test", map[string]string{}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	type Config struct {
		Port int    `config:"server.port" default:"8080"`
		Host string `config:"server.host" default:"localhost"`
	}

	var c Config
	if err := Bind(cfg, &c); err != nil {
		t.Fatalf("bind: %v", err)
	}

	if c.Port != 8080 {
		t.Fatalf("port = %d, want 8080", c.Port)
	}
	if c.Host != "localhost" {
		t.Fatalf("host = %q, want localhost", c.Host)
	}
}

func TestBindConfigOverridesDefault(t *testing.T) {
	cfg, err := Load(context.Background(), NewMapSource("test", map[string]string{
		"server.port": "3000",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	type Config struct {
		Port int `config:"server.port" default:"8080"`
	}

	var c Config
	if err := Bind(cfg, &c); err != nil {
		t.Fatalf("bind: %v", err)
	}

	if c.Port != 3000 {
		t.Fatalf("port = %d, want 3000", c.Port)
	}
}

func TestBindNestedStruct(t *testing.T) {
	cfg, err := Load(context.Background(), NewMapSource("test", map[string]string{
		"server.port":   "9090",
		"database.host": "db.local",
		"database.port": "5432",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	type DatabaseConfig struct {
		Host string `config:"database.host"`
		Port int    `config:"database.port"`
	}

	type AppConfig struct {
		Port int `config:"server.port"`
		DB   DatabaseConfig
	}

	var c AppConfig
	if err := Bind(cfg, &c); err != nil {
		t.Fatalf("bind: %v", err)
	}

	if c.Port != 9090 {
		t.Fatalf("port = %d, want 9090", c.Port)
	}
	if c.DB.Host != "db.local" {
		t.Fatalf("db.host = %q, want db.local", c.DB.Host)
	}
	if c.DB.Port != 5432 {
		t.Fatalf("db.port = %d, want 5432", c.DB.Port)
	}
}

func TestBindNestedStructUsesFieldNamePrefix(t *testing.T) {
	cfg, err := Load(context.Background(), NewMapSource("test", map[string]string{
		"db.host": "db.local",
		"db.port": "5432",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	type DatabaseConfig struct {
		Host string
		Port int
	}

	type AppConfig struct {
		DB DatabaseConfig
	}

	var c AppConfig
	if err := Bind(cfg, &c); err != nil {
		t.Fatalf("bind: %v", err)
	}

	if c.DB.Host != "db.local" {
		t.Fatalf("db.host = %q, want db.local", c.DB.Host)
	}
	if c.DB.Port != 5432 {
		t.Fatalf("db.port = %d, want 5432", c.DB.Port)
	}
}

func TestBindRejectsNonPointer(t *testing.T) {
	cfg, _ := Load(context.Background(), NewMapSource("test", map[string]string{}))

	type Config struct{}
	var c Config
	if err := Bind(cfg, c); err == nil {
		t.Fatal("expected error for non-pointer target")
	}
}

func TestBindRejectsNilPointer(t *testing.T) {
	cfg, _ := Load(context.Background(), NewMapSource("test", map[string]string{}))

	type Config struct{}
	var c *Config
	if err := Bind(cfg, c); err == nil {
		t.Fatal("expected error for nil pointer target")
	}
}

func TestBindRejectsNonStruct(t *testing.T) {
	cfg, _ := Load(context.Background(), NewMapSource("test", map[string]string{}))

	s := "not a struct"
	if err := Bind(cfg, &s); err == nil {
		t.Fatal("expected error for non-struct pointer target")
	}
}

func TestBindAutoKeyFromFieldName(t *testing.T) {
	cfg, err := Load(context.Background(), NewMapSource("test", map[string]string{
		"port": "3000",
		"host": "example.com",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	type Config struct {
		Port int    `default:"8080"`
		Host string `default:"localhost"`
	}

	var c Config
	if err := Bind(cfg, &c); err != nil {
		t.Fatalf("bind: %v", err)
	}

	if c.Port != 3000 {
		t.Fatalf("port = %d, want 3000", c.Port)
	}
	if c.Host != "example.com" {
		t.Fatalf("host = %q, want example.com", c.Host)
	}
}

func TestBindInvalidInt(t *testing.T) {
	cfg, err := Load(context.Background(), NewMapSource("test", map[string]string{
		"port": "not-a-number",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	type Config struct {
		Port int `config:"port"`
	}

	var c Config
	if err := Bind(cfg, &c); err == nil {
		t.Fatal("expected error for invalid int value")
	}
}

func TestBindInvalidBool(t *testing.T) {
	cfg, err := Load(context.Background(), NewMapSource("test", map[string]string{
		"debug": "not-a-bool",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	type Config struct {
		Debug bool `config:"debug"`
	}

	var c Config
	if err := Bind(cfg, &c); err == nil {
		t.Fatal("expected error for invalid bool value")
	}
}
