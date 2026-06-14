// Package server implements a raw TCP echo server.
// Each connection is handled in its own goroutine (goroutine-per-connection model).
package server

import (
	"io"
	"log"
	"net"
)

// Server listens on Addr and echoes every byte back to the sender.
type Server struct {
	Addr string
	ln   net.Listener
}

func New(addr string) *Server { return &Server{Addr: addr} }

// Start begins accepting connections. Blocks until Close is called.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	s.ln = ln
	log.Printf("tcp-server listening on %s", s.Addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err // closed by Close()
		}
		go handle(conn)
	}
}

// Close stops the server.
func (s *Server) Close() error {
	if s.ln != nil {
		return s.ln.Close()
	}
	return nil
}

// handle echoes all bytes from conn back to the sender, then closes.
func handle(conn net.Conn) {
	defer conn.Close()
	if _, err := io.Copy(conn, conn); err != nil {
		log.Printf("conn %s: %v", conn.RemoteAddr(), err)
	}
}
