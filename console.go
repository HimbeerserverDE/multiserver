package main

import (
	"unicode/utf8"

	"github.com/tncardoso/gocurses"
)

var consoleInput []rune

func draw(msgs []string) {
	gocurses.Clear()

	row, _ := gocurses.Getmaxyx()

	i := len(msgs)
	for _, msg := range msgs {
		gocurses.Mvaddstr(row-i-1, 0, msg)
		i--
	}
	gocurses.Mvaddstr(row-i-1, 0, "> "+string(consoleInput))

	gocurses.Refresh()
}

func initCurses(l *Logger) {
	gocurses.Initscr()
	gocurses.Cbreak()
	gocurses.Noecho()
	gocurses.Stdscr.Keypad(true)

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
}