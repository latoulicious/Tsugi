package config

import (
	"fmt"
	"net"
	"strings"
)

type Config struct {
	Addr        string // HTTP listen address (TSUGI_ADDR)
	DatabaseURL string // Postgres URL (TSUGI_DATABASE_URL); required by the CLI, not serve
	Target      string // deploy target name (TSUGI_TARGET)
	DeployDir   string // deploy/ root holding bin/ and targets/ (TSUGI_DEPLOY_DIR)
}

const (
	defaultAddr      = ":8080"
	defaultTarget    = "lazyscan"
	defaultDeployDir = "deploy"
)

func Load(getenv func(string) string) (*Config, error) {
	cfg := &Config{
		Addr:        envOr(getenv, "TSUGI_ADDR", defaultAddr),
		DatabaseURL: strings.TrimSpace(getenv("TSUGI_DATABASE_URL")),
		Target:      envOr(getenv, "TSUGI_TARGET", defaultTarget),
		DeployDir:   envOr(getenv, "TSUGI_DEPLOY_DIR", defaultDeployDir),
	}
	if _, _, err := net.SplitHostPort(cfg.Addr); err != nil {
		return nil, fmt.Errorf("TSUGI_ADDR %q: %w", cfg.Addr, err)
	}
	return cfg, nil
}

func envOr(getenv func(string) string, key, fallback string) string {
	if v := strings.TrimSpace(getenv(key)); v != "" {
		return v
	}
	return fallback
}
