package pkg

type FileRegister struct {
}

// it is empty because i dont need to send any additional data for peer registration to central server if i need i will add it later
type PeerRegistration struct {
}
type DataMessage struct {
	Payload any
}

type RequestChunkData struct {
	FileId string
}
type ChunkRequestToPeer struct {
	FileId    string
	ChunkHash string
	ChunkId   uint32
	ChunkName string
}

type RegisterSeeder struct {
	FileId   string
	PeerAddr string
}
