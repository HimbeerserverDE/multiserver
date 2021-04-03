package main

import (
	"log"
	"os"
	"unicode/utf8"

	"github.com/tncardoso/gocurses"
)

var logReady chan struct{}

var sep []byte = []byte(`
+-----------+
| Seperator |
+-----------+

`)

func WriteAppend(name string, data []byte, perm os.FileMode) error {
	os.Mkdir("log", 0777)

	b, err := os.ReadFile(name)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	err = os.WriteFile(name, append(b, data...), perm)
	if err != nil {
		return err
	}

	return nil
}

func appendPop(max int, a Logger, v ...string) Logger {
	if len(a) < max {
		return append(a, v...)
	} else {
		for i := 1; i <= len(a); i++ {
			a[i-1] = a[i]
		}
		return append(a[max-len(v):], v...)
	}
}

type Logger []string

func newLogger() *Logger {
	initCurses()
	os.Rename("log/latest.txt", "log/last.txt")

	l := &Logger{}

	go func() {
		for {
			var ch rune
			ch1 := gocurses.Stdscr.Getch() % 255
			if ch1 > 0x7F {
				ch2 := gocurses.Stdscr.Getch()
				ch, _ = utf8.DecodeRune([]byte{byte(ch1), byte(ch2)})
			} else {
				ch = rune(ch1)
			}

			switch ch {
			case '\b':
				if len(consoleInput) > 0 {
					consoleInput = consoleInput[:len(consoleInput)-1]
				}
			case '\n':
				consoleInput = []rune{}
			default:
				consoleInput = append(consoleInput, ch)
			}

			draw([]string(*l))
		}
	}()

	return l
}

func (l *Logger) Write(p []byte) (int, error) {
	row, _ := gocurses.Getmaxyx()
	*l = appendPop(row-1, *l, string(p))
	draw([]string(*l))

	// Write to file
	WriteAppend("log/latest.txt", p, 0666)

	return len(p), nil
}

func LogReady() <-chan struct{} {
	if logReady == nil {
		logReady = make(chan struct{})
	}

	return logReady
}

func init() {
	l := newLogger()
	log.SetOutput(l)

	go func() {
		for logReady == nil {
		}
		close(logReady)
	}()
}
