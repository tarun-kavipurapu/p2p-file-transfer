package pkg

import "net"

type Peer interface {
	net.Conn
	Send([]byte) error
	CloseStream()
}

type Transport interface {
	Addr() string
	Dial(string, string) error
	ListenAndAccept(string) error
	Consume() <-chan msg
	Close() error
}

// type CommHandler interface {
// 	ClientComm
// 	ServerComm
// }
