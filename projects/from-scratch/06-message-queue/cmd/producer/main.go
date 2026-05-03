package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", ":9001")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	for i := 1; i <= 5; i++ {
		fmt.Fprintf(conn, "PUB events message-%d\n", i)
		time.Sleep(500 * time.Millisecond)
	}
}
