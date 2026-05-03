package tailer_test

import (
	"os"
	"testing"
	"time"

	"tour_of_go/projects/from-scratch/08-log-aggregator/internal/tailer"
)

func TestTailer_EmitsNewLines(t *testing.T) {
	f, err := os.CreateTemp("", "tailer-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	tl := tailer.New(f.Name())
	go tl.Start()
	defer tl.Stop()

	time.Sleep(50 * time.Millisecond) // let tailer seek to end
	f.WriteString("hello world\n")

	select {
	case line := <-tl.Lines:
		if line != "hello world\n" {
			t.Fatalf("want 'hello world\\n', got %q", line)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for line")
	}
}
