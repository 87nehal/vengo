package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/87nehal/vengo/config"
	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/data"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

func main() {
	ctx := context.Background()

	cfg, err := config.Load(ctx, config.NewFileSource("examples/data-sql/application.toml"))
	if err != nil {
		cfg, err = config.Load(ctx, config.NewFileSource("application.toml"))
		if err != nil {
			log.Fatal(err)
		}
	}
	if os.Getenv("VENGO_DATA_SQL_DSN") != "" {
		cfg, err = config.Load(ctx, config.NewFileSource("examples/data-sql/application.toml"), config.NewEnvSource("VENGO_"))
		if err != nil {
			log.Fatal(err)
		}
	}

	app := core.New("data-sql", data.New(data.WithMigrations(migrations)))
	app.SetConfig(cfg)

	if err := app.Start(ctx); err != nil {
		log.Fatal(err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.Stop(shutdownCtx); err != nil {
			log.Printf("stop app: %v", err)
		}
	}()

	db, err := core.Get[*sql.DB](app, data.DBServiceName)
	if err != nil {
		log.Fatal(err)
	}

	manager := data.NewTxManager(db)
	if err := manager.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO widgets (name) VALUES (?)`, "example")
		return err
	}); err != nil {
		log.Fatal(err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM widgets`).Scan(&count); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("widgets: %d\n", count)
}
