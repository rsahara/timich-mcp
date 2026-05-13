package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var ErrNotPaired = errors.New("timich-mcp is not paired")

const (
	defaultStateDirName = ".local/state/timich-mcp"
	stateFileName       = "state.json"
)

type File struct {
	AgentBaseURL          string    `json:"agentBaseURL"`
	AccessToken           string    `json:"accessToken"`
	RefreshToken          string    `json:"refreshToken"`
	AccessTokenExpiresAt  time.Time `json:"accessTokenExpiresAt"`
	RefreshTokenExpiresAt time.Time `json:"refreshTokenExpiresAt"`
	DeviceName            string    `json:"deviceName,omitempty"`
	AgentID               string    `json:"agentId,omitempty"`
	AgentName             string    `json:"agentName,omitempty"`
	DeviceID              string    `json:"deviceId,omitempty"`
	AccessMode            string    `json:"accessMode,omitempty"`
	PairedAt              time.Time `json:"pairedAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type Store struct {
	Dir string
}

func DefaultDir() string {
	if value := os.Getenv("TIMICH_MCP_STATE_DIR"); value != "" {
		return value
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".", defaultStateDirName)
	}
	return filepath.Join(home, defaultStateDirName)
}

func NewStore(dir string) *Store {
	if dir == "" {
		dir = DefaultDir()
	}
	return &Store{Dir: dir}
}

func (s *Store) Path() string {
	return filepath.Join(s.Dir, stateFileName)
}

func (s *Store) Load() (File, error) {
	raw, err := os.ReadFile(s.Path())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return File{}, ErrNotPaired
		}
		return File{}, fmt.Errorf("read state: %w", err)
	}
	var file File
	if err := json.Unmarshal(raw, &file); err != nil {
		return File{}, fmt.Errorf("decode state: %w", err)
	}
	if file.AgentBaseURL == "" || file.RefreshToken == "" {
		return File{}, ErrNotPaired
	}
	return file, nil
}

func (s *Store) Save(file File) error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	if err := os.Chmod(s.Dir, 0o700); err != nil {
		return fmt.Errorf("protect state directory: %w", err)
	}

	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	raw = append(raw, '\n')

	tmpPath := filepath.Join(s.Dir, fmt.Sprintf(".%s.%d.tmp", stateFileName, time.Now().UnixNano()))
	if err := os.WriteFile(tmpPath, raw, 0o600); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("protect state file: %w", err)
	}
	if err := os.Rename(tmpPath, s.Path()); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace state: %w", err)
	}
	return nil
}

func (s *Store) Delete() error {
	if err := os.Remove(s.Path()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove state: %w", err)
	}
	return nil
}
