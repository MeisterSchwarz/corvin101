package logreader

import (
	"bufio"
	"errors"
	"io"
	"os"
)

// Replay keeps the original API for other callers.
func Replay(path string, handler LineHandler) error {
	_, err := ReplayWithOffset(path, handler)
	return err
}

// ReplayWithOffset returns the last complete-line byte offset.
// The trailing partial line is left for the live reader.
func ReplayWithOffset(path string, handler LineHandler) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	var offset int64
	for {
		line, readErr := reader.ReadString('\n')
		if readErr == nil {
			offset += int64(len(line))
			handler(line)
			continue
		}
		if errors.Is(readErr, io.EOF) {
			return offset, nil
		}
		return offset, readErr
	}
}
