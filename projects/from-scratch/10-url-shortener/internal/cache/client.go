// Package cache provides a TCP client for the 07-distributed-cache RESP server.
package cache

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

// Client is a minimal RESP client for the from-scratch cache server.
type Client struct {
	conn net.Conn
	r    *bufio.Reader
}

func Dial(addr string) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, r: bufio.NewReader(conn)}, nil
}

func (c *Client) Close() { c.conn.Close() }

// Set stores key=value with optional TTL seconds (0 = no expiry).
func (c *Client) Set(key, value string, ttlSec int) error {
	var cmd string
	if ttlSec > 0 {
		cmd = fmt.Sprintf("*5\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n$2\r\nEX\r\n$%d\r\n%d\r\n",
			len(key), key, len(value), value, len(fmt.Sprint(ttlSec)), ttlSec)
	} else {
		cmd = fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n",
			len(key), key, len(value), value)
	}
	_, err := c.conn.Write([]byte(cmd))
	if err != nil {
		return err
	}
	_, err = c.r.ReadString('\n')
	return err
}

// Get retrieves a value by key. Returns ("", ErrNotFound) if missing.
func (c *Client) Get(key string) (string, error) {
	cmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(key), key)
	c.conn.Write([]byte(cmd))
	line, err := c.r.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "$-1" {
		return "", fmt.Errorf("not found")
	}
	if strings.HasPrefix(line, "$") {
		val, err := c.r.ReadString('\n')
		return strings.TrimRight(val, "\r\n"), err
	}
	return "", fmt.Errorf("unexpected response: %s", line)
}

// Del removes a key.
func (c *Client) Del(key string) error {
	cmd := fmt.Sprintf("*2\r\n$3\r\nDEL\r\n$%d\r\n%s\r\n", len(key), key)
	c.conn.Write([]byte(cmd))
	_, err := c.r.ReadString('\n')
	return err
}
