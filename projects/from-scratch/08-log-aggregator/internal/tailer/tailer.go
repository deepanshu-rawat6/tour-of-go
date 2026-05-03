// Package tailer watches a file and emits new lines as they are appended.
package tailer

import (
	"bufio"
	"io"
	"os"
	"time"
)

// Tailer polls a file for new lines and sends them to Lines.
type Tailer struct {
	Path   string
	Lines  chan string
	done   chan struct{}
}

func New(path string) *Tailer {
	return &Tailer{Path: path, Lines: make(chan string, 256), done: make(chan struct{})}
}

// Start begins tailing. Call in a goroutine.
func (t *Tailer) Start() error {
	f, err := os.Open(t.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	// Seek to end so we only tail new lines
	f.Seek(0, io.SeekEnd)
	r := bufio.NewReader(f)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-t.done:
			return nil
		case <-ticker.C:
			for {
				line, err := r.ReadString('\n')
				if len(line) > 0 {
					t.Lines <- line
				}
				if err != nil {
					break
				}
			}
		}
	}
}

func (t *Tailer) Stop() { close(t.done) }
