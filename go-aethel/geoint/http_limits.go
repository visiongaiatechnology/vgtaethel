package geoint

// STATUS: DIAMANT VGT SUPREME

import (
	"fmt"
	"io"
)

func readBoundedBody(reader io.Reader, maximumBytes int64) ([]byte, error) {
	if reader == nil || maximumBytes <= 0 {
		return nil, fmt.Errorf("invalid response boundary")
	}
	data, err := io.ReadAll(io.LimitReader(reader, maximumBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maximumBytes {
		return nil, fmt.Errorf("upstream response exceeds size boundary")
	}
	return data, nil
}

