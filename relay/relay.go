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
	usage := `relay

Usage:
  relay [--listen=<interface> --forward=<destination>]
  relay -h | --help
  relay --version

Options:
  -h --help                Show this screen.
  --version                Show version.
  --listen=<interface>     Interface to listen on [default: localhost:9999].
  --forward=<destination>  Destination to forward to [default: localhost:6666].`

	arguments, _ := docopt.Parse(usage, nil, true, "Relay v1.0.0", false)

	listenValue, hasListen := arguments["--listen"]
	listen := "localhost:9999"
	if hasListen {
		listenString, isString := listenValue.(string)
		if isString && len(listenString) > 0 {
			listen = listenString
		}
	}

	forwardValue, hasForward := arguments["--forward"]
	forward := "localhost:6666"
	if hasForward {
		forwardString, isString := forwardValue.(string)
		if isString && len(forwardString) > 0 {
			forward = forwardString
		}
	}

	listener, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatal("Cannot listen:", err)
		os.Exit(1)
	}
	defer listener.Close()

	for {
		acceptedConnection, err := listener.Accept()
		if err != nil {
			log.Fatal("Could not accept connection:", err)
			os.Exit(2)
		}
		go relayConnections(acceptedConnection, forward)
	}
}

func relayConnections(acceptedConnection net.Conn, remoteAddr string) {
	log.Println("connection accepted, relaying", acceptedConnection.RemoteAddr())
	defer acceptedConnection.Close()

	remote, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		log.Fatal("Could not make relay connection:", err)
		return
	}
	defer remote.Close()

	listenerInput := make(chan []byte, 100)
	forwardInput := make(chan []byte, 100)
	go connectionInput(acceptedConnection, listenerInput)
	go connectionInput(remote, forwardInput)

	for {
		select {
		case data, ok := <-listenerInput:
			{
				if !ok {
					log.Println("Input channel closed.")
					return
				}
				_, err := remote.Write(data)
				if err != nil {
					log.Println("Error writing data to channel:", err)
					return
				}
			}
		case data, ok := <-forwardInput:
			{
				if !ok {
					log.Println("Input channel closed.")
					return
				}
				_, err := acceptedConnection.Write(data)
				if err != nil {
					log.Println("Error writing data to channel:", err)
					return
				}
			}
		}
	}
}

func connectionInput(conn net.Conn, input chan []byte) {
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
