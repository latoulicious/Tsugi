package config

import (
	"fmt"
	"net"
	"strings"
)

type Config struct {
	Addr string // HTTP listen address (TSUGI_ADDR)
}

const defaultAddr = ":8080"

func Load(getenv func(string) string) (*Config, error) {
	cfg := &Config{
		Addr: envOr(getenv, "TSUGI_ADDR", defaultAddr),
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
