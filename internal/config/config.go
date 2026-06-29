package config

import (
	"fmt"
	"net"
	"strings"
)

type Config struct {
	Addr        string // HTTP listen address (TSUGI_ADDR)
	AgentAddr   string // gRPC agent listen address (TSUGI_AGENT_ADDR); loopback only, never tunnel
	DatabaseURL string // Postgres URL (TSUGI_DATABASE_URL); required by the CLI and serve (hosts the agent)
	Target      string // deploy target name (TSUGI_TARGET)
	DeployDir   string // deploy/ root holding bin/ and targets/ (TSUGI_DEPLOY_DIR)
}

const (
	defaultAddr      = "127.0.0.1:8090" // loopback (tunnel fronts it); :8080 is dozzle on the box
	defaultAgentAddr = "127.0.0.1:8091" // write-plane gRPC; loopback only, never tunneled
	defaultTarget    = "lazyscan"
	defaultDeployDir = "deploy"
)

func Load(getenv func(string) string) (*Config, error) {
	cfg := &Config{
		Addr:        envOr(getenv, "TSUGI_ADDR", defaultAddr),
		AgentAddr:   envOr(getenv, "TSUGI_AGENT_ADDR", defaultAgentAddr),
		DatabaseURL: strings.TrimSpace(getenv("TSUGI_DATABASE_URL")),
		Target:      envOr(getenv, "TSUGI_TARGET", defaultTarget),
		DeployDir:   envOr(getenv, "TSUGI_DEPLOY_DIR", defaultDeployDir),
	}
	if _, _, err := net.SplitHostPort(cfg.Addr); err != nil {
		return nil, fmt.Errorf("TSUGI_ADDR %q: %w", cfg.Addr, err)
	}
	if err := requireLoopback("TSUGI_AGENT_ADDR", cfg.AgentAddr); err != nil {
		return nil, err
	}
	return cfg, nil
}

// requireLoopback rejects any agent bind that isn't a literal loopback IP — a
// hostname would defer the off-box guarantee to /etc/hosts (write plane).
func requireLoopback(key, addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("%s %q: %w", key, addr, err)
	}
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("%s %q: must bind a loopback IP (127.0.0.1 or ::1)", key, addr)
	}
	return nil
}

func envOr(getenv func(string) string, key, fallback string) string {
	if v := strings.TrimSpace(getenv(key)); v != "" {
		return v
	}
	return fallback
}
