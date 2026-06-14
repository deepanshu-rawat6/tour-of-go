// Package mq provides a TCP client for the 06-message-queue server.
package mq

import (
	"fmt"
	"net"
	"time"
)

// Client publishes messages to the from-scratch message queue.
type Client struct {
	conn net.Conn
}

func Dial(addr string) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

func (c *Client) Close() { c.conn.Close() }

// Publish sends a message to the given topic.
func (c *Client) Publish(topic, payload string) error {
	_, err := fmt.Fprintf(c.conn, "PUB %s %s\n", topic, payload)
	return err
}
