package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/latoulicious/Tsugi/internal/deployment"
	"github.com/latoulicious/Tsugi/internal/release"
)

// --- fakes -----------------------------------------------------------------

type fakeReleases struct{ items []*release.Release }

func (f *fakeReleases) Create(_ context.Context, r *release.Release) error {
	r.ID = int64(len(f.items) + 1)
	f.items = append(f.items, r)
	return nil
}

func (f *fakeReleases) GetByVersion(_ context.Context, v string) (*release.Release, error) {
	for _, r := range f.items {
		if r.Version == v {
			return r, nil
		}
	}
	return nil, release.ErrNotFound
}

// List returns newest-first (matches the pgx ORDER BY created_at DESC).
func (f *fakeReleases) List(context.Context) ([]*release.Release, error) {
	out := make([]*release.Release, 0, len(f.items))
	for i := len(f.items) - 1; i >= 0; i-- {
		out = append(out, f.items[i])
	}
	return out, nil
}

// UpdateStatus is a no-op: the cli already mutated the shared pointer via the
// state machine; the fake holds that same pointer.
func (f *fakeReleases) UpdateStatus(context.Context, int64, release.Status) error { return nil }

type fakeDeployments struct{ items []*deployment.Deployment }

func (f *fakeDeployments) Create(_ context.Context, d *deployment.Deployment) error {
	d.ID = int64(len(f.items) + 1)
	f.items = append(f.items, d)
	return nil
}
func (f *fakeDeployments) List(context.Context) ([]*deployment.Deployment, error) {
	return f.items, nil
}
func (f *fakeDeployments) ListByEnvironment(context.Context, deployment.Environment) ([]*deployment.Deployment, error) {
	return f.items, nil
}
func (f *fakeDeployments) UpdateStatus(context.Context, int64, deployment.Status) error { return nil }

type fakeTx struct {
	r *fakeReleases
	d *fakeDeployments
}

func (t fakeTx) WithTx(ctx context.Context, fn func(release.Repository, deployment.Repository) error) error {
	return fn(t.r, t.d)
}

type fakeGit struct {
	head     string
	subjects []string
}

func (g fakeGit) HeadSHA(context.Context, string) (string, error) { return g.head, nil }
func (g fakeGit) Subjects(context.Context, string, string, string) ([]string, error) {
	return g.subjects, nil
}

type fakeDeployer struct {
	err  error
	ref  string
	runs int
}

func (d *fakeDeployer) Run(_ context.Context, _, _, ref string) error {
	d.runs++
	d.ref = ref
	return d.err
}

// --- helpers ---------------------------------------------------------------

func newApp(g fakeGit, dep *fakeDeployer) (*App, *fakeReleases, *fakeDeployments, *bytes.Buffer) {
	rels := &fakeReleases{}
	deps := &fakeDeployments{}
	buf := &bytes.Buffer{}
	app := &App{
		Releases:        rels,
		Deployments:     deps,
		Tx:              fakeTx{rels, deps},
		Git:             g,
		Deployer:        dep,
		Target:          "lazyscan",
		StagingCheckout: func() (string, error) { return "staging", nil },
		Now:             func() time.Time { return time.Unix(0, 0).UTC() },
		Out:             buf,
	}
	return app, rels, deps, buf
}

// --- tests -----------------------------------------------------------------

func TestCreate(t *testing.T) {
	ctx := context.Background()
	app, rels, _, _ := newApp(fakeGit{head: "abcdef1234", subjects: []string{"feat(x): add thing"}}, &fakeDeployer{})

	if err := app.Create(ctx, "v1.0.0"); err != nil {
		t.Fatal(err)
	}
	if len(rels.items) != 1 {
		t.Fatalf("want 1 release, got %d", len(rels.items))
	}
	r := rels.items[0]
	if r.Status() != release.StatusStaging {
		t.Fatalf("status = %q, want staging", r.Status())
	}
	if r.CommitSHA != "abcdef1234" || !strings.Contains(r.Changelog, "add thing") {
		t.Fatalf("commit/changelog wrong: %q / %q", r.CommitSHA, r.Changelog)
	}
}

func TestPromoteRequiresStaging(t *testing.T) {
	ctx := context.Background()
	app, rels, _, _ := newApp(fakeGit{head: "abcdef1234"}, &fakeDeployer{})
	_ = app.Create(ctx, "v1.0.0")
	if err := app.Promote(ctx, "v1.0.0"); err != nil { // staging -> production ok
		t.Fatal(err)
	}
	// now production, promoting again must fail
	if err := app.Promote(ctx, "v1.0.0"); err == nil {
		t.Fatal("expected error promoting a non-staging release")
	}
	if rels.items[0].Status() != release.StatusProduction {
		t.Fatalf("status = %q, want production", rels.items[0].Status())
	}
}

func TestPromoteArchivesPrevious(t *testing.T) {
	ctx := context.Background()
	dep := &fakeDeployer{}
	app, rels, deps, _ := newApp(fakeGit{head: "aaaaaaa1"}, dep)
	_ = app.Create(ctx, "v1.0.0")
	if err := app.Promote(ctx, "v1.0.0"); err != nil {
		t.Fatal(err)
	}
	// second release supersedes the first
	app.Git = fakeGit{head: "bbbbbbb2"}
	_ = app.Create(ctx, "v1.1.0")
	if err := app.Promote(ctx, "v1.1.0"); err != nil {
		t.Fatal(err)
	}

	first := mustGet(t, rels, "v1.0.0")
	second := mustGet(t, rels, "v1.1.0")
	if first.Status() != release.StatusArchived {
		t.Fatalf("v1.0.0 = %q, want archived", first.Status())
	}
	if second.Status() != release.StatusProduction {
		t.Fatalf("v1.1.0 = %q, want production", second.Status())
	}
	if got := countSucceeded(deps); got != 2 {
		t.Fatalf("succeeded deployments = %d, want 2", got)
	}
}

func TestRollback(t *testing.T) {
	ctx := context.Background()
	dep := &fakeDeployer{}
	app, rels, _, _ := newApp(fakeGit{head: "aaaaaaa1"}, dep)
	_ = app.Create(ctx, "v1.0.0")
	_ = app.Promote(ctx, "v1.0.0")
	app.Git = fakeGit{head: "bbbbbbb2"}
	_ = app.Create(ctx, "v1.1.0")
	_ = app.Promote(ctx, "v1.1.0") // v1.0.0 now archived

	if err := app.Rollback(ctx, "v1.0.0"); err != nil {
		t.Fatal(err)
	}
	if dep.ref != "aaaaaaa1" {
		t.Fatalf("rollback deployed ref %q, want the v1.0.0 commit", dep.ref)
	}
	if mustGet(t, rels, "v1.0.0").Status() != release.StatusProduction {
		t.Fatal("v1.0.0 should be back in production")
	}
	if mustGet(t, rels, "v1.1.0").Status() != release.StatusArchived {
		t.Fatal("v1.1.0 should be archived after rollback")
	}
}

func TestDeployFailureKeepsStaging(t *testing.T) {
	ctx := context.Background()
	dep := &fakeDeployer{err: errors.New("compose boom")}
	app, rels, deps, _ := newApp(fakeGit{head: "aaaaaaa1"}, dep)
	_ = app.Create(ctx, "v1.0.0")

	if err := app.Promote(ctx, "v1.0.0"); err == nil {
		t.Fatal("expected deploy error")
	}
	if rels.items[0].Status() != release.StatusStaging {
		t.Fatalf("status = %q, want staging (no advance on failed deploy)", rels.items[0].Status())
	}
	if deps.items[0].Status != deployment.StatusFailed {
		t.Fatalf("deployment = %q, want failed", deps.items[0].Status)
	}
}

func mustGet(t *testing.T, r *fakeReleases, v string) *release.Release {
	t.Helper()
	rel, err := r.GetByVersion(context.Background(), v)
	if err != nil {
		t.Fatal(err)
	}
	return rel
}

func countSucceeded(d *fakeDeployments) int {
	n := 0
	for _, x := range d.items {
		if x.Status == deployment.StatusSucceeded {
			n++
		}
	}
	return n
}
