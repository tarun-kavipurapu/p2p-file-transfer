
#### Register file
- request->fileid file size chunksize and chunk information


### Request file


### Todo
- add the received file to the metadata of the peer
- make a mechanism to combine all the chunks and verify the file with the hash id 
- Encryption od daata while transfering 
- Register the file received peer to the Central server chunks information 
- Ensure that load balancing is happening as Thought

To enhance your P2P file transfer CLI application, you can consider adding the following features:

1. **Progress Reporting**: Show progress of file transfers.
2. **Encryption**: Encrypt data during transfer for security.
3. **Resumable Transfers**: Allow pausing and resuming of file transfers.
4. **Peer Discovery**: Implement dynamic peer discovery instead of static IP addresses.
5. **File Integrity Check**: Verify the integrity of transferred files using hashes.
6. **Peer Status Monitoring**: Monitor the status of connected peers.
7. **Concurrency Control**: Limit the number of concurrent transfers.
8. **File Search**: Allow searching for files available on the network.
9. **Logging**: Implement detailed logging for debugging and auditing.
10. **Configuration Management**: Add support for configuration files and environment variables.
11. **Web Interface**: Provide a web interface for users to interact with the system.
12. **File Compression**: Compress files before transfer to save bandwidth.

### Example Implementations

#### Progress Reporting

You can use a library like `progressbar` to show transfer progress.

```go
import (
	"github.com/schollz/progressbar/v3"
)

func (p *PeerServer) fileRequest(fileId string, chunkIndex uint32, peerAddr string, chunkHash string) error {
	// ... previous code ...
	bar := progressbar.NewOptions64(
		fileSize,
		progressbar.OptionSetDescription(fmt.Sprintf("Downloading chunk %d", chunkId)),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=",
			SaucerHead:    "[green]>",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
	if _, err := io.Copy(io.MultiWriter(file, bar), peer); err != nil {
		log.Printf("Failed to copy file: %v\n", err)
		panic("Error ")
	}
	log.Printf("Successfully received and saved chunk_%d.chunk\n", chunkId)
	return nil
}
```

#### Encryption

Use a library like `crypto` to encrypt/decrypt data.

```go
import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

func encrypt(data []byte, passphrase string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(passphrase))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func decrypt(data []byte, passphrase string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(passphrase))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
```

#### Resumable Transfers

Implementing resumable transfers requires tracking the progress of each file and chunk. You can store metadata about the transfer status and resume from the last known state.

```go
import (
	"encoding/json"
	"os"
)

type TransferStatus struct {
	FileID     string
	ChunkIndex uint32
	Offset     int64
}

func saveTransferStatus(status TransferStatus) error {
	file, err := os.Create(fmt.Sprintf("%s_transfer_status.json", status.FileID))
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(status)
}

func loadTransferStatus(fileID string) (TransferStatus, error) {
	var status TransferStatus
	file, err := os.Open(fmt.Sprintf("%s_transfer_status.json", fileID))
	if err != nil {
		return status, err
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(&status)
	return status, err
}
```

#### Peer Discovery

You can use a discovery service or protocol like mDNS for peer discovery.

```go
import (
	"github.com/grandcat/zeroconf"
	"context"
	"log"
	"time"
)

func discoverPeers(service string) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err)
	}

	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			log.Println(entry)
		}
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	err = resolver.Browse(ctx, service, "local.", entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err)
	}

	<-ctx.Done()
}
```

#### File Integrity Check

Verify the integrity of transferred files using hashes.

```go
func verifyFile(filePath string, expectedHash string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false
	}
	return hex.EncodeToString(hash.Sum(nil)) == expectedHash
}
```

### Implementing Subcommands for New Features

#### cmd/progress.go
```go
package cmd

import (
	"log"
	"github.com/spf13/cobra"
)

var showProgress bool

var progressCmd = &cobra.Command{
	Use:   "progress",
	Short: "Show progress of file transfers",
	Run: func(cmd *cobra.Command, args []string) {
		if showProgress {
			// Implement progress reporting logic here
			log.Println("Showing progress...")
		} else {
			log.Println("Progress reporting is disabled.")
		}
	},
}

func init() {
	rootCmd.AddCommand(progressCmd)
	progressCmd.Flags().BoolVarP(&showProgress, "show", "s", false, "Show progress of file transfers")
}
```

#### cmd/encrypt.go
```go
package cmd

import (
	"log"
	"github.com/spf13/cobra"
)

var passphrase string

var encryptCmd = &cobra.Command{
	Use:   "encrypt",
	Short: "Encrypt files before transfer",
	Run: func(cmd *cobra.Command, args []string) {
		// Implement encryption logic here
		log.Println("Encrypting files with passphrase:", passphrase)
	},
}

func init() {
	rootCmd.AddCommand(encryptCmd)
	encryptCmd.Flags().StringVarP(&passphrase, "passphrase", "p", "", "Passphrase for encryption")
	encryptCmd.MarkFlagRequired("passphrase")
}
```

### Conclusion

By adding these features, you can significantly enhance the functionality and usability of your P2P file transfer application. The `cobra` CLI framework helps in organizing commands and subcommands, making your application more user-friendly and extensible.