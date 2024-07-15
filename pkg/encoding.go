package pkg

import (
	"encoding/gob"
	"io"
)

type Decoder interface {
	Reader(io.Reader, *msg) error
}

type GOBDecoder struct{}

func (dec GOBDecoder) Decode(r io.Reader, msg *msg) error {
	return gob.NewDecoder(r).Decode(msg)
}

type DefaultDecoder struct {
}

func (dec DefaultDecoder) Reader(r io.Reader, msg *msg) error {

	buf := make([]byte, 2048)
	n, err := r.Read(buf)
	if err != nil {
		return err
	}
	msg.Payload = buf[:n]

	return nil
}
