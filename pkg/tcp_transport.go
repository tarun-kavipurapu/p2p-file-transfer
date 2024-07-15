package pkg

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
)

//any messaage received by Transport  or peer componetn is sent into the msg channel

/*
basically when ever a new peer or central server is connected it goes to handle Conn and from there a new Peer is created and this peer is connected
we can perform some actions like adding the peer to the ppeer map usin t.Onpeer function
*/

// this is a client component
type TCPPeer struct {
	net.Conn

	// if we dial and retrieve a conn => outbound == true
	// if we accept and retrieve a conn => outbound == false

	outbound bool

	wg *sync.WaitGroup
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {

	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,

		wg: &sync.WaitGroup{},
	}
}

func (p *TCPPeer) CloseStream() {
	p.wg.Done()
}

func (p *TCPPeer) Send(b []byte) error {

	_, err := p.Write(b)
	return err
}

type TransportOpts struct {
	ListenAddr string
	// HandshakeFunc HandshakeFunc
	Decoder Decoder
	OnPeer  func(Peer, string) error // Add connection type parameter
}

// this is a server componets which also has dial componet
type TCPTransport struct {
	TransportOpts

	listener net.Listener

	msgch chan msg
}

func NewTCPTransport(opts TransportOpts) *TCPTransport {

	return &TCPTransport{
		TransportOpts: opts,
		msgch:         make(chan msg, 1024),
	}
}

// Addr implements the Transport interface return the address
// the transport is accepting connections.
func (t *TCPTransport) Addr() string {
	return t.ListenAddr
}

// Consume implements the Tranport interface, which will return read-only channel
// for reading the incoming messages received from another peer in the network.
func (t *TCPTransport) Consume() <-chan msg {
	return t.msgch
}

// Close implements the Transport interface.
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

// Dial implements the Transport interface.
func (t *TCPTransport) Dial(addr string, connType string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	go t.handleConn(conn, true, connType)

	return nil
}

func (s *TCPTransport) ListenAndAccept(connType string) error {
	var err error

	s.listener, err = net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return err
	}

	go s.startAcceptLoop(connType)

	log.Printf("TCP transport listening on port: %s\n", s.ListenAddr)

	return nil
}

func (t *TCPTransport) startAcceptLoop(connType string) {
	for {
		conn, err := t.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}

		if err != nil {
			fmt.Printf("TCP accept error: %s\n", err)
		}

		go t.handleConn(conn, false, connType)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool, connType string) {
	var err error

	defer func() {
		fmt.Printf("dropping peer connection: %s", err)
		conn.Close()
	}()

	peer := NewTCPPeer(conn, outbound)

	// if err = s.HandshakeFunc(peer); err != nil {
	// 	return
	// }

	// we use this to connect the peres map and the peer created in the hanndleConn
	if t.OnPeer != nil {
		if err = t.OnPeer(peer, connType); err != nil {
			return
		}
	}

	// Read loop
	for {
		msg := msg{}
		err = t.Decoder.Reader(conn, &msg)
		if err != nil {
			return
		}
		msg.From = conn.RemoteAddr().String()
		// log.Println(msg.Payload)

		t.msgch <- msg

	}
}
