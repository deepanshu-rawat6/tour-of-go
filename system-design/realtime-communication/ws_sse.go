// ws_sse.go — Real-Time Communication Demos
//
// Run with: go run ws_sse.go
//
// Three routes:
//   /ws      — WebSocket echo server (manual RFC 6455 handshake)
//   /events  — Server-Sent Events (SSE) stream
//   /poll    — Long polling endpoint
//
// Test with:
//   WebSocket: wscat -c ws://localhost:8080/ws
//              OR: curl -v --include --no-buffer \
//                    -H "Upgrade: websocket" \
//                    -H "Connection: Upgrade" \
//                    -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
//                    -H "Sec-WebSocket-Version: 13" \
//                    http://localhost:8080/ws
//
//   SSE:       curl -N http://localhost:8080/events
//
//   Long poll: curl http://localhost:8080/poll
//              (response arrives ~3s later with a message)

package main

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Part A: WebSocket Echo Server (manual RFC 6455 implementation, stdlib only)
// ─────────────────────────────────────────────────────────────────────────────
//
// WebSocket upgrade flow:
//  1. Client sends HTTP GET with "Upgrade: websocket" + Sec-WebSocket-Key header
//  2. Server responds with 101 Switching Protocols + Sec-WebSocket-Accept (HMAC)
//  3. Both sides switch to framed binary protocol on the same TCP connection
//
// Why gorilla/websocket in production?
//   • Handles all frame types (ping/pong/close), masking, fragmentation
//   • gorilla/websocket is the de-facto standard: github.com/gorilla/websocket
//   • Here we implement just enough of RFC 6455 to illustrate the protocol.

// wsGUID is the magic GUID defined in RFC 6455 §1.3 for the Accept key derivation.
const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// wsAcceptKey computes the Sec-WebSocket-Accept value from the client key.
//   accept = base64(SHA1(clientKey + wsGUID))
func wsAcceptKey(clientKey string) string {
	h := sha1.New()
	h.Write([]byte(strings.TrimSpace(clientKey) + wsGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// wsReadFrame reads a single WebSocket frame from the connection.
// RFC 6455 §5.2 — minimal implementation (text frames, no fragmentation).
//
// Frame layout (simplified):
//
//	Byte 0: FIN(1) + RSV(3) + opcode(4)
//	Byte 1: MASK(1) + payload len(7)   [extended len handled for ≤65535 bytes]
//	[Masking key: 4 bytes if MASK=1]
//	[Payload]
func wsReadFrame(conn net.Conn) (opcode byte, payload []byte, err error) {
	header := make([]byte, 2)
	if _, err = io.ReadFull(conn, header); err != nil {
		return 0, nil, err
	}

	// fin := (header[0] >> 7) & 1  // we ignore FIN for this demo
	opcode = header[0] & 0x0F
	masked := (header[1] >> 7) & 1
	payloadLen := int(header[1] & 0x7F)

	// Extended payload length
	switch payloadLen {
	case 126:
		ext := make([]byte, 2)
		if _, err = io.ReadFull(conn, ext); err != nil {
			return 0, nil, err
		}
		payloadLen = int(binary.BigEndian.Uint16(ext))
	case 127:
		ext := make([]byte, 8)
		if _, err = io.ReadFull(conn, ext); err != nil {
			return 0, nil, err
		}
		payloadLen = int(binary.BigEndian.Uint64(ext))
	}

	// Masking key (clients MUST mask frames per RFC 6455 §5.3)
	var maskKey [4]byte
	if masked == 1 {
		if _, err = io.ReadFull(conn, maskKey[:]); err != nil {
			return 0, nil, err
		}
	}

	// Read payload
	payload = make([]byte, payloadLen)
	if _, err = io.ReadFull(conn, payload); err != nil {
		return 0, nil, err
	}

	// Unmask payload
	if masked == 1 {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}

	return opcode, payload, nil
}

// wsWriteFrame writes a single unmasked WebSocket text frame (server → client).
// Servers MUST NOT mask frames per RFC 6455 §5.1.
func wsWriteFrame(conn net.Conn, opcode byte, payload []byte) error {
	frame := make([]byte, 0, 10+len(payload))

	// Byte 0: FIN=1, RSV=000, opcode
	frame = append(frame, 0x80|opcode)

	// Byte 1+: payload length (no masking from server)
	l := len(payload)
	switch {
	case l <= 125:
		frame = append(frame, byte(l))
	case l <= 65535:
		frame = append(frame, 126)
		ext := make([]byte, 2)
		binary.BigEndian.PutUint16(ext, uint16(l))
		frame = append(frame, ext...)
	default:
		frame = append(frame, 127)
		ext := make([]byte, 8)
		binary.BigEndian.PutUint64(ext, uint64(l))
		frame = append(frame, ext...)
	}

	frame = append(frame, payload...)
	_, err := conn.Write(frame)
	return err
}

// wsHandler handles HTTP → WebSocket upgrade and runs an echo loop.
func wsHandler(w http.ResponseWriter, r *http.Request) {
	// Validate upgrade request
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") ||
		!strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") {
		http.Error(w, "Expected WebSocket upgrade", http.StatusBadRequest)
		return
	}

	clientKey := r.Header.Get("Sec-WebSocket-Key")
	if clientKey == "" {
		http.Error(w, "Missing Sec-WebSocket-Key", http.StatusBadRequest)
		return
	}

	// Hijack the connection — take it out of net/http's hands
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "WebSocket not supported (no Hijacker)", http.StatusInternalServerError)
		return
	}
	conn, buf, err := hj.Hijack()
	if err != nil {
		log.Printf("ws hijack error: %v", err)
		return
	}
	defer conn.Close()

	// Send 101 Switching Protocols response directly on the raw connection.
	// This is the WebSocket handshake response per RFC 6455 §4.2.2.
	acceptKey := wsAcceptKey(clientKey)
	handshake := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n" +
		"\r\n"
	if _, err = buf.WriteString(handshake); err != nil {
		log.Printf("ws handshake write error: %v", err)
		return
	}
	if err = buf.Flush(); err != nil {
		log.Printf("ws handshake flush error: %v", err)
		return
	}

	log.Println("[WS] Client connected, entering echo loop")

	// Welcome frame
	_ = wsWriteFrame(conn, 0x01, []byte("Welcome! Send me text and I'll echo it back."))

	// Echo loop: read frames and echo them back
	for {
		conn.SetDeadline(time.Now().Add(60 * time.Second))
		opcode, payload, err := wsReadFrame(conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("[WS] read frame error: %v", err)
			}
			break
		}

		switch opcode {
		case 0x01: // text frame
			log.Printf("[WS] received text: %q", payload)
			echo := fmt.Sprintf("echo: %s (at %s)", payload, time.Now().Format("15:04:05"))
			if err := wsWriteFrame(conn, 0x01, []byte(echo)); err != nil {
				log.Printf("[WS] write error: %v", err)
				return
			}
		case 0x08: // close frame — send close back then exit
			log.Println("[WS] client sent close frame")
			_ = wsWriteFrame(conn, 0x08, []byte{})
			return
		case 0x09: // ping — send pong
			_ = wsWriteFrame(conn, 0x0A, payload)
		default:
			log.Printf("[WS] unhandled opcode: 0x%02X", opcode)
		}
	}
	log.Println("[WS] Client disconnected")
}

// ─────────────────────────────────────────────────────────────────────────────
// Part B: Server-Sent Events (SSE)
// ─────────────────────────────────────────────────────────────────────────────
//
// SSE is a one-way, server→client stream over a regular HTTP connection.
// The client opens a persistent GET request; the server never closes it and
// keeps sending "data: ...\n\n" chunks. The browser's EventSource API handles
// reconnection automatically.
//
// Key headers:
//   Content-Type: text/event-stream    — tells browser this is SSE
//   Cache-Control: no-cache            — prevent proxy buffering
//   X-Accel-Buffering: no              — disable nginx buffering
//
// SSE event format (each message):
//   id: <optional-event-id>\n
//   event: <optional-event-type>\n
//   data: <your payload>\n
//   \n                               ← blank line terminates the event
//
// JavaScript client:
//   const es = new EventSource('/events');
//   es.onmessage = (e) => console.log(e.data);
//   es.addEventListener('tick', (e) => console.log('tick:', e.data));
//   es.onerror = () => console.log('reconnecting…');
//
// Reconnection: If the connection drops, EventSource reconnects automatically
// after ~3 s and sends "Last-Event-ID" header so the server can resume.

func sseHandler(w http.ResponseWriter, r *http.Request) {
	// http.Flusher is the interface that lets us push bytes to the client
	// without waiting for the handler to return.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported (no Flusher)", http.StatusInternalServerError)
		return
	}

	// Required SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx proxy buffering
	w.WriteHeader(http.StatusOK)

	// Send an initial comment to flush proxy buffers (2048-byte padding trick)
	fmt.Fprintf(w, ": SSE stream connected\n\n")
	flusher.Flush()

	log.Printf("[SSE] Client %s connected", r.RemoteAddr)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	eventID := 0

	for {
		select {
		case <-r.Context().Done():
			// Client disconnected — r.Context() is cancelled when the connection closes
			log.Printf("[SSE] Client %s disconnected", r.RemoteAddr)
			return

		case t := <-ticker.C:
			eventID++
			// SSE event with id, named event type, and data fields
			fmt.Fprintf(w, "id: %d\n", eventID)
			fmt.Fprintf(w, "event: tick\n")
			fmt.Fprintf(w, "data: {\"time\":%q,\"id\":%d}\n", t.Format("15:04:05"), eventID)
			fmt.Fprintf(w, "\n") // blank line = end of this event
			flusher.Flush()
			log.Printf("[SSE] Sent event #%d", eventID)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Part C: Long Polling
// ─────────────────────────────────────────────────────────────────────────────
//
// Long polling is the simplest "push" technique that works over plain HTTP/1.0:
//  1. Client sends GET /poll
//  2. Server holds the connection open until:
//     a. New data is ready   → respond immediately with the data
//     b. Timeout (e.g. 30s) → respond with 204 No Content (client polls again)
//     c. Client disconnects → abandon silently
//  3. Client receives the response and immediately issues a new GET /poll
//
// Compared to SSE:
//   SSE:          persistent single connection, multiplexed events
//   Long polling: one request per message, works through any HTTP proxy
//
// Implementation uses a broadcast channel.  In production you'd use pub/sub
// (Redis, NATS) so multiple server instances can broadcast.

// pollBroker distributes messages to all waiting long-poll clients.
type pollBroker struct {
	mu       sync.Mutex
	waiters  []chan string
}

var broker = &pollBroker{}

// publish sends a message to all currently waiting long-poll clients.
func (b *pollBroker) publish(msg string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.waiters {
		select {
		case ch <- msg:
		default: // drop if the waiter already has a message
		}
	}
	b.waiters = b.waiters[:0]
}

// wait registers a waiter channel and returns it.
func (b *pollBroker) wait() chan string {
	ch := make(chan string, 1)
	b.mu.Lock()
	b.waiters = append(b.waiters, ch)
	b.mu.Unlock()
	return ch
}

// remove cleans up a waiter channel that timed out or disconnected.
func (b *pollBroker) remove(ch chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, w := range b.waiters {
		if w == ch {
			b.waiters = append(b.waiters[:i], b.waiters[i+1:]...)
			return
		}
	}
}

func pollHandler(w http.ResponseWriter, r *http.Request) {
	// 30-second timeout — standard for long polling
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	ch := broker.wait()

	select {
	case msg := <-ch:
		// New data arrived — return it immediately
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","message":%q,"time":%q}`+"\n",
			msg, time.Now().Format(time.RFC3339))
		log.Printf("[POLL] Delivered message to %s", r.RemoteAddr)

	case <-ctx.Done():
		// Timeout or client disconnect
		broker.remove(ch)
		if r.Context().Err() != nil {
			// Client disconnected before timeout
			log.Printf("[POLL] Client %s disconnected", r.RemoteAddr)
			return
		}
		// Timeout — tell client to poll again
		w.WriteHeader(http.StatusNoContent) // 204: no new data yet
		log.Printf("[POLL] Timeout for %s, returning 204", r.RemoteAddr)
	}
}

// publishLoop simulates a backend event source by publishing a random message
// every 3–7 seconds. In real life this would be driven by DB changes, MQ
// messages, etc.
func publishLoop() {
	for {
		sleep := time.Duration(3+rand.Intn(5)) * time.Second
		time.Sleep(sleep)
		msg := fmt.Sprintf("server-event-%d", time.Now().Unix())
		log.Printf("[BROKER] Publishing: %q", msg)
		broker.publish(msg)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// main
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	mux := http.NewServeMux()

	// Part A: WebSocket echo
	mux.HandleFunc("/ws", wsHandler)

	// Part B: SSE stream
	mux.HandleFunc("/events", sseHandler)

	// Part C: Long polling
	mux.HandleFunc("/poll", pollHandler)

	// Index page with quick instructions
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `Real-Time Communication Demo
=============================

Routes:
  /ws      WebSocket echo server (RFC 6455 manual handshake)
  /events  Server-Sent Events stream (tick every 2s)
  /poll    Long polling endpoint

Testing:
  WebSocket:
    wscat -c ws://localhost:8080/ws
    (then type messages — they are echoed back)

  SSE:
    curl -N http://localhost:8080/events
    (streams JSON tick events until Ctrl-C)

  Long poll:
    curl http://localhost:8080/poll
    (blocks until a backend event fires, then returns JSON)
    # Loop to simulate a real client:
    while true; do curl -s http://localhost:8080/poll; echo; done
`)
	})

	// Start the background publisher for long polling
	go publishLoop()

	addr := ":8080"
	fmt.Printf("Real-time server listening on http://localhost%s\n\n", addr)
	fmt.Println("  WebSocket:   wscat -c ws://localhost:8080/ws")
	fmt.Println("  SSE:         curl -N http://localhost:8080/events")
	fmt.Println("  Long poll:   curl http://localhost:8080/poll")
	fmt.Println()

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 0, // must be 0 for SSE/long-poll (streaming responses)
		IdleTimeout:  120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
