package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/anon55555/mt/rudp"
)

var ChatCommandPrefix string = "#"

type chatCommand struct {
	help     string
	privs    map[string]bool
	console  bool
	function func(*Conn, string)
}

var chatCommands map[string]chatCommand
var onChatMsg []func(*Conn, string) bool

var onServerChatMsg []func(*Conn, string) bool

// RegisterChatCommand registers a callback function that is called
// when a client executes the command and has the required privileges
func RegisterChatCommand(name, help string, privs map[string]bool, console bool, function func(*Conn, string)) {
	chatCommands[name] = chatCommand{
		help:     help,
		privs:    privs,
		console:  console,
		function: function,
	}
}

// Help returns the help string of a chatCommand
func (c chatCommand) Help() string { return c.help }

// RegisterOnChatMessage registers a callback function that is called
// when a client sends a chat message
// If a callback function returns true the message is not forwarded
// to the minetest server
func RegisterOnChatMessage(function func(*Conn, string) bool) {
	onChatMsg = append(onChatMsg, function)
}

// RegisterOnServerChatMessage registers a callback function
// that is called when a server sends a chat message
// If a callback function returns true the message is not forwarded
// to the minetest clients
func RegisterOnServerChatMessage(function func(*Conn, string) bool) {
	onServerChatMsg = append(onServerChatMsg, function)
}

func processChatMessage(c *Conn, pkt rudp.Pkt) bool {
	r := ByteReader(pkt)

	wstr := make([]byte, r.Len()-4)
	r.ReadAt(wstr, 4)

	s := string(narrow(wstr))
	if strings.HasPrefix(s, ChatCommandPrefix) {
		// Chat command
		s = strings.Replace(s, ChatCommandPrefix, "", 1)
		params := strings.Split(s, " ")

		// Priv check
		allow, err := c.CheckPrivs(chatCommands[params[0]].privs)
		if err != nil {
			log.Print(err)
			return true
		}

		if !allow {
			str := "You do not have permission to run this command! Required privileges: " + strings.Replace(encodePrivs(chatCommands[params[0]].privs), "|", " ", -1)
			wstr := wider([]byte(str))

			w := bytes.NewBuffer([]byte{0x00, ToClientChatMessage})
			WriteUint8(w, 1)
			WriteUint8(w, 0)
			WriteBytes16(w, []byte{})
			WriteUint16(w, uint16(len(str)))
			w.Write(wstr)
			WriteUint64(w, uint64(time.Now().Unix()))

			ack, err := c.Send(rudp.Pkt{Reader: w})
			if err != nil {
				log.Print(err)
			}
			<-ack

			return true
		}

		// Callback
		// Existance check
		if chatCommands[params[0]].function == nil {
			str := "Unknown command " + params[0] + "."
			wstr := wider([]byte(str))

			w := bytes.NewBuffer([]byte{0x00, ToClientChatMessage})
			WriteUint8(w, 1)
			WriteUint8(w, 0)
			WriteBytes16(w, []byte{})
			WriteUint16(w, uint16(len(str)))
			w.Write(wstr)
			WriteUint64(w, uint64(time.Now().Unix()))

			ack, err := c.Send(rudp.Pkt{Reader: w})
			if err != nil {
				log.Print(err)
			}
			<-ack

			return true
		}

		chatCommands[params[0]].function(c, strings.Join(params[1:], " "))
		return true
	} else {
		// Regular message
		noforward := false
		for i := range onChatMsg {
			if onChatMsg[i](c, s) {
				noforward = true
			}
		}
		return noforward
	}
}

func processServerChatMessage(c *Conn, pkt rudp.Pkt) bool {
	r := ByteReader(pkt)

	r.Seek(4, io.SeekStart)

	wstr := make([]byte, r.Len())
	r.Read(wstr)

	s := string(narrow(wstr))
	noforward := false
	for i := range onServerChatMsg {
		if onServerChatMsg[i](c, s) {
			noforward = true
		}
	}

	return noforward
}

// SendChatMsg sends a chat message to a Conn if it isn't a server
func (c *Conn) SendChatMsg(msg string) {
	if c.IsSrv() {
		return
	}

	wstr := wider([]byte(msg))

	w := bytes.NewBuffer([]byte{0x00, ToClientChatMessage})
	WriteUint8(w, 1)
	WriteUint8(w, 0)
	WriteBytes16(w, []byte{})
	WriteUint16(w, uint16(len(msg)))
	w.Write(wstr)
	WriteUint64(w, uint64(time.Now().Unix()))

	ack, err := c.Send(rudp.Pkt{Reader: w})
	if err != nil {
		log.Print(err)
	}
	<-ack
}

// ChatSendAll sends a chat message to all connected client Conns
func ChatSendAll(msg string) {
	for _, c := range Conns() {
		go c.SendChatMsg(msg)
	}
}

// Colorize prepends a color escape sequence to a string
func Colorize(text, color string) string {
	return string(0x1b) + "(c@" + color + ")" + text + string(0x1b) + "(c@#FFF)"
}

func narrow(b []byte) []byte {
	if len(b)%2 != 0 {
		return nil
	}

	e := make([]uint16, len(b)/2)

	for i := 0; i < len(b); i += 2 {
		e[i/2] = binary.BigEndian.Uint16(b[i : 2+i])
	}

	return []byte(string(utf16.Decode(e)))
}

func wider(b []byte) []byte {
	r := make([]byte, len(b)*2)

	e := utf16.Encode([]rune(string(b)))

	for i := range e {
		binary.BigEndian.PutUint16(r[i*2:2+i*2], e[i])
	}

	return r
}

func init() {
	chatCommands = make(map[string]chatCommand)

	// Read cmd prefix from config
	prefix, ok := ConfKey("command_prefix").(string)
	if ok {
		ChatCommandPrefix = prefix
	}
}
