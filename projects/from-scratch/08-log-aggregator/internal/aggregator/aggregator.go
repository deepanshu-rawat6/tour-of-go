// Package aggregator receives log lines and stores them for querying.
package aggregator

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// LogEntry is a single log line with metadata.
type LogEntry struct {
	Timestamp time.Time
	Source    string
	Line      string
}

// Aggregator stores log entries and serves queries.
type Aggregator struct {
	mu      sync.RWMutex
	entries []LogEntry
	ln      net.Listener
}

func New() *Aggregator { return &Aggregator{} }

// Ingest adds a log entry.
func (a *Aggregator) Ingest(source, line string) {
	a.mu.Lock()
	a.entries = append(a.entries, LogEntry{
		Timestamp: time.Now(),
		Source:    source,
		Line:      strings.TrimSpace(line),
	})
	a.mu.Unlock()
}

// Search returns entries matching query and source filter.
func (a *Aggregator) Search(query, source string, limit int) []LogEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	var results []LogEntry
	for i := len(a.entries) - 1; i >= 0 && len(results) < limit; i-- {
		e := a.entries[i]
		if source != "" && e.Source != source {
			continue
		}
		if query != "" && !strings.Contains(e.Line, query) {
			continue
		}
		results = append(results, e)
	}
	return results
}

// StartTCP listens for shipper connections on addr.
// Protocol: each line is "SOURCE\tLOG_LINE\n"
func (a *Aggregator) StartTCP(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	a.ln = ln
	log.Printf("aggregator TCP on %s", addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go a.handleConn(conn)
	}
}

func (a *Aggregator) handleConn(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			a.Ingest(parts[0], parts[1])
		}
	}
}

func (a *Aggregator) Close() error {
	if a.ln != nil {
		return a.ln.Close()
	}
	return nil
}

// FormatEntry formats a log entry for display.
func FormatEntry(e LogEntry) string {
	return fmt.Sprintf("[%s] %s: %s", e.Timestamp.Format("15:04:05"), e.Source, e.Line)
}
