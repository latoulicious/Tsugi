// Package agent is the Tsugi write-plane gRPC server (PLAN §11): the localhost-only
// act-plane Yagura dials. Contract lives in proto/tsugi_agent.proto.
package agent

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/latoulicious/Tsugi/internal/agentpb"
	"github.com/latoulicious/Tsugi/internal/deployment"
	"github.com/latoulicious/Tsugi/internal/release"
)

// releaseLister and deploymentLister are the read slices the agent needs; narrow
// consumer interfaces keep the write repos out of the read plane (ISP).
type releaseLister interface {
	List(ctx context.Context) ([]*release.Release, error)
}

type deploymentLister interface {
	List(ctx context.Context) ([]*deployment.Deployment, error)
}

// Server implements agentpb.TsugiAgentServer: read RPCs here (P5.2);
// Deploy/Rollback/Promote stay Unimplemented via the embedded stub (P5.4).
type Server struct {
	agentpb.UnimplementedTsugiAgentServer
	releases    releaseLister
	deployments deploymentLister
	service     string // target app name, surfaced as Release.service / Deployment.service
}

func New(releases releaseLister, deployments deploymentLister, service string) *Server {
	return &Server{releases: releases, deployments: deployments, service: service}
}

var _ agentpb.TsugiAgentServer = (*Server)(nil)

// ListReleases denormalizes each release with its latest deployment time (the
// proto Release is a release ⨝ deployment view).
func (s *Server) ListReleases(ctx context.Context, _ *agentpb.ListReleasesReq) (*agentpb.ListReleasesResp, error) {
	rels, err := s.releases.List(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list releases: %v", err)
	}
	deps, err := s.deployments.List(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list deployments: %v", err)
	}
	latest := latestByRelease(deps)
	out := make([]*agentpb.Release, 0, len(rels))
	for _, r := range rels {
		deployedAt := r.CreatedAt.Unix() // staging deploy time == create time; prod uses its deployment row
		if ts, ok := latest[r.ID]; ok {
			deployedAt = ts
		}
		out = append(out, &agentpb.Release{
			Env:        envForStatus(r.Status()),
			Service:    s.service,
			Commit:     r.CommitSHA,
			Tag:        r.Version,
			DeployedAt: deployedAt,
			Status:     string(r.Status()),
		})
	}
	return &agentpb.ListReleasesResp{Releases: out}, nil
}

// ListDeployments returns deployment history, joining each row to its release's
// commit (the deployments table holds only release_id).
func (s *Server) ListDeployments(ctx context.Context, _ *agentpb.ListDeploymentsReq) (*agentpb.ListDeploymentsResp, error) {
	rels, err := s.releases.List(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list releases: %v", err)
	}
	deps, err := s.deployments.List(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list deployments: %v", err)
	}
	commitByRelease := make(map[int64]string, len(rels))
	for _, r := range rels {
		commitByRelease[r.ID] = r.CommitSHA
	}
	out := make([]*agentpb.Deployment, 0, len(deps))
	for _, d := range deps {
		out = append(out, &agentpb.Deployment{
			Env:        string(d.Environment),
			Service:    s.service,
			Commit:     commitByRelease[d.ReleaseID],
			Status:     string(d.Status),
			DeployedAt: d.DeployedAt.Unix(),
		})
	}
	return &agentpb.ListDeploymentsResp{Deployments: out}, nil
}

// latestByRelease maps each release id to its most recent deployment time (unix
// seconds), comparing timestamps so it does not depend on row order.
func latestByRelease(deps []*deployment.Deployment) map[int64]int64 {
	latest := make(map[int64]int64, len(deps))
	for _, d := range deps {
		if ts := d.DeployedAt.Unix(); ts > latest[d.ReleaseID] {
			latest[d.ReleaseID] = ts
		}
	}
	return latest
}

// envForStatus denormalizes a release's lifecycle status to the env it sits in:
// production/archived are production-side, the rest staging-side.
func envForStatus(s release.Status) string {
	switch s {
	case release.StatusProduction, release.StatusArchived:
		return string(deployment.EnvProduction)
	default:
		return string(deployment.EnvStaging)
	}
}
