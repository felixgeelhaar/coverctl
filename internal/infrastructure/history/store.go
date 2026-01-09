package history

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"

	"github.com/felixgeelhaar/coverctl/internal/domain"
)

// DefaultMaxEntries is the default number of history entries to keep.
const DefaultMaxEntries = 100

// FileStore provides JSON file-based storage for coverage history.
type FileStore struct {
	Path       string
	MaxEntries int
}

// fileLock represents a file-based lock for concurrent access protection.
type fileLock struct {
	file *os.File
}

// acquireLock creates an exclusive lock on the history file.
// This prevents race conditions when multiple processes access the file.
func (s *FileStore) acquireLock() (*fileLock, error) {
	lockPath := s.Path + ".lock"
	dir := filepath.Dir(lockPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}

	// #nosec G304 -- Path is derived from trusted config
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}

	// Acquire exclusive lock (blocking)
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		_ = file.Close() // Best-effort close on lock failure
		return nil, err
	}

	return &fileLock{file: file}, nil
}

// release releases the file lock.
func (l *fileLock) release() error {
	if l.file == nil {
		return nil
	}
	// Release lock - best-effort, always close file afterwards
	unlockErr := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	closeErr := l.file.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}

// Load reads the history from the JSON file.
// Returns an empty history if the file doesn't exist.
func (s *FileStore) Load() (domain.History, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.History{}, nil
		}
		return domain.History{}, err
	}

	var h domain.History
	if err := json.Unmarshal(data, &h); err != nil {
		return domain.History{}, err
	}

	return h, nil
}

// Save writes the history to the JSON file.
func (s *FileStore) Save(h domain.History) error {
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}

	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.Path, data, 0o600)
}

// Append adds a new entry to the history and saves it.
// If MaxEntries is set, older entries are removed to maintain the limit.
// Uses file locking to prevent race conditions with concurrent processes.
func (s *FileStore) Append(entry domain.HistoryEntry) error {
	// Acquire exclusive lock to prevent race conditions
	lock, err := s.acquireLock()
	if err != nil {
		return err
	}
	defer lock.release()

	h, err := s.Load()
	if err != nil {
		return err
	}

	h.Entries = append(h.Entries, entry)

	// Trim to max entries if configured
	max := s.MaxEntries
	if max == 0 {
		max = DefaultMaxEntries
	}
	if len(h.Entries) > max {
		h.Entries = h.Entries[len(h.Entries)-max:]
	}

	return s.Save(h)
}
