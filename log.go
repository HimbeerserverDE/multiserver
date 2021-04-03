package main

import (
	"fmt"
	"log"
	"os"
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

type Logger struct {
}

func newLogger() *Logger {
	os.Rename("log/latest.txt", "log/last.txt")

	return &Logger{}
}

func (l *Logger) Write(p []byte) (int, error) {
	fmt.Print(string(p))

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
