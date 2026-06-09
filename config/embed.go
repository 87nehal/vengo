package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type EmbedSource struct {
	fsys fs.FS
	path string
}

func NewEmbedSource(fsys fs.FS, path string) EmbedSource {
	return EmbedSource{fsys: fsys, path: path}
}

func (s EmbedSource) Name() string {
	return "embed:" + s.path
}

func (s EmbedSource) Load(context.Context) (map[string]string, error) {
	data, err := fs.ReadFile(s.fsys, s.path)
	if err != nil {
		return nil, fmt.Errorf("read embedded config %q: %w", s.path, err)
	}

	var raw map[string]any
	switch strings.ToLower(filepath.Ext(s.path)) {
	case ".toml":
		if err := toml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse embedded toml %q: %w", s.path, err)
		}
	case ".json":
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse embedded json %q: %w", s.path, err)
		}
	default:
		return nil, fmt.Errorf("unsupported embedded config extension %q (use .toml or .json)", filepath.Ext(s.path))
	}

	values := make(map[string]string)
	flatten("", raw, values)
	return values, nil
}
