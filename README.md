# p2p-file-trannsfer

### Done 
- Registering the file with the central server


- divide the file into chunks in the peer side
- cretae chunk meta Data for each chunk and map with file metadata
- divide the file into chunks
  - Read the file create a buffer of chunks size
  - read the buffer size of the file
- need to create seperate communication functionality in the peer to communicate with the peer
- A very simple thing everyone should have a listener and a Dialer 
- - the peerswithchunks should contain the ip of the peer server not that of the peer client
- - write the logic to extract that specifc chunk from the peer 
- if the peer is now availabile then it should try to extract the chunk from another peer which has registered 
- and i want to paralleleize the operation of extracting the peer
- i have to decide how to take which chunks from which peer


### Todo
- i have written the code of seperate chunks to the disk but i am facing the error and not getting desired result may be problem of 
- add the handshake functionality 


My logic 
- 


## Reigster Peer
```
{
    "type": "register_peer",
    "from": "peer_id:port",
    "payload": {
        "addr": "peer_address",
        "port": peer_port
    }
}
```

## Register File
```
{
    "type": "register_file",
    "from": "peer_id:port",
    "payload": {
        "file_id": "unique_file_id",
        "file_name": "file_name",
        "file_extension": "file_extension",
        "file_size": file_size,
        "chunk_size": chunk_size,
        "chunks": [
            {
                "chunk_index": chunk_index,
                "peers_with_chunk": ["peer_id1", "peer_id2"]
            },
            ...
        ]
    }
}

- peer will send this messaage to the central server 
- central server will receive this and send response accordingly 
```
### Request Chunks Messsage
```
{
    "type": "request_chunks",
    "from": "peer_id:port",
    "payload": {
        "file_id": "unique_file_id"
    }
}
```

### Response with FileData
```
{
    "type": "response_chunks",
    "from": "server",
    "payload": {
        "file_id": "unique_file_id",
        "file_name": "file_name",
        "file_extension": "file_extension",
        "file_size": file_size,
        "chunk_size": chunk_size,
        "chunks": [
            {
                "chunk_index": chunk_index,
                "peers_with_chunk": ["peer_id1", "peer_id2"]
            },
            ...
        ]
    }
}


```




### Idea

- Peer will peroform handshake with the Central server then the central server will display all the files it contains and using cli App  we will display all the files present in the system and the peer will select the file and send the file download request to the central server