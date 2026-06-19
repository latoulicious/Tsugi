package release

import (
	"errors"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	now := time.Date(2026, 6, 19, 20, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		version string
		commit  string
		prev    string
		wantErr error
	}{
		{"ok with prev", "v1.2.0", "abc123", "def456", nil},
		{"ok first release", "v1.0.0", "abc123", "", nil},
		{"trims whitespace", "  v1.2.0  ", " abc123 ", "", nil},
		{"empty version", "", "abc123", "", ErrEmptyVersion},
		{"no v prefix", "1.2.0", "abc123", "", ErrInvalidVersion},
		{"empty commit", "v1.2.0", "", "", ErrEmptyCommit},
		{"prev equals commit", "v1.2.0", "abc123", "abc123", ErrSameCommit},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := New(tc.version, tc.commit, tc.prev, now)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && r.Status() != StatusDraft {
				t.Fatalf("status = %q, want %q", r.Status(), StatusDraft)
			}
		})
	}
}

func TestTransitionTo(t *testing.T) {
	tests := []struct {
		name    string
		from    Status
		target  Status
		wantErr error
	}{
		{"draft to created", StatusDraft, StatusCreated, nil},
		{"created to staging", StatusCreated, StatusStaging, nil},
		{"staging to production", StatusStaging, StatusProduction, nil},
		{"production to archived", StatusProduction, StatusArchived, nil},
		{"rollback archived to production", StatusArchived, StatusProduction, nil},
		{"skip stage", StatusDraft, StatusStaging, ErrInvalidTransition},
		{"backwards", StatusStaging, StatusCreated, ErrInvalidTransition},
		{"archived skip", StatusArchived, StatusStaging, ErrInvalidTransition},
		{"unknown target", StatusDraft, Status("bogus"), ErrInvalidStatus},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &Release{status: tc.from}
			err := r.TransitionTo(tc.target)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			want := tc.from
			if tc.wantErr == nil {
				want = tc.target
			}
			if r.Status() != want {
				t.Fatalf("status = %q, want %q", r.Status(), want)
			}
		})
	}
}

func TestRehydrate(t *testing.T) {
	now := time.Date(2026, 6, 19, 20, 0, 0, 0, time.UTC)
	r, err := Rehydrate(7, "v1.2.0", "abc123", "def456", "## Features\n", StatusProduction, now)
	if err != nil {
		t.Fatalf("rehydrate: %v", err)
	}
	if r.ID != 7 || r.Changelog != "## Features\n" || r.Status() != StatusProduction {
		t.Fatalf("got id=%d changelog=%q status=%q", r.ID, r.Changelog, r.Status())
	}

	if _, err := Rehydrate(7, "v1.2.0", "abc123", "", "", Status("bogus"), now); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("bad status err = %v, want %v", err, ErrInvalidStatus)
	}
	if _, err := Rehydrate(7, "", "abc123", "", "", StatusDraft, now); !errors.Is(err, ErrEmptyVersion) {
		t.Fatalf("propagates New validation err = %v, want %v", err, ErrEmptyVersion)
	}
}

func TestLifecycle(t *testing.T) {
	r, err := New("v1.2.0", "abc123", "def456", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []Status{StatusCreated, StatusStaging, StatusProduction, StatusArchived} {
		if err := r.TransitionTo(s); err != nil {
			t.Fatalf("transition to %q: %v", s, err)
		}
	}
}
