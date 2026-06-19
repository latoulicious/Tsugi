package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/latoulicious/Tsugi/internal/changelog"
	"github.com/latoulicious/Tsugi/internal/deployment"
	"github.com/latoulicious/Tsugi/internal/release"
)

// GitReader reads commit history from a checkout (real impl: internal/git).
type GitReader interface {
	HeadSHA(ctx context.Context, dir string) (string, error)
	Subjects(ctx context.Context, dir, prev, head string) ([]string, error)
}

// Deployer runs an actual deploy (real impl: internal/deploy shelling deploy.sh).
type Deployer interface {
	Run(ctx context.Context, target, env, ref string) error
}

// TxRunner advances release status and deployment outcome atomically.
type TxRunner interface {
	WithTx(ctx context.Context, fn func(release.Repository, deployment.Repository) error) error
}

// App is the release CLI; deps are injected so the orchestration is testable
// without a live DB, git, or docker.
type App struct {
	Releases        release.Repository
	Deployments     deployment.Repository
	Tx              TxRunner
	Git             GitReader
	Deployer        Deployer
	Target          string
	StagingCheckout string
	Now             func() time.Time
	Out             io.Writer
}

const usage = "usage: tsugi release <create|list|show|promote|rollback> [version]"

func (a *App) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "list":
		return a.List(ctx)
	case "create":
		return a.withVersion(rest, func(v string) error { return a.Create(ctx, v) })
	case "show":
		return a.withVersion(rest, func(v string) error { return a.Show(ctx, v) })
	case "promote":
		return a.withVersion(rest, func(v string) error { return a.Promote(ctx, v) })
	case "rollback":
		return a.withVersion(rest, func(v string) error { return a.Rollback(ctx, v) })
	default:
		return fmt.Errorf("unknown release command %q\n%s", cmd, usage)
	}
}

func (a *App) withVersion(args []string, fn func(string) error) error {
	if len(args) != 1 {
		return errors.New(usage)
	}
	return fn(args[0])
}

func (a *App) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

// Create snapshots the validated staging commit, generates release notes, and
// records the release at Staging (it is already deployed on staging per PLAN).
func (a *App) Create(ctx context.Context, version string) error {
	head, err := a.Git.HeadSHA(ctx, a.StagingCheckout)
	if err != nil {
		return err
	}
	prev := ""
	if last, err := a.latest(ctx); err != nil {
		return err
	} else if last != nil {
		prev = last.CommitSHA
	}
	subjects, err := a.Git.Subjects(ctx, a.StagingCheckout, prev, head)
	if err != nil {
		return err
	}
	rel, err := release.New(version, head, prev, a.now())
	if err != nil {
		return err
	}
	for _, s := range []release.Status{release.StatusCreated, release.StatusStaging} {
		if err := rel.TransitionTo(s); err != nil {
			return err
		}
	}
	rel.Changelog = changelog.Generate(subjects)
	if err := a.Releases.Create(ctx, rel); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "created %s on staging (%s)\n", rel.Version, short(head))
	return nil
}

func (a *App) List(ctx context.Context) error {
	rels, err := a.Releases.List(ctx)
	if err != nil {
		return err
	}
	if len(rels) == 0 {
		fmt.Fprintln(a.Out, "no releases")
		return nil
	}
	tw := tabwriter.NewWriter(a.Out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "VERSION\tSTATUS\tCOMMIT\tCREATED")
	for _, r := range rels {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.Version, r.Status(), short(r.CommitSHA), r.CreatedAt.Format(time.RFC3339))
	}
	return tw.Flush()
}

func (a *App) Show(ctx context.Context, version string) error {
	r, err := a.Releases.GetByVersion(ctx, version)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "version: %s\nstatus:  %s\ncommit:  %s\nprev:    %s\ncreated: %s\n\n%s",
		r.Version, r.Status(), r.CommitSHA, dashIfEmpty(r.PreviousCommitSHA), r.CreatedAt.Format(time.RFC3339), r.Changelog)
	return nil
}

// Promote ships a validated staging release to production.
func (a *App) Promote(ctx context.Context, version string) error {
	r, err := a.Releases.GetByVersion(ctx, version)
	if err != nil {
		return err
	}
	if r.Status() != release.StatusStaging {
		return fmt.Errorf("release %s is %s, must be staging to promote", version, r.Status())
	}
	return a.deploy(ctx, r)
}

// Rollback re-deploys a previously archived production release.
func (a *App) Rollback(ctx context.Context, version string) error {
	r, err := a.Releases.GetByVersion(ctx, version)
	if err != nil {
		return err
	}
	if r.Status() != release.StatusArchived {
		return fmt.Errorf("release %s is %s, can only roll back to an archived release", version, r.Status())
	}
	return a.deploy(ctx, r)
}

// deploy records a pending production deployment, runs the real deploy at the
// release commit, then advances release + outcome atomically (or marks failed).
func (a *App) deploy(ctx context.Context, r *release.Release) error {
	dep, err := deployment.New(r.ID, deployment.EnvProduction, a.now())
	if err != nil {
		return err
	}
	if err := a.Deployments.Create(ctx, dep); err != nil {
		return err
	}
	if derr := a.Deployer.Run(ctx, a.Target, "prod", r.CommitSHA); derr != nil {
		if err := dep.MarkFailed(); err != nil {
			return errors.Join(derr, err)
		}
		if err := a.Deployments.UpdateStatus(ctx, dep.ID, dep.Status); err != nil {
			return errors.Join(derr, err)
		}
		return derr
	}
	return a.Tx.WithTx(ctx, func(rr release.Repository, dr deployment.Repository) error {
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
		if err := dr.UpdateStatus(ctx, dep.ID, dep.Status); err != nil {
			return err
		}
		fmt.Fprintf(a.Out, "deployed %s to production (%s)\n", r.Version, short(r.CommitSHA))
		return nil
	})
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

func (a *App) latest(ctx context.Context) (*release.Release, error) {
	rels, err := a.Releases.List(ctx)
	if err != nil || len(rels) == 0 {
		return nil, err
	}
	return rels[0], nil
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
