package config_test

import (
	"context"
	"testing"

	"github.com/87nehal/vengo/config"
)

type BenchConfig struct {
	Name    string `config:"app.name" default:"hello"`
	Port    int    `config:"server.port" default:"8080"`
	Timeout int    `config:"server.timeout" default:"5"`
	Secure  bool   `config:"server.secure" default:"true"`
}

func BenchmarkConfigLoadAndBind(b *testing.B) {
	ctx := context.Background()
	values := map[string]string{
		"app.name":       "benchmark-app",
		"server.port":    "9999",
		"server.timeout": "10",
		"server.secure":  "false",
	}
	src := config.NewMapSource("bench-map", values)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg, err := config.Load(ctx, src)
		if err != nil {
			b.Fatal(err)
		}
		var target BenchConfig
		if err := config.Bind(cfg, &target); err != nil {
			b.Fatal(err)
		}
	}
}
