# internal/config

Resolves Tsugi's runtime configuration from the environment.

## Symbols

- `Config` (struct) — resolved runtime config. Fields: `Addr` (HTTP),
  `AgentAddr` (gRPC agent), `DatabaseURL`, `Target`, `DeployDir`.
- `Load(getenv func(string) string) (*Config, error)` — reads config through
  `getenv` (`os.Getenv` in production, a stub in tests). Empty values fall back
  to defaults; rejects an `Addr` that is not a valid `host:port` and an
  `AgentAddr` that is not loopback (the write plane must never be tunnelable).

## Environment

| Var | Default | Meaning |
|---|---|---|
| `TSUGI_ADDR` | `127.0.0.1:8090` | HTTP listen address |
| `TSUGI_AGENT_ADDR` | `127.0.0.1:8091` | gRPC agent address; loopback only |
| `TSUGI_DATABASE_URL` | — | Postgres URL; required by `serve` and the CLI |
| `TSUGI_TARGET` | `lazyscan` | deploy target name |
| `TSUGI_DEPLOY_DIR` | `deploy` | deploy root (`bin/`, `targets/`) |
