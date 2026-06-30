// Package deployflow is the production-deploy use case shared by the release CLI
// and the write-plane agent, so the promotion state machine (record deployment →
// run deploy → archive previous + advance release atomically) lives in one place.
package deployflow

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/latoulicious/Tsugi/internal/deployment"
	"github.com/latoulicious/Tsugi/internal/release"
)

// LogSink receives one line of deploy output, tagged by its stream (stdout|stderr).
// The CLI writes lines to a terminal; the agent forwards them over gRPC.
type LogSink interface {
	Line(stream, text string)
}

// Deployer runs an actual deploy, streaming output to sink (real impl:
// internal/deploy shelling deploy.sh).
type Deployer interface {
	Run(ctx context.Context, target, env, ref string, sink LogSink) error
}

// TxRunner advances release status and deployment outcome atomically.
type TxRunner interface {
	WithTx(ctx context.Context, fn func(release.Repository, deployment.Repository) error) error
}

// Service deploys a release to production and reconciles release+deployment state.
// Deps are injected so the orchestration is testable without a live DB or docker.
type Service struct {
	Deployments deployment.Repository
	Tx          TxRunner
	Deployer    Deployer
	Target      string
	Now         func() time.Time
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// ToProduction records a pending production deployment, runs the real deploy at
// the release commit (streaming to sink), then advances release + outcome
// atomically — or marks the deployment failed and returns the deploy error.
func (s Service) ToProduction(ctx context.Context, r *release.Release, sink LogSink) error {
	dep, err := deployment.New(r.ID, deployment.EnvProduction, s.now())
	if err != nil {
		return err
	}
	if err := s.Deployments.Create(ctx, dep); err != nil {
		return err
	}
	if derr := s.Deployer.Run(ctx, s.Target, "prod", r.CommitSHA, sink); derr != nil {
		if err := dep.MarkFailed(); err != nil {
			return errors.Join(derr, err)
		}
		if err := s.Deployments.UpdateStatus(ctx, dep.ID, dep.Status); err != nil {
			return errors.Join(derr, err)
		}
		return derr
	}
	if err := s.Tx.WithTx(ctx, func(rr release.Repository, dr deployment.Repository) error {
		if err := archivePrevious(ctx, rr, r.ID); err != nil {
			return err
		}
		if err := r.TransitionTo(release.StatusProduction); err != nil {
			return err
		}
		if err := rr.UpdateStatus(ctx, r.ID, r.Status()); err != nil {
			return err
		}
		if err := dep.MarkSucceeded(); err != nil {
			return err
		}
		return dr.UpdateStatus(ctx, dep.ID, dep.Status)
	}); err != nil {
		return err
	}
	// Only report success once the transaction has committed — a rollback must not
	// leave a "deployed" line behind on the stream.
	sink.Line("stdout", fmt.Sprintf("deployed %s to production (%s)", r.Version, short(r.CommitSHA)))
	return nil
}

// archivePrevious demotes the current production release (only one at a time).
func archivePrevious(ctx context.Context, rr release.Repository, exceptID int64) error {
	rels, err := rr.List(ctx)
	if err != nil {
		return err
	}
	for _, r := range rels {
		if r.ID == exceptID || r.Status() != release.StatusProduction {
			continue
		}
		if err := r.TransitionTo(release.StatusArchived); err != nil {
			return err
		}
		if err := rr.UpdateStatus(ctx, r.ID, r.Status()); err != nil {
			return err
		}
	}
	return nil
}

// WriterSink adapts an io.Writer to LogSink, one line per Write (the CLI's sink).
type WriterSink struct{ W io.Writer }

func (s WriterSink) Line(_, text string) { fmt.Fprintln(s.W, text) }

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
