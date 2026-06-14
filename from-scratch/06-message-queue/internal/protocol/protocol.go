// Package protocol defines the simple text protocol for the message queue TCP server.
//
// Commands (client → server):
//   PUB <topic> <payload>\n   — publish a message
//   SUB <topic>\n             — subscribe to a topic
//
// Server → client:
//   MSG <topic> <payload>\n   — delivered message
//   OK\n                      — acknowledgement
//   ERR <reason>\n            — error
package protocol

import (
	"fmt"
	"strings"
)

const (
	CmdPub = "PUB"
	CmdSub = "SUB"
	MsgMsg = "MSG"
	MsgOK  = "OK"
	MsgErr = "ERR"
)

func FormatMsg(topic, payload string) string {
	return fmt.Sprintf("MSG %s %s\n", topic, payload)
}

func FormatOK() string { return "OK\n" }

func FormatErr(reason string) string { return fmt.Sprintf("ERR %s\n", reason) }

// Parse splits a raw line into command and arguments.
func Parse(line string) (cmd string, args []string) {
	parts := strings.SplitN(strings.TrimSpace(line), " ", 3)
	if len(parts) == 0 {
		return "", nil
	}
	return strings.ToUpper(parts[0]), parts[1:]
}
