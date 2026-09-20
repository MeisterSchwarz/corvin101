package logreader

import (
	"bufio"
	"errors"
	"io"
	"os"
)

// Replay liest eine bestehende Wizard101-Logdatei vollständig
// von Anfang bis Ende.
//
// Wir verwenden absichtlich bufio.Reader statt bufio.Scanner.
// Wizard101 kann sehr große einzelne Logzeilen erzeugen.
// Scanner besitzt dagegen ein Token-Limit und kann deshalb mit
//
//	bufio.Scanner: token too long
//
// abbrechen.
func Replay(
	path string,
	handler LineHandler,
) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')

		// ReadString kann gleichzeitig Daten UND io.EOF
		// zurückgeben. Deshalb die letzte unvollständige
		// Zeile trotzdem noch verarbeiten.
		if len(line) > 0 {
			handler(line)
		}

		if err == nil {
			continue
		}

		if errors.Is(err, io.EOF) {
			return nil
		}

		return err
	}
}
