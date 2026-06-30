// Package agent is the Tsugi write-plane gRPC server (PLAN §11): the localhost-only
// act-plane Yagura dials. Contract lives in proto/tsugi_agent.proto.
package agent

import (
	"context"
	"regexp"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/latoulicious/Tsugi/internal/agentpb"
	"github.com/latoulicious/Tsugi/internal/deployflow"
	"github.com/latoulicious/Tsugi/internal/deployment"
	"github.com/latoulicious/Tsugi/internal/release"
)

// A deploy shells docker/git on the host; cap a wedged one so it can't hold the
// single-flight lock forever.
const deployTimeout = 15 * time.Minute

// commitRe mirrors deploy.sh's --ref guard (a git SHA, no branch names/options).
var commitRe = regexp.MustCompile(`^[0-9a-fA-F]{7,40}$`)

// releaseLister and deploymentLister are the read slices the agent needs; narrow
// consumer interfaces keep the write repos out of the read plane (ISP).
type releaseLister interface {
	List(ctx context.Context) ([]*release.Release, error)
}

type deploymentLister interface {
	List(ctx context.Context) ([]*deployment.Deployment, error)
}

// Server implements agentpb.TsugiAgentServer: read RPCs plus the write plane
// (Promote/Rollback). Deploy (staging) stays Unimplemented via the embedded stub.
type Server struct {
	agentpb.UnimplementedTsugiAgentServer
	releases    releaseLister
	deployments deploymentLister
	flow        *deployflow.Service
	service     string // target app name, surfaced as Release.service / Deployment.service
	// deploy serializes write RPCs: two deploys racing the same checkout + compose
	// project would corrupt each other.
	deploy sync.Mutex
}

func New(releases releaseLister, deployments deploymentLister, flow *deployflow.Service, service string) *Server {
	return &Server{releases: releases, deployments: deployments, flow: flow, service: service}
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

// Promote ships a staging release to production, streaming the deploy log.
func (s *Server) Promote(req *agentpb.DeployReq, stream agentpb.TsugiAgent_PromoteServer) error {
	return s.runDeploy(req, stream, release.StatusStaging)
}

// Rollback re-deploys an archived release to production, streaming the deploy log.
func (s *Server) Rollback(req *agentpb.DeployReq, stream agentpb.TsugiAgent_RollbackServer) error {
	return s.runDeploy(req, stream, release.StatusArchived)
}

// runDeploy validates the request, resolves the release by commit, guards its
// status, then streams the shared production-deploy use case to the client.
func (s *Server) runDeploy(req *agentpb.DeployReq, stream agentpb.TsugiAgent_PromoteServer, want release.Status) error {
	if err := s.validate(req); err != nil {
		return err
	}
	if !s.deploy.TryLock() {
		return status.Error(codes.FailedPrecondition, "a deploy is already in progress")
	}
	defer s.deploy.Unlock()

	rel, err := s.findByCommit(stream.Context(), req.GetCommit())
	if err != nil {
		return err
	}
	if rel.Status() != want {
		return status.Errorf(codes.FailedPrecondition, "release %s is %s, must be %s", rel.Version, rel.Status(), want)
	}

	// Detach the deploy from the stream: a dropped viewer must not abort a deploy
	// mid-build. The DB outcome still lands, so the release row updates regardless.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(stream.Context()), deployTimeout)
	defer cancel()

	sink := streamSink{stream: stream}
	if derr := s.flow.ToProduction(ctx, rel, sink); derr != nil {
		sink.Line("stderr", "deploy failed: "+derr.Error())
		return status.Errorf(codes.Internal, "deploy %s: %v", rel.Version, derr)
	}
	return nil
}

// validate rejects anything the trust boundary shouldn't pass on to the host:
// wrong service, non-production env, or a commit that isn't a bare git SHA.
func (s *Server) validate(req *agentpb.DeployReq) error {
	if req.GetService() != s.service {
		return status.Errorf(codes.InvalidArgument, "unknown service %q", req.GetService())
	}
	if req.GetEnv() != string(deployment.EnvProduction) {
		return status.Errorf(codes.InvalidArgument, "env must be %q", deployment.EnvProduction)
	}
	if !commitRe.MatchString(req.GetCommit()) {
		return status.Error(codes.InvalidArgument, "commit must be a 7-40 char hex sha")
	}
	return nil
}

func (s *Server) findByCommit(ctx context.Context, commit string) (*release.Release, error) {
	rels, err := s.releases.List(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list releases: %v", err)
	}
	for _, r := range rels {
		if r.CommitSHA == commit {
			return r, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "no release for commit %s", commit)
}

// streamSink forwards deploy output to the gRPC client. Sends are best-effort: a
// dropped client just drops the line, the deploy continues (runDeploy detaches it).
type streamSink struct {
	stream agentpb.TsugiAgent_PromoteServer
}

func (s streamSink) Line(stream, text string) {
	_ = s.stream.Send(&agentpb.LogLine{Ts: time.Now().UnixMilli(), Stream: stream, Text: text})
}
