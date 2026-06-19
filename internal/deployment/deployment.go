package deployment

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Environment string

const (
	EnvStaging    Environment = "staging"
	EnvProduction Environment = "production"
)

func (e Environment) Valid() bool {
	return e == EnvStaging || e == EnvProduction
}

type Status string

const (
	StatusPending   Status = "pending"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
)

func (s Status) Valid() bool {
	return s == StatusPending || s == StatusSucceeded || s == StatusFailed
}

var (
	ErrInvalidID          = errors.New("deployment: id must be positive")
	ErrInvalidReleaseID   = errors.New("deployment: release id must be positive")
	ErrInvalidEnvironment = errors.New("deployment: invalid environment")
	ErrInvalidStatus      = errors.New("deployment: invalid status")
	ErrNotPending         = errors.New("deployment: outcome only settable while pending")
	ErrNotFound           = errors.New("deployment: not found")
)

// Deployment is the P5 history record: one release deployed to one environment.
// ID is 0 until persisted; the store assigns it via RETURNING.
type Deployment struct {
	ID          int64
	ReleaseID   int64
	Environment Environment
	Status      Status
	DeployedAt  time.Time
}

// New starts a deployment at StatusPending; the executor sets the outcome (P6).
func New(releaseID int64, env Environment, deployedAt time.Time) (*Deployment, error) {
	if releaseID <= 0 {
		return nil, ErrInvalidReleaseID
	}
	if !env.Valid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidEnvironment, env)
	}
	return &Deployment{
		ReleaseID:   releaseID,
		Environment: env,
		Status:      StatusPending,
		DeployedAt:  deployedAt,
	}, nil
}

// Rehydrate reconstructs a persisted deployment from a stored row (DB reads).
func Rehydrate(id, releaseID int64, env Environment, status Status, deployedAt time.Time) (*Deployment, error) {
	if id <= 0 {
		return nil, ErrInvalidID
	}
	if releaseID <= 0 {
		return nil, ErrInvalidReleaseID
	}
	if !env.Valid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidEnvironment, env)
	}
	if !status.Valid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidStatus, status)
	}
	return &Deployment{
		ID:          id,
		ReleaseID:   releaseID,
		Environment: env,
		Status:      status,
		DeployedAt:  deployedAt,
	}, nil
}

// MarkSucceeded/MarkFailed record the deploy outcome; only a pending deployment
// can transition (the executor calls these once the deploy returns).
func (d *Deployment) MarkSucceeded() error { return d.setOutcome(StatusSucceeded) }
func (d *Deployment) MarkFailed() error    { return d.setOutcome(StatusFailed) }

func (d *Deployment) setOutcome(s Status) error {
	if d.Status != StatusPending {
		return ErrNotPending
	}
	d.Status = s
	return nil
}

// Repository persists and queries deployment history; pgx impl in internal/postgres.
type Repository interface {
	Create(ctx context.Context, d *Deployment) error
	List(ctx context.Context) ([]*Deployment, error)
	ListByEnvironment(ctx context.Context, env Environment) ([]*Deployment, error)
	UpdateStatus(ctx context.Context, id int64, status Status) error
}
