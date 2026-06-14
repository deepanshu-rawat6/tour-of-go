// Package raw implements a minimal HTTP/1.1 server on top of net.Listener.
// It parses the request line and headers manually, then dispatches to handlers.
package raw

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

// HandlerFunc handles an HTTP request and writes a response via ResponseWriter.
type HandlerFunc func(w *ResponseWriter, r *Request)

// Request holds the parsed HTTP/1.1 request.
type Request struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    string
}

// ResponseWriter writes an HTTP/1.1 response to the underlying connection.
type ResponseWriter struct {
	conn    net.Conn
	headers map[string]string
	status  int
}

func newResponseWriter(conn net.Conn) *ResponseWriter {
	return &ResponseWriter{conn: conn, headers: map[string]string{"Content-Type": "text/plain"}, status: 200}
}

func (w *ResponseWriter) Header(key, val string) { w.headers[key] = val }
func (w *ResponseWriter) Status(code int)         { w.status = code }

func (w *ResponseWriter) Write(body string) {
	w.headers["Content-Length"] = fmt.Sprintf("%d", len(body))
	statusText := map[int]string{200: "OK", 201: "Created", 404: "Not Found", 405: "Method Not Allowed"}
	fmt.Fprintf(w.conn, "HTTP/1.1 %d %s\r\n", w.status, statusText[w.status])
	for k, v := range w.headers {
		fmt.Fprintf(w.conn, "%s: %s\r\n", k, v)
	}
	fmt.Fprintf(w.conn, "\r\n%s", body)
}

// Server is a minimal HTTP/1.1 server.
type Server struct {
	Addr   string
	routes map[string]HandlerFunc
	ln     net.Listener
}

func New(addr string) *Server {
	return &Server{Addr: addr, routes: make(map[string]HandlerFunc)}
}

// Handle registers a handler for the given path (any method).
func (s *Server) Handle(path string, h HandlerFunc) { s.routes[path] = h }

// Start begins accepting connections.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	s.ln = ln
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(conn)
	}
}

func (s *Server) Close() error {
	if s.ln != nil {
		return s.ln.Close()
	}
	return nil
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	w := newResponseWriter(conn)

	// Parse request line: "GET /path HTTP/1.1"
	line, err := r.ReadString('\n')
	if err != nil {
		return
	}
	parts := strings.Fields(strings.TrimSpace(line))
	if len(parts) < 2 {
		return
	}
	req := &Request{Method: parts[0], Path: parts[1], Headers: make(map[string]string)}

	// Parse headers until blank line
	for {
		hline, err := r.ReadString('\n')
		if err != nil || strings.TrimSpace(hline) == "" {
			break
		}
		if idx := strings.Index(hline, ":"); idx > 0 {
			req.Headers[strings.TrimSpace(hline[:idx])] = strings.TrimSpace(hline[idx+1:])
		}
	}

	// Dispatch
	h, ok := s.routes[req.Path]
	if !ok {
		w.Status(404)
		w.Write("404 not found")
		return
	}
	h(w, req)
}
