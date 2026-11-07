package logreader

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"time"
)

type LineHandler func(string)

type Reader struct {
	path        string
	startOffset int64
}

func New(path string) *Reader { return &Reader{path: path, startOffset: -1} }

// FromOffset starts at the exact byte position reached by ReplayWithOffset.
func (r *Reader) FromOffset(offset int64) *Reader {
	if offset < 0 {
		offset = 0
	}
	r.startOffset = offset
	return r
}

func (r *Reader) Watch(ctx context.Context, handler LineHandler) error {
	offset := r.startOffset
	for {
		if ctx.Err() != nil {
			return nil
		}
		next, reopen, err := r.watchFile(ctx, handler, offset)
		if err != nil {
			return err
		}
		if !reopen {
			return nil
		}
		offset = next
	}
}

func (r *Reader) watchFile(ctx context.Context, handler LineHandler, offset int64) (int64, bool, error) {
	file, err := os.Open(r.path)
	if err != nil {
		return offset, false, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return offset, false, err
	}
	if offset < 0 {
		offset = info.Size()
	} // backwards-compatible live-only use
	if offset > info.Size() {
		offset = 0
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return offset, false, err
	}
	reader := bufio.NewReader(file)
	position := offset
	for {
		if ctx.Err() != nil {
			return position, false, nil
		}
		line, readErr := reader.ReadString('\n')
		if readErr == nil {
			position += int64(len(line))
			handler(line)
			continue
		}
		if !errors.Is(readErr, io.EOF) {
			return position, false, readErr
		}
		// A partial line must be re-read when its newline arrives.
		if len(line) > 0 {
			if _, err := file.Seek(position, io.SeekStart); err != nil {
				return position, false, err
			}
			reader.Reset(file)
		}
		pathInfo, statErr := os.Stat(r.path)
		if statErr == nil {
			if !os.SameFile(info, pathInfo) || pathInfo.Size() < position {
				return 0, true, nil
			}
		}
		select {
		case <-ctx.Done():
			return position, false, nil
		case <-time.After(100 * time.Millisecond):
		}
	}
}
