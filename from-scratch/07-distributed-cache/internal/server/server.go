// Package server implements the Redis-compatible TCP server.
package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"tour_of_go/projects/from-scratch/07-distributed-cache/internal/resp"
	"tour_of_go/projects/from-scratch/07-distributed-cache/internal/store"
)

type Server struct {
	Addr  string
	store *store.Store
	ln    net.Listener
}

func New(addr string, s *store.Store) *Server { return &Server{Addr: addr, store: s} }

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	s.ln = ln
	log.Printf("resp-server listening on %s (redis-cli -p %s)", s.Addr, strings.TrimPrefix(s.Addr, ":"))
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handle(conn)
	}
}

func (s *Server) Close() error {
	if s.ln != nil {
		return s.ln.Close()
	}
	return nil
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		v, err := resp.Parse(r)
		if err != nil {
			return
		}
		cmd, args, err := resp.Command(v)
		if err != nil {
			conn.Write([]byte(resp.Error("invalid command")))
			continue
		}
		conn.Write([]byte(s.exec(cmd, args)))
	}
}

func (s *Server) exec(cmd string, args []string) string {
	switch cmd {
	case "PING":
		if len(args) > 0 {
			return resp.BulkString(args[0])
		}
		return resp.SimpleString("PONG")
	case "SET":
		if len(args) < 2 {
			return resp.Error("SET requires key value")
		}
		var ttl time.Duration
		for i := 2; i < len(args)-1; i++ {
			if strings.ToUpper(args[i]) == "EX" {
				n, err := strconv.Atoi(args[i+1])
				if err == nil {
					ttl = time.Duration(n) * time.Second
				}
			}
		}
		s.store.Set(args[0], args[1], ttl)
		return resp.SimpleString("OK")
	case "GET":
		if len(args) < 1 {
			return resp.Error("GET requires key")
		}
		v, ok := s.store.Get(args[0])
		if !ok {
			return resp.NullBulk()
		}
		return resp.BulkString(v)
	case "DEL":
		n := s.store.Del(args...)
		return resp.Integer(int64(n))
	case "EXISTS":
		if len(args) < 1 {
			return resp.Error("EXISTS requires key")
		}
		if s.store.Exists(args[0]) {
			return resp.Integer(1)
		}
		return resp.Integer(0)
	case "TTL":
		if len(args) < 1 {
			return resp.Error("TTL requires key")
		}
		return resp.Integer(s.store.TTL(args[0]))
	case "KEYS":
		return resp.Array(s.store.Keys())
	case "INFO":
		return resp.BulkString(fmt.Sprintf("# Server\nredis_version:0.1.0\nkeys:%d\n", len(s.store.Keys())))
	default:
		return resp.Error(fmt.Sprintf("unknown command '%s'", cmd))
	}
}
