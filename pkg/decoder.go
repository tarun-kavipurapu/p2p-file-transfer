package pkg

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
)

type Decoder struct{}

func RegisterTypes() {
	gob.Register(Message{})
	gob.Register(FileMetaData{})
	gob.Register(PeerMetadata{})
	gob.Register(ChunkMetadata{})
}

func (d *Decoder) ReadBytes(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		return nil, fmt.Errorf("error reading bytes: %w", err)
	}
	return buf.Bytes(), nil
}

func (d *Decoder) DecodeFromBytes(data []byte, v interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("error decoding bytes: %w", err)
	}
	return nil
}

func (d *Decoder) EncodeToBytes(data interface{}) (bytes.Buffer, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(data); err != nil {
		return buf, fmt.Errorf("error encoding to bytes: %w", err)
	}
	return buf, nil
}

func (d *Decoder) DecodeIncomingMessage(r io.Reader, message *Message) error {
	dec := gob.NewDecoder(r)
	if err := dec.Decode(message); err != nil {
		return fmt.Errorf("error decoding incoming message: %w", err)
	}
	log.Println("Incoming Message", message.Type)
	return nil
}
