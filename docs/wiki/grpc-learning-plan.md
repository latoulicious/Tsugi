# Tsugi × gRPC — Learning Plan

> **Status: learning fork, not roadmap.** Overkill for one VPS, on purpose.
> Branch-only (`feat/grpc-agent`). Goal: full gRPC surface — unary → server-stream
> → bidi → mTLS — mapped onto Tsugi's real deploy flow. Does not replace the
> shipping single-binary deployer until/unless a second host appears.

## Topology (the big picture)

Agent lives **per host**, not per project. A host runs many target projects; one
`tsugi-agent` daemon per host manages all their compose stacks.

```
        you / CI  ──►  tsugi  (control plane: CLI + release state in PG)
                          │
                          │  tsugi promote lazyscan v1.2      ← ONE target, normally
                          ▼  gRPC (mTLS)
        ┌─────────────────┼──────────────────┐
        ▼                 ▼                   ▼
     host A            host B              host C
   tsugi-agent       tsugi-agent         tsugi-agent      ← 1 agent / host
        │                 │                   │
   docker compose:   docker compose:     docker compose:
   LazyScan, Tsugi   Kanjo               Koutei            ← many projects / host
```

- `tsugi promote <target> <version>` → control plane resolves *which host* the
  target lives on → **one** gRPC call to that host's agent → agent runs that
  project's compose. Not "every project".
- **Fan-out / batch** ("release everything") = a `for` loop over targets calling
  each host's agent in parallel. A broadcast op layered on the single-target
  path — build the unit first.
- Today Tsugi runs on the same VPS as LazyScan → control plane + agent same box,
  gRPC over localhost. Multi-host is the growth story the agent unlocks.

## Why it fits (the seam)

Tsugi today **fuses two jobs in one binary**:
- *decides* — `internal/release` state machine (`Draft→Created→Staging→Production→Archived`), state in PG.
- *executes* — `internal/deploy.Script` shells `deploy/bin/deploy.sh`, streams stdout to `os.Stdout`.

The seam already exists: `cli.Deployer` interface (`Run(ctx, target, env, ref) error`).
gRPC splits decide vs execute across a wire:

- **`tsugi`** (existing) = control plane. Owns release state + state machine. Touches no Docker.
- **`tsugi-agent`** (new) = per-host daemon. Only thing that runs `docker compose` + `git checkout`.

Existing verbs map ~1:1 to RPCs:

| Tsugi today | RPC | Type | Concept learned |
|---|---|---|---|
| `GET /version` (server.go) | `GetVersion()` | unary | proto→gen→serve→call loop |
| `release promote` → `Deployer.Run` | `Deploy(spec) → stream LogLine` | server-streaming | live deploy output over wire |
| `release rollback` (ref set) | `Rollback(spec) → stream LogLine` | server-streaming | same RPC, ref-pinned |
| health gate (new) | `Promote(stream) ↔ (progress, health)` | bidi + deadline | abort + auto-rollback |
| all of above | interceptors | mTLS + audit | privileged-daemon auth |

## Files added (additive — nothing existing rewritten until P2)

```
proto/agent.proto                 # service Agent { GetVersion, Deploy, Rollback, Promote }
internal/agentpb/                 # generated (protoc-gen-go + -go-grpc)
cmd/tsugi-agent/main.go           # the daemon binary
internal/agentserver/server.go    # implements Agent svc; wraps internal/version + internal/deploy
internal/deploy/grpcclient.go     # NEW cli.Deployer impl: gRPC client, streams logs to stdout
Makefile                          # +proto target
```

`internal/deploy.Script` stays — config picks local-exec vs gRPC. No regression.

## Phases (each runnable, one concept)

### P0 — tooling
- Add `proto/agent.proto`, install `protoc-gen-go` + `protoc-gen-go-grpc` (or `buf`).
- `make proto` → generates `internal/agentpb/`.
- Deps: `google.golang.org/grpc`, `google.golang.org/protobuf`.
- Done = generated stubs compile.

### P1 — `GetVersion` unary  ← start here
- `agent.proto`: `rpc GetVersion(Empty) returns (VersionInfo)` mirroring `version.Info{version,commit,deployed_at}`.
- `internal/agentserver`: implement, return `version.Get()`. Same data Tsugi already serves over HTTP — new transport only.
- `cmd/tsugi-agent`: serve on `127.0.0.1:9090`.
- Control-plane: tiny client, new CLI verb `tsugi agent-version` calls it, prints.
- **Proves the whole loop end to end. Smallest possible win.**

### P2 — `Deploy` server-streaming
- `rpc Deploy(DeploySpec) returns (stream LogLine)`. `DeploySpec{target, env, ref}` = exact args of `deploy.Run`.
- Agent: run `deploy.sh` via `exec.CommandContext`, scan stdout, send each line as `LogLine`. (Refactor `deploy.Run` to take an `io.Writer`/line callback instead of hardcoded `os.Stdout`.)
- `internal/deploy/grpcclient.go`: implements `cli.Deployer`; opens stream, prints `LogLine`s to stdout → drop-in for `Script`.
- Wire `App.Deployer = GRPCClient` behind a config flag (`TSUGI_AGENT_ADDR`). Unset = local `Script` (unchanged).
- **Concept: server-streaming, ctx cancellation propagates to remote `exec`.**

### P3 — `Rollback`
- Same RPC shape, `ref` pinned to a prior commit. Reuses Deploy plumbing. Cheap once P2 lands.

### P4 — mTLS + interceptors
- Agent runs Docker = privileged → MUST authenticate caller.
- `credentials.NewTLS` mutual auth, self-signed CA in `deploy/` (sandbox).
- Unary + stream interceptor: audit-log every RPC (who/what/when) via `slog`.
- **Concept: gRPC security + middleware. Real reason, not toy.**

### P5 — bidi health-gate (the payoff)
- `rpc Promote(stream PromoteMsg) returns (stream PromoteEvent)`.
- Control plane streams "go", agent streams `{progress, container_health}` from target's `/healthz`.
- Control plane sets a deadline; unhealthy in N s → send "abort" → agent rolls back. Auto-rollback over bidi.
- **Concept: bidirectional streaming + deadline + cancellation. The hard one.**

## Lazy guardrails
- `ponytail:` comment on the agent: "single host today; agent split earns it at host #2 / docker-isolation. Learning-forward."
- One self-check per phase: P1 a `cmd/tsugi-agent` smoke test (dial + GetVersion asserts version != ""), P2 assert N stdin lines stream back in order.
- Don't build service discovery / TLS rotation / multi-agent registry — YAGNI until host #2 is real.
