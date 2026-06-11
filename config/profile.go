package config

import (
	"context"
	"os"
	"path/filepath"
)

var configExtensions = []string{".toml", ".json"}

var searchDirs = []string{".", "config"}

func DefaultSources(profile string) []Source {
	var sources []Source

	dirs := searchDirs
	if envDir := os.Getenv("VENGO_CONFIG_DIR"); envDir != "" {
		dirs = []string{envDir}
	}

	for _, dir := range dirs {
		for _, ext := range configExtensions {
			path := filepath.Join(dir, "application"+ext)
			if fileExists(path) {
				sources = append(sources, NewFileSource(path))
				break
			}
		}
	}

	if profile != "" {
		for _, dir := range dirs {
			for _, ext := range configExtensions {
				path := filepath.Join(dir, "application-"+profile+ext)
				if fileExists(path) {
					sources = append(sources, NewFileSource(path))
					break
				}
			}
		}
	}

	sources = append(sources, NewEnvSource("APP_"))

	return sources
}

func LoadDefaults(ctx context.Context, profile string) (*Config, error) {
	return Load(ctx, DefaultSources(profile)...)
}

func ActiveProfile() string {
	if profile := os.Getenv("APP_PROFILE"); profile != "" {
		return profile
	}
	if profile := os.Getenv("VENGO_PROFILE"); profile != "" {
		return profile
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
