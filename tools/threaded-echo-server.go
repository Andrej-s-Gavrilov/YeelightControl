package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Use %s ip:port\n", os.Args[0])
		os.Exit(1)
	}
	service := os.Args[1]
	fmt.Printf("Service is working on %s\n", service)
	tcpAddr, err := net.ResolveTCPAddr("tcp", service)
	checkError(err)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handlerClient(conn)
	}

}

func handlerClient(conn net.Conn) {
	defer func() {
		fmt.Printf("[%s] Client %s ->X-> Local addres is %s\n", time.Now().Format("15:04:05.000"), conn.RemoteAddr().String(), conn.LocalAddr().String())
		conn.Close()
	}()

	fmt.Printf("[%s] Client %s ->V-> Local addres is %s\n", time.Now().Format("15:04:05.000"), conn.RemoteAddr().String(), conn.LocalAddr().String())
	var buf [512]byte
	for {
		n, err := conn.Read(buf[0:])
		if err != nil {
			return
		}
		fmt.Printf("[%s] Recived message: %s", time.Now().Format("15:04:05.000"), string(buf[0:n]))
		_, err2 := conn.Write(buf[0:n])
		if err2 != nil {
			return
		}

	}
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
