package config

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Source interface {
	Name() string
	Load(context.Context) (map[string]string, error)
}

type Entry struct {
	Key      string
	Value    string
	Source   string
	Redacted bool
}

type Config struct {
	values map[string]Entry
}

func Load(ctx context.Context, sources ...Source) (*Config, error) {
	config := &Config{values: make(map[string]Entry)}
	for _, source := range sources {
		if source == nil {
			return nil, fmt.Errorf("config source cannot be nil")
		}
		values, err := source.Load(ctx)
		if err != nil {
			return nil, fmt.Errorf("load config source %q: %w", source.Name(), err)
		}
		for key, value := range values {
			config.values[key] = Entry{Key: key, Value: value, Source: source.Name()}
		}
	}
	return config, nil
}

func (c *Config) Get(key string) (string, bool) {
	entry, exists := c.values[key]
	return entry.Value, exists
}

func (c *Config) Must(key string, fallback string) string {
	value, exists := c.Get(key)
	if !exists {
		return fallback
	}
	return value
}

func (c *Config) Report() []Entry {
	report := make([]Entry, 0, len(c.values))
	for _, entry := range c.values {
		if sensitiveKey(entry.Key) {
			entry.Value = "<redacted>"
			entry.Redacted = true
		}
		report = append(report, entry)
	}
	sort.Slice(report, func(left, right int) bool {
		return report[left].Key < report[right].Key
	})
	return report
}

func (c *Config) Keys() []string {
	keys := make([]string, 0, len(c.values))
	for key := range c.values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (c *Config) SourceOf(key string) (string, bool) {
	entry, exists := c.values[key]
	if !exists {
		return "", false
	}
	return entry.Source, true
}

type MapSource struct {
	name   string
	values map[string]string
}

func NewMapSource(name string, values map[string]string) MapSource {
	copyValues := make(map[string]string, len(values))
	for key, value := range values {
		copyValues[key] = value
	}
	return MapSource{name: name, values: copyValues}
}

func (s MapSource) Name() string {
	return s.name
}

func (s MapSource) Load(context.Context) (map[string]string, error) {
	values := make(map[string]string, len(s.values))
	for key, value := range s.values {
		values[key] = value
	}
	return values, nil
}

type EnvSource struct {
	prefix string
}

func NewEnvSource(prefix string) EnvSource {
	return EnvSource{prefix: prefix}
}

func (s EnvSource) Name() string {
	if s.prefix == "" {
		return "env"
	}
	return "env:" + s.prefix
}

func (s EnvSource) Load(context.Context) (map[string]string, error) {
	values := map[string]string{}
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if s.prefix != "" {
			if !strings.HasPrefix(key, s.prefix) {
				continue
			}
			key = strings.TrimPrefix(key, s.prefix)
		}
		key = strings.ToLower(strings.ReplaceAll(key, "_", "."))
		values[key] = value
	}
	return values, nil
}

func sensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	markers := []string{"password", "secret", "token", "credential", "api.key", "apikey", "private.key"}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
