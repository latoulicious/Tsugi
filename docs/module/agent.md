# internal/agent

Tsugi's write-plane gRPC server (Yagura PLAN §11): the localhost-only act-plane
Yagura's read plane dials. Contract: `proto/tsugi_agent.proto` (`package
tsugi.agent.v1`); generated stubs in `internal/agentpb`.

## Symbols

- `New(releases, deployments, flow *deployflow.Service, service string) *Server` —
  builds the service over two narrow read interfaces (`List(ctx)` on releases and
  deployments), the shared deploy orchestrator `flow` (write RPCs), and the target
  name surfaced as `service`. `flow` may be nil for read-only servers.
- `Server` — implements `agentpb.TsugiAgentServer`. Embeds
  `UnimplementedTsugiAgentServer`, so unimplemented RPCs return
  `codes.Unimplemented`. A `sync.Mutex` serializes the write RPCs (single-flight).

## RPCs

| RPC | State | Maps to |
|---|---|---|
| `ListReleases` | implemented (P5.2) | releases ⨝ latest deployment; env derived from release status, `deployed_at` from the deployment row else `created_at` |
| `ListDeployments` | implemented (P5.2) | deployment history, `commit` joined from the release |
| `Promote` | implemented (P5.4) | resolve release by commit, guard `staging`, stream `deployflow.ToProduction` |
| `Rollback` | implemented (P5.4) | resolve release by commit, guard `archived`, stream `deployflow.ToProduction` |
| `Deploy` | `Unimplemented` | staging deploy has no release-lifecycle hook; deferred |

## Write RPCs (P5.4)

`Promote`/`Rollback` validate at the boundary (service match, `production` env,
hex commit mirroring `deploy.sh`'s `--ref` guard), take the single-flight lock,
resolve the release by `CommitSHA`, guard its status, then stream the shared
`internal/deployflow` orchestrator. The deploy runs under
`context.WithoutCancel` + a 15m cap so a dropped client never aborts it mid-build;
the deployment row's outcome lands regardless.

## Serving

Registered in `cmd/tsugi serve` on `TSUGI_AGENT_ADDR` (loopback only, default
`127.0.0.1:8091`), alongside the HTTP server. Reflection is enabled so `grpcurl`
can introspect it on the box. Never tunneled — the write plane stays on-host.
