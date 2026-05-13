package state

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestStoreSaveLoadDelete(t *testing.T) {
	store := NewStore(t.TempDir())
	now := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	file := File{
		AgentBaseURL:          "http://10.0.1.4:8082",
		AccessToken:           "access",
		RefreshToken:          "refresh",
		AccessTokenExpiresAt:  now.Add(time.Hour),
		RefreshTokenExpiresAt: now.Add(30 * 24 * time.Hour),
		DeviceName:            "Timich MCP on test",
		PairedAt:              now,
		UpdatedAt:             now,
	}

	if err := store.Save(file); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	stat, err := os.Stat(store.Path())
	if err != nil {
		t.Fatalf("stat state file: %v", err)
	}
	if got := stat.Mode().Perm(); got != 0o600 {
		t.Fatalf("state file mode = %o, want 0600", got)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.AgentBaseURL != file.AgentBaseURL || loaded.RefreshToken != file.RefreshToken {
		t.Fatalf("Load() = %+v, want %+v", loaded, file)
	}

	if err := store.Delete(); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrNotPaired) {
		t.Fatalf("Load() after Delete() error = %v, want ErrNotPaired", err)
	}
}
