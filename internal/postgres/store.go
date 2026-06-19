package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/latoulicious/Tsugi/internal/deployment"
	"github.com/latoulicious/Tsugi/internal/release"
)

const queryTimeout = 5 * time.Second

// Store aggregates the pgx-backed repositories over one pool.
// ponytail: no WithTx/DBTX yet — no atomic multi-write use case until P6 promotion.
type Store struct {
	Releases    release.Repository
	Deployments deployment.Repository
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{
		Releases:    &releaseRepo{pool: pool},
		Deployments: &deploymentRepo{pool: pool},
	}
}

type releaseRepo struct{ pool *pgxpool.Pool }

func (r *releaseRepo) Create(ctx context.Context, rel *release.Release) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	const q = `INSERT INTO releases (version, commit_sha, previous_commit_sha, changelog, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	row := r.pool.QueryRow(ctx, q,
		rel.Version, rel.CommitSHA, rel.PreviousCommitSHA, rel.Changelog, string(rel.Status()), rel.CreatedAt)
	if err := row.Scan(&rel.ID); err != nil {
		return fmt.Errorf("insert release: %w", err)
	}
	return nil
}

func (r *releaseRepo) GetByVersion(ctx context.Context, version string) (*release.Release, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	const q = `SELECT id, version, commit_sha, previous_commit_sha, changelog, status, created_at
		FROM releases WHERE version = $1`
	rel, err := scanRelease(r.pool.QueryRow(ctx, q, version))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, release.ErrNotFound
	}
	return rel, err
}

func (r *releaseRepo) List(ctx context.Context) ([]*release.Release, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	const q = `SELECT id, version, commit_sha, previous_commit_sha, changelog, status, created_at
		FROM releases ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query releases: %w", err)
	}
	defer rows.Close()
	var out []*release.Release
	for rows.Next() {
		rel, err := scanRelease(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rel)
	}
	return out, rows.Err()
}

// scanRow is the read surface shared by QueryRow and Rows (both expose Scan).
type scanRow interface {
	Scan(dest ...any) error
}

func scanRelease(row scanRow) (*release.Release, error) {
	var (
		id        int64
		version   string
		commit    string
		prev      string
		changelog string
		status    string
		createdAt time.Time
	)
	if err := row.Scan(&id, &version, &commit, &prev, &changelog, &status, &createdAt); err != nil {
		return nil, fmt.Errorf("scan release: %w", err)
	}
	return release.Rehydrate(id, version, commit, prev, changelog, release.Status(status), createdAt)
}

type deploymentRepo struct{ pool *pgxpool.Pool }

func (d *deploymentRepo) Create(ctx context.Context, dep *deployment.Deployment) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	const q = `INSERT INTO deployments (release_id, environment, status, deployed_at)
		VALUES ($1, $2, $3, $4) RETURNING id`
	row := d.pool.QueryRow(ctx, q, dep.ReleaseID, string(dep.Environment), string(dep.Status), dep.DeployedAt)
	if err := row.Scan(&dep.ID); err != nil {
		return fmt.Errorf("insert deployment: %w", err)
	}
	return nil
}

func (d *deploymentRepo) List(ctx context.Context) ([]*deployment.Deployment, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	const q = `SELECT id, release_id, environment, status, deployed_at
		FROM deployments ORDER BY deployed_at DESC`
	return d.queryDeployments(ctx, q)
}

func (d *deploymentRepo) ListByEnvironment(ctx context.Context, env deployment.Environment) ([]*deployment.Deployment, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	const q = `SELECT id, release_id, environment, status, deployed_at
		FROM deployments WHERE environment = $1 ORDER BY deployed_at DESC`
	return d.queryDeployments(ctx, q, string(env))
}

func (d *deploymentRepo) queryDeployments(ctx context.Context, q string, args ...any) ([]*deployment.Deployment, error) {
	rows, err := d.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query deployments: %w", err)
	}
	defer rows.Close()
	var out []*deployment.Deployment
	for rows.Next() {
		var (
			id         int64
			releaseID  int64
			env        string
			status     string
			deployedAt time.Time
		)
		if err := rows.Scan(&id, &releaseID, &env, &status, &deployedAt); err != nil {
			return nil, fmt.Errorf("scan deployment: %w", err)
		}
		dep, err := deployment.Rehydrate(id, releaseID, deployment.Environment(env), deployment.Status(status), deployedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, dep)
	}
	return out, rows.Err()
}
