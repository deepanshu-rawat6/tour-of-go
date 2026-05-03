// Command client connects to the echo server and sends a message.
package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	addr := ":9000"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	msg := "hello, tcp-server!"
	conn.Write([]byte(msg))
	conn.(*net.TCPConn).CloseWrite()

	buf := make([]byte, len(msg))
	conn.Read(buf)
	fmt.Printf("echo: %s\n", buf)
}
