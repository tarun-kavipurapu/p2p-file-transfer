package pkg

import (
	"bytes"
	"fmt"
	"io"
)

func ReadBytes(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		return nil, fmt.Errorf("error reading bytes: %w", err)
	}
	return buf.Bytes(), nil
}
