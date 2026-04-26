package wal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Entry is a single log entry persisted to the WAL.
type Entry struct {
	Term    uint64 `json:"term"`
	Index   uint64 `json:"index"`
	Command []byte `json:"command"`
}

// WAL is an append-only write-ahead log backed by a single file.
type WAL struct {
	f    *os.File
	path string
}

// Open opens or creates the WAL file at dir/wal.log.
func Open(dir string) (*WAL, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "wal.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening WAL: %w", err)
	}
	return &WAL{f: f, path: path}, nil
}

// Append writes an entry to the WAL and fsyncs.
func (w *WAL) Append(e Entry) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if _, err := w.f.Write(data); err != nil {
		return err
	}
	return w.f.Sync()
}

// ReadAll reads all entries from the WAL (used for recovery on startup).
func (w *WAL) ReadAll() ([]Entry, error) {
	if _, err := w.f.Seek(0, 0); err != nil {
		return nil, err
	}
	var entries []Entry
	scanner := bufio.NewScanner(w.f)
	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			return nil, fmt.Errorf("corrupt WAL entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, scanner.Err()
}

// Truncate rewrites the WAL keeping only entries with Index < fromIndex.
// Used when a follower must discard conflicting log entries.
func (w *WAL) Truncate(fromIndex uint64) error {
	entries, err := w.ReadAll()
	if err != nil {
		return err
	}
	// Rewrite file from scratch
	if err := w.f.Truncate(0); err != nil {
		return err
	}
	if _, err := w.f.Seek(0, 0); err != nil {
		return err
	}
	for _, e := range entries {
		if e.Index < fromIndex {
			if err := w.Append(e); err != nil {
				return err
			}
		}
	}
	return nil
}

// Close closes the underlying file.
func (w *WAL) Close() error { return w.f.Close() }
