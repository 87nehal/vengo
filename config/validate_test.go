package config

import (
	"context"
	"errors"
	"testing"
)

func TestValidateRequiredString(t *testing.T) {
	type Config struct {
		Name string `validate:"required"`
	}

	c := Config{Name: ""}
	if err := Validate(&c); err == nil {
		t.Fatal("expected validation error for empty required string")
	}

	c.Name = "hello"
	if err := Validate(&c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateNonZero(t *testing.T) {
	type Config struct {
		Port int `validate:"nonzero"`
	}

	c := Config{Port: 0}
	if err := Validate(&c); err == nil {
		t.Fatal("expected validation error for zero int")
	}

	c.Port = 8080
	if err := Validate(&c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateNestedStruct(t *testing.T) {
	type DB struct {
		Host string `validate:"required"`
	}
	type Config struct {
		DB DB
	}

	c := Config{DB: DB{Host: ""}}
	if err := Validate(&c); err == nil {
		t.Fatal("expected validation error for nested required field")
	}

	c.DB.Host = "localhost"
	if err := Validate(&c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCustomInterface(t *testing.T) {
	type Config struct {
		Port int
	}

	c := &validatingConfig{inner: Config{Port: 0}}
	if err := Validate(c); err == nil {
		t.Fatal("expected custom validation error")
	}

	c.inner.Port = 8080
	if err := Validate(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

type validatingConfig struct {
	inner struct {
		Port int
	}
}

func (v *validatingConfig) Validate() error {
	if v.inner.Port <= 0 {
		return errors.New("port must be positive")
	}
	return nil
}

func TestBindAndValidate(t *testing.T) {
	cfg, err := Load(context.Background(), NewMapSource("test", map[string]string{
		"server.port": "9090",
		"server.host": "localhost",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	type ServerConfig struct {
		Port int    `config:"server.port" validate:"nonzero"`
		Host string `config:"server.host" validate:"required"`
	}

	var s ServerConfig
	if err := BindAndValidate(cfg, &s); err != nil {
		t.Fatalf("bind and validate: %v", err)
	}
	if s.Port != 9090 {
		t.Fatalf("port = %d, want 9090", s.Port)
	}
}

func TestBindAndValidateFailsOnMissingRequired(t *testing.T) {
	cfg, err := Load(context.Background(), NewMapSource("test", map[string]string{
		"server.port": "9090",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	type ServerConfig struct {
		Port int    `config:"server.port"`
		Host string `config:"server.host" validate:"required"`
	}

	var s ServerConfig
	if err := BindAndValidate(cfg, &s); err == nil {
		t.Fatal("expected validation error for missing required field")
	}
}

func TestValidateRejectsNil(t *testing.T) {
	type Config struct{}
	var c *Config
	if err := Validate(c); err == nil {
		t.Fatal("expected error for nil target")
	}
}

func TestValidateRejectsNonStruct(t *testing.T) {
	s := "not a struct"
	if err := Validate(&s); err == nil {
		t.Fatal("expected error for non-struct target")
	}
}
