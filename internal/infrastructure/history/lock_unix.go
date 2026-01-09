//go:build unix

package history

import (
	"os"
	"path/filepath"
	"syscall"
)

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
