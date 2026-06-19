package release

import (
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
)

// Release is the P3 domain entity; persistence (P5) and CLI (P6) come later.
// status is guarded so the state machine is the only path that mutates it.
type Release struct {
	Version           string
	CommitSHA         string
	PreviousCommitSHA string
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

func (r *Release) Status() Status { return r.status }

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
