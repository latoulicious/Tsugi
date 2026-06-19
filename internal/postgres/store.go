package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/latoulicious/Tsugi/internal/deployment"
	"github.com/latoulicious/Tsugi/internal/release"
)

const queryTimeout = 5 * time.Second

// DBTX is the query surface shared by the pool and a transaction, so the repos
// run the same code inside or outside WithTx.
type DBTX interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Store aggregates the pgx-backed repositories over one pool.
type Store struct {
	pool        *pgxpool.Pool
	Releases    release.Repository
	Deployments deployment.Repository
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{
		pool:        pool,
		Releases:    &releaseRepo{db: pool},
		Deployments: &deploymentRepo{db: pool},
	}
}

// Connect opens a pgx pool; the caller owns Close.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	return pool, nil
}

// WithTx runs fn against tx-scoped repos, committing on success. Promotion and
// rollback advance release status and the deployment outcome together.
func (s *Store) WithTx(ctx context.Context, fn func(release.Repository, deployment.Repository) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := fn(&releaseRepo{db: tx}, &deploymentRepo{db: tx}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type releaseRepo struct{ db DBTX }

func (r *releaseRepo) Create(ctx context.Context, rel *release.Release) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	const q = `INSERT INTO releases (version, commit_sha, previous_commit_sha, changelog, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	row := r.db.QueryRow(ctx, q,
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
	rel, err := scanRelease(r.db.QueryRow(ctx, q, version))
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
	rows, err := r.db.Query(ctx, q)
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

func (r *releaseRepo) UpdateStatus(ctx context.Context, id int64, status release.Status) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	const q = `UPDATE releases SET status = $2 WHERE id = $1`
	tag, err := r.db.Exec(ctx, q, id, string(status))
	if err != nil {
		return fmt.Errorf("update release status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return release.ErrNotFound
	}
	return nil
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

type deploymentRepo struct{ db DBTX }

func (d *deploymentRepo) Create(ctx context.Context, dep *deployment.Deployment) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	const q = `INSERT INTO deployments (release_id, environment, status, deployed_at)
		VALUES ($1, $2, $3, $4) RETURNING id`
	row := d.db.QueryRow(ctx, q, dep.ReleaseID, string(dep.Environment), string(dep.Status), dep.DeployedAt)
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

func (d *deploymentRepo) UpdateStatus(ctx context.Context, id int64, status deployment.Status) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	const q = `UPDATE deployments SET status = $2 WHERE id = $1`
	tag, err := d.db.Exec(ctx, q, id, string(status))
	if err != nil {
		return fmt.Errorf("update deployment status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return deployment.ErrNotFound
	}
	return nil
}

func (d *deploymentRepo) queryDeployments(ctx context.Context, q string, args ...any) ([]*deployment.Deployment, error) {
	rows, err := d.db.Query(ctx, q, args...)
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
