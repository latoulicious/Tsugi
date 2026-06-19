package deployment

import (
	"errors"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	now := time.Date(2026, 6, 19, 20, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		releaseID int64
		env       Environment
		wantErr   error
	}{
		{"ok staging", 1, EnvStaging, nil},
		{"ok production", 9, EnvProduction, nil},
		{"zero release id", 0, EnvStaging, ErrInvalidReleaseID},
		{"negative release id", -1, EnvStaging, ErrInvalidReleaseID},
		{"bad environment", 1, Environment("local"), ErrInvalidEnvironment},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, err := New(tc.releaseID, tc.env, now)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && d.Status != StatusPending {
				t.Fatalf("status = %q, want %q", d.Status, StatusPending)
			}
		})
	}
}

func TestRehydrate(t *testing.T) {
	now := time.Date(2026, 6, 19, 20, 0, 0, 0, time.UTC)
	d, err := Rehydrate(3, 1, EnvProduction, StatusSucceeded, now)
	if err != nil {
		t.Fatalf("rehydrate: %v", err)
	}
	if d.ID != 3 || d.ReleaseID != 1 || d.Status != StatusSucceeded {
		t.Fatalf("got id=%d release=%d status=%q", d.ID, d.ReleaseID, d.Status)
	}

	if _, err := Rehydrate(0, 1, EnvStaging, StatusPending, now); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("zero id err = %v, want %v", err, ErrInvalidID)
	}
	if _, err := Rehydrate(3, 0, EnvStaging, StatusPending, now); !errors.Is(err, ErrInvalidReleaseID) {
		t.Fatalf("zero release id err = %v, want %v", err, ErrInvalidReleaseID)
	}
	if _, err := Rehydrate(3, 1, EnvStaging, Status("bogus"), now); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("bad status err = %v, want %v", err, ErrInvalidStatus)
	}
}
