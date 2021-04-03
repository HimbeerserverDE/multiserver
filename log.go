package main

import (
	"log"
	"os"

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
		for i := 0; i < len(v); i++ {
			for j := 1; j < len(a); j++ {
				a[j-1] = a[j]
			}
		}
		return append(a[:max-len(v)], v...)
	}
}

type Logger []string

func newLogger() *Logger {
	os.Rename("log/latest.txt", "log/last.txt")

	l := &Logger{}
	initCurses(l)
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
