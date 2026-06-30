package agent

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/latoulicious/Tsugi/internal/agentpb"
	"github.com/latoulicious/Tsugi/internal/deployflow"
	"github.com/latoulicious/Tsugi/internal/deployment"
	"github.com/latoulicious/Tsugi/internal/release"
)

// --- fakes (write plane) ---------------------------------------------------

// rwReleases is a writable release repo; UpdateStatus is a no-op because the
// state machine already mutated the shared pointer.
type rwReleases struct{ items []*release.Release }

func (r *rwReleases) Create(context.Context, *release.Release) error { return nil }
func (r *rwReleases) GetByVersion(context.Context, string) (*release.Release, error) {
	return nil, release.ErrNotFound
}
func (r *rwReleases) List(context.Context) ([]*release.Release, error)          { return r.items, nil }
func (r *rwReleases) UpdateStatus(context.Context, int64, release.Status) error { return nil }

type rwDeployments struct{ items []*deployment.Deployment }

func (d *rwDeployments) Create(_ context.Context, dep *deployment.Deployment) error {
	dep.ID = int64(len(d.items) + 1)
	d.items = append(d.items, dep)
	return nil
}
func (d *rwDeployments) List(context.Context) ([]*deployment.Deployment, error) { return d.items, nil }
func (d *rwDeployments) ListByEnvironment(context.Context, deployment.Environment) ([]*deployment.Deployment, error) {
	return d.items, nil
}
func (d *rwDeployments) UpdateStatus(_ context.Context, id int64, s deployment.Status) error {
	for _, x := range d.items {
		if x.ID == id {
			x.Status = s
		}
	}
	return nil
}

type rwTx struct {
	r release.Repository
	d deployment.Repository
}

func (t rwTx) WithTx(ctx context.Context, fn func(release.Repository, deployment.Repository) error) error {
	return fn(t.r, t.d)
}

type fakeDeployer struct {
	err error
	ref string
}

func (d *fakeDeployer) Run(_ context.Context, _, _, ref string, sink deployflow.LogSink) error {
	d.ref = ref
	sink.Line("stdout", "building...")
	return d.err
}

// recStream records the streamed log lines. Embeds the interface so the unused
// ServerStream methods exist; only Send/Context are exercised.
type recStream struct {
	grpc.ServerStream
	ctx   context.Context
	lines []*agentpb.LogLine
}

func (s *recStream) Send(l *agentpb.LogLine) error { s.lines = append(s.lines, l); return nil }
func (s *recStream) Context() context.Context      { return s.ctx }

// --- helpers ---------------------------------------------------------------

func writeServer(rels *rwReleases) (*Server, *rwDeployments, *fakeDeployer) {
	deps := &rwDeployments{}
	dep := &fakeDeployer{}
	flow := &deployflow.Service{
		Deployments: deps,
		Tx:          rwTx{rels, deps},
		Deployer:    dep,
		Target:      "lazyscan",
		Now:         func() time.Time { return time.Unix(0, 0) },
	}
	return New(rels, fakeDeployments{}, flow, "lazyscan"), deps, dep
}

func strptr(s string) *string { return &s }

func deployReq(service, env, commit string) *agentpb.DeployReq {
	return &agentpb.DeployReq{Service: service, Env: env, Commit: strptr(commit)}
}

// --- tests -----------------------------------------------------------------

func TestPromoteStreamsAndAdvances(t *testing.T) {
	rel := mustRelease(t, 1, "v1.0.0", "aaaaaaa1", release.StatusStaging, time.Unix(1000, 0))
	s, deps, dep := writeServer(&rwReleases{items: []*release.Release{rel}})

	st := &recStream{ctx: context.Background()}
	if err := s.Promote(deployReq("lazyscan", "production", "aaaaaaa1"), st); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if rel.Status() != release.StatusProduction {
		t.Fatalf("status = %q, want production", rel.Status())
	}
	if dep.ref != "aaaaaaa1" {
		t.Fatalf("deployed ref %q, want aaaaaaa1", dep.ref)
	}
	if len(st.lines) == 0 {
		t.Fatal("no log lines streamed")
	}
	if len(deps.items) != 1 || deps.items[0].Status != deployment.StatusSucceeded {
		t.Fatalf("deployment = %+v, want one succeeded", deps.items)
	}
}

func TestPromoteFailureMarksFailed(t *testing.T) {
	rel := mustRelease(t, 1, "v1.0.0", "aaaaaaa1", release.StatusStaging, time.Unix(1000, 0))
	s, deps, dep := writeServer(&rwReleases{items: []*release.Release{rel}})
	dep.err = context.DeadlineExceeded

	err := s.Promote(deployReq("lazyscan", "production", "aaaaaaa1"), &recStream{ctx: context.Background()})
	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want Internal", status.Code(err))
	}
	if rel.Status() != release.StatusStaging {
		t.Fatalf("status = %q, want staging (no advance on failed deploy)", rel.Status())
	}
	if len(deps.items) != 1 || deps.items[0].Status != deployment.StatusFailed {
		t.Fatalf("deployment = %+v, want one failed", deps.items)
	}
}

func TestRollbackRequiresArchived(t *testing.T) {
	rel := mustRelease(t, 1, "v1.0.0", "aaaaaaa1", release.StatusProduction, time.Unix(1000, 0))
	s, _, _ := writeServer(&rwReleases{items: []*release.Release{rel}})

	err := s.Rollback(deployReq("lazyscan", "production", "aaaaaaa1"), &recStream{ctx: context.Background()})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("code = %v, want FailedPrecondition", status.Code(err))
	}
}

func TestPromoteRejectsBadRequest(t *testing.T) {
	rel := mustRelease(t, 1, "v1.0.0", "aaaaaaa1", release.StatusStaging, time.Unix(1000, 0))
	s, _, _ := writeServer(&rwReleases{items: []*release.Release{rel}})

	cases := map[string]*agentpb.DeployReq{
		"wrong service": deployReq("other", "production", "aaaaaaa1"),
		"non-prod env":  deployReq("lazyscan", "staging", "aaaaaaa1"),
		"bad commit":    deployReq("lazyscan", "production", "not-a-sha"),
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			if got := status.Code(s.Promote(req, &recStream{ctx: context.Background()})); got != codes.InvalidArgument {
				t.Fatalf("code = %v, want InvalidArgument", got)
			}
		})
	}
}

func TestPromoteUnknownCommit(t *testing.T) {
	s, _, _ := writeServer(&rwReleases{})
	err := s.Promote(deployReq("lazyscan", "production", "abcdef1"), &recStream{ctx: context.Background()})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("code = %v, want NotFound", status.Code(err))
	}
}
