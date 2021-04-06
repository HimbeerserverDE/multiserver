package main

import (
	"log"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/tncardoso/gocurses"
)

var consoleInput []rune

func draw(msgs []string) {
	prompt, ok := ConfKey("console_prompt").(string)
	if !ok {
		prefix, ok := ConfKey("command_prefix").(string)
		if !ok {
			prefix = "#"
		}

		prompt = prefix + ">"
	}

	gocurses.Clear()

	row, _ := gocurses.Getmaxyx()

	i := len(msgs)
	for _, msg := range msgs {
		gocurses.Mvaddstr(row-i-1, 0, msg)
		i--
	}
	gocurses.Mvaddstr(row-i-1, 0, prompt+string(consoleInput))

	gocurses.Refresh()
}

func autoComplete(all []string, current string) string {
	if len(all) > 1 && len(current) > 0 {
		for k, v := range all {
			if v == current {
				if k+1 < len(all) {
					return all[k+1]
				} else {
					return all[0]
				}
			}
		}
	} else if len(all) >= 1 {
		return all[0]
	}

	return ""
}

func autoCompleteName(input []rune) []rune {
	var name, tmpName []rune
	for i := len(input) - 1; i >= 0; i-- {
		if input[i] == ' ' {
			input = input[:i+1]
			break
		}
		tmpName = append(tmpName, input[i])
	}

	for i := len(tmpName) - 1; i >= 0; i-- {
		name = append(name, tmpName[i])
	}

	var names []string
	for _, c2 := range Conns() {
		names = append(names, c2.Username())
	}

	return append(input, []rune(autoComplete(names, string(name)))...)
}

func autoCompleteCommand(input []rune) []rune {
	var cmds []string
	for cmd := range chatCommands {
		cmds = append(cmds, cmd)
	}
	sort.Strings(cmds)

	return []rune(autoComplete(cmds, string(input)))
}

func initCurses(l *Logger) {
	gocurses.Initscr()
	gocurses.Cbreak()
	gocurses.Noecho()
	gocurses.Stdscr.Keypad(true)

	go func() {
		h := &History{}

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
			case 3:
				consoleInput = h.Next()
			case 4:
				consoleInput = h.Prev(consoleInput)
			case '\b':
				if len(consoleInput) > 0 {
					consoleInput = consoleInput[:len(consoleInput)-1]
				}
			case '\t':
				if strings.Count(string(consoleInput), " ") > 0 {
					consoleInput = autoCompleteName(consoleInput)
				} else {
					consoleInput = autoCompleteCommand(consoleInput)
				}
			case '\n':
				params := strings.Split(string(consoleInput), " ")
				h.Add(consoleInput)
				consoleInput = []rune{}

				if chatCommands[params[0]].function == nil {
					log.Print("Unknown command " + params[0] + ".")
					continue
				}

				if !chatCommands[params[0]].console {
					log.Print("This command is not available to the console!")
					continue
				}

				chatCommands[params[0]].function(nil, strings.Join(params[1:], " "))
			default:
				consoleInput = append(consoleInput, ch)
			}

			draw(l.visible)
		}
	}()
}

type History struct {
	lines [][]rune
	i     int
}

func (h *History) Add(line []rune) {
	for k, v := range h.lines {
		if string(v) == string(line) {
			if k+1 < len(h.lines) {
				h.lines = append(h.lines[:k], h.lines[k+1:]...)
			} else {
				h.lines = h.lines[:k]
			}
		}
	}

	h.lines = append(h.lines, line)
	h.i = 0
}

func (h *History) Prev(current []rune) []rune {
	h.i++
	i := len(h.lines) - h.i
	if i < 0 || i >= len(h.lines) {
		h.i--
		return current
	}

	return h.lines[i]
}

func (h *History) Next() []rune {
	h.i--
	if h.i < 1 {
		h.i = 0
		return []rune{}
	}

	return h.lines[len(h.lines)-h.i]
}
