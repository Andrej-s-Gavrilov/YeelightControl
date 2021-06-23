package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"
)

var (
	ip, port  string
	osSignals = make(chan os.Signal, 1)
)

func init() {
	const (
		default_ip   = "127.0.0.1"
		default_port = "55000"
	)
	flag.StringVar(&ip, "ip", default_ip, "Ip address for binding")
	flag.StringVar(&port, "port", default_port, "Port for binding")
}

func main() {
	var err error
	var tcpAddr *net.TCPAddr
	var listener *net.TCPListener
	flag.Parse()

	fmt.Printf("Service is working on %s:%s\n", ip, port)
	if tcpAddr, err = net.ResolveTCPAddr("tcp", ip+":"+port); err != nil {
		fmt.Printf("Fatal error: %s", err.Error())
		os.Exit(1)
	}

	if listener, err = net.ListenTCP("tcp", tcpAddr); err != nil {
		fmt.Printf("Fatal error: %s", err.Error())
		os.Exit(1)
	}

	signal.Notify(osSignals, os.Interrupt)
	go interruptionHandler(*listener)

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

func interruptionHandler(listenr net.TCPListener) {
	signal := <-osSignals
	fmt.Printf("Recived signal: %s\n", signal.String())
	if err := listenr.Close(); err != nil {
		fmt.Printf("Error during closing listener. Error: %s", err.Error())
	}
	fmt.Printf("Application is stoped")
	os.Exit(0)
}
