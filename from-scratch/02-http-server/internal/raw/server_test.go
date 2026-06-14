package raw_test

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"tour_of_go/projects/from-scratch/02-http-server/internal/raw"
)

func startRaw(t *testing.T) string {
	t.Helper()
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()

	s := raw.New(addr)
	s.Handle("/", func(w *raw.ResponseWriter, r *raw.Request) { w.Write("hello") })
	s.Handle("/health", func(w *raw.ResponseWriter, r *raw.Request) { w.Write("ok") })
	go s.Start()
	t.Cleanup(func() { s.Close() })
	time.Sleep(20 * time.Millisecond)
	return addr
}

func TestRaw_GET_Root(t *testing.T) {
	addr := startRaw(t)
	resp, err := http.Get("http://" + addr + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello" {
		t.Fatalf("want hello, got %s", body)
	}
}

func TestRaw_GET_Health(t *testing.T) {
	addr := startRaw(t)
	resp, _ := http.Get("http://" + addr + "/health")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}

func TestRaw_404(t *testing.T) {
	addr := startRaw(t)
	resp, _ := http.Get("http://" + addr + "/missing")
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("want 404, got %d", resp.StatusCode)
	}
}

func TestRaw_RawRequest(t *testing.T) {
	addr := startRaw(t)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprintf(conn, "GET /health HTTP/1.1\r\nHost: localhost\r\n\r\n")
	buf := make([]byte, 512)
	n, _ := conn.Read(buf)
	resp := string(buf[:n])
	if !strings.Contains(resp, "200 OK") {
		t.Fatalf("expected 200 OK in response, got: %s", resp)
	}
}
