# internal/config

Resolves Tsugi's runtime configuration from the environment.

## Symbols

- `Config` (struct) — resolved runtime config. Fields: `Addr` (HTTP listen
  address).
- `Load(getenv func(string) string) (*Config, error)` — reads config through
  `getenv` (`os.Getenv` in production, a stub in tests). Empty values fall back
  to defaults; rejects an `Addr` that is not a valid `host:port`.

## Environment

| Var | Default | Meaning |
|---|---|---|
| `TSUGI_ADDR` | `:8080` | HTTP listen address |
