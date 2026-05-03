// Package server wraps the broker in a TCP server speaking the text protocol.
package server

import (
	"bufio"
	"log"
	"net"

	"tour_of_go/projects/from-scratch/06-message-queue/internal/protocol"
	"tour_of_go/projects/from-scratch/06-message-queue/internal/queue"
)

type Server struct {
	Addr   string
	broker *queue.Broker
	ln     net.Listener
}

func New(addr string, b *queue.Broker) *Server { return &Server{Addr: addr, broker: b} }

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	s.ln = ln
	log.Printf("mq-server listening on %s", s.Addr)
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
	scanner := bufio.NewScanner(conn)
	var subscriptions []struct {
		topic string
		ch    <-chan queue.Message
	}
	defer func() {
		for _, sub := range subscriptions {
			s.broker.Unsubscribe(sub.topic, sub.ch)
		}
	}()

	for scanner.Scan() {
		line := scanner.Text()
		cmd, args := protocol.Parse(line)
		switch cmd {
		case protocol.CmdPub:
			if len(args) < 2 {
				conn.Write([]byte(protocol.FormatErr("PUB requires topic and payload")))
				continue
			}
			s.broker.Publish(queue.Message{Topic: args[0], Payload: args[1]})
			conn.Write([]byte(protocol.FormatOK()))

		case protocol.CmdSub:
			if len(args) < 1 {
				conn.Write([]byte(protocol.FormatErr("SUB requires topic")))
				continue
			}
			topic := args[0]
			ch := s.broker.Subscribe(topic)
			subscriptions = append(subscriptions, struct {
				topic string
				ch    <-chan queue.Message
			}{topic, ch})
			conn.Write([]byte(protocol.FormatOK()))
			// Forward messages to client in a goroutine
			go func(topic string, ch <-chan queue.Message) {
				for msg := range ch {
					conn.Write([]byte(protocol.FormatMsg(msg.Topic, msg.Payload)))
				}
			}(topic, ch)

		default:
			conn.Write([]byte(protocol.FormatErr("unknown command")))
		}
	}
}
