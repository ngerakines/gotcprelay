package main

import (
	"github.com/docopt/docopt.go"
	"log"
	"net"
	"os"
)

const (
	RECV_BUF_LEN = 1024
)

func main() {
	usage := `echoserver

Usage:
  echoserver [--listen=<interface> --forward=<destination>]
  echoserver -h | --help
  echoserver --version

Options:
  -h --help                Show this screen.
  --version                Show version.
  --listen=<interface>     Interface to listen on [default: localhost:6666].`

	arguments, _ := docopt.Parse(usage, nil, true, "echoserver v1.0.0", false)

	listenValue, hasListen := arguments["--listen"]
	listen := "localhost:6666"
	if hasListen {
		listenString, isString := listenValue.(string)
		if isString && len(listenString) > 0 {
			listen = listenString
		}
	}

	listener, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatal("Cannot listen:", err)
		os.Exit(1)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("Could not accept connection:", err)
			os.Exit(2)
		}
		go echoFunc(conn)
	}
}

func echoFunc(conn net.Conn) {

	input := make(chan []byte, 100)
	go echoInput(conn, input)

	for {
		select {
		case data, ok := <-input:
			{
				if !ok {
					log.Println("Input channel closed.")
					return
				}
				_, err := conn.Write(data)
				if err != nil {
					log.Println("Error writing data to channel:", err)
					return
				}
			}
		}
	}
}

func echoInput(conn net.Conn, input chan []byte) {
	for {
		buf := make([]byte, RECV_BUF_LEN)
		n, err := conn.Read(buf)
		if err != nil {
			close(input)
			return
		}
		log.Println("received ", n, " bytes of data =", string(buf))
		input <- buf
	}
}
