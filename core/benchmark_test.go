package core_test

import (
	"context"
	"testing"

	"github.com/87nehal/vengo/core"
)

type DepA struct{ Val string }
type DepB struct {
	A *DepA `inject:""`
}
type DepC struct {
	B *DepB `inject:""`
}

func BenchmarkAppStartupAndResolve(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app := core.New("bench-app")

		_ = core.Provide(app, func() *DepA {
			return &DepA{Val: "bench"}
		})
		_ = core.Provide(app, func() *DepB {
			return &DepB{}
		})
		_ = core.Provide(app, func() *DepC {
			return &DepC{}
		})

		_ = app.Start(ctx)
		_, _ = core.Resolve[*DepC](app)
		_ = app.Stop(ctx)
	}
}
