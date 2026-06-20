package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/latoulicious/Tsugi/internal/cli"
	"github.com/latoulicious/Tsugi/internal/config"
	"github.com/latoulicious/Tsugi/internal/deploy"
	"github.com/latoulicious/Tsugi/internal/git"
	"github.com/latoulicious/Tsugi/internal/postgres"
	"github.com/latoulicious/Tsugi/internal/server"
	"github.com/latoulicious/Tsugi/internal/version"
)

const (
	shutdownTimeout = 10 * time.Second
	usage           = "usage: tsugi <serve|migrate|release|help>"
)

func main() {
	args := os.Args[1:]
	cmd := "serve"
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}
	var err error
	switch cmd {
	case "serve":
		err = run()
	case "migrate":
		err = runMigrate(args)
	case "release":
		err = runRelease(args)
	case "help", "--help", "-h":
		fmt.Println(usage)
	default:
		err = fmt.Errorf("unknown command %q\n%s", cmd, usage)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "tsugi:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	srv := server.New(cfg, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		v := version.Get()
		logger.Info("tsugi listening", "addr", cfg.Addr, "version", v.Version, "commit", v.Commit)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	logger.Info("shutting down", "timeout", shutdownTimeout)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	logger.Info("shutdown complete")
	return nil
}

func runMigrate(args []string) error {
	ctx := context.Background()
	pool, _, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()
	dir := "up"
	if len(args) > 0 {
		dir = args[0]
	}
	switch dir {
	case "up":
		return postgres.MigrateUp(ctx, pool)
	case "down":
		return postgres.MigrateDown(ctx, pool)
	default:
		return fmt.Errorf("migrate: unknown direction %q (up|down)", dir)
	}
}

func runRelease(args []string) error {
	ctx := context.Background()
	pool, cfg, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()
	store := postgres.New(pool)
	app := &cli.App{
		Releases:    store.Releases,
		Deployments: store.Deployments,
		Tx:          store,
		Git:         git.Default{},
		Deployer:    deploy.Script{BinDir: filepath.Join(cfg.DeployDir, "bin")},
		Target:      cfg.Target,
		// lazy so read-only commands don't require target.env
		StagingCheckout: func() (string, error) { return deploy.StagingCheckout(cfg.DeployDir, cfg.Target) },
		Out:             os.Stdout,
	}
	return app.Run(ctx, args)
}

// openStore loads config and opens the pool; both CLI paths need a database URL.
func openStore(ctx context.Context) (*pgxpool.Pool, *config.Config, error) {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return nil, nil, err
	}
	if cfg.DatabaseURL == "" {
		return nil, nil, errors.New("TSUGI_DATABASE_URL is required")
	}
	pool, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	return pool, cfg, nil
}
