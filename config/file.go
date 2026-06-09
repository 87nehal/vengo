package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type FileSource struct {
	path string
}

func NewFileSource(path string) FileSource {
	return FileSource{path: path}
}

func (s FileSource) Name() string {
	return "file:" + s.path
}

func (s FileSource) Load(context.Context) (map[string]string, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("read config file %q: %w", s.path, err)
	}

	var raw map[string]any
	switch strings.ToLower(filepath.Ext(s.path)) {
	case ".toml":
		if err := toml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse toml %q: %w", s.path, err)
		}
	case ".json":
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse json %q: %w", s.path, err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file extension %q (use .toml or .json)", filepath.Ext(s.path))
	}

	values := make(map[string]string)
	flatten("", raw, values)
	return values, nil
}

func flatten(prefix string, raw map[string]any, out map[string]string) {
	for key, value := range raw {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		switch v := value.(type) {
		case map[string]any:
			flatten(fullKey, v, out)
		default:
			out[fullKey] = fmt.Sprint(value)
		}
	}
}
