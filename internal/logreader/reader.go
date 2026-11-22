package logreader

import (
	"bufio"
	"bytes"
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
	onReset     func()
}

func New(path string) *Reader { return &Reader{path: path, startOffset: -1} }

// OnReset resets parser/state before replaying a replaced or rewritten log.
func (r *Reader) OnReset(handler func()) *Reader { r.onReset = handler; return r }

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
		if r.onReset != nil {
			r.onReset()
		}
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
		if r.onReset != nil {
			r.onReset()
		}
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return offset, false, err
	}
	reader := bufio.NewReader(file)
	position := offset
	// Two clients can overwrite the same log without shrinking or replacing
	// it. Remember consumed content rather than relying only on file size.
	anchor := make([]byte, min(int64(256), position))
	if len(anchor) > 0 {
		if _, err := file.ReadAt(anchor, position-int64(len(anchor))); err != nil {
			return position, false, err
		}
	}
	checkAnchor := false
	for {
		if ctx.Err() != nil {
			return position, false, nil
		}
		if checkAnchor && len(anchor) > 0 {
			current := make([]byte, len(anchor))
			_, err := file.ReadAt(current, position-int64(len(anchor)))
			if err != nil && !errors.Is(err, io.EOF) {
				return position, false, err
			}
			if errors.Is(err, io.EOF) || !bytes.Equal(anchor, current) {
				return 0, true, nil
			}
		}
		checkAnchor = false
		line, readErr := reader.ReadString('\n')
		if readErr == nil {
			position += int64(len(line))
			anchor = []byte(line[max(0, len(line)-256):])
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
			checkAnchor = true
		}
	}
}
