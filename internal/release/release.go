package release

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Status string

const (
	StatusDraft      Status = "draft"
	StatusCreated    Status = "created"
	StatusStaging    Status = "staging"
	StatusProduction Status = "production"
	StatusArchived   Status = "archived"
)

// linear lifecycle from PLAN.md; rollback/abandon edges land in P6
var transitions = map[Status][]Status{
	StatusDraft:      {StatusCreated},
	StatusCreated:    {StatusStaging},
	StatusStaging:    {StatusProduction},
	StatusProduction: {StatusArchived},
	StatusArchived:   {},
}

func (s Status) Valid() bool {
	_, ok := transitions[s]
	return ok
}

func (s Status) CanTransitionTo(target Status) bool {
	for _, next := range transitions[s] {
		if next == target {
			return true
		}
	}
	return false
}

var (
	ErrEmptyVersion      = errors.New("release: version is empty")
	ErrInvalidVersion    = errors.New("release: version must start with 'v'")
	ErrEmptyCommit       = errors.New("release: commit sha is empty")
	ErrSameCommit        = errors.New("release: previous commit equals current commit")
	ErrInvalidStatus     = errors.New("release: invalid status")
	ErrInvalidTransition = errors.New("release: invalid status transition")
	ErrNotFound          = errors.New("release: not found")
)

// Release is the P3 domain entity; persistence (P5) and CLI (P6) come later.
// status is guarded so the state machine is the only path that mutates it.
// ID is 0 until persisted; the store assigns it via RETURNING.
type Release struct {
	ID                int64
	Version           string
	CommitSHA         string
	PreviousCommitSHA string
	Changelog         string
	CreatedAt         time.Time
	status            Status
}

func New(version, commitSHA, previousCommitSHA string, createdAt time.Time) (*Release, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return nil, ErrEmptyVersion
	}
	// ponytail: prefix check only, not full semver; tighten when tags are parsed (P4/P6)
	if !strings.HasPrefix(version, "v") {
		return nil, ErrInvalidVersion
	}
	commitSHA = strings.TrimSpace(commitSHA)
	if commitSHA == "" {
		return nil, ErrEmptyCommit
	}
	previousCommitSHA = strings.TrimSpace(previousCommitSHA)
	if previousCommitSHA != "" && previousCommitSHA == commitSHA {
		return nil, ErrSameCommit
	}
	return &Release{
		Version:           version,
		CommitSHA:         commitSHA,
		PreviousCommitSHA: previousCommitSHA,
		CreatedAt:         createdAt,
		status:            StatusDraft,
	}, nil
}

// Rehydrate reconstructs a persisted release from a stored row, setting status
// directly (DB reads bypass the state machine; New stays the Draft-only path).
func Rehydrate(id int64, version, commitSHA, previousCommitSHA, changelog string, status Status, createdAt time.Time) (*Release, error) {
	r, err := New(version, commitSHA, previousCommitSHA, createdAt)
	if err != nil {
		return nil, err
	}
	if !status.Valid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidStatus, status)
	}
	r.ID = id
	r.Changelog = changelog
	r.status = status
	return r, nil
}

func (r *Release) Status() Status { return r.status }

// Repository persists and queries releases; pgx impl in internal/postgres.
type Repository interface {
	Create(ctx context.Context, r *Release) error
	GetByVersion(ctx context.Context, version string) (*Release, error)
	List(ctx context.Context) ([]*Release, error)
}

func (r *Release) TransitionTo(target Status) error {
	if !target.Valid() {
		return fmt.Errorf("%w: %q", ErrInvalidStatus, target)
	}
	if !r.status.CanTransitionTo(target) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, r.status, target)
	}
	r.status = target
	return nil
}
