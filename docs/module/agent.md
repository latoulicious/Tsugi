# internal/agent

Tsugi's write-plane gRPC server (Yagura PLAN §11): the localhost-only act-plane
Yagura's read plane dials. Contract: `proto/tsugi_agent.proto` (`package
tsugi.agent.v1`); generated stubs in `internal/agentpb`.

## Symbols

- `New(releases, deployments, service string) *Server` — builds the service over
  two narrow read interfaces (`List(ctx)` on releases and deployments) plus the
  target name surfaced as `service`.
- `Server` — implements `agentpb.TsugiAgentServer`. Embeds
  `UnimplementedTsugiAgentServer`, so unimplemented RPCs return
  `codes.Unimplemented`.

## RPCs

| RPC | State | Maps to |
|---|---|---|
| `ListReleases` | implemented (P5.2) | releases ⨝ latest deployment; env derived from release status, `deployed_at` from the deployment row else `created_at` |
| `ListDeployments` | implemented (P5.2) | deployment history, `commit` joined from the release |
| `Deploy` / `Rollback` / `Promote` | `Unimplemented` (P5.4) | server-streamed `LogLine` |

## Serving

Registered in `cmd/tsugi serve` on `TSUGI_AGENT_ADDR` (loopback only, default
`127.0.0.1:8091`), alongside the HTTP server. Reflection is enabled so `grpcurl`
can introspect it on the box. Never tunneled — the write plane stays on-host.
