package pkg

import (
	"fmt"
	"log"
	"net"
	"sync"
)

type TCPClient struct {
	net.Conn

	// if we dial and retrieve a conn => outbound == true
	// if we accept and retrieve a conn => outbound == false
	outbound bool

	wg *sync.WaitGroup
}

func NewTCPClient(conn net.Conn, outbound bool) *TCPClient {
	return &TCPClient{
		Conn:     conn,
		outbound: outbound,
		wg:       &sync.WaitGroup{},
	}
}

func (p *TCPClient) CloseStream() {
	p.wg.Done()
}
func (p *TCPClient) Send(data []byte) error {
	_, err := p.Conn.Write(data)
	return err
}

// TCPOptions contains options for TCPTransport
type TCPOptions struct {
	Decoder Decoder
}

// TCPServer represents a TCP server
type TCPTransport struct {
	TCPOptions
	listenerAddr string
	ln           net.Listener
	msgch        chan Message
	quitch       chan struct{}
}

// NewTCPTransport creates a new TCP server
func NewTCPTransport(listener string, decoder Decoder) *TCPTransport { // Added decoder parameter
	return &TCPTransport{
		TCPOptions:   TCPOptions{Decoder: decoder},
		listenerAddr: listener,
		msgch:        make(chan Message),
		quitch:       make(chan struct{}),
	}
}

// Start starts the TCP server
func (t *TCPTransport) Start() error {
	ln, err := net.Listen("tcp", t.listenerAddr)
	if err != nil {
		return err
	}
	t.ln = ln
	fmt.Println("Server listening on", t.listenerAddr)

	go t.startLoop()
	<-t.quitch
	return nil
}
func (t *TCPTransport) Consume() <-chan Message {
	return t.msgch
}
func (t *TCPTransport) Close() error {
	return t.ln.Close()
}

func (t *TCPTransport) startLoop() {
	for {
		conn, err := t.ln.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		log.Println("Connection accepted from", conn.RemoteAddr().String())
		go t.readLoop(conn)
	}
}

func (t *TCPTransport) Dial(addr string) (net.Conn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return conn, err

}

func (t *TCPTransport) readLoop(conn net.Conn) {
	defer func() {
		conn.Close()
	}()
	// peer := NewTCPPeer(conn, outbound)

	// if err = t.HandshakeFunc(peer); err != nil {
	// 	return
	// }

	// if t.OnPeer != nil {
	// 	if err = t.OnPeer(peer); err != nil {
	// 		return
	// 	}
	// }

	for {
		message := Message{}
		err := t.Decoder.DecodeIncomingMessage(conn, &message)
		if err != nil {
			log.Println("Error decoding message:", err)
			return
		}
		// message.From = conn.RemoteAddr().String()
		t.msgch <- message
		// log.Println(message)

		echoMessage := []byte("Echoing the message: " + string(message.Payload))
		_, err = conn.Write(echoMessage)
		if err != nil {
			log.Println("Error writing message back to client:", err)
			return
		}
	}
}
