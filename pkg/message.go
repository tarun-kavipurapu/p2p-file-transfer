package pkg

const (
	IncomingStream  = 0x2
	IncomingMessage = 0x1
)

type msg struct {
	From     string
	FromAddr string
	Stream   bool
	Payload  []byte
}
