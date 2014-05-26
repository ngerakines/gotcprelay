package main

import (
	"github.com/docopt/docopt.go"
	"github.com/ngerakines/gotcprelay/common"
	"log"
	"net"
	"time"
)

const (
	RECV_BUF_LEN = 1024
)

type ConnectionPair struct {
	id   int
	conn net.Conn
}

func main() {
	usage := `echoserver

Usage:
  echoserver [--listen=<interface>]
  echoserver -h | --help
  echoserver --version

Options:
  -h --help                Show this screen.
  --version                Show version.
  --listen=<interface>     Interface to listen on [default: localhost:9998].`

	arguments, _ := docopt.Parse(usage, nil, true, "echoserver v1.0.0", false)

	listen := common.ArgOrDefault("--listen", "localhost:9998", arguments)

	// id to bool (true if has received activity)
	connections := make(map[int]bool)
	connectionPairs := make(chan ConnectionPair, 100)
	connectionActivity := make(chan int, 100)
	closedConnections := make(chan int, 100)

	lastId := 0

	for {
		select {
		case connection, ok := <-connectionPairs:
			{
				if !ok {
					return
				}
				connections[connection.id] = false
				go echoFunc(connection.conn)
			}
		case connection, ok := <-connectionActivity:
			{
				if !ok {
					return
				}
				connections[connection] = true
			}
		case connection, ok := <-closedConnections:
			{
				if !ok {
					return
				}
				delete(connections, connection)
			}
		case <-time.After(10 * time.Second):
			{
				count := 0
				for _, hasReceivedData := range connections {
					if hasReceivedData {
						count++
					}
				}
				if count == len(connections) {
					for i := 0; i < 3; i++ {
						lastId++
						createConn(listen, lastId, connectionPairs)
					}
				}
			}
		}
	}
}

func echoFunc(conn net.Conn) {
	defer conn.Close()

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

func createConn(remoteAddress string, id int, connectionPairs chan ConnectionPair) {
	connection, err := net.Dial("tcp", remoteAddress)
	if err != nil {
		log.Fatal("Could not make connection:", err)
		return
	}
	connectionPairs <- ConnectionPair{id, connection}
}
