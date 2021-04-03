package main

import (
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

func initCurses() {
	gocurses.Initscr()
	gocurses.Cbreak()
	gocurses.Noecho()
	gocurses.Stdscr.Keypad(true)
}
