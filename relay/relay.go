package main

import (
	"github.com/docopt/docopt.go"
	"github.com/ngerakines/gotcprelay/common"
	"log"
	"net"
	"os"
	"time"
)

const (
	RECV_BUF_LEN = 1024
)

// ConnectionQueue is a basic FIFO queue based on a circular list that resizes as needed.
type ConnectionQueue struct {
	nodes []net.Conn
	head  int
	tail  int
	count int
}

// Push adds a node to the queue.
func (q *ConnectionQueue) Push(conn net.Conn) {
	if q.head == q.tail && q.count > 0 {
		nodes := make([]net.Conn, len(q.nodes)*2)
		copy(nodes, q.nodes[q.head:])
		copy(nodes[len(q.nodes)-q.head:], q.nodes[:q.head])
		q.head = 0
		q.tail = len(q.nodes)
		q.nodes = nodes
	}
	q.nodes[q.tail] = conn
	q.tail = (q.tail + 1) % len(q.nodes)
	q.count++
}

// Pop removes and returns a node from the queue in first to last order.
func (q *ConnectionQueue) Pop() net.Conn {
	if q.count == 0 {
		return nil
	}
	node := q.nodes[q.head]
	q.head = (q.head + 1) % len(q.nodes)
	q.count--
	return node
}

func (q *ConnectionQueue) Size() int {
	return q.count
}

func main() {
	log.Println("testing...")
	usage := `relay

Usage:
  relay [--external=<interface> --internal=<destination>]
  relay -h | --help
  relay --version

Options:
  -h --help                 Show this screen.
  --version                 Show version.
  --external=<interface>    Interface to listen on [default: localhost:9999].
  --internal=<destination>  Destination to forward to [default: localhost:9998].`

	arguments, _ := docopt.Parse(usage, nil, true, "Relay v1.0.0", false)
	log.Println(arguments)

	external := common.ArgOrDefault("--external", "localhost:9998", arguments)
	internal := common.ArgOrDefault("--internal", "localhost:9999", arguments)

	log.Println("Listening to external", external)
	externalListener := listenOrExit(external)
	defer externalListener.Close()

	log.Println("Listening to internal", internal)
	internalListener := listenOrExit(internal)
	defer internalListener.Close()

	externalConnectionsCh := make(chan net.Conn, 100)
	internalConnectionsCh := make(chan net.Conn, 100)

	externalConnectionsQueue := &ConnectionQueue{nodes: make([]net.Conn, 3)}
	internalConnectionsQueue := &ConnectionQueue{nodes: make([]net.Conn, 3)}

	go getConnections(externalListener, externalConnectionsCh)
	go getConnections(internalListener, internalConnectionsCh)

	log.Println("Starting main run loop.")

	for {
		select {
		case externalConnection := <-externalConnectionsCh:
			{
				externalConnectionsQueue.Push(externalConnection)
			}
		case internalConnection := <-internalConnectionsCh:
			{
				internalConnectionsQueue.Push(internalConnection)
			}
		case <-time.After(10 * time.Second):
			log.Println("No connections, tick")
		}
		// NKG: It is important to note that there isn't any code that checks
		// to see if the connections are still open/active once they've been
		// put into the queue.
		if externalConnectionsQueue.Size() > 0 && internalConnectionsQueue.Size() > 0 {
			externalConnection := externalConnectionsQueue.Pop()
			internalConnection := internalConnectionsQueue.Pop()
			go relayPair(externalConnection, internalConnection)
		}
	}
}

func listenOrExit(listenInterface string) net.Listener {
	listener, err := net.Listen("tcp", listenInterface)
	if err != nil {
		log.Fatal("Cannot listen:", err)
		os.Exit(1)
	}
	return listener
}

func getConnections(listener net.Listener, connectionsCh chan net.Conn) {
	log.Println("Listening for connections with", listener)
	for {
		acceptedConnection, err := listener.Accept()
		if err != nil {
			log.Fatal("Could not accept connection:", err)
			os.Exit(2)
		}
		log.Println("Got a connection on", listener)
		connectionsCh <- acceptedConnection
	}
}

func relayPair(externalConnection, internalConnection net.Conn) {
	defer externalConnection.Close()
	defer internalConnection.Close()

	externalInput := make(chan []byte, 100)
	internalInput := make(chan []byte, 100)
	go connectionInput(externalConnection, externalInput)
	go connectionInput(internalConnection, internalInput)

	for {
		select {
		case data, ok := <-externalInput:
			{
				if !ok {
					log.Println("External input channel closed.")
					return
				}
				_, err := internalConnection.Write(data)
				if err != nil {
					log.Println("Error writing data to channel:", err)
					return
				}
			}
		case data, ok := <-internalInput:
			{
				if !ok {
					log.Println("Internal input channel closed.")
					return
				}
				_, err := externalConnection.Write(data)
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
