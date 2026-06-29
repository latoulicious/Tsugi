package agent

import (
	"context"
	"testing"
	"time"

	"github.com/latoulicious/Tsugi/internal/agentpb"
	"github.com/latoulicious/Tsugi/internal/deployment"
	"github.com/latoulicious/Tsugi/internal/release"
)

type fakeReleases []*release.Release

func (f fakeReleases) List(context.Context) ([]*release.Release, error) { return f, nil }

type fakeDeployments []*deployment.Deployment

func (f fakeDeployments) List(context.Context) ([]*deployment.Deployment, error) { return f, nil }

func TestListReleasesDenormalizes(t *testing.T) {
	created := time.Unix(1000, 0)
	prodDeploy := time.Unix(2000, 0)
	rels := fakeReleases{
		mustRelease(t, 1, "v1.0.0", "aaaaaaa", release.StatusProduction, created),
		mustRelease(t, 2, "v1.1.0", "bbbbbbb", release.StatusStaging, created),
	}
	deps := fakeDeployments{
		mustDeployment(t, 10, 1, deployment.EnvProduction, deployment.StatusSucceeded, prodDeploy),
	}

	resp, err := New(rels, deps, "lazyscan").ListReleases(context.Background(), &agentpb.ListReleasesReq{})
	if err != nil {
		t.Fatalf("ListReleases: %v", err)
	}
	got := resp.GetReleases()
	if len(got) != 2 {
		t.Fatalf("releases = %d, want 2", len(got))
	}
	// production release: env from status, deployed_at from its deployment row.
	assertRelease(t, got[0], &agentpb.Release{
		Env: "production", Service: "lazyscan", Commit: "aaaaaaa",
		Tag: "v1.0.0", DeployedAt: 2000, Status: "production",
	})
	// staging release: no deployment row → deployed_at falls back to created_at.
	assertRelease(t, got[1], &agentpb.Release{
		Env: "staging", Service: "lazyscan", Commit: "bbbbbbb",
		Tag: "v1.1.0", DeployedAt: 1000, Status: "staging",
	})
}

func TestListDeploymentsJoinsCommit(t *testing.T) {
	rels := fakeReleases{
		mustRelease(t, 1, "v1.0.0", "aaaaaaa", release.StatusProduction, time.Unix(1000, 0)),
	}
	deps := fakeDeployments{
		mustDeployment(t, 10, 1, deployment.EnvProduction, deployment.StatusSucceeded, time.Unix(2000, 0)),
	}

	resp, err := New(rels, deps, "lazyscan").ListDeployments(context.Background(), &agentpb.ListDeploymentsReq{})
	if err != nil {
		t.Fatalf("ListDeployments: %v", err)
	}
	got := resp.GetDeployments()
	if len(got) != 1 {
		t.Fatalf("deployments = %d, want 1", len(got))
	}
	d := got[0]
	if d.GetEnv() != "production" || d.GetService() != "lazyscan" || d.GetCommit() != "aaaaaaa" ||
		d.GetStatus() != "succeeded" || d.GetDeployedAt() != 2000 {
		t.Errorf("deployment = %+v, want env=production service=lazyscan commit=aaaaaaa status=succeeded deployed_at=2000", d)
	}
}

func assertRelease(t *testing.T, got, want *agentpb.Release) {
	t.Helper()
	if got.GetEnv() != want.GetEnv() || got.GetService() != want.GetService() ||
		got.GetCommit() != want.GetCommit() || got.GetTag() != want.GetTag() ||
		got.GetDeployedAt() != want.GetDeployedAt() || got.GetStatus() != want.GetStatus() {
		t.Errorf("release = %+v, want %+v", got, want)
	}
}

func mustRelease(t *testing.T, id int64, version, commit string, s release.Status, createdAt time.Time) *release.Release {
	t.Helper()
	r, err := release.Rehydrate(id, version, commit, "", "", s, createdAt)
	if err != nil {
		t.Fatalf("rehydrate release: %v", err)
	}
	return r
}

func mustDeployment(t *testing.T, id, releaseID int64, env deployment.Environment, s deployment.Status, at time.Time) *deployment.Deployment {
	t.Helper()
	d, err := deployment.Rehydrate(id, releaseID, env, s, at)
	if err != nil {
		t.Fatalf("rehydrate deployment: %v", err)
	}
	return d
}
