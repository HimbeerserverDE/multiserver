package main

import (
	"encoding/binary"
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
	function func(*Peer, string)
}

var chatCommands map[string]chatCommand
var onChatMsg []func(*Peer, string) bool

var onServerChatMsg []func(*Peer, string) bool

// RegisterChatCommand registers a callback function that is called
// when a client executes the command and has the required privileges
func RegisterChatCommand(name string, privs map[string]bool, help string, function func(*Peer, string)) {
	chatCommands[name] = chatCommand{
		privs:    privs,
		help:     help,
		function: function,
	}
}

// Help returns the help string of a chatCommand
func (c chatCommand) Help() string { return c.help }

// RegisterOnChatMessage registers a callback function that is called
// when a client sends a chat message
// If a callback function returns true the message is not forwarded
// to the minetest server
func RegisterOnChatMessage(function func(*Peer, string) bool) {
	onChatMsg = append(onChatMsg, function)
}

// RegisterOnServerChatMessage registers a callback function
// that is called when a server sends a chat message
// If a callback function returns true the message is not forwarded
// to the minetest clients
func RegisterOnServerChatMessage(function func(*Peer, string) bool) {
	onServerChatMsg = append(onServerChatMsg, function)
}

func processChatMessage(p *Peer, pkt rudp.Pkt) bool {
	s := string(narrow(pkt.Data[4:]))
	if strings.HasPrefix(s, ChatCommandPrefix) {
		// Chat command
		s = strings.Replace(s, ChatCommandPrefix, "", 1)
		params := strings.Split(s, " ")

		// Priv check
		allow, err := p.CheckPrivs(chatCommands[params[0]].privs)
		if err != nil {
			log.Print(err)
			return true
		}

		if !allow {
			str := "You do not have permission to run this command! Required privileges: " + strings.Replace(encodePrivs(chatCommands[params[0]].privs), "|", " ", -1)
			wstr := wider([]byte(str))

			data := make([]byte, 16+len(wstr))
			data[0] = uint8(0x00)
			data[1] = uint8(ToClientChatMessage)
			data[2] = uint8(0x01)
			data[3] = uint8(0x00)
			data[4] = uint8(0x00)
			data[5] = uint8(0x00)
			binary.BigEndian.PutUint16(data[6:8], uint16(len(str)))
			copy(data[8:8+len(wstr)], wstr)
			data[8+len(wstr)] = uint8(0x00)
			data[9+len(wstr)] = uint8(0x00)
			data[10+len(wstr)] = uint8(0x00)
			data[11+len(wstr)] = uint8(0x00)
			binary.BigEndian.PutUint32(data[12+len(wstr):16+len(wstr)], uint32(time.Now().Unix()))

			ack, err := p.Send(rudp.Pkt{Data: data})
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

			data := make([]byte, 16+len(wstr))
			data[0] = uint8(0x00)
			data[1] = uint8(ToClientChatMessage)
			data[2] = uint8(0x01)
			data[3] = uint8(0x00)
			data[4] = uint8(0x00)
			data[5] = uint8(0x00)
			binary.BigEndian.PutUint16(data[6:8], uint16(len(str)))
			copy(data[8:8+len(wstr)], wstr)
			data[8+len(wstr)] = uint8(0x00)
			data[9+len(wstr)] = uint8(0x00)
			data[10+len(wstr)] = uint8(0x00)
			data[11+len(wstr)] = uint8(0x00)
			binary.BigEndian.PutUint32(data[12+len(wstr):16+len(wstr)], uint32(time.Now().Unix()))

			ack, err := p.Send(rudp.Pkt{Data: data})
			if err != nil {
				log.Print(err)
			}
			<-ack

			return true
		}

		chatCommands[params[0]].function(p, strings.Join(params[1:], " "))
		return true
	} else {
		// Regular message
		noforward := false
		for i := range onChatMsg {
			if onChatMsg[i](p, s) {
				noforward = true
			}
		}
		return noforward
	}
}

func processServerChatMessage(p *Peer, pkt rudp.Pkt) bool {
	s := string(narrow(pkt.Data[4:]))
	noforward := false
	for i := range onServerChatMsg {
		if onServerChatMsg[i](p, s) {
			noforward = true
		}
	}
	return noforward
}

// SendChatMsg sends a chat message to the Peer if it isn't a server
func (p *Peer) SendChatMsg(msg string) {
	if p.IsSrv() {
		return
	}

	wstr := wider([]byte(msg))

	data := make([]byte, 16+len(wstr))
	data[0] = uint8(0x00)
	data[1] = uint8(ToClientChatMessage)
	data[2] = uint8(0x01)
	data[3] = uint8(0x00)
	data[4] = uint8(0x00)
	data[5] = uint8(0x00)
	binary.BigEndian.PutUint16(data[6:8], uint16(len(msg)))
	copy(data[8:8+len(wstr)], wstr)
	data[8+len(wstr)] = uint8(0x00)
	data[9+len(wstr)] = uint8(0x00)
	data[10+len(wstr)] = uint8(0x00)
	data[11+len(wstr)] = uint8(0x00)
	binary.BigEndian.PutUint32(data[12+len(wstr):16+len(wstr)], uint32(time.Now().Unix()))

	ack, err := p.Send(rudp.Pkt{Data: data})
	if err != nil {
		log.Print(err)
	}
	<-ack
}

// ChatSendAll sends a chat message to all connected client Peers
func ChatSendAll(msg string) {
	for _, p := range Peers() {
		go p.SendChatMsg(msg)
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
	prefix, ok := GetConfKey("command_prefix").(string)
	if ok {
		ChatCommandPrefix = prefix
	}
}
