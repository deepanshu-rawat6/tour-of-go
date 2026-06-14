package server_test

import (
	"net"
	"sync"
	"testing"
	"time"

	"tour_of_go/projects/from-scratch/01-tcp-server/internal/server"
)

func startServer(t *testing.T) string {
	t.Helper()
	s := server.New("127.0.0.1:0")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	s2 := server.New(addr)
	go s2.Start()
	t.Cleanup(func() { s2.Close() })
	time.Sleep(20 * time.Millisecond) // let server start
	_ = s
	return addr
}

func TestEcho(t *testing.T) {
	s := server.New("127.0.0.1:0")
	// Use a free port via net.Listen trick
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()

	srv := server.New(addr)
	go srv.Start()
	defer srv.Close()
	time.Sleep(20 * time.Millisecond)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	msg := []byte("hello from-scratch")
	conn.Write(msg)
	conn.(*net.TCPConn).CloseWrite()

	buf := make([]byte, len(msg))
	if _, err := conn.Read(buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != string(msg) {
		t.Fatalf("want %q, got %q", msg, buf)
	}
	_ = s
}

func TestConcurrentConnections(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()

	srv := server.New(addr)
	go srv.Start()
	defer srv.Close()
	time.Sleep(20 * time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				t.Errorf("dial: %v", err)
				return
			}
			defer conn.Close()
			msg := []byte("ping")
			conn.Write(msg)
			conn.(*net.TCPConn).CloseWrite()
			buf := make([]byte, 4)
			conn.Read(buf)
		}(i)
	}
	wg.Wait()
}
