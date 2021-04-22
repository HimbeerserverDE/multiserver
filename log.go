package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tncardoso/gocurses"
)

const MaxLogMSGs = 1024

var logReady chan struct{}

func appendPop(max int, a []string, v ...string) []string {
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

type Logger struct {
	lines  []string
	all    []byte
	offset int
}

func newLogger() *Logger {
	l := &Logger{}
	initCurses(l)
	return l
}

func (l *Logger) Write(p []byte) (int, error) {
	rows, _ := gocurses.Getmaxyx()

	for i, line := range strings.Split(string(p)[:len(p)-1], "\n") {
		if i > 0 {
			t := time.Now()
			date := fmt.Sprintf("%.4d/%.2d/%.2d %.2d:%.2d:%.2d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())

			line = date + " " + line
		}

		l.lines = appendPop(MaxLogMSGs, l.lines, line)
		if l.offset > 0 {
			l.offset += 1
		}
	}

	start := len(l.lines) - rows + 1 - l.offset
	if start < 0 {
		start = 0
	}

	draw(l.lines[start : len(l.lines)-l.offset])
	l.all = append(l.all, p...)
	return len(p), nil
}

func (l *Logger) Close() {
	os.Mkdir("log", 0777)

	os.Rename("log/latest.txt", "log/last.txt")
	os.WriteFile("log/latest.txt", l.all, 0666)
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
