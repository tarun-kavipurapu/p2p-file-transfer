package pkg

type Message struct {
	Type     string //request type is sent by the peer
	From     string //from is added by the central server
	FromAddr string
	Payload  []byte //payload data is also sent by the peer incase of request  and by central server incase of respon
}

// func (m *Message) Serialize() ([]byte, error) {
// 	// Serialize the message to a byte slice. This example uses JSON for simplicity.
// 	return json.Marshal(m)
// }

// func (m *Message) Deserialize(data []byte) error {
// 	// Deserialize the byte slice back into the Message struct.
// 	return json.Unmarshal(data, m)
// }
