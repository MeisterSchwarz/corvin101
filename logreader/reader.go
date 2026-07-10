package logreader

import (
	"bufio"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type LineHandler func(string)

var (
	stopChan chan struct{}
	doneChan chan struct{}
	mu       sync.Mutex
)

func watchLoop(path string, handlers []LineHandler) {
	defer close(doneChan)

	openFile := func() (*os.File, *bufio.Reader, int64, error) {
		f, err := os.Open(path)
		if err != nil {
			return nil, nil, 0, err
		}

		info, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, nil, 0, err
		}

		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			f.Close()
			return nil, nil, 0, err
		}

		return f, bufio.NewReader(f), info.Size(), nil
	}

	file, reader, lastSize, err := openFile()
	if err != nil {
		log.Printf("[logreader] open failed: %v", err)
		return
	}
	defer file.Close()

	for {
		select {
		case <-stopChan:
			return

		default:
			info, err := file.Stat()
			if err != nil || info.Size() < lastSize {
				file.Close()

				file, reader, lastSize, err = openFile()
				if err != nil {
					log.Printf("[logreader] reopen failed: %v", err)
					time.Sleep(time.Second)
					continue
				}
			}

			_ = file.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			line, err := reader.ReadString('\n')
			if err != nil {
				continue
			}

			lastSize += int64(len(line))
			line = strings.TrimSpace(line)

			for _, handler := range handlers {
				handler(line)
			}
		}
	}
}

func ReadAllLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	return strings.Split(
		strings.TrimRight(string(data), "\n"),
		"\n",
	)
}

func Watch(path string, handlers ...LineHandler) {
	mu.Lock()
	defer mu.Unlock()

	stopChan = make(chan struct{})
	doneChan = make(chan struct{})

	go watchLoop(path, handlers)
}

func Stop() {
	mu.Lock()
	defer mu.Unlock()

	if stopChan == nil {
		return
	}

	close(stopChan)
	<-doneChan

	stopChan = nil
	doneChan = nil
}
