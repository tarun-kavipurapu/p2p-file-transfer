Sure, here's a `README.md` file for your project, which provides an overview, installation instructions, usage details, and a description of the functionality.


# P2P File Transfer

A peer-to-peer (P2P) file transfer system implemented in Go. This system allows for efficient file sharing and chunk-based data transfer between peers, utilizing a central server for peer discovery and file metadata management.

## Features

- **Peer Discovery**: Utilizes a central server for dynamic peer discovery and registration.
- **Chunk-Based File Transfer**: Files are divided into chunks for efficient transfer and storage.
- **Concurrent Transfers**: Supports concurrent chunk downloads to optimize transfer speed.
- **File Integrity Verification**: Ensures file integrity through hash verification.
- **Error Handling**: Robust error handling and logging for reliable operation.
- **Configurable Transport**: Supports TCP transport for peer communication.

## Installation

1. Clone the repository:
    ```sh
    git clone https://github.com/yourusername/p2p-file-transfer.git
    cd p2p-file-transfer
    ```

2. Build the project:
    ```sh
    go build -o p2p-transfer
    ```

## Usage

### Starting the Central Server

The central server is responsible for managing peer registration and file metadata. Start the central server with:

```sh
./p2p-transfer central-server --addr <address:port>
```

### Starting a Peer Server

A peer server handles file requests and transfers. Start a peer server with:

```sh
./p2p-transfer peer-server --central-server-addr <central-server-address:port> --addr <peer-address:port>
```

### Registering a File

To register a file with the peer network, use the following command:

```sh
./p2p-transfer register-file --path <file-path> --peer-addr <peer-address:port>
```

### Requesting a File

To request a file from the network, use the following command:

```sh
./p2p-transfer request-file --file-id <file-id> --peer-addr <peer-address:port>
```

## Command-Line Interface (CLI)

The application uses a CLI for interacting with the system. The following commands are available:

- `central-server`: Starts the central server.
- `peer-server`: Starts a peer server.
- `register-file`: Registers a file with the peer network.
- `request-file`: Requests a file from the peer network.

Each command has its own set of flags and options for configuration.

### Example

Starting the central server:
```sh
./p2p-transfer central-server --addr 127.0.0.1:8000
```

Starting a peer server:
```sh
./p2p-transfer peer-server --central-server-addr 127.0.0.1:8000 --addr 127.0.0.1:8001
```

Registering a file:
```sh
./p2p-transfer register-file --path /path/to/file.txt --peer-addr 127.0.0.1:8001
```

Requesting a file:
```sh
./p2p-transfer request-file --file-id <file-id> --peer-addr 127.0.0.1:8001
```

## Project Structure

- `pkg/`: Contains the main package for message types and peer management.
- `peer/`: Contains the peer server logic and file handling functions.
- `centralserver/`: Contains the central server logic for peer registration and metadata management.
- `cmd/`: Contains the CLI commands for interacting with the system.

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request with your changes.

