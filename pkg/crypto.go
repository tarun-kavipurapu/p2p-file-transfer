package pkg

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func HashFile(file *os.File) (string, error) {
	hash := sha256.New()
	defer file.Seek(0, io.SeekStart)

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		return "", err
	}
	if _, err := hash.Write(buf.Bytes()); err != nil {
		return "", fmt.Errorf("error updating hash: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
