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
	path string
}

func New(path string) *Reader {
	return &Reader{
		path: path,
	}
}

func (r *Reader) Watch(
	ctx context.Context,
	handler LineHandler,
) error {
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		reopen, err := r.watchFile(
			ctx,
			handler,
		)

		if err != nil {
			return err
		}

		if !reopen {
			return nil
		}
	}
}

func (r *Reader) watchFile(
	ctx context.Context,
	handler LineHandler,
) (bool, error) {
	file, err := os.Open(r.path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Replay verarbeitet vorher den bestehenden Inhalt.
	// Der Live-Reader beginnt deshalb am Dateiende.
	position, err := file.Seek(
		0,
		io.SeekEnd,
	)
	if err != nil {
		return false, err
	}

	reader := bufio.NewReader(file)

	for {
		select {
		case <-ctx.Done():
			return false, nil

		default:
		}

		line, err := reader.ReadString('\n')

		if err == nil {
			position += int64(len(line))

			handler(line)
			continue
		}

		if !errors.Is(err, io.EOF) {
			return false, err
		}

		info, statErr := os.Stat(r.path)
		if statErr != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// WizardClient.log wurde geleert bzw.
		// durch eine neue Datei ersetzt.
		if info.Size() < position {
			return true, nil
		}

		time.Sleep(100 * time.Millisecond)
	}
}
